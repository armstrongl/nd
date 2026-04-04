package tui

import (
	"errors"
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

// Init() uses DeployEngine — when the engine is unavailable the cmd must
// return a deploymentsLoadedMsg with a non-nil error (not panic or hang).
func TestRemove_InitCmd_EngineUnavailable(t *testing.T) {
	svc := newMockServices()
	// Default mock: DeployEngine returns nil, nil.
	s := NewStyles(true)
	m := newRemoveScreen(svc, s, true)
	cmd := m.Init()
	msg := cmd()

	loaded, ok := msg.(deploymentsLoadedMsg)
	if !ok {
		t.Fatalf("expected deploymentsLoadedMsg, got %T", msg)
	}
	if loaded.err == nil {
		t.Fatal("expected error when deploy engine is nil")
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

func TestRemove_EscWhileLoading_SendsBackMsg(t *testing.T) {
	svc := newMockServices()
	s := NewStyles(true)
	m := newRemoveScreen(svc, s, true)
	// assetForm is nil (still loading) — ESC must still navigate back.
	_, cmd := m.updateSelectAssets(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("expected BackMsg cmd on ESC, got nil")
	}
	msg := cmd()
	if _, ok := msg.(BackMsg); !ok {
		t.Fatalf("expected BackMsg, got %T", msg)
	}
}

func TestRemove_EscOnConfirm_SendsBackMsg(t *testing.T) {
	svc := newMockServices()
	s := NewStyles(true)
	m := newRemoveScreen(svc, s, true)
	m.step = removeConfirm
	_, cmd := m.updateConfirm(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("expected BackMsg cmd on ESC, got nil")
	}
	msg := cmd()
	if _, ok := msg.(BackMsg); !ok {
		t.Fatalf("expected BackMsg, got %T", msg)
	}
}

func TestRemove_LoadingView_BeforeAssetsLoaded(t *testing.T) {
	svc := newMockServices()
	s := NewStyles(true)
	m := newRemoveScreen(svc, s, true)

	// Before deploymentsLoadedMsg arrives, view must show loading — not "nothing deployed".
	v := m.View()
	if strings.Contains(v.Content, NothingDeployed()) {
		t.Fatalf("expected loading message before assets loaded, got NothingDeployed(): %q", v.Content)
	}
	if !strings.Contains(v.Content, "Loading") {
		t.Fatalf("expected loading message before assets loaded, got %q", v.Content)
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

// TestRemove_ScrollBeforeFirstRender verifies that pressing j/k on the result
// screen works even when View() has never been called (resultLines not yet
// populated by the lazy initialisation in viewResult).
func TestRemove_ScrollBeforeFirstRender(t *testing.T) {
	svc := newMockServices()
	s := NewStyles(true)
	m := newRemoveScreen(svc, s, true)
	m.step = removeResult
	m.succeeded = 3
	m.height = 40

	// Ensure resultLines is nil — simulating a j keypress before first render.
	if m.resultLines != nil {
		t.Fatal("precondition: resultLines should be nil before any Update/View")
	}

	updated, _ := m.updateResult(tea.KeyPressMsg(tea.Key{Code: 'j', Text: "j"}))
	rm := updated.(*removeScreen)

	if len(rm.resultLines) == 0 {
		t.Fatal("resultLines should be populated after updateResult, not remain empty")
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

// Compile-time assertion: removeScreen satisfies FullHelpProvider.
var _ FullHelpProvider = (*removeScreen)(nil)

func TestRemove_FullHelpItems_SelectAssets(t *testing.T) {
	svc := newMockServices()
	s := NewStyles(true)
	m := newRemoveScreen(svc, s, true)
	m.step = removeSelectAssets

	items := m.FullHelpItems()

	hasToggle := false
	hasEnterConfirm := false
	for _, item := range items {
		if item.Key == "x/space" && item.Desc == "toggle" {
			hasToggle = true
		}
		if item.Key == "enter" && item.Desc == "confirm" {
			hasEnterConfirm = true
		}
	}
	if !hasToggle {
		t.Errorf("FullHelpItems at selectAssets should include 'x/space toggle'; got: %v", items)
	}
	if !hasEnterConfirm {
		t.Errorf("FullHelpItems at selectAssets should include 'enter confirm'; got: %v", items)
	}
}

func TestRemove_FullHelpItems_Confirm(t *testing.T) {
	svc := newMockServices()
	s := NewStyles(true)
	m := newRemoveScreen(svc, s, true)
	m.step = removeConfirm

	items := m.FullHelpItems()

	hasEnterConfirm := false
	for _, item := range items {
		if item.Key == "enter" && item.Desc == "confirm" {
			hasEnterConfirm = true
		}
	}
	if !hasEnterConfirm {
		t.Errorf("FullHelpItems at confirm should include 'enter confirm'; got: %v", items)
	}
}

func TestRemove_FullHelpItems_Result(t *testing.T) {
	svc := newMockServices()
	s := NewStyles(true)
	m := newRemoveScreen(svc, s, true)
	m.step = removeResult

	items := m.FullHelpItems()

	hasEnterReturn := false
	for _, item := range items {
		if item.Key == "enter" && item.Desc == "return" {
			hasEnterReturn = true
		}
	}
	if !hasEnterReturn {
		t.Errorf("FullHelpItems at result should include 'enter return'; got: %v", items)
	}
}

func TestRemove_RunningViewShowsAssetCount(t *testing.T) {
	svc := newMockServices()
	s := NewStyles(true)
	m := newRemoveScreen(svc, s, true)
	m.step = removeRunning
	m.selected = []string{"s:skills/a", "s:skills/b"}

	v := m.viewRunning()
	if !strings.Contains(v.Content, "2 asset(s)") {
		t.Errorf("running view should show asset count, got: %q", v.Content)
	}
}
