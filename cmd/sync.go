package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newSyncCmd(app *App) *cobra.Command {
	var sourceID string

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Repair symlinks and optionally pull git sources",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()

			// If --source specified, git pull that source first
			if sourceID != "" {
				sm, err := app.SourceManager()
				if err != nil {
					return err
				}
				if !app.Quiet {
					printHuman(w, "Syncing source %q...\n", sourceID)
				}
				if app.DryRun {
					printHuman(w, "[dry-run] would git pull source %q\n", sourceID)
				} else {
					if err := sm.SyncSource(sourceID); err != nil {
						return fmt.Errorf("sync source %q: %w", sourceID, err)
					}
					if !app.Quiet {
						printHuman(w, "Source %q updated.\n", sourceID)
					}
				}
			}

			// Repair symlinks
			eng, err := app.DeployEngine()
			if err != nil {
				return err
			}

			if app.DryRun {
				checks, err := eng.Check()
				if err != nil {
					return fmt.Errorf("check deployments: %w", err)
				}
				if app.JSON {
					return printJSON(w, checks, true)
				}
				if len(checks) == 0 {
					printHuman(w, "[dry-run] all deployments healthy, nothing to repair.\n")
				} else {
					for _, hc := range checks {
						printHuman(w, "[dry-run] would repair %s/%s: %s\n",
							hc.Deployment.AssetType, hc.Deployment.AssetName, hc.Detail)
					}
				}
				return nil
			}

			result, err := eng.Sync()
			if err != nil {
				return fmt.Errorf("sync deployments: %w", err)
			}

			if app.JSON {
				return printJSON(w, result, false)
			}

			if !app.Quiet {
				for _, d := range result.Repaired {
					printHuman(w, "Repaired %s/%s\n", d.AssetType, d.AssetName)
				}
				for _, d := range result.Removed {
					printHuman(w, "Removed  %s/%s (source gone)\n", d.AssetType, d.AssetName)
				}
				for _, w := range result.Warnings {
					printHuman(cmd.ErrOrStderr(), "Warning: %s\n", w)
				}
				if len(result.Repaired) == 0 && len(result.Removed) == 0 {
					printHuman(w, "All deployments healthy.\n")
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&sourceID, "source", "", "sync a specific git source")
	return cmd
}
