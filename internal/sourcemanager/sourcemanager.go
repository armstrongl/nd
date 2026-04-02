package sourcemanager

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/armstrongl/nd/internal/asset"
	"github.com/armstrongl/nd/internal/builtin"
	"github.com/armstrongl/nd/internal/config"
	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/source"
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

	appendBuiltinSource(&cfg)

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
			URL:   entry.URL,
			Alias: entry.Alias,
			Order: i,
		}
	}
	return sources
}

// SyncSource pulls updates for a Git source. Returns an error if the source
// is not found or is not a Git source. Uses --ff-only to avoid merge commits.
func (sm *SourceManager) SyncSource(sourceID string) error {
	var entry *config.SourceEntry
	for i := range sm.cfg.Sources {
		if sm.cfg.Sources[i].ID == sourceID {
			entry = &sm.cfg.Sources[i]
			break
		}
	}
	if entry == nil {
		return fmt.Errorf("source %q not found", sourceID)
	}

	if entry.Type != nd.SourceGit {
		return fmt.Errorf("source %q is type %q, not git", sourceID, entry.Type)
	}

	return gitPull(entry.Path)
}

// appendBuiltinSource adds the built-in source as the last (lowest priority)
// entry in cfg.Sources. If the cache extraction fails, a warning is printed
// to stderr but execution continues.
func appendBuiltinSource(cfg *config.Config) {
	cachePath, err := builtin.Path()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: builtin source unavailable: %v\n", err)
		return
	}

	cfg.Sources = append(cfg.Sources, config.SourceEntry{
		ID:    nd.BuiltinSourceID,
		Type:  nd.SourceBuiltin,
		Path:  cachePath,
		Alias: "nd",
	})
}

// ScanSummary holds the result of a full scan across all sources.
type ScanSummary struct {
	Index    *asset.Index
	Warnings []string
	Errors   []error
}

// Scan discovers all assets across all registered sources and builds an index.
// Unavailable sources produce warnings but do not fail the scan (NFR-006).
func (sm *SourceManager) Scan() (*ScanSummary, error) {
	var allAssets []asset.Asset
	var allWarnings []string
	var allErrors []error

	for _, entry := range sm.cfg.Sources {
		result := ScanSource(entry.ID, entry.Path)
		allAssets = append(allAssets, result.Assets...)
		allWarnings = append(allWarnings, result.Warnings...)
		allErrors = append(allErrors, result.Errors...)
	}

	return &ScanSummary{
		Index:    asset.NewIndex(allAssets),
		Warnings: allWarnings,
		Errors:   allErrors,
	}, nil
}
