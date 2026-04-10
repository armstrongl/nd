package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func newSettingsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "settings",
		Short:   "Manage nd settings",
		Example: `  nd settings edit`,
		Annotations: map[string]string{
			"docs.guides": "configuration",
		},
	}

	cmd.AddCommand(newSettingsEditCmd(app))
	return cmd
}

func newSettingsEditCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "Open settings in your editor",
		Example: `  # Open config in your default editor
  nd settings edit`,
		Annotations: map[string]string{
			"docs.guides": "configuration,getting-started,troubleshooting,asset-types/hooks,asset-types/output-styles",
		},
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()
			configPath := app.ConfigPath

			// Check config exists
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				return fmt.Errorf("config not found at %s; run 'nd init' first", configPath)
			}

			if app.DryRun {
				printHuman(w, "[dry-run] would open %s in editor\n", configPath)
				return nil
			}

			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = os.Getenv("VISUAL")
			}
			if editor == "" {
				editor = "vi"
			}

			editorCmd := exec.Command(editor, configPath)
			editorCmd.Stdin = os.Stdin
			editorCmd.Stdout = os.Stdout
			editorCmd.Stderr = os.Stderr
			return editorCmd.Run()
		},
	}
}
