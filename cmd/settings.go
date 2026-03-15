package cmd

import (
	"github.com/spf13/cobra"
)

func newSettingsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "settings",
		Short: "Manage nd settings",
	}

	cmd.AddCommand(newSettingsEditCmd(app))
	return cmd
}

func newSettingsEditCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "Open settings in your editor",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil // TODO: implement
		},
	}
}
