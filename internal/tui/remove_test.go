package tui

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

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

// H5: InputActive returns true during form steps
func TestRemove_InputActive_SelectAssets(t *testing.T) {
	svc := newMockServices()
	s := NewStyles(true)
	m := newRemoveScreen(svc, s, true)
	m.step = removeSelectAssets

	if !m.InputActive() {
		t.Fatal("InputActive() at selectAssets step should be true")
	}
}

func TestRemove_InputActive_Confirm(t *testing.T) {
	svc := newMockServices()
	s := NewStyles(true)
	m := newRemoveScreen(svc, s, true)
	m.step = removeConfirm

	if !m.InputActive() {
		t.Fatal("InputActive() at confirm step should be true")
	}
}

func TestRemove_InputActive_Running(t *testing.T) {
	svc := newMockServices()
	s := NewStyles(true)
	m := newRemoveScreen(svc, s, true)
	m.step = removeRunning

	if m.InputActive() {
		t.Fatal("InputActive() at running step should be false")
	}
}

func TestRemove_InputActive_Result(t *testing.T) {
	svc := newMockServices()
	s := NewStyles(true)
	m := newRemoveScreen(svc, s, true)
	m.step = removeResult

	if m.InputActive() {
		t.Fatal("InputActive() at result step should be false")
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
	if rm.vp == nil {
		t.Fatal("viewport should be created on error path")
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

// M3: Tests for removeBulkCmd
func TestRemoveBulkCmd_AllSucceed(t *testing.T) {
	mockEng := &mockBulkRemoveEngine{
		removeBulkFn: func(reqs []deploy.RemoveRequest) (*deploy.BulkRemoveResult, error) {
			return &deploy.BulkRemoveResult{Succeeded: reqs}, nil
		},
	}

	reqs := []deploy.RemoveRequest{
		{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "a"}, Scope: nd.ScopeGlobal},
		{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "b"}, Scope: nd.ScopeGlobal},
	}

	cmd := removeBulkCmd(mockEng, reqs)
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
}

func TestRemoveBulkCmd_PartialFailure(t *testing.T) {
	mockEng := &mockBulkRemoveEngine{
		removeBulkFn: func(reqs []deploy.RemoveRequest) (*deploy.BulkRemoveResult, error) {
			var result deploy.BulkRemoveResult
			for _, req := range reqs {
				if req.Identity.Name == "broken" {
					result.Failed = append(result.Failed, deploy.RemoveError{
						Identity: req.Identity,
						Err:      errors.New("cannot remove"),
					})
				} else {
					result.Succeeded = append(result.Succeeded, req)
				}
			}
			return &result, nil
		},
	}

	reqs := []deploy.RemoveRequest{
		{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "good"}, Scope: nd.ScopeGlobal},
		{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "broken"}, Scope: nd.ScopeGlobal},
	}

	cmd := removeBulkCmd(mockEng, reqs)
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

func TestRemoveBulkCmd_TotalFailure(t *testing.T) {
	mockEng := &mockBulkRemoveEngine{
		removeBulkFn: func(reqs []deploy.RemoveRequest) (*deploy.BulkRemoveResult, error) {
			return nil, errors.New("lock error")
		},
	}

	reqs := []deploy.RemoveRequest{
		{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "a"}, Scope: nd.ScopeGlobal},
		{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "b"}, Scope: nd.ScopeGlobal},
	}

	cmd := removeBulkCmd(mockEng, reqs)
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

// mockBulkRemoveEngine is a test double for the bulkRemover interface.
type mockBulkRemoveEngine struct {
	removeBulkFn func([]deploy.RemoveRequest) (*deploy.BulkRemoveResult, error)
}

func (m *mockBulkRemoveEngine) RemoveBulk(reqs []deploy.RemoveRequest) (*deploy.BulkRemoveResult, error) {
	if m.removeBulkFn != nil {
		return m.removeBulkFn(reqs)
	}
	return &deploy.BulkRemoveResult{Succeeded: reqs}, nil
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

// H3: buildRemoveRequests uses deployment's recorded scope, not current scope
func TestRemove_BuildRemoveRequests_UsesDeploymentScope(t *testing.T) {
	svc := newMockServices()
	// Mock returns global scope, but deployments are project-scoped
	svc.getScopeFn = func() nd.Scope { return nd.ScopeGlobal }

	s := NewStyles(true)
	m := newRemoveScreen(svc, s, true)
	m.deployments = []state.Deployment{
		{
			SourceID:    "src",
			AssetType:   nd.AssetSkill,
			AssetName:   "greeting",
			Scope:       nd.ScopeProject,
			ProjectPath: "/my/project",
		},
	}
	m.selected = []string{m.deployments[0].Identity().String()}

	reqs := m.buildRemoveRequests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	if reqs[0].Scope != nd.ScopeProject {
		t.Errorf("expected scope %q, got %q", nd.ScopeProject, reqs[0].Scope)
	}
	if reqs[0].ProjectRoot != "/my/project" {
		t.Errorf("expected project root %q, got %q", "/my/project", reqs[0].ProjectRoot)
	}
}

// --- Viewport wrapping tests (Unit 4) ---

// Verify that remove result with many failures renders via viewport when given dimensions.
func TestRemove_ResultViewport_ManyFailures(t *testing.T) {
	svc := newMockServices()
	s := NewStyles(true)
	m := newRemoveScreen(svc, s, true)
	m.pendingWidth = 80
	m.pendingHeight = 10
	m.step = removeRunning

	// Generate many failures to exceed viewport height.
	var failures []deploy.RemoveError
	for i := 0; i < 30; i++ {
		failures = append(failures, deploy.RemoveError{
			Identity: asset.Identity{
				SourceID: "src",
				Type:     nd.AssetSkill,
				Name:     fmt.Sprintf("asset-%03d", i),
			},
			Err: fmt.Errorf("permission denied"),
		})
	}

	msg := removeDoneMsg{succeeded: 0, failed: failures}
	updated, _ := m.Update(msg)
	rm := updated.(*removeScreen)

	if rm.vp == nil {
		t.Fatal("viewport should be initialized after transitioning to result")
	}
	if rm.vp.Width() != 80 {
		t.Fatalf("viewport width = %d, want 80", rm.vp.Width())
	}
	if rm.vp.Height() != 10 {
		t.Fatalf("viewport height = %d, want 10", rm.vp.Height())
	}

	v := rm.View()
	if v.Content == "" {
		t.Fatal("viewport-wrapped result view should not be empty")
	}
	// Underlying content should contain failure details.
	content := rm.viewResultContent()
	if !strings.Contains(content, "asset-000") {
		t.Error("result content should contain first failed asset")
	}
	if !strings.Contains(content, "30 failed") {
		t.Errorf("result content should show '30 failed'; got:\n%s", content)
	}
}

// Verify that j/k scroll keys are forwarded to viewport at remove result step.
func TestRemove_ResultViewport_ScrollForwarding(t *testing.T) {
	svc := newMockServices()
	s := NewStyles(true)
	m := newRemoveScreen(svc, s, true)
	m.pendingWidth = 80
	m.pendingHeight = 5
	m.step = removeRunning

	var failures []deploy.RemoveError
	for i := 0; i < 30; i++ {
		failures = append(failures, deploy.RemoveError{
			Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: fmt.Sprintf("a-%d", i)},
			Err:      errors.New("err"),
		})
	}

	m.Update(removeDoneMsg{succeeded: 0, failed: failures})

	// Send 'j' key — should be forwarded to viewport.
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	if cmd != nil {
		msg := cmd()
		switch msg.(type) {
		case PopToRootMsg, BackMsg:
			t.Fatal("j key should not produce navigation messages at result step")
		}
	}
}

// Verify enter at remove result step still emits PopToRootMsg with viewport active.
func TestRemove_ResultViewport_EnterStillReturns(t *testing.T) {
	svc := newMockServices()
	s := NewStyles(true)
	m := newRemoveScreen(svc, s, true)
	m.pendingWidth = 80
	m.pendingHeight = 10
	m.step = removeRunning

	m.Update(removeDoneMsg{succeeded: 3, failed: nil})

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter at result should emit a command even with viewport active")
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
func TestRemove_ResultViewport_ScreenSizeUpdates(t *testing.T) {
	svc := newMockServices()
	s := NewStyles(true)
	m := newRemoveScreen(svc, s, true)
	m.pendingWidth = 80
	m.pendingHeight = 10
	m.step = removeRunning

	m.Update(removeDoneMsg{succeeded: 1, failed: nil})

	if m.vp == nil {
		t.Fatal("viewport should exist after result transition")
	}

	m.Update(ScreenSizeMsg{Width: 120, Height: 25})

	if m.vp.Width() != 120 {
		t.Fatalf("viewport width after resize = %d, want 120", m.vp.Width())
	}
	if m.vp.Height() != 25 {
		t.Fatalf("viewport height after resize = %d, want 25", m.vp.Height())
	}
}

// Verify fallback rendering when viewport has zero dimensions.
func TestRemove_ResultViewport_FallbackWithoutDimensions(t *testing.T) {
	svc := newMockServices()
	s := NewStyles(true)
	m := newRemoveScreen(svc, s, true)
	m.step = removeResult
	m.succeeded = 3

	v := m.View()
	if v.Content == "" {
		t.Fatal("View() should fall back to raw string when viewport has zero dimensions")
	}
	if !strings.Contains(v.Content, "3") {
		t.Error("fallback view should contain succeeded count")
	}
}

// H2: dry-run mode shows preview
func TestRemove_DryRunView(t *testing.T) {
	svc := newMockServices()
	s := NewStyles(true)
	m := newRemoveScreen(svc, s, true)
	m.step = removeResult
	m.dryRun = true
	m.dryReqs = []deploy.RemoveRequest{
		{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "greeting"}, Scope: nd.ScopeGlobal},
	}

	v := m.View()
	if !strings.Contains(v.Content, "DRY RUN") {
		t.Errorf("dry-run view should contain 'DRY RUN'; got:\n%s", v.Content)
	}
	if !strings.Contains(v.Content, "greeting") {
		t.Errorf("dry-run view should list assets; got:\n%s", v.Content)
	}
}
