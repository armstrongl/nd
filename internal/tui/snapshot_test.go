package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/armstrongl/nd/internal/deploy"
	"github.com/armstrongl/nd/internal/profile"
)

// Compile-time check: snapshotScreen satisfies Screen.
var _ Screen = (*snapshotScreen)(nil)

func TestSnapshotScreen_Title(t *testing.T) {
	s := newSnapshotScreen(newMockServices(), NewStyles(true), true)
	if got := s.Title(); got != "Snapshots" {
		t.Fatalf("Title() = %q, want %q", got, "Snapshots")
	}
}

func TestSnapshotScreen_InputActive_Save(t *testing.T) {
	s := newSnapshotScreen(newMockServices(), NewStyles(true), true)
	s.step = snapshotSaveName
	if !s.InputActive() {
		t.Fatal("InputActive() = false during save name input, want true")
	}
}

func TestSnapshotScreen_InputActive_Menu(t *testing.T) {
	s := newSnapshotScreen(newMockServices(), NewStyles(true), true)
	s.step = snapshotMenu
	if !s.InputActive() {
		t.Fatal("InputActive() = false on menu step, want true (form active)")
	}
}

func TestSnapshotScreen_InitReturnsCmd(t *testing.T) {
	s := newSnapshotScreen(newMockServices(), NewStyles(true), true)
	cmd := s.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil cmd")
	}
}

func TestSnapshotScreen_LoadingView(t *testing.T) {
	s := newSnapshotScreen(newMockServices(), NewStyles(true), true)
	v := s.View()
	if !strings.Contains(v.Content, "Loading") {
		t.Errorf("loading view should contain 'Loading', got: %q", v.Content)
	}
}

func TestSnapshotScreen_MenuView_AfterLoad(t *testing.T) {
	s := newSnapshotScreen(newMockServices(), NewStyles(true), true)
	s.Update(snapshotLoadedMsg{snapshots: nil, err: nil})
	s.Init() // init the form

	v := s.View()
	if v.Content == "" {
		t.Fatal("menu view should not be empty after load")
	}
}

func TestSnapshotScreen_LoadError(t *testing.T) {
	s := newSnapshotScreen(newMockServices(), NewStyles(true), true)
	testErr := fmt.Errorf("profile store unavailable")
	s.Update(snapshotLoadedMsg{snapshots: nil, err: testErr})

	v := s.View()
	if !strings.Contains(v.Content, "profile store unavailable") {
		t.Errorf("error view should show error message, got: %q", v.Content)
	}
}

func TestSnapshotScreen_ListView(t *testing.T) {
	s := newSnapshotScreen(newMockServices(), NewStyles(true), true)
	snapshots := []profile.SnapshotSummary{
		{Name: "before-upgrade", DeploymentCount: 5},
		{Name: "auto-2026-03-22", DeploymentCount: 3},
	}
	s.Update(snapshotLoadedMsg{snapshots: snapshots, err: nil})
	s.step = snapshotList

	v := s.View()
	if !strings.Contains(v.Content, "before-upgrade") {
		t.Errorf("list view should show snapshot names, got: %q", v.Content)
	}
	if !strings.Contains(v.Content, "auto-2026-03-22") {
		t.Errorf("list view should show all snapshots, got: %q", v.Content)
	}
}

func TestSnapshotScreen_SaveDone_Success(t *testing.T) {
	s := newSnapshotScreen(newMockServices(), NewStyles(true), true)
	s.Update(snapshotSavedMsg{name: "my-snap", err: nil})

	v := s.View()
	if !strings.Contains(v.Content, "my-snap") {
		t.Errorf("saved view should mention snapshot name, got: %q", v.Content)
	}
}

func TestSnapshotScreen_SaveDone_Error(t *testing.T) {
	s := newSnapshotScreen(newMockServices(), NewStyles(true), true)
	s.Update(snapshotSavedMsg{err: fmt.Errorf("disk full")})

	v := s.View()
	if !strings.Contains(v.Content, "disk full") {
		t.Errorf("save error view should show error, got: %q", v.Content)
	}
}

func TestSnapshotScreen_RestoreDone_Success(t *testing.T) {
	s := newSnapshotScreen(newMockServices(), NewStyles(true), true)
	result := &profile.RestoreResult{
		SnapshotName: "before-upgrade",
		Deployed:     &deploy.BulkDeployResult{},
	}
	s.Update(snapshotRestoredMsg{result: result, err: nil})

	v := s.View()
	if !strings.Contains(v.Content, "before-upgrade") {
		t.Errorf("restored view should mention snapshot name, got: %q", v.Content)
	}
}

func TestSnapshotScreen_RestoreDone_Error(t *testing.T) {
	s := newSnapshotScreen(newMockServices(), NewStyles(true), true)
	s.Update(snapshotRestoredMsg{result: nil, err: fmt.Errorf("snapshot not found")})

	v := s.View()
	if !strings.Contains(v.Content, "snapshot not found") {
		t.Errorf("restore error view should show error, got: %q", v.Content)
	}
}

func TestSnapshotScreen_RefreshHeaderAfterRestore(t *testing.T) {
	s := newSnapshotScreen(newMockServices(), NewStyles(true), true)
	result := &profile.RestoreResult{SnapshotName: "snap"}
	_, cmd := s.Update(snapshotRestoredMsg{result: result, err: nil})

	if cmd == nil {
		t.Fatal("restore should emit a cmd")
	}
	msg := cmd()
	switch msg.(type) {
	case RefreshHeaderMsg:
		// OK
	default:
		t.Errorf("restore should emit RefreshHeaderMsg, got %T", msg)
	}
}
