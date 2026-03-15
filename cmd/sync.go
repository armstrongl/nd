package cmd

import (
	"github.com/spf13/cobra"
)

func newSyncCmd(app *App) *cobra.Command {
	var sourceID string

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Repair symlinks and optionally pull git sources",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = sourceID
			return nil // TODO: implement
		},
	}
	cmd.Flags().StringVar(&sourceID, "source", "", "sync a specific git source")
	return cmd
}
