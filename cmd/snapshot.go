package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/oplog"
	"github.com/armstrongl/nd/internal/profile"
	"github.com/spf13/cobra"
)

func newSnapshotCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "Manage deployment snapshots",
		Long:  "Save, restore, list, and delete point-in-time deployment snapshots.",
		Example: `  nd snapshot save before-update
  nd snapshot list
  nd snapshot restore before-update`,
		Annotations: map[string]string{
			"docs.guides": "profiles-and-snapshots",
		},
	}

	cmd.AddCommand(
		newSnapshotSaveCmd(app),
		newSnapshotRestoreCmd(app),
		newSnapshotListCmd(app),
		newSnapshotDeleteCmd(app),
	)

	return cmd
}

func newSnapshotSaveCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "save <name>",
		Short: "Save current deployments as a named snapshot",
		Example: `  # Save current deployments as a snapshot
  nd snapshot save before-update`,
		Annotations: map[string]string{
			"docs.guides": "profiles-and-snapshots,getting-started",
		},
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()
			name := args[0]

			pstore, err := app.ProfileStore()
			if err != nil {
				return err
			}

			sstore := app.StateStore()
			st, _, err := sstore.Load()
			if err != nil {
				return fmt.Errorf("load deployment state: %w", err)
			}

			entries := profile.DeploymentsToEntries(st.Deployments)
			snap := profile.Snapshot{
				Version:     nd.SchemaVersion,
				Name:        name,
				CreatedAt:   time.Now().Truncate(time.Second),
				Auto:        false,
				Deployments: entries,
			}

			if err := pstore.SaveSnapshot(snap); err != nil {
				return err
			}

			app.LogOp(oplog.LogEntry{
				Timestamp: time.Now().Truncate(time.Second),
				Operation: oplog.OpSnapshotSave,
				Succeeded: len(entries),
				Detail:    name,
			})

			if app.JSON {
				return printJSON(w, snap, app.DryRun)
			}
			if !app.Quiet {
				printHuman(w, "Saved snapshot %q with %d deployments.\n", name, len(entries))
			}
			return nil
		},
	}
}

func newSnapshotRestoreCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore <name>",
		Short: "Restore deployments from a snapshot",
		Example: `  # Restore deployments from a snapshot
  nd snapshot restore before-update

  # Preview what would change
  nd snapshot restore before-update --dry-run`,
		Annotations: map[string]string{
			"docs.guides": "profiles-and-snapshots",
		},
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()

			var name string
			if len(args) > 0 {
				name = args[0]
			} else {
				if app.JSON {
					return fmt.Errorf("requires a snapshot name argument; run 'nd snapshot list --json' to see snapshots")
				}
				if !isTerminal() {
					return fmt.Errorf("requires a snapshot name argument; run 'nd snapshot list' to see snapshots")
				}
				completionInitApp(app)
				completions, _ := completeSnapshotNames(app, "")
				if len(completions) == 0 {
					return fmt.Errorf("no snapshots to restore")
				}
				names := extractChoiceNames(completions)
				choice, err := promptChoice(cmd.InOrStdin(), w, "Select snapshot to restore:", names)
				if err != nil {
					return err
				}
				name = choice
			}

			pstore, err := app.ProfileStore()
			if err != nil {
				return err
			}

			// Try user snapshot first, then auto
			snap, err := pstore.GetSnapshot(name, false)
			if err != nil {
				snap, err = pstore.GetSnapshot(name, true)
				if err != nil {
					return fmt.Errorf("snapshot %q not found", name)
				}
			}

			if app.DryRun {
				if app.JSON {
					return printJSON(w, snap, true)
				}
				for _, e := range snap.Deployments {
					printHuman(w, "[dry-run] would restore %s/%s from %s\n", e.AssetType, e.AssetName, e.SourceID)
				}
				return nil
			}

			if !app.Quiet {
				printHuman(w, "Will restore %d deployments from snapshot %q.\n", len(snap.Deployments), name)
			}
			ok, err := confirm(cmd.InOrStdin(), w, "Proceed with restore?", app.Yes)
			if err != nil {
				return err
			}
			if !ok {
				printHuman(w, "Restore cancelled.\n")
				return nil
			}

			profMgr, err := app.ProfileManager()
			if err != nil {
				return err
			}

			summary, err := app.ScanIndex()
			if err != nil {
				return fmt.Errorf("scan sources: %w", err)
			}

			eng, err := app.DeployEngine()
			if err != nil {
				return fmt.Errorf("init deploy engine: %w", err)
			}

			result, err := profMgr.Restore(name, eng, summary.Index)
			if err != nil {
				return err
			}

			{
				succeeded, failed := 0, len(result.MissingAssets)
				if result.Deployed != nil {
					succeeded += len(result.Deployed.Succeeded)
					failed += len(result.Deployed.Failed)
				}
				app.LogOp(oplog.LogEntry{
					Timestamp: time.Now().Truncate(time.Second),
					Operation: oplog.OpSnapshotRestore,
					Succeeded: succeeded,
					Failed:    failed,
					Detail:    name,
				})
			}

			if app.JSON {
				return printJSON(w, result, false)
			}

			if !app.Quiet {
				if result.Removed != nil {
					for _, s := range result.Removed.Succeeded {
						printHuman(w, "Removed %s/%s\n", s.Identity.Type, s.Identity.Name)
					}
				}
				if result.Deployed != nil {
					for _, s := range result.Deployed.Succeeded {
						printHuman(w, "Restored %s/%s\n", s.Deployment.AssetType, s.Deployment.AssetName)
					}
				}
				for _, m := range result.MissingAssets {
					printHuman(cmd.ErrOrStderr(), "Warning: asset %s/%s not found in sources\n", m.AssetType, m.AssetName)
				}
				printHuman(w, "Snapshot %q restored.\n", name)
			}
			return nil
		},
	}
	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeSnapshotNames(app, toComplete)
	}
	return cmd
}

func newSnapshotListCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all snapshots",
		Example: `  # List all snapshots
  nd snapshot list

  # Output as JSON
  nd snapshot list --json`,
		Annotations: map[string]string{
			"docs.guides": "profiles-and-snapshots",
		},
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()

			pstore, err := app.ProfileStore()
			if err != nil {
				return err
			}

			snapshots, err := pstore.ListSnapshots()
			if err != nil {
				return err
			}

			if app.JSON {
				return printJSON(w, snapshots, false)
			}

			if len(snapshots) == 0 {
				printHuman(w, "No snapshots found.\n")
				return nil
			}

			for _, s := range snapshots {
				autoTag := ""
				if s.Auto {
					autoTag = " (auto)"
				}
				printHuman(w, "  %-40s %d deployments  %s%s\n",
					s.Name, s.DeploymentCount,
					s.CreatedAt.Format("2006-01-02 15:04"), autoTag)
			}
			return nil
		},
	}
}

func newSnapshotDeleteCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a snapshot",
		Example: `  # Delete a snapshot
  nd snapshot delete before-update

  # Delete without confirmation
  nd snapshot delete before-update --yes`,
		Annotations: map[string]string{
			"docs.guides": "profiles-and-snapshots",
		},
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()

			var name string
			if len(args) > 0 {
				name = args[0]
			} else {
				if app.JSON {
					return fmt.Errorf("requires a snapshot name argument; run 'nd snapshot list --json' to see snapshots")
				}
				if !isTerminal() {
					return fmt.Errorf("requires a snapshot name argument; run 'nd snapshot list' to see snapshots")
				}
				completionInitApp(app)
				completions, _ := completeSnapshotNames(app, "")
				if len(completions) == 0 {
					return fmt.Errorf("no snapshots to delete")
				}
				names := extractChoiceNames(completions)
				choice, err := promptChoice(cmd.InOrStdin(), w, "Select snapshot to delete:", names)
				if err != nil {
					return err
				}
				name = choice
			}

			pstore, err := app.ProfileStore()
			if err != nil {
				return err
			}

			if !app.Quiet {
				printHuman(w, "Will delete snapshot %q.\n", name)
			}
			ok, err := confirm(cmd.InOrStdin(), w, "Proceed?", app.Yes)
			if err != nil {
				return err
			}
			if !ok {
				printHuman(w, "Cancelled.\n")
				return nil
			}

			// Try user snapshot first, then auto
			err = pstore.DeleteSnapshot(name, false)
			if err != nil {
				if !strings.Contains(err.Error(), "not found") {
					return err
				}
				err = pstore.DeleteSnapshot(name, true)
				if err != nil {
					return fmt.Errorf("snapshot %q not found", name)
				}
			}

			if app.JSON {
				return printJSON(w, map[string]string{"deleted": name}, false)
			}
			if !app.Quiet {
				printHuman(w, "Deleted snapshot %q.\n", name)
			}
			return nil
		},
	}
	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeSnapshotNames(app, toComplete)
	}
	return cmd
}
