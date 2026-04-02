package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/armstrongl/nd/internal/asset"
	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/sourcemanager"
)

// Compile-time check: browseScreen satisfies Screen.
var _ Screen = (*browseScreen)(nil)

// Compile-time check: browseScreen implements FullHelpProvider.
var _ FullHelpProvider = (*browseScreen)(nil)

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

// --- Cursor navigation tests ---

// browseSeedAssets is a test helper that creates a browseScreen loaded with N test assets.
func browseSeedAssets(t *testing.T, n int, deployed map[string]bool) *browseScreen {
	t.Helper()
	raw := make([]asset.Asset, n)
	for i := range n {
		raw[i] = asset.Asset{
			Identity: asset.Identity{
				SourceID: "src",
				Type:     nd.AssetSkill,
				Name:     fmt.Sprintf("asset-%d", i),
			},
		}
	}
	idx := asset.NewIndex(raw)
	s := newBrowseScreen(newMockServices(), NewStyles(true), true)
	s.Update(browseLoadedMsg{assets: idx.All(), deployed: deployed})
	return s
}

func TestBrowseScreen_CursorStartsAtZero(t *testing.T) {
	s := browseSeedAssets(t, 3, nil)
	if s.cursor != 0 {
		t.Fatalf("cursor = %d, want 0", s.cursor)
	}
}

func TestBrowseScreen_CursorMovesDownWithJ(t *testing.T) {
	s := browseSeedAssets(t, 3, nil)
	s.Update(tea.KeyPressMsg(tea.Key{Code: 'j', Text: "j"}))
	if s.cursor != 1 {
		t.Fatalf("cursor = %d after j, want 1", s.cursor)
	}
}

func TestBrowseScreen_CursorMovesDownWithArrow(t *testing.T) {
	s := browseSeedAssets(t, 3, nil)
	s.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	if s.cursor != 1 {
		t.Fatalf("cursor = %d after down, want 1", s.cursor)
	}
}

func TestBrowseScreen_CursorMovesUpWithK(t *testing.T) {
	s := browseSeedAssets(t, 3, nil)
	s.cursor = 2
	s.Update(tea.KeyPressMsg(tea.Key{Code: 'k', Text: "k"}))
	if s.cursor != 1 {
		t.Fatalf("cursor = %d after k, want 1", s.cursor)
	}
}

func TestBrowseScreen_CursorMovesUpWithArrow(t *testing.T) {
	s := browseSeedAssets(t, 3, nil)
	s.cursor = 2
	s.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp}))
	if s.cursor != 1 {
		t.Fatalf("cursor = %d after up, want 1", s.cursor)
	}
}

func TestBrowseScreen_CursorClampsAtTop(t *testing.T) {
	s := browseSeedAssets(t, 3, nil)
	s.Update(tea.KeyPressMsg(tea.Key{Code: 'k', Text: "k"}))
	if s.cursor != 0 {
		t.Fatalf("cursor = %d after k at top, want 0", s.cursor)
	}
}

func TestBrowseScreen_CursorClampsAtBottom(t *testing.T) {
	s := browseSeedAssets(t, 3, nil)
	s.cursor = 2
	s.Update(tea.KeyPressMsg(tea.Key{Code: 'j', Text: "j"}))
	if s.cursor != 2 {
		t.Fatalf("cursor = %d after j at bottom, want 2", s.cursor)
	}
}

func TestBrowseScreen_CursorResetOnFilterChange(t *testing.T) {
	raw := []asset.Asset{
		{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "alpha"}},
		{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "bravo"}},
		{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "charlie"}},
	}
	idx := asset.NewIndex(raw)
	s := newBrowseScreen(newMockServices(), NewStyles(true), true)
	s.Update(browseLoadedMsg{assets: idx.All()})

	// Move cursor to 2, then apply filter that shows fewer items.
	s.cursor = 2
	s.filtering = true
	s.Update(tea.KeyPressMsg(tea.Key{Code: 'b', Text: "b"})) // filter = "b" -> only "bravo"

	if s.cursor >= len(s.visibleAssets()) {
		t.Fatalf("cursor = %d should be clamped to visible count %d",
			s.cursor, len(s.visibleAssets()))
	}
}

func TestBrowseScreen_ViewShowsCursorIndicator(t *testing.T) {
	s := browseSeedAssets(t, 3, nil)
	s.cursor = 1
	v := s.View()
	if !strings.Contains(v.Content, GlyphArrow) {
		t.Errorf("view should show cursor glyph %q, got: %q", GlyphArrow, v.Content)
	}
}

func TestBrowseScreen_EnterEmitsNavigateMsg(t *testing.T) {
	s := browseSeedAssets(t, 3, nil)
	_, cmd := s.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd == nil {
		t.Fatal("enter should produce a cmd, got nil")
	}
	msg := cmd()
	nav, ok := msg.(NavigateMsg)
	if !ok {
		t.Fatalf("enter cmd produced %T, want NavigateMsg", msg)
	}
	if _, ok := nav.Screen.(*deployScreen); !ok {
		t.Fatalf("NavigateMsg.Screen is %T, want *deployScreen", nav.Screen)
	}
}

func TestBrowseScreen_EnterOnDeployedAssetShowsMessage(t *testing.T) {
	raw := []asset.Asset{
		{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "deployed-one"}},
	}
	idx := asset.NewIndex(raw)
	key := raw[0].String()
	s := newBrowseScreen(newMockServices(), NewStyles(true), true)
	s.Update(browseLoadedMsg{assets: idx.All(), deployed: map[string]bool{key: true}})

	_, cmd := s.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))

	// Should not navigate — the asset is already deployed.
	if cmd != nil {
		t.Fatal("enter on deployed asset should return nil cmd, not navigate")
	}

	// Should show feedback in the view.
	v := s.View()
	if !strings.Contains(v.Content, "already deployed") {
		t.Errorf("view should show 'already deployed' message, got: %q", v.Content)
	}
}

func TestBrowseScreen_EnterOnEmptyListDoesNothing(t *testing.T) {
	s := newBrowseScreen(newMockServices(), NewStyles(true), true)
	s.Update(browseLoadedMsg{assets: nil})

	_, cmd := s.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd != nil {
		t.Fatal("enter on empty list should produce nil cmd")
	}
}

// --- FullHelpProvider tests ---

func TestBrowseScreen_FullHelpItems_Default(t *testing.T) {
	s := newBrowseScreen(newMockServices(), NewStyles(true), true)
	items := s.FullHelpItems()

	// Should include navigation keys, /, and enter.
	var keys []string
	for _, item := range items {
		keys = append(keys, item.Key)
	}
	joined := strings.Join(keys, " ")
	for _, want := range []string{"j/k", "enter", "/"} {
		if !strings.Contains(joined, want) {
			t.Errorf("FullHelpItems missing %q, got keys: %v", want, keys)
		}
	}
}

func TestBrowseScreen_FullHelpItems_Filtering(t *testing.T) {
	s := newBrowseScreen(newMockServices(), NewStyles(true), true)
	s.filtering = true
	items := s.FullHelpItems()

	if len(items) != 3 {
		t.Fatalf("filtering help should have 3 items, got %d: %v", len(items), items)
	}

	var keys []string
	for _, item := range items {
		keys = append(keys, item.Key)
	}
	joined := strings.Join(keys, " ")
	// Positive: filtering-specific keys present.
	for _, want := range []string{"esc", "enter", "backspace"} {
		if !strings.Contains(joined, want) {
			t.Errorf("filtering help should include %q, got: %v", want, keys)
		}
	}
	// Negative: default navigation keys must be absent.
	for _, absent := range []string{"j/k", "/", "q"} {
		if strings.Contains(joined, absent) {
			t.Errorf("filtering help should NOT include %q, got: %v", absent, keys)
		}
	}
}

func TestBrowseScreen_ErrorPathRendersError(t *testing.T) {
	s := newBrowseScreen(newMockServices(), NewStyles(true), true)
	s.Update(browseLoadedMsg{err: fmt.Errorf("scan failed")})

	if s.err == nil {
		t.Fatal("err should be set after error browseLoadedMsg")
	}

	v := s.View()
	if !strings.Contains(v.Content, "scan failed") {
		t.Errorf("error view should contain error message, got: %q", v.Content)
	}
	if !strings.Contains(v.Content, "esc") {
		t.Errorf("error view should mention esc to go back, got: %q", v.Content)
	}
}

func TestBrowseScreen_EnterInFilterModePreservesFilter(t *testing.T) {
	s := browseSeedAssets(t, 3, nil)
	s.filtering = true
	s.filter = "asset"

	s.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))

	if s.filtering {
		t.Fatal("enter should exit filter mode")
	}
	if s.filter != "asset" {
		t.Fatalf("enter should preserve filter text, got %q", s.filter)
	}
}

func TestBrowseScreen_KeysIgnoredBeforeLoad(t *testing.T) {
	s := newBrowseScreen(newMockServices(), NewStyles(true), true)
	// Not loaded yet — keys should be no-ops.
	_, cmd := s.Update(tea.KeyPressMsg(tea.Key{Code: 'j', Text: "j"}))
	if cmd != nil {
		t.Fatal("keys before load should produce nil cmd")
	}
	if s.filtering {
		t.Fatal("/ before load should not enter filter mode")
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
