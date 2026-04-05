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

func TestStatusScreen_Title(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)
	if got := s.Title(); got != "Status" {
		t.Fatalf("Title() = %q, want %q", got, "Status")
	}
}

func TestStatusScreen_InputActive(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)
	if s.InputActive() {
		t.Fatal("InputActive() = true, want false")
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
	for _, key := range []string{"d", "r"} {
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

	// Simulate receiving the loaded message.
	s.Update(statusLoadedMsg{entries: entries})

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

	// Simulate receiving empty results.
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

	s.Update(statusLoadedMsg{entries: entries})

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

	s.Update(statusLoadedMsg{entries: entries})

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
	loaded := loadStatus(svc, 0)
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

	s.Update(statusLoadedMsg{entries: entries})

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

	s.Update(statusLoadedMsg{entries: entries})

	v := s.View()
	// The type header should show the count "(3)".
	if !strings.Contains(v.Content, "(3)") {
		t.Errorf("View() should contain count '(3)' for 3 skills, got %q", v.Content)
	}
}

// --- Scroll / windowing tests ---

func TestStatusScreen_WindowSizeMsgSetsHeight(t *testing.T) {
	s := newStatusScreen(newMockServices(), NewStyles(true), true)
	s.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	if s.height != 24 {
		t.Fatalf("height = %d, want 24", s.height)
	}
}

func TestStatusScreen_JKScrollsView(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)

	// Load enough entries to fill more than one screen page.
	var entries []deploy.StatusEntry
	for i := range 20 {
		entries = append(entries, deploy.StatusEntry{
			Deployment: state.Deployment{
				AssetType: nd.AssetSkill,
				AssetName: fmt.Sprintf("skill-%02d", i),
				SourceID:  "src",
				Scope:     nd.ScopeGlobal,
			},
			Health: state.HealthOK,
		})
	}
	s.Update(statusLoadedMsg{entries: entries})
	s.Update(tea.WindowSizeMsg{Width: 80, Height: 12}) // contentHeight = 12-4 = 8

	initial := s.scroll.offset

	// Scroll down.
	s.Update(tea.KeyPressMsg(tea.Key{Code: 'j', Text: "j"}))
	if s.scroll.offset <= initial {
		t.Fatalf("scroll.offset after j = %d, want > %d", s.scroll.offset, initial)
	}

	// Scroll back up.
	s.Update(tea.KeyPressMsg(tea.Key{Code: 'k', Text: "k"}))
	if s.scroll.offset != initial {
		t.Fatalf("scroll.offset after k = %d, want %d", s.scroll.offset, initial)
	}
}

func TestStatusScreen_IndicatorShownWhenScrolled(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)

	var entries []deploy.StatusEntry
	for i := range 20 {
		entries = append(entries, deploy.StatusEntry{
			Deployment: state.Deployment{
				AssetType: nd.AssetSkill,
				AssetName: fmt.Sprintf("skill-%02d", i),
				SourceID:  "src",
				Scope:     nd.ScopeGlobal,
			},
			Health: state.HealthOK,
		})
	}
	s.Update(statusLoadedMsg{entries: entries})
	s.Update(tea.WindowSizeMsg{Width: 80, Height: 12})

	// Small terminal, list doesn't fit — ↓ indicator should appear.
	v := s.View()
	if !strings.Contains(v.Content, "↓") {
		t.Errorf("↓ indicator expected for oversized list, got: %q", v.Content)
	}
}

func TestStatusScreen_HelpItemsIncludeScrollKey(t *testing.T) {
	s := newStatusScreen(newMockServices(), NewStyles(true), true)
	var keys []string
	for _, item := range s.HelpItems() {
		keys = append(keys, item.Key)
	}
	found := false
	for _, k := range keys {
		if k == "j/k" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("HelpItems() should include j/k scroll key, got: %v", keys)
	}
}

func TestStatusScreen_SlashEntersFilterMode(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)
	s.loaded = true
	s.entries = testStatusEntries()
	s.renderedLines = splitLines(s.buildContent())

	s.Update(tea.KeyPressMsg(tea.Key{Code: '/'}))

	if !s.filtering {
		t.Fatal("expected filtering=true after pressing /")
	}
	if !s.InputActive() {
		t.Fatal("expected InputActive()=true while filtering")
	}
}

func TestStatusScreen_FilterMatchesAssetName(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)
	s.loaded = true
	s.entries = testStatusEntries()
	s.renderedLines = splitLines(s.buildContent())

	s.filter = "greeting"
	filtered := s.filteredEntries()
	if len(filtered) != 1 {
		t.Fatalf("expected 1 match for 'greeting', got %d", len(filtered))
	}
	if filtered[0].Deployment.AssetName != "greeting" {
		t.Fatalf("expected match 'greeting', got %q", filtered[0].Deployment.AssetName)
	}
}

func TestStatusScreen_FilterMatchesSourceID(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)
	s.loaded = true
	s.entries = testStatusEntries()

	s.filter = "dotfiles"
	filtered := s.filteredEntries()
	// All test entries have source "dotfiles"
	if len(filtered) != len(s.entries) {
		t.Fatalf("expected all entries to match 'dotfiles', got %d/%d", len(filtered), len(s.entries))
	}
}

func TestStatusScreen_FilterIsCaseInsensitive(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)
	s.loaded = true
	s.entries = testStatusEntries()

	s.filter = "GREETING"
	filtered := s.filteredEntries()
	if len(filtered) != 1 {
		t.Fatalf("expected 1 match for 'GREETING' (case-insensitive), got %d", len(filtered))
	}
}

func TestStatusScreen_EscClearsFilter(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)
	s.loaded = true
	s.entries = testStatusEntries()
	s.renderedLines = splitLines(s.buildContent())
	s.filtering = true
	s.filter = "test"

	s.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))

	if s.filtering {
		t.Fatal("expected filtering=false after esc")
	}
	if s.filter != "" {
		t.Fatalf("expected filter cleared, got %q", s.filter)
	}
}

func TestStatusScreen_ViewShowsFilterInputWhenFiltering(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)
	s.loaded = true
	s.entries = testStatusEntries()
	s.renderedLines = splitLines(s.buildContent())
	s.filtering = true
	s.filter = "test"

	v := s.View()
	if !strings.Contains(v.Content, "filter: test") {
		t.Error("expected view to show filter input")
	}
}

func TestStatusScreen_HelpShowsFilterHintWhenNotFiltering(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)
	items := s.HelpItems()
	found := false
	for _, item := range items {
		if item.Key == "/" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected / filter in help items when not filtering")
	}
}

func TestStatusScreen_EnterConfirmsFilter(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)
	s.loaded = true
	s.entries = testStatusEntries()
	s.renderedLines = splitLines(s.buildContent())
	s.filtering = true
	s.filter = "greeting"

	s.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))

	if s.filtering {
		t.Fatal("expected filtering=false after enter (confirm)")
	}
	if s.filter != "greeting" {
		t.Fatalf("enter should preserve filter text, got %q", s.filter)
	}
}

func TestStatusScreen_BackspaceOnEmptyFilterDoesNothing(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)
	s.loaded = true
	s.entries = testStatusEntries()
	s.renderedLines = splitLines(s.buildContent())
	s.filtering = true
	s.filter = ""

	s.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyBackspace}))

	if s.filter != "" {
		t.Fatalf("backspace on empty filter should leave it empty, got %q", s.filter)
	}
	// Should still be filtering (backspace doesn't exit filter mode).
	if !s.filtering {
		t.Fatal("backspace on empty should not exit filter mode")
	}
}

func TestStatusScreen_HelpShowsEscWhenFiltering(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)
	s.filtering = true
	items := s.HelpItems()
	found := false
	for _, item := range items {
		if item.Key == "esc" && item.Desc == "clear filter" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'esc clear filter' in help items when filtering")
	}
}

func TestStatusScreen_ScopeSwitchedMsg_ResetsAndReloads(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)

	// Load some data first.
	s.Update(statusLoadedMsg{entries: testStatusEntries()})
	if !s.loaded {
		t.Fatal("precondition: should be loaded")
	}

	_, cmd := s.Update(ScopeSwitchedMsg{})

	if s.loaded {
		t.Error("loaded should be false after ScopeSwitchedMsg")
	}
	if s.entries != nil {
		t.Error("entries should be nil after ScopeSwitchedMsg")
	}
	if s.renderedLines != nil {
		t.Error("renderedLines should be nil after ScopeSwitchedMsg")
	}
	if cmd == nil {
		t.Fatal("ScopeSwitchedMsg should return Init cmd to reload")
	}
}

// --- Generation counter (stale message guard) tests ---

func TestStatusScreen_ScopeSwitchedMsg_IncrementsGeneration(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)

	// Load initial data so the screen is in a non-trivial state.
	s.Update(statusLoadedMsg{entries: testStatusEntries()})
	before := s.generation

	s.Update(ScopeSwitchedMsg{})

	if s.generation != before+1 {
		t.Fatalf("generation = %d, want %d", s.generation, before+1)
	}
}

func TestStatusScreen_StaleLoadedMsgDiscarded(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)

	// Load initial data, then scope-switch (generation goes from 0 to 1).
	s.Update(statusLoadedMsg{entries: testStatusEntries()})
	s.Update(ScopeSwitchedMsg{})

	if s.generation != 1 {
		t.Fatalf("precondition: generation = %d, want 1", s.generation)
	}

	// Simulate a stale loaded message arriving from the old scope (generation 0).
	staleEntries := []deploy.StatusEntry{
		{
			Deployment: state.Deployment{AssetName: "stale", AssetType: nd.AssetSkill, SourceID: "old", Scope: nd.ScopeGlobal},
			Health:     state.HealthOK,
		},
	}
	s.Update(statusLoadedMsg{entries: staleEntries, generation: 0})

	// The stale message should be discarded.
	if s.loaded {
		t.Error("loaded should remain false after stale statusLoadedMsg")
	}
	if s.entries != nil {
		t.Error("entries should remain nil after stale statusLoadedMsg")
	}
}

func TestStatusScreen_FreshLoadedMsgAccepted(t *testing.T) {
	svc := newMockServices()
	s := newStatusScreen(svc, NewStyles(true), true)

	// Scope switch increments generation to 1.
	s.Update(ScopeSwitchedMsg{})

	// Fresh message with matching generation should be accepted.
	freshEntries := []deploy.StatusEntry{
		{
			Deployment: state.Deployment{AssetName: "fresh", AssetType: nd.AssetSkill, SourceID: "new", Scope: nd.ScopeGlobal},
			Health:     state.HealthOK,
		},
	}
	s.Update(statusLoadedMsg{entries: freshEntries, generation: 1})

	if !s.loaded {
		t.Error("loaded should be true after fresh statusLoadedMsg")
	}
	if len(s.entries) != 1 || s.entries[0].Deployment.AssetName != "fresh" {
		t.Errorf("entries should contain fresh data, got %v", s.entries)
	}
}

func testStatusEntries() []deploy.StatusEntry {
	return []deploy.StatusEntry{
		{
			Deployment: state.Deployment{
				AssetName: "greeting",
				AssetType: nd.AssetType("skills"),
				SourceID:  "dotfiles",
				Scope:     nd.ScopeGlobal,
			},
			Health: state.HealthOK,
		},
		{
			Deployment: state.Deployment{
				AssetName: "review",
				AssetType: nd.AssetType("skills"),
				SourceID:  "dotfiles",
				Scope:     nd.ScopeGlobal,
			},
			Health: state.HealthOK,
		},
		{
			Deployment: state.Deployment{
				AssetName: "debug-agent",
				AssetType: nd.AssetType("agents"),
				SourceID:  "dotfiles",
				Scope:     nd.ScopeGlobal,
			},
			Health: state.HealthBroken,
		},
	}
}
