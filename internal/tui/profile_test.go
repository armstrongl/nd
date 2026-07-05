package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/armstrongl/nd/internal/agent"
	"github.com/armstrongl/nd/internal/asset"
	"github.com/armstrongl/nd/internal/deploy"
	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/profile"
	"github.com/armstrongl/nd/internal/sourcemanager"
	"github.com/armstrongl/nd/internal/state"
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
	// Active profile should have the shared active marker.
	if !strings.Contains(v.Content, GlyphActive) {
		t.Errorf("active profile should show active glyph %q, got: %q", GlyphActive, v.Content)
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

// double-fire guard — creating flag prevents repeated runCreate calls
func TestProfileScreen_DoubleFireGuard_Create(t *testing.T) {
	s := newProfileScreen(newMockServices(), NewStyles(true), true)
	s.step = profileCreateName
	s.creating = true

	_, cmd := s.updateCreateForm(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("updateCreateForm should return nil cmd when creating guard is set")
	}
}

func TestProfileScreen_CreateReloadsList(t *testing.T) {
	s := newProfileScreen(newMockServices(), NewStyles(true), true)

	// A successful create should emit a reload command.
	_, cmd := s.Update(profileCreatedMsg{name: "new-profile", err: nil})
	if cmd == nil {
		t.Fatal("create success should emit a reload cmd")
	}
	msg := cmd()
	switch msg.(type) {
	case profileLoadedMsg:
		// OK: create triggers a fresh load of the profile list.
	default:
		t.Fatalf("reload cmd should yield profileLoadedMsg, got %T", msg)
	}
	if s.step != profileDone {
		t.Fatalf("step = %v after create, want profileDone", s.step)
	}

	// The reload result arriving while the confirmation is shown should refresh
	// the cached slice without bouncing the user back to the menu.
	profiles := []profile.ProfileSummary{{Name: "new-profile", AssetCount: 3}}
	s.Update(profileLoadedMsg{profiles: profiles, active: "new-profile"})

	if len(s.profiles) != 1 || s.profiles[0].Name != "new-profile" {
		t.Fatalf("profiles not reloaded on done screen: %+v", s.profiles)
	}
	if s.step != profileDone {
		t.Fatalf("step = %v after reload, want profileDone preserved", s.step)
	}
	v := s.View()
	if !strings.Contains(v.Content, "created.") {
		t.Errorf("done view should still show success text, got: %q", v.Content)
	}
}

// taskmd k63tsg: a project-scope profile switch must thread the resolved project
// root (from GetProjectRoot) through mgr.Switch into the deploy requests. The
// threaded root is observable as the deployed Deployment's ProjectPath, even
// when the TUI was launched in the default global scope inside a project.
func TestProfileScreen_RunSwitch_ProjectScopeThreadsProjectRoot(t *testing.T) {
	projectRoot := t.TempDir()

	pstore := profile.NewStore(
		filepath.Join(t.TempDir(), "profiles"),
		filepath.Join(t.TempDir(), "snapshots"),
	)
	if err := pstore.CreateProfile(profile.Profile{Version: 1, Name: "base"}); err != nil {
		t.Fatalf("create base profile: %v", err)
	}
	if err := pstore.CreateProfile(profile.Profile{
		Version: 1,
		Name:    "work",
		Assets: []profile.ProfileAsset{{
			SourceID:  "local",
			AssetType: nd.AssetSkill,
			AssetName: "demo",
			Scope:     nd.ScopeProject,
		}},
	}); err != nil {
		t.Fatalf("create work profile: %v", err)
	}

	// Parent dir must already exist so the state file lock can be acquired.
	sstore := state.NewStore(filepath.Join(t.TempDir(), "deployments.yaml"))
	mgr := profile.NewManager(pstore, sstore)

	// Agent that supports skills so switching produces a real deployment record.
	ag := &agent.Agent{
		Name:           "claude-code",
		ProjectDir:     ".claude",
		SupportedTypes: []nd.AssetType{nd.AssetSkill},
	}
	eng := deploy.New(sstore, ag, "")

	idx := asset.NewIndex([]asset.Asset{{
		Identity:   asset.Identity{SourceID: "local", Type: nd.AssetSkill, Name: "demo"},
		SourcePath: filepath.Join(t.TempDir(), "src", "demo"),
	}})

	svc := newMockServices()
	svc.getScopeFn = func() nd.Scope { return nd.ScopeProject }
	svc.getProjectRootFn = func() string { return projectRoot }
	svc.profileManagerFn = func() (*profile.Manager, error) { return mgr, nil }
	svc.deployEngineFn = func() (*deploy.Engine, error) { return eng, nil }
	svc.scanIndexFn = func() (*sourcemanager.ScanSummary, error) {
		return &sourcemanager.ScanSummary{Index: idx}, nil
	}

	s := newProfileScreen(svc, NewStyles(true), true)
	s.active = "base"
	s.switchChoice = "work"

	msg, ok := s.runSwitch()().(profileSwitchedMsg)
	if !ok {
		t.Fatal("runSwitch cmd did not return a profileSwitchedMsg")
	}
	if msg.err != nil {
		t.Fatalf("switch returned error: %v", msg.err)
	}
	if msg.result == nil || msg.result.Deployed == nil || len(msg.result.Deployed.Succeeded) == 0 {
		t.Fatalf("expected a deployed asset, got result %+v", msg.result)
	}
	if got := msg.result.Deployed.Succeeded[0].Deployment.ProjectPath; got != projectRoot {
		t.Errorf("deployed ProjectPath = %q, want resolved project root %q", got, projectRoot)
	}
}
