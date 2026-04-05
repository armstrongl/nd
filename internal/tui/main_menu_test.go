package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
)

// Compile-time assertion: mainMenuScreen satisfies Screen (and therefore tea.Model).
var _ Screen = (*mainMenuScreen)(nil)

func TestMainMenu_NewReturnsNonNil(t *testing.T) {
	s := NewStyles(true)
	m := newMainMenuScreen(newMockServices(), s, true)
	if m == nil {
		t.Fatal("newMainMenuScreen returned nil")
	}
}

func TestMainMenu_Title(t *testing.T) {
	s := NewStyles(true)
	m := newMainMenuScreen(newMockServices(), s, true)
	if got := m.Title(); got != "Main Menu" {
		t.Fatalf("Title() = %q, want %q", got, "Main Menu")
	}
}

func TestMainMenu_InputActive(t *testing.T) {
	s := NewStyles(true)
	m := newMainMenuScreen(newMockServices(), s, true)
	if m.InputActive() {
		t.Fatal("InputActive() = true, want false")
	}
}

func TestMainMenu_FormNotNil(t *testing.T) {
	s := NewStyles(true)
	m := newMainMenuScreen(newMockServices(), s, true)
	if m.form == nil {
		t.Fatal("form field is nil after construction")
	}
}

func TestMainMenu_InitReturnsCmd(t *testing.T) {
	s := NewStyles(true)
	m := newMainMenuScreen(newMockServices(), s, true)
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil; huh forms need initialization commands")
	}
}

func TestMainMenu_InitialStateIsNormal(t *testing.T) {
	s := NewStyles(true)
	m := newMainMenuScreen(newMockServices(), s, true)
	if m.form.State != huh.StateNormal {
		t.Fatalf("form.State = %d, want StateNormal (%d)", m.form.State, huh.StateNormal)
	}
}

func TestMainMenu_ViewReturnsNonEmpty(t *testing.T) {
	s := NewStyles(true)
	m := newMainMenuScreen(newMockServices(), s, true)
	// Init the form so it has content to render.
	m.Init()
	v := m.View()
	if v.Content == "" {
		t.Fatal("View() returned empty content")
	}
}

func TestMainMenu_HandleSelectionQuit(t *testing.T) {
	s := NewStyles(true)
	m := newMainMenuScreen(newMockServices(), s, true)
	m.choice = "quit"

	cmd := m.handleSelection()
	if cmd == nil {
		t.Fatal("handleSelection() returned nil for quit, want tea.Quit")
	}

	// tea.Quit returns a QuitMsg when called. Verify the cmd produces
	// the correct message type.
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("quit cmd produced %T, want tea.QuitMsg", msg)
	}
}

func TestMainMenu_HandleSelectionUnset(t *testing.T) {
	s := NewStyles(true)
	m := newMainMenuScreen(newMockServices(), s, true)
	m.choice = ""

	cmd := m.handleSelection()
	if cmd != nil {
		t.Fatal("handleSelection() returned non-nil for empty choice, want nil")
	}
}

func TestMainMenu_HandleSelectionWiredScreens(t *testing.T) {
	s := NewStyles(true)
	m := newMainMenuScreen(newMockServices(), s, true)

	// These choices are wired to real screens and should return NavigateMsg.
	wiredChoices := []string{
		"deploy", "remove", "status", "browse", "doctor",
		"profile", "snapshot", "pin", "source", "scope", "settings",
	}
	for _, choice := range wiredChoices {
		m.choice = choice
		cmd := m.handleSelection()
		if cmd == nil {
			t.Errorf("handleSelection() for %q returned nil, want NavigateMsg cmd", choice)
			continue
		}
		msg := cmd()
		if _, ok := msg.(NavigateMsg); !ok {
			t.Errorf("handleSelection() for %q produced %T, want NavigateMsg", choice, msg)
		}
	}
}

func TestMainMenu_HandleSelectionExport(t *testing.T) {
	s := NewStyles(true)
	m := newMainMenuScreen(newMockServices(), s, true)
	m.choice = "export"

	cmd := m.handleSelection()
	if cmd == nil {
		t.Fatal("handleSelection() for export returned nil, want BackMsg cmd")
	}
	msg := cmd()
	if _, ok := msg.(BackMsg); !ok {
		t.Fatalf("export cmd produced %T, want BackMsg", msg)
	}
}

func TestMainMenu_ChoiceDefaultFirstOption(t *testing.T) {
	s := NewStyles(true)
	m := newMainMenuScreen(newMockServices(), s, true)
	// huh Select initializes the bound value to the first option.
	if m.choice != "deploy" {
		t.Fatalf("choice = %q, want %q (first option default)", m.choice, "deploy")
	}
}

func TestMainMenu_DarkAndLightModes(t *testing.T) {
	// Verify both dark and light modes produce valid screens.
	for _, isDark := range []bool{true, false} {
		s := NewStyles(isDark)
		m := newMainMenuScreen(newMockServices(), s, isDark)
		if m.isDark != isDark {
			t.Fatalf("isDark = %v, want %v", m.isDark, isDark)
		}
	}
}

func TestMainMenu_StylesPreserved(t *testing.T) {
	s := NewStyles(true)
	m := newMainMenuScreen(newMockServices(), s, true)
	// Verify the styles field is set (basic structural check).
	if !m.styles.Bold.GetBold() {
		t.Fatal("styles.Bold should have bold attribute set")
	}
}

func TestMainMenu_HasGroupSeparators(t *testing.T) {
	s := NewStyles(true)
	m := newMainMenuScreen(newMockServices(), s, true)
	m.Init() // huh form needs Init before View renders options

	// The menu should contain group separator options with sentinel values.
	// Separators appear between logical groups, not before the first item.
	v := m.View()
	content := v.Content

	// Group headers should appear in rendered output.
	for _, header := range []string{"Manage", "System"} {
		if !strings.Contains(content, header) {
			t.Errorf("View() does not contain group header %q", header)
		}
	}
}

func TestMainMenu_SeparatorsAreNoOp(t *testing.T) {
	s := NewStyles(true)
	m := newMainMenuScreen(newMockServices(), s, true)

	// Separator sentinel values should return nil from handleSelection.
	for _, sep := range []string{menuSepManage, menuSepSystem} {
		m.choice = sep
		cmd := m.handleSelection()
		if cmd != nil {
			t.Errorf("handleSelection() for separator %q returned non-nil, want nil", sep)
		}
	}
}

func TestMainMenu_SeparatorDoesNotFreeze(t *testing.T) {
	s := NewStyles(true)
	m := newMainMenuScreen(newMockServices(), s, true)

	// Simulate the form completing with a separator value.
	// Force the form state to completed and set choice to a separator.
	m.choice = menuSepManage
	m.form.State = huh.StateCompleted

	// Call Update — this triggers handleSelection which returns nil.
	// The fix should reset navigated to false so the menu stays responsive.
	updated, _ := m.Update(nil)
	menu := updated.(*mainMenuScreen)
	if menu.navigated {
		t.Fatal("navigated should be false after selecting a separator (menu would freeze)")
	}
}

func TestMainMenu_UnknownChoiceDoesNotFreeze(t *testing.T) {
	s := NewStyles(true)
	m := newMainMenuScreen(newMockServices(), s, true)

	// Simulate the form completing with an unknown value.
	m.choice = "totally_bogus_value"
	m.form.State = huh.StateCompleted

	updated, _ := m.Update(nil)
	menu := updated.(*mainMenuScreen)
	if menu.navigated {
		t.Fatal("navigated should be false after selecting an unknown choice (menu would freeze)")
	}
}

func TestMainMenu_FirstOptionIsDeployNotSeparator(t *testing.T) {
	s := NewStyles(true)
	m := newMainMenuScreen(newMockServices(), s, true)
	// huh auto-selects option 0. It must be "deploy", not a separator.
	if m.choice != "deploy" {
		t.Fatalf("choice = %q after construction, want %q (first option must be a real item, not separator)", m.choice, "deploy")
	}
}

