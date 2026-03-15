package cmd

import (
	"github.com/spf13/cobra"
)

func newPinCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "pin <asset>",
		Short: "Pin an asset to prevent profile switches from removing it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil // TODO: implement
		},
	}
}

func newUnpinCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "unpin <asset>",
		Short: "Unpin an asset, allowing profile switches to manage it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil // TODO: implement
		},
	}
}
