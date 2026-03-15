package cmd

import (
	"github.com/spf13/cobra"
)

func newDoctorCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check nd configuration and deployment health",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil // TODO: implement
		},
	}
}
