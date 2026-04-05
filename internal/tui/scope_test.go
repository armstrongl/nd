package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/armstrongl/nd/internal/nd"
)

var _ Screen = (*scopeScreen)(nil)

func TestScope_NewReturnsNonNil(t *testing.T) {
	s := NewStyles(true)
	m := newScopeScreen(newMockServices(), s, true)
	if m == nil {
		t.Fatal("newScopeScreen returned nil")
	}
}

func TestScope_Title(t *testing.T) {
	s := NewStyles(true)
	m := newScopeScreen(newMockServices(), s, true)
	if got := m.Title(); got != "Switch Scope" {
		t.Fatalf("Title() = %q, want %q", got, "Switch Scope")
	}
}

func TestScope_InputActiveBeforeNavigation(t *testing.T) {
	s := NewStyles(true)
	m := newScopeScreen(newMockServices(), s, true)
	if !m.InputActive() {
		t.Fatal("InputActive() = false before navigation, want true (form is active)")
	}
}

func TestScope_InputActiveAfterNavigation(t *testing.T) {
	s := NewStyles(true)
	m := newScopeScreen(newMockServices(), s, true)
	m.navigated = true
	if m.InputActive() {
		t.Fatal("InputActive() = true after navigation, want false")
	}
}

func TestScope_InitReturnsCmd(t *testing.T) {
	s := NewStyles(true)
	m := newScopeScreen(newMockServices(), s, true)
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil")
	}
}

func TestScope_SelectGlobalEmitsScopeSwitchedMsg(t *testing.T) {
	svc := newMockServices()
	s := NewStyles(true)
	m := newScopeScreen(svc, s, true)
	m.choice = "global"
	m.form.State = huh.StateCompleted

	_, cmd := m.Update(nil)
	if cmd == nil {
		t.Fatal("Update() returned nil cmd after form completion")
	}

	// The cmd should produce a BatchMsg containing ScopeSwitchedMsg,
	// RefreshHeaderMsg, and PopToRootMsg.
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected tea.BatchMsg, got %T", msg)
	}

	var hasScopeSwitch, hasRefresh, hasPopToRoot bool
	for _, c := range batch {
		if c == nil {
			continue
		}
		switch c().(type) {
		case ScopeSwitchedMsg:
			hasScopeSwitch = true
		case RefreshHeaderMsg:
			hasRefresh = true
		case PopToRootMsg:
			hasPopToRoot = true
		}
	}
	if !hasScopeSwitch {
		t.Error("batch should contain ScopeSwitchedMsg")
	}
	if !hasRefresh {
		t.Error("batch should contain RefreshHeaderMsg")
	}
	if !hasPopToRoot {
		t.Error("batch should contain PopToRootMsg")
	}
}

func TestScope_HandleSelectionCallsResetForScope(t *testing.T) {
	svc := newMockServices()
	svc.getProjectRootFn = func() string { return "/some/project" }
	s := NewStyles(true)
	m := newScopeScreen(svc, s, true)

	m.choice = "global"
	cmd := m.handleScopeSelection()
	if cmd == nil {
		t.Fatal("handleScopeSelection() returned nil")
	}

	// Verify ResetForScope was called with the right scope.
	if len(svc.resetCalls) != 1 {
		t.Fatalf("expected 1 ResetForScope call, got %d", len(svc.resetCalls))
	}
	if svc.resetCalls[0].Scope != "global" {
		t.Errorf("expected scope 'global', got %q", svc.resetCalls[0].Scope)
	}

	// The cmd should produce a batch with all three messages.
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected tea.BatchMsg, got %T", msg)
	}

	var hasScopeSwitch, hasPopToRoot bool
	for _, c := range batch {
		if c == nil {
			continue
		}
		switch c().(type) {
		case ScopeSwitchedMsg:
			hasScopeSwitch = true
		case PopToRootMsg:
			hasPopToRoot = true
		}
	}
	if !hasScopeSwitch {
		t.Error("batch should contain ScopeSwitchedMsg")
	}
	if !hasPopToRoot {
		t.Error("batch should contain PopToRootMsg")
	}
}

func TestScope_NoProjectRootShowsError(t *testing.T) {
	svc := newMockServices()
	// GetProjectRoot defaults to "" in mock — no project root available
	s := NewStyles(true)
	m := newScopeScreen(svc, s, true)

	m.choice = "project"
	cmd := m.handleScopeSelection()

	// handleScopeSelection returns nil cmd and transitions to error step.
	if cmd != nil {
		t.Fatal("handleScopeSelection() should return nil cmd on missing project root")
	}
	if m.step != scopeShowError {
		t.Fatalf("step = %d, want scopeShowError (%d)", m.step, scopeShowError)
	}
	wantMsg := "Cannot switch to project scope: no project root detected."
	if m.errorMsg != wantMsg {
		t.Fatalf("errorMsg = %q, want %q", m.errorMsg, wantMsg)
	}

	// Should NOT call ResetForScope.
	if len(svc.resetCalls) != 0 {
		t.Fatalf("expected 0 ResetForScope calls, got %d", len(svc.resetCalls))
	}
}

func TestScope_NoProjectRootViewShowsErrorMessage(t *testing.T) {
	svc := newMockServices()
	s := NewStyles(true)
	m := newScopeScreen(svc, s, true)

	m.choice = "project"
	m.handleScopeSelection()

	v := m.View()
	if !strings.Contains(v.Content, "Cannot switch to project scope: no project root detected.") {
		t.Fatalf("View() should contain error message, got:\n%s", v.Content)
	}
	if !strings.Contains(v.Content, "Press enter to return.") {
		t.Fatalf("View() should contain 'Press enter to return.' hint, got:\n%s", v.Content)
	}
}

func TestScope_NoProjectRootEnterEmitsPopToRoot(t *testing.T) {
	svc := newMockServices()
	s := NewStyles(true)
	m := newScopeScreen(svc, s, true)

	// Trigger the error state.
	m.choice = "project"
	m.handleScopeSelection()
	if m.step != scopeShowError {
		t.Fatalf("expected scopeShowError step, got %d", m.step)
	}

	// Pressing enter on the error screen should emit PopToRootMsg.
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Update(enter) should return a cmd on error screen")
	}

	msg := cmd()
	if _, ok := msg.(PopToRootMsg); !ok {
		t.Fatalf("expected PopToRootMsg, got %T", msg)
	}
}

func TestScope_NoProjectRootInputActiveIsFalse(t *testing.T) {
	svc := newMockServices()
	s := NewStyles(true)
	m := newScopeScreen(svc, s, true)

	m.choice = "project"
	m.handleScopeSelection()

	if m.InputActive() {
		t.Fatal("InputActive() should be false during error step")
	}
}

func TestScope_ReverseToggle_ProjectToGlobal(t *testing.T) {
	svc := newMockServices()
	svc.getScopeFn = func() nd.Scope { return nd.ScopeProject }
	svc.getProjectRootFn = func() string { return "/some/project" }
	s := NewStyles(true)

	// Scope screen initializes with current scope as default.
	m := newScopeScreen(svc, s, true)
	if m.choice != "project" {
		t.Fatalf("choice should default to current scope 'project', got %q", m.choice)
	}

	// Simulate user selecting "global".
	m.choice = "global"
	cmd := m.handleScopeSelection()
	if cmd == nil {
		t.Fatal("handleScopeSelection() returned nil")
	}

	if len(svc.resetCalls) != 1 {
		t.Fatalf("expected 1 ResetForScope call, got %d", len(svc.resetCalls))
	}
	if svc.resetCalls[0].Scope != nd.ScopeGlobal {
		t.Errorf("expected scope %q, got %q", nd.ScopeGlobal, svc.resetCalls[0].Scope)
	}
}
