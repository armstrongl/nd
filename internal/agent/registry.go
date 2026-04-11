package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/armstrongl/nd/internal/config"
	"github.com/armstrongl/nd/internal/nd"
)

// Registry manages agent detection, lookup, and config override application.
type Registry struct {
	agents     []Agent
	defaultCfg string
	detected   bool
	lookPath   func(string) (string, error)
	stat       func(string) (os.FileInfo, error)
	runCommand func(name string, args ...string) ([]byte, error) // nil = skip binary verification
}

// New creates a Registry with known agent definitions and applies config overrides.
func New(cfg config.Config) *Registry {
	homeDir := "~"
	if u, err := user.Current(); err == nil {
		homeDir = u.HomeDir
	}

	agents := []Agent{
		{
			Name:                "claude-code",
			GlobalDir:           filepath.Join(homeDir, ".claude"),
			ProjectDir:          ".claude",
			SourceAlias:         "claude",
			Binary:              "claude",
			SupportedTypes:      nd.DeployableAssetTypes(),
			DefaultContextFile:  "",
			ContextInProjectDir: false,
			VersionPattern:      `(?i)claude`,
		},
		{
			Name:                "copilot",
			GlobalDir:           filepath.Join(homeDir, ".copilot"),
			ProjectDir:          ".github",
			SourceAlias:         "copilot",
			Binary:              "copilot",
			SupportedTypes:      []nd.AssetType{nd.AssetSkill, nd.AssetAgent, nd.AssetContext},
			DefaultContextFile:  "copilot-instructions.md",
			ContextInProjectDir: true,
			VersionPattern:      `(?i)copilot|github\.copilot`,
		},
	}

	for i := range agents {
		for _, override := range cfg.Agents {
			if override.Name == agents[i].Name {
				if override.GlobalDir != "" {
					agents[i].GlobalDir = expandHome(override.GlobalDir, homeDir)
				}
				if override.ProjectDir != "" {
					agents[i].ProjectDir = override.ProjectDir
				}
				if override.SourceAlias != "" {
					agents[i].SourceAlias = override.SourceAlias
				}
			}
		}
	}

	return &Registry{
		agents:     agents,
		defaultCfg: cfg.DefaultAgent,
		lookPath:   exec.LookPath,
		stat:       os.Stat,
		runCommand: defaultRunCommand,
	}
}

// SetLookPath replaces the PATH lookup function (for testing).
func (r *Registry) SetLookPath(fn func(string) (string, error)) {
	r.lookPath = fn
}

// SetStat replaces the filesystem stat function (for testing).
func (r *Registry) SetStat(fn func(string) (os.FileInfo, error)) {
	r.stat = fn
}

// SetRunCommand replaces the binary execution function (for testing).
// Set to nil to skip binary verification (legacy behavior).
func (r *Registry) SetRunCommand(fn func(name string, args ...string) ([]byte, error)) {
	r.runCommand = fn
}

// All returns all known agents (detected or not).
func (r *Registry) All() []Agent {
	result := make([]Agent, len(r.agents))
	copy(result, r.agents)
	return result
}

// defaultRunCommand executes a binary with a 5-second timeout, capturing both stdout and stderr.
func defaultRunCommand(name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

// Detect probes the system for installed agents (PATH + config dir + binary verification).
// Safe to call multiple times (subsequent calls are no-ops).
func (r *Registry) Detect() DetectionResult {
	if r.detected {
		return DetectionResult{Agents: r.All()}
	}

	var warnings []string
	anyDetected := false

	for i := range r.agents {
		binary := r.agents[i].Binary
		if binary != "" {
			if _, err := r.lookPath(binary); err == nil {
				// Binary found in PATH — verify identity via version output
				if r.verifyBinary(binary, r.agents[i].VersionPattern) {
					r.agents[i].InPath = true
				}
				// If verification fails, InPath stays false (name collision)
			}
		}

		dirExists := false
		if _, err := r.stat(r.agents[i].GlobalDir); err == nil {
			dirExists = true
		}

		r.agents[i].Detected = r.agents[i].InPath || dirExists
		if r.agents[i].Detected {
			anyDetected = true
		}
	}

	if !anyDetected {
		warnings = append(warnings,
			"No coding agents detected. Install Claude Code or configure a custom agent path in config.yaml (see: nd settings edit).")
	}

	r.detected = true
	return DetectionResult{Agents: r.All(), Warnings: warnings}
}

// verifyBinary runs "<binary> --version" and checks if output matches the agent's version pattern.
// Returns true if runCommand is nil (skip verification) or if the pattern matches.
// Returns false on any error, timeout, or pattern mismatch.
func (r *Registry) verifyBinary(binary, pattern string) bool {
	if r.runCommand == nil || pattern == "" {
		return true // no verification configured — trust lookPath result
	}
	output, err := r.runCommand(binary, "--version")
	if err != nil {
		return false
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	return re.Match(output)
}

// Get returns the agent with the given name, or an error if not found.
func (r *Registry) Get(name string) (*Agent, error) {
	for i := range r.agents {
		if r.agents[i].Name == name {
			return &r.agents[i], nil
		}
	}
	return nil, fmt.Errorf("agent %q not found", name)
}

// Default returns the default agent: the one named in config.DefaultAgent if
// detected, otherwise the first detected agent, otherwise an error.
// Calls Detect() automatically if not already called.
func (r *Registry) Default() (*Agent, error) {
	if !r.detected {
		r.Detect()
	}

	if r.defaultCfg != "" {
		for i := range r.agents {
			if r.agents[i].Name == r.defaultCfg && r.agents[i].Detected {
				return &r.agents[i], nil
			}
		}
	}

	for i := range r.agents {
		if r.agents[i].Detected {
			return &r.agents[i], nil
		}
	}

	return nil, fmt.Errorf("no coding agents detected; install Claude Code or configure a custom agent path in config.yaml")
}

func expandHome(path, homeDir string) string {
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(homeDir, path[2:])
	}
	if path == "~" {
		return homeDir
	}
	return path
}
