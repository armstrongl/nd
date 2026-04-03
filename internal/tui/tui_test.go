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

// PopToRootMsg must recreate the main menu so its huh form is fresh.
// The old instance has navigated=true which blocks all input.
// BackMsg returning to root must recreate the main menu so its huh form is fresh.
func TestBackMsg_RecreatesMainMenu(t *testing.T) {
	m := newTestModel()

	// Mark the root menu stale (as happens after a selection is made).
	stale := m.screens[0].(*mainMenuScreen)
	stale.navigated = true

	// Push a second screen and go back.
	m.screens = append(m.screens, stubScreen{title: "Deploy"})
	updated, cmd := m.Update(BackMsg{})
	m2 := updated.(Model)

	if len(m2.screens) != 1 {
		t.Fatalf("expected 1 screen after BackMsg, got %d", len(m2.screens))
	}
	fresh, ok := m2.screens[0].(*mainMenuScreen)
	if !ok {
		t.Fatalf("expected *mainMenuScreen at root, got %T", m2.screens[0])
	}
	if fresh.navigated {
		t.Error("recreated main menu should not have navigated=true")
	}
	if cmd == nil {
		t.Fatal("expected Init cmd from fresh main menu")
	}
}

// Esc returning to root must recreate the main menu so its huh form is fresh.
func TestGlobalKey_Esc_RecreatesMainMenu(t *testing.T) {
	m := newTestModel()

	stale := m.screens[0].(*mainMenuScreen)
	stale.navigated = true

	m.screens = append(m.screens, stubScreen{title: "Status"})
	updated, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	m2 := updated.(Model)

	if len(m2.screens) != 1 {
		t.Fatalf("expected 1 screen after esc, got %d", len(m2.screens))
	}
	fresh, ok := m2.screens[0].(*mainMenuScreen)
	if !ok {
		t.Fatalf("expected *mainMenuScreen at root, got %T", m2.screens[0])
	}
	if fresh.navigated {
		t.Error("recreated main menu should not have navigated=true")
	}
	if cmd == nil {
		t.Fatal("expected Init cmd from fresh main menu")
	}
}

func TestPopToRootMsg_RecreatesMainMenu(t *testing.T) {
	m := newTestModel()

	// Exhaust the main menu by marking it navigated.
	stale := m.screens[0].(*mainMenuScreen)
	stale.navigated = true

	m.screens = append(m.screens, stubScreen{title: "Deploy"})

	updated, cmd := m.Update(PopToRootMsg{})
	m2 := updated.(Model)

	if len(m2.screens) != 1 {
		t.Fatalf("expected 1 screen, got %d", len(m2.screens))
	}
	fresh, ok := m2.screens[0].(*mainMenuScreen)
	if !ok {
		t.Fatalf("expected *mainMenuScreen at root, got %T", m2.screens[0])
	}
	if fresh.navigated {
		t.Error("recreated main menu should not have navigated=true")
	}
	if cmd == nil {
		t.Fatal("expected Init cmd from fresh main menu")
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
