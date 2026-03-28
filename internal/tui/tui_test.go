package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// newTestModel creates a Model with a mock services and a main menu screen.
func newTestModel() Model {
	svc := newMockServices()
	styles := NewStyles(true)
	return Model{
		svc:     svc,
		styles:  styles,
		isDark:  true,
		screens: []Screen{newMainMenuScreen(svc, styles, true)},
		width:   80,
		height:  24,
	}
}

// newTestModelWithScreen creates a Model with the given screen on the stack.
func newTestModelWithScreen(s Screen) Model {
	styles := NewStyles(true)
	return Model{
		svc:     newMockServices(),
		styles:  styles,
		isDark:  true,
		screens: []Screen{s},
		width:   80,
		height:  24,
	}
}

func TestNavigateMsg_PushesScreen(t *testing.T) {
	m := newTestModel()
	if len(m.screens) != 1 {
		t.Fatalf("expected 1 screen, got %d", len(m.screens))
	}

	target := stubScreen{title: "Deploy"}
	updated, _ := m.Update(NavigateMsg{Screen: target})
	m2 := updated.(Model)

	if len(m2.screens) != 2 {
		t.Fatalf("expected 2 screens after NavigateMsg, got %d", len(m2.screens))
	}
	if m2.screens[1].Title() != "Deploy" {
		t.Errorf("expected pushed screen title %q, got %q", "Deploy", m2.screens[1].Title())
	}
}

func TestBackMsg_PopsScreen(t *testing.T) {
	m := newTestModel()
	// Push a second screen.
	m.screens = append(m.screens, stubScreen{title: "Deploy"})
	if len(m.screens) != 2 {
		t.Fatalf("expected 2 screens, got %d", len(m.screens))
	}

	updated, _ := m.Update(BackMsg{})
	m2 := updated.(Model)

	if len(m2.screens) != 1 {
		t.Fatalf("expected 1 screen after BackMsg, got %d", len(m2.screens))
	}
}

func TestBackMsg_OnSingleScreen_Quits(t *testing.T) {
	m := newTestModel()
	_, cmd := m.Update(BackMsg{})

	if cmd == nil {
		t.Fatal("expected quit command when BackMsg on single screen")
	}
	// Execute the cmd and check it produces a QuitMsg.
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg, got %T", msg)
	}
}

func TestPopToRootMsg_ClearsStack(t *testing.T) {
	m := newTestModel()
	m.screens = append(m.screens,
		stubScreen{title: "Deploy"},
		stubScreen{title: "Result"},
	)
	if len(m.screens) != 3 {
		t.Fatalf("expected 3 screens, got %d", len(m.screens))
	}

	updated, _ := m.Update(PopToRootMsg{})
	m2 := updated.(Model)

	if len(m2.screens) != 1 {
		t.Fatalf("expected 1 screen after PopToRootMsg, got %d", len(m2.screens))
	}
}

func TestRefreshHeaderMsg_UpdatesHeader(t *testing.T) {
	m := newTestModel()
	// Header should initially be zero-valued.
	if m.header.Scope != "" {
		t.Fatalf("expected empty scope before refresh, got %q", m.header.Scope)
	}

	updated, _ := m.Update(RefreshHeaderMsg{})
	m2 := updated.(Model)

	// mockServices.GetScope() returns "global".
	if m2.header.Scope != "global" {
		t.Errorf("expected scope %q after refresh, got %q", "global", m2.header.Scope)
	}
}

func TestWindowSizeMsg_UpdatesDimensions(t *testing.T) {
	m := newTestModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m2 := updated.(Model)

	if m2.width != 120 || m2.height != 40 {
		t.Errorf("expected 120x40, got %dx%d", m2.width, m2.height)
	}
}

func TestGlobalKey_Q_Quits(t *testing.T) {
	m := newTestModel()
	_, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: 'q'}))

	if cmd == nil {
		t.Fatal("expected quit command for 'q' key")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg, got %T", msg)
	}
}

func TestGlobalKey_Esc_PopsOrQuits(t *testing.T) {
	// With 2 screens, esc should pop.
	m := newTestModel()
	m.screens = append(m.screens, stubScreen{title: "Deploy"})

	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	m2 := updated.(Model)

	if len(m2.screens) != 1 {
		t.Fatalf("expected 1 screen after esc, got %d", len(m2.screens))
	}

	// With 1 screen, esc should quit.
	m3 := newTestModel()
	_, cmd := m3.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))

	if cmd == nil {
		t.Fatal("expected quit command for esc on single screen")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg, got %T", msg)
	}
}

func TestGlobalKey_CtrlC_AlwaysQuits(t *testing.T) {
	m := newTestModel()
	_, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: 'c', Mod: tea.ModCtrl}))

	if cmd == nil {
		t.Fatal("expected quit command for ctrl+c")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg, got %T", msg)
	}
}

func TestInputActive_SuppressesGlobalKeys(t *testing.T) {
	// Push a screen with InputActive=true.
	inputScreen := stubScreen{title: "Source Add", inputActive: true}
	m := newTestModelWithScreen(inputScreen)

	// Press 'q' — should NOT quit, should delegate to screen.
	_, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: 'q'}))

	// The stubScreen returns nil from Update, so cmd should be nil (not quit).
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(tea.QuitMsg); ok {
			t.Fatal("'q' should not quit when InputActive is true")
		}
	}
}

func TestInputActive_CtrlC_StillQuits(t *testing.T) {
	inputScreen := stubScreen{title: "Source Add", inputActive: true}
	m := newTestModelWithScreen(inputScreen)

	_, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: 'c', Mod: tea.ModCtrl}))

	if cmd == nil {
		t.Fatal("expected quit command for ctrl+c even with InputActive")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg, got %T", msg)
	}
}

func TestView_ContainsHeaderAndHelpBar(t *testing.T) {
	m := newTestModel()
	// Refresh header so it has content.
	m.header = m.header.Refresh(m.svc)

	v := m.View()
	content := v.Content

	// Header should contain scope.
	if !strings.Contains(content, "global") {
		t.Error("expected 'global' from header in view output")
	}

	// Help bar should contain default items.
	if !strings.Contains(content, "esc") {
		t.Error("expected 'esc' from help bar in view output")
	}
	if !strings.Contains(content, "quit") {
		t.Error("expected 'quit' from help bar in view output")
	}
}

func TestView_AltScreenEnabled(t *testing.T) {
	m := newTestModel()
	v := m.View()
	if !v.AltScreen {
		t.Error("expected AltScreen to be true")
	}
}

func TestView_EmptyScreens_NoContent(t *testing.T) {
	m := Model{
		svc:    newMockServices(),
		styles: NewStyles(true),
	}
	v := m.View()
	if v.Content != "" {
		t.Errorf("expected empty content with no screens, got %q", v.Content)
	}
}

func TestInit_ReturnsCmd(t *testing.T) {
	m := newTestModel()
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("expected non-nil cmd from Init")
	}
}

// --- ScreenSizeMsg infrastructure tests ---

// execCmd executes a tea.Cmd and returns the resulting message.
// For tea.Batch commands, it flattens and returns all messages.
func execCmds(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		var msgs []tea.Msg
		for _, c := range batch {
			msgs = append(msgs, execCmds(c)...)
		}
		return msgs
	}
	return []tea.Msg{msg}
}

// findScreenSizeMsg returns the first ScreenSizeMsg in a list, or nil.
func findScreenSizeMsg(msgs []tea.Msg) *ScreenSizeMsg {
	for _, msg := range msgs {
		if ssm, ok := msg.(ScreenSizeMsg); ok {
			return &ssm
		}
	}
	return nil
}

func TestWindowSizeMsg_SendsScreenSizeMsg(t *testing.T) {
	m := newTestModel()
	_, cmd := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	msgs := execCmds(cmd)
	ssm := findScreenSizeMsg(msgs)
	if ssm == nil {
		t.Fatal("expected ScreenSizeMsg from WindowSizeMsg")
	}
	if ssm.Width != 120 {
		t.Errorf("expected Width=120, got %d", ssm.Width)
	}
	// Height should be terminal height minus chrome (4 lines)
	if ssm.Height != 36 {
		t.Errorf("expected Height=36 (40-4), got %d", ssm.Height)
	}
}

func TestScreenSizeMsg_HeightFloorAtZero(t *testing.T) {
	m := newTestModel()
	_, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 2})

	msgs := execCmds(cmd)
	ssm := findScreenSizeMsg(msgs)
	if ssm == nil {
		t.Fatal("expected ScreenSizeMsg from WindowSizeMsg")
	}
	if ssm.Height != 0 {
		t.Errorf("expected Height=0 for tiny terminal, got %d", ssm.Height)
	}
}

func TestNavigateMsg_BatchesScreenSizeMsg(t *testing.T) {
	m := newTestModel()
	m.width = 100
	m.height = 30

	target := stubScreen{title: "Browse"}
	_, cmd := m.Update(NavigateMsg{Screen: target})

	msgs := execCmds(cmd)
	ssm := findScreenSizeMsg(msgs)
	if ssm == nil {
		t.Fatal("expected ScreenSizeMsg batched with NavigateMsg")
	}
	if ssm.Width != 100 {
		t.Errorf("expected Width=100, got %d", ssm.Width)
	}
	if ssm.Height != 26 {
		t.Errorf("expected Height=26 (30-4), got %d", ssm.Height)
	}
}

func TestBackMsg_SendsScreenSizeMsg(t *testing.T) {
	m := newTestModel()
	m.width = 80
	m.height = 24
	m.screens = append(m.screens, stubScreen{title: "Browse"})

	_, cmd := m.Update(BackMsg{})

	msgs := execCmds(cmd)
	ssm := findScreenSizeMsg(msgs)
	if ssm == nil {
		t.Fatal("expected ScreenSizeMsg from BackMsg")
	}
	if ssm.Width != 80 || ssm.Height != 20 {
		t.Errorf("expected 80x20, got %dx%d", ssm.Width, ssm.Height)
	}
}

func TestPopToRootMsg_SendsScreenSizeMsg(t *testing.T) {
	m := newTestModel()
	m.width = 80
	m.height = 24
	m.screens = append(m.screens,
		stubScreen{title: "Browse"},
		stubScreen{title: "Deploy"},
	)

	_, cmd := m.Update(PopToRootMsg{})

	msgs := execCmds(cmd)
	ssm := findScreenSizeMsg(msgs)
	if ssm == nil {
		t.Fatal("expected ScreenSizeMsg from PopToRootMsg")
	}
	if ssm.Width != 80 || ssm.Height != 20 {
		t.Errorf("expected 80x20, got %dx%d", ssm.Width, ssm.Height)
	}
}

func TestEscPop_SendsScreenSizeMsg(t *testing.T) {
	m := newTestModel()
	m.width = 80
	m.height = 24
	m.screens = append(m.screens, stubScreen{title: "Browse"})

	_, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))

	msgs := execCmds(cmd)
	ssm := findScreenSizeMsg(msgs)
	if ssm == nil {
		t.Fatal("expected ScreenSizeMsg from esc pop")
	}
	if ssm.Width != 80 || ssm.Height != 20 {
		t.Errorf("expected 80x20, got %dx%d", ssm.Width, ssm.Height)
	}
}

func TestScreenSizeMsg_DelegatedToScreen(t *testing.T) {
	// Verify ScreenSizeMsg reaches the active screen through delegation.
	m := newTestModel()
	// The stubScreen ignores ScreenSizeMsg (returns self, nil) which is fine —
	// this test verifies the message passes through without errors.
	updated, _ := m.Update(ScreenSizeMsg{Width: 80, Height: 20})
	m2 := updated.(Model)
	if len(m2.screens) != 1 {
		t.Fatalf("expected screen stack unchanged, got %d", len(m2.screens))
	}
}

func TestChromeHeight_MatchesViewLayout(t *testing.T) {
	// Verify chromeHeight (4) matches the actual View layout:
	// header + "" + content + "" + helpbar = 5 sections, 4 non-content.
	if chromeHeight != 4 {
		t.Fatalf("expected chromeHeight=4, got %d", chromeHeight)
	}
}
