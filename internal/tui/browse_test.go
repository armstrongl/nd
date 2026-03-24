package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/armstrongl/nd/internal/asset"
	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/sourcemanager"
)

// Compile-time check: browseScreen satisfies Screen.
var _ Screen = (*browseScreen)(nil)

func TestBrowseScreen_Title(t *testing.T) {
	s := newBrowseScreen(newMockServices(), NewStyles(true), true)
	if got := s.Title(); got != "Browse" {
		t.Fatalf("Title() = %q, want %q", got, "Browse")
	}
}

func TestBrowseScreen_InputActive_WhenFiltering(t *testing.T) {
	s := newBrowseScreen(newMockServices(), NewStyles(true), true)
	s.filtering = true
	if !s.InputActive() {
		t.Fatal("InputActive() = false while filtering, want true")
	}
}

func TestBrowseScreen_InputActive_WhenNotFiltering(t *testing.T) {
	s := newBrowseScreen(newMockServices(), NewStyles(true), true)
	s.filtering = false
	if s.InputActive() {
		t.Fatal("InputActive() = true when not filtering, want false")
	}
}

func TestBrowseScreen_InitReturnsCmd(t *testing.T) {
	s := newBrowseScreen(newMockServices(), NewStyles(true), true)
	cmd := s.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil cmd")
	}
}

func TestBrowseScreen_LoadingView(t *testing.T) {
	s := newBrowseScreen(newMockServices(), NewStyles(true), true)
	v := s.View()
	if !strings.Contains(v.Content, "Loading") {
		t.Errorf("loading view should contain 'Loading', got: %q", v.Content)
	}
}

func TestBrowseScreen_ViewWithAssets(t *testing.T) {
	svc := newMockServices()
	assets := []asset.Asset{
		{Identity: asset.Identity{SourceID: "src1", Type: nd.AssetSkill, Name: "my-skill"}},
		{Identity: asset.Identity{SourceID: "src1", Type: nd.AssetRule, Name: "my-rule"}},
	}
	idx := asset.NewIndex(assets)
	svc.scanIndexFn = func() (*sourcemanager.ScanSummary, error) {
		return &sourcemanager.ScanSummary{Index: idx}, nil
	}

	s := newBrowseScreen(svc, NewStyles(true), true)
	s.Update(browseLoadedMsg{assets: idx.All()})

	v := s.View()
	if !strings.Contains(v.Content, "my-skill") {
		t.Errorf("view should show asset names, got: %q", v.Content)
	}
	if !strings.Contains(v.Content, "my-rule") {
		t.Errorf("view should show all assets, got: %q", v.Content)
	}
}

func TestBrowseScreen_ViewNoAssets(t *testing.T) {
	s := newBrowseScreen(newMockServices(), NewStyles(true), true)
	s.Update(browseLoadedMsg{assets: nil})

	v := s.View()
	if !strings.Contains(v.Content, "No assets") {
		t.Errorf("empty view should mention no assets, got: %q", v.Content)
	}
}

func TestBrowseScreen_FilterNarrowsResults(t *testing.T) {
	svc := newMockServices()
	assets := []asset.Asset{
		{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "deploy-helper"}},
		{Identity: asset.Identity{SourceID: "s", Type: nd.AssetRule, Name: "python-rules"}},
	}
	idx := asset.NewIndex(assets)
	svc.scanIndexFn = func() (*sourcemanager.ScanSummary, error) {
		return &sourcemanager.ScanSummary{Index: idx}, nil
	}

	s := newBrowseScreen(svc, NewStyles(true), true)
	s.Update(browseLoadedMsg{assets: idx.All()})

	// Apply filter "deploy"
	s.filter = "deploy"

	v := s.View()
	if !strings.Contains(v.Content, "deploy-helper") {
		t.Errorf("filter 'deploy' should show deploy-helper, got: %q", v.Content)
	}
	if strings.Contains(v.Content, "python-rules") {
		t.Errorf("filter 'deploy' should hide python-rules, got: %q", v.Content)
	}
}

func TestBrowseScreen_SlashEntersFilterMode(t *testing.T) {
	s := newBrowseScreen(newMockServices(), NewStyles(true), true)
	s.Update(browseLoadedMsg{assets: nil})

	s.Update(tea.KeyPressMsg(tea.Key{Code: '/', Text: "/"}))

	if !s.filtering {
		t.Fatal("pressing / should enter filter mode")
	}
}

func TestBrowseScreen_EscExitsFilterMode(t *testing.T) {
	s := newBrowseScreen(newMockServices(), NewStyles(true), true)
	s.filtering = true
	s.filter = "foo"

	s.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}))

	if s.filtering {
		t.Fatal("esc should exit filter mode")
	}
	if s.filter != "" {
		t.Fatalf("esc should clear filter, got: %q", s.filter)
	}
}

func TestBrowseScreen_FilterCharsAppend(t *testing.T) {
	s := newBrowseScreen(newMockServices(), NewStyles(true), true)
	s.filtering = true

	s.Update(tea.KeyPressMsg(tea.Key{Code: 'a', Text: "a"}))
	s.Update(tea.KeyPressMsg(tea.Key{Code: 'b', Text: "b"}))
	s.Update(tea.KeyPressMsg(tea.Key{Code: 'c', Text: "c"}))

	if s.filter != "abc" {
		t.Fatalf("filter should be %q, got %q", "abc", s.filter)
	}
}

func TestBrowseScreen_BackspaceDeletesFilterChar(t *testing.T) {
	s := newBrowseScreen(newMockServices(), NewStyles(true), true)
	s.filtering = true
	s.filter = "foo"

	s.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyBackspace}))

	if s.filter != "fo" {
		t.Fatalf("backspace should remove last char, got %q", s.filter)
	}
}

func TestBrowseScreen_DeployedMarkerShown(t *testing.T) {
	assets := []asset.Asset{
		{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "deployed-skill"}},
		{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "available-skill"}},
	}
	idx := asset.NewIndex(assets)

	s := newBrowseScreen(newMockServices(), NewStyles(true), true)
	deployedKey := "s:skills/deployed-skill"
	s.Update(browseLoadedMsg{
		assets:   idx.All(),
		deployed: map[string]bool{deployedKey: true},
	})

	v := s.View()
	// deployed assets show a marker (e.g. "*")
	if !strings.Contains(v.Content, "*") {
		t.Errorf("deployed asset should show '*' marker, got: %q", v.Content)
	}
}

func TestBrowseScreen_TypeAndSourceShown(t *testing.T) {
	assets := []asset.Asset{
		{Identity: asset.Identity{SourceID: "my-source", Type: nd.AssetRule, Name: "go-rules"}},
	}
	idx := asset.NewIndex(assets)

	s := newBrowseScreen(newMockServices(), NewStyles(true), true)
	s.Update(browseLoadedMsg{assets: idx.All()})

	v := s.View()
	if !strings.Contains(v.Content, string(nd.AssetRule)) {
		t.Errorf("view should show asset type, got: %q", v.Content)
	}
	if !strings.Contains(v.Content, "my-source") {
		t.Errorf("view should show source ID, got: %q", v.Content)
	}
}
