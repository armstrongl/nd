package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/armstrongl/nd/internal/deploy"
	"github.com/armstrongl/nd/internal/profile"
)

// Compile-time check: profileScreen satisfies Screen.
var _ Screen = (*profileScreen)(nil)

func TestProfileScreen_Title(t *testing.T) {
	s := newProfileScreen(newMockServices(), NewStyles(true), true)
	if got := s.Title(); got != "Profiles" {
		t.Fatalf("Title() = %q, want %q", got, "Profiles")
	}
}

func TestProfileScreen_InputActive_Create(t *testing.T) {
	s := newProfileScreen(newMockServices(), NewStyles(true), true)
	s.step = profileCreateName
	if !s.InputActive() {
		t.Fatal("InputActive() = false during create name input, want true")
	}
}

func TestProfileScreen_InputActive_Menu(t *testing.T) {
	s := newProfileScreen(newMockServices(), NewStyles(true), true)
	if s.InputActive() {
		t.Fatal("InputActive() = true on menu step, want false")
	}
}

func TestProfileScreen_InitReturnsCmd(t *testing.T) {
	s := newProfileScreen(newMockServices(), NewStyles(true), true)
	cmd := s.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil cmd")
	}
}

func TestProfileScreen_LoadingView(t *testing.T) {
	s := newProfileScreen(newMockServices(), NewStyles(true), true)
	v := s.View()
	if !strings.Contains(v.Content, "Loading") {
		t.Errorf("loading view should contain 'Loading', got: %q", v.Content)
	}
}

func TestProfileScreen_LoadError(t *testing.T) {
	s := newProfileScreen(newMockServices(), NewStyles(true), true)
	s.Update(profileLoadedMsg{err: fmt.Errorf("profile store missing")})

	v := s.View()
	if !strings.Contains(v.Content, "profile store missing") {
		t.Errorf("error view should show error, got: %q", v.Content)
	}
}

func TestProfileScreen_ListView(t *testing.T) {
	s := newProfileScreen(newMockServices(), NewStyles(true), true)
	profiles := []profile.ProfileSummary{
		{Name: "go-dev", AssetCount: 12},
		{Name: "python-work", AssetCount: 8},
	}
	s.Update(profileLoadedMsg{profiles: profiles, active: "go-dev"})
	s.step = profileList

	v := s.View()
	if !strings.Contains(v.Content, "go-dev") {
		t.Errorf("list view should show profile names, got: %q", v.Content)
	}
	if !strings.Contains(v.Content, "python-work") {
		t.Errorf("list view should show all profiles, got: %q", v.Content)
	}
}

func TestProfileScreen_ListViewActiveMarker(t *testing.T) {
	s := newProfileScreen(newMockServices(), NewStyles(true), true)
	profiles := []profile.ProfileSummary{{Name: "go-dev"}}
	s.Update(profileLoadedMsg{profiles: profiles, active: "go-dev"})
	s.step = profileList

	v := s.View()
	// Active profile should have a marker.
	if !strings.Contains(v.Content, "*") {
		t.Errorf("active profile should show '*' marker, got: %q", v.Content)
	}
}

func TestProfileScreen_SwitchDone_Success(t *testing.T) {
	s := newProfileScreen(newMockServices(), NewStyles(true), true)
	result := &profile.SwitchResult{
		ToProfile: "python-work",
		Diff: profile.SwitchDiff{
			Deploy: []profile.ProfileAsset{{AssetName: "py-rules"}},
			Remove: []profile.ProfileAsset{{AssetName: "go-linter"}},
		},
		Deployed: &deploy.BulkDeployResult{},
		Removed:  &deploy.BulkRemoveResult{},
	}
	s.Update(profileSwitchedMsg{result: result, err: nil})

	v := s.View()
	if !strings.Contains(v.Content, "python-work") {
		t.Errorf("switch result should mention target profile, got: %q", v.Content)
	}
}

func TestProfileScreen_SwitchDone_Error(t *testing.T) {
	s := newProfileScreen(newMockServices(), NewStyles(true), true)
	s.Update(profileSwitchedMsg{err: fmt.Errorf("profile not found")})

	v := s.View()
	if !strings.Contains(v.Content, "profile not found") {
		t.Errorf("switch error view should show error, got: %q", v.Content)
	}
}

func TestProfileScreen_RefreshHeaderAfterSwitch(t *testing.T) {
	s := newProfileScreen(newMockServices(), NewStyles(true), true)
	result := &profile.SwitchResult{ToProfile: "python-work"}
	_, cmd := s.Update(profileSwitchedMsg{result: result, err: nil})

	if cmd == nil {
		t.Fatal("switch done should emit a cmd")
	}
	msg := cmd()
	switch msg.(type) {
	case RefreshHeaderMsg:
		// OK
	default:
		t.Errorf("switch done should emit RefreshHeaderMsg, got %T", msg)
	}
}

func TestProfileScreen_CreateDone_Success(t *testing.T) {
	s := newProfileScreen(newMockServices(), NewStyles(true), true)
	s.Update(profileCreatedMsg{name: "new-profile", err: nil})

	v := s.View()
	if !strings.Contains(v.Content, "new-profile") {
		t.Errorf("create result should mention profile name, got: %q", v.Content)
	}
}

func TestProfileScreen_CreateDone_Error(t *testing.T) {
	s := newProfileScreen(newMockServices(), NewStyles(true), true)
	s.Update(profileCreatedMsg{err: fmt.Errorf("profile already exists")})

	v := s.View()
	if !strings.Contains(v.Content, "profile already exists") {
		t.Errorf("create error view should show error, got: %q", v.Content)
	}
}
