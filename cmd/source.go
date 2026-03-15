package cmd

import (
	"github.com/spf13/cobra"
)

func newSourceCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "source",
		Short: "Manage asset sources",
		Long:  "Add, remove, and list asset source directories.",
	}

	cmd.AddCommand(
		newSourceAddCmd(app),
		newSourceRemoveCmd(app),
		newSourceListCmd(app),
	)

	return cmd
}

func newSourceAddCmd(app *App) *cobra.Command {
	var alias string

	cmd := &cobra.Command{
		Use:   "add <path|url>",
		Short: "Register a new asset source",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = alias
			return nil // TODO: implement
		},
	}
	cmd.Flags().StringVar(&alias, "alias", "", "human-readable alias for the source")
	return cmd
}

func newSourceRemoveCmd(app *App) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "remove <source-id>",
		Short: "Remove a registered source",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = force
			return nil // TODO: implement
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "skip confirmation")
	return cmd
}

func newSourceListCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List registered sources",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil // TODO: implement
		},
	}
}
