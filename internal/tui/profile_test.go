package tui

import (
	tea "charm.land/bubbletea/v2"
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
	s.step = profileMenu
	if !s.InputActive() {
		t.Fatal("InputActive() = false on menu step, want true (form active)")
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

// --- Viewport wrapping tests (Unit 5) ---

func TestProfileScreen_ScreenSizeMsg_StoresPending(t *testing.T) {
	s := newProfileScreen(newMockServices(), NewStyles(true), true)
	s.Update(ScreenSizeMsg{Width: 80, Height: 30})
	if s.pendingWidth != 80 || s.pendingHeight != 30 {
		t.Fatalf("expected pending 80x30, got %dx%d", s.pendingWidth, s.pendingHeight)
	}
}

func TestProfileScreen_ScreenSizeMsg_UpdatesExistingViewport(t *testing.T) {
	s := newProfileScreen(newMockServices(), NewStyles(true), true)
	s.Update(ScreenSizeMsg{Width: 80, Height: 30})
	s.Update(profileLoadedMsg{profiles: []profile.ProfileSummary{{Name: "p1", AssetCount: 5}}, active: "p1"})
	s.step = profileList
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

func TestProfileScreen_ListViewport_InitAppliesPendingDimensions(t *testing.T) {
	s := newProfileScreen(newMockServices(), NewStyles(true), true)
	s.Update(ScreenSizeMsg{Width: 100, Height: 40})
	s.profiles = []profile.ProfileSummary{{Name: "p1"}}
	s.step = profileList
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

func TestProfileScreen_ListViewShowsContent(t *testing.T) {
	s := newProfileScreen(newMockServices(), NewStyles(true), true)
	s.Update(ScreenSizeMsg{Width: 80, Height: 30})
	profiles := []profile.ProfileSummary{
		{Name: "go-dev", AssetCount: 12},
		{Name: "python-work", AssetCount: 8},
	}
	s.Update(profileLoadedMsg{profiles: profiles, active: "go-dev"})
	s.step = profileList
	s.initListViewport()

	v := s.View()
	if !strings.Contains(v.Content, "go-dev") {
		t.Errorf("list view should show profile names, got: %q", v.Content)
	}
	if !strings.Contains(v.Content, "python-work") {
		t.Errorf("list view should show all profiles, got: %q", v.Content)
	}
}

func TestProfileScreen_ListViewActiveMarkerInViewport(t *testing.T) {
	s := newProfileScreen(newMockServices(), NewStyles(true), true)
	s.Update(ScreenSizeMsg{Width: 80, Height: 30})
	profiles := []profile.ProfileSummary{{Name: "go-dev"}}
	s.Update(profileLoadedMsg{profiles: profiles, active: "go-dev"})
	s.step = profileList
	s.initListViewport()

	v := s.View()
	if !strings.Contains(v.Content, "*") {
		t.Errorf("active profile should show '*' marker, got: %q", v.Content)
	}
}

func TestProfileScreen_ListViewFooterVisible(t *testing.T) {
	s := newProfileScreen(newMockServices(), NewStyles(true), true)
	s.Update(ScreenSizeMsg{Width: 80, Height: 30})
	profiles := []profile.ProfileSummary{{Name: "p1", AssetCount: 1}}
	s.Update(profileLoadedMsg{profiles: profiles, active: "p1"})
	s.step = profileList
	s.initListViewport()

	v := s.View()
	if !strings.Contains(v.Content, "Press esc to go back.") {
		t.Errorf("footer hint should be visible, got: %q", v.Content)
	}
}

func TestProfileScreen_ListViewManyProfilesFooterStillVisible(t *testing.T) {
	s := newProfileScreen(newMockServices(), NewStyles(true), true)
	s.Update(ScreenSizeMsg{Width: 80, Height: 5})
	profiles := make([]profile.ProfileSummary, 50)
	for i := range profiles {
		profiles[i] = profile.ProfileSummary{Name: fmt.Sprintf("prof-%03d", i), AssetCount: i}
	}
	s.Update(profileLoadedMsg{profiles: profiles, active: "prof-000"})
	s.step = profileList
	s.initListViewport()

	v := s.View()
	if !strings.Contains(v.Content, "Press esc to go back.") {
		t.Errorf("footer hint should remain visible with overflow, got: %q", v.Content)
	}
}

func TestProfileScreen_UpdateListForwardsToViewport(t *testing.T) {
	s := newProfileScreen(newMockServices(), NewStyles(true), true)
	s.Update(ScreenSizeMsg{Width: 80, Height: 30})
	profiles := make([]profile.ProfileSummary, 50)
	for i := range profiles {
		profiles[i] = profile.ProfileSummary{Name: fmt.Sprintf("prof-%03d", i), AssetCount: i}
	}
	s.Update(profileLoadedMsg{profiles: profiles, active: "prof-000"})
	s.step = profileList
	s.initListViewport()

	_, cmd := s.Update(tea.KeyPressMsg{Code: 106}) // 'j'
	_ = cmd
}
