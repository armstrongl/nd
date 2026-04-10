package cmd

import (
	"fmt"
	"time"

	"github.com/armstrongl/nd/internal/oplog"
	"github.com/spf13/cobra"
)

func newSyncCmd(app *App) *cobra.Command {
	var sourceID string

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Repair symlinks and optionally pull git sources",
		Example: `  # Repair all broken symlinks
  nd sync

  # Pull and repair a specific git source
  nd sync --source my-git-source

  # Preview what would be repaired
  nd sync --dry-run`,
		Annotations: map[string]string{
			"docs.guides":  "getting-started,creating-sources,troubleshooting",
			"docs.related": "nd source add,nd status",
		},
		Args: cobra.NoArgs,
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
					app.LogOp(oplog.LogEntry{
						Timestamp: time.Now(),
						Operation: oplog.OpSourceSync,
						Succeeded: 1,
						Detail:    sourceID,
					})
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

			app.LogOp(oplog.LogEntry{
				Timestamp: time.Now(),
				Operation: oplog.OpSync,
				Succeeded: len(result.Repaired),
				Failed:    len(result.Removed),
				Detail:    fmt.Sprintf("%d repaired, %d removed", len(result.Repaired), len(result.Removed)),
			})

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
				for _, warn := range result.Warnings {
					printHuman(cmd.ErrOrStderr(), "Warning: %s\n", warn)
				}
				if len(result.Repaired) == 0 && len(result.Removed) == 0 {
					printHuman(w, "All deployments healthy.\n")
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&sourceID, "source", "", "sync a specific git source")
	cmd.RegisterFlagCompletionFunc("source", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeSourceIDs(app, toComplete)
	})
	return cmd
}
