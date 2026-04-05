package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/armstrongl/nd/internal/sourcemanager"
)

var _ Screen = (*firstRunScreen)(nil)

func TestFirstRun_NewReturnsNonNil(t *testing.T) {
	s := NewStyles(true)
	m := newFirstRunScreen(newMockServices(), s, true)
	if m == nil {
		t.Fatal("newFirstRunScreen returned nil")
	}
}

func TestFirstRun_Title(t *testing.T) {
	s := NewStyles(true)
	m := newFirstRunScreen(newMockServices(), s, true)
	if got := m.Title(); got != "Welcome" {
		t.Fatalf("Title() = %q, want %q", got, "Welcome")
	}
}

func TestFirstRun_InputActiveBeforeNavigation(t *testing.T) {
	s := NewStyles(true)
	m := newFirstRunScreen(newMockServices(), s, true)
	if !m.InputActive() {
		t.Fatal("InputActive() = false before navigation, want true (form is active)")
	}
}

func TestFirstRun_InputActiveAfterNavigation(t *testing.T) {
	s := NewStyles(true)
	m := newFirstRunScreen(newMockServices(), s, true)
	m.navigated = true
	if m.InputActive() {
		t.Fatal("InputActive() = true after navigation, want false")
	}
}

func TestFirstRun_InitReturnsCmd(t *testing.T) {
	s := NewStyles(true)
	m := newFirstRunScreen(newMockServices(), s, true)
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil")
	}
}

func TestFirstRun_AddSourceNavigatesToSourceScreen(t *testing.T) {
	s := NewStyles(true)
	m := newFirstRunScreen(newMockServices(), s, true)
	m.choice = "add"
	m.form.State = huh.StateCompleted

	_, cmd := m.Update(nil)
	if cmd == nil {
		t.Fatal("Update() returned nil cmd after choosing 'add'")
	}

	// Execute the cmd — it should produce a NavigateMsg.
	msg := cmd()
	if _, ok := msg.(NavigateMsg); !ok {
		t.Fatalf("expected NavigateMsg, got %T", msg)
	}
}

func TestFirstRun_QuitReturnsQuitMsg(t *testing.T) {
	s := NewStyles(true)
	m := newFirstRunScreen(newMockServices(), s, true)
	m.choice = "quit"
	m.form.State = huh.StateCompleted

	_, cmd := m.Update(nil)
	if cmd == nil {
		t.Fatal("Update() returned nil cmd after choosing 'quit'")
	}

	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg, got %T", msg)
	}
}

// ISSUE-005: When the huh form is aborted (e.g., via esc producing StateAborted),
// the firstRunScreen must emit tea.Quit instead of entering a dead state.
func TestFirstRun_StateAborted_Quits(t *testing.T) {
	s := NewStyles(true)
	m := newFirstRunScreen(newMockServices(), s, true)
	m.form.State = huh.StateAborted

	_, cmd := m.Update(nil)
	if cmd == nil {
		t.Fatal("Update() returned nil cmd on StateAborted, expected tea.Quit")
	}

	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg on StateAborted, got %T", msg)
	}
}

// ISSUE-005: After StateAborted, navigated must be true so further updates are no-ops.
func TestFirstRun_StateAborted_SetsNavigated(t *testing.T) {
	s := NewStyles(true)
	m := newFirstRunScreen(newMockServices(), s, true)
	m.form.State = huh.StateAborted

	updated, _ := m.Update(nil)
	f := updated.(*firstRunScreen)
	if !f.navigated {
		t.Fatal("expected navigated=true after StateAborted")
	}
}

func TestHasUserSources_FalseWhenNilManager(t *testing.T) {
	svc := newMockServices()
	// Default mock returns nil, nil for SourceManager
	if hasUserSources(svc) {
		t.Fatal("expected false when SourceManager returns nil")
	}
}

func TestHasUserSources_FalseWhenOnlyBuiltin(t *testing.T) {
	svc := newMockServices()
	// SourceManager returns nil by default, which means Config() would panic.
	// The check should handle nil manager gracefully.
	if hasUserSources(svc) {
		t.Fatal("expected false when no user sources")
	}
}

func TestHasUserSources_FalseWithRealManagerBuiltinOnly(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	// Non-existent config → LoadConfig returns defaults (no user sources).
	// New() appends only the builtin source.
	sm, err := sourcemanager.New(configPath, "")
	if err != nil {
		t.Fatalf("sourcemanager.New: %v", err)
	}

	svc := newMockServices()
	svc.sourceManagerFn = func() (*sourcemanager.SourceManager, error) {
		return sm, nil
	}

	if hasUserSources(svc) {
		t.Fatal("expected false with only builtin source from real SourceManager")
	}
}

func TestHasUserSources_TrueWithRealManagerMixedSources(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	srcDir := filepath.Join(dir, "my-source")
	if err := os.Mkdir(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cfgYAML := fmt.Sprintf(`version: 1
default_scope: global
default_agent: claude-code
symlink_strategy: absolute
sources:
  - id: user-src
    type: local
    path: %s
`, srcDir)
	if err := os.WriteFile(configPath, []byte(cfgYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	sm, err := sourcemanager.New(configPath, "")
	if err != nil {
		t.Fatalf("sourcemanager.New: %v", err)
	}

	svc := newMockServices()
	svc.sourceManagerFn = func() (*sourcemanager.SourceManager, error) {
		return sm, nil
	}

	if !hasUserSources(svc) {
		t.Fatal("expected true with user + builtin sources from real SourceManager")
	}
}

func TestHasUserSources_FalseWhenSourceManagerErrors(t *testing.T) {
	svc := newMockServices()
	svc.sourceManagerFn = func() (*sourcemanager.SourceManager, error) {
		return nil, fmt.Errorf("config corrupted")
	}

	if hasUserSources(svc) {
		t.Fatal("expected false when SourceManager returns error")
	}
}
