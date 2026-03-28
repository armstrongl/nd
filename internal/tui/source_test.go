package tui

import (
	tea "charm.land/bubbletea/v2"
	"fmt"
	"strings"
	"testing"

	"github.com/armstrongl/nd/internal/source"
)

// Compile-time check: sourceScreen satisfies Screen.
var _ Screen = (*sourceScreen)(nil)

func TestSourceScreen_Title(t *testing.T) {
	s := newSourceScreen(newMockServices(), NewStyles(true), true)
	if got := s.Title(); got != "Sources" {
		t.Fatalf("Title() = %q, want %q", got, "Sources")
	}
}

func TestSourceScreen_InputActive_AddLocal(t *testing.T) {
	s := newSourceScreen(newMockServices(), NewStyles(true), true)
	s.step = sourceAddLocalInput
	if !s.InputActive() {
		t.Fatal("InputActive() = false during add local input, want true")
	}
}

func TestSourceScreen_InputActive_AddGit(t *testing.T) {
	s := newSourceScreen(newMockServices(), NewStyles(true), true)
	s.step = sourceAddGitInput
	if !s.InputActive() {
		t.Fatal("InputActive() = false during add git input, want true")
	}
}

func TestSourceScreen_InputActive_Menu(t *testing.T) {
	s := newSourceScreen(newMockServices(), NewStyles(true), true)
	s.step = sourceMenu
	if !s.InputActive() {
		t.Fatal("InputActive() = false on menu step, want true (form active)")
	}
}

func TestSourceScreen_InitReturnsCmd(t *testing.T) {
	s := newSourceScreen(newMockServices(), NewStyles(true), true)
	cmd := s.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil cmd")
	}
}

func TestSourceScreen_LoadingView(t *testing.T) {
	s := newSourceScreen(newMockServices(), NewStyles(true), true)
	v := s.View()
	if !strings.Contains(v.Content, "Loading") {
		t.Errorf("loading view should contain 'Loading', got: %q", v.Content)
	}
}

func TestSourceScreen_ListView(t *testing.T) {
	s := newSourceScreen(newMockServices(), NewStyles(true), true)
	sources := []source.Source{
		{ID: "local-src", Path: "/home/user/assets"},
		{ID: "remote-src", URL: "https://github.com/org/nd-assets", Path: "/tmp/nd-remote-src"},
	}
	s.Update(sourceLoadedMsg{sources: sources, err: nil})
	s.step = sourceList

	v := s.View()
	if !strings.Contains(v.Content, "local-src") {
		t.Errorf("list view should show source IDs, got: %q", v.Content)
	}
	if !strings.Contains(v.Content, "remote-src") {
		t.Errorf("list view should show all sources, got: %q", v.Content)
	}
}

func TestSourceScreen_LoadError(t *testing.T) {
	s := newSourceScreen(newMockServices(), NewStyles(true), true)
	s.Update(sourceLoadedMsg{err: fmt.Errorf("source manager unavailable")})

	v := s.View()
	if !strings.Contains(v.Content, "source manager unavailable") {
		t.Errorf("error view should show error, got: %q", v.Content)
	}
}

func TestSourceScreen_AddLocalDone_Success(t *testing.T) {
	s := newSourceScreen(newMockServices(), NewStyles(true), true)
	src := &source.Source{ID: "my-local", Path: "/home/user/nd-assets"}
	s.Update(sourceAddedMsg{src: src, err: nil})

	v := s.View()
	if !strings.Contains(v.Content, "my-local") {
		t.Errorf("add result should mention source ID, got: %q", v.Content)
	}
}

func TestSourceScreen_AddDone_Error(t *testing.T) {
	s := newSourceScreen(newMockServices(), NewStyles(true), true)
	s.Update(sourceAddedMsg{err: fmt.Errorf("path does not exist")})

	v := s.View()
	if !strings.Contains(v.Content, "path does not exist") {
		t.Errorf("add error view should show error, got: %q", v.Content)
	}
}

func TestSourceScreen_RemoveDone_Success(t *testing.T) {
	s := newSourceScreen(newMockServices(), NewStyles(true), true)
	s.Update(sourceRemovedMsg{id: "old-src", err: nil})

	v := s.View()
	if !strings.Contains(v.Content, "old-src") {
		t.Errorf("remove result should mention source ID, got: %q", v.Content)
	}
}

func TestSourceScreen_RemoveDone_Error(t *testing.T) {
	s := newSourceScreen(newMockServices(), NewStyles(true), true)
	s.Update(sourceRemovedMsg{err: fmt.Errorf("source not found")})

	v := s.View()
	if !strings.Contains(v.Content, "source not found") {
		t.Errorf("remove error view should show error, got: %q", v.Content)
	}
}

func TestSourceScreen_SyncDone_Success(t *testing.T) {
	s := newSourceScreen(newMockServices(), NewStyles(true), true)
	s.Update(sourceSyncedMsg{synced: 2, errors: nil})

	v := s.View()
	if !strings.Contains(v.Content, "2") {
		t.Errorf("sync result should show synced count, got: %q", v.Content)
	}
}

func TestSourceScreen_SyncDone_PartialError(t *testing.T) {
	s := newSourceScreen(newMockServices(), NewStyles(true), true)
	s.Update(sourceSyncedMsg{synced: 1, errors: []error{fmt.Errorf("git pull failed")}})

	v := s.View()
	if !strings.Contains(v.Content, "git pull failed") {
		t.Errorf("sync partial error should show errors, got: %q", v.Content)
	}
}

func TestSourceScreen_RefreshHeaderAfterSync(t *testing.T) {
	s := newSourceScreen(newMockServices(), NewStyles(true), true)
	_, cmd := s.Update(sourceSyncedMsg{synced: 1, errors: nil})

	if cmd == nil {
		t.Fatal("sync done should emit a cmd")
	}
	msg := cmd()
	switch msg.(type) {
	case RefreshHeaderMsg:
		// OK
	default:
		t.Errorf("sync done should emit RefreshHeaderMsg, got %T", msg)
	}
}

func TestSourceScreen_MenuView_AfterLoad(t *testing.T) {
	sources := []source.Source{{ID: "s1"}}
	s := newSourceScreen(newMockServices(), NewStyles(true), true)
	s.Update(sourceLoadedMsg{sources: sources, err: nil})
	s.Init()

	v := s.View()
	if v.Content == "" {
		t.Fatal("menu view should not be empty after load")
	}
}

// --- Viewport wrapping tests (Unit 5) ---

func TestSourceScreen_ScreenSizeMsg_StoresPending(t *testing.T) {
	s := newSourceScreen(newMockServices(), NewStyles(true), true)
	s.Update(ScreenSizeMsg{Width: 80, Height: 30})
	// pendingHeight stores the footer-adjusted value (30-1=29).
	if s.pendingWidth != 80 || s.pendingHeight != 29 {
		t.Fatalf("expected pending 80x29, got %dx%d", s.pendingWidth, s.pendingHeight)
	}
}

func TestSourceScreen_ScreenSizeMsg_UpdatesExistingViewport(t *testing.T) {
	s := newSourceScreen(newMockServices(), NewStyles(true), true)
	// Store pending dimensions
	s.Update(ScreenSizeMsg{Width: 80, Height: 30})
	// Load sources and navigate to list
	s.Update(sourceLoadedMsg{sources: []source.Source{{ID: "s1", Path: "/a"}}, err: nil})
	s.step = sourceList
	s.initListViewport()
	// Now send a new ScreenSizeMsg — should update the viewport directly
	s.Update(ScreenSizeMsg{Width: 120, Height: 50})
	if s.vp == nil {
		t.Fatal("viewport should exist after initListViewport")
	}
	if s.vp.Width() != 120 {
		t.Fatalf("expected vp width 120, got %d", s.vp.Width())
	}
	// Height should be msg.Height - 1 for footer
	if s.vp.Height() != 49 {
		t.Fatalf("expected vp height 49 (50-1 for footer), got %d", s.vp.Height())
	}
}

func TestSourceScreen_ListViewport_InitAppliesPendingDimensions(t *testing.T) {
	s := newSourceScreen(newMockServices(), NewStyles(true), true)
	s.Update(ScreenSizeMsg{Width: 100, Height: 40})
	s.sources = []source.Source{{ID: "s1", Path: "/a"}}
	s.step = sourceList
	s.initListViewport()
	if s.vp == nil {
		t.Fatal("viewport should be created")
	}
	if s.vp.Width() != 100 {
		t.Fatalf("expected vp width 100, got %d", s.vp.Width())
	}
	// pendingHeight already stores the footer-adjusted value (40-1=39).
	if s.vp.Height() != 39 {
		t.Fatalf("expected vp height 39 (40-1), got %d", s.vp.Height())
	}
}

func TestSourceScreen_ListViewShowsContent(t *testing.T) {
	s := newSourceScreen(newMockServices(), NewStyles(true), true)
	s.Update(ScreenSizeMsg{Width: 80, Height: 30})
	sources := []source.Source{
		{ID: "local-src", Path: "/home/user/assets"},
		{ID: "remote-src", URL: "https://github.com/org/nd-assets", Path: "/tmp/nd-remote-src"},
	}
	s.Update(sourceLoadedMsg{sources: sources, err: nil})
	s.step = sourceList
	s.initListViewport()

	v := s.View()
	if !strings.Contains(v.Content, "local-src") {
		t.Errorf("list view should show source IDs, got: %q", v.Content)
	}
	if !strings.Contains(v.Content, "remote-src") {
		t.Errorf("list view should show all sources, got: %q", v.Content)
	}
}

func TestSourceScreen_ListViewFooterVisible(t *testing.T) {
	s := newSourceScreen(newMockServices(), NewStyles(true), true)
	s.Update(ScreenSizeMsg{Width: 80, Height: 30})
	sources := []source.Source{{ID: "s1", Path: "/a"}}
	s.Update(sourceLoadedMsg{sources: sources, err: nil})
	s.step = sourceList
	s.initListViewport()

	v := s.View()
	if !strings.Contains(v.Content, "Press esc to go back.") {
		t.Errorf("footer hint should be visible, got: %q", v.Content)
	}
}

func TestSourceScreen_ListViewManySourcesFooterStillVisible(t *testing.T) {
	s := newSourceScreen(newMockServices(), NewStyles(true), true)
	// Small viewport height to force overflow
	s.Update(ScreenSizeMsg{Width: 80, Height: 5})
	sources := make([]source.Source, 50)
	for i := range sources {
		sources[i] = source.Source{ID: fmt.Sprintf("src-%03d", i), Path: fmt.Sprintf("/path/%d", i)}
	}
	s.Update(sourceLoadedMsg{sources: sources, err: nil})
	s.step = sourceList
	s.initListViewport()

	v := s.View()
	if !strings.Contains(v.Content, "Press esc to go back.") {
		t.Errorf("footer hint should remain visible with overflow, got: %q", v.Content)
	}
}

func TestSourceScreen_ListViewEmptyStillWorks(t *testing.T) {
	s := newSourceScreen(newMockServices(), NewStyles(true), true)
	s.Update(ScreenSizeMsg{Width: 80, Height: 30})
	s.Update(sourceLoadedMsg{sources: nil, err: nil})
	s.step = sourceList
	s.initListViewport()

	v := s.View()
	// Empty list should show NoSources message
	if v.Content == "" {
		t.Error("empty list view should not be blank")
	}
}

func TestSourceScreen_UpdateListForwardsToViewport(t *testing.T) {
	s := newSourceScreen(newMockServices(), NewStyles(true), true)
	s.Update(ScreenSizeMsg{Width: 80, Height: 30})
	sources := make([]source.Source, 50)
	for i := range sources {
		sources[i] = source.Source{ID: fmt.Sprintf("src-%03d", i), Path: fmt.Sprintf("/path/%d", i)}
	}
	s.Update(sourceLoadedMsg{sources: sources, err: nil})
	s.step = sourceList
	s.initListViewport()

	// Capture view before key press.
	before := s.View().Content

	// Send a 'j' key press — should be forwarded to viewport for scrolling.
	s.Update(tea.KeyPressMsg(tea.Key{Code: 'j'}))

	after := s.View().Content
	// The viewport should have processed the key (view may differ if scrolled).
	// At minimum, the viewport must still produce content.
	if after == "" {
		t.Fatal("viewport should still produce content after key press")
	}
	_ = before // used to verify no panic; content may or may not change depending on viewport size
}

func TestSourceScreen_ScreenSizeMsg_ZeroHeight_NoPanic(t *testing.T) {
	s := newSourceScreen(newMockServices(), NewStyles(true), true)
	// ScreenSizeMsg with Height=0 should not cause negative viewport height.
	s.Update(ScreenSizeMsg{Width: 80, Height: 0})
	if s.pendingHeight != 0 {
		t.Fatalf("expected pendingHeight=0, got %d", s.pendingHeight)
	}
	// Create viewport with zero height — should not panic.
	s.sources = []source.Source{{ID: "s1", Path: "/a"}}
	s.step = sourceList
	s.initListViewport()
	if s.vp == nil {
		t.Fatal("viewport should be created even with zero height")
	}
}
