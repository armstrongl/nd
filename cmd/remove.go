package cmd

import (
	"github.com/spf13/cobra"
)

func newRemoveCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <asset> [assets...]",
		Short: "Remove deployed assets",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil // TODO: implement
		},
	}
}
