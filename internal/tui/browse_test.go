package tui

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/list"

	"github.com/armstrongl/nd/internal/asset"
	"github.com/armstrongl/nd/internal/nd"
)

// ansiRe matches ANSI escape sequences for stripping in tests.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// stripANSI removes ANSI escape sequences from a string for deterministic assertions.
func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

// Compile-time check: browseScreen satisfies Screen and HelpProvider.
var _ Screen = (*browseScreen)(nil)
var _ HelpProvider = (*browseScreen)(nil)

// tryRunCmd executes a tea.Cmd with a short timeout. Returns nil if the cmd
// blocks (e.g. textinput.Blink timer). This prevents tests from stalling on
// timer-based commands.
func tryRunCmd(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	ch := make(chan tea.Msg, 1)
	go func() { ch <- cmd() }()
	select {
	case msg := <-ch:
		return msg
	case <-time.After(10 * time.Millisecond):
		return nil
	}
}

// sendMsg sends a message to the browse screen and drains any returned commands
// by executing them and feeding the resulting messages back, up to maxDrain times.
// Commands that block (e.g. blink timers) are skipped via a short timeout.
func sendMsg(s *browseScreen, msg tea.Msg, maxDrain int) {
	_, cmd := s.Update(msg)
	for i := 0; i < maxDrain && cmd != nil; i++ {
		result := tryRunCmd(cmd)
		if result == nil {
			break
		}
		// tea.Batch returns a BatchMsg (slice of Cmds).
		if batch, ok := result.(tea.BatchMsg); ok {
			for _, bc := range batch {
				bResult := tryRunCmd(bc)
				if bResult != nil {
					_, cmd = s.Update(bResult)
				}
			}
			continue
		}
		_, cmd = s.Update(result)
	}
}

func TestBrowseScreen_Title(t *testing.T) {
	s := newBrowseScreen(newMockServices(), testStyles(), true)
	if got := s.Title(); got != "Browse" {
		t.Fatalf("Title() = %q, want %q", got, "Browse")
	}
}

func TestBrowseScreen_InputActive_DefaultFalse(t *testing.T) {
	s := newBrowseScreen(newMockServices(), testStyles(), true)
	if s.InputActive() {
		t.Fatal("InputActive() = true before list created, want false")
	}
}

func TestBrowseScreen_InputActive_UnfilteredFalse(t *testing.T) {
	s := newBrowseScreen(newMockServices(), testStyles(), true)
	sendMsg(s, ScreenSizeMsg{Width: 80, Height: 40}, 0)
	sendMsg(s, browseLoadedMsg{assets: makeTestAssets("a"), deployed: nil}, 0)
	if s.InputActive() {
		t.Fatal("InputActive() = true when unfiltered, want false")
	}
}

func TestBrowseScreen_InitReturnsCmd(t *testing.T) {
	s := newBrowseScreen(newMockServices(), testStyles(), true)
	cmd := s.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil cmd")
	}
}

func TestBrowseScreen_LoadingView(t *testing.T) {
	s := newBrowseScreen(newMockServices(), testStyles(), true)
	v := s.View()
	if !strings.Contains(v.Content, "Loading") {
		t.Errorf("loading view should contain 'Loading', got: %q", v.Content)
	}
}

func TestBrowseScreen_ViewWithAssets(t *testing.T) {
	s := newBrowseScreen(newMockServices(), testStyles(), true)
	sendMsg(s, ScreenSizeMsg{Width: 80, Height: 40}, 0)
	sendMsg(s, browseLoadedMsg{
		assets:   makeTestAssets("my-skill", "my-rule"),
		deployed: nil,
	}, 0)

	v := s.View()
	if !strings.Contains(v.Content, "my-skill") {
		t.Errorf("view should show asset names, got: %q", v.Content)
	}
	if !strings.Contains(v.Content, "my-rule") {
		t.Errorf("view should show all assets, got: %q", v.Content)
	}
}

func TestBrowseScreen_ViewNoAssets(t *testing.T) {
	s := newBrowseScreen(newMockServices(), testStyles(), true)
	sendMsg(s, browseLoadedMsg{assets: nil}, 0)

	v := s.View()
	if !strings.Contains(v.Content, "No assets") {
		t.Errorf("empty view should mention no assets, got: %q", v.Content)
	}
}

func TestBrowseScreen_FilterNarrowsResults(t *testing.T) {
	s := newBrowseScreen(newMockServices(), testStyles(), true)
	sendMsg(s, ScreenSizeMsg{Width: 80, Height: 40}, 0)
	sendMsg(s, browseLoadedMsg{
		assets:   makeTestAssets("deploy-helper", "python-rules"),
		deployed: nil,
	}, 0)

	// Enter filter mode with '/'
	sendMsg(s, tea.KeyPressMsg(tea.Key{Code: '/', Text: "/"}), 5)

	// Type "deploy" — each character triggers async FilterMatchesMsg
	for _, ch := range "deploy" {
		sendMsg(s, tea.KeyPressMsg(tea.Key{Code: rune(ch), Text: string(ch)}), 5)
	}

	// Accept filter with enter
	sendMsg(s, tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}), 5)

	v := s.View()
	plain := stripANSI(v.Content)
	if !strings.Contains(plain, "deploy-helper") {
		t.Errorf("filter 'deploy' should show deploy-helper, got: %q", plain)
	}
	if strings.Contains(plain, "python-rules") {
		t.Errorf("filter 'deploy' should hide python-rules, got: %q", plain)
	}
}

func TestBrowseScreen_EscClearsAppliedFilter(t *testing.T) {
	s := newBrowseScreen(newMockServices(), testStyles(), true)
	sendMsg(s, ScreenSizeMsg{Width: 80, Height: 40}, 0)
	sendMsg(s, browseLoadedMsg{
		assets:   makeTestAssets("alpha", "beta"),
		deployed: nil,
	}, 0)

	// Enter filter, type text, accept with enter
	sendMsg(s, tea.KeyPressMsg(tea.Key{Code: '/', Text: "/"}), 5)
	sendMsg(s, tea.KeyPressMsg(tea.Key{Code: 'a', Text: "a"}), 5)
	sendMsg(s, tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}), 5)

	// Filter should be applied (InputActive true)
	if !s.InputActive() {
		t.Fatal("InputActive() should be true with filter applied")
	}

	// Esc clears the filter (does not navigate back — the list handles it)
	sendMsg(s, tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}), 5)

	if s.InputActive() {
		t.Fatal("InputActive() should be false after esc clears filter")
	}

	// Both items should be visible again
	v := s.View()
	if !strings.Contains(v.Content, "alpha") || !strings.Contains(v.Content, "beta") {
		t.Errorf("esc should clear filter, showing all items, got: %q", v.Content)
	}
}

func TestBrowseScreen_EscNavigatesBackWhenUnfiltered(t *testing.T) {
	s := newBrowseScreen(newMockServices(), testStyles(), true)
	sendMsg(s, ScreenSizeMsg{Width: 80, Height: 40}, 0)
	sendMsg(s, browseLoadedMsg{
		assets:   makeTestAssets("alpha"),
		deployed: nil,
	}, 0)

	// When no filter is active, InputActive is false so the root model
	// handles esc as back navigation.
	if s.InputActive() {
		t.Fatal("InputActive() should be false when unfiltered")
	}
}

func TestBrowseScreen_QDoesNotQuitList(t *testing.T) {
	s := newBrowseScreen(newMockServices(), testStyles(), true)
	sendMsg(s, ScreenSizeMsg{Width: 80, Height: 40}, 0)
	sendMsg(s, browseLoadedMsg{
		assets:   makeTestAssets("alpha"),
		deployed: nil,
	}, 0)

	_, cmd := s.Update(tea.KeyPressMsg(tea.Key{Code: 'q', Text: "q"}))

	// DisableQuitKeybindings means q should not produce tea.Quit
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(tea.QuitMsg); ok {
			t.Fatal("q should not quit the list (DisableQuitKeybindings)")
		}
	}
}

func TestBrowseScreen_ScreenSizeBeforeLoad(t *testing.T) {
	s := newBrowseScreen(newMockServices(), testStyles(), true)

	// Send dimensions before data loads.
	sendMsg(s, ScreenSizeMsg{Width: 120, Height: 50}, 0)

	if s.pendingWidth != 120 || s.pendingHeight != 50 {
		t.Fatalf("pending dimensions should be stored, got %dx%d", s.pendingWidth, s.pendingHeight)
	}

	// Now load data — list should be created with the pending dimensions.
	sendMsg(s, browseLoadedMsg{
		assets:   makeTestAssets("test-asset"),
		deployed: nil,
	}, 0)

	if s.list == nil {
		t.Fatal("list should be created after browseLoadedMsg")
	}
	if s.list.Width() != 120 || s.list.Height() != 50 {
		t.Fatalf("list dimensions should match pending, got %dx%d", s.list.Width(), s.list.Height())
	}
}

func TestBrowseScreen_ScreenSizeAfterLoad(t *testing.T) {
	s := newBrowseScreen(newMockServices(), testStyles(), true)
	sendMsg(s, ScreenSizeMsg{Width: 80, Height: 40}, 0)
	sendMsg(s, browseLoadedMsg{
		assets:   makeTestAssets("test-asset"),
		deployed: nil,
	}, 0)

	// Resize after list is created.
	sendMsg(s, ScreenSizeMsg{Width: 100, Height: 30}, 0)

	if s.list.Width() != 100 || s.list.Height() != 30 {
		t.Fatalf("list should be resized, got %dx%d", s.list.Width(), s.list.Height())
	}
}

func TestBrowseScreen_EmptyStateNoAssetsMessage(t *testing.T) {
	s := newBrowseScreen(newMockServices(), testStyles(), true)
	sendMsg(s, browseLoadedMsg{assets: nil}, 0)

	v := s.View()
	if !strings.Contains(v.Content, "No assets") {
		t.Errorf("empty state should show NoAssets message, got: %q", v.Content)
	}
}

func TestBrowseScreen_DeployedMarkerShown(t *testing.T) {
	assets := makeTestAssets("deployed-skill", "available-skill")
	deployedKey := assets[0].String()

	s := newBrowseScreen(newMockServices(), testStyles(), true)
	sendMsg(s, ScreenSizeMsg{Width: 80, Height: 40}, 0)
	sendMsg(s, browseLoadedMsg{
		assets:   assets,
		deployed: map[string]bool{deployedKey: true},
	}, 0)

	v := s.View()
	if !strings.Contains(v.Content, "*") {
		t.Errorf("deployed asset should show '*' marker, got: %q", v.Content)
	}
}

func TestBrowseScreen_TypeAndSourceShown(t *testing.T) {
	s := newBrowseScreen(newMockServices(), testStyles(), true)
	sendMsg(s, ScreenSizeMsg{Width: 80, Height: 40}, 0)
	sendMsg(s, browseLoadedMsg{
		assets: []*asset.Asset{
			{Identity: asset.Identity{SourceID: "my-source", Type: nd.AssetRule, Name: "go-rules"}},
		},
	}, 0)

	v := s.View()
	if !strings.Contains(v.Content, string(nd.AssetRule)) {
		t.Errorf("view should show asset type, got: %q", v.Content)
	}
	if !strings.Contains(v.Content, "my-source") {
		t.Errorf("view should show source ID, got: %q", v.Content)
	}
}

func TestBrowseScreen_HelpItems(t *testing.T) {
	s := newBrowseScreen(newMockServices(), testStyles(), true)
	items := s.HelpItems()
	if len(items) == 0 {
		t.Fatal("HelpItems() returned empty slice")
	}
	found := false
	for _, item := range items {
		if item.Key == "/" {
			found = true
		}
	}
	if !found {
		t.Error("HelpItems() should contain '/' filter key")
	}
}

func TestBrowseScreen_ErrorView(t *testing.T) {
	s := newBrowseScreen(newMockServices(), testStyles(), true)
	sendMsg(s, browseLoadedMsg{err: fmt.Errorf("scan failed")}, 0)

	v := s.View()
	if !strings.Contains(v.Content, "Error") {
		t.Errorf("error view should contain 'Error', got: %q", v.Content)
	}
	if !strings.Contains(v.Content, "scan failed") {
		t.Errorf("error view should contain error message, got: %q", v.Content)
	}
}

func TestBrowseScreen_FilterStateInputActive(t *testing.T) {
	s := newBrowseScreen(newMockServices(), testStyles(), true)
	sendMsg(s, ScreenSizeMsg{Width: 80, Height: 40}, 0)
	sendMsg(s, browseLoadedMsg{
		assets:   makeTestAssets("alpha", "beta"),
		deployed: nil,
	}, 0)

	// Before filtering: InputActive false
	if s.InputActive() {
		t.Fatal("InputActive() should be false when unfiltered")
	}

	// Start filtering with /
	sendMsg(s, tea.KeyPressMsg(tea.Key{Code: '/', Text: "/"}), 5)

	// While actively typing in the filter, InputActive should be true
	if !s.InputActive() {
		t.Fatal("InputActive() should be true while filtering")
	}

	// Cancel filtering with esc
	sendMsg(s, tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}), 5)

	// After canceling: InputActive false
	if s.InputActive() {
		t.Fatal("InputActive() should be false after canceling filter")
	}
}

func TestBrowseScreen_ListCreatedOnLoad(t *testing.T) {
	s := newBrowseScreen(newMockServices(), testStyles(), true)
	if s.list != nil {
		t.Fatal("list should be nil before loading")
	}

	sendMsg(s, browseLoadedMsg{
		assets:   makeTestAssets("my-asset"),
		deployed: nil,
	}, 0)

	if s.list == nil {
		t.Fatal("list should be created after browseLoadedMsg")
	}
}

func TestBrowseScreen_AssetItemFilterValue(t *testing.T) {
	item := assetItem{
		name:    "my-skill",
		desc:    "skills · local",
		filterV: "my-skill skills local",
	}
	if got := item.FilterValue(); got != "my-skill skills local" {
		t.Errorf("FilterValue() = %q, want %q", got, "my-skill skills local")
	}
}

func TestBrowseScreen_AssetItemDeployedTitle(t *testing.T) {
	deployed := assetItem{name: "x", deployed: true}
	notDeployed := assetItem{name: "y", deployed: false}

	if !strings.Contains(deployed.Title(), "*") {
		t.Errorf("deployed item Title should contain '*', got: %q", deployed.Title())
	}
	if strings.Contains(notDeployed.Title(), "*") {
		t.Errorf("non-deployed item Title should not contain '*', got: %q", notDeployed.Title())
	}
}

// --- helpers ---

// makeTestAssets creates a slice of test assets with the given names, all of type AssetSkill.
func makeTestAssets(names ...string) []*asset.Asset {
	var assets []asset.Asset
	for _, name := range names {
		assets = append(assets, asset.Asset{
			Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: name},
		})
	}
	idx := asset.NewIndex(assets)
	return idx.All()
}

// Ensure list import is used (for compile-time interface checks).
var _ list.Item = assetItem{}
