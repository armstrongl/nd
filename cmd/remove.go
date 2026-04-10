package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/armstrongl/nd/internal/asset"
	"github.com/armstrongl/nd/internal/deploy"
	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/oplog"
	"github.com/spf13/cobra"
)

func newRemoveCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <asset> [assets...]",
		Short: "Remove deployed assets",
		Example: `  # Remove a deployed asset
  nd remove skills/greeting

  # Remove multiple assets
  nd remove skills/greeting commands/hello

  # Skip confirmation prompt
  nd remove skills/greeting --yes

  # Preview what would be removed
  nd remove skills/greeting --dry-run`,
		Annotations: map[string]string{
			"docs.guides": "getting-started,how-nd-works,profiles-and-snapshots",
		},
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()

			// Interactive picker when no args provided
			if len(args) == 0 {
				if app.JSON {
					return fmt.Errorf("requires at least one asset argument; run 'nd status --json' to see deployed assets")
				}
				if !isTerminal() {
					return fmt.Errorf("requires at least one asset argument; run 'nd status' to see deployed assets")
				}
				completionInitApp(app)
				completions, _ := completeDeployedAssets(app, "")
				if len(completions) == 0 {
					return fmt.Errorf("no deployed assets to remove")
				}
				names := extractChoiceNames(completions)
				choice, err := promptChoice(cmd.InOrStdin(), w, "Select asset to remove:", names)
				if err != nil {
					return err
				}
				args = []string{choice}
			}

			eng, err := app.DeployEngine()
			if err != nil {
				return err
			}

			entries, err := eng.Status()
			if err != nil {
				return fmt.Errorf("load deployment status: %w", err)
			}

			// Resolve each asset reference to a deployed entry
			var reqs []deploy.RemoveRequest
			for _, ref := range args {
				dep, err := findDeployedAsset(entries, ref, app.Scope, app.ProjectRoot)
				if err != nil {
					return withExitCode(nd.ExitInvalidUsage, err)
				}

				d := dep.Deployment

				// Warn if pinned
				if d.Origin == nd.OriginPinned && !app.Yes {
					ok, err := confirm(cmd.InOrStdin(), w,
						fmt.Sprintf("Asset %s/%s is pinned. Remove anyway?", d.AssetType, d.AssetName),
						app.Yes,
					)
					if err != nil {
						return err
					}
					if !ok {
						if !app.Quiet {
							printHuman(w, "Skipped pinned asset %s/%s\n", d.AssetType, d.AssetName)
						}
						continue
					}
				}

				// Confirm non-pinned removal (skip if --yes or --dry-run)
				if d.Origin != nd.OriginPinned && !app.DryRun {
					ok, err := confirm(cmd.InOrStdin(), w,
						fmt.Sprintf("Remove %s/%s?", d.AssetType, d.AssetName),
						app.Yes,
					)
					if err != nil {
						return err
					}
					if !ok {
						if !app.Quiet {
							printHuman(w, "Skipped %s/%s\n", d.AssetType, d.AssetName)
						}
						continue
					}
				}

				reqs = append(reqs, deploy.RemoveRequest{
					Identity: asset.Identity{
						SourceID: d.SourceID,
						Type:     d.AssetType,
						Name:     d.AssetName,
					},
					Scope:       d.Scope,
					ProjectRoot: d.ProjectPath,
				})
			}

			if len(reqs) == 0 {
				return nil
			}

			if app.DryRun {
				if app.JSON {
					return printJSON(w, reqs, true)
				}
				for _, r := range reqs {
					printHuman(w, "[dry-run] would remove %s/%s\n", r.Identity.Type, r.Identity.Name)
				}
				return nil
			}

			if len(reqs) == 1 {
				if err := eng.Remove(reqs[0]); err != nil {
					return err
				}
				app.LogOp(oplog.LogEntry{
					Timestamp: time.Now(),
					Operation: oplog.OpRemove,
					Assets:    []asset.Identity{reqs[0].Identity},
					Scope:     app.Scope,
					Succeeded: 1,
				})
				if app.JSON {
					return printJSON(w, map[string]string{
						"removed": fmt.Sprintf("%s/%s", reqs[0].Identity.Type, reqs[0].Identity.Name),
					}, false)
				}
				if !app.Quiet {
					printHuman(w, "Removed %s/%s\n", reqs[0].Identity.Type, reqs[0].Identity.Name)
				}
				return nil
			}

			bulkResult, err := eng.RemoveBulk(reqs)
			if err != nil {
				return err
			}

			var logAssets []asset.Identity
			for _, r := range reqs {
				logAssets = append(logAssets, r.Identity)
			}
			app.LogOp(oplog.LogEntry{
				Timestamp: time.Now(),
				Operation: oplog.OpRemove,
				Assets:    logAssets,
				Scope:     app.Scope,
				Succeeded: len(bulkResult.Succeeded),
				Failed:    len(bulkResult.Failed),
			})

			if app.JSON {
				return printJSON(w, bulkResult, false)
			}

			if !app.Quiet {
				for _, s := range bulkResult.Succeeded {
					printHuman(w, "Removed %s/%s\n", s.Identity.Type, s.Identity.Name)
				}
				for _, f := range bulkResult.Failed {
					printHuman(cmd.ErrOrStderr(), "Failed: %s/%s: %v\n", f.Identity.Type, f.Identity.Name, f.Err)
				}
			}

			if len(bulkResult.Failed) > 0 {
				if !app.Quiet {
					if name := latestAutoSnapshot(app); name != "" {
						printHuman(w, "Auto-snapshot saved. Restore with: nd snapshot restore %s\n", name)
					}
				}
				return withExitCode(nd.ExitPartialFailure,
					fmt.Errorf("%d of %d removals failed", len(bulkResult.Failed), len(reqs)))
			}
			return nil
		},
	}
	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeDeployedAssets(app, toComplete)
	}
	return cmd
}

// findDeployedAsset finds a deployment matching a reference string.
// Supports "name" and "type/name" formats, filtered by scope.
func findDeployedAsset(entries []deploy.StatusEntry, ref string, scope nd.Scope, projectRoot string) (*deploy.StatusEntry, error) {
	var assetType nd.AssetType
	name := ref

	if parts := strings.SplitN(ref, "/", 2); len(parts) == 2 {
		assetType = nd.AssetType(parts[0])
		name = parts[1]
	}

	var matches []deploy.StatusEntry
	for _, e := range entries {
		if e.Deployment.Scope != scope {
			continue
		}
		if scope == nd.ScopeProject && e.Deployment.ProjectPath != projectRoot {
			continue
		}
		if !strings.EqualFold(e.Deployment.AssetName, name) {
			continue
		}
		if assetType != "" && e.Deployment.AssetType != assetType {
			continue
		}
		matches = append(matches, e)
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no deployed asset matching %q in %s scope", ref, scope)
	}
	if len(matches) == 1 {
		return &matches[0], nil
	}

	var candidates []string
	for _, m := range matches {
		candidates = append(candidates, fmt.Sprintf("  %s/%s (from %s)", m.Deployment.AssetType, m.Deployment.AssetName, m.Deployment.SourceID))
	}
	return nil, fmt.Errorf("ambiguous asset %q — matches:\n%s\nUse type/name format to disambiguate",
		ref, strings.Join(candidates, "\n"))
}
