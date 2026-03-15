package sourcemanager

import (
	"fmt"
	"path/filepath"

	"github.com/larah/nd/internal/config"
	"github.com/larah/nd/internal/source"
)

// SourceManager owns the full source lifecycle: config, registration,
// scanning, and sync.
type SourceManager struct {
	configPath string
	sourcesDir string // derived from configPath: <configDir>/sources/
	projectDir string
	cfg        config.Config
}

// New creates a SourceManager by loading the global config and optionally
// merging a project config. If the global config file does not exist,
// defaults are used (first-run experience).
func New(configPath string, projectDir string) (*SourceManager, error) {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	if projectDir != "" {
		projectConfigPath := filepath.Join(projectDir, ".nd", "config.yaml")
		pc, err := LoadProjectConfig(projectConfigPath)
		if err != nil {
			return nil, fmt.Errorf("load project config: %w", err)
		}
		cfg = MergeConfigs(cfg, pc)
	}

	return &SourceManager{
		configPath: configPath,
		sourcesDir: filepath.Join(filepath.Dir(configPath), "sources"),
		projectDir: projectDir,
		cfg:        cfg,
	}, nil
}

// Config returns the current merged configuration.
func (sm *SourceManager) Config() *config.Config {
	return &sm.cfg
}

// Sources returns all registered sources with availability status.
func (sm *SourceManager) Sources() []source.Source {
	sources := make([]source.Source, len(sm.cfg.Sources))
	for i, entry := range sm.cfg.Sources {
		sources[i] = source.Source{
			ID:    entry.ID,
			Type:  entry.Type,
			Path:  entry.Path,
			Alias: entry.Alias,
			Order: i,
		}
	}
	return sources
}
