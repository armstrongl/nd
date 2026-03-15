package sourcemanager

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/larah/nd/internal/config"
	"github.com/larah/nd/internal/nd"
)

// DefaultConfig returns a Config with built-in defaults.
func DefaultConfig() config.Config {
	return config.Config{
		Version:         nd.SchemaVersion,
		DefaultScope:    nd.ScopeGlobal,
		DefaultAgent:    "claude-code",
		SymlinkStrategy: nd.SymlinkAbsolute,
		Sources:         []config.SourceEntry{},
	}
}

// LoadConfig reads and validates a config file. If the file does not exist,
// returns defaults (first-run experience).
func LoadConfig(path string) (config.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return DefaultConfig(), nil
		}
		return config.Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg config.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return config.Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}

	if errs := cfg.Validate(); len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Error()
		}
		return config.Config{}, errors.New(strings.Join(msgs, "; "))
	}

	return cfg, nil
}

// LoadProjectConfig reads a project-level config file.
// Returns nil if the file does not exist.
func LoadProjectConfig(path string) (*config.ProjectConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read project config: %w", err)
	}

	var pc config.ProjectConfig
	if err := yaml.Unmarshal(data, &pc); err != nil {
		return nil, fmt.Errorf("parse project config %s: %w", path, err)
	}

	return &pc, nil
}

// MergeConfigs merges a global config with an optional project config.
// Project fields override global when non-nil. Sources are appended
// (global first for priority per FR-016a). Agent overrides from project
// replace global entries by agent name.
func MergeConfigs(global config.Config, project *config.ProjectConfig) config.Config {
	if project == nil {
		return global
	}

	merged := global

	if project.DefaultScope != nil {
		merged.DefaultScope = *project.DefaultScope
	}
	if project.DefaultAgent != nil {
		merged.DefaultAgent = *project.DefaultAgent
	}
	if project.SymlinkStrategy != nil {
		merged.SymlinkStrategy = *project.SymlinkStrategy
	}

	if len(project.Sources) > 0 {
		combined := make([]config.SourceEntry, 0, len(merged.Sources)+len(project.Sources))
		combined = append(combined, merged.Sources...)
		combined = append(combined, project.Sources...)
		merged.Sources = combined
	}

	if len(project.Agents) > 0 {
		agentMap := make(map[string]config.AgentOverride)
		for _, a := range merged.Agents {
			agentMap[a.Name] = a
		}
		for _, a := range project.Agents {
			agentMap[a.Name] = a
		}
		names := make([]string, 0, len(agentMap))
		for name := range agentMap {
			names = append(names, name)
		}
		sort.Strings(names)
		merged.Agents = make([]config.AgentOverride, 0, len(agentMap))
		for _, name := range names {
			merged.Agents = append(merged.Agents, agentMap[name])
		}
	}

	return merged
}

// WriteConfig writes a config to disk using atomic writes (NFR-010).
func WriteConfig(path string, cfg config.Config) error {
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return nd.AtomicWrite(path, data)
}
