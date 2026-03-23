package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// stubScreen is a minimal Screen implementation for compile-time and runtime checks.
type stubScreen struct {
	title       string
	inputActive bool
}

func (s stubScreen) Init() tea.Cmd                       { return nil }
func (s stubScreen) Update(tea.Msg) (tea.Model, tea.Cmd) { return s, nil }
func (s stubScreen) View() tea.View                      { return tea.NewView("stub") }
func (s stubScreen) Title() string                       { return s.title }
func (s stubScreen) InputActive() bool                   { return s.inputActive }

// Compile-time assertion: stubScreen satisfies Screen (and therefore tea.Model).
var _ Screen = stubScreen{}

func TestNavigateMsg_IsDistinctType(t *testing.T) {
	var msg interface{} = NavigateMsg{}
	if _, ok := msg.(NavigateMsg); !ok {
		t.Fatal("expected NavigateMsg type assertion to succeed")
	}
	if _, ok := msg.(BackMsg); ok {
		t.Fatal("NavigateMsg should not satisfy BackMsg")
	}
}

func TestBackMsg_IsDistinctType(t *testing.T) {
	var msg interface{} = BackMsg{}
	if _, ok := msg.(BackMsg); !ok {
		t.Fatal("expected BackMsg type assertion to succeed")
	}
	if _, ok := msg.(NavigateMsg); ok {
		t.Fatal("BackMsg should not satisfy NavigateMsg")
	}
}

func TestPopToRootMsg_IsDistinctType(t *testing.T) {
	var msg interface{} = PopToRootMsg{}
	if _, ok := msg.(PopToRootMsg); !ok {
		t.Fatal("expected PopToRootMsg type assertion to succeed")
	}
	if _, ok := msg.(BackMsg); ok {
		t.Fatal("PopToRootMsg should not satisfy BackMsg")
	}
}

func TestRefreshHeaderMsg_IsDistinctType(t *testing.T) {
	var msg interface{} = RefreshHeaderMsg{}
	if _, ok := msg.(RefreshHeaderMsg); !ok {
		t.Fatal("expected RefreshHeaderMsg type assertion to succeed")
	}
	if _, ok := msg.(PopToRootMsg); ok {
		t.Fatal("RefreshHeaderMsg should not satisfy PopToRootMsg")
	}
}

func TestNavigateMsg_CarriesScreen(t *testing.T) {
	screen := stubScreen{title: "Deploy", inputActive: false}
	msg := NavigateMsg{Screen: screen}

	if msg.Screen == nil {
		t.Fatal("expected NavigateMsg.Screen to be non-nil")
	}
	if msg.Screen.Title() != "Deploy" {
		t.Fatalf("expected Screen.Title() = %q, got %q", "Deploy", msg.Screen.Title())
	}
	if msg.Screen.InputActive() {
		t.Fatal("expected Screen.InputActive() = false")
	}
}

func TestNavigateMsg_CarriesScreenWithInputActive(t *testing.T) {
	screen := stubScreen{title: "Source Add", inputActive: true}
	msg := NavigateMsg{Screen: screen}

	if !msg.Screen.InputActive() {
		t.Fatal("expected Screen.InputActive() = true")
	}
}

func TestStubScreen_SatisfiesTeaModel(t *testing.T) {
	// Verify the Screen interface embeds tea.Model by checking all three methods.
	var s Screen = stubScreen{title: "Test"}

	cmd := s.Init()
	if cmd != nil {
		t.Fatal("expected Init() to return nil")
	}

	updated, cmd := s.Update(nil)
	if updated == nil {
		t.Fatal("expected Update() to return non-nil model")
	}
	if cmd != nil {
		t.Fatal("expected Update() to return nil cmd")
	}

	v := s.View()
	if v.Content == "" {
		t.Fatal("expected View() to return non-empty content")
	}
}

func TestAllMessageTypes_SwitchDispatch(t *testing.T) {
	// Verify all four message types can be dispatched in a type switch,
	// which is the pattern the root model uses.
	messages := []interface{}{
		NavigateMsg{Screen: stubScreen{title: "Deploy"}},
		BackMsg{},
		PopToRootMsg{},
		RefreshHeaderMsg{},
	}

	for i, msg := range messages {
		matched := false
		switch msg.(type) {
		case NavigateMsg:
			if i != 0 {
				t.Fatalf("NavigateMsg matched at index %d, expected 0", i)
			}
			matched = true
		case BackMsg:
			if i != 1 {
				t.Fatalf("BackMsg matched at index %d, expected 1", i)
			}
			matched = true
		case PopToRootMsg:
			if i != 2 {
				t.Fatalf("PopToRootMsg matched at index %d, expected 2", i)
			}
			matched = true
		case RefreshHeaderMsg:
			if i != 3 {
				t.Fatalf("RefreshHeaderMsg matched at index %d, expected 3", i)
			}
			matched = true
		}
		if !matched {
			t.Fatalf("message at index %d was not matched by type switch", i)
		}
	}
}
