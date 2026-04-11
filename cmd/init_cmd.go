package cmd

import (
	"fmt"
	"io"
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

			// Check if config already exists
			if _, err := os.Stat(app.ConfigPath); err == nil {
				return fmt.Errorf("config already exists at %s; edit with 'nd settings edit'", app.ConfigPath)
			}

			configDir, err := runInitSetup(cmd, app)
			if err != nil {
				return err
			}

			// Detect agents and display status
			reg := initRegistry(app)
			detResult := reg.Detect()
			agentStatus := agentDetectionMap(detResult)

			if !app.JSON && !app.Quiet {
				displayAgentDetection(w, detResult)
			}

			// Deploy built-in assets using the registry
			deployed, deployErr := deployBuiltinAssets(cmd, app, configDir, reg, app.initAgent)

			if app.JSON {
				result := map[string]interface{}{
					"config_path":     app.ConfigPath,
					"config_dir":      configDir,
					"agents_detected": agentStatus,
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

// runInitSetup creates the config directory structure and writes the default
// config file. Returns the config directory path. This is shared between the
// init command and the first-run prompt in persistentPreRun.
func runInitSetup(cmd *cobra.Command, app *App) (string, error) {
	configPath := app.ConfigPath
	configDir := filepath.Dir(configPath)

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
			return "", fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	defaultCfg := `version: 1
default_scope: global
default_agent: claude-code
symlink_strategy: absolute
sources: []
`
	if err := os.WriteFile(configPath, []byte(defaultCfg), 0o644); err != nil {
		return "", fmt.Errorf("write config: %w", err)
	}

	if !app.JSON && !app.Quiet {
		printHuman(cmd.OutOrStdout(), "Initialized nd at %s\n", configDir)
	}

	return configDir, nil
}

// initRegistry returns the agent registry for the init command.
// Uses app.initRegistry if set (tests), otherwise creates a fresh registry
// from the default config values.
func initRegistry(app *App) *agent.Registry {
	if app.initRegistry != nil {
		return app.initRegistry
	}
	cfg := config.Config{
		Version:         1,
		DefaultScope:    nd.ScopeGlobal,
		DefaultAgent:    "claude-code",
		SymlinkStrategy: nd.SymlinkAbsolute,
	}
	return agent.New(cfg)
}

// displayAgentDetection prints the agent detection status line to the writer.
// Format: "Detected agents: claude-code (yes), copilot (no)"
func displayAgentDetection(w io.Writer, result agent.DetectionResult) {
	line := "Detected agents:"
	for i, ag := range result.Agents {
		status := "yes"
		if !ag.Detected {
			status = "no"
		}
		if i > 0 {
			line += fmt.Sprintf(", %s (%s)", ag.Name, status)
		} else {
			line += fmt.Sprintf(" %s (%s)", ag.Name, status)
		}
	}
	printHuman(w, "%s\n", line)
}

// agentDetectionMap builds a map of agent name -> detected (bool) for JSON output.
func agentDetectionMap(result agent.DetectionResult) map[string]bool {
	m := make(map[string]bool, len(result.Agents))
	for _, ag := range result.Agents {
		m[ag.Name] = ag.Detected
	}
	return m
}

// deployBuiltinAssets extracts and deploys all built-in assets during init.
// Uses the provided registry to resolve the default agent. If ag is non-nil,
// it overrides the registry's default (test use only).
// Returns the number of deployed assets and any error.
func deployBuiltinAssets(cmd *cobra.Command, app *App, configDir string, reg *agent.Registry, ag *agent.Agent) (int, error) {
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

	// Auto-detect agent from registry if not provided
	if ag == nil {
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
