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

// mockServices is a configurable test double for the Services interface.
// All methods return sensible zero values by default. Override individual
// function fields to inject custom behavior in tests.
type mockServices struct {
	sourceManagerFn  func() (*sourcemanager.SourceManager, error)
	scanIndexFn      func() (*sourcemanager.ScanSummary, error)
	agentRegistryFn  func() (*agent.Registry, error)
	defaultAgentFn   func() (*agent.Agent, error)
	deployEngineFn   func() (*deploy.Engine, error)
	stateStoreFn     func() *state.Store
	profileManagerFn func() (*profile.Manager, error)
	profileStoreFn   func() (*profile.Store, error)
	opLogFn          func() *oplog.Writer
	getScopeFn       func() nd.Scope
	getConfigPathFn  func() string
	isDryRunFn       func() bool
	resetForScopeFn  func(scope nd.Scope, projectRoot string)

	// Track ResetForScope calls for assertions.
	resetCalls []resetForScopeCall
}

type resetForScopeCall struct {
	Scope       nd.Scope
	ProjectRoot string
}

func newMockServices() *mockServices {
	return &mockServices{}
}

func (m *mockServices) SourceManager() (*sourcemanager.SourceManager, error) {
	if m.sourceManagerFn != nil {
		return m.sourceManagerFn()
	}
	return nil, nil
}

func (m *mockServices) ScanIndex() (*sourcemanager.ScanSummary, error) {
	if m.scanIndexFn != nil {
		return m.scanIndexFn()
	}
	return &sourcemanager.ScanSummary{}, nil
}

func (m *mockServices) AgentRegistry() (*agent.Registry, error) {
	if m.agentRegistryFn != nil {
		return m.agentRegistryFn()
	}
	return nil, nil
}

func (m *mockServices) DefaultAgent() (*agent.Agent, error) {
	if m.defaultAgentFn != nil {
		return m.defaultAgentFn()
	}
	return &agent.Agent{Name: "claude-code"}, nil
}

func (m *mockServices) DeployEngine() (*deploy.Engine, error) {
	if m.deployEngineFn != nil {
		return m.deployEngineFn()
	}
	return nil, nil
}

func (m *mockServices) StateStore() *state.Store {
	if m.stateStoreFn != nil {
		return m.stateStoreFn()
	}
	return nil
}

func (m *mockServices) ProfileManager() (*profile.Manager, error) {
	if m.profileManagerFn != nil {
		return m.profileManagerFn()
	}
	return nil, nil
}

func (m *mockServices) ProfileStore() (*profile.Store, error) {
	if m.profileStoreFn != nil {
		return m.profileStoreFn()
	}
	return nil, nil
}

func (m *mockServices) OpLog() *oplog.Writer {
	if m.opLogFn != nil {
		return m.opLogFn()
	}
	return nil
}

func (m *mockServices) GetScope() nd.Scope {
	if m.getScopeFn != nil {
		return m.getScopeFn()
	}
	return nd.ScopeGlobal
}

func (m *mockServices) GetConfigPath() string {
	if m.getConfigPathFn != nil {
		return m.getConfigPathFn()
	}
	return "/tmp/nd-test/config.yaml"
}

func (m *mockServices) IsDryRun() bool {
	if m.isDryRunFn != nil {
		return m.isDryRunFn()
	}
	return false
}

func (m *mockServices) ResetForScope(scope nd.Scope, projectRoot string) {
	m.resetCalls = append(m.resetCalls, resetForScopeCall{
		Scope:       scope,
		ProjectRoot: projectRoot,
	})
	if m.resetForScopeFn != nil {
		m.resetForScopeFn(scope, projectRoot)
	}
}
