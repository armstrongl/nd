package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/armstrongl/nd/internal/deploy"
	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/state"
)

// Compile-time assertions: statusScreen satisfies Screen and HelpProvider.
var _ Screen = (*statusScreen)(nil)
var _ HelpProvider = (*statusScreen)(nil)

// testScreenSize is a standard ScreenSizeMsg used in tests to give the viewport
// reasonable dimensions. Must be sent BEFORE statusLoadedMsg (two-phase init).
var testScreenSize = ScreenSizeMsg{Width: 80, Height: 40}

// sendStatusLoaded sends a ScreenSizeMsg followed by a statusLoadedMsg through
// Update, simulating the two-phase init that happens in the real TUI.
func sendStatusLoaded(s *statusScreen, entries []deploy.StatusEntry, err error) {
	s.Update(testScreenSize)
	s.Update(statusLoadedMsg{entries: entries, err: err})
}

// keyMsg creates a tea.KeyPressMsg for a printable character.
func keyMsg(ch rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: ch, Text: string(ch)}
}

// specialKeyMsg creates a tea.KeyPressMsg for a special key (enter, esc, backspace).
func specialKeyMsg(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code}
}

func TestStatusScreen_Title(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)
	if got := s.Title(); got != "Status" {
		t.Fatalf("Title() = %q, want %q", got, "Status")
	}
}

func TestStatusScreen_InputActiveDefaultFalse(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)
	if s.InputActive() {
		t.Fatal("InputActive() = true, want false when filter not active")
	}
}

func TestStatusScreen_InputActiveTrueWhenFiltering(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)
	entries := []deploy.StatusEntry{
		{
			Deployment: state.Deployment{AssetType: nd.AssetSkill, AssetName: "a", SourceID: "s1", Scope: nd.ScopeGlobal},
			Health:     state.HealthOK,
		},
	}
	sendStatusLoaded(s, entries, nil)

	// Activate filter.
	s.Update(keyMsg('/'))
	if !s.InputActive() {
		t.Fatal("InputActive() = false, want true when filtering")
	}
}

func TestStatusScreen_HelpItems(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)
	items := s.HelpItems()
	if len(items) == 0 {
		t.Fatal("HelpItems() returned empty slice")
	}

	// Verify expected help items are present.
	found := make(map[string]bool)
	for _, item := range items {
		found[item.Key] = true
	}
	for _, key := range []string{"d", "r", "f", "/"} {
		if !found[key] {
			t.Errorf("HelpItems() missing key %q", key)
		}
	}
}

func TestStatusScreen_ViewBeforeLoading(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)

	v := s.View()
	if !strings.Contains(v.Content, "Loading") {
		t.Fatalf("View() before loading should contain 'Loading', got %q", v.Content)
	}
}

func TestStatusScreen_ViewWithEntries(t *testing.T) {
	svc := newMockServices()
	styles := NewStyles(true)
	s := newStatusScreen(svc, styles, true)

	entries := []deploy.StatusEntry{
		{
			Deployment: state.Deployment{
				AssetType: nd.AssetSkill,
				AssetName: "greeting",
				SourceID:  "my-source",
				Scope:     nd.ScopeGlobal,
			},
			Health: state.HealthOK,
		},
		{
			Deployment: state.Deployment{
				AssetType: nd.AssetSkill,
				AssetName: "code-review",
				SourceID:  "my-source",
				Scope:     nd.ScopeGlobal,
			},
			Health: state.HealthBroken,
		},
		{
			Deployment: state.Deployment{
				AssetType: nd.AssetAgent,
				AssetName: "reviewer",
				SourceID:  "other-source",
				Scope:     nd.ScopeProject,
			},
			Health: state.HealthDrifted,
		},
	}

	sendStatusLoaded(s, entries, nil)

	v := s.View()
	content := v.Content

	// Should contain asset names.
	if !strings.Contains(content, "greeting") {
		t.Errorf("View() should contain 'greeting', got %q", content)
	}
	if !strings.Contains(content, "code-review") {
		t.Errorf("View() should contain 'code-review', got %q", content)
	}
	if !strings.Contains(content, "reviewer") {
		t.Errorf("View() should contain 'reviewer', got %q", content)
	}

	// Should contain type headers.
	if !strings.Contains(content, string(nd.AssetSkill)) {
		t.Errorf("View() should contain type header %q", nd.AssetSkill)
	}
	if !strings.Contains(content, string(nd.AssetAgent)) {
		t.Errorf("View() should contain type header %q", nd.AssetAgent)
	}

	// Should contain deployment count.
	if !strings.Contains(content, "3 deployed") {
		t.Errorf("View() should contain '3 deployed', got %q", content)
	}

	// Should contain issue count (2 issues: 1 broken + 1 drifted).
	if !strings.Contains(content, "2 issues") {
		t.Errorf("View() should contain '2 issues', got %q", content)
	}
}

func TestStatusScreen_ViewWithEmptyEntries(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)

	// Simulate receiving empty results (no ScreenSizeMsg needed for empty state).
	s.Update(statusLoadedMsg{entries: nil})

	v := s.View()
	nothingMsg := NothingDeployed()
	if !strings.Contains(v.Content, nothingMsg) {
		t.Fatalf("View() with no entries should contain NothingDeployed() message, got %q", v.Content)
	}
}

func TestStatusScreen_ViewWithError(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)

	testErr := fmt.Errorf("state file corrupted")
	s.Update(statusLoadedMsg{err: testErr})

	v := s.View()
	if !strings.Contains(v.Content, "Error") {
		t.Fatalf("View() with error should contain 'Error', got %q", v.Content)
	}
	if !strings.Contains(v.Content, "state file corrupted") {
		t.Fatalf("View() with error should contain error text, got %q", v.Content)
	}
}

func TestStatusScreen_IssueCountingOnlyCountsNonOK(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)

	entries := []deploy.StatusEntry{
		{
			Deployment: state.Deployment{AssetType: nd.AssetSkill, AssetName: "a", SourceID: "s1", Scope: nd.ScopeGlobal},
			Health:     state.HealthOK,
		},
		{
			Deployment: state.Deployment{AssetType: nd.AssetSkill, AssetName: "b", SourceID: "s1", Scope: nd.ScopeGlobal},
			Health:     state.HealthOK,
		},
		{
			Deployment: state.Deployment{AssetType: nd.AssetSkill, AssetName: "c", SourceID: "s1", Scope: nd.ScopeGlobal},
			Health:     state.HealthMissing,
		},
		{
			Deployment: state.Deployment{AssetType: nd.AssetAgent, AssetName: "d", SourceID: "s1", Scope: nd.ScopeGlobal},
			Health:     state.HealthOrphaned,
		},
	}

	sendStatusLoaded(s, entries, nil)

	if s.issues != 2 {
		t.Fatalf("issues = %d, want 2 (1 missing + 1 orphaned)", s.issues)
	}

	v := s.View()
	if !strings.Contains(v.Content, "2 issues") {
		t.Errorf("View() should contain '2 issues', got %q", v.Content)
	}

	// 4 total deployed.
	if !strings.Contains(v.Content, "4 deployed") {
		t.Errorf("View() should contain '4 deployed', got %q", v.Content)
	}
}

func TestStatusScreen_AllHealthyNoIssuesSuffix(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)

	entries := []deploy.StatusEntry{
		{
			Deployment: state.Deployment{AssetType: nd.AssetSkill, AssetName: "a", SourceID: "s1", Scope: nd.ScopeGlobal},
			Health:     state.HealthOK,
		},
	}

	sendStatusLoaded(s, entries, nil)

	if s.issues != 0 {
		t.Fatalf("issues = %d, want 0", s.issues)
	}

	v := s.View()
	if strings.Contains(v.Content, "issues") {
		t.Errorf("View() should NOT contain 'issues' when all healthy, got %q", v.Content)
	}
}

func TestStatusScreen_HealthGlyphs(t *testing.T) {
	cases := []struct {
		health state.HealthStatus
		glyph  string
	}{
		{state.HealthOK, GlyphOK},
		{state.HealthBroken, GlyphBroken},
		{state.HealthDrifted, GlyphDrifted},
		{state.HealthOrphaned, GlyphOrphan},
		{state.HealthMissing, GlyphMissing},
	}

	for _, tc := range cases {
		t.Run(tc.health.String(), func(t *testing.T) {
			got := healthGlyph(tc.health)
			if got != tc.glyph {
				t.Errorf("healthGlyph(%v) = %q, want %q", tc.health, got, tc.glyph)
			}
		})
	}
}

func TestStatusScreen_HealthGlyphDefault(t *testing.T) {
	// An unknown HealthStatus value should return "??" as fallback.
	got := healthGlyph(state.HealthStatus(99))
	if got != "??" {
		t.Errorf("healthGlyph(99) = %q, want %q", got, "??")
	}
}

func TestStatusScreen_InitReturnsCmd(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)

	cmd := s.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil, expected a command to load status")
	}
}

func TestStatusScreen_LoadStatusReturnsMsg(t *testing.T) {
	svc := newMockServices()

	// Default mockServices.DeployEngine() returns (nil, nil), which should
	// produce an error in loadStatus since engine is nil.
	loaded := loadStatus(svc)
	if loaded.err == nil {
		t.Fatal("loadStatus() with nil engine should return an error")
	}
}

func TestStatusScreen_GroupingByType(t *testing.T) {
	svc := newMockServices()
	styles := NewStyles(true)
	s := newStatusScreen(svc, styles, true)

	entries := []deploy.StatusEntry{
		{
			Deployment: state.Deployment{AssetType: nd.AssetAgent, AssetName: "reviewer", SourceID: "s1", Scope: nd.ScopeGlobal},
			Health:     state.HealthOK,
		},
		{
			Deployment: state.Deployment{AssetType: nd.AssetSkill, AssetName: "greeting", SourceID: "s1", Scope: nd.ScopeGlobal},
			Health:     state.HealthOK,
		},
		{
			Deployment: state.Deployment{AssetType: nd.AssetAgent, AssetName: "coder", SourceID: "s1", Scope: nd.ScopeGlobal},
			Health:     state.HealthOK,
		},
	}

	sendStatusLoaded(s, entries, nil)

	v := s.View()
	content := v.Content

	// Both type headers should appear.
	agentsIdx := strings.Index(content, string(nd.AssetAgent))
	skillsIdx := strings.Index(content, string(nd.AssetSkill))

	if agentsIdx == -1 {
		t.Fatal("View() should contain agents type header")
	}
	if skillsIdx == -1 {
		t.Fatal("View() should contain skills type header")
	}

	// AssetAgent = "agents" sorts before AssetSkill = "skills".
	if agentsIdx > skillsIdx {
		t.Errorf("agents type header should appear before skills (agents at %d, skills at %d)", agentsIdx, skillsIdx)
	}
}

func TestStatusScreen_TypeCountInHeader(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)

	entries := []deploy.StatusEntry{
		{
			Deployment: state.Deployment{AssetType: nd.AssetSkill, AssetName: "a", SourceID: "s1", Scope: nd.ScopeGlobal},
			Health:     state.HealthOK,
		},
		{
			Deployment: state.Deployment{AssetType: nd.AssetSkill, AssetName: "b", SourceID: "s1", Scope: nd.ScopeGlobal},
			Health:     state.HealthOK,
		},
		{
			Deployment: state.Deployment{AssetType: nd.AssetSkill, AssetName: "c", SourceID: "s1", Scope: nd.ScopeGlobal},
			Health:     state.HealthOK,
		},
	}

	sendStatusLoaded(s, entries, nil)

	v := s.View()
	// The type header should show the count "(3)".
	if !strings.Contains(v.Content, "(3)") {
		t.Errorf("View() should contain count '(3)' for 3 skills, got %q", v.Content)
	}
}

// --- New tests for cursor, filter, viewport, and shortcuts ---

func TestStatusScreen_CursorStartsAtFirstItem(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)

	entries := []deploy.StatusEntry{
		{
			Deployment: state.Deployment{AssetType: nd.AssetSkill, AssetName: "alpha", SourceID: "s1", Scope: nd.ScopeGlobal},
			Health:     state.HealthOK,
		},
		{
			Deployment: state.Deployment{AssetType: nd.AssetSkill, AssetName: "beta", SourceID: "s1", Scope: nd.ScopeGlobal},
			Health:     state.HealthOK,
		},
	}

	sendStatusLoaded(s, entries, nil)

	// Cursor should start at index 0 (first selectable item, not the header).
	if s.cursor != 0 {
		t.Fatalf("cursor = %d, want 0", s.cursor)
	}
	if len(s.selectableLines) != 2 {
		t.Fatalf("selectableLines = %d items, want 2", len(s.selectableLines))
	}

	// The view should contain the cursor indicator on the first item.
	v := s.View()
	if !strings.Contains(v.Content, ">") {
		t.Errorf("View() should contain cursor indicator '>', got %q", v.Content)
	}
}

func TestStatusScreen_CursorSkipsHeaders(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)

	entries := []deploy.StatusEntry{
		{
			Deployment: state.Deployment{AssetType: nd.AssetAgent, AssetName: "reviewer", SourceID: "s1", Scope: nd.ScopeGlobal},
			Health:     state.HealthOK,
		},
		{
			Deployment: state.Deployment{AssetType: nd.AssetSkill, AssetName: "greeting", SourceID: "s1", Scope: nd.ScopeGlobal},
			Health:     state.HealthOK,
		},
	}

	sendStatusLoaded(s, entries, nil)

	// Cursor at 0 should be on first item ("reviewer"), not a header.
	if s.cursor != 0 {
		t.Fatalf("cursor = %d, want 0", s.cursor)
	}

	// Move cursor down - should go to next selectable item, skipping the
	// "skills" header and blank line between groups.
	s.Update(keyMsg('j'))
	if s.cursor != 1 {
		t.Fatalf("after j, cursor = %d, want 1", s.cursor)
	}

	// The selectable lines should not include header or blank lines.
	if len(s.selectableLines) != 2 {
		t.Fatalf("selectableLines = %d items, want 2 (one per asset)", len(s.selectableLines))
	}
}

func TestStatusScreen_CursorStopsAtBoundaries(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)

	entries := []deploy.StatusEntry{
		{
			Deployment: state.Deployment{AssetType: nd.AssetSkill, AssetName: "a", SourceID: "s1", Scope: nd.ScopeGlobal},
			Health:     state.HealthOK,
		},
		{
			Deployment: state.Deployment{AssetType: nd.AssetSkill, AssetName: "b", SourceID: "s1", Scope: nd.ScopeGlobal},
			Health:     state.HealthOK,
		},
	}

	sendStatusLoaded(s, entries, nil)

	// Move up at top should stay at 0.
	s.Update(keyMsg('k'))
	if s.cursor != 0 {
		t.Fatalf("cursor at top after k = %d, want 0", s.cursor)
	}

	// Move to bottom.
	s.Update(keyMsg('j'))
	if s.cursor != 1 {
		t.Fatalf("cursor after j = %d, want 1", s.cursor)
	}

	// Move past bottom should stay at 1.
	s.Update(keyMsg('j'))
	if s.cursor != 1 {
		t.Fatalf("cursor at bottom after j = %d, want 1", s.cursor)
	}
}

func TestStatusScreen_JKMovement(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)

	entries := []deploy.StatusEntry{
		{
			Deployment: state.Deployment{AssetType: nd.AssetSkill, AssetName: "a", SourceID: "s1", Scope: nd.ScopeGlobal},
			Health:     state.HealthOK,
		},
		{
			Deployment: state.Deployment{AssetType: nd.AssetSkill, AssetName: "b", SourceID: "s1", Scope: nd.ScopeGlobal},
			Health:     state.HealthOK,
		},
		{
			Deployment: state.Deployment{AssetType: nd.AssetSkill, AssetName: "c", SourceID: "s1", Scope: nd.ScopeGlobal},
			Health:     state.HealthOK,
		},
	}

	sendStatusLoaded(s, entries, nil)

	// j moves forward.
	s.Update(keyMsg('j'))
	if s.cursor != 1 {
		t.Fatalf("cursor after j = %d, want 1", s.cursor)
	}

	s.Update(keyMsg('j'))
	if s.cursor != 2 {
		t.Fatalf("cursor after 2x j = %d, want 2", s.cursor)
	}

	// k moves backward.
	s.Update(keyMsg('k'))
	if s.cursor != 1 {
		t.Fatalf("cursor after k = %d, want 1", s.cursor)
	}
}

func TestStatusScreen_ShortcutD_NavigatesToDeploy(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)

	entries := []deploy.StatusEntry{
		{
			Deployment: state.Deployment{AssetType: nd.AssetSkill, AssetName: "a", SourceID: "s1", Scope: nd.ScopeGlobal},
			Health:     state.HealthOK,
		},
	}
	sendStatusLoaded(s, entries, nil)

	_, cmd := s.Update(keyMsg('d'))
	if cmd == nil {
		t.Fatal("pressing d should return a cmd")
	}
	msg := cmd()
	nav, ok := msg.(NavigateMsg)
	if !ok {
		t.Fatalf("cmd should produce NavigateMsg, got %T", msg)
	}
	if _, ok := nav.Screen.(*deployScreen); !ok {
		t.Errorf("NavigateMsg.Screen should be *deployScreen, got %T", nav.Screen)
	}
}

func TestStatusScreen_ShortcutR_NavigatesToRemove(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)

	entries := []deploy.StatusEntry{
		{
			Deployment: state.Deployment{AssetType: nd.AssetSkill, AssetName: "a", SourceID: "s1", Scope: nd.ScopeGlobal},
			Health:     state.HealthOK,
		},
	}
	sendStatusLoaded(s, entries, nil)

	_, cmd := s.Update(keyMsg('r'))
	if cmd == nil {
		t.Fatal("pressing r should return a cmd")
	}
	msg := cmd()
	nav, ok := msg.(NavigateMsg)
	if !ok {
		t.Fatalf("cmd should produce NavigateMsg, got %T", msg)
	}
	if _, ok := nav.Screen.(*removeScreen); !ok {
		t.Errorf("NavigateMsg.Screen should be *removeScreen, got %T", nav.Screen)
	}
}

func TestStatusScreen_ShortcutF_NavigatesToDoctor(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)

	entries := []deploy.StatusEntry{
		{
			Deployment: state.Deployment{AssetType: nd.AssetSkill, AssetName: "a", SourceID: "s1", Scope: nd.ScopeGlobal},
			Health:     state.HealthOK,
		},
	}
	sendStatusLoaded(s, entries, nil)

	_, cmd := s.Update(keyMsg('f'))
	if cmd == nil {
		t.Fatal("pressing f should return a cmd")
	}
	msg := cmd()
	nav, ok := msg.(NavigateMsg)
	if !ok {
		t.Fatalf("cmd should produce NavigateMsg, got %T", msg)
	}
	if _, ok := nav.Screen.(*doctorScreen); !ok {
		t.Errorf("NavigateMsg.Screen should be *doctorScreen, got %T", nav.Screen)
	}
}

func TestStatusScreen_ViewportScrollsOnLongList(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)

	// Create enough entries to exceed a short viewport height.
	var entries []deploy.StatusEntry
	for i := 0; i < 30; i++ {
		entries = append(entries, deploy.StatusEntry{
			Deployment: state.Deployment{
				AssetType: nd.AssetSkill,
				AssetName: fmt.Sprintf("skill-%02d", i),
				SourceID:  "s1",
				Scope:     nd.ScopeGlobal,
			},
			Health: state.HealthOK,
		})
	}

	// Use a small viewport height to force scrolling.
	s.Update(ScreenSizeMsg{Width: 80, Height: 10})
	s.Update(statusLoadedMsg{entries: entries})

	// Viewport should be ready.
	if !s.vpReady {
		t.Fatal("viewport should be ready after loading entries")
	}

	// The viewport should have content, even if not all items are visible.
	v := s.View()
	if v.Content == "" {
		t.Fatal("View() should not be empty with 30 entries")
	}

	// Move cursor to the last item.
	for i := 0; i < 29; i++ {
		s.Update(keyMsg('j'))
	}
	if s.cursor != 29 {
		t.Fatalf("cursor after 29 j presses = %d, want 29", s.cursor)
	}

	// The last item should be visible in the content.
	v = s.View()
	if !strings.Contains(v.Content, "skill-29") {
		t.Errorf("View() should contain 'skill-29' after scrolling to end, got %q", v.Content)
	}
}

func TestStatusScreen_FilterNarrowsItems(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)

	entries := []deploy.StatusEntry{
		{
			Deployment: state.Deployment{AssetType: nd.AssetSkill, AssetName: "greeting", SourceID: "s1", Scope: nd.ScopeGlobal},
			Health:     state.HealthOK,
		},
		{
			Deployment: state.Deployment{AssetType: nd.AssetSkill, AssetName: "code-review", SourceID: "s1", Scope: nd.ScopeGlobal},
			Health:     state.HealthOK,
		},
		{
			Deployment: state.Deployment{AssetType: nd.AssetAgent, AssetName: "reviewer", SourceID: "s1", Scope: nd.ScopeGlobal},
			Health:     state.HealthOK,
		},
	}

	sendStatusLoaded(s, entries, nil)

	// Activate filter.
	s.Update(keyMsg('/'))
	if !s.filtering {
		t.Fatal("should be in filtering mode after /")
	}

	// Type "review" to filter.
	for _, ch := range "review" {
		s.Update(keyMsg(ch))
	}

	if s.filter != "review" {
		t.Fatalf("filter = %q, want %q", s.filter, "review")
	}

	// Cursor should reset to 0 (first matching item).
	if s.cursor != 0 {
		t.Fatalf("cursor after filter = %d, want 0", s.cursor)
	}

	// View should contain matching items and not non-matching ones.
	v := s.View()
	if !strings.Contains(v.Content, "code-review") {
		t.Errorf("View() should contain 'code-review' (matches filter)")
	}
	if !strings.Contains(v.Content, "reviewer") {
		t.Errorf("View() should contain 'reviewer' (matches filter)")
	}

	// "greeting" does not match "review".
	if strings.Contains(v.Content, "greeting") {
		t.Errorf("View() should NOT contain 'greeting' (does not match filter)")
	}
}

func TestStatusScreen_FilterClearsOnEsc(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)

	entries := []deploy.StatusEntry{
		{
			Deployment: state.Deployment{AssetType: nd.AssetSkill, AssetName: "alpha", SourceID: "s1", Scope: nd.ScopeGlobal},
			Health:     state.HealthOK,
		},
		{
			Deployment: state.Deployment{AssetType: nd.AssetSkill, AssetName: "beta", SourceID: "s1", Scope: nd.ScopeGlobal},
			Health:     state.HealthOK,
		},
	}

	sendStatusLoaded(s, entries, nil)

	// Activate and type filter.
	s.Update(keyMsg('/'))
	s.Update(keyMsg('a'))
	if s.filter != "a" {
		t.Fatalf("filter = %q, want %q", s.filter, "a")
	}

	// Esc should clear filter and exit filtering mode.
	s.Update(specialKeyMsg(tea.KeyEscape))
	if s.filtering {
		t.Fatal("should not be filtering after esc")
	}
	if s.filter != "" {
		t.Fatalf("filter = %q, want empty after esc", s.filter)
	}

	// All items should be visible again.
	v := s.View()
	if !strings.Contains(v.Content, "alpha") {
		t.Errorf("View() should contain 'alpha' after clearing filter")
	}
	if !strings.Contains(v.Content, "beta") {
		t.Errorf("View() should contain 'beta' after clearing filter")
	}
}

func TestStatusScreen_FilterNoMatchShowsMessage(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)

	entries := []deploy.StatusEntry{
		{
			Deployment: state.Deployment{AssetType: nd.AssetSkill, AssetName: "alpha", SourceID: "s1", Scope: nd.ScopeGlobal},
			Health:     state.HealthOK,
		},
	}

	sendStatusLoaded(s, entries, nil)

	// Activate filter and type something that doesn't match.
	s.Update(keyMsg('/'))
	for _, ch := range "zzz" {
		s.Update(keyMsg(ch))
	}

	v := s.View()
	if !strings.Contains(v.Content, "No assets match") {
		t.Errorf("View() should show 'No assets match' message when filter has no results, got %q", v.Content)
	}
}

func TestStatusScreen_ScreenSizeMsgBeforeLoad_StoresPending(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)

	// Send ScreenSizeMsg before data loads.
	s.Update(ScreenSizeMsg{Width: 120, Height: 50})

	if s.pendingWidth != 120 {
		t.Errorf("pendingWidth = %d, want 120", s.pendingWidth)
	}
	if s.pendingHeight != 50 {
		t.Errorf("pendingHeight = %d, want 50", s.pendingHeight)
	}

	// Viewport should not exist yet.
	if s.vpReady {
		t.Fatal("viewport should not be ready before data loads")
	}
}

func TestStatusScreen_ScreenSizeMsgAfterLoad_SizesViewport(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)

	entries := []deploy.StatusEntry{
		{
			Deployment: state.Deployment{AssetType: nd.AssetSkill, AssetName: "a", SourceID: "s1", Scope: nd.ScopeGlobal},
			Health:     state.HealthOK,
		},
	}

	sendStatusLoaded(s, entries, nil)

	if !s.vpReady {
		t.Fatal("viewport should be ready after loading entries")
	}

	// Send a new ScreenSizeMsg to resize.
	s.Update(ScreenSizeMsg{Width: 100, Height: 30})

	// Viewport should reflect new dimensions.
	if s.vp.Width() != 100 {
		t.Errorf("viewport width = %d, want 100", s.vp.Width())
	}
	if s.vp.Height() != 30 {
		t.Errorf("viewport height = %d, want 30", s.vp.Height())
	}
}

func TestStatusScreen_FilterBarVisibleWhenFiltering(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)

	entries := []deploy.StatusEntry{
		{
			Deployment: state.Deployment{AssetType: nd.AssetSkill, AssetName: "alpha", SourceID: "s1", Scope: nd.ScopeGlobal},
			Health:     state.HealthOK,
		},
	}

	sendStatusLoaded(s, entries, nil)

	// Activate filter.
	s.Update(keyMsg('/'))

	v := s.View()
	// Filter bar should contain the / indicator.
	if !strings.Contains(v.Content, "/") {
		t.Errorf("View() should show filter bar with '/' when filtering, got %q", v.Content)
	}
}

func TestStatusScreen_FilterBackspaceRemovesChar(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)

	entries := []deploy.StatusEntry{
		{
			Deployment: state.Deployment{AssetType: nd.AssetSkill, AssetName: "alpha", SourceID: "s1", Scope: nd.ScopeGlobal},
			Health:     state.HealthOK,
		},
	}

	sendStatusLoaded(s, entries, nil)

	// Activate filter and type.
	s.Update(keyMsg('/'))
	s.Update(keyMsg('a'))
	s.Update(keyMsg('b'))
	if s.filter != "ab" {
		t.Fatalf("filter = %q, want %q", s.filter, "ab")
	}

	// Backspace removes last char.
	s.Update(specialKeyMsg(tea.KeyBackspace))
	if s.filter != "a" {
		t.Fatalf("filter after backspace = %q, want %q", s.filter, "a")
	}
}

func TestStatusScreen_FilterEnterKeepsFilterApplied(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)

	entries := []deploy.StatusEntry{
		{
			Deployment: state.Deployment{AssetType: nd.AssetSkill, AssetName: "alpha", SourceID: "s1", Scope: nd.ScopeGlobal},
			Health:     state.HealthOK,
		},
		{
			Deployment: state.Deployment{AssetType: nd.AssetSkill, AssetName: "beta", SourceID: "s1", Scope: nd.ScopeGlobal},
			Health:     state.HealthOK,
		},
	}

	sendStatusLoaded(s, entries, nil)

	// Activate filter, type "alpha", then press enter.
	s.Update(keyMsg('/'))
	for _, ch := range "alpha" {
		s.Update(keyMsg(ch))
	}
	s.Update(specialKeyMsg(tea.KeyEnter))

	// Filter should still be applied but not in filtering mode.
	if s.filtering {
		t.Fatal("should not be in filtering mode after enter")
	}
	if s.filter != "alpha" {
		t.Fatalf("filter = %q, want %q", s.filter, "alpha")
	}

	// View should only show matching items.
	v := s.View()
	if !strings.Contains(v.Content, "alpha") {
		t.Errorf("View() should contain 'alpha'")
	}
	if strings.Contains(v.Content, "beta") {
		t.Errorf("View() should NOT contain 'beta' when filter is 'alpha'")
	}
}

func TestStatusScreen_EmptyStateUnchanged(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)

	// Send ScreenSizeMsg and then empty entries.
	s.Update(testScreenSize)
	s.Update(statusLoadedMsg{entries: nil})

	// Should show nothing deployed message, no viewport.
	v := s.View()
	nothingMsg := NothingDeployed()
	if !strings.Contains(v.Content, nothingMsg) {
		t.Fatalf("View() with no entries should contain NothingDeployed() message, got %q", v.Content)
	}
	if s.vpReady {
		t.Fatal("viewport should not be created for empty entries")
	}
}
