package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/armstrongl/nd/internal/agent"
	"github.com/armstrongl/nd/internal/deploy"
	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/oplog"
	"github.com/armstrongl/nd/internal/profile"
	"github.com/armstrongl/nd/internal/sourcemanager"
	"github.com/armstrongl/nd/internal/state"
)

// App holds configuration derived from flags and lazily initialized services.
type App struct {
	// Set from flags/env in root PersistentPreRunE
	ConfigPath  string
	Scope       nd.Scope
	ProjectRoot string
	BackupDir   string
	Verbose     bool
	Quiet       bool
	JSON        bool
	DryRun      bool
	NoColor     bool
	Yes         bool

	// Lazily initialized
	sm      *sourcemanager.SourceManager
	reg     *agent.Registry
	eng     *deploy.Engine
	profMgr *profile.Manager
	pstore  *profile.Store
	sstore  *state.Store
	ol      *oplog.Writer
}

// SourceManager returns the source manager, creating it on first call.
func (a *App) SourceManager() (*sourcemanager.SourceManager, error) {
	if a.sm != nil {
		return a.sm, nil
	}
	sm, err := sourcemanager.New(a.ConfigPath, a.ProjectRoot)
	if err != nil {
		return nil, fmt.Errorf("init source manager: %w", err)
	}
	a.sm = sm
	return a.sm, nil
}

// AgentRegistry returns the agent registry, creating it on first call.
func (a *App) AgentRegistry() (*agent.Registry, error) {
	if a.reg != nil {
		return a.reg, nil
	}
	sm, err := a.SourceManager()
	if err != nil {
		return nil, err
	}
	a.reg = agent.New(*sm.Config())
	return a.reg, nil
}

// DefaultAgent returns the default detected agent.
func (a *App) DefaultAgent() (*agent.Agent, error) {
	reg, err := a.AgentRegistry()
	if err != nil {
		return nil, err
	}
	return reg.Default()
}

// DeployEngine returns the deploy engine, creating it on first call.
func (a *App) DeployEngine() (*deploy.Engine, error) {
	if a.eng != nil {
		return a.eng, nil
	}
	ag, err := a.DefaultAgent()
	if err != nil {
		return nil, err
	}
	sstore := a.StateStore()
	eng := deploy.New(sstore, ag, a.BackupDir)

	// Wire auto-snapshot saver
	pstore, err := a.ProfileStore()
	if err == nil {
		eng.SetSnapshotSaver(pstore)
	}

	a.eng = eng
	return a.eng, nil
}

// ProfileManager returns the profile manager, creating it on first call.
func (a *App) ProfileManager() (*profile.Manager, error) {
	if a.profMgr != nil {
		return a.profMgr, nil
	}
	pstore, err := a.ProfileStore()
	if err != nil {
		return nil, err
	}
	a.profMgr = profile.NewManager(pstore, a.StateStore())
	return a.profMgr, nil
}

// ProfileStore returns the profile store, creating it on first call.
func (a *App) ProfileStore() (*profile.Store, error) {
	if a.pstore != nil {
		return a.pstore, nil
	}
	configDir := filepath.Dir(a.ConfigPath)
	a.pstore = profile.NewStore(
		filepath.Join(configDir, "profiles"),
		filepath.Join(configDir, "snapshots"),
	)
	return a.pstore, nil
}

// StateStore returns the state store, creating it on first call.
func (a *App) StateStore() *state.Store {
	if a.sstore != nil {
		return a.sstore
	}
	configDir := filepath.Dir(a.ConfigPath)
	a.sstore = state.NewStore(filepath.Join(configDir, "state", "deployments.yaml"))
	return a.sstore
}

// ResolveProjectRoot finds the project root when scope is project.
// Uses nd.FindProjectRoot from cwd if ProjectRoot is not already set.
func (a *App) ResolveProjectRoot() (string, error) {
	if a.ProjectRoot != "" {
		return a.ProjectRoot, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	root, err := nd.FindProjectRoot(cwd)
	if err != nil {
		return "", fmt.Errorf("find project root: %w", err)
	}
	a.ProjectRoot = root
	return root, nil
}

// OpLog returns the operation log writer, creating it on first call.
func (a *App) OpLog() *oplog.Writer {
	if a.ol != nil {
		return a.ol
	}
	configDir := filepath.Dir(a.ConfigPath)
	a.ol = oplog.NewWriter(filepath.Join(configDir, "logs"))
	return a.ol
}

// LogOp logs an operation entry best-effort (errors are silently ignored).
func (a *App) LogOp(entry oplog.LogEntry) {
	_ = a.OpLog().Log(entry)
}

// GetScope returns the current deployment scope.
// Named GetScope (not Scope) to avoid collision with the Scope field.
func (a *App) GetScope() nd.Scope { return a.Scope }

// IsDryRun returns whether dry-run mode is active.
// Named IsDryRun (not DryRun) to avoid collision with the DryRun field.
func (a *App) IsDryRun() bool { return a.DryRun }

// GetConfigPath returns the path to the config file.
// Named GetConfigPath (not ConfigPath) to avoid collision with the ConfigPath field.
func (a *App) GetConfigPath() string { return a.ConfigPath }

// GetProjectRoot returns the resolved project root path.
// Named GetProjectRoot (not ProjectRoot) to avoid collision with the ProjectRoot field.
func (a *App) GetProjectRoot() string { return a.ProjectRoot }

// ResetForScope nils all cached services so they reinitialize for a new scope.
// Used by the TUI when switching between global and project scope mid-session.
func (a *App) ResetForScope(scope nd.Scope, projectRoot string) {
	a.Scope = scope
	a.ProjectRoot = projectRoot
	a.sm = nil
	a.reg = nil
	a.eng = nil
	a.profMgr = nil
	a.pstore = nil
	a.sstore = nil
	a.ol = nil
}

// ScanIndex scans all sources and returns the scan summary.
func (a *App) ScanIndex() (*sourcemanager.ScanSummary, error) {
	sm, err := a.SourceManager()
	if err != nil {
		return nil, err
	}
	return sm.Scan()
}
