package tui

import (
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/armstrongl/nd/internal/asset"
	"github.com/armstrongl/nd/internal/deploy"
	"github.com/armstrongl/nd/internal/nd"
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
	if ds.InputActive() {
		t.Fatal("InputActive() at pickType step should be false")
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
	if ds.step != deployPickType {
		t.Fatalf("initial step = %d, want deployPickType (%d)", ds.step, deployPickType)
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

	// Check for success count
	if !containsStr(content, "1 succeeded") {
		t.Errorf("result view missing success count; got:\n%s", content)
	}

	// Check for failure count
	if !containsStr(content, "1 failed") {
		t.Errorf("result view missing failure count; got:\n%s", content)
	}
}

func TestDeploy_ResultView_ShowsErrorDetails(t *testing.T) {
	ds := newTestDeployScreen(deployResult)
	v := ds.View()
	content := v.Content

	// Error details should include asset name and error
	if !containsStr(content, "broken") {
		t.Errorf("result view missing failed asset name; got:\n%s", content)
	}
	if !containsStr(content, "permission denied") {
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

	if !containsStr(content, "2 succeeded") {
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

	if !containsStr(content, "2 failed") {
		t.Errorf("result view missing failure count; got:\n%s", content)
	}
}

func TestDeploy_DeployCmd_AllSucceed(t *testing.T) {
	// Create a mock engine function
	deployer := func(req deploy.DeployRequest) (*deploy.DeployResult, error) {
		return &deploy.DeployResult{
			Deployment: state.Deployment{
				AssetName: req.Asset.Name,
				AssetType: req.Asset.Type,
			},
		}, nil
	}

	reqs := []deploy.DeployRequest{
		{Asset: asset.Asset{Identity: asset.Identity{Name: "a", Type: nd.AssetSkill}}},
		{Asset: asset.Asset{Identity: asset.Identity{Name: "b", Type: nd.AssetRule}}},
	}

	cmd := deployAllCmd(deployer, reqs)
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

func TestDeploy_DeployCmd_SomeFail(t *testing.T) {
	callCount := 0
	deployer := func(req deploy.DeployRequest) (*deploy.DeployResult, error) {
		callCount++
		if callCount == 2 {
			return nil, fmt.Errorf("disk full")
		}
		return &deploy.DeployResult{
			Deployment: state.Deployment{
				AssetName: req.Asset.Name,
				AssetType: req.Asset.Type,
			},
		}, nil
	}

	reqs := []deploy.DeployRequest{
		{Asset: asset.Asset{Identity: asset.Identity{Name: "a", Type: nd.AssetSkill}}},
		{Asset: asset.Asset{Identity: asset.Identity{Name: "b", Type: nd.AssetRule}}},
		{Asset: asset.Asset{Identity: asset.Identity{Name: "c", Type: nd.AssetCommand}}},
	}

	cmd := deployAllCmd(deployer, reqs)
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

func TestDeploy_DeployCmd_Empty(t *testing.T) {
	deployer := func(req deploy.DeployRequest) (*deploy.DeployResult, error) {
		return &deploy.DeployResult{}, nil
	}

	cmd := deployAllCmd(deployer, nil)
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

	if !containsStr(content, "scan failed") {
		t.Errorf("error view should show error message; got:\n%s", content)
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

func TestDeploy_BackFromResult(t *testing.T) {
	ds := newTestDeployScreen(deployResult)

	// Enter key at result step should emit BackMsg
	msg := tea.KeyPressMsg{Code: tea.KeyEnter}
	_, cmd := ds.Update(msg)

	if cmd == nil {
		t.Fatal("expected a command from enter at result step")
	}

	result := cmd()
	if _, ok := result.(BackMsg); !ok {
		t.Fatalf("enter at result produced %T, want BackMsg", result)
	}
}

func TestDeploy_EscFromResult(t *testing.T) {
	ds := newTestDeployScreen(deployResult)

	msg := tea.KeyPressMsg{Code: tea.KeyEscape}
	_, cmd := ds.Update(msg)

	if cmd == nil {
		t.Fatal("expected a command from esc at result step")
	}

	result := cmd()
	if _, ok := result.(BackMsg); !ok {
		t.Fatalf("esc at result produced %T, want BackMsg", result)
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

// containsStr checks if substr is present in s.
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
