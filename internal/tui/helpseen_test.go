package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestHelpSeen_Lifecycle(t *testing.T) {
	dir := t.TempDir()
	svc := newMockServices()
	svc.getConfigPathFn = func() string { return filepath.Join(dir, "config.yaml") }

	if helpSeen(svc) {
		t.Fatal("helpSeen should be false before the flag is written")
	}

	// The flag must live beside other nd state.
	want := filepath.Join(dir, "state", "help_seen")
	if got := helpSeenPath(svc); got != want {
		t.Fatalf("helpSeenPath = %q, want %q", got, want)
	}

	if err := markHelpSeen(svc); err != nil {
		t.Fatalf("markHelpSeen: %v", err)
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("flag file not created at %s: %v", want, err)
	}
	if !helpSeen(svc) {
		t.Fatal("helpSeen should be true after markHelpSeen")
	}
}

func TestFirstRunTip_DismissedAndPersistedOnFirstKey(t *testing.T) {
	dir := t.TempDir()
	svc := newMockServices()
	svc.getConfigPathFn = func() string { return filepath.Join(dir, "config.yaml") }
	styles := NewStyles(true)

	m := Model{
		svc:         svc,
		styles:      styles,
		isDark:      true,
		screens:     []Screen{newMainMenuScreen(svc, styles, true)},
		width:       80,
		height:      24,
		firstRunTip: true,
	}

	// The tip line should render while firstRunTip is set.
	if !containsTip(m.View().Content) {
		t.Fatalf("expected first-run tip in view; got:\n%s", m.View().Content)
	}

	if helpSeen(svc) {
		t.Fatal("precondition: help_seen flag must not exist yet")
	}

	// First key press dismisses the tip and persists the flag.
	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: 'j'}))
	m2 := updated.(Model)

	if m2.firstRunTip {
		t.Error("firstRunTip should be false after the first key press")
	}
	if !helpSeen(svc) {
		t.Error("help_seen flag should be persisted after the first key press")
	}
	if containsTip(m2.View().Content) {
		t.Errorf("tip should no longer render after dismissal; got:\n%s", m2.View().Content)
	}
}

func TestFirstRunTip_NotShownWhenHelpSeen(t *testing.T) {
	dir := t.TempDir()
	svc := newMockServices()
	svc.getConfigPathFn = func() string { return filepath.Join(dir, "config.yaml") }
	if err := markHelpSeen(svc); err != nil {
		t.Fatalf("markHelpSeen: %v", err)
	}

	// Run's gating logic: firstRunTip is set to !helpSeen(svc).
	if !helpSeen(svc) {
		t.Fatal("expected helpSeen true after markHelpSeen")
	}
	styles := NewStyles(true)
	m := Model{
		svc:         svc,
		styles:      styles,
		isDark:      true,
		screens:     []Screen{newMainMenuScreen(svc, styles, true)},
		width:       80,
		height:      24,
		firstRunTip: !helpSeen(svc),
	}
	if m.firstRunTip {
		t.Fatal("firstRunTip should be false when help_seen flag is present")
	}
	if containsTip(m.View().Content) {
		t.Errorf("tip should not render when help_seen flag is present; got:\n%s", m.View().Content)
	}
}

func containsTip(content string) bool {
	return strings.Contains(content, "Press ? for help at any time")
}
