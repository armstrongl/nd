package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/armstrongl/nd/internal/agent"
	"github.com/armstrongl/nd/internal/builtin"
	"github.com/armstrongl/nd/internal/config"
	"github.com/armstrongl/nd/internal/deploy"
	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/sourcemanager"
	"github.com/armstrongl/nd/internal/state"
	"github.com/spf13/cobra"
)

func newInitCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:  "init",
		Short: "Initialize nd configuration",
		Long:  "Interactive walkthrough to set up nd for the first time.\n\nCreates the config directory structure, writes a default config file, and\ndeploys built-in assets (skills, commands, agents) to your coding agent's\nconfig directory. Use --yes to skip the deploy confirmation prompt.",
		Example: `  # Interactive setup
  nd init

  # Non-interactive setup (skip prompts)
  nd init --yes`,
		Annotations: map[string]string{
			"docs.guides": "getting-started,configuration",
		},
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

			if !app.JSON && !app.Quiet {
				printHuman(w, "Initialized nd at %s\n", configDir)
			}

			// Deploy built-in assets
			deployed, deployErr := deployBuiltinAssets(cmd, app, configDir, app.initAgent)

			if app.JSON {
				result := map[string]interface{}{
					"config_path": configPath,
					"config_dir":  configDir,
				}
				if deployed > 0 {
					result["builtin_deployed"] = deployed
				}
				return printJSON(w, result, false)
			}

			return deployErr
		},
	}
}

// deployBuiltinAssets extracts and deploys all built-in assets during init.
// If ag is nil, the agent is auto-detected from a fresh config. Pass a non-nil
// agent in tests to control the deploy target directory.
// Returns the number of deployed assets and any error.
func deployBuiltinAssets(cmd *cobra.Command, app *App, configDir string, ag *agent.Agent) (int, error) {
	w := cmd.OutOrStdout()

	// Extract the builtin cache
	builtinPath, err := builtin.Path()
	if err != nil {
		if !app.Quiet {
			printHuman(cmd.ErrOrStderr(), "warning: could not extract built-in assets: %v\n", err)
		}
		return 0, nil
	}

	// Scan the builtin source for assets
	scanResult := sourcemanager.ScanSource(nd.BuiltinSourceID, builtinPath)
	if len(scanResult.Assets) == 0 {
		return 0, nil
	}

	// Build the summary for the prompt
	typeCounts := make(map[nd.AssetType]int)
	for _, a := range scanResult.Assets {
		typeCounts[a.Type]++
	}

	total := len(scanResult.Assets)
	promptMsg := fmt.Sprintf("Deploy %d built-in asset(s)", total)
	parts := buildAssetCountParts(typeCounts)
	if len(parts) > 0 {
		promptMsg += " (" + joinParts(parts) + ")"
	}
	promptMsg += "?"

	// Decide whether to deploy
	shouldDeploy := app.Yes || app.JSON
	if !shouldDeploy {
		confirmed, err := confirm(cmd.InOrStdin(), w, promptMsg, false)
		if err != nil {
			// Non-interactive: default to deploying
			shouldDeploy = true
		} else {
			shouldDeploy = confirmed
		}
	}

	if !shouldDeploy {
		if !app.Quiet && !app.JSON {
			printHuman(w, "Skipped. Deploy later with 'nd deploy --source builtin'\n")
		}
		return 0, nil
	}

	// Auto-detect agent if not provided
	if ag == nil {
		cfg := config.Config{
			Version:         1,
			DefaultScope:    nd.ScopeGlobal,
			DefaultAgent:    "claude-code",
			SymlinkStrategy: nd.SymlinkAbsolute,
		}
		reg := agent.New(cfg)
		detected, err := reg.Default()
		if err != nil {
			if !app.Quiet {
				printHuman(cmd.ErrOrStderr(), "warning: no agent detected, skipping built-in deploy: %v\n", err)
			}
			return 0, nil
		}
		ag = detected
	}

	// Create the deploy engine
	sstore := state.NewStore(filepath.Join(configDir, "state", "deployments.yaml"))
	backupDir := filepath.Join(configDir, "backups")
	eng := deploy.New(sstore, ag, backupDir)

	// Build deploy requests for all assets
	reqs := make([]deploy.DeployRequest, len(scanResult.Assets))
	for i, a := range scanResult.Assets {
		reqs[i] = deploy.DeployRequest{
			Asset:    a,
			Scope:    nd.ScopeGlobal,
			Origin:   nd.OriginManual,
			Strategy: nd.SymlinkAbsolute,
		}
	}

	bulkResult, err := eng.DeployBulk(reqs)
	if err != nil {
		return 0, fmt.Errorf("deploy built-in assets: %w", err)
	}

	deployed := len(bulkResult.Succeeded)
	if !app.Quiet && !app.JSON {
		printHuman(w, "Deployed %d built-in asset(s)\n", deployed)
		for _, f := range bulkResult.Failed {
			printHuman(cmd.ErrOrStderr(), "Failed: %s/%s: %v\n", f.AssetType, f.AssetName, f.Err)
		}
	}

	return deployed, nil
}

// buildAssetCountParts builds description fragments like "3 skills", "2 commands".
func buildAssetCountParts(counts map[nd.AssetType]int) []string {
	// Use a fixed order to keep output deterministic
	order := []nd.AssetType{
		nd.AssetSkill, nd.AssetCommand, nd.AssetAgent, nd.AssetRule,
		nd.AssetOutputStyle, nd.AssetContext, nd.AssetPlugin, nd.AssetHook,
	}
	var parts []string
	for _, t := range order {
		if n, ok := counts[t]; ok && n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", n, t))
		}
	}
	return parts
}

// joinParts joins string slices with ", ".
func joinParts(parts []string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += ", "
		}
		result += p
	}
	return result
}
