package tui

import (
	"errors"
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
	if !s.InputActive() {
		t.Fatal("InputActive() = false on menu step, want true (form active)")
	}
}

func TestSettingsScreen_InputActive_ScopeSwitch(t *testing.T) {
	s := newSettingsScreen(newMockServices(), NewStyles(true), true)
	s.step = settingsSwitchScope
	if !s.InputActive() {
		t.Fatal("InputActive() = false on scope switch step, want true (form active)")
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

func TestSettingsScreen_ScopeSwitch_ProjectWithNoRootShowsError(t *testing.T) {
	svc := newMockServices()
	// GetProjectRoot defaults to "" — no project root
	s := newSettingsScreen(svc, NewStyles(true), true)

	s.Update(settingsActionMsg{action: "scope"})
	s.Update(settingsScopeSelectedMsg{scope: "project"})

	// Should NOT call ResetForScope.
	if len(svc.resetCalls) != 0 {
		t.Fatalf("expected 0 ResetForScope calls, got %d", len(svc.resetCalls))
	}

	// Should show an error message in the result step.
	if s.step != settingsShowResult {
		t.Fatalf("step should be settingsShowResult, got %d", s.step)
	}
	if !strings.Contains(s.result, "Cannot switch") {
		t.Errorf("result should contain guard message, got %q", s.result)
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

func TestSettingsScreen_EditorFinished_ReturnsToSettingsMenu(t *testing.T) {
	s := newSettingsScreen(newMockServices(), NewStyles(true), true)

	// Simulate the editor exiting successfully.
	_, cmd := s.Update(editorFinishedMsg{err: nil})

	// Should return to settings menu, not emit BackMsg.
	if s.step != settingsMenu {
		t.Fatalf("step = %d after editorFinishedMsg, want settingsMenu (%d)", s.step, settingsMenu)
	}
	if s.form == nil {
		t.Fatal("form is nil after editorFinishedMsg, want rebuilt menu")
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd (form.Init) after editorFinishedMsg")
	}

	// Verify it did NOT produce a BackMsg.
	msg := cmd()
	if _, ok := msg.(BackMsg); ok {
		t.Fatal("editorFinishedMsg should not produce BackMsg; user should stay on settings screen")
	}
}

func TestSettingsScreen_EditorFinished_ErrorSurfaced(t *testing.T) {
	s := newSettingsScreen(newMockServices(), NewStyles(true), true)

	// Simulate the editor exiting with an error.
	editorErr := errors.New("editor crashed: exit status 1")
	s.Update(editorFinishedMsg{err: editorErr})

	// Should show the error in the result step.
	if s.step != settingsShowResult {
		t.Fatalf("step = %d after editorFinishedMsg with error, want settingsShowResult (%d)", s.step, settingsShowResult)
	}
	if !strings.Contains(s.result, "Editor error") {
		t.Errorf("result should contain 'Editor error', got %q", s.result)
	}
	if !strings.Contains(s.result, "exit status 1") {
		t.Errorf("result should contain the original error message, got %q", s.result)
	}
}
