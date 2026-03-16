package tuiapp_test

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/armstrongl/nd/internal/agent"
	"github.com/armstrongl/nd/internal/deploy"
	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/profile"
	"github.com/armstrongl/nd/internal/state"
	"github.com/armstrongl/nd/internal/tui"
	tuiapp "github.com/armstrongl/nd/internal/tui/app"
)

// sendKeys sends a sequence of key events through the model update chain.
func sendKeys(m tea.Model, keys ...tea.Msg) tea.Model {
	for _, k := range keys {
		m, _ = m.Update(k)
	}
	return m
}

// setupDashboard creates an App and navigates through picker→menu→dashboard.
func setupDashboard(t *testing.T) tea.Model {
	t.Helper()
	app := newTestApp()
	m := sendKeys(app,
		tea.WindowSizeMsg{Width: 120, Height: 40},
		tea.KeyPressMsg{Code: tea.KeyEnter}, // picker → menu
		tea.KeyPressMsg{Code: tea.KeyEnter}, // menu → dashboard
	)
	return m
}

// setupDashboardWithData creates a dashboard and populates it with status entries.
func setupDashboardWithData(t *testing.T) tea.Model {
	t.Helper()
	m := setupDashboard(t)
	entries := []deploy.StatusEntry{
		{
			Deployment: state.Deployment{
				SourceID:  "local-1",
				AssetName: "go-backend",
				AssetType: nd.AssetSkill,
				Scope:     nd.ScopeGlobal,
				Origin:    nd.OriginManual,
			},
			Health: state.HealthOK,
		},
		{
			Deployment: state.Deployment{
				SourceID:  "local-1",
				AssetName: "review-agent",
				AssetType: nd.AssetAgent,
				Scope:     nd.ScopeGlobal,
				Origin:    nd.OriginManual,
			},
			Health: state.HealthOK,
		},
		{
			Deployment: state.Deployment{
				SourceID:  "local-1",
				AssetName: "broken-skill",
				AssetType: nd.AssetSkill,
				Scope:     nd.ScopeGlobal,
				Origin:    nd.OriginManual,
			},
			Health: state.HealthBroken,
		},
	}
	m, _ = m.Update(tui.StatusResultMsg{Entries: entries})
	return m
}

// --- Integration tests ---

func TestIntegration_LaunchFlow(t *testing.T) {
	// picker → menu → dashboard
	app := newTestApp()
	m := sendKeys(app,
		tea.WindowSizeMsg{Width: 100, Height: 30},
	)
	view := m.(tuiapp.App).View().Content
	if !strings.Contains(view, "Setup") {
		t.Error("should start in picker")
	}

	m = sendKeys(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	view = m.(tuiapp.App).View().Content
	if !strings.Contains(view, "Main Menu") {
		t.Error("should be in menu after picker")
	}

	m = sendKeys(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	view = m.(tuiapp.App).View().Content
	if !strings.Contains(view, "No assets deployed") {
		t.Error("should show empty dashboard")
	}
}

func TestIntegration_DashboardWithData(t *testing.T) {
	m := setupDashboardWithData(t)
	view := m.(tuiapp.App).View().Content

	if strings.Contains(view, "No assets deployed") {
		t.Error("should not show empty state with data")
	}
	if !strings.Contains(view, "go-backend") {
		t.Error("should show go-backend skill")
	}
	if !strings.Contains(view, "broken-skill") {
		t.Error("should show broken-skill")
	}
}

func TestIntegration_TabNavigation(t *testing.T) {
	m := setupDashboardWithData(t)

	// Navigate to Skills tab (tab index 1)
	m = sendKeys(m, tea.KeyPressMsg{Code: tea.KeyRight})
	view := m.(tuiapp.App).View().Content
	// Skills tab should filter to only skills
	if strings.Contains(view, "review-agent") {
		t.Error("Skills tab should not show agents")
	}

	// Navigate to Agents tab (tab index 2)
	m = sendKeys(m, tea.KeyPressMsg{Code: tea.KeyRight})
	view = m.(tuiapp.App).View().Content
	if strings.Contains(view, "go-backend") {
		t.Error("Agents tab should not show skills")
	}
}

func TestIntegration_DashboardEscBackToMenu(t *testing.T) {
	m := setupDashboard(t)
	m = sendKeys(m, tea.KeyPressMsg{Code: tea.KeyEscape})
	view := m.(tuiapp.App).View().Content
	if !strings.Contains(view, "Main Menu") {
		t.Error("esc should return to menu")
	}
}

func TestIntegration_DeployFlow(t *testing.T) {
	m := setupDashboardWithData(t)

	// Press d to open fuzzy finder
	m = sendKeys(m, tea.KeyPressMsg{Text: "d"})
	view := m.(tuiapp.App).View().Content
	if !strings.Contains(view, "Deploy Asset") {
		t.Error("d should open fuzzy finder")
	}

	// Esc to cancel
	m = sendKeys(m, tea.KeyPressMsg{Code: tea.KeyEscape})
	view = m.(tuiapp.App).View().Content
	if strings.Contains(view, "Deploy Asset") {
		t.Error("esc should close fuzzy finder")
	}
}

func TestIntegration_RemoveFlow(t *testing.T) {
	m := setupDashboardWithData(t)

	// Press r to start remove
	m = sendKeys(m, tea.KeyPressMsg{Text: "r"})
	view := m.(tuiapp.App).View().Content
	if !strings.Contains(view, "Remove") {
		t.Error("r should open confirm dialog")
	}

	// Press n to cancel
	m = sendKeys(m, tea.KeyPressMsg{Text: "n"})
	view = m.(tuiapp.App).View().Content
	if strings.Contains(view, "y/n?") {
		t.Error("n should close confirm dialog")
	}
}

func TestIntegration_SyncFlow(t *testing.T) {
	m := setupDashboard(t)

	// Press s to sync — this triggers an async command
	_, cmd := m.Update(tea.KeyPressMsg{Text: "s"})
	if cmd == nil {
		t.Error("s should trigger sync command")
	}
}

func TestIntegration_FixFlow(t *testing.T) {
	m := setupDashboard(t)

	_, cmd := m.Update(tea.KeyPressMsg{Text: "f"})
	if cmd == nil {
		t.Error("f should trigger fix/check command")
	}
}

func TestIntegration_ProfileSwitchFlow(t *testing.T) {
	// Create app with profiles
	app := tuiapp.New(
		&mockDeployer{syncResult: &deploy.SyncResult{}},
		&mockProfileSwitcher{
			active: "default",
			profiles: []profile.ProfileSummary{
				{Name: "default", AssetCount: 5},
				{Name: "go-dev", AssetCount: 12},
			},
		},
		&mockSourceScanner{},
		&mockAgentDetector{
			agents: []agent.Agent{{Name: "Claude Code", Detected: true}},
		},
		false,
		func() (string, error) { return "/tmp/project", nil },
	)
	m := sendKeys(app,
		tea.WindowSizeMsg{Width: 120, Height: 40},
		tea.KeyPressMsg{Code: tea.KeyEnter}, // picker
		tea.KeyPressMsg{Code: tea.KeyEnter}, // menu
	)

	// Press P to switch profile
	m = sendKeys(m, tea.KeyPressMsg{Text: "P"})
	view := m.(tuiapp.App).View().Content
	if !strings.Contains(view, "Switch Profile") {
		t.Errorf("P should open profile picker, got:\n%s", view)
	}
	if !strings.Contains(view, "default") {
		t.Error("should show default profile")
	}
	if !strings.Contains(view, "go-dev") {
		t.Error("should show go-dev profile")
	}

	// Esc to cancel
	m = sendKeys(m, tea.KeyPressMsg{Code: tea.KeyEscape})
	view = m.(tuiapp.App).View().Content
	if strings.Contains(view, "Switch Profile") {
		t.Error("esc should close profile picker")
	}
}

func TestIntegration_SnapshotSaveFlow(t *testing.T) {
	m := setupDashboard(t)

	// Press W to open snapshot prompt
	m = sendKeys(m, tea.KeyPressMsg{Text: "W"})
	view := m.(tuiapp.App).View().Content
	if !strings.Contains(view, "Save Snapshot") {
		t.Errorf("W should open snapshot prompt, got:\n%s", view)
	}

	// Type a name
	m = sendKeys(m,
		tea.KeyPressMsg{Text: "t"},
		tea.KeyPressMsg{Text: "e"},
		tea.KeyPressMsg{Text: "s"},
		tea.KeyPressMsg{Text: "t"},
	)

	// Enter to save — should produce a command
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Error("enter with name should trigger save command")
	}
}

func TestIntegration_EmptyStates(t *testing.T) {
	m := setupDashboard(t)
	view := m.(tuiapp.App).View().Content
	if !strings.Contains(view, "No assets deployed") {
		t.Error("empty dashboard should show no assets message")
	}
}

func TestIntegration_ResponsiveLayout(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
	}{
		{"large", 200, 50},
		{"medium", 80, 24},
		{"small", 60, 15},
		{"minimum", 40, 10},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app := newTestApp()
			m := sendKeys(app,
				tea.WindowSizeMsg{Width: tc.width, Height: tc.height},
				tea.KeyPressMsg{Code: tea.KeyEnter},
				tea.KeyPressMsg{Code: tea.KeyEnter},
			)
			view := m.(tuiapp.App).View().Content
			if view == "" {
				t.Errorf("%s: view should not be empty", tc.name)
			}
		})
	}
}

func TestIntegration_TooSmallTerminal(t *testing.T) {
	app := newTestApp()
	m := sendKeys(app, tea.WindowSizeMsg{Width: 20, Height: 5})
	view := m.(tuiapp.App).View().Content
	if !strings.Contains(view, "too small") {
		t.Error("should show too small message")
	}
}

func TestIntegration_ToastLifecycle(t *testing.T) {
	m := setupDashboard(t)

	// Show toast
	m, _ = m.Update(tui.ToastMsg{Message: "hello", Level: tui.ToastSuccess})
	view := m.(tuiapp.App).View().Content
	if !strings.Contains(view, "hello") {
		t.Error("toast should be visible")
	}

	// Dismiss
	m, _ = m.Update(tui.ToastDismissMsg{})
	view = m.(tuiapp.App).View().Content
	if strings.Contains(view, "hello") {
		t.Error("toast should be dismissed")
	}
}

func TestIntegration_StatusResultError(t *testing.T) {
	m := setupDashboard(t)
	m, _ = m.Update(tui.StatusResultMsg{
		Err: errMock("status failed"),
	})
	view := m.(tuiapp.App).View().Content
	if !strings.Contains(view, "status failed") {
		t.Error("should show error toast")
	}
}

func TestIntegration_SyncResultMsg(t *testing.T) {
	m := setupDashboard(t)
	m, _ = m.Update(tui.SyncResultMsg{
		Result: &deploy.SyncResult{
			Repaired: []state.Deployment{{AssetName: "a"}},
			Removed:  []state.Deployment{{AssetName: "b"}, {AssetName: "c"}},
		},
	})
	view := m.(tuiapp.App).View().Content
	if !strings.Contains(view, "1 repaired") {
		t.Error("should show sync result toast")
	}
}

func TestIntegration_DeployResultMsg(t *testing.T) {
	m := setupDashboard(t)
	m, _ = m.Update(tui.DeployResultMsg{
		Result: &deploy.DeployResult{},
	})
	view := m.(tuiapp.App).View().Content
	if !strings.Contains(view, "Deployed successfully") {
		t.Error("should show deploy success toast")
	}
}

func TestIntegration_RemoveResultMsg(t *testing.T) {
	m := setupDashboard(t)
	m, _ = m.Update(tui.RemoveResultMsg{})
	view := m.(tuiapp.App).View().Content
	if !strings.Contains(view, "Asset removed") {
		t.Error("should show remove success toast")
	}
}

func TestIntegration_SnapshotSaveResultMsg(t *testing.T) {
	m := setupDashboard(t)
	m, _ = m.Update(tui.SnapshotSaveMsg{Name: "my-snap"})
	view := m.(tuiapp.App).View().Content
	if !strings.Contains(view, "my-snap") {
		t.Error("should show snapshot save toast")
	}
}

func TestIntegration_MenuNavigation(t *testing.T) {
	app := newTestApp()
	m := sendKeys(app,
		tea.WindowSizeMsg{Width: 100, Height: 30},
		tea.KeyPressMsg{Code: tea.KeyEnter}, // picker → menu
	)

	// Navigate down to Skills
	m = sendKeys(m, tea.KeyPressMsg{Code: tea.KeyDown})
	// Select Skills
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	// Should be in dashboard with Skills tab
	view := m.(tuiapp.App).View().Content
	if !strings.Contains(view, "No assets deployed") && !strings.Contains(view, "Skills") {
		t.Error("should go to dashboard via menu item")
	}
}

// errMock is a simple error type for testing.
type errMock string

func (e errMock) Error() string { return string(e) }
