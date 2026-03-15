package cmd

import (
	"github.com/spf13/cobra"
)

func newUninstallCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Remove all nd-managed symlinks and optionally config",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil // TODO: implement
		},
	}
}
