package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
)

// Compile-time check: settingsScreen satisfies Screen.
var _ Screen = (*settingsScreen)(nil)

func TestSettingsScreen_Title(t *testing.T) {
	s := newSettingsScreen(newMockServices(), NewStyles(true), true)
	if got := s.Title(); got != "Settings" {
		t.Fatalf("Title() = %q, want %q", got, "Settings")
	}
}

func TestSettingsScreen_InputActive_Menu(t *testing.T) {
	s := newSettingsScreen(newMockServices(), NewStyles(true), true)
	// Menu step uses huh Select which is not text input.
	if s.InputActive() {
		t.Fatal("InputActive() = true on menu step, want false")
	}
}

func TestSettingsScreen_InputActive_ScopeSwitch(t *testing.T) {
	s := newSettingsScreen(newMockServices(), NewStyles(true), true)
	s.step = settingsSwitchScope
	// Scope switch uses a huh Select form — not text input.
	if s.InputActive() {
		t.Fatal("InputActive() = true on scope switch step, want false")
	}
}

func TestSettingsScreen_InitReturnsCmd(t *testing.T) {
	s := newSettingsScreen(newMockServices(), NewStyles(true), true)
	cmd := s.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil cmd")
	}
}

func TestSettingsScreen_ViewMenuNonEmpty(t *testing.T) {
	s := newSettingsScreen(newMockServices(), NewStyles(true), true)
	s.Init()
	v := s.View()
	if v.Content == "" {
		t.Fatal("View() returned empty content on menu step")
	}
}

func TestSettingsScreen_FormStateNormal(t *testing.T) {
	s := newSettingsScreen(newMockServices(), NewStyles(true), true)
	if s.form == nil {
		t.Fatal("form is nil after construction")
	}
	if s.form.State != huh.StateNormal {
		t.Fatalf("form.State = %d, want StateNormal", s.form.State)
	}
}

func TestSettingsScreen_ShowConfigPath(t *testing.T) {
	svc := newMockServices()
	s := newSettingsScreen(svc, NewStyles(true), true)

	// Simulate selecting "path".
	s.Update(settingsActionMsg{action: "path"})

	v := s.View()
	if !strings.Contains(v.Content, "/tmp/nd-test/config.yaml") {
		t.Errorf("config path view should show config path, got: %q", v.Content)
	}
}

func TestSettingsScreen_ShowVersion(t *testing.T) {
	s := newSettingsScreen(newMockServices(), NewStyles(true), true)

	s.Update(settingsActionMsg{action: "version"})

	v := s.View()
	if !strings.Contains(v.Content, "nd version") {
		t.Errorf("version view should contain version string, got: %q", v.Content)
	}
}

func TestSettingsScreen_EnterOnResult_ReturnsToMenu(t *testing.T) {
	s := newSettingsScreen(newMockServices(), NewStyles(true), true)
	s.step = settingsShowResult
	s.result = "some info"

	_, cmd := s.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd == nil {
		t.Fatal("enter on result step should emit a cmd")
	}
}

func TestSettingsScreen_ScopeSwitchStep_FormNotNil(t *testing.T) {
	s := newSettingsScreen(newMockServices(), NewStyles(true), true)
	s.Update(settingsActionMsg{action: "scope"})

	if s.step != settingsSwitchScope {
		t.Fatalf("step = %d after scope action, want settingsSwitchScope", s.step)
	}
	if s.scopeForm == nil {
		t.Fatal("scopeForm is nil after scope action")
	}
}

func TestSettingsScreen_ScopeSwitch_CallsResetForScope(t *testing.T) {
	svc := newMockServices()
	s := newSettingsScreen(svc, NewStyles(true), true)

	// Trigger scope selection.
	s.Update(settingsActionMsg{action: "scope"})

	// Simulate completing scope form with "global".
	s.Update(settingsScopeSelectedMsg{scope: "global"})

	if len(svc.resetCalls) != 1 {
		t.Fatalf("expected 1 ResetForScope call, got %d", len(svc.resetCalls))
	}
}
