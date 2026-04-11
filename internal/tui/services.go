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
	StateStore() *state.Store

	// Profiles & snapshots
	ProfileManager() (*profile.Manager, error)
	ProfileStore() (*profile.Store, error)

	// Operation logging
	OpLog() *oplog.Writer

	// Display state — named to avoid collision with App field names
	GetScope() nd.Scope
	GetConfigPath() string
	GetProjectRoot() string
	IsDryRun() bool

	// Mid-session reset (scope/agent switching).
	// Nils all cached services so they reinitialize for the new scope.
	ResetForScope(scope nd.Scope, projectRoot string)
}
