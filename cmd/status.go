package cmd

import (
	"fmt"

	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/state"
	"github.com/spf13/cobra"
)

func newStatusCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show deployment status and health",
		Example: `  # Show all deployed assets and their health
  nd status

  # Output as JSON for scripting
  nd status --json

  # Show project-scope deployments
  nd status --scope project`,
		Annotations: map[string]string{
			"docs.guides":  "getting-started,troubleshooting",
			"docs.related": "nd doctor,nd deploy",
		},
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()

			agents, err := app.DeployAgents()
			if err != nil {
				return err
			}

			// Prune ghost deployments for all agents once (PruneAll ignores the
			// engine's bound agent, so a single call covers every agent).
			if eng0, engErr := app.DeployEngineFor(agents[0]); engErr == nil {
				if pruned, pruneErr := eng0.PruneAll(); pruneErr != nil {
					if !app.Quiet {
						printHuman(cmd.ErrOrStderr(), "warning: prune failed: %v\n", pruneErr)
					}
				} else if pruned > 0 && !app.Quiet {
					printHuman(cmd.ErrOrStderr(), "Pruned %d stale deployment(s)\n", pruned)
				}
			}

			// Collect status across every detected agent (per-agent engines each
			// see only their own deployments in the shared state file).
			var filtered []statusDisplay
			for _, ag := range agents {
				eng, engErr := app.DeployEngineFor(ag)
				if engErr != nil {
					return engErr
				}
				entries, err := eng.Status()
				if err != nil {
					return fmt.Errorf("load status: %w", err)
				}
				for _, e := range entries {
					if app.Scope == nd.ScopeProject && e.Deployment.Scope == nd.ScopeProject &&
						e.Deployment.ProjectPath != app.ProjectRoot {
						continue
					}
					agentName := e.Deployment.Agent
					if agentName == "" {
						agentName = "claude-code"
					}
					filtered = append(filtered, statusDisplay{
						AssetType: string(e.Deployment.AssetType),
						AssetName: e.Deployment.AssetName,
						Source:    e.Deployment.SourceID,
						Scope:     string(e.Deployment.Scope),
						Origin:    string(e.Deployment.Origin),
						Agent:     agentName,
						Health:    e.Health.String(),
						Detail:    e.Detail,
					})
				}
			}

			// Active profile
			var activeProfile string
			profMgr, profErr := app.ProfileManager()
			if profErr == nil {
				activeProfile, _ = profMgr.ActiveProfile()
			}

			if app.JSON {
				result := struct {
					ActiveProfile string          `json:"active_profile,omitempty"`
					Deployments   []statusDisplay `json:"deployments"`
				}{
					ActiveProfile: activeProfile,
					Deployments:   filtered,
				}
				return printJSON(w, result, app.DryRun)
			}

			if activeProfile != "" {
				printHuman(w, "Active profile: %s\n\n", activeProfile)
			}

			if len(filtered) == 0 {
				printHuman(w, "No deployments.\n")
				return nil
			}

			// Group by asset type
			grouped := make(map[string][]statusDisplay)
			var order []string
			for _, d := range filtered {
				if _, seen := grouped[d.AssetType]; !seen {
					order = append(order, d.AssetType)
				}
				grouped[d.AssetType] = append(grouped[d.AssetType], d)
			}

			for _, t := range order {
				printHuman(w, "%s:\n", t)
				for _, d := range grouped[t] {
					healthMark := "✓"
					if d.Health != state.HealthOK.String() {
						healthMark = "✗"
					}
					printHuman(w, "  %s %-25s  %-8s  %-8s  %-12s  %s\n",
						healthMark, d.AssetName, d.Scope, d.Origin, d.Agent, d.Source)
				}
			}

			return nil
		},
	}
}

type statusDisplay struct {
	AssetType string `json:"asset_type"`
	AssetName string `json:"asset_name"`
	Source    string `json:"source"`
	Scope     string `json:"scope"`
	Origin    string `json:"origin"`
	Agent     string `json:"agent"`
	Health    string `json:"health"`
	Detail    string `json:"detail,omitempty"`
}
