package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/armstrongl/nd/internal/deploy"
	"github.com/armstrongl/nd/internal/state"
)

// Compile-time check: doctorScreen satisfies Screen.
var _ Screen = (*doctorScreen)(nil)

func TestDoctorScreen_Title(t *testing.T) {
	s := newDoctorScreen(newMockServices(), NewStyles(true), true)
	if got := s.Title(); got != "Doctor" {
		t.Fatalf("Title() = %q, want %q", got, "Doctor")
	}
}

func TestDoctorScreen_InputActive_DuringConfirm(t *testing.T) {
	s := newDoctorScreen(newMockServices(), NewStyles(true), true)
	s.step = doctorConfirm
	if !s.InputActive() {
		t.Fatal("InputActive() = false during confirm step, want true")
	}
}

func TestDoctorScreen_InputActive_OtherSteps(t *testing.T) {
	s := newDoctorScreen(newMockServices(), NewStyles(true), true)
	for _, step := range []doctorStep{doctorLoading, doctorFixing, doctorDone} {
		s.step = step
		if s.InputActive() {
			t.Errorf("InputActive() = true at step %d, want false", step)
		}
	}
}

func TestDoctorScreen_InitReturnsCmd(t *testing.T) {
	s := newDoctorScreen(newMockServices(), NewStyles(true), true)
	cmd := s.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil cmd")
	}
}

func TestDoctorScreen_LoadingView(t *testing.T) {
	s := newDoctorScreen(newMockServices(), NewStyles(true), true)
	v := s.View()
	if !strings.Contains(v.Content, "Loading") && !strings.Contains(v.Content, "Scanning") {
		t.Errorf("loading view should contain loading indicator, got: %q", v.Content)
	}
}

func TestDoctorScreen_AllHealthyView(t *testing.T) {
	s := newDoctorScreen(newMockServices(), NewStyles(true), true)

	// Simulate receiving an empty issues list (all healthy).
	s.Update(doctorCheckedMsg{issues: nil, err: nil})

	v := s.View()
	if !strings.Contains(v.Content, "healthy") {
		t.Errorf("all-healthy view should mention 'healthy', got: %q", v.Content)
	}
}

func TestDoctorScreen_IssuesView(t *testing.T) {
	s := newDoctorScreen(newMockServices(), NewStyles(true), true)

	issues := []state.HealthCheck{
		{
			Deployment: state.Deployment{AssetName: "my-skill", AssetType: "skills"},
			Status:     state.HealthBroken,
			Detail:     "target does not exist",
		},
		{
			Deployment: state.Deployment{AssetName: "my-rule", AssetType: "rules"},
			Status:     state.HealthDrifted,
			Detail:     "symlink points to wrong target",
		},
	}

	s.Update(doctorCheckedMsg{issues: issues, err: nil})

	v := s.View()
	if !strings.Contains(v.Content, "my-skill") {
		t.Errorf("issues view should show asset names, got: %q", v.Content)
	}
	if !strings.Contains(v.Content, "my-rule") {
		t.Errorf("issues view should show all asset names, got: %q", v.Content)
	}
	if !strings.Contains(v.Content, "2") {
		t.Errorf("issues view should mention issue count, got: %q", v.Content)
	}
}

func TestDoctorScreen_ErrorView(t *testing.T) {
	s := newDoctorScreen(newMockServices(), NewStyles(true), true)
	testErr := fmt.Errorf("state file locked")
	s.Update(doctorCheckedMsg{issues: nil, err: testErr})

	v := s.View()
	if !strings.Contains(v.Content, "state file locked") {
		t.Errorf("error view should show error message, got: %q", v.Content)
	}
}

func TestDoctorScreen_SyncDone_Success(t *testing.T) {
	s := newDoctorScreen(newMockServices(), NewStyles(true), true)

	result := &deploy.SyncResult{
		Repaired: []state.Deployment{{AssetName: "my-skill"}},
		Removed:  []state.Deployment{{AssetName: "my-rule"}},
	}
	s.Update(doctorSyncedMsg{result: result, err: nil})

	v := s.View()
	if !strings.Contains(v.Content, "1") {
		t.Errorf("result view should show counts, got: %q", v.Content)
	}
}

func TestDoctorScreen_SyncDone_Error(t *testing.T) {
	s := newDoctorScreen(newMockServices(), NewStyles(true), true)
	testErr := fmt.Errorf("repair failed")
	s.Update(doctorSyncedMsg{result: nil, err: testErr})

	v := s.View()
	if !strings.Contains(v.Content, "repair failed") {
		t.Errorf("sync error view should show error, got: %q", v.Content)
	}
}

func TestDoctorScreen_EnterOnDone_EmitsPopToRootAndRefresh(t *testing.T) {
	s := newDoctorScreen(newMockServices(), NewStyles(true), true)
	s.step = doctorDone
	s.syncResult = &deploy.SyncResult{}

	_, cmd := s.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter on done step should emit a cmd")
	}

	// tea.Batch returns a BatchMsg ([]tea.Cmd).
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("enter on done should emit tea.BatchMsg, got %T", msg)
	}

	var hasPopToRoot, hasRefresh bool
	for _, c := range batch {
		if c == nil {
			continue
		}
		switch c().(type) {
		case PopToRootMsg:
			hasPopToRoot = true
		case RefreshHeaderMsg:
			hasRefresh = true
		}
	}
	if !hasPopToRoot {
		t.Error("batch should contain PopToRootMsg")
	}
	if !hasRefresh {
		t.Error("batch should contain RefreshHeaderMsg")
	}
}

func TestDoctorScreen_HealthGlyphs_InIssuesList(t *testing.T) {
	s := newDoctorScreen(newMockServices(), NewStyles(true), true)

	issues := []state.HealthCheck{
		{Status: state.HealthBroken, Deployment: state.Deployment{AssetName: "a"}},
		{Status: state.HealthDrifted, Deployment: state.Deployment{AssetName: "b"}},
		{Status: state.HealthMissing, Deployment: state.Deployment{AssetName: "c"}},
	}
	s.Update(doctorCheckedMsg{issues: issues, err: nil})

	v := s.View()
	if !strings.Contains(v.Content, GlyphBroken) {
		t.Errorf("broken glyph %q not found in view", GlyphBroken)
	}
	if !strings.Contains(v.Content, GlyphDrifted) {
		t.Errorf("drifted glyph %q not found in view", GlyphDrifted)
	}
	if !strings.Contains(v.Content, GlyphMissing) {
		t.Errorf("missing glyph %q not found in view", GlyphMissing)
	}
}

// --- Viewport wrapping tests (Unit 4) ---

// Verify that doctor done step with fix results renders via viewport when given dimensions.
func TestDoctorScreen_DoneViewport_FixResults(t *testing.T) {
	s := newDoctorScreen(newMockServices(), NewStyles(true), true)
	s.pendingWidth = 80
	s.pendingHeight = 10

	result := &deploy.SyncResult{
		Repaired: []state.Deployment{
			{AssetName: "skill-a"},
			{AssetName: "skill-b"},
		},
		Removed: []state.Deployment{
			{AssetName: "orphan-rule"},
		},
		Warnings: []string{"warning one", "warning two"},
	}

	s.Update(doctorSyncedMsg{result: result, err: nil})

	if s.vp == nil {
		t.Fatal("viewport should be initialized after transitioning to done step")
	}
	if s.vp.Width() != 80 {
		t.Fatalf("viewport width = %d, want 80", s.vp.Width())
	}
	if s.vp.Height() != 10 {
		t.Fatalf("viewport height = %d, want 10", s.vp.Height())
	}

	v := s.View()
	if v.Content == "" {
		t.Fatal("viewport-wrapped done view should not be empty")
	}

	content := s.viewDoneContent()
	if !strings.Contains(content, "2") {
		t.Errorf("done content should show repaired count; got:\n%s", content)
	}
}

// Verify that j/k scroll keys are forwarded to viewport at doctor done step.
func TestDoctorScreen_DoneViewport_ScrollForwarding(t *testing.T) {
	s := newDoctorScreen(newMockServices(), NewStyles(true), true)
	s.pendingWidth = 80
	s.pendingHeight = 5

	result := &deploy.SyncResult{
		Repaired: []state.Deployment{{AssetName: "a"}},
		Warnings: []string{"w1", "w2", "w3", "w4", "w5", "w6", "w7", "w8"},
	}
	s.Update(doctorSyncedMsg{result: result, err: nil})

	// Send 'j' key — should be forwarded to viewport for scrolling.
	_, cmd := s.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	if cmd != nil {
		msg := cmd()
		switch msg.(type) {
		case PopToRootMsg, BackMsg:
			t.Fatal("j key should not produce navigation messages at done step")
		}
	}
}

// Verify enter at doctor done step still emits PopToRootMsg with viewport active.
func TestDoctorScreen_DoneViewport_EnterStillReturns(t *testing.T) {
	s := newDoctorScreen(newMockServices(), NewStyles(true), true)
	s.pendingWidth = 80
	s.pendingHeight = 10

	s.Update(doctorSyncedMsg{result: &deploy.SyncResult{}, err: nil})

	_, cmd := s.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter at done step should emit a command even with viewport active")
	}

	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected tea.BatchMsg, got %T", msg)
	}

	var hasPopToRoot bool
	for _, c := range batch {
		if c == nil {
			continue
		}
		if _, ok := c().(PopToRootMsg); ok {
			hasPopToRoot = true
		}
	}
	if !hasPopToRoot {
		t.Error("enter should emit PopToRootMsg even with viewport active")
	}
}

// Verify ScreenSizeMsg updates viewport dimensions after creation.
func TestDoctorScreen_DoneViewport_ScreenSizeUpdates(t *testing.T) {
	s := newDoctorScreen(newMockServices(), NewStyles(true), true)
	s.pendingWidth = 80
	s.pendingHeight = 10

	s.Update(doctorSyncedMsg{result: &deploy.SyncResult{}, err: nil})

	if s.vp == nil {
		t.Fatal("viewport should exist after done transition")
	}

	s.Update(ScreenSizeMsg{Width: 120, Height: 25})

	if s.vp.Width() != 120 {
		t.Fatalf("viewport width after resize = %d, want 120", s.vp.Width())
	}
	if s.vp.Height() != 25 {
		t.Fatalf("viewport height after resize = %d, want 25", s.vp.Height())
	}
}

// Verify fallback rendering when viewport has zero dimensions.
func TestDoctorScreen_DoneViewport_FallbackWithoutDimensions(t *testing.T) {
	s := newDoctorScreen(newMockServices(), NewStyles(true), true)
	// No pending dimensions — viewport will have 0x0.
	s.Update(doctorCheckedMsg{issues: nil, err: nil})

	v := s.View()
	if v.Content == "" {
		t.Fatal("View() should fall back to raw string when viewport has zero dimensions")
	}
	if !strings.Contains(v.Content, "healthy") {
		t.Errorf("fallback view should contain 'healthy'; got:\n%s", v.Content)
	}
}

// Doctor confirm step is NOT viewport-wrapped. The huh.Confirm form should
// render directly without any viewport involvement.
func TestDoctorScreen_ConfirmNotViewportWrapped(t *testing.T) {
	s := newDoctorScreen(newMockServices(), NewStyles(true), true)
	s.pendingWidth = 80
	s.pendingHeight = 10

	issues := []state.HealthCheck{
		{
			Deployment: state.Deployment{AssetName: "my-skill", AssetType: "skills"},
			Status:     state.HealthBroken,
			Detail:     "target does not exist",
		},
	}
	s.Update(doctorCheckedMsg{issues: issues, err: nil})

	// At confirm step, viewport should not be created.
	if s.vp != nil {
		t.Fatal("viewport should NOT be created during confirm step")
	}
	if s.step != doctorConfirm {
		t.Fatalf("step = %d, want doctorConfirm (%d)", s.step, doctorConfirm)
	}

	// The confirm view should render issue details and form.
	v := s.View()
	if !strings.Contains(v.Content, "my-skill") {
		t.Errorf("confirm view should show asset names; got:\n%s", v.Content)
	}
	if !strings.Contains(v.Content, "1 issue") {
		t.Errorf("confirm view should show issue count; got:\n%s", v.Content)
	}
}

// Verify that ScreenSizeMsg before viewport creation stores pending dimensions.
func TestDoctorScreen_ScreenSizeBeforeViewport_StoresPending(t *testing.T) {
	s := newDoctorScreen(newMockServices(), NewStyles(true), true)

	// No viewport yet (still at loading step).
	s.Update(ScreenSizeMsg{Width: 100, Height: 30})

	if s.pendingWidth != 100 {
		t.Fatalf("pendingWidth = %d, want 100", s.pendingWidth)
	}
	if s.pendingHeight != 30 {
		t.Fatalf("pendingHeight = %d, want 30", s.pendingHeight)
	}
	if s.vp != nil {
		t.Fatal("viewport should not be created by ScreenSizeMsg alone")
	}
}

func TestDoctorScreen_RefreshHeaderEmittedAfterSync(t *testing.T) {
	s := newDoctorScreen(newMockServices(), NewStyles(true), true)

	_, cmd := s.Update(doctorSyncedMsg{result: &deploy.SyncResult{}, err: nil})
	if cmd == nil {
		t.Fatal("doctorSyncedMsg should emit a cmd")
	}

	msg := cmd()
	switch msg.(type) {
	case RefreshHeaderMsg:
		// OK
	default:
		t.Errorf("doctorSyncedMsg should emit RefreshHeaderMsg, got %T", msg)
	}
}
