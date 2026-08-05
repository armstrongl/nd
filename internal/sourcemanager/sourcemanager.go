package sourcemanager

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/armstrongl/nd/internal/asset"
	"github.com/armstrongl/nd/internal/builtin"
	"github.com/armstrongl/nd/internal/config"
	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/source"
)

// SourceManager owns the full source lifecycle: config, registration,
// scanning, and sync.
type SourceManager struct {
	configPath        string
	sourcesDir        string // derived from configPath: <configDir>/sources/
	projectDir        string
	projectConfigPath string // <projectDir>/.nd/config.yaml; empty when projectDir is ""

	mu          sync.Mutex    // guards cfg and the observed-stat fields below
	cfg         config.Config // the current merged config (global + project + builtin)
	configStat  statInfo      // last observed on-disk state of configPath
	projectStat statInfo      // last observed on-disk state of projectConfigPath
}

// statInfo captures the mtime and size of a config file so external edits can
// be detected cheaply. A missing (or un-stat-able) file is the zero value.
type statInfo struct {
	mtime time.Time
	size  int64
}

// statConfigFile stats path, returning the zero statInfo when the path is empty
// or the file cannot be stat'd (treated as "no file observed").
func statConfigFile(path string) statInfo {
	if path == "" {
		return statInfo{}
	}
	info, err := os.Stat(path)
	if err != nil {
		return statInfo{}
	}
	return statInfo{mtime: info.ModTime(), size: info.Size()}
}

// equal reports whether two observations describe the same on-disk state.
func (s statInfo) equal(other statInfo) bool {
	return s.size == other.size && s.mtime.Equal(other.mtime)
}

// loadMergedConfig runs the config-loading pipeline: global config, optional
// project overlay, then the injected builtin source. Shared by New and
// reloadConfigIfChanged so both yield an identically-shaped cfg.
func loadMergedConfig(configPath, projectConfigPath string) (config.Config, error) {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return config.Config{}, fmt.Errorf("load config: %w", err)
	}

	if projectConfigPath != "" {
		pc, err := LoadProjectConfig(projectConfigPath)
		if err != nil {
			return config.Config{}, fmt.Errorf("load project config: %w", err)
		}
		cfg = MergeConfigs(cfg, pc)
	}

	appendBuiltinSource(&cfg)
	return cfg, nil
}

// New creates a SourceManager by loading the global config and optionally
// merging a project config. If the global config file does not exist,
// defaults are used (first-run experience).
func New(configPath string, projectDir string) (*SourceManager, error) {
	projectConfigPath := ""
	if projectDir != "" {
		projectConfigPath = filepath.Join(projectDir, ".nd", "config.yaml")
	}

	cfg, err := loadMergedConfig(configPath, projectConfigPath)
	if err != nil {
		return nil, err
	}

	return &SourceManager{
		configPath:        configPath,
		sourcesDir:        filepath.Join(filepath.Dir(configPath), "sources"),
		projectDir:        projectDir,
		projectConfigPath: projectConfigPath,
		cfg:               cfg,
		configStat:        statConfigFile(configPath),
		projectStat:       statConfigFile(projectConfigPath),
	}, nil
}

// reloadConfigIfChanged re-stats the global and project config files and, when
// either changed since last observed, re-runs the load pipeline and replaces
// sm.cfg. This lets a long-lived process (the TUI, built from one memoized
// SourceManager) pick up an external edit — e.g. a concurrent `nd source
// add/remove` in another shell — without a restart. The in-process mutators
// (AddLocal/AddGit/Remove) call recordConfigStat after their own WriteConfig so
// the file they just wrote reads as already observed: that both avoids a
// redundant reload and prevents a reload from clobbering the in-memory mutation.
//
// The caller must hold sm.mu.
func (sm *SourceManager) reloadConfigIfChanged() error {
	global := statConfigFile(sm.configPath)
	project := statConfigFile(sm.projectConfigPath)

	if global.equal(sm.configStat) && project.equal(sm.projectStat) {
		return nil
	}

	cfg, err := loadMergedConfig(sm.configPath, sm.projectConfigPath)
	if err != nil {
		return err
	}

	sm.cfg = cfg
	sm.configStat = global
	sm.projectStat = project
	return nil
}

// recordConfigStat re-observes the global config file after a self-initiated
// WriteConfig. Only the global file is updated: the mutators never write the
// project config, so an un-observed external project-config edit must still be
// detected by the next reloadConfigIfChanged. Caller must hold sm.mu.
func (sm *SourceManager) recordConfigStat() {
	sm.configStat = statConfigFile(sm.configPath)
}

// Config returns a snapshot copy of the current merged configuration.
//
// It deliberately does NOT reload from disk; it reflects the config as of the
// last New or Scan. Callers needing the freshest source set after an external
// edit must go through Scan/ScanIndex, which reloads internally. Returning a
// copy (rather than a live pointer) keeps reads safe against a concurrent Scan
// that replaces sm.cfg. No caller mutates through this pointer.
func (sm *SourceManager) Config() *config.Config {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	c := sm.cfg
	return &c
}

// Sources returns all registered sources with availability status.
func (sm *SourceManager) Sources() []source.Source {
	sm.mu.Lock()
	defer sm.mu.Unlock()
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
	sm.mu.Lock()
	var (
		path  string
		typ   nd.SourceType
		found bool
	)
	for i := range sm.cfg.Sources {
		if sm.cfg.Sources[i].ID == sourceID {
			path = sm.cfg.Sources[i].Path
			typ = sm.cfg.Sources[i].Type
			found = true
			break
		}
	}
	sm.mu.Unlock()

	if !found {
		return fmt.Errorf("source %q not found", sourceID)
	}
	if typ != nd.SourceGit {
		return fmt.Errorf("source %q is type %q, not git", sourceID, typ)
	}

	return gitPull(path)
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
//
// Before scanning it reloads the config if the underlying file(s) changed, so a
// long-lived process observes external edits. The source set is snapshotted
// under the lock; the filesystem walk then runs unlocked (ScanSource re-walks
// each source every call, so in-source file additions are always reflected —
// there is deliberately no per-source mtime short-circuit).
func (sm *SourceManager) Scan() (*ScanSummary, error) {
	sm.mu.Lock()
	if err := sm.reloadConfigIfChanged(); err != nil {
		sm.mu.Unlock()
		return nil, fmt.Errorf("reload config: %w", err)
	}
	entries := make([]config.SourceEntry, len(sm.cfg.Sources))
	copy(entries, sm.cfg.Sources)
	sm.mu.Unlock()

	var allAssets []asset.Asset
	var allWarnings []string
	var allErrors []error

	for _, entry := range entries {
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
