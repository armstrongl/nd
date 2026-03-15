package cmd

import (
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
			return nil // TODO: implement
		},
	}
}

func newSnapshotRestoreCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "restore <name>",
		Short: "Restore deployments from a snapshot",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil // TODO: implement
		},
	}
}

func newSnapshotListCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all snapshots",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil // TODO: implement
		},
	}
}

func newSnapshotDeleteCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a snapshot",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil // TODO: implement
		},
	}
}
