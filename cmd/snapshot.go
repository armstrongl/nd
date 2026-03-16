package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/profile"
	"github.com/spf13/cobra"
)

func newSnapshotCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "Manage deployment snapshots",
		Long:  "Save, restore, list, and delete point-in-time deployment snapshots.",
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
	return &cobra.Command{
		Use:   "restore <name>",
		Short: "Restore deployments from a snapshot",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()
			name := args[0]

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
}

func newSnapshotListCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all snapshots",
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
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a snapshot",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()
			name := args[0]

			pstore, err := app.ProfileStore()
			if err != nil {
				return err
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
}
