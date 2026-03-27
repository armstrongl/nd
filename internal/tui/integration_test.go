package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/armstrongl/nd/internal/asset"
	"github.com/armstrongl/nd/internal/deploy"
	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/state"
)

// =============================================================================
// Task 7.1 Step 1: Full deploy flow
// =============================================================================

func TestIntegration_DeployFlow_MenuToResultToRoot(t *testing.T) {
	m := newTestModel()

	// 1. Navigate to deploy screen.
	ds := newDeployScreen(m.svc, m.styles, m.isDark)
	updated, _ := m.Update(NavigateMsg{Screen: ds})
	m = updated.(Model)

	if len(m.screens) != 2 {
		t.Fatalf("expected 2 screens after NavigateMsg, got %d", len(m.screens))
	}
	if m.screens[1].Title() != "Deploy" {
		t.Fatalf("expected Deploy screen, got %q", m.screens[1].Title())
	}

	// 2. Simulate scan completing with assets.
	updated, _ = m.Update(scanDoneMsg{
		assets: []*asset.Asset{
			{Identity: asset.Identity{SourceID: "local", Type: nd.AssetSkill, Name: "go-test"}, SourcePath: "/src/skills/go-test"},
			{Identity: asset.Identity{SourceID: "local", Type: nd.AssetRule, Name: "no-magic"}, SourcePath: "/src/rules/no-magic.md"},
		},
	})
	m = updated.(Model)

	// Deploy screen should now be on selectAssets step.
	dscreen := m.screens[1].(*deployScreen)
	if dscreen.step != deploySelectAssets {
		t.Fatalf("expected deploySelectAssets step, got %d", dscreen.step)
	}

	// 3. Simulate deploy completing.
	updated, cmd := m.Update(deployDoneMsg{
		succeeded: []deploy.DeployResult{
			{Deployment: state.Deployment{AssetName: "go-test", AssetType: nd.AssetSkill}},
			{Deployment: state.Deployment{AssetName: "no-magic", AssetType: nd.AssetRule}},
		},
	})
	m = updated.(Model)

	// Should be on result step.
	dscreen = m.screens[1].(*deployScreen)
	if dscreen.step != deployResult {
		t.Fatalf("expected deployResult step, got %d", dscreen.step)
	}

	// The cmd should be a RefreshHeaderMsg emitter.
	if cmd == nil {
		t.Fatal("expected RefreshHeaderMsg cmd after deploy done")
	}

	// 4. Verify result view shows success.
	v := m.screens[1].View()
	if !strings.Contains(v.Content, "2 succeeded") {
		t.Errorf("result view missing '2 succeeded'; got:\n%s", v.Content)
	}
	if !strings.Contains(v.Content, "2 of 2 succeeded") {
		t.Errorf("result view missing summary; got:\n%s", v.Content)
	}

	// 5. Press enter — should emit PopToRootMsg batch.
	updated, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(Model)

	if cmd == nil {
		t.Fatal("expected batch cmd from enter at result")
	}

	// 6. Execute the PopToRootMsg from the batch.
	updated, _ = m.Update(PopToRootMsg{})
	m = updated.(Model)

	if len(m.screens) != 1 {
		t.Fatalf("expected 1 screen after PopToRootMsg, got %d", len(m.screens))
	}
	if m.screens[0].Title() != "Main Menu" {
		t.Fatalf("expected Main Menu, got %q", m.screens[0].Title())
	}
}

func TestIntegration_DeployFlow_PartialFailure(t *testing.T) {
	m := newTestModel()

	ds := newDeployScreen(m.svc, m.styles, m.isDark)
	updated, _ := m.Update(NavigateMsg{Screen: ds})
	m = updated.(Model)

	// Scan returns assets.
	updated, _ = m.Update(scanDoneMsg{
		assets: []*asset.Asset{
			{Identity: asset.Identity{SourceID: "local", Type: nd.AssetSkill, Name: "a"}},
			{Identity: asset.Identity{SourceID: "local", Type: nd.AssetSkill, Name: "b"}},
		},
	})
	m = updated.(Model)

	// Deploy with partial failure.
	updated, _ = m.Update(deployDoneMsg{
		succeeded: []deploy.DeployResult{
			{Deployment: state.Deployment{AssetName: "a", AssetType: nd.AssetSkill}},
		},
		failed: []deploy.DeployError{
			{AssetName: "b", AssetType: nd.AssetSkill, Err: fmt.Errorf("permission denied")},
		},
	})
	m = updated.(Model)

	v := m.screens[1].View()
	if !strings.Contains(v.Content, "1 succeeded") {
		t.Errorf("expected '1 succeeded'; got:\n%s", v.Content)
	}
	if !strings.Contains(v.Content, "1 failed") {
		t.Errorf("expected '1 failed'; got:\n%s", v.Content)
	}
	if !strings.Contains(v.Content, "permission denied") {
		t.Errorf("expected error details; got:\n%s", v.Content)
	}
}

// =============================================================================
// Task 7.1 Step 2: Navigation edge cases
// =============================================================================

func TestIntegration_RapidEscPresses(t *testing.T) {
	m := newTestModel()

	// Push 3 screens (all with InputActive=false so esc works).
	m.screens = append(m.screens,
		stubScreen{title: "Status"},
		stubScreen{title: "Doctor"},
	)
	if len(m.screens) != 3 {
		t.Fatalf("expected 3 screens, got %d", len(m.screens))
	}

	esc := tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape})

	// First esc: pop Doctor.
	updated, _ := m.Update(esc)
	m = updated.(Model)
	if len(m.screens) != 2 {
		t.Fatalf("expected 2 screens after first esc, got %d", len(m.screens))
	}

	// Second esc: pop Status.
	updated, _ = m.Update(esc)
	m = updated.(Model)
	if len(m.screens) != 1 {
		t.Fatalf("expected 1 screen after second esc, got %d", len(m.screens))
	}

	// Third esc: on main menu — should quit.
	_, cmd := m.Update(esc)
	if cmd == nil {
		t.Fatal("expected quit cmd on esc at root screen")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("expected QuitMsg, got %T", msg)
	}
}

func TestIntegration_PopToRootFromDeepStack(t *testing.T) {
	m := newTestModel()

	// Build a deep stack: menu -> deploy -> result context (simulated with stubs).
	m.screens = append(m.screens,
		stubScreen{title: "Deploy"},
		stubScreen{title: "Result"},
		stubScreen{title: "Deep"},
	)
	if len(m.screens) != 4 {
		t.Fatalf("expected 4 screens, got %d", len(m.screens))
	}

	updated, _ := m.Update(PopToRootMsg{})
	m = updated.(Model)

	if len(m.screens) != 1 {
		t.Fatalf("expected 1 screen after PopToRootMsg from depth 4, got %d", len(m.screens))
	}
	if m.screens[0].Title() != "Main Menu" {
		t.Fatalf("expected Main Menu at root, got %q", m.screens[0].Title())
	}
}

func TestIntegration_BackMsgFromMainMenu_Quits(t *testing.T) {
	m := newTestModel()

	_, cmd := m.Update(BackMsg{})
	if cmd == nil {
		t.Fatal("expected quit cmd on BackMsg at root")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("expected QuitMsg from BackMsg at root, got %T", msg)
	}
}

func TestIntegration_NavigateAndBackPreservesRoot(t *testing.T) {
	m := newTestModel()

	// Push a screen, then pop it.
	updated, _ := m.Update(NavigateMsg{Screen: stubScreen{title: "Status"}})
	m = updated.(Model)
	if len(m.screens) != 2 {
		t.Fatalf("expected 2 screens, got %d", len(m.screens))
	}

	updated, _ = m.Update(BackMsg{})
	m = updated.(Model)
	if len(m.screens) != 1 {
		t.Fatalf("expected 1 screen after BackMsg, got %d", len(m.screens))
	}
	if m.screens[0].Title() != "Main Menu" {
		t.Fatalf("expected Main Menu preserved, got %q", m.screens[0].Title())
	}
}

// =============================================================================
// Task 7.1 Step 3: Key routing with input screens
// =============================================================================

func TestIntegration_InputActive_QDoesNotQuit(t *testing.T) {
	m := newTestModel()

	// Push a screen with InputActive=true (simulates source add path input).
	m.screens = append(m.screens, stubScreen{title: "Source Add", inputActive: true})

	// Press 'q' — should NOT quit.
	_, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: 'q'}))
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(tea.QuitMsg); ok {
			t.Fatal("'q' should not quit when InputActive is true")
		}
	}

	// Press 'esc' — should NOT pop/quit either (delegated to screen).
	_, cmd = m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(tea.QuitMsg); ok {
			t.Fatal("'esc' should not quit when InputActive is true")
		}
	}
}

func TestIntegration_InputActive_CtrlC_AlwaysQuits(t *testing.T) {
	m := newTestModel()
	m.screens = append(m.screens, stubScreen{title: "Source Add", inputActive: true})

	_, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: 'c', Mod: tea.ModCtrl}))
	if cmd == nil {
		t.Fatal("expected quit cmd for ctrl+c with InputActive")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("expected QuitMsg from ctrl+c, got %T", msg)
	}
}

func TestIntegration_InputActive_TransitionToNonInput(t *testing.T) {
	m := newTestModel()

	// Start with InputActive=true.
	inputScreen := stubScreen{title: "Deploy", inputActive: true}
	m.screens = append(m.screens, inputScreen)

	// 'q' should not quit.
	_, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: 'q'}))
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(tea.QuitMsg); ok {
			t.Fatal("'q' should not quit with InputActive")
		}
	}

	// Replace top screen with InputActive=false.
	m.screens[len(m.screens)-1] = stubScreen{title: "Result", inputActive: false}

	// Now 'q' should quit.
	_, cmd = m.Update(tea.KeyPressMsg(tea.Key{Code: 'q'}))
	if cmd == nil {
		t.Fatal("expected quit cmd for 'q' after InputActive becomes false")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("expected QuitMsg, got %T", msg)
	}
}

// =============================================================================
// Task 7.1 Step 4: Empty states
// =============================================================================

func TestIntegration_StatusScreen_EmptyState(t *testing.T) {
	m := newTestModel()

	ss := newStatusScreen(m.svc, m.styles, m.isDark)
	updated, _ := m.Update(NavigateMsg{Screen: ss})
	m = updated.(Model)

	// Simulate empty status load.
	updated, _ = m.Update(statusLoadedMsg{entries: nil})
	m = updated.(Model)

	v := m.screens[1].View()
	expectedMsg := NothingDeployed()
	if !strings.Contains(v.Content, expectedMsg) {
		t.Errorf("status empty state should contain NothingDeployed(); got:\n%s", v.Content)
	}
}

func TestIntegration_DeployScreen_AllAlreadyDeployed(t *testing.T) {
	m := newTestModel()

	ds := newDeployScreen(m.svc, m.styles, m.isDark)
	updated, _ := m.Update(NavigateMsg{Screen: ds})
	m = updated.(Model)

	// Scan returns no undeployed assets.
	updated, _ = m.Update(scanDoneMsg{assets: nil})
	m = updated.(Model)

	dscreen := m.screens[1].(*deployScreen)
	if dscreen.step != deployResult {
		t.Fatalf("expected deployResult step for empty scan, got %d", dscreen.step)
	}

	v := m.screens[1].View()
	if !strings.Contains(v.Content, "already deployed") {
		t.Errorf("deploy empty state should contain 'already deployed'; got:\n%s", v.Content)
	}
}

func TestIntegration_DeployScreen_ScanError(t *testing.T) {
	m := newTestModel()

	ds := newDeployScreen(m.svc, m.styles, m.isDark)
	updated, _ := m.Update(NavigateMsg{Screen: ds})
	m = updated.(Model)

	// Scan returns error.
	updated, _ = m.Update(scanDoneMsg{err: fmt.Errorf("source unavailable")})
	m = updated.(Model)

	dscreen := m.screens[1].(*deployScreen)
	if dscreen.step != deployResult {
		t.Fatalf("expected deployResult step for scan error, got %d", dscreen.step)
	}

	v := m.screens[1].View()
	if !strings.Contains(v.Content, "source unavailable") {
		t.Errorf("deploy error state should show error; got:\n%s", v.Content)
	}
}

func TestIntegration_DoctorScreen_NoIssues(t *testing.T) {
	m := newTestModel()

	ds := newDoctorScreen(m.svc, m.styles, m.isDark)
	updated, _ := m.Update(NavigateMsg{Screen: ds})
	m = updated.(Model)

	// Doctor check finds no issues.
	updated, _ = m.Update(doctorCheckedMsg{issues: nil})
	m = updated.(Model)

	v := m.screens[1].View()
	if !strings.Contains(v.Content, "healthy") {
		t.Errorf("doctor with no issues should show 'healthy'; got:\n%s", v.Content)
	}
}

// =============================================================================
// Task 7.2: Polish — automated tests
// =============================================================================

// Step 1: NO_COLOR / unstyled — verify output is readable without ANSI.
func TestPolish_UnstyledHeaderReadable(t *testing.T) {
	h := Header{
		Profile:  "default",
		Scope:    "global",
		Agent:    "claude-code",
		Deployed: 5,
		Issues:   2,
		DryRun:   false,
	}
	s := testStyles() // No ANSI escapes.

	got := h.View(s, 80)

	// All elements should be present in plain text.
	for _, want := range []string{"default", "global", "claude-code", "5 deployed", "2 issues"} {
		if !strings.Contains(got, want) {
			t.Errorf("unstyled header missing %q; got: %q", want, got)
		}
	}

	// No ANSI escape sequences.
	if strings.Contains(got, "\033[") {
		t.Error("unstyled header should not contain ANSI escapes")
	}
}

func TestPolish_UnstyledHelpBarReadable(t *testing.T) {
	s := testStyles()
	hb := HelpBar{}
	screen := stubScreen{title: "Test"}

	got := hb.View(s, screen, 80)

	for _, want := range []string{"esc", "back", "q", "quit"} {
		if !strings.Contains(got, want) {
			t.Errorf("unstyled help bar missing %q; got: %q", want, got)
		}
	}

	if strings.Contains(got, "\033[") {
		t.Error("unstyled help bar should not contain ANSI escapes")
	}
}

func TestPolish_UnstyledStatusViewReadable(t *testing.T) {
	svc := newMockServices()
	s := testStyles()
	ss := &statusScreen{svc: svc, styles: s, isDark: true}

	ss.Update(statusLoadedMsg{
		entries: []deploy.StatusEntry{
			{
				Deployment: state.Deployment{AssetType: nd.AssetSkill, AssetName: "test-skill", SourceID: "s1", Scope: nd.ScopeGlobal},
				Health:     state.HealthOK,
			},
			{
				Deployment: state.Deployment{AssetType: nd.AssetSkill, AssetName: "broken-skill", SourceID: "s1", Scope: nd.ScopeGlobal},
				Health:     state.HealthBroken,
			},
		},
	})

	v := ss.View()
	for _, want := range []string{"test-skill", "broken-skill", "2 deployed", "1 issues"} {
		if !strings.Contains(v.Content, want) {
			t.Errorf("unstyled status view missing %q; got:\n%s", want, v.Content)
		}
	}
}

// Step 2: Narrow terminal (60 columns).
func TestPolish_NarrowTerminal_HeaderRenders(t *testing.T) {
	h := Header{
		Profile:  "default",
		Scope:    "global",
		Agent:    "claude-code",
		Deployed: 5,
		Issues:   0,
	}
	s := testStyles()

	// 60 columns — narrower than typical.
	got := h.View(s, 60)

	// Should still contain essential info.
	if !strings.Contains(got, "default") {
		t.Error("narrow header missing profile")
	}
	if !strings.Contains(got, "deployed") {
		t.Error("narrow header missing deployed count")
	}
}

func TestPolish_NarrowTerminal_HelpBarRenders(t *testing.T) {
	s := testStyles()
	hb := HelpBar{}
	screen := stubScreen{title: "Test"}

	got := hb.View(s, screen, 60)

	// Should contain at least esc and quit.
	if !strings.Contains(got, "esc") {
		t.Error("narrow help bar missing esc")
	}
	if !strings.Contains(got, "q") {
		t.Error("narrow help bar missing q")
	}
}

func TestPolish_NarrowTerminal_EmptyStateReadable(t *testing.T) {
	// Empty state messages should not break at narrow widths.
	msgs := []string{
		NoSources(),
		NoAssets(),
		NothingDeployed(),
		NoProfiles(),
		NoSnapshots(),
		AllDeployed("skills"),
	}
	for _, msg := range msgs {
		if msg == "" {
			t.Errorf("empty state message should not be empty")
		}
	}
}

// Step 3: --dry-run in TUI.
func TestPolish_DryRun_HeaderShowsIndicator(t *testing.T) {
	h := Header{
		Profile:  "default",
		Scope:    "global",
		Agent:    "claude-code",
		Deployed: 5,
		Issues:   0,
		DryRun:   true,
	}
	s := testStyles()

	got := h.View(s, 80)
	if !strings.Contains(got, "[DRY RUN]") {
		t.Errorf("dry-run header should contain '[DRY RUN]'; got: %q", got)
	}
}

func TestPolish_DryRun_DeployViewShowsWouldDeploy(t *testing.T) {
	ds := &deployScreen{
		styles: testStyles(),
		step:   deployResult,
		dryRun: true,
		dryReqs: []deploy.DeployRequest{
			{Asset: asset.Asset{Identity: asset.Identity{SourceID: "local", Type: nd.AssetSkill, Name: "test"}}},
		},
	}

	v := ds.View()
	if !strings.Contains(v.Content, "DRY RUN") {
		t.Errorf("dry-run deploy should show 'DRY RUN'; got:\n%s", v.Content)
	}
	if !strings.Contains(v.Content, "Would deploy") {
		t.Errorf("dry-run deploy should show 'Would deploy'; got:\n%s", v.Content)
	}
}

func TestPolish_DryRun_RemoveViewShowsWouldRemove(t *testing.T) {
	ms := &removeScreen{
		styles: testStyles(),
		step:   removeResult,
		dryRun: true,
		dryReqs: []deploy.RemoveRequest{
			{Identity: asset.Identity{Type: nd.AssetSkill, Name: "test"}},
		},
	}

	v := ms.View()
	if !strings.Contains(v.Content, "DRY RUN") {
		t.Errorf("dry-run remove should show 'DRY RUN'; got:\n%s", v.Content)
	}
	if !strings.Contains(v.Content, "Would remove") {
		t.Errorf("dry-run remove should show 'Would remove'; got:\n%s", v.Content)
	}
}

func TestPolish_DryRun_HeaderRefreshSetsDryRunFlag(t *testing.T) {
	svc := newMockServices()
	svc.isDryRunFn = func() bool { return true }

	h := Header{}
	h = h.Refresh(svc)

	if !h.DryRun {
		t.Error("Header.Refresh() should set DryRun from Services.IsDryRun()")
	}
}

// =============================================================================
// Cross-cutting: Verify view composition
// =============================================================================

func TestIntegration_ViewComposition_AllSections(t *testing.T) {
	m := newTestModel()
	m.header = m.header.Refresh(m.svc)
	m.width = 80
	m.height = 24

	v := m.View()
	content := v.Content

	// Header section: scope from mock.
	if !strings.Contains(content, "global") {
		t.Error("composed view missing header scope")
	}

	// Help bar section.
	if !strings.Contains(content, "esc") {
		t.Error("composed view missing help bar")
	}
	if !strings.Contains(content, "quit") {
		t.Error("composed view missing quit in help bar")
	}

	// Alt screen should be enabled.
	if !v.AltScreen {
		t.Error("composed view should enable alt screen")
	}
}

func TestIntegration_EmptyScreenStack_SafeView(t *testing.T) {
	m := Model{
		svc:    newMockServices(),
		styles: NewStyles(true),
	}

	// View with empty screen stack should not panic.
	v := m.View()
	if v.Content != "" {
		t.Errorf("empty stack view should be empty, got %q", v.Content)
	}
}

func TestIntegration_EmptyScreenStack_KeyPress_NoPanic(t *testing.T) {
	m := Model{
		svc:    newMockServices(),
		styles: NewStyles(true),
	}

	// Key press with empty screen stack should not panic.
	updated, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: 'q'}))
	if updated == nil {
		t.Fatal("Update returned nil model")
	}
	if cmd != nil {
		t.Fatal("expected nil cmd for key press on empty stack")
	}
}
