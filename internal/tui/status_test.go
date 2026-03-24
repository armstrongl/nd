package tui

import (
	"fmt"
	"strings"
	"testing"

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
