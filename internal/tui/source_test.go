package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

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

func TestSourceScreen_SyncDone_AllErrors(t *testing.T) {
	s := newSourceScreen(newMockServices(), NewStyles(true), true)
	s.Update(sourceSyncedMsg{synced: 0, errors: []error{fmt.Errorf("clone failed")}})

	// A total sync failure must be routed through the same s.err branch used
	// by load failures, not the generic done view.
	if s.err == nil {
		t.Fatal("sync failure should set s.err so it uses the shared error branch")
	}

	v := s.View()
	if !strings.Contains(v.Content, "clone failed") {
		t.Errorf("sync error view should show the error, got: %q", v.Content)
	}
	// The shared error branch ends with the "esc" hint; the generic done view
	// ends with the "enter" hint. Assert we render via the former, matching
	// TestSourceScreen_LoadError's presentation.
	if !strings.Contains(v.Content, "Press esc to go back.") {
		t.Errorf("sync error should render via the consistent error branch (esc hint), got: %q", v.Content)
	}
	if strings.Contains(v.Content, "Press enter to return.") {
		t.Errorf("sync error should not render via the generic done view (enter hint), got: %q", v.Content)
	}
}

func TestSourceScreen_SyncDone_PartialError_ShowsCountAndBranch(t *testing.T) {
	s := newSourceScreen(newMockServices(), NewStyles(true), true)
	s.Update(sourceSyncedMsg{synced: 2, errors: []error{fmt.Errorf("git pull failed")}})

	if s.err == nil {
		t.Fatal("partial sync failure should set s.err for the shared error branch")
	}

	v := s.View()
	// Partial success must keep BOTH the synced count and the error text.
	if !strings.Contains(v.Content, "synced 2") {
		t.Errorf("partial sync error should still show the synced count, got: %q", v.Content)
	}
	if !strings.Contains(v.Content, "git pull failed") {
		t.Errorf("partial sync error should show the error, got: %q", v.Content)
	}
	if !strings.Contains(v.Content, "Press esc to go back.") {
		t.Errorf("partial sync error should render via the consistent error branch, got: %q", v.Content)
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

// double-fire guard — adding flag prevents repeated runAdd calls
func TestSourceScreen_DoubleFireGuard_Add(t *testing.T) {
	s := newSourceScreen(newMockServices(), NewStyles(true), true)
	s.step = sourceAddGitInput
	s.adding = true

	_, cmd := s.updateAddForm(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("updateAddForm should return nil cmd when adding guard is set")
	}
}
