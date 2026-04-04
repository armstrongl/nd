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

func TestDoctorScreen_ScopeSwitchedMsg_ResetsAndReloads(t *testing.T) {
	s := newDoctorScreen(newMockServices(), NewStyles(true), true)
	// Put in a non-loading state with stale data.
	s.step = doctorDone
	s.issues = []state.HealthCheck{
		{Deployment: state.Deployment{AssetName: "stale"}, Status: state.HealthBroken},
	}
	s.err = fmt.Errorf("old error")

	_, cmd := s.Update(ScopeSwitchedMsg{})

	if s.step != doctorLoading {
		t.Errorf("step should be doctorLoading after ScopeSwitchedMsg, got %d", s.step)
	}
	if s.issues != nil {
		t.Error("issues should be nil after ScopeSwitchedMsg")
	}
	if s.err != nil {
		t.Errorf("err should be nil after ScopeSwitchedMsg, got %v", s.err)
	}
	if cmd == nil {
		t.Fatal("ScopeSwitchedMsg should return Init cmd to reload")
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
