package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/armstrongl/nd/internal/asset"
	"github.com/armstrongl/nd/internal/deploy"
	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/state"
)

// Compile-time assertion: removeScreen satisfies Screen.
var _ Screen = (*removeScreen)(nil)

func TestRemove_Title(t *testing.T) {
	svc := newMockServices()
	s := NewStyles(true)
	m := newRemoveScreen(svc, s, true)
	if got := m.Title(); got != "Remove" {
		t.Fatalf("Title() = %q, want %q", got, "Remove")
	}
}

func TestRemove_InputActive(t *testing.T) {
	svc := newMockServices()
	s := NewStyles(true)
	m := newRemoveScreen(svc, s, true)
	if m.InputActive() {
		t.Fatal("InputActive() = true, want false")
	}
}

func TestRemove_InitialState(t *testing.T) {
	svc := newMockServices()
	s := NewStyles(true)
	m := newRemoveScreen(svc, s, true)

	if m.step != removeSelectAssets {
		t.Fatalf("initial step = %d, want removeSelectAssets (%d)", m.step, removeSelectAssets)
	}
	if m.svc == nil {
		t.Fatal("svc is nil")
	}
	if m.err != nil {
		t.Fatalf("initial err = %v, want nil", m.err)
	}
	if m.succeeded != 0 {
		t.Fatalf("initial succeeded = %d, want 0", m.succeeded)
	}
	if len(m.failed) != 0 {
		t.Fatalf("initial failed = %d, want 0", len(m.failed))
	}
}

func TestRemove_InitReturnsCmd(t *testing.T) {
	svc := newMockServices()
	s := NewStyles(true)
	m := newRemoveScreen(svc, s, true)
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil, want a cmd to load deployments")
	}
}

func TestRemove_DeploymentsLoadedMsg_Empty(t *testing.T) {
	svc := newMockServices()
	s := NewStyles(true)
	m := newRemoveScreen(svc, s, true)

	msg := deploymentsLoadedMsg{
		deployments: nil,
		err:         nil,
	}

	updated, _ := m.Update(msg)
	rm := updated.(*removeScreen)

	if len(rm.deployments) != 0 {
		t.Fatalf("deployments count = %d, want 0", len(rm.deployments))
	}
}

func TestRemove_DeploymentsLoadedMsg_WithDeployments(t *testing.T) {
	svc := newMockServices()
	s := NewStyles(true)
	m := newRemoveScreen(svc, s, true)

	deps := []state.Deployment{
		{
			SourceID:   "local-src",
			AssetType:  nd.AssetSkill,
			AssetName:  "greeting",
			SourcePath: "/src/skills/greeting",
			LinkPath:   "/home/.config/claude/skills/greeting",
			Scope:      nd.ScopeGlobal,
			DeployedAt: time.Now(),
		},
		{
			SourceID:   "local-src",
			AssetType:  nd.AssetAgent,
			AssetName:  "reviewer",
			SourcePath: "/src/agents/reviewer",
			LinkPath:   "/home/.config/claude/agents/reviewer",
			Scope:      nd.ScopeGlobal,
			DeployedAt: time.Now(),
		},
	}

	msg := deploymentsLoadedMsg{deployments: deps, err: nil}
	updated, _ := m.Update(msg)
	rm := updated.(*removeScreen)

	if len(rm.deployments) != 2 {
		t.Fatalf("deployments count = %d, want 2", len(rm.deployments))
	}
	if rm.assetForm == nil {
		t.Fatal("assetForm is nil after loading deployments")
	}
}

func TestRemove_DeploymentsLoadedMsg_Error(t *testing.T) {
	svc := newMockServices()
	s := NewStyles(true)
	m := newRemoveScreen(svc, s, true)

	msg := deploymentsLoadedMsg{err: errors.New("disk error")}
	updated, _ := m.Update(msg)
	rm := updated.(*removeScreen)

	if rm.err == nil {
		t.Fatal("err is nil, want error")
	}
	if rm.err.Error() != "disk error" {
		t.Fatalf("err = %q, want %q", rm.err.Error(), "disk error")
	}
}

func TestRemove_RemoveDoneMsg_AllSucceeded(t *testing.T) {
	svc := newMockServices()
	s := NewStyles(true)
	m := newRemoveScreen(svc, s, true)
	m.step = removeRunning

	msg := removeDoneMsg{succeeded: 3, failed: nil}
	updated, _ := m.Update(msg)
	rm := updated.(*removeScreen)

	if rm.step != removeResult {
		t.Fatalf("step = %d, want removeResult (%d)", rm.step, removeResult)
	}
	if rm.succeeded != 3 {
		t.Fatalf("succeeded = %d, want 3", rm.succeeded)
	}
	if len(rm.failed) != 0 {
		t.Fatalf("failed count = %d, want 0", len(rm.failed))
	}
}

func TestRemove_RemoveDoneMsg_PartialFailure(t *testing.T) {
	svc := newMockServices()
	s := NewStyles(true)
	m := newRemoveScreen(svc, s, true)
	m.step = removeRunning

	failures := []deploy.RemoveError{
		{
			Identity: asset.Identity{SourceID: "src", Type: nd.AssetSkill, Name: "broken"},
			Err:      errors.New("permission denied"),
		},
	}
	msg := removeDoneMsg{succeeded: 2, failed: failures}
	updated, _ := m.Update(msg)
	rm := updated.(*removeScreen)

	if rm.step != removeResult {
		t.Fatalf("step = %d, want removeResult (%d)", rm.step, removeResult)
	}
	if rm.succeeded != 2 {
		t.Fatalf("succeeded = %d, want 2", rm.succeeded)
	}
	if len(rm.failed) != 1 {
		t.Fatalf("failed count = %d, want 1", len(rm.failed))
	}
}

func TestRemove_ResultView_ShowsSucceeded(t *testing.T) {
	svc := newMockServices()
	s := NewStyles(true)
	m := newRemoveScreen(svc, s, true)
	m.step = removeResult
	m.succeeded = 5

	v := m.View()
	if !strings.Contains(v.Content, "5") {
		t.Fatalf("result view should contain succeeded count '5', got %q", v.Content)
	}
	if !strings.Contains(v.Content, "removed") {
		t.Fatalf("result view should contain 'removed', got %q", v.Content)
	}
}

func TestRemove_ResultView_ShowsErrors(t *testing.T) {
	svc := newMockServices()
	s := NewStyles(true)
	m := newRemoveScreen(svc, s, true)
	m.step = removeResult
	m.succeeded = 1
	m.failed = []deploy.RemoveError{
		{
			Identity: asset.Identity{SourceID: "src", Type: nd.AssetSkill, Name: "broken"},
			Err:      errors.New("permission denied"),
		},
	}

	v := m.View()
	if !strings.Contains(v.Content, "1 failed") {
		t.Fatalf("result view should contain '1 failed', got %q", v.Content)
	}
	if !strings.Contains(v.Content, "permission denied") {
		t.Fatalf("result view should contain error message, got %q", v.Content)
	}
}

func TestRemove_EmptyView_WhenNothingDeployed(t *testing.T) {
	svc := newMockServices()
	s := NewStyles(true)
	m := newRemoveScreen(svc, s, true)

	// Simulate loading with no deployments.
	msg := deploymentsLoadedMsg{deployments: nil, err: nil}
	updated, _ := m.Update(msg)
	rm := updated.(*removeScreen)

	v := rm.View()
	if !strings.Contains(v.Content, NothingDeployed()) {
		t.Fatalf("expected NothingDeployed() message in view, got %q", v.Content)
	}
}

func TestRemove_ErrorView(t *testing.T) {
	svc := newMockServices()
	s := NewStyles(true)
	m := newRemoveScreen(svc, s, true)
	m.err = errors.New("failed to load state")

	v := m.View()
	if !strings.Contains(v.Content, "failed to load state") {
		t.Fatalf("expected error in view, got %q", v.Content)
	}
}

func TestRemoveCmd_AllSucceed(t *testing.T) {
	removeCalls := 0
	mockEng := &mockRemoveEngine{
		removeFn: func(_ deploy.RemoveRequest) error {
			removeCalls++
			return nil
		},
	}

	reqs := []deploy.RemoveRequest{
		{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "a"}, Scope: nd.ScopeGlobal},
		{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "b"}, Scope: nd.ScopeGlobal},
	}

	cmd := removeCmd(mockEng, reqs)
	msg := cmd()

	done, ok := msg.(removeDoneMsg)
	if !ok {
		t.Fatalf("expected removeDoneMsg, got %T", msg)
	}
	if done.succeeded != 2 {
		t.Fatalf("succeeded = %d, want 2", done.succeeded)
	}
	if len(done.failed) != 0 {
		t.Fatalf("failed = %d, want 0", len(done.failed))
	}
	if removeCalls != 2 {
		t.Fatalf("remove called %d times, want 2", removeCalls)
	}
}

func TestRemoveCmd_PartialFailure(t *testing.T) {
	mockEng := &mockRemoveEngine{
		removeFn: func(req deploy.RemoveRequest) error {
			if req.Identity.Name == "broken" {
				return errors.New("cannot remove")
			}
			return nil
		},
	}

	reqs := []deploy.RemoveRequest{
		{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "good"}, Scope: nd.ScopeGlobal},
		{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "broken"}, Scope: nd.ScopeGlobal},
	}

	cmd := removeCmd(mockEng, reqs)
	msg := cmd()

	done, ok := msg.(removeDoneMsg)
	if !ok {
		t.Fatalf("expected removeDoneMsg, got %T", msg)
	}
	if done.succeeded != 1 {
		t.Fatalf("succeeded = %d, want 1", done.succeeded)
	}
	if len(done.failed) != 1 {
		t.Fatalf("failed = %d, want 1", len(done.failed))
	}
	if done.failed[0].Identity.Name != "broken" {
		t.Fatalf("failed identity = %q, want %q", done.failed[0].Identity.Name, "broken")
	}
}

func TestRemoveCmd_AllFail(t *testing.T) {
	mockEng := &mockRemoveEngine{
		removeFn: func(_ deploy.RemoveRequest) error {
			return errors.New("disk full")
		},
	}

	reqs := []deploy.RemoveRequest{
		{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "a"}, Scope: nd.ScopeGlobal},
		{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "b"}, Scope: nd.ScopeGlobal},
	}

	cmd := removeCmd(mockEng, reqs)
	msg := cmd()

	done := msg.(removeDoneMsg)
	if done.succeeded != 0 {
		t.Fatalf("succeeded = %d, want 0", done.succeeded)
	}
	if len(done.failed) != 2 {
		t.Fatalf("failed = %d, want 2", len(done.failed))
	}
}

func TestRemove_RunningView(t *testing.T) {
	svc := newMockServices()
	s := NewStyles(true)
	m := newRemoveScreen(svc, s, true)
	m.step = removeRunning

	v := m.View()
	if !strings.Contains(v.Content, "Removing") {
		t.Fatalf("running view should contain 'Removing', got %q", v.Content)
	}
}

// mockRemoveEngine is a test double for the remove engine interface.
type mockRemoveEngine struct {
	removeFn func(deploy.RemoveRequest) error
}

func (m *mockRemoveEngine) Remove(req deploy.RemoveRequest) error {
	if m.removeFn != nil {
		return m.removeFn(req)
	}
	return nil
}

func TestRemove_DeploymentLabel(t *testing.T) {
	d := state.Deployment{
		SourceID:  "my-source",
		AssetType: nd.AssetSkill,
		AssetName: "greeting",
	}

	got := deploymentLabel(d)
	want := "skills/greeting (my-source)"
	if got != want {
		t.Fatalf("deploymentLabel() = %q, want %q", got, want)
	}
}

func TestRemove_DeploymentLabel_Context(t *testing.T) {
	d := state.Deployment{
		SourceID:  "my-source",
		AssetType: nd.AssetContext,
		AssetName: "CLAUDE.md",
	}

	got := deploymentLabel(d)
	want := "CLAUDE.md (my-source)"
	if got != want {
		t.Fatalf("deploymentLabel() = %q, want %q", got, want)
	}
}
