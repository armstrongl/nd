package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
)

type mainMenuScreen struct {
	form   *huh.Form
	choice string
	styles Styles
	isDark bool
}

func newMainMenuScreen(styles Styles, isDark bool) *mainMenuScreen {
	m := &mainMenuScreen{
		styles: styles,
		isDark: isDark,
	}

	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("nd").
				Options(
					huh.NewOption("Deploy assets", "deploy"),
					huh.NewOption("Remove assets", "remove"),
					huh.NewOption("Browse assets", "browse"),
					huh.NewOption("View status", "status"),
					huh.NewOption("Run doctor", "doctor"),
					huh.NewOption("Switch profile", "profile"),
					huh.NewOption("Manage snapshots", "snapshot"),
					huh.NewOption("Pin/Unpin assets", "pin"),
					huh.NewOption("Manage sources", "source"),
					huh.NewOption("Export plugin", "export"),
					huh.NewOption("Settings", "settings"),
					huh.NewOption("Quit", "quit"),
				).
				Value(&m.choice),
		),
	).WithTheme(huh.ThemeFunc(huh.ThemeCatppuccin))

	return m
}

// Screen interface
func (m *mainMenuScreen) Title() string    { return "Main Menu" }
func (m *mainMenuScreen) InputActive() bool { return false }

// Init initializes the embedded huh form.
func (m *mainMenuScreen) Init() tea.Cmd {
	return m.form.Init()
}

// Update delegates to the huh form and checks for completion.
func (m *mainMenuScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Check if form completed before this update.
	if m.form.State == huh.StateCompleted {
		return m, m.handleSelection()
	}

	// Delegate to the huh form. Form.Update returns (huh.Model, tea.Cmd)
	// where huh.Model is the compat.Model interface, not tea.Model.
	model, cmd := m.form.Update(msg)
	if f, ok := model.(*huh.Form); ok {
		m.form = f
	}

	// Check again after update.
	if m.form.State == huh.StateCompleted {
		return m, m.handleSelection()
	}

	return m, cmd
}

// View renders the form, converting the string output to tea.View.
func (m *mainMenuScreen) View() tea.View {
	return tea.NewView(m.form.View())
}

// handleSelection maps the selected menu choice to a tea.Cmd.
func (m *mainMenuScreen) handleSelection() tea.Cmd {
	switch m.choice {
	case "quit":
		return tea.Quit
	default:
		// Other selections will be wired to real screens in later phases.
		return nil
	}
}
