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

func TestSourceScreen_RefreshHeaderAfterSync(t *testing.T) {
	s := newSourceScreen(newMockServices(), NewStyles(true), true)
	_, cmd := s.Update(sourceSyncedMsg{synced: 1, errors: nil})

	// Sync now batches a reload alongside the header refresh; the header
	// refresh must still be emitted.
	_, hasRefresh := batchContains(t, cmd)
	if !hasRefresh {
		t.Error("sync done should emit RefreshHeaderMsg")
	}
}

func TestSourceScreen_AddReloadsList(t *testing.T) {
	s := newSourceScreen(newMockServices(), NewStyles(true), true)
	src := &source.Source{ID: "my-local", Path: "/home/user/nd-assets"}
	_, cmd := s.Update(sourceAddedMsg{src: src, err: nil})

	hasLoaded, hasRefresh := batchContains(t, cmd)
	if !hasLoaded {
		t.Error("add success batch should contain a reload (sourceLoadedMsg)")
	}
	if !hasRefresh {
		t.Error("add success batch should contain RefreshHeaderMsg")
	}
}

func TestSourceScreen_RemoveReloadsList(t *testing.T) {
	s := newSourceScreen(newMockServices(), NewStyles(true), true)
	_, cmd := s.Update(sourceRemovedMsg{id: "old-src", err: nil})

	hasLoaded, hasRefresh := batchContains(t, cmd)
	if !hasLoaded {
		t.Error("remove success batch should contain a reload (sourceLoadedMsg)")
	}
	if !hasRefresh {
		t.Error("remove success batch should contain RefreshHeaderMsg")
	}
}

func TestSourceScreen_SyncReloadsList(t *testing.T) {
	s := newSourceScreen(newMockServices(), NewStyles(true), true)
	_, cmd := s.Update(sourceSyncedMsg{synced: 1, errors: nil})

	hasLoaded, hasRefresh := batchContains(t, cmd)
	if !hasLoaded {
		t.Error("sync batch should contain a reload (sourceLoadedMsg)")
	}
	if !hasRefresh {
		t.Error("sync batch should contain RefreshHeaderMsg")
	}
}

func TestSourceScreen_ReloadPreservesDoneView(t *testing.T) {
	s := newSourceScreen(newMockServices(), NewStyles(true), true)
	src := &source.Source{ID: "my-local", Path: "/home/user/nd-assets"}
	s.Update(sourceAddedMsg{src: src, err: nil})
	if s.step != sourceDone {
		t.Fatalf("step = %v after add, want sourceDone", s.step)
	}

	// The reload result arriving while the confirmation is shown should refresh
	// the cached slice without bouncing the user back to the menu.
	sources := []source.Source{{ID: "my-local", Path: "/home/user/nd-assets"}}
	s.Update(sourceLoadedMsg{sources: sources, err: nil})

	if len(s.sources) != 1 || s.sources[0].ID != "my-local" {
		t.Fatalf("sources not reloaded on done screen: %+v", s.sources)
	}
	if s.step != sourceDone {
		t.Fatalf("step = %v after reload, want sourceDone preserved", s.step)
	}
	v := s.View()
	if !strings.Contains(v.Content, "added.") {
		t.Errorf("done view should still show success text, got: %q", v.Content)
	}
}

// batchContains invokes a tea.Batch command and reports whether its child
// commands yield a sourceLoadedMsg (the reload) and a RefreshHeaderMsg.
func batchContains(t *testing.T, cmd tea.Cmd) (hasLoaded, hasRefresh bool) {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected a non-nil batch cmd")
	}
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected tea.BatchMsg, got %T", msg)
	}
	for _, c := range batch {
		if c == nil {
			continue
		}
		switch c().(type) {
		case sourceLoadedMsg:
			hasLoaded = true
		case RefreshHeaderMsg:
			hasRefresh = true
		}
	}
	return hasLoaded, hasRefresh
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
