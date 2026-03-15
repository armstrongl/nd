package cmd

import (
	"fmt"
	"strings"

	"github.com/larah/nd/internal/asset"
	"github.com/larah/nd/internal/deploy"
	"github.com/larah/nd/internal/nd"
	"github.com/spf13/cobra"
)

func newRemoveCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <asset> [assets...]",
		Short: "Remove deployed assets",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()

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
				return withExitCode(nd.ExitPartialFailure,
					fmt.Errorf("%d of %d removals failed", len(bulkResult.Failed), len(reqs)))
			}
			return nil
		},
	}
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
