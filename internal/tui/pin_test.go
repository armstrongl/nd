package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/state"
)

// Compile-time check: pinScreen satisfies Screen.
var _ Screen = (*pinScreen)(nil)

func TestPinScreen_Title(t *testing.T) {
	s := newPinScreen(newMockServices(), NewStyles(true), true)
	if got := s.Title(); got != "Pin/Unpin" {
		t.Fatalf("Title() = %q, want %q", got, "Pin/Unpin")
	}
}

func TestPinScreen_InputActive_Select(t *testing.T) {
	s := newPinScreen(newMockServices(), NewStyles(true), true)
	s.step = pinSelect
	if !s.InputActive() {
		t.Fatal("InputActive() = false on select step, want true (form active)")
	}
}

func TestPinScreen_InputActive_Confirm(t *testing.T) {
	s := newPinScreen(newMockServices(), NewStyles(true), true)
	s.step = pinConfirm
	if !s.InputActive() {
		t.Fatal("InputActive() = false on confirm step, want true (form active)")
	}
}

func TestPinScreen_InitReturnsCmd(t *testing.T) {
	s := newPinScreen(newMockServices(), NewStyles(true), true)
	cmd := s.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil cmd")
	}
}

func TestPinScreen_LoadingView(t *testing.T) {
	s := newPinScreen(newMockServices(), NewStyles(true), true)
	v := s.View()
	if !strings.Contains(v.Content, "Loading") {
		t.Errorf("loading view should contain 'Loading', got: %q", v.Content)
	}
}

func TestPinScreen_EmptyDeployments(t *testing.T) {
	s := newPinScreen(newMockServices(), NewStyles(true), true)
	s.Update(pinLoadedMsg{deployments: nil, err: nil})

	v := s.View()
	if !strings.Contains(v.Content, "Nothing deployed") && !strings.Contains(v.Content, "No deployed") {
		t.Errorf("empty state should mention no deployed assets, got: %q", v.Content)
	}
}

func TestPinScreen_LoadError(t *testing.T) {
	s := newPinScreen(newMockServices(), NewStyles(true), true)
	s.Update(pinLoadedMsg{err: fmt.Errorf("state locked")})

	v := s.View()
	if !strings.Contains(v.Content, "state locked") {
		t.Errorf("error view should show error, got: %q", v.Content)
	}
}

func TestPinScreen_WithDeployments_HasForm(t *testing.T) {
	s := newPinScreen(newMockServices(), NewStyles(true), true)
	deployments := []state.Deployment{
		{AssetName: "my-skill", AssetType: nd.AssetSkill, SourceID: "src1", Origin: nd.OriginManual},
		{AssetName: "my-rule", AssetType: nd.AssetRule, SourceID: "src1", Origin: nd.OriginPinned},
	}
	s.Update(pinLoadedMsg{deployments: deployments, err: nil})

	if s.step != pinSelect {
		t.Fatalf("step should be pinSelect after load, got %d", s.step)
	}
	if s.assetForm == nil {
		t.Fatal("assetForm should be set after load with deployments")
	}
}

func TestPinScreen_PinnedAssetsPreSelected(t *testing.T) {
	s := newPinScreen(newMockServices(), NewStyles(true), true)
	pinned := state.Deployment{AssetName: "pinned-skill", AssetType: nd.AssetSkill, SourceID: "s", Origin: nd.OriginPinned}
	unpinned := state.Deployment{AssetName: "normal-skill", AssetType: nd.AssetSkill, SourceID: "s", Origin: nd.OriginManual}
	s.Update(pinLoadedMsg{deployments: []state.Deployment{pinned, unpinned}, err: nil})

	// Pinned assets should be pre-selected.
	pinnedKey := pinned.Identity().String()
	found := false
	for _, k := range s.selected {
		if k == pinnedKey {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("pinned asset %q should be pre-selected, selected: %v", pinnedKey, s.selected)
	}
}

func TestPinScreen_DoneMsg_Success(t *testing.T) {
	s := newPinScreen(newMockServices(), NewStyles(true), true)
	s.Update(pinDoneMsg{pinned: 1, unpinned: 1, err: nil})

	v := s.View()
	if !strings.Contains(v.Content, "1") {
		t.Errorf("done view should show counts, got: %q", v.Content)
	}
}

func TestPinScreen_DoneMsg_Error(t *testing.T) {
	s := newPinScreen(newMockServices(), NewStyles(true), true)
	s.Update(pinDoneMsg{err: fmt.Errorf("engine error")})

	v := s.View()
	if !strings.Contains(v.Content, "engine error") {
		t.Errorf("error view should show error, got: %q", v.Content)
	}
}

func TestPinScreen_RefreshHeaderAfterDone(t *testing.T) {
	s := newPinScreen(newMockServices(), NewStyles(true), true)
	_, cmd := s.Update(pinDoneMsg{pinned: 1, unpinned: 0, err: nil})

	if cmd == nil {
		t.Fatal("pin done should emit a cmd")
	}
	msg := cmd()
	switch msg.(type) {
	case RefreshHeaderMsg:
		// OK
	default:
		t.Errorf("pin done should emit RefreshHeaderMsg, got %T", msg)
	}
}
