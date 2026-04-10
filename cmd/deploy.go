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

func newDeployCmd(app *App) *cobra.Command {
	var (
		assetType string
		relative  bool
		absolute  bool
	)

	cmd := &cobra.Command{
		Use:   "deploy <asset> [assets...]",
		Short: "Deploy assets by creating symlinks",
		Long: `Deploy one or more assets by creating symlinks from source to agent config.

Asset references can be:
  name Search all types for matching name
  type/name Search specific type (e.g., skills/greeting)`,
		Example: `  # Deploy a single asset
  nd deploy skills/greeting

  # Deploy by name (if unique across types)
  nd deploy greeting

  # Deploy multiple assets at once
  nd deploy skills/greeting commands/hello agents/researcher

  # Filter by type
  nd deploy --type skills greeting

  # Deploy to project scope
  nd deploy skills/greeting --scope project

  # Use relative symlinks
  nd deploy skills/greeting --relative

  # Script-friendly: skip prompts, output JSON
  nd deploy skills/greeting --yes --json`,
		Annotations: map[string]string{
			"docs.guides": "getting-started,how-nd-works,profiles-and-snapshots,creating-sources,asset-types/context",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()

			// Interactive picker when no args provided
			if len(args) == 0 {
				if app.JSON {
					return fmt.Errorf("requires at least one asset argument; run 'nd list --json' to see available assets")
				}
				if !isTerminal() {
					return fmt.Errorf("requires at least one asset argument; run 'nd list' to see available assets")
				}
				completionInitApp(app)
				scanResult, err := app.ScanIndex()
				if err != nil {
					return fmt.Errorf("scan sources: %w", err)
				}
				agentAlias := ""
				if ag, err := app.DefaultAgent(); err == nil {
					agentAlias = ag.SourceAlias
				}
				var completions []string
				for _, a := range scanResult.Index.FilterByAgent(agentAlias) {
					completions = append(completions, fmt.Sprintf("%s/%s\t%s from %s", a.Type, a.Name, a.Type, a.SourceID))
				}
				if len(completions) == 0 {
					return fmt.Errorf("no assets available; add a source with 'nd source add <path>'")
				}
				names := extractChoiceNames(completions)
				choice, err := promptChoice(cmd.InOrStdin(), w, "Select asset to deploy:", names)
				if err != nil {
					return err
				}
				args = []string{choice}
			}

			summary, err := app.ScanIndex()
			if err != nil {
				return fmt.Errorf("scan sources: %w", err)
			}
			index := summary.Index

			// Print conflict warnings
			for _, c := range index.Conflicts() {
				if !app.Quiet {
					printHuman(cmd.ErrOrStderr(), "warning: %s/%s exists in both %s and %s (using %s)\n",
						c.Type, c.Name, c.Winner, c.Loser, c.Winner)
				}
			}

			// Resolve each asset reference
			var assets []asset.Asset
			for _, ref := range args {
				resolved, err := resolveAssetRef(index, ref, nd.AssetType(assetType))
				if err != nil {
					return withExitCode(nd.ExitInvalidUsage, err)
				}
				assets = append(assets, *resolved)
			}

			eng, err := app.DeployEngine()
			if err != nil {
				return err
			}

			// Prune ghost deployments (best-effort)
			if pruned, pruneErr := eng.Prune(); pruneErr != nil {
				if !app.Quiet {
					printHuman(cmd.ErrOrStderr(), "warning: prune failed: %v\n", pruneErr)
				}
			} else if pruned > 0 && !app.Quiet {
				printHuman(cmd.ErrOrStderr(), "Pruned %d stale deployment(s)\n", pruned)
			}

			if app.DryRun {
				if app.JSON {
					type dryRunEntry struct {
						AssetType string `json:"asset_type"`
						AssetName string `json:"asset_name"`
						Source    string `json:"source"`
					}
					entries := make([]dryRunEntry, len(assets))
					for i, a := range assets {
						entries[i] = dryRunEntry{
							AssetType: string(a.Type),
							AssetName: a.Name,
							Source:    a.SourceID,
						}
					}
					return printJSON(w, entries, true)
				}
				for _, a := range assets {
					printHuman(w, "[dry-run] would deploy %s/%s from %s\n", a.Type, a.Name, a.SourceID)
				}
				return nil
			}

			// Resolve symlink strategy: flag > config > default (absolute)
			strategy := nd.SymlinkAbsolute
			if sm, smErr := app.SourceManager(); smErr == nil {
				cfg := sm.Config()
				if cfg.SymlinkStrategy != "" {
					strategy = cfg.SymlinkStrategy
				}
			}
			if relative {
				strategy = nd.SymlinkRelative
			} else if absolute {
				strategy = nd.SymlinkAbsolute
			}

			// Build deploy requests
			reqs := make([]deploy.DeployRequest, len(assets))
			for i, a := range assets {
				reqs[i] = deploy.DeployRequest{
					Asset:       a,
					Scope:       app.Scope,
					ProjectRoot: app.ProjectRoot,
					Origin:      nd.OriginManual,
					Strategy:    strategy,
				}
			}

			if len(reqs) == 1 {
				result, err := eng.Deploy(reqs[0])
				if err != nil {
					return err
				}
				app.LogOp(oplog.LogEntry{
					Timestamp: time.Now(),
					Operation: oplog.OpDeploy,
					Assets:    []asset.Identity{reqs[0].Asset.Identity},
					Scope:     app.Scope,
					Succeeded: 1,
				})
				if app.JSON {
					return printJSON(w, result, false)
				}
				if !app.Quiet {
					printHuman(w, "Deployed %s/%s\n", reqs[0].Asset.Type, reqs[0].Asset.Name)
					printSettingsReminder(w, reqs[0].Asset.Type)
				}
				return nil
			}

			bulkResult, err := eng.DeployBulk(reqs)
			if err != nil {
				return err
			}

			var logAssets []asset.Identity
			for _, r := range reqs {
				logAssets = append(logAssets, r.Asset.Identity)
			}
			app.LogOp(oplog.LogEntry{
				Timestamp: time.Now(),
				Operation: oplog.OpDeploy,
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
					printHuman(w, "Deployed %s/%s\n", s.Deployment.AssetType, s.Deployment.AssetName)
				}
				for _, f := range bulkResult.Failed {
					printHuman(cmd.ErrOrStderr(), "Failed: %s/%s: %v\n", f.AssetType, f.AssetName, f.Err)
				}
				// Print settings reminder once if any deployed type needs it
				settingsTypes := make(map[nd.AssetType]bool)
				for _, s := range bulkResult.Succeeded {
					if s.Deployment.AssetType.RequiresSettingsRegistration() {
						settingsTypes[s.Deployment.AssetType] = true
					}
				}
				for t := range settingsTypes {
					printSettingsReminder(w, t)
				}
			}

			if len(bulkResult.Failed) > 0 {
				if !app.Quiet {
					if name := latestAutoSnapshot(app); name != "" {
						printHuman(w, "Auto-snapshot saved. Restore with: nd snapshot restore %s\n", name)
					}
				}
				return withExitCode(nd.ExitPartialFailure,
					fmt.Errorf("%d of %d deployments failed", len(bulkResult.Failed), len(reqs)))
			}
			return nil
		},
	}
	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		completionInitApp(app)
		summary, err := app.ScanIndex()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		agentAlias := ""
		if ag, err := app.DefaultAgent(); err == nil {
			agentAlias = ag.SourceAlias
		}
		var names []string
		for _, a := range summary.Index.FilterByAgent(agentAlias) {
			name := fmt.Sprintf("%s/%s", a.Type, a.Name)
			if toComplete == "" || strings.HasPrefix(name, toComplete) || strings.HasPrefix(a.Name, toComplete) {
				names = append(names, fmt.Sprintf("%s/%s\t%s from %s", a.Type, a.Name, a.Type, a.SourceID))
			}
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
	cmd.Flags().StringVar(&assetType, "type", "", "asset type filter (skills, commands, rules, etc.)")
	cmd.RegisterFlagCompletionFunc("type", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		var types []string
		for _, t := range nd.AllAssetTypes() {
			types = append(types, string(t))
		}
		return types, cobra.ShellCompDirectiveNoFileComp
	})
	cmd.Flags().BoolVar(&relative, "relative", false, "use relative symlinks (overrides config)")
	cmd.Flags().BoolVar(&absolute, "absolute", false, "use absolute symlinks (overrides config)")
	cmd.MarkFlagsMutuallyExclusive("relative", "absolute")
	return cmd
}

// resolveAssetRef resolves an asset reference string to a single asset.
// Formats: "name", "type/name"
func resolveAssetRef(index *asset.Index, ref string, typeFilter nd.AssetType) (*asset.Asset, error) {
	// Check for type/name format
	if parts := strings.SplitN(ref, "/", 2); len(parts) == 2 {
		assetType := nd.AssetType(parts[0])
		name := parts[1]
		a := index.SearchByTypeAndName(assetType, name)
		if a == nil {
			return nil, fmt.Errorf("asset %s/%s not found", assetType, name)
		}
		return a, nil
	}

	// Name-only search, optionally filtered by --type
	if typeFilter != "" {
		a := index.SearchByTypeAndName(typeFilter, ref)
		if a == nil {
			return nil, fmt.Errorf("asset %s/%s not found", typeFilter, ref)
		}
		return a, nil
	}

	matches := index.SearchByName(ref)
	if len(matches) == 0 {
		return nil, fmt.Errorf("asset %q not found in any source", ref)
	}
	if len(matches) == 1 {
		return matches[0], nil
	}

	// Ambiguous: print candidates
	var candidates []string
	for _, m := range matches {
		candidates = append(candidates, fmt.Sprintf("  %s/%s (from %s)", m.Type, m.Name, m.SourceID))
	}
	return nil, fmt.Errorf("ambiguous asset %q — matches:\n%s\nUse type/name format to disambiguate",
		ref, strings.Join(candidates, "\n"))
}

// printSettingsReminder prints a reminder for asset types that need settings.json registration.
func printSettingsReminder(w interface{ Write([]byte) (int, error) }, t nd.AssetType) {
	if !t.RequiresSettingsRegistration() {
		return
	}
	fmt.Fprintf(w, "Note: %s require manual registration in settings.json\n", t)
}
