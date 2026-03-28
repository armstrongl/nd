package tui

import (
	tea "charm.land/bubbletea/v2"
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

// --- Viewport wrapping tests (Unit 5) ---

func TestSnapshotScreen_ScreenSizeMsg_StoresPending(t *testing.T) {
	s := newSnapshotScreen(newMockServices(), NewStyles(true), true)
	s.Update(ScreenSizeMsg{Width: 80, Height: 30})
	if s.pendingWidth != 80 || s.pendingHeight != 30 {
		t.Fatalf("expected pending 80x30, got %dx%d", s.pendingWidth, s.pendingHeight)
	}
}

func TestSnapshotScreen_ScreenSizeMsg_UpdatesExistingViewport(t *testing.T) {
	s := newSnapshotScreen(newMockServices(), NewStyles(true), true)
	s.Update(ScreenSizeMsg{Width: 80, Height: 30})
	s.Update(snapshotLoadedMsg{snapshots: []profile.SnapshotSummary{{Name: "snap1", DeploymentCount: 3}}, err: nil})
	s.step = snapshotList
	s.initListViewport()
	s.Update(ScreenSizeMsg{Width: 120, Height: 50})
	if s.vp == nil {
		t.Fatal("viewport should exist after initListViewport")
	}
	if s.vp.Width() != 120 {
		t.Fatalf("expected vp width 120, got %d", s.vp.Width())
	}
	if s.vp.Height() != 49 {
		t.Fatalf("expected vp height 49 (50-1 for footer), got %d", s.vp.Height())
	}
}

func TestSnapshotScreen_ListViewport_InitAppliesPendingDimensions(t *testing.T) {
	s := newSnapshotScreen(newMockServices(), NewStyles(true), true)
	s.Update(ScreenSizeMsg{Width: 100, Height: 40})
	s.snapshots = []profile.SnapshotSummary{{Name: "snap1"}}
	s.step = snapshotList
	s.initListViewport()
	if s.vp == nil {
		t.Fatal("viewport should be created")
	}
	if s.vp.Width() != 100 {
		t.Fatalf("expected vp width 100, got %d", s.vp.Width())
	}
	if s.vp.Height() != 39 {
		t.Fatalf("expected vp height 39 (40-1), got %d", s.vp.Height())
	}
}

func TestSnapshotScreen_ListViewShowsContent(t *testing.T) {
	s := newSnapshotScreen(newMockServices(), NewStyles(true), true)
	s.Update(ScreenSizeMsg{Width: 80, Height: 30})
	snapshots := []profile.SnapshotSummary{
		{Name: "before-upgrade", DeploymentCount: 5},
		{Name: "auto-2026-03-22", DeploymentCount: 3},
	}
	s.Update(snapshotLoadedMsg{snapshots: snapshots, err: nil})
	s.step = snapshotList
	s.initListViewport()

	v := s.View()
	if !strings.Contains(v.Content, "before-upgrade") {
		t.Errorf("list view should show snapshot names, got: %q", v.Content)
	}
	if !strings.Contains(v.Content, "auto-2026-03-22") {
		t.Errorf("list view should show all snapshots, got: %q", v.Content)
	}
}

func TestSnapshotScreen_ListViewFooterVisible(t *testing.T) {
	s := newSnapshotScreen(newMockServices(), NewStyles(true), true)
	s.Update(ScreenSizeMsg{Width: 80, Height: 30})
	snapshots := []profile.SnapshotSummary{{Name: "snap1", DeploymentCount: 1}}
	s.Update(snapshotLoadedMsg{snapshots: snapshots, err: nil})
	s.step = snapshotList
	s.initListViewport()

	v := s.View()
	if !strings.Contains(v.Content, "Press esc to go back.") {
		t.Errorf("footer hint should be visible, got: %q", v.Content)
	}
}

func TestSnapshotScreen_ListViewManySnapshotsFooterStillVisible(t *testing.T) {
	s := newSnapshotScreen(newMockServices(), NewStyles(true), true)
	s.Update(ScreenSizeMsg{Width: 80, Height: 5})
	snapshots := make([]profile.SnapshotSummary, 50)
	for i := range snapshots {
		snapshots[i] = profile.SnapshotSummary{Name: fmt.Sprintf("snap-%03d", i), DeploymentCount: i}
	}
	s.Update(snapshotLoadedMsg{snapshots: snapshots, err: nil})
	s.step = snapshotList
	s.initListViewport()

	v := s.View()
	if !strings.Contains(v.Content, "Press esc to go back.") {
		t.Errorf("footer hint should remain visible with overflow, got: %q", v.Content)
	}
}

func TestSnapshotScreen_UpdateListForwardsToViewport(t *testing.T) {
	s := newSnapshotScreen(newMockServices(), NewStyles(true), true)
	s.Update(ScreenSizeMsg{Width: 80, Height: 30})
	snapshots := make([]profile.SnapshotSummary, 50)
	for i := range snapshots {
		snapshots[i] = profile.SnapshotSummary{Name: fmt.Sprintf("snap-%03d", i), DeploymentCount: i}
	}
	s.Update(snapshotLoadedMsg{snapshots: snapshots, err: nil})
	s.step = snapshotList
	s.initListViewport()

	_, cmd := s.Update(tea.KeyPressMsg{Code: 106}) // 'j'
	_ = cmd
}

func TestSnapshotScreen_EmptyListStillWorks(t *testing.T) {
	s := newSnapshotScreen(newMockServices(), NewStyles(true), true)
	s.Update(ScreenSizeMsg{Width: 80, Height: 30})
	s.Update(snapshotLoadedMsg{snapshots: nil, err: nil})
	s.step = snapshotList
	s.initListViewport()

	v := s.View()
	if !strings.Contains(v.Content, "No snapshots saved yet.") {
		t.Errorf("empty snapshot list should show 'no snapshots' message, got: %q", v.Content)
	}
}
