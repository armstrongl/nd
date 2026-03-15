package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newInitCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize nd configuration",
		Long:  "Interactive walkthrough to set up nd for the first time.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()
			configPath := app.ConfigPath
			configDir := filepath.Dir(configPath)

			// Check if config already exists
			if _, err := os.Stat(configPath); err == nil {
				return fmt.Errorf("config already exists at %s; edit with 'nd settings edit'", configPath)
			}

			// Create directory structure
			dirs := []string{
				configDir,
				filepath.Join(configDir, "profiles"),
				filepath.Join(configDir, "snapshots"),
				filepath.Join(configDir, "snapshots", "user"),
				filepath.Join(configDir, "snapshots", "auto"),
				filepath.Join(configDir, "state"),
			}
			for _, dir := range dirs {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					return fmt.Errorf("create directory %s: %w", dir, err)
				}
			}

			// Write default config
			defaultCfg := `version: 1
default_scope: global
default_agent: claude-code
symlink_strategy: absolute
sources: []
`
			if err := os.WriteFile(configPath, []byte(defaultCfg), 0o644); err != nil {
				return fmt.Errorf("write config: %w", err)
			}

			if app.JSON {
				return printJSON(w, map[string]string{
					"config_path": configPath,
					"config_dir":  configDir,
				}, false)
			}
			if !app.Quiet {
				printHuman(w, "Initialized nd at %s\n", configDir)
			}
			return nil
		},
	}
}
