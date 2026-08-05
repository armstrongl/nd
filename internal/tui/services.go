package tui

import (
	"github.com/armstrongl/nd/internal/agent"
	"github.com/armstrongl/nd/internal/deploy"
	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/oplog"
	"github.com/armstrongl/nd/internal/profile"
	"github.com/armstrongl/nd/internal/sourcemanager"
	"github.com/armstrongl/nd/internal/state"
)

// Services provides access to nd's service layer.
// cmd.App satisfies this interface with small additions (GetScope, IsDryRun,
// GetConfigPath, ResetForScope).
type Services interface {
	// Source management
	SourceManager() (*sourcemanager.SourceManager, error)
	ScanIndex() (*sourcemanager.ScanSummary, error)

	// Agent management
	AgentRegistry() (*agent.Registry, error)
	ActiveAgent() (*agent.Agent, error)

	// Deployment
	DeployEngine() (*deploy.Engine, error)
	DeployEngineFor(ag *agent.Agent) (*deploy.Engine, error)
	StateStore() *state.Store

	// Profiles & snapshots
	ProfileManager() (*profile.Manager, error)
	ProfileStore() (*profile.Store, error)

	// Operation logging
	OpLog() *oplog.Writer

	// Display state — named to avoid collision with App field names
	GetScope() nd.Scope
	GetConfigPath() string
	// GetProjectRoot returns the project root, resolving it on demand from cwd
	// when it was not populated at launch (see ResolveProjectRoot). Returns ""
	// when cwd is not inside a project; callers treat "" as "not in a project".
	GetProjectRoot() string
	// ResolveProjectRoot discovers the project root on demand (from cwd via
	// nd.FindProjectRoot) when it was not populated at launch, e.g. when the
	// TUI is started in the default global scope from inside a project.
	ResolveProjectRoot() (string, error)
	IsDryRun() bool

	// Mid-session reset (scope/agent switching).
	// Nils all cached services so they reinitialize for the new scope.
	ResetForScope(scope nd.Scope, projectRoot string)
}
