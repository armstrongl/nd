package cmd

import (
	"github.com/spf13/cobra"
)

func newListCmd(app *App) *cobra.Command {
	var (
		assetType  string
		sourceID   string
		pattern    string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available assets from all sources",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = assetType
			_ = sourceID
			_ = pattern
			return nil // TODO: implement
		},
	}
	cmd.Flags().StringVar(&assetType, "type", "", "filter by asset type")
	cmd.Flags().StringVar(&sourceID, "source", "", "filter by source ID")
	cmd.Flags().StringVar(&pattern, "pattern", "", "filter by name pattern")
	return cmd
}
