package tui

// Tests in this file close the TUI test-coverage gaps identified by the TUI
// audit (taskmd agtaqm). They cover four behaviour groups that were
// implemented but largely untested — huh form-abort handling, OpLog recording,
// dry-run previews, and nil-DeployEngine error paths — plus a handful of
// list/scroll/empty-state edge cases.
//
// All tests build screen structs directly with testStyles() (unstyled,
// deterministic) and newMockServices(), overriding the mock's *Fn fields to
// inject behaviour. None of them start a real Bubble Tea program.
//
// The abort tests rely on huh.Form.Update short-circuiting when the form is
// already non-Normal (see charm.land/huh/v2 form.go: "If the form is aborted or
// completed there's no need to update it"). Setting form.State = StateAborted
// and then calling the owning update method exercises the screen's abort branch
// exactly as pressing Esc/Ctrl+C inside the form would.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"

	"github.com/armstrongl/nd/internal/asset"
	"github.com/armstrongl/nd/internal/deploy"
	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/oplog"
	"github.com/armstrongl/nd/internal/profile"
	"github.com/armstrongl/nd/internal/source"
	"github.com/armstrongl/nd/internal/sourcemanager"
	"github.com/armstrongl/nd/internal/state"
)

// readSingleOplogEntry reads operations.log from dir and decodes the one entry
// it must contain, failing the test if the file is missing, empty, or has more
// than one line.
func readSingleOplogEntry(t *testing.T, dir string) oplog.LogEntry {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "operations.log"))
	if err != nil {
		t.Fatalf("read operations.log: %v", err)
	}
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		t.Fatal("operations.log is empty; expected exactly one entry")
	}
	lines := strings.Split(trimmed, "\n")
	if len(lines) != 1 {
		t.Fatalf("expected exactly one oplog entry, got %d lines: %q", len(lines), trimmed)
	}
	var entry oplog.LogEntry
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatalf("decode oplog entry: %v", err)
	}
	return entry
}

// =============================================================================
// StateAborted (huh form abort) transition tests
// =============================================================================

// --- deploy ---

func TestDeploy_PickTypeAbort_SendsBackMsg(t *testing.T) {
	ds := newDeployScreen(newMockServices(), testStyles(), true)
	ds.typeForm.State = huh.StateAborted

	_, cmd := ds.updatePickType(nil)
	if cmd == nil {
		t.Fatal("expected BackMsg cmd when type form is aborted, got nil")
	}
	if _, ok := cmd().(BackMsg); !ok {
		t.Fatalf("expected BackMsg, got %T", cmd())
	}
}

func TestDeploy_SelectAssetsAbort_SendsBackMsg(t *testing.T) {
	ds := newTestDeployScreen(deploySelectAssets)
	ds.assets = testAssets()
	ds.buildAssetForm()
	ds.assetForm.State = huh.StateAborted

	_, cmd := ds.updateSelectAssets(nil)
	if cmd == nil {
		t.Fatal("expected BackMsg cmd when asset form is aborted, got nil")
	}
	if _, ok := cmd().(BackMsg); !ok {
		t.Fatalf("expected BackMsg, got %T", cmd())
	}
}

func TestDeploy_ConflictConfirmAbort_MovesToResult(t *testing.T) {
	ds := newTestDeployScreen(deployRunning)
	ds.step = deployConflictConfirm
	ds.firstSucceeded = []deploy.DeployResult{
		{Deployment: state.Deployment{AssetName: "ok-skill", AssetType: nd.AssetSkill}},
	}
	ds.conflictFails = []deploy.DeployError{
		{AssetName: "greeting", AssetType: nd.AssetSkill, Err: makeConflictError("greeting", "/p")},
	}
	ds.conflictReqs = []deploy.DeployRequest{{ForceReplace: true}}
	ds.buildConflictForm()
	ds.conflictForm.State = huh.StateAborted

	updated, _ := ds.updateConflictConfirm(nil)
	ds2 := updated.(*deployScreen)

	if ds2.step != deployResult {
		t.Fatalf("step after conflict abort = %d, want deployResult (%d)", ds2.step, deployResult)
	}
	// Cancelling conflict resolution surfaces the first-run success plus the
	// unresolved conflict as a failure.
	if len(ds2.succeeded) != 1 {
		t.Errorf("succeeded = %d, want 1 (first-run success preserved)", len(ds2.succeeded))
	}
	if len(ds2.failed) != 1 {
		t.Errorf("failed = %d, want 1 (unresolved conflict)", len(ds2.failed))
	}
}

// --- remove ---

func TestRemove_SelectAssetsAbort_SendsBackMsg(t *testing.T) {
	m := newRemoveScreen(newMockServices(), testStyles(), true)
	m.Update(deploymentsLoadedMsg{deployments: []state.Deployment{
		{SourceID: "s", AssetType: nd.AssetSkill, AssetName: "a", Scope: nd.ScopeGlobal},
	}})
	if m.assetForm == nil {
		t.Fatal("precondition: assetForm should be built after deploymentsLoadedMsg")
	}
	m.assetForm.State = huh.StateAborted

	_, cmd := m.updateSelectAssets(nil)
	if cmd == nil {
		t.Fatal("expected BackMsg cmd when asset form is aborted, got nil")
	}
	if _, ok := cmd().(BackMsg); !ok {
		t.Fatalf("expected BackMsg, got %T", cmd())
	}
}

func TestRemove_ConfirmAbort_SendsBackMsg(t *testing.T) {
	m := newRemoveScreen(newMockServices(), testStyles(), true)
	m.deployments = []state.Deployment{
		{SourceID: "s", AssetType: nd.AssetSkill, AssetName: "a", Scope: nd.ScopeGlobal},
	}
	m.selected = []string{m.deployments[0].Identity().String()}
	m.transitionToConfirm()
	if m.confirmForm == nil {
		t.Fatal("precondition: confirmForm should be built by transitionToConfirm")
	}
	m.confirmForm.State = huh.StateAborted

	_, cmd := m.updateConfirm(nil)
	if cmd == nil {
		t.Fatal("expected BackMsg cmd when confirm form is aborted, got nil")
	}
	if _, ok := cmd().(BackMsg); !ok {
		t.Fatalf("expected BackMsg, got %T", cmd())
	}
}

// --- doctor ---

func TestDoctorScreen_ConfirmAbort_SendsBackMsg(t *testing.T) {
	d := newDoctorScreen(newMockServices(), testStyles(), true)
	d.Update(doctorCheckedMsg{issues: []state.HealthCheck{
		{Deployment: state.Deployment{AssetName: "broken"}, Status: state.HealthBroken},
	}})
	if d.step != doctorConfirm || d.confirmForm == nil {
		t.Fatalf("precondition: expected doctorConfirm with a form, got step %d", d.step)
	}
	d.confirmForm.State = huh.StateAborted

	_, cmd := d.updateConfirm(nil)
	if cmd == nil {
		t.Fatal("expected BackMsg cmd when confirm form is aborted, got nil")
	}
	if _, ok := cmd().(BackMsg); !ok {
		t.Fatalf("expected BackMsg, got %T", cmd())
	}
}

// --- profile ---

func TestProfileScreen_MenuAbort_SendsBackMsg(t *testing.T) {
	s := newProfileScreen(newMockServices(), testStyles(), true)
	s.buildMenu()
	s.menuForm.State = huh.StateAborted

	_, cmd := s.updateMenu(nil)
	if cmd == nil {
		t.Fatal("expected BackMsg cmd when menu form is aborted, got nil")
	}
	if _, ok := cmd().(BackMsg); !ok {
		t.Fatalf("expected BackMsg, got %T", cmd())
	}
}

func TestProfileScreen_SwitchAbort_ReturnsToMenu(t *testing.T) {
	s := newProfileScreen(newMockServices(), testStyles(), true)
	s.profiles = []profile.ProfileSummary{{Name: "go-dev"}}
	s.buildSwitchForm()
	if s.step != profileSwitch || s.switchForm == nil {
		t.Fatalf("precondition: expected profileSwitch with a form, got step %d", s.step)
	}
	s.switchForm.State = huh.StateAborted

	s.updateSwitchForm(nil)
	if s.step != profileMenu {
		t.Fatalf("step after switch abort = %d, want profileMenu (%d)", s.step, profileMenu)
	}
}

func TestProfileScreen_CreateAbort_ReturnsToMenu(t *testing.T) {
	s := newProfileScreen(newMockServices(), testStyles(), true)
	s.buildCreateForm()
	if s.step != profileCreateName || s.createForm == nil {
		t.Fatalf("precondition: expected profileCreateName with a form, got step %d", s.step)
	}
	s.createForm.State = huh.StateAborted

	s.updateCreateForm(nil)
	if s.step != profileMenu {
		t.Fatalf("step after create abort = %d, want profileMenu (%d)", s.step, profileMenu)
	}
}

// --- snapshot ---

func TestSnapshotScreen_MenuAbort_SendsBackMsg(t *testing.T) {
	s := newSnapshotScreen(newMockServices(), testStyles(), true)
	s.buildMenu()
	s.menuForm.State = huh.StateAborted

	_, cmd := s.updateMenu(nil)
	if cmd == nil {
		t.Fatal("expected BackMsg cmd when menu form is aborted, got nil")
	}
	if _, ok := cmd().(BackMsg); !ok {
		t.Fatalf("expected BackMsg, got %T", cmd())
	}
}

func TestSnapshotScreen_SaveAbort_ReturnsToMenu(t *testing.T) {
	s := newSnapshotScreen(newMockServices(), testStyles(), true)
	s.buildSaveForm()
	if s.step != snapshotSaveName || s.saveForm == nil {
		t.Fatalf("precondition: expected snapshotSaveName with a form, got step %d", s.step)
	}
	s.saveForm.State = huh.StateAborted

	s.updateSaveForm(nil)
	if s.step != snapshotMenu {
		t.Fatalf("step after save abort = %d, want snapshotMenu (%d)", s.step, snapshotMenu)
	}
}

func TestSnapshotScreen_RestoreSelectAbort_ReturnsToMenu(t *testing.T) {
	s := newSnapshotScreen(newMockServices(), testStyles(), true)
	s.snapshots = []profile.SnapshotSummary{{Name: "snap", DeploymentCount: 1}}
	s.buildRestoreForm()
	if s.step != snapshotRestoreSelect || s.restoreForm == nil {
		t.Fatalf("precondition: expected snapshotRestoreSelect with a form, got step %d", s.step)
	}
	s.restoreForm.State = huh.StateAborted

	s.updateRestoreSelect(nil)
	if s.step != snapshotMenu {
		t.Fatalf("step after restore-select abort = %d, want snapshotMenu (%d)", s.step, snapshotMenu)
	}
}

func TestSnapshotScreen_RestoreConfirmAbort_ReturnsToMenu(t *testing.T) {
	s := newSnapshotScreen(newMockServices(), testStyles(), true)
	s.snapshots = []profile.SnapshotSummary{{Name: "snap", DeploymentCount: 1}}
	s.buildRestoreForm()
	s.restoreChoice = "snap"
	s.buildRestoreConfirm()
	if s.confirmForm == nil {
		t.Fatal("precondition: confirmForm should be built by buildRestoreConfirm")
	}
	s.confirmForm.State = huh.StateAborted

	s.updateRestoreSelect(nil)
	if s.step != snapshotMenu {
		t.Fatalf("step after restore-confirm abort = %d, want snapshotMenu (%d)", s.step, snapshotMenu)
	}
}

// --- source ---

func TestSourceScreen_MenuAbort_SendsBackMsg(t *testing.T) {
	s := newSourceScreen(newMockServices(), testStyles(), true)
	s.buildMenu()
	s.menuForm.State = huh.StateAborted

	_, cmd := s.updateMenu(nil)
	if cmd == nil {
		t.Fatal("expected BackMsg cmd when menu form is aborted, got nil")
	}
	if _, ok := cmd().(BackMsg); !ok {
		t.Fatalf("expected BackMsg, got %T", cmd())
	}
}

func TestSourceScreen_AddAbort_ReturnsToMenu(t *testing.T) {
	s := newSourceScreen(newMockServices(), testStyles(), true)
	s.buildAddForm("local")
	if s.step != sourceAddLocalInput || s.addForm == nil {
		t.Fatalf("precondition: expected sourceAddLocalInput with a form, got step %d", s.step)
	}
	s.addForm.State = huh.StateAborted

	s.updateAddForm(nil)
	if s.step != sourceMenu {
		t.Fatalf("step after add abort = %d, want sourceMenu (%d)", s.step, sourceMenu)
	}
}

func TestSourceScreen_RemoveSelectAbort_ReturnsToMenu(t *testing.T) {
	s := newSourceScreen(newMockServices(), testStyles(), true)
	s.sources = []source.Source{{ID: "s1"}}
	s.buildRemoveForm()
	if s.step != sourceRemoveSelect || s.removeForm == nil {
		t.Fatalf("precondition: expected sourceRemoveSelect with a form, got step %d", s.step)
	}
	s.removeForm.State = huh.StateAborted

	s.updateRemove(nil)
	if s.step != sourceMenu {
		t.Fatalf("step after remove-select abort = %d, want sourceMenu (%d)", s.step, sourceMenu)
	}
}

func TestSourceScreen_RemoveConfirmAbort_ReturnsToMenu(t *testing.T) {
	s := newSourceScreen(newMockServices(), testStyles(), true)
	s.sources = []source.Source{{ID: "s1"}}
	s.buildRemoveForm()
	s.removeChoice = "s1"
	s.buildRemoveConfirm()
	if s.step != sourceRemoveConfirm || s.confirmForm == nil {
		t.Fatalf("precondition: expected sourceRemoveConfirm with a form, got step %d", s.step)
	}
	s.confirmForm.State = huh.StateAborted

	s.updateRemove(nil)
	if s.step != sourceMenu {
		t.Fatalf("step after remove-confirm abort = %d, want sourceMenu (%d)", s.step, sourceMenu)
	}
}

// --- pin ---

func TestPinScreen_SelectAbort_SendsBackMsg(t *testing.T) {
	s := newPinScreen(newMockServices(), testStyles(), true)
	s.Update(pinLoadedMsg{deployments: []state.Deployment{
		{SourceID: "s", AssetType: nd.AssetSkill, AssetName: "a", Origin: nd.OriginManual},
	}})
	if s.assetForm == nil {
		t.Fatal("precondition: assetForm should be built after pinLoadedMsg")
	}
	s.assetForm.State = huh.StateAborted

	_, cmd := s.updateSelect(nil)
	if cmd == nil {
		t.Fatal("expected BackMsg cmd when asset form is aborted, got nil")
	}
	if _, ok := cmd().(BackMsg); !ok {
		t.Fatalf("expected BackMsg, got %T", cmd())
	}
}

func TestPinScreen_ConfirmAbort_SendsBackMsg(t *testing.T) {
	s := newPinScreen(newMockServices(), testStyles(), true)
	dep := state.Deployment{SourceID: "s", AssetType: nd.AssetSkill, AssetName: "a", Origin: nd.OriginManual}
	s.Update(pinLoadedMsg{deployments: []state.Deployment{dep}})
	// Select the (currently unpinned) asset so the diff is non-zero and a
	// confirm form is built rather than short-circuiting back.
	s.selected = []string{dep.Identity().String()}
	s.buildConfirm()
	if s.step != pinConfirm || s.confirmForm == nil {
		t.Fatalf("precondition: expected pinConfirm with a form, got step %d", s.step)
	}
	s.confirmForm.State = huh.StateAborted

	_, cmd := s.updateConfirm(nil)
	if cmd == nil {
		t.Fatal("expected BackMsg cmd when confirm form is aborted, got nil")
	}
	if _, ok := cmd().(BackMsg); !ok {
		t.Fatalf("expected BackMsg, got %T", cmd())
	}
}

// --- settings ---

func TestSettingsScreen_MenuAbort_SendsBackMsg(t *testing.T) {
	s := newSettingsScreen(newMockServices(), testStyles(), true)
	s.form.State = huh.StateAborted

	_, cmd := s.updateMenu(nil)
	if cmd == nil {
		t.Fatal("expected BackMsg cmd when menu form is aborted, got nil")
	}
	if _, ok := cmd().(BackMsg); !ok {
		t.Fatalf("expected BackMsg, got %T", cmd())
	}
}

func TestSettingsScreen_ScopeFormAbort_ReturnsToMenu(t *testing.T) {
	s := newSettingsScreen(newMockServices(), testStyles(), true)
	s.buildScopeForm()
	if s.step != settingsSwitchScope || s.scopeForm == nil {
		t.Fatalf("precondition: expected settingsSwitchScope with a form, got step %d", s.step)
	}
	s.scopeForm.State = huh.StateAborted

	s.updateScopeForm(nil)
	if s.step != settingsMenu {
		t.Fatalf("step after scope-form abort = %d, want settingsMenu (%d)", s.step, settingsMenu)
	}
}

// =============================================================================
// OpLog recording tests
// =============================================================================

func TestDeploy_LogOplog_WritesDeployEntry(t *testing.T) {
	dir := t.TempDir()
	w := oplog.NewWriter(dir)
	svc := newMockServices()
	svc.opLogFn = func() *oplog.Writer { return w }
	svc.getScopeFn = func() nd.Scope { return nd.ScopeProject }

	ds := newTestDeployScreen(deployRunning)
	ds.svc = svc

	// Drive Update(deployDoneMsg) with no conflicts so the deploy.go:224 path
	// (logOplog on the no-conflict branch) runs.
	updated, _ := ds.Update(deployDoneMsg{
		succeeded: []deploy.DeployResult{
			{Deployment: state.Deployment{SourceID: "local", AssetType: nd.AssetSkill, AssetName: "go-test"}},
		},
		failed: []deploy.DeployError{
			{AssetName: "broken", AssetType: nd.AssetRule, Err: fmt.Errorf("permission denied")},
		},
	})
	if updated.(*deployScreen).step != deployResult {
		t.Fatalf("precondition: expected deployResult step, got %d", updated.(*deployScreen).step)
	}

	entry := readSingleOplogEntry(t, dir)
	if entry.Operation != oplog.OpDeploy {
		t.Errorf("Operation = %q, want %q", entry.Operation, oplog.OpDeploy)
	}
	if entry.Succeeded != 1 {
		t.Errorf("Succeeded = %d, want 1 (matches ds.succeeded)", entry.Succeeded)
	}
	if entry.Failed != 1 {
		t.Errorf("Failed = %d, want 1 (matches ds.failed)", entry.Failed)
	}
	if entry.Scope != nd.ScopeProject {
		t.Errorf("Scope = %q, want %q (from svc.GetScope())", entry.Scope, nd.ScopeProject)
	}
	if len(entry.Assets) != 1 {
		t.Fatalf("Assets len = %d, want 1 (asset.Identity list from succeeded)", len(entry.Assets))
	}
	got := entry.Assets[0]
	want := asset.Identity{SourceID: "local", Type: nd.AssetSkill, Name: "go-test"}
	if got != want {
		t.Errorf("Assets[0] = %+v, want %+v", got, want)
	}
	if entry.Timestamp.IsZero() {
		t.Error("Timestamp should be non-zero (set via time.Now())")
	}
}

func TestRemove_RemoveDoneMsg_WritesOplogEntry(t *testing.T) {
	dir := t.TempDir()
	w := oplog.NewWriter(dir)
	svc := newMockServices()
	svc.opLogFn = func() *oplog.Writer { return w }
	svc.getScopeFn = func() nd.Scope { return nd.ScopeGlobal }

	m := newRemoveScreen(svc, testStyles(), true)
	m.step = removeRunning

	m.Update(removeDoneMsg{
		succeeded: 2,
		failed: []deploy.RemoveError{
			{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "broken"}, Err: fmt.Errorf("locked")},
		},
	})

	entry := readSingleOplogEntry(t, dir)
	if entry.Operation != oplog.OpRemove {
		t.Errorf("Operation = %q, want %q", entry.Operation, oplog.OpRemove)
	}
	if entry.Succeeded != 2 {
		t.Errorf("Succeeded = %d, want 2", entry.Succeeded)
	}
	if entry.Failed != 1 {
		t.Errorf("Failed = %d, want 1", entry.Failed)
	}
	if entry.Scope != nd.ScopeGlobal {
		t.Errorf("Scope = %q, want %q", entry.Scope, nd.ScopeGlobal)
	}
	if entry.Timestamp.IsZero() {
		t.Error("Timestamp should be non-zero (set via time.Now())")
	}
}

// =============================================================================
// Dry-run behaviour tests
// =============================================================================

func TestDeploy_DryRun_SkipsEngineAndPreviews(t *testing.T) {
	svc := newMockServices()
	svc.isDryRunFn = func() bool { return true }
	svc.deployEngineFn = func() (*deploy.Engine, error) {
		t.Fatal("DeployEngine must not be called in dry-run mode")
		return nil, nil
	}

	ds := newTestDeployScreen(deploySelectAssets)
	ds.svc = svc
	ds.assets = testAssets()
	ds.selected = []string{assetKey(ds.assets[0])}

	ds.startDeploy()

	if !ds.dryRun {
		t.Error("dryRun should be true after startDeploy in dry-run mode")
	}
	if ds.step != deployResult {
		t.Errorf("step = %d, want deployResult (%d)", ds.step, deployResult)
	}

	v := ds.View()
	if !strings.Contains(v.Content, "[DRY RUN]") {
		t.Errorf("dry-run view should contain '[DRY RUN]'; got:\n%s", v.Content)
	}
	if !strings.Contains(v.Content, "go-test") {
		t.Errorf("dry-run view should list asset names; got:\n%s", v.Content)
	}
}

func TestRemove_DryRun_SkipsEngineAndPreviews(t *testing.T) {
	svc := newMockServices()
	svc.isDryRunFn = func() bool { return true }
	svc.deployEngineFn = func() (*deploy.Engine, error) {
		t.Fatal("DeployEngine must not be called in dry-run mode")
		return nil, nil
	}

	m := newRemoveScreen(svc, testStyles(), true)
	m.deployments = []state.Deployment{
		{SourceID: "s", AssetType: nd.AssetSkill, AssetName: "greeting", Scope: nd.ScopeGlobal},
	}
	m.selected = []string{m.deployments[0].Identity().String()}

	m.transitionToRunning()

	if !m.dryRun {
		t.Error("dryRun should be true after transitionToRunning in dry-run mode")
	}
	if m.step != removeResult {
		t.Errorf("step = %d, want removeResult (%d)", m.step, removeResult)
	}
	if len(m.dryReqs) != 1 {
		t.Fatalf("dryReqs = %d, want 1", len(m.dryReqs))
	}

	v := m.View()
	if !strings.Contains(v.Content, "Would remove") {
		t.Errorf("dry-run remove view should contain 'Would remove'; got:\n%s", v.Content)
	}
}

// =============================================================================
// Nil DeployEngine safety tests
// =============================================================================

func TestDeploy_StartDeploy_NilEngine_UserFacingError(t *testing.T) {
	// Default mock: DeployEngine() returns (nil, nil), IsDryRun() false.
	ds := newTestDeployScreen(deploySelectAssets)
	ds.assets = testAssets()
	ds.selected = []string{assetKey(ds.assets[0])}

	cmd := ds.startDeploy()

	if cmd != nil {
		t.Error("startDeploy should return nil cmd when the engine is nil")
	}
	if ds.err == nil {
		t.Fatal("ds.err should be set when the deploy engine is nil")
	}
	if ds.err.Error() != "deploy engine not available" {
		t.Errorf("ds.err = %q, want %q", ds.err.Error(), "deploy engine not available")
	}
	// User-facing: rendered by the screen's error view, not a panic/empty string.
	v := ds.View()
	if !strings.Contains(v.Content, "deploy engine not available") {
		t.Errorf("error view should render the message; got:\n%s", v.Content)
	}
}

func TestRemove_TransitionToRunning_NilEngine_UserFacingError(t *testing.T) {
	m := newRemoveScreen(newMockServices(), testStyles(), true)
	m.deployments = []state.Deployment{
		{SourceID: "s", AssetType: nd.AssetSkill, AssetName: "a", Scope: nd.ScopeGlobal},
	}
	m.selected = []string{m.deployments[0].Identity().String()}

	_, cmd := m.transitionToRunning()

	if cmd != nil {
		t.Error("transitionToRunning should return nil cmd when the engine is nil")
	}
	if m.err == nil {
		t.Fatal("m.err should be set when the deploy engine is nil")
	}
	if m.err.Error() != "deploy engine not available" {
		t.Errorf("m.err = %q, want %q", m.err.Error(), "deploy engine not available")
	}
	v := m.View()
	if !strings.Contains(v.Content, "deploy engine not available") {
		t.Errorf("error view should render the message; got:\n%s", v.Content)
	}
}

func TestProfileScreen_RunSwitch_NilEngine_UserFacingError(t *testing.T) {
	dir := t.TempDir()
	mgr := profile.NewManager(
		profile.NewStore(filepath.Join(dir, "profiles"), filepath.Join(dir, "snapshots")),
		state.NewStore(filepath.Join(dir, "deployments.yaml")),
	)
	svc := newMockServices()
	svc.profileManagerFn = func() (*profile.Manager, error) { return mgr, nil }
	// DeployEngine stays nil (default mock) so runSwitch hits the engine guard.

	s := newProfileScreen(svc, testStyles(), true)
	s.switchChoice = "target"

	msg := s.runSwitch()()
	switched, ok := msg.(profileSwitchedMsg)
	if !ok {
		t.Fatalf("runSwitch cmd returned %T, want profileSwitchedMsg", msg)
	}
	if switched.err == nil {
		t.Fatal("profileSwitchedMsg.err should be non-nil when the engine is nil")
	}
	if switched.err.Error() != "deploy engine not available" {
		t.Errorf("err = %q, want %q", switched.err.Error(), "deploy engine not available")
	}

	updated, _ := s.Update(switched)
	v := updated.(*profileScreen).View()
	if !strings.Contains(v.Content, "deploy engine not available") {
		t.Errorf("error should be rendered by the screen; got:\n%s", v.Content)
	}
}

func TestSnapshotScreen_RunRestore_NilEngine_UserFacingError(t *testing.T) {
	dir := t.TempDir()
	mgr := profile.NewManager(
		profile.NewStore(filepath.Join(dir, "profiles"), filepath.Join(dir, "snapshots")),
		state.NewStore(filepath.Join(dir, "deployments.yaml")),
	)
	svc := newMockServices()
	svc.profileManagerFn = func() (*profile.Manager, error) { return mgr, nil }
	// DeployEngine stays nil (default mock) so runRestore hits the engine guard.

	s := newSnapshotScreen(svc, testStyles(), true)
	s.restoreChoice = "snap"

	msg := s.runRestore()()
	restored, ok := msg.(snapshotRestoredMsg)
	if !ok {
		t.Fatalf("runRestore cmd returned %T, want snapshotRestoredMsg", msg)
	}
	if restored.err == nil {
		t.Fatal("snapshotRestoredMsg.err should be non-nil when the engine is nil")
	}
	if restored.err.Error() != "deploy engine not available" {
		t.Errorf("err = %q, want %q", restored.err.Error(), "deploy engine not available")
	}

	updated, _ := s.Update(restored)
	v := updated.(*snapshotScreen).View()
	if !strings.Contains(v.Content, "deploy engine not available") {
		t.Errorf("error should be rendered by the screen; got:\n%s", v.Content)
	}
}

// =============================================================================
// Symlink strategy / request-building tests
// =============================================================================

func TestDeploy_StartDeploy_DefaultStrategy(t *testing.T) {
	// Default mock: SourceManager() returns (nil, nil), so the strategy falls
	// back to the nd.SymlinkAbsolute default.
	ds := newTestDeployScreen(deploySelectAssets)
	ds.assets = testAssets()
	ds.selected = []string{assetKey(ds.assets[0]), assetKey(ds.assets[2])}

	ds.startDeploy()

	if len(ds.reqs) != 2 {
		t.Fatalf("reqs = %d, want 2", len(ds.reqs))
	}
	for _, req := range ds.reqs {
		if req.Strategy != nd.SymlinkAbsolute {
			t.Errorf("req %q Strategy = %q, want default %q", req.Asset.Name, req.Strategy, nd.SymlinkAbsolute)
		}
	}
}

func TestDeploy_StartDeploy_ConfigStrategyOverride(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	cfgYAML := "version: 1\n" +
		"default_scope: global\n" +
		"default_agent: claude-code\n" +
		"symlink_strategy: relative\n"
	if err := os.WriteFile(configPath, []byte(cfgYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	sm, err := sourcemanager.New(configPath, "")
	if err != nil {
		t.Fatalf("sourcemanager.New: %v", err)
	}
	if sm.Config().SymlinkStrategy != nd.SymlinkRelative {
		t.Fatalf("precondition: config strategy = %q, want %q", sm.Config().SymlinkStrategy, nd.SymlinkRelative)
	}

	svc := newMockServices()
	svc.sourceManagerFn = func() (*sourcemanager.SourceManager, error) { return sm, nil }

	ds := newTestDeployScreen(deploySelectAssets)
	ds.svc = svc
	ds.assets = testAssets()
	ds.selected = []string{assetKey(ds.assets[0])}

	ds.startDeploy()

	if len(ds.reqs) != 1 {
		t.Fatalf("reqs = %d, want 1", len(ds.reqs))
	}
	if ds.reqs[0].Strategy != nd.SymlinkRelative {
		t.Errorf("Strategy = %q, want %q from config override", ds.reqs[0].Strategy, nd.SymlinkRelative)
	}
}

func TestDeploy_StartDeploy_SetsSourcePath(t *testing.T) {
	ds := newTestDeployScreen(deploySelectAssets)
	ds.assets = testAssets()
	ds.selected = []string{
		assetKey(ds.assets[0]),
		assetKey(ds.assets[1]),
		assetKey(ds.assets[2]),
	}

	ds.startDeploy()

	if len(ds.reqs) != 3 {
		t.Fatalf("reqs = %d, want 3", len(ds.reqs))
	}
	sourcePathByName := make(map[string]string, len(ds.reqs))
	for _, req := range ds.reqs {
		sourcePathByName[req.Asset.Name] = req.Asset.SourcePath
	}
	for _, a := range ds.assets {
		if got := sourcePathByName[a.Name]; got != a.SourcePath {
			t.Errorf("req for %q has SourcePath %q, want %q", a.Name, got, a.SourcePath)
		}
	}
}

func TestDoctorScreen_BrokenAndMissing_RenderedInView(t *testing.T) {
	d := newDoctorScreen(newMockServices(), testStyles(), true)
	d.Update(doctorCheckedMsg{issues: []state.HealthCheck{
		{Deployment: state.Deployment{AssetName: "broken-link"}, Status: state.HealthBroken, Detail: "target missing"},
		{Deployment: state.Deployment{AssetName: "missing-link"}, Status: state.HealthMissing, Detail: "symlink deleted"},
	}})

	v := d.View()
	if !strings.Contains(v.Content, "broken-link") || !strings.Contains(v.Content, "missing-link") {
		t.Errorf("doctor view should list stale/broken asset names; got:\n%s", v.Content)
	}
	if !strings.Contains(v.Content, GlyphBroken) {
		t.Errorf("HealthBroken should render GlyphBroken via styleGlyphWith; got:\n%s", v.Content)
	}
	if !strings.Contains(v.Content, GlyphMissing) {
		t.Errorf("HealthMissing should render GlyphMissing via styleGlyphWith; got:\n%s", v.Content)
	}
}

func TestDeploy_WrappedConflictError_MovesToConflictConfirm(t *testing.T) {
	ds := newTestDeployScreen(deployRunning)
	ds.reqs = []deploy.DeployRequest{
		{Asset: asset.Asset{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "greeting"}}},
	}
	// A ConflictError wrapped in another error must still be matched by
	// errors.As and route into the conflict-resolution step.
	wrapped := fmt.Errorf("bulk deploy: %w", makeConflictError("greeting", "/target"))
	updated, cmd := ds.Update(deployDoneMsg{
		failed: []deploy.DeployError{
			{AssetName: "greeting", AssetType: nd.AssetSkill, Err: wrapped},
		},
	})
	ds2 := updated.(*deployScreen)

	if ds2.step != deployConflictConfirm {
		t.Fatalf("step = %d, want deployConflictConfirm (%d)", ds2.step, deployConflictConfirm)
	}
	if cmd == nil {
		t.Fatal("expected conflict form Init cmd")
	}
	if len(ds2.conflictReqs) != 1 || !ds2.conflictReqs[0].ForceReplace {
		t.Fatal("conflictReqs should contain one request with ForceReplace=true")
	}
}

// =============================================================================
// Miscellaneous coverage gaps
// =============================================================================

func TestStatusScreen_FilterEdgeCases(t *testing.T) {
	s := newStatusScreen(newMockServices(), testStyles(), true)
	s.loaded = true
	s.entries = testStatusEntries()
	s.renderedLines = splitLines(s.buildContent())

	// Empty query: all rows shown.
	if got := len(s.filteredEntries()); got != len(s.entries) {
		t.Errorf("empty filter should show all %d entries, got %d", len(s.entries), got)
	}

	// No-match query: 0 rows and a "0/N matching" footer.
	s.filter.text = "zzz-no-such-asset"
	if got := len(s.filteredEntries()); got != 0 {
		t.Errorf("no-match filter should show 0 entries, got %d", got)
	}
	s.rebuildRendered()
	footer := fmt.Sprintf("0/%d matching", len(s.entries))
	if v := s.View(); !strings.Contains(v.Content, footer) {
		t.Errorf("no-match view should show %q footer; got:\n%s", footer, v.Content)
	}

	// Esc clears the filter and deactivates input.
	s.filter.active = true
	s.filter.text = "greeting"
	s.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	if s.filter.text != "" {
		t.Errorf("esc should clear filter text, got %q", s.filter.text)
	}
	if s.InputActive() {
		t.Error("InputActive() should be false after esc clears the filter")
	}
}

func TestBrowseScreen_FilterEdgeCases(t *testing.T) {
	raw := []asset.Asset{
		{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "alpha"}},
		{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "bravo"}},
	}
	idx := asset.NewIndex(raw)
	s := newBrowseScreen(newMockServices(), testStyles(), true)
	s.Update(browseLoadedMsg{assets: idx.All()})

	// Empty query: all rows visible.
	if got := len(s.visibleAssets()); got != 2 {
		t.Errorf("empty filter should show all 2 assets, got %d", got)
	}

	// No-match query: 0 visible and the "No assets match the filter" view.
	s.filter.text = "zzz-nomatch"
	if got := len(s.visibleAssets()); got != 0 {
		t.Errorf("no-match filter should show 0 assets, got %d", got)
	}
	if v := s.View(); !strings.Contains(v.Content, "No assets match the filter") {
		t.Errorf("no-match view should show 'No assets match the filter'; got:\n%s", v.Content)
	}

	// Esc clears the filter and deactivates input.
	s.filter.active = true
	s.filter.text = "alpha"
	s.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	if s.filter.text != "" {
		t.Errorf("esc should clear filter text, got %q", s.filter.text)
	}
	if s.InputActive() {
		t.Error("InputActive() should be false after esc clears the filter")
	}
}

// TestDeploy_ResultScroll_Boundaries drives the deploy result view (a
// RenderScrolledLines consumer) through listScroll's boundary behaviour:
// scroll-up is a no-op at offset 0 and scroll-down clamps at the bottom.
func TestDeploy_ResultScroll_Boundaries(t *testing.T) {
	ds := newTestDeployScreen(deployResult)
	ds.height = 10 // contentHeight = 10 - 4 = 6
	ds.succeeded = nil
	for i := range 20 {
		ds.succeeded = append(ds.succeeded, deploy.DeployResult{
			Deployment: state.Deployment{AssetName: fmt.Sprintf("skill-%02d", i), AssetType: nd.AssetSkill},
		})
	}
	ds.failed = nil
	ds.resultLines = nil

	up := tea.KeyPressMsg(tea.Key{Code: 'k', Text: "k"})
	down := tea.KeyPressMsg(tea.Key{Code: 'j', Text: "j"})

	// At offset 0, scroll-up is a no-op.
	ds.updateResult(up)
	if ds.scroll.offset != 0 {
		t.Fatalf("offset after k at top = %d, want 0 (no-op)", ds.scroll.offset)
	}

	// Scroll down well past the end; offset must clamp to total-page.
	for range 50 {
		ds.updateResult(down)
	}
	maxOffset := len(ds.resultLines) - ds.contentHeight()
	if maxOffset < 0 {
		maxOffset = 0
	}
	if maxOffset == 0 {
		t.Fatal("precondition: result list should exceed one page so the clamp is meaningful")
	}
	if ds.scroll.offset != maxOffset {
		t.Fatalf("offset after many j = %d, want clamped %d", ds.scroll.offset, maxOffset)
	}

	// One more scroll-down at the bottom stays clamped.
	ds.updateResult(down)
	if ds.scroll.offset != maxOffset {
		t.Fatalf("offset should stay clamped at %d, got %d", maxOffset, ds.scroll.offset)
	}
}

// --- Empty-state rendering ---
//
// NothingDeployed() (status/remove) and NoAssets() (browse) empty states are
// already covered by TestStatusScreen_ViewWithEmptyEntries,
// TestRemove_EmptyView_WhenNothingDeployed, and TestBrowseScreen_ViewNoAssets.
// The tests below close the remaining gaps: profile, snapshot, and source.

func TestProfileScreen_EmptyListView_NoProfiles(t *testing.T) {
	s := newProfileScreen(newMockServices(), testStyles(), true)
	s.profiles = nil
	s.step = profileList

	v := s.View()
	if !strings.Contains(v.Content, NoProfiles()) {
		t.Errorf("empty profile list should render NoProfiles(); got:\n%s", v.Content)
	}
}

func TestSourceScreen_EmptyListView_NoSources(t *testing.T) {
	s := newSourceScreen(newMockServices(), testStyles(), true)
	s.sources = nil
	s.step = sourceList

	v := s.View()
	if !strings.Contains(v.Content, NoSources()) {
		t.Errorf("empty source list should render NoSources(); got:\n%s", v.Content)
	}
}

// Note: snapshotScreen.viewList renders a literal "No snapshots saved yet."
// empty-state message rather than the NoSnapshots() helper from empty.go, so
// this test asserts the observable empty-state behaviour (the "No snapshots"
// message) instead of the helper's exact text. Changing the screen to use the
// helper would be a production change, which is out of scope for this chore.
func TestSnapshotScreen_EmptyListView(t *testing.T) {
	s := newSnapshotScreen(newMockServices(), testStyles(), true)
	s.snapshots = nil
	s.step = snapshotList

	v := s.View()
	if !strings.Contains(v.Content, "No snapshots") {
		t.Errorf("empty snapshot list should render an empty-state message; got:\n%s", v.Content)
	}
}
