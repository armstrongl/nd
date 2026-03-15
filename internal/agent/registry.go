package agent

import (
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/larah/nd/internal/config"
)

// Registry manages agent detection, lookup, and config override application.
type Registry struct {
	agents     []Agent
	defaultCfg string
	detected   bool
	lookPath   func(string) (string, error)
	stat       func(string) (os.FileInfo, error)
}

// New creates a Registry with known agent definitions and applies config overrides.
func New(cfg config.Config) *Registry {
	homeDir := "~"
	if u, err := user.Current(); err == nil {
		homeDir = u.HomeDir
	}

	agents := []Agent{
		{
			Name:       "claude-code",
			GlobalDir:  filepath.Join(homeDir, ".claude"),
			ProjectDir: ".claude",
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
			}
		}
	}

	return &Registry{
		agents:     agents,
		defaultCfg: cfg.DefaultAgent,
		lookPath:   exec.LookPath,
		stat:       os.Stat,
	}
}

// All returns all known agents (detected or not).
func (r *Registry) All() []Agent {
	result := make([]Agent, len(r.agents))
	copy(result, r.agents)
	return result
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
