package cmd

import (
	"fmt"

	"github.com/larah/nd/internal/nd"
	"github.com/larah/nd/internal/state"
	"github.com/spf13/cobra"
)

func newStatusCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show deployment status and health",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()

			eng, err := app.DeployEngine()
			if err != nil {
				return err
			}

			entries, err := eng.Status()
			if err != nil {
				return fmt.Errorf("load status: %w", err)
			}

			// Filter by scope/project
			var filtered []statusDisplay
			for _, e := range entries {
				if app.Scope == nd.ScopeProject && e.Deployment.Scope == nd.ScopeProject &&
					e.Deployment.ProjectPath != app.ProjectRoot {
					continue
				}
				filtered = append(filtered, statusDisplay{
					AssetType: string(e.Deployment.AssetType),
					AssetName: e.Deployment.AssetName,
					Source:    e.Deployment.SourceID,
					Scope:     string(e.Deployment.Scope),
					Origin:    string(e.Deployment.Origin),
					Health:    e.Health.String(),
					Detail:    e.Detail,
				})
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
					printHuman(w, "  %s %-25s  %-8s  %-8s  %s\n",
						healthMark, d.AssetName, d.Scope, d.Origin, d.Source)
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
	Health    string `json:"health"`
	Detail    string `json:"detail,omitempty"`
}
