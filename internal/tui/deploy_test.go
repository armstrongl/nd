package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"

	"github.com/armstrongl/nd/internal/agent"
	"github.com/armstrongl/nd/internal/asset"
	"github.com/armstrongl/nd/internal/config"
	"github.com/armstrongl/nd/internal/deploy"
	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/sourcemanager"
	"github.com/armstrongl/nd/internal/state"
)

// Compile-time assertion: deployScreen satisfies Screen (and therefore tea.Model).
var _ Screen = (*deployScreen)(nil)

// --- helpers ---

// testStyles is defined in header_test.go (unstyled for deterministic output).

func testAssets() []*asset.Asset {
	return []*asset.Asset{
		{
			Identity:   asset.Identity{SourceID: "local", Type: nd.AssetSkill, Name: "go-test"},
			SourcePath: "/src/skills/go-test",
		},
		{
			Identity:   asset.Identity{SourceID: "local", Type: nd.AssetSkill, Name: "lint"},
			SourcePath: "/src/skills/lint",
		},
		{
			Identity:   asset.Identity{SourceID: "local", Type: nd.AssetRule, Name: "no-magic"},
			SourcePath: "/src/rules/no-magic.md",
		},
	}
}

func testDeployments() []state.Deployment {
	return []state.Deployment{
		{
			SourceID:  "local",
			AssetType: nd.AssetSkill,
			AssetName: "lint",
			LinkPath:  "/home/.config/claude/skills/lint",
		},
	}
}

// newTestDeployScreen creates a deployScreen in a given step with canned data.
func newTestDeployScreen(step deployStep) *deployScreen {
	s := testStyles()
	svc := newMockServices()
	ds := &deployScreen{
		svc:    svc,
		styles: s,
		isDark: true,
		step:   step,
	}
	switch step {
	case deployResult:
		ds.succeeded = []deploy.DeployResult{
			{Deployment: state.Deployment{AssetName: "go-test", AssetType: nd.AssetSkill}},
		}
		ds.failed = []deploy.DeployError{
			{AssetName: "broken", AssetType: nd.AssetRule, Err: fmt.Errorf("permission denied")},
		}
	}
	return ds
}

// --- tests ---

func TestDeploy_NewReturnsNonNil(t *testing.T) {
	svc := newMockServices()
	s := testStyles()
	ds := newDeployScreen(svc, s, true)
	if ds == nil {
		t.Fatal("newDeployScreen returned nil")
	}
}

func TestDeploy_Title(t *testing.T) {
	ds := newTestDeployScreen(deployPickType)
	if got := ds.Title(); got != "Deploy" {
		t.Fatalf("Title() = %q, want %q", got, "Deploy")
	}
}

func TestDeploy_InputActive_PickType(t *testing.T) {
	ds := newTestDeployScreen(deployPickType)
	if !ds.InputActive() {
		t.Fatal("InputActive() at pickType step should be true (form active)")
	}
}

func TestDeploy_InputActive_SelectAssets(t *testing.T) {
	ds := newTestDeployScreen(deploySelectAssets)
	// selectAssets step has a MultiSelect which is considered input-active
	if !ds.InputActive() {
		t.Fatal("InputActive() at selectAssets step should be true")
	}
}

func TestDeploy_InputActive_Running(t *testing.T) {
	ds := newTestDeployScreen(deployRunning)
	if ds.InputActive() {
		t.Fatal("InputActive() at running step should be false")
	}
}

func TestDeploy_InputActive_Result(t *testing.T) {
	ds := newTestDeployScreen(deployResult)
	if ds.InputActive() {
		t.Fatal("InputActive() at result step should be false")
	}
}

func TestDeploy_InitialStep(t *testing.T) {
	svc := newMockServices()
	s := testStyles()
	ds := newDeployScreen(svc, s, true)
	// The scope picker is now the first step, shown before the type picker.
	if ds.step != deployPickScope {
		t.Fatalf("initial step = %d, want deployPickScope (%d)", ds.step, deployPickScope)
	}
}

func TestDeploy_InputActive_PickScope(t *testing.T) {
	ds := newTestDeployScreen(deployPickScope)
	if !ds.InputActive() {
		t.Fatal("InputActive() at pickScope step should be true (form active)")
	}
}

func TestDeploy_ScopeFormNotNil(t *testing.T) {
	svc := newMockServices()
	s := testStyles()
	ds := newDeployScreen(svc, s, true)
	if ds.scopeForm == nil {
		t.Fatal("scopeForm should not be nil after construction")
	}
	// Pre-selection defaults to the current session scope.
	if ds.scopeChoice != string(svc.GetScope()) {
		t.Fatalf("scopeChoice = %q, want %q", ds.scopeChoice, string(svc.GetScope()))
	}
}

func TestDeploy_PickScope_GlobalAdvancesToType(t *testing.T) {
	svc := newMockServices() // GetScope() defaults to global, GetProjectRoot() to ""
	s := testStyles()
	ds := newDeployScreen(svc, s, true)

	ds.scopeChoice = "global"
	ds.scopeForm.State = huh.StateCompleted

	updated, cmd := ds.updatePickScope(nil)
	ds2 := updated.(*deployScreen)

	if ds2.step != deployPickType {
		t.Fatalf("step after selecting global = %d, want deployPickType (%d)", ds2.step, deployPickType)
	}
	if cmd == nil {
		t.Fatal("expected a command (typeForm.Init) after advancing to the type picker")
	}
	if ds2.err != nil {
		t.Fatalf("unexpected error after selecting global: %v", ds2.err)
	}
}

func TestDeploy_PickScope_ProjectNoRootShowsError(t *testing.T) {
	svc := newMockServices() // GetProjectRoot() defaults to "" (no project root)
	s := testStyles()
	ds := newDeployScreen(svc, s, true)

	ds.scopeChoice = "project"
	ds.scopeForm.State = huh.StateCompleted

	updated, _ := ds.updatePickScope(nil)
	ds2 := updated.(*deployScreen)

	if ds2.err == nil {
		t.Fatal("expected an error when selecting project scope with no project root")
	}
	if !strings.Contains(ds2.err.Error(), "no project root detected") {
		t.Errorf("error should mention 'no project root detected'; got: %v", ds2.err)
	}
	if ds2.step == deployPickType {
		t.Error("should not advance to the type picker when project root is missing")
	}
	if len(svc.resetCalls) != 0 {
		t.Errorf("ResetForScope should not be called when project root is missing; got %d calls", len(svc.resetCalls))
	}
}

func TestDeploy_EscOnPickScope_SendsBackMsg(t *testing.T) {
	svc := newMockServices()
	s := testStyles()
	ds := newDeployScreen(svc, s, true)
	_, cmd := ds.updatePickScope(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("expected BackMsg cmd on ESC at pickScope, got nil")
	}
	if _, ok := cmd().(BackMsg); !ok {
		t.Fatalf("expected BackMsg, got %T", cmd())
	}
}

func TestDeploy_FullHelpItems_PickScope(t *testing.T) {
	ds := newTestDeployScreen(deployPickScope)
	items := ds.FullHelpItems()

	hasEnterSelect := false
	for _, item := range items {
		if item.Key == "enter" && item.Desc == "select" {
			hasEnterSelect = true
		}
	}
	if !hasEnterSelect {
		t.Errorf("FullHelpItems at pickScope should include 'enter select'; got: %v", items)
	}
}

func TestDeploy_TypeFormNotNil(t *testing.T) {
	svc := newMockServices()
	s := testStyles()
	ds := newDeployScreen(svc, s, true)
	if ds.typeForm == nil {
		t.Fatal("typeForm should not be nil after construction")
	}
}

func TestDeploy_DeployDoneMsg_TransitionsToResult(t *testing.T) {
	ds := newTestDeployScreen(deployRunning)

	msg := deployDoneMsg{
		succeeded: []deploy.DeployResult{
			{Deployment: state.Deployment{AssetName: "go-test", AssetType: nd.AssetSkill}},
		},
		failed: []deploy.DeployError{
			{AssetName: "broken", AssetType: nd.AssetRule, Err: fmt.Errorf("fail")},
		},
	}

	updated, _ := ds.Update(msg)
	result := updated.(*deployScreen)

	if result.step != deployResult {
		t.Fatalf("step after deployDoneMsg = %d, want deployResult (%d)", result.step, deployResult)
	}
	if len(result.succeeded) != 1 {
		t.Fatalf("succeeded count = %d, want 1", len(result.succeeded))
	}
	if len(result.failed) != 1 {
		t.Fatalf("failed count = %d, want 1", len(result.failed))
	}
}

func TestDeploy_ResultView_ShowsCounts(t *testing.T) {
	ds := newTestDeployScreen(deployResult)
	v := ds.View()
	content := v.Content

	if content == "" {
		t.Fatal("View() at result step returned empty content")
	}

	if !strings.Contains(content, "1 succeeded") {
		t.Errorf("result view missing success count; got:\n%s", content)
	}

	if !strings.Contains(content, "1 failed") {
		t.Errorf("result view missing failure count; got:\n%s", content)
	}
}

func TestDeploy_ResultView_ShowsErrorDetails(t *testing.T) {
	ds := newTestDeployScreen(deployResult)
	v := ds.View()
	content := v.Content

	if !strings.Contains(content, "broken") {
		t.Errorf("result view missing failed asset name; got:\n%s", content)
	}
	if !strings.Contains(content, "permission denied") {
		t.Errorf("result view missing error message; got:\n%s", content)
	}
}

func TestDeploy_ResultView_AllSucceeded(t *testing.T) {
	ds := newTestDeployScreen(deployResult)
	ds.succeeded = []deploy.DeployResult{
		{Deployment: state.Deployment{AssetName: "go-test", AssetType: nd.AssetSkill}},
		{Deployment: state.Deployment{AssetName: "no-magic", AssetType: nd.AssetRule}},
	}
	ds.failed = nil

	v := ds.View()
	content := v.Content

	if !strings.Contains(content, "2 succeeded") {
		t.Errorf("result view missing success count; got:\n%s", content)
	}
}

func TestDeploy_ResultView_AllFailed(t *testing.T) {
	ds := newTestDeployScreen(deployResult)
	ds.succeeded = nil
	ds.failed = []deploy.DeployError{
		{AssetName: "a", AssetType: nd.AssetSkill, Err: fmt.Errorf("err1")},
		{AssetName: "b", AssetType: nd.AssetRule, Err: fmt.Errorf("err2")},
	}

	v := ds.View()
	content := v.Content

	if !strings.Contains(content, "2 failed") {
		t.Errorf("result view missing failure count; got:\n%s", content)
	}
}

// M12: Summary line now shows "X of Y succeeded" instead of "Y of Y"
func TestDeploy_ResultView_SummaryShowsSucceededCount(t *testing.T) {
	ds := newTestDeployScreen(deployResult)
	ds.succeeded = []deploy.DeployResult{
		{Deployment: state.Deployment{AssetName: "a", AssetType: nd.AssetSkill}},
	}
	ds.failed = []deploy.DeployError{
		{AssetName: "b", AssetType: nd.AssetRule, Err: fmt.Errorf("err")},
	}

	v := ds.View()
	if !strings.Contains(v.Content, "1 of 2 succeeded") {
		t.Errorf("summary should show '1 of 2 succeeded'; got:\n%s", v.Content)
	}
}

func TestDeploy_DeployBulkCmd_AllSucceed(t *testing.T) {
	deployer := func(reqs []deploy.DeployRequest) (*deploy.BulkDeployResult, error) {
		var result deploy.BulkDeployResult
		for _, req := range reqs {
			result.Succeeded = append(result.Succeeded, deploy.DeployResult{
				Deployment: state.Deployment{
					AssetName: req.Asset.Name,
					AssetType: req.Asset.Type,
				},
			})
		}
		return &result, nil
	}

	reqs := []deploy.DeployRequest{
		{Asset: asset.Asset{Identity: asset.Identity{Name: "a", Type: nd.AssetSkill}}},
		{Asset: asset.Asset{Identity: asset.Identity{Name: "b", Type: nd.AssetRule}}},
	}

	cmd := deployBulkCmd(deployer, reqs, "claude-code")
	msg := cmd()

	done, ok := msg.(deployDoneMsg)
	if !ok {
		t.Fatalf("cmd returned %T, want deployDoneMsg", msg)
	}
	if len(done.succeeded) != 2 {
		t.Fatalf("succeeded = %d, want 2", len(done.succeeded))
	}
	if len(done.failed) != 0 {
		t.Fatalf("failed = %d, want 0", len(done.failed))
	}
}

func TestDeploy_DeployBulkCmd_PartialFailure(t *testing.T) {
	deployer := func(reqs []deploy.DeployRequest) (*deploy.BulkDeployResult, error) {
		var result deploy.BulkDeployResult
		for _, req := range reqs {
			if req.Asset.Name == "b" {
				result.Failed = append(result.Failed, deploy.DeployError{
					AssetName: req.Asset.Name,
					AssetType: req.Asset.Type,
					Err:       fmt.Errorf("disk full"),
				})
			} else {
				result.Succeeded = append(result.Succeeded, deploy.DeployResult{
					Deployment: state.Deployment{
						AssetName: req.Asset.Name,
						AssetType: req.Asset.Type,
					},
				})
			}
		}
		return &result, nil
	}

	reqs := []deploy.DeployRequest{
		{Asset: asset.Asset{Identity: asset.Identity{Name: "a", Type: nd.AssetSkill}}},
		{Asset: asset.Asset{Identity: asset.Identity{Name: "b", Type: nd.AssetRule}}},
		{Asset: asset.Asset{Identity: asset.Identity{Name: "c", Type: nd.AssetCommand}}},
	}

	cmd := deployBulkCmd(deployer, reqs, "claude-code")
	msg := cmd()

	done := msg.(deployDoneMsg)
	if len(done.succeeded) != 2 {
		t.Fatalf("succeeded = %d, want 2", len(done.succeeded))
	}
	if len(done.failed) != 1 {
		t.Fatalf("failed = %d, want 1", len(done.failed))
	}
	if done.failed[0].AssetName != "b" {
		t.Fatalf("failed asset = %q, want %q", done.failed[0].AssetName, "b")
	}
}

func TestDeploy_DeployBulkCmd_TotalFailure(t *testing.T) {
	deployer := func(reqs []deploy.DeployRequest) (*deploy.BulkDeployResult, error) {
		return nil, fmt.Errorf("lock acquisition failed")
	}

	reqs := []deploy.DeployRequest{
		{Asset: asset.Asset{Identity: asset.Identity{Name: "a", Type: nd.AssetSkill}}},
		{Asset: asset.Asset{Identity: asset.Identity{Name: "b", Type: nd.AssetRule}}},
	}

	cmd := deployBulkCmd(deployer, reqs, "claude-code")
	msg := cmd()

	done := msg.(deployDoneMsg)
	if len(done.succeeded) != 0 {
		t.Fatalf("succeeded = %d, want 0", len(done.succeeded))
	}
	if len(done.failed) != 2 {
		t.Fatalf("failed = %d, want 2", len(done.failed))
	}
	// Total-failure errors are tagged with the target agent name.
	if done.failed[0].Agent != "claude-code" {
		t.Fatalf("failed[0].Agent = %q, want %q", done.failed[0].Agent, "claude-code")
	}
}

func TestDeploy_DeployBulkCmd_Empty(t *testing.T) {
	deployer := func(reqs []deploy.DeployRequest) (*deploy.BulkDeployResult, error) {
		return &deploy.BulkDeployResult{}, nil
	}

	cmd := deployBulkCmd(deployer, nil, "claude-code")
	msg := cmd()

	done := msg.(deployDoneMsg)
	if len(done.succeeded) != 0 || len(done.failed) != 0 {
		t.Fatalf("expected empty results for empty requests")
	}
}

func TestDeploy_ErrorView(t *testing.T) {
	ds := newTestDeployScreen(deployPickType)
	ds.err = fmt.Errorf("scan failed: source unavailable")

	v := ds.View()
	content := v.Content

	if !strings.Contains(content, "scan failed") {
		t.Errorf("error view should show error message; got:\n%s", content)
	}
	// M6: error view now includes hint to press esc
	if !strings.Contains(content, "esc") {
		t.Errorf("error view should hint to press esc; got:\n%s", content)
	}
}

// L7: "All deployed" info message should not show as error
func TestDeploy_InfoView_AllDeployed(t *testing.T) {
	ds := newTestDeployScreen(deployResult)
	ds.succeeded = nil
	ds.failed = nil
	ds.info = AllDeployed("skills")

	v := ds.View()
	content := v.Content

	if strings.Contains(content, "Error") {
		t.Errorf("info view should not contain 'Error'; got:\n%s", content)
	}
	if !strings.Contains(content, "already deployed") {
		t.Errorf("info view should contain 'already deployed'; got:\n%s", content)
	}
}

// startDeploy with nothing selected should land on the result step and render a
// visible "No assets selected." notice rather than silently bouncing back.
func TestDeploy_StartDeploy_EmptySelection_ShowsInfo(t *testing.T) {
	ds := newTestDeployScreen(deploySelectAssets)
	ds.selected = nil // nothing selected

	cmd := ds.startDeploy()
	if cmd != nil {
		t.Fatalf("empty-selection startDeploy should not emit a BackMsg cmd, got non-nil cmd")
	}
	if ds.step != deployResult {
		t.Fatalf("step = %d, want deployResult (%d)", ds.step, deployResult)
	}
	if ds.info == "" {
		t.Fatal("info should be set when no assets are selected")
	}
	v := ds.View()
	if !strings.Contains(v.Content, "No assets selected") {
		t.Fatalf("view should contain 'No assets selected'; got:\n%s", v.Content)
	}
}

func TestDeploy_FilterUndeployed(t *testing.T) {
	all := testAssets()
	deployed := testDeployments()

	result := filterUndeployed(all, deployed)

	// "lint" is already deployed, so only "go-test" and "no-magic" should remain
	if len(result) != 2 {
		t.Fatalf("filterUndeployed returned %d assets, want 2", len(result))
	}

	names := map[string]bool{}
	for _, a := range result {
		names[a.Name] = true
	}
	if !names["go-test"] {
		t.Error("expected go-test in undeployed list")
	}
	if !names["no-magic"] {
		t.Error("expected no-magic in undeployed list")
	}
	if names["lint"] {
		t.Error("lint should not be in undeployed list (already deployed)")
	}
}

func TestDeploy_FilterUndeployed_AllDeployed(t *testing.T) {
	all := []*asset.Asset{
		{Identity: asset.Identity{SourceID: "local", Type: nd.AssetSkill, Name: "lint"}, SourcePath: "/src/skills/lint"},
	}
	deployed := testDeployments()

	result := filterUndeployed(all, deployed)
	if len(result) != 0 {
		t.Fatalf("filterUndeployed returned %d assets, want 0", len(result))
	}
}

func TestDeploy_FilterUndeployed_NoneDeployed(t *testing.T) {
	all := testAssets()
	result := filterUndeployed(all, nil)
	if len(result) != len(all) {
		t.Fatalf("filterUndeployed returned %d assets, want %d", len(result), len(all))
	}
}

// Regression: j/k scroll must work before the first View() call.
// resultLines was only lazily populated in viewResult(), so pressing j before
// the first render left resultLines empty and ScrollDown became a no-op.
func TestDeploy_ScrollBeforeFirstRender(t *testing.T) {
	ds := newTestDeployScreen(deployResult)
	// Do NOT call ds.View() — simulate user pressing j as the very first action.
	if len(ds.resultLines) != 0 {
		t.Fatal("precondition: resultLines should be empty before any render")
	}

	msg := tea.KeyPressMsg(tea.Key{Code: 'j', Text: "j"})
	updated, _ := ds.Update(msg)
	ds2 := updated.(*deployScreen)

	if len(ds2.resultLines) == 0 {
		t.Fatal("resultLines should be populated after j is pressed, even without a prior View() call")
	}
}

// M7: Enter at result emits PopToRootMsg (not BackMsg)
func TestDeploy_EnterFromResult(t *testing.T) {
	ds := newTestDeployScreen(deployResult)

	msg := tea.KeyPressMsg{Code: tea.KeyEnter}
	_, cmd := ds.Update(msg)

	if cmd == nil {
		t.Fatal("expected a command from enter at result step")
	}

	// tea.Batch returns a function that yields multiple messages.
	// We just verify the command is non-nil (the batch itself).
}

// H4: esc at result step is handled by root model (not by deployScreen)
// so we verify that updateResult does NOT handle esc.
func TestDeploy_EscAtResult_NoCmd(t *testing.T) {
	ds := newTestDeployScreen(deployResult)

	msg := tea.KeyPressMsg{Code: tea.KeyEscape}
	_, cmd := ds.Update(msg)

	if cmd != nil {
		t.Fatal("esc at result should not produce a command (handled by root)")
	}
}

func TestDeploy_AssetKey(t *testing.T) {
	a := &asset.Asset{
		Identity: asset.Identity{SourceID: "local", Type: nd.AssetSkill, Name: "go-test"},
	}
	key := assetKey(a)
	want := "local:skills/go-test"
	if key != want {
		t.Fatalf("assetKey = %q, want %q", key, want)
	}
}

func TestDeploy_DeploymentKey(t *testing.T) {
	d := state.Deployment{
		SourceID:  "local",
		AssetType: nd.AssetSkill,
		AssetName: "go-test",
	}
	key := deploymentKey(d)
	want := "local:skills/go-test"
	if key != want {
		t.Fatalf("deploymentKey = %q, want %q", key, want)
	}
}

func TestDeploy_TypeDisplayNames(t *testing.T) {
	names := typeDisplayNames()
	if len(names) == 0 {
		t.Fatal("typeDisplayNames returned empty slice")
	}
	// First entry should be "All types"
	if names[0].label != "All types" {
		t.Fatalf("first type display = %q, want %q", names[0].label, "All types")
	}
}

// H1: double-fire guard — scanning flag prevents repeated startScan calls
func TestDeploy_DoubleFireGuard_PickType(t *testing.T) {
	ds := newTestDeployScreen(deployPickType)
	ds.scanning = true

	// Any message should be a no-op when scanning is true
	_, cmd := ds.updatePickType(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("updatePickType should return nil cmd when scanning guard is set")
	}
}

// H1: double-fire guard — deploying flag prevents repeated startDeploy calls
func TestDeploy_DoubleFireGuard_SelectAssets(t *testing.T) {
	ds := newTestDeployScreen(deploySelectAssets)
	ds.deploying = true

	_, cmd := ds.updateSelectAssets(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("updateSelectAssets should return nil cmd when deploying guard is set")
	}
}

// M6: Scan error transitions to deployResult (not stuck at deployPickType)
func TestDeploy_ScanError_TransitionsToResult(t *testing.T) {
	ds := newTestDeployScreen(deployPickType)

	msg := scanDoneMsg{err: fmt.Errorf("scan error")}
	updated, _ := ds.Update(msg)
	result := updated.(*deployScreen)

	if result.step != deployResult {
		t.Fatalf("step after scan error = %d, want deployResult (%d)", result.step, deployResult)
	}
	if result.err == nil {
		t.Fatal("err should be set after scan error")
	}
}

// M6: Empty scan (all deployed) transitions to deployResult
func TestDeploy_ScanEmpty_TransitionsToResult(t *testing.T) {
	ds := newTestDeployScreen(deployPickType)

	msg := scanDoneMsg{assets: nil}
	updated, _ := ds.Update(msg)
	result := updated.(*deployScreen)

	if result.step != deployResult {
		t.Fatalf("step after empty scan = %d, want deployResult (%d)", result.step, deployResult)
	}
	if result.info == "" {
		t.Fatal("info should be set when all assets are deployed")
	}
}

// H2: dry-run mode shows preview instead of executing
func TestDeploy_DryRunView(t *testing.T) {
	ds := newTestDeployScreen(deployResult)
	ds.succeeded = nil
	ds.failed = nil
	ds.dryRun = true
	ds.dryReqs = []deploy.DeployRequest{
		{Asset: asset.Asset{Identity: asset.Identity{SourceID: "local", Type: nd.AssetSkill, Name: "go-test"}}},
	}

	v := ds.View()
	content := v.Content

	if !strings.Contains(content, "DRY RUN") {
		t.Errorf("dry-run view should contain 'DRY RUN'; got:\n%s", content)
	}
	if !strings.Contains(content, "go-test") {
		t.Errorf("dry-run view should list assets; got:\n%s", content)
	}
}

// taskmd k63tsg: a project-scope deploy must thread the resolved project root
// (from the resolving GetProjectRoot accessor) into every DeployRequest, even
// when the TUI was launched in the default global scope inside a project.
func TestDeploy_StartDeploy_ProjectScopeThreadsProjectRoot(t *testing.T) {
	const resolvedRoot = "/resolved/project"
	svc := newMockServices()
	svc.getScopeFn = func() nd.Scope { return nd.ScopeProject }
	svc.getProjectRootFn = func() string { return resolvedRoot }
	svc.isDryRunFn = func() bool { return true } // stop after building reqs

	assets := testAssets()
	ds := &deployScreen{
		svc:      svc,
		styles:   testStyles(),
		isDark:   true,
		step:     deploySelectAssets,
		assets:   assets,
		selected: []string{assetKey(assets[0]), assetKey(assets[2])},
	}

	ds.startDeploy()

	if len(ds.reqs) != 2 {
		t.Fatalf("startDeploy built %d requests, want 2", len(ds.reqs))
	}
	for i, r := range ds.reqs {
		if r.Scope != nd.ScopeProject {
			t.Errorf("reqs[%d].Scope = %q, want project", i, r.Scope)
		}
		if r.ProjectRoot != resolvedRoot {
			t.Errorf("reqs[%d].ProjectRoot = %q, want %q", i, r.ProjectRoot, resolvedRoot)
		}
	}
}

// Compile-time assertion: deployScreen satisfies FullHelpProvider.
var _ FullHelpProvider = (*deployScreen)(nil)

func TestDeploy_FullHelpItems_PickType(t *testing.T) {
	ds := newTestDeployScreen(deployPickType)
	items := ds.FullHelpItems()

	hasEnterSelect := false
	for _, item := range items {
		if item.Key == "enter" && item.Desc == "select" {
			hasEnterSelect = true
		}
	}
	if !hasEnterSelect {
		t.Errorf("FullHelpItems at pickType should include 'enter select'; got: %v", items)
	}
}

func TestDeploy_FullHelpItems_SelectAssets(t *testing.T) {
	ds := newTestDeployScreen(deploySelectAssets)
	items := ds.FullHelpItems()

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

func TestDeploy_FullHelpItems_Result(t *testing.T) {
	ds := newTestDeployScreen(deployResult)
	items := ds.FullHelpItems()

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

func TestDeploy_EscOnPickType_SendsBackMsg(t *testing.T) {
	ds := newTestDeployScreen(deployPickType)
	_, cmd := ds.updatePickType(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("expected BackMsg cmd on ESC at pickType, got nil")
	}
	if _, ok := cmd().(BackMsg); !ok {
		t.Fatalf("expected BackMsg, got %T", cmd())
	}
}

func TestDeploy_EscOnSelectAssets_SendsBackMsg(t *testing.T) {
	ds := newTestDeployScreen(deploySelectAssets)
	_, cmd := ds.updateSelectAssets(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("expected BackMsg cmd on ESC at selectAssets, got nil")
	}
	if _, ok := cmd().(BackMsg); !ok {
		t.Fatalf("expected BackMsg, got %T", cmd())
	}
}

// --- Conflict resolution tests ---

func makeConflictError(assetName, path string) *nd.ConflictError {
	return &nd.ConflictError{
		TargetPath:   path,
		ExistingKind: nd.FileKindForeignSymlink,
		AssetName:    assetName,
	}
}

func TestDeploy_ConflictFails_MovesToConflictConfirm(t *testing.T) {
	ds := newTestDeployScreen(deployRunning)
	// Populate original requests so buildForceRequests can look them up.
	ds.reqs = []deploy.DeployRequest{
		{Asset: asset.Asset{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "greeting"}}},
	}

	msg := deployDoneMsg{
		failed: []deploy.DeployError{
			{AssetName: "greeting", AssetType: nd.AssetSkill, Err: makeConflictError("greeting", "/p")},
		},
	}
	updated, cmd := ds.Update(msg)
	ds2 := updated.(*deployScreen)

	if ds2.step != deployConflictConfirm {
		t.Fatalf("step = %d, want deployConflictConfirm (%d)", ds2.step, deployConflictConfirm)
	}
	if cmd == nil {
		t.Fatal("expected Init cmd for conflict form")
	}
	if ds2.conflictForm == nil {
		t.Fatal("conflictForm should be set")
	}
	if len(ds2.conflictReqs) != 1 || !ds2.conflictReqs[0].ForceReplace {
		t.Fatal("conflictReqs should contain one request with ForceReplace=true")
	}
}

func TestDeploy_ConflictConfirm_ViewShowsAssetNames(t *testing.T) {
	ds := newTestDeployScreen(deployRunning)
	ds.step = deployConflictConfirm
	ds.conflictFails = []deploy.DeployError{
		{AssetName: "greeting", AssetType: nd.AssetSkill, Err: makeConflictError("greeting", "/p")},
	}
	ds.buildConflictForm()

	content := ds.viewConflictConfirm().Content
	if !strings.Contains(content, "greeting") {
		t.Errorf("viewConflictConfirm() should mention 'greeting', got:\n%s", content)
	}
	if !strings.Contains(content, "1 asset") {
		t.Errorf("viewConflictConfirm() should mention asset count, got:\n%s", content)
	}
}

func TestDeploy_ConflictCancel_MovesToResultWithAllFailures(t *testing.T) {
	ds := newTestDeployScreen(deployRunning)
	ds.firstSucceeded = []deploy.DeployResult{
		{Deployment: state.Deployment{AssetName: "ok-skill", AssetType: nd.AssetSkill}},
	}
	ds.firstFailed = []deploy.DeployError{
		{AssetName: "perm-fail", AssetType: nd.AssetRule, Err: fmt.Errorf("permission denied")},
	}
	ds.conflictFails = []deploy.DeployError{
		{AssetName: "greeting", AssetType: nd.AssetSkill, Err: makeConflictError("greeting", "/p")},
	}
	ds.conflictReqs = []deploy.DeployRequest{{ForceReplace: true}}

	updated, _ := ds.cancelConflictResolution()
	ds2 := updated.(*deployScreen)

	if ds2.step != deployResult {
		t.Fatalf("step = %d, want deployResult", ds2.step)
	}
	if len(ds2.succeeded) != 1 || ds2.succeeded[0].Deployment.AssetName != "ok-skill" {
		t.Errorf("succeeded should contain the first-run success, got %v", ds2.succeeded)
	}
	// Both the non-conflict failure and the conflict failure should appear.
	if len(ds2.failed) != 2 {
		t.Fatalf("failed = %d, want 2 (perm-fail + greeting)", len(ds2.failed))
	}
}

func TestDeploy_SecondPass_MergesResults(t *testing.T) {
	ds := newTestDeployScreen(deployRunning)
	// Simulate state after first pass + user confirmed.
	ds.conflictReqs = []deploy.DeployRequest{{ForceReplace: true}} // non-nil = second pass
	ds.firstSucceeded = []deploy.DeployResult{
		{Deployment: state.Deployment{AssetName: "first-ok", AssetType: nd.AssetSkill}},
	}
	ds.firstFailed = nil

	msg := deployDoneMsg{
		succeeded: []deploy.DeployResult{
			{Deployment: state.Deployment{AssetName: "greeting", AssetType: nd.AssetSkill}},
		},
	}
	updated, _ := ds.Update(msg)
	ds2 := updated.(*deployScreen)

	if ds2.step != deployResult {
		t.Fatalf("step = %d, want deployResult", ds2.step)
	}
	if len(ds2.succeeded) != 2 {
		t.Fatalf("merged succeeded = %d, want 2", len(ds2.succeeded))
	}
}

func TestDeploy_ResultView_NoManualRemoveHint(t *testing.T) {
	// Conflicts that reach the result view (after cancelling resolution) should not
	// show the old "remove manually" hint — it was replaced by interactive resolution.
	ds := newTestDeployScreen(deployResult)
	ds.failed = []deploy.DeployError{
		{AssetName: "greeting", AssetType: nd.AssetSkill, Err: makeConflictError("greeting", "/p")},
	}

	content := ds.viewResult().Content
	if strings.Contains(content, "manually") {
		t.Errorf("result view should not tell user to remove manually; got:\n%s", content)
	}
	if !strings.Contains(content, "greeting") {
		t.Errorf("result view should still mention the failed asset; got:\n%s", content)
	}
}

func TestDeploy_RunningViewShowsAssetCount(t *testing.T) {
	ds := newTestDeployScreen(deployRunning)
	ds.reqs = []deploy.DeployRequest{
		{Asset: asset.Asset{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "a"}}},
		{Asset: asset.Asset{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "b"}}},
		{Asset: asset.Asset{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "c"}}},
	}

	v := ds.View()
	if !strings.Contains(v.Content, "3 asset(s)") {
		t.Errorf("running view should show asset count, got: %q", v.Content)
	}
}

// --- Multi-agent selection tests ---

// registryWithDetected builds a registry where the named agents are "detected"
// (their binary is found on PATH). Binary verification is skipped.
func registryWithDetected(detected map[string]bool) *agent.Registry {
	reg := agent.New(config.Config{})
	reg.SetRunCommand(nil) // skip binary verification
	reg.SetLookPath(func(bin string) (string, error) {
		switch bin {
		case "claude":
			if detected["claude-code"] {
				return "/usr/bin/claude", nil
			}
		case "copilot":
			if detected["copilot"] {
				return "/usr/bin/copilot", nil
			}
		}
		return "", exec.ErrNotFound
	})
	reg.SetStat(func(string) (os.FileInfo, error) { return nil, os.ErrNotExist })
	return reg
}

// testSourceManagerWithDeployAgents builds a real SourceManager whose config
// sets default_deploy_agents.
func testSourceManagerWithDeployAgents(t *testing.T, agentNames []string) *sourcemanager.SourceManager {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	lines := []string{
		"version: 1",
		"default_scope: global",
		"default_agent: claude-code",
		"symlink_strategy: absolute",
		"default_deploy_agents:",
	}
	for _, a := range agentNames {
		lines = append(lines, "  - "+a)
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	sm, err := sourcemanager.New(path, "")
	if err != nil {
		t.Fatalf("new source manager: %v", err)
	}
	return sm
}

func TestDeploy_DetectedAgents_OnlyDetected(t *testing.T) {
	ds := newTestDeployScreen(deployPickType)
	svc := ds.svc.(*mockServices)
	svc.agentRegistryFn = func() (*agent.Registry, error) {
		return registryWithDetected(map[string]bool{"claude-code": true, "copilot": false}), nil
	}

	got := ds.detectedAgents()
	if len(got) != 1 || got[0].Name != "claude-code" {
		t.Fatalf("detectedAgents() = %v, want only claude-code", got)
	}
}

func TestDeploy_AfterPickType_TwoDetectedShowsPicker(t *testing.T) {
	ds := newTestDeployScreen(deployPickType)
	svc := ds.svc.(*mockServices)
	svc.agentRegistryFn = func() (*agent.Registry, error) {
		return registryWithDetected(map[string]bool{"claude-code": true, "copilot": true}), nil
	}

	updated, cmd := ds.afterPickType()
	ds2 := updated.(*deployScreen)
	if ds2.step != deployPickAgents {
		t.Fatalf("step = %d, want deployPickAgents (%d)", ds2.step, deployPickAgents)
	}
	if ds2.agentForm == nil {
		t.Fatal("agentForm should be built when 2+ agents are detected")
	}
	if cmd == nil {
		t.Fatal("expected Init cmd for agent form")
	}
}

func TestDeploy_AfterPickType_OneDetectedSkipsPicker(t *testing.T) {
	ds := newTestDeployScreen(deployPickType)
	svc := ds.svc.(*mockServices)
	svc.agentRegistryFn = func() (*agent.Registry, error) {
		return registryWithDetected(map[string]bool{"claude-code": true}), nil
	}

	updated, _ := ds.afterPickType()
	ds2 := updated.(*deployScreen)
	if ds2.step == deployPickAgents {
		t.Fatal("picker should be skipped when exactly one agent is detected")
	}
	if len(ds2.selectedAgents) != 1 || ds2.selectedAgents[0] != "claude-code" {
		t.Fatalf("selectedAgents = %v, want [claude-code]", ds2.selectedAgents)
	}
}

func TestDeploy_AfterPickType_ConfigSkipsPicker(t *testing.T) {
	ds := newTestDeployScreen(deployPickType)
	svc := ds.svc.(*mockServices)
	svc.sourceManagerFn = func() (*sourcemanager.SourceManager, error) {
		return testSourceManagerWithDeployAgents(t, []string{"claude-code", "copilot"}), nil
	}
	// Even with 2 agents detected, config default_deploy_agents takes precedence.
	svc.agentRegistryFn = func() (*agent.Registry, error) {
		return registryWithDetected(map[string]bool{"claude-code": true, "copilot": true}), nil
	}

	updated, _ := ds.afterPickType()
	ds2 := updated.(*deployScreen)
	if ds2.step == deployPickAgents {
		t.Fatal("picker should be skipped when default_deploy_agents is configured")
	}
	if len(ds2.selectedAgents) != 2 {
		t.Fatalf("selectedAgents = %v, want the 2 configured agents", ds2.selectedAgents)
	}
}

func TestDeploy_FirstPassBatches_SingleAgent(t *testing.T) {
	ds := newTestDeployScreen(deployPickType)
	svc := ds.svc.(*mockServices)
	svc.agentRegistryFn = func() (*agent.Registry, error) {
		return registryWithDetected(map[string]bool{"claude-code": true}), nil
	}
	svc.deployEngineForFn = func(ag *agent.Agent) (*deploy.Engine, error) {
		return deploy.New(nil, ag, ""), nil
	}
	ds.selectedAgents = []string{"claude-code"}

	base := []deploy.DeployRequest{
		{Asset: asset.Asset{Identity: asset.Identity{SourceID: "local", Type: nd.AssetSkill, Name: "greeting"}}},
	}
	batches, err := ds.firstPassBatches(base)
	if err != nil {
		t.Fatalf("firstPassBatches: %v", err)
	}
	if len(batches) != 1 {
		t.Fatalf("batches = %d, want 1", len(batches))
	}
	if batches[0].name != "claude-code" || len(batches[0].reqs) != 1 {
		t.Fatalf("batch = {name:%q, reqs:%d}, want {claude-code, 1}", batches[0].name, len(batches[0].reqs))
	}
}

func TestDeploy_FirstPassBatches_TwoAgents(t *testing.T) {
	ds := newTestDeployScreen(deployPickType)
	svc := ds.svc.(*mockServices)
	svc.agentRegistryFn = func() (*agent.Registry, error) {
		return registryWithDetected(map[string]bool{"claude-code": true, "copilot": true}), nil
	}
	svc.deployEngineForFn = func(ag *agent.Agent) (*deploy.Engine, error) {
		return deploy.New(nil, ag, ""), nil
	}
	ds.selectedAgents = []string{"claude-code", "copilot"}

	base := []deploy.DeployRequest{
		{Asset: asset.Asset{Identity: asset.Identity{SourceID: "local", Type: nd.AssetSkill, Name: "greeting"}}},
	}
	batches, err := ds.firstPassBatches(base)
	if err != nil {
		t.Fatalf("firstPassBatches: %v", err)
	}
	if len(batches) != 2 {
		t.Fatalf("batches = %d, want 2 (one per agent)", len(batches))
	}
	if batches[0].name != "claude-code" || batches[1].name != "copilot" {
		t.Fatalf("batch names = %q,%q, want claude-code,copilot", batches[0].name, batches[1].name)
	}
	for _, b := range batches {
		if len(b.reqs) != 1 {
			t.Errorf("agent %q got %d reqs, want 1 (same asset once per agent)", b.name, len(b.reqs))
		}
	}
}

func TestDeploy_RunBulk_TagsTotalFailureAgent(t *testing.T) {
	deployer := func([]deploy.DeployRequest) (*deploy.BulkDeployResult, error) {
		return nil, fmt.Errorf("lock failed")
	}
	reqs := []deploy.DeployRequest{
		{Asset: asset.Asset{Identity: asset.Identity{Name: "a", Type: nd.AssetSkill}}},
	}
	_, failed := runBulk(deployer, reqs, "copilot")
	if len(failed) != 1 || failed[0].Agent != "copilot" {
		t.Fatalf("runBulk failed = %+v, want 1 tagged copilot", failed)
	}
}

func TestDeploy_ResultView_ShowsTargetAgent(t *testing.T) {
	ds := newTestDeployScreen(deployResult)
	ds.succeeded = []deploy.DeployResult{
		{Deployment: state.Deployment{AssetName: "greeting", AssetType: nd.AssetSkill, Agent: "copilot"}},
	}
	ds.failed = nil

	content := ds.View().Content
	if !strings.Contains(content, "copilot") {
		t.Errorf("result view should label the target agent; got:\n%s", content)
	}
	if !strings.Contains(content, GlyphArrow) {
		t.Errorf("result view should show the arrow to the agent; got:\n%s", content)
	}
}
