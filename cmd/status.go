package cmd

import (
	"github.com/spf13/cobra"
)

func newStatusCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show deployment status and health",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil // TODO: implement
		},
	}
}
