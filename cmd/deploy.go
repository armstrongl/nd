package cmd

import (
	"github.com/spf13/cobra"
)

func newDeployCmd(app *App) *cobra.Command {
	var assetType string

	cmd := &cobra.Command{
		Use:   "deploy <asset> [assets...]",
		Short: "Deploy assets by creating symlinks",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = assetType
			return nil // TODO: implement
		},
	}
	cmd.Flags().StringVar(&assetType, "type", "", "asset type filter (skills, commands, rules, etc.)")
	return cmd
}
