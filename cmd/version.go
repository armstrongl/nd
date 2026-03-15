package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/larah/nd/internal/version"
)

func newVersionCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print nd version information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if app.JSON {
				return printJSON(os.Stdout, map[string]string{
					"version": version.Version,
					"commit":  version.Commit,
					"date":    version.Date,
				}, false)
			}
			printHuman(os.Stdout, "%s\n", version.String())
			return nil
		},
	}
	return cmd
}
