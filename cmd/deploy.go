package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/armstrongl/nd/internal/agent"
	"github.com/armstrongl/nd/internal/asset"
	"github.com/armstrongl/nd/internal/deploy"
	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/oplog"
	"github.com/armstrongl/nd/internal/sourcemanager"
	"github.com/spf13/cobra"
)

func newDeployCmd(app *App) *cobra.Command {
	var (
		assetType string
		relative  bool
		absolute  bool
		agents    []string
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
			"docs.guides":  "getting-started,how-nd-works,profiles-and-snapshots,creating-sources,asset-types/context",
			"docs.related": "nd remove,nd list,nd status",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()

			// Scan at most once. The interactive picker (no args) scans to build
			// its choices and that same summary is reused to resolve the pick;
			// the args-supplied path scans once below. summary stays nil until a
			// scan happens so the JSON/non-terminal guards can return early
			// without scanning at all.
			var summary *sourcemanager.ScanSummary

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
				summary = scanResult
				agentAlias := ""
				if ag, err := app.ActiveAgent(); err == nil {
					agentAlias = ag.SourceAlias
				}
				var completions []string
				for _, a := range summary.Index.FilterByAgent(agentAlias) {
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

			if summary == nil {
				scanResult, err := app.ScanIndex()
				if err != nil {
					return fmt.Errorf("scan sources: %w", err)
				}
				summary = scanResult
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

			// Resolve deployment scope (may prompt interactively; an explicit
			// --scope flag or a non-interactive invocation skips the prompt).
			// Resolved before the multi-agent branch so every deploy path
			// (single- and multi-agent) honors the chosen scope.
			resolvedScope, err := resolveDeployScope(cmd, app)
			if err != nil {
				return err
			}
			if resolvedScope == nd.ScopeProject {
				if _, err := app.ResolveProjectRoot(); err != nil {
					return err
				}
			}
			app.Scope = resolvedScope

			// Resolve target agents: --agents flag > config default_deploy_agents.
			// When neither is set, fall through to the single active-agent path
			// below (byte-for-byte unchanged).
			targetNames := agents
			if len(targetNames) == 0 {
				if sm, smErr := app.SourceManager(); smErr == nil {
					targetNames = sm.Config().DefaultDeployAgents
				}
			}
			if len(targetNames) > 0 {
				return runMultiAgentDeploy(cmd, app, assets, targetNames, relative, absolute)
			}

			eng, err := app.DeployEngine()
			if err != nil {
				return err
			}

			// Prune ghost deployments for all agents (best-effort pre-op cleanup)
			if pruned, pruneErr := eng.PruneAll(); pruneErr != nil {
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
				unsupported := 0
				for _, f := range bulkResult.Failed {
					if f.UnsupportedType {
						unsupported++
						continue
					}
					printHuman(cmd.ErrOrStderr(), "Failed: %s/%s: %v\n", f.AssetType, f.AssetName, f.Err)
				}
				if unsupported > 0 {
					ag, _ := app.ActiveAgent()
					name := "unknown"
					if ag != nil {
						name = ag.Name
					}
					printHuman(cmd.ErrOrStderr(), "Skipped %d asset(s) (unsupported by agent %s)\n", unsupported, name)
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
		if ag, err := app.ActiveAgent(); err == nil {
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
	cmd.Flags().StringSliceVar(&agents, "agents", nil, "comma-separated target agents (overrides config default_deploy_agents)")
	cmd.RegisterFlagCompletionFunc("agents", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return agent.KnownAgentNames(), cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}

// runMultiAgentDeploy deploys the given assets to each named target agent,
// building one deploy engine per agent. Used when --agents or config
// default_deploy_agents selects explicit targets.
func runMultiAgentDeploy(cmd *cobra.Command, app *App, assets []asset.Asset, targetNames []string, relative, absolute bool) error {
	w := cmd.OutOrStdout()

	reg, err := app.AgentRegistry()
	if err != nil {
		return err
	}
	reg.Detect()

	// Resolve and validate every requested agent (dedup, must be known + detected).
	var targets []*agent.Agent
	seen := make(map[string]bool)
	for _, name := range targetNames {
		if seen[name] {
			continue
		}
		seen[name] = true
		ag, gErr := reg.Get(name)
		if gErr != nil {
			return withExitCode(nd.ExitInvalidUsage,
				fmt.Errorf("unknown agent %q; run 'nd doctor' to list detected agents", name))
		}
		if !ag.Detected {
			return withExitCode(nd.ExitInvalidUsage,
				fmt.Errorf("agent %q is not detected on this system; install it or check config", name))
		}
		targets = append(targets, ag)
	}

	// Resolve symlink strategy: flag > config > default (absolute)
	strategy := nd.SymlinkAbsolute
	if sm, smErr := app.SourceManager(); smErr == nil {
		if cfg := sm.Config(); cfg.SymlinkStrategy != "" {
			strategy = cfg.SymlinkStrategy
		}
	}
	if relative {
		strategy = nd.SymlinkRelative
	} else if absolute {
		strategy = nd.SymlinkAbsolute
	}

	// Prune ghost deployments for all agents (best-effort pre-op cleanup).
	if eng0, engErr := app.DeployEngineFor(targets[0]); engErr == nil {
		if pruned, pruneErr := eng0.PruneAll(); pruneErr != nil {
			if !app.Quiet {
				printHuman(cmd.ErrOrStderr(), "warning: prune failed: %v\n", pruneErr)
			}
		} else if pruned > 0 && !app.Quiet {
			printHuman(cmd.ErrOrStderr(), "Pruned %d stale deployment(s)\n", pruned)
		}
	}

	// Dry-run: preview per agent without executing.
	if app.DryRun {
		if app.JSON {
			type dryRunEntry struct {
				AssetType string `json:"asset_type"`
				AssetName string `json:"asset_name"`
				Source    string `json:"source"`
				Agent     string `json:"agent"`
			}
			var entries []dryRunEntry
			for _, ag := range targets {
				for _, a := range assets {
					entries = append(entries, dryRunEntry{
						AssetType: string(a.Type),
						AssetName: a.Name,
						Source:    a.SourceID,
						Agent:     ag.Name,
					})
				}
			}
			return printJSON(w, entries, true)
		}
		for _, ag := range targets {
			for _, a := range assets {
				printHuman(w, "[dry-run] would deploy %s/%s from %s -> %s\n", a.Type, a.Name, a.SourceID, ag.Name)
			}
		}
		return nil
	}

	// Deploy the same asset set to each target agent, merging results.
	var succeeded []deploy.DeployResult
	var failed []deploy.DeployError
	for _, ag := range targets {
		eng, engErr := app.DeployEngineFor(ag)
		if engErr != nil {
			return engErr
		}
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
		bulk, bErr := eng.DeployBulk(reqs)
		if bErr != nil {
			return bErr
		}
		succeeded = append(succeeded, bulk.Succeeded...)
		failed = append(failed, bulk.Failed...)
	}

	var logAssets []asset.Identity
	for _, a := range assets {
		logAssets = append(logAssets, a.Identity)
	}
	app.LogOp(oplog.LogEntry{
		Timestamp: time.Now(),
		Operation: oplog.OpDeploy,
		Assets:    logAssets,
		Scope:     app.Scope,
		Succeeded: len(succeeded),
		Failed:    len(failed),
	})

	if app.JSON {
		return printJSON(w, deploy.BulkDeployResult{Succeeded: succeeded, Failed: failed}, false)
	}

	if !app.Quiet {
		for _, s := range succeeded {
			printHuman(w, "Deployed %s/%s -> %s\n", s.Deployment.AssetType, s.Deployment.AssetName, s.Deployment.Agent)
		}
		unsupportedByAgent := make(map[string]int)
		for _, f := range failed {
			if f.UnsupportedType {
				unsupportedByAgent[f.Agent]++
				continue
			}
			printHuman(cmd.ErrOrStderr(), "Failed: %s/%s -> %s: %v\n", f.AssetType, f.AssetName, f.Agent, f.Err)
		}
		for name, count := range unsupportedByAgent {
			printHuman(cmd.ErrOrStderr(), "Skipped %d asset(s) (unsupported by agent %s)\n", count, name)
		}
		// Print settings reminder once per type that needs registration.
		settingsTypes := make(map[nd.AssetType]bool)
		for _, s := range succeeded {
			if s.Deployment.AssetType.RequiresSettingsRegistration() {
				settingsTypes[s.Deployment.AssetType] = true
			}
		}
		for t := range settingsTypes {
			printSettingsReminder(w, t)
		}
	}

	if len(failed) > 0 {
		if !app.Quiet {
			if name := latestAutoSnapshot(app); name != "" {
				printHuman(w, "Auto-snapshot saved. Restore with: nd snapshot restore %s\n", name)
			}
		}
		return withExitCode(nd.ExitPartialFailure,
			fmt.Errorf("%d of %d deployments failed", len(failed), len(assets)*len(targets)))
	}
	return nil
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
