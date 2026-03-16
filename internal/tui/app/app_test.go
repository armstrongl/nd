package tuiapp_test

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/armstrongl/nd/internal/agent"
	"github.com/armstrongl/nd/internal/asset"
	"github.com/armstrongl/nd/internal/deploy"
	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/profile"
	"github.com/armstrongl/nd/internal/source"
	"github.com/armstrongl/nd/internal/sourcemanager"
	"github.com/armstrongl/nd/internal/state"
	"github.com/armstrongl/nd/internal/tui"
	tuiapp "github.com/armstrongl/nd/internal/tui/app"
)

// --- Mock services ---

type mockDeployer struct {
	statusEntries []deploy.StatusEntry
	healthChecks  []state.HealthCheck
	syncResult    *deploy.SyncResult
	deployResult  *deploy.DeployResult
	deployErr     error
	removeErr     error
	statusErr     error
	checkErr      error
	syncErr       error
}

func (m *mockDeployer) Deploy(_ deploy.DeployRequest) (*deploy.DeployResult, error) {
	return m.deployResult, m.deployErr
}

func (m *mockDeployer) DeployBulk(_ []deploy.DeployRequest) (*deploy.BulkDeployResult, error) {
	return &deploy.BulkDeployResult{}, nil
}

func (m *mockDeployer) Remove(_ deploy.RemoveRequest) error {
	return m.removeErr
}

func (m *mockDeployer) RemoveBulk(_ []deploy.RemoveRequest) (*deploy.BulkRemoveResult, error) {
	return &deploy.BulkRemoveResult{}, nil
}

func (m *mockDeployer) Status() ([]deploy.StatusEntry, error) {
	return m.statusEntries, m.statusErr
}

func (m *mockDeployer) Check() ([]state.HealthCheck, error) {
	return m.healthChecks, m.checkErr
}

func (m *mockDeployer) Sync() (*deploy.SyncResult, error) {
	return m.syncResult, m.syncErr
}

func (m *mockDeployer) SetOrigin(_ asset.Identity, _ nd.Scope, _ string, _ nd.DeployOrigin) error {
	return nil
}

type mockProfileSwitcher struct {
	active    string
	profiles  []profile.ProfileSummary
	snapshots []profile.SnapshotSummary
	activeErr error
}

func (m *mockProfileSwitcher) ActiveProfile() (string, error) {
	return m.active, m.activeErr
}

func (m *mockProfileSwitcher) Switch(_, _ string) (*profile.SwitchResult, error) {
	return &profile.SwitchResult{}, nil
}

func (m *mockProfileSwitcher) Restore(_ string) (*profile.RestoreResult, error) {
	return &profile.RestoreResult{}, nil
}

func (m *mockProfileSwitcher) SaveSnapshot(_ string) error {
	return nil
}

func (m *mockProfileSwitcher) ListProfiles() ([]profile.ProfileSummary, error) {
	return m.profiles, nil
}

func (m *mockProfileSwitcher) ListSnapshots() ([]profile.SnapshotSummary, error) {
	return m.snapshots, nil
}

type mockSourceScanner struct {
	sources []source.Source
}

func (m *mockSourceScanner) Sources() []source.Source {
	return m.sources
}

func (m *mockSourceScanner) Scan() (*sourcemanager.ScanSummary, error) {
	return &sourcemanager.ScanSummary{}, nil
}

func (m *mockSourceScanner) SyncSource(_ string) error {
	return nil
}

type mockAgentDetector struct {
	agents []agent.Agent
}

func (m *mockAgentDetector) Detect() agent.DetectionResult {
	return agent.DetectionResult{Agents: m.agents}
}

func (m *mockAgentDetector) Default() (*agent.Agent, error) {
	if len(m.agents) == 0 {
		return nil, fmt.Errorf("no agents detected")
	}
	return &m.agents[0], nil
}

func (m *mockAgentDetector) All() []agent.Agent {
	return m.agents
}

// --- Helper to create a default App for testing ---

func newTestApp() tuiapp.App {
	return tuiapp.New(
		&mockDeployer{
			syncResult: &deploy.SyncResult{},
		},
		&mockProfileSwitcher{
			active: "default",
		},
		&mockSourceScanner{},
		&mockAgentDetector{
			agents: []agent.Agent{
				{Name: "Claude Code", Detected: true},
			},
		},
		false,
		func() (string, error) { return "/tmp/project", nil },
	)
}

func appView(a tuiapp.App) string {
	return a.View().Content
}

// --- Tests ---

func TestNewAppStartsInPickerState(t *testing.T) {
	app := newTestApp()
	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(tuiapp.App)
	view := appView(a)
	if !strings.Contains(view, "Setup") {
		t.Errorf("expected picker view on init, got:\n%s", view)
	}
}

func TestAppInit(t *testing.T) {
	app := newTestApp()
	cmd := app.Init()
	if cmd == nil {
		t.Error("Init should return a command for agent detection")
	}
}

func TestAppWindowResize(t *testing.T) {
	app := newTestApp()
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	model, _ := app.Update(msg)
	a := model.(tuiapp.App)
	view := appView(a)
	if view == "" {
		t.Error("view should not be empty after resize")
	}
}

func TestAppTooSmallTerminal(t *testing.T) {
	app := newTestApp()
	msg := tea.WindowSizeMsg{Width: 30, Height: 5}
	model, _ := app.Update(msg)
	a := model.(tuiapp.App)
	view := appView(a)
	if !strings.Contains(view, "too small") {
		t.Errorf("expected too small message, got:\n%s", view)
	}
}

func TestAppPickerToMenu(t *testing.T) {
	app := newTestApp()
	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	// Press Enter to confirm picker (single agent auto-selects)
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	view := model.(tuiapp.App).View().Content
	if !strings.Contains(view, "Main Menu") {
		t.Errorf("expected menu after picker confirmation, got:\n%s", view)
	}
}

func TestAppMenuToDashboard(t *testing.T) {
	app := newTestApp()
	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // picker -> menu
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // menu -> dashboard
	view := model.(tuiapp.App).View().Content
	if !strings.Contains(view, "No assets deployed") && !strings.Contains(view, "Overview") {
		t.Errorf("expected dashboard view, got:\n%s", view)
	}
}

func TestAppDashboardEscToMenu(t *testing.T) {
	app := newTestApp()
	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // picker -> menu
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // menu -> dashboard
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	view := model.(tuiapp.App).View().Content
	if !strings.Contains(view, "Main Menu") {
		t.Errorf("expected menu after esc from dashboard, got:\n%s", view)
	}
}

func TestAppMenuQuit(t *testing.T) {
	app := newTestApp()
	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // picker -> menu
	_, cmd := model.Update(tea.KeyPressMsg{Text: "q"})
	if cmd == nil {
		t.Error("q in menu should produce a quit command")
	}
}

func TestAppToastMsg(t *testing.T) {
	app := newTestApp()
	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // picker -> menu
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // menu -> dashboard

	model, _ = model.Update(tui.ToastMsg{Message: "Test toast", Level: tui.ToastSuccess})
	view := model.(tuiapp.App).View().Content
	if !strings.Contains(view, "Test toast") {
		t.Errorf("expected toast in view, got:\n%s", view)
	}
}

func TestAppToastDismiss(t *testing.T) {
	app := newTestApp()
	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	model, _ = model.Update(tui.ToastMsg{Message: "Temp", Level: tui.ToastInfo})
	model, _ = model.Update(tui.ToastDismissMsg{})
	view := model.(tuiapp.App).View().Content
	if strings.Contains(view, "Temp") {
		t.Error("toast should be dismissed")
	}
}

func TestAppStatusResultMsg(t *testing.T) {
	app := newTestApp()
	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	entries := []deploy.StatusEntry{
		{
			Deployment: state.Deployment{
				AssetName: "test-skill",
				AssetType: nd.AssetSkill,
				Scope:     nd.ScopeGlobal,
			},
			Health: state.HealthOK,
		},
	}
	model, _ = model.Update(tui.StatusResultMsg{Entries: entries})
	view := model.(tuiapp.App).View().Content
	if strings.Contains(view, "No assets deployed") {
		t.Error("table should have rows after status result")
	}
}

func TestAppTabNavigation(t *testing.T) {
	app := newTestApp()
	model, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // picker
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // menu -> dashboard

	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	view := model.(tuiapp.App).View().Content
	if view == "" {
		t.Error("view should not be empty after tab switch")
	}
}
