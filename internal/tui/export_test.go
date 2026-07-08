package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

var _ Screen = (*exportScreen)(nil)

func TestExport_NewReturnsNonNil(t *testing.T) {
	s := NewStyles(true)
	m := newExportScreen(newMockServices(), s, true)
	if m == nil {
		t.Fatal("newExportScreen returned nil")
	}
}

func TestExport_Title(t *testing.T) {
	s := NewStyles(true)
	m := newExportScreen(newMockServices(), s, true)
	if got := m.Title(); got != "Export" {
		t.Fatalf("Title() = %q, want %q", got, "Export")
	}
}

func TestExport_InputActiveIsFalse(t *testing.T) {
	s := NewStyles(true)
	m := newExportScreen(newMockServices(), s, true)
	if m.InputActive() {
		t.Fatal("InputActive() = true, want false (notice screen has no text input)")
	}
}

func TestExport_InitReturnsNil(t *testing.T) {
	s := NewStyles(true)
	m := newExportScreen(newMockServices(), s, true)
	if cmd := m.Init(); cmd != nil {
		t.Fatal("Init() returned non-nil cmd, want nil")
	}
}

func TestExport_ViewMentionsNdExport(t *testing.T) {
	s := NewStyles(true)
	m := newExportScreen(newMockServices(), s, true)

	v := m.View()
	if v.Content == "" {
		t.Fatal("View() returned empty content")
	}
	if !strings.Contains(v.Content, "nd export") {
		t.Fatalf("View() should mention 'nd export', got:\n%s", v.Content)
	}
	if !strings.Contains(v.Content, "Press enter to return.") {
		t.Fatalf("View() should contain 'Press enter to return.' hint, got:\n%s", v.Content)
	}
}

func TestExport_EnterEmitsPopToRoot(t *testing.T) {
	s := NewStyles(true)
	m := newExportScreen(newMockServices(), s, true)

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Update(enter) returned nil cmd, want PopToRootMsg cmd")
	}
	if _, ok := cmd().(PopToRootMsg); !ok {
		t.Fatalf("Update(enter) produced %T, want PopToRootMsg", cmd())
	}
}

func TestExport_EscEmitsPopToRoot(t *testing.T) {
	s := NewStyles(true)
	m := newExportScreen(newMockServices(), s, true)

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("Update(esc) returned nil cmd, want PopToRootMsg cmd")
	}
	if _, ok := cmd().(PopToRootMsg); !ok {
		t.Fatalf("Update(esc) produced %T, want PopToRootMsg", cmd())
	}
}

func TestExport_OtherKeyIsNoOp(t *testing.T) {
	s := NewStyles(true)
	m := newExportScreen(newMockServices(), s, true)

	model, cmd := m.Update(tea.KeyPressMsg{Code: 'x'})
	if cmd != nil {
		t.Fatalf("Update(x) returned non-nil cmd, want nil (only enter/esc act)")
	}
	if model != m {
		t.Fatal("Update(x) should return the same screen model")
	}
}
