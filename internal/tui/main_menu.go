package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
)

// Sentinel values for menu group separator options.
// These are not selectable — handleSelection returns nil for them.
const (
	menuSepManage = "__sep_manage__"
	menuSepSystem = "__sep_system__"
)

type mainMenuScreen struct {
	svc       Services
	form      *huh.Form
	choice    string
	styles    Styles
	isDark    bool
	navigated bool // guards against double-fire after form completion
}

func newMainMenuScreen(svc Services, styles Styles, isDark bool) *mainMenuScreen {
	m := &mainMenuScreen{
		svc:    svc,
		styles: styles,
		isDark: isDark,
	}

	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("nd").
				Options(
					// ── Actions (first group, no header — first option must be real)
					huh.NewOption("Deploy assets", "deploy"),
					huh.NewOption("Remove assets", "remove"),
					huh.NewOption("Browse assets", "browse"),
					huh.NewOption("View status", "status"),
					huh.NewOption("Run doctor", "doctor"),
					// ── Manage
					huh.NewOption("── Manage ──", menuSepManage),
					huh.NewOption("Switch profile", "profile"),
					huh.NewOption("Manage snapshots", "snapshot"),
					huh.NewOption("Pin/Unpin assets", "pin"),
					huh.NewOption("Manage sources", "source"),
					huh.NewOption("Export plugin", "export"),
					// ── System
					huh.NewOption("── System ──", menuSepSystem),
					huh.NewOption("Switch scope", "scope"),
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
	if m.navigated {
		return m, nil
	}

	model, cmd := m.form.Update(msg)
	if f, ok := model.(*huh.Form); ok {
		m.form = f
	}

	if m.form.State == huh.StateCompleted {
		m.navigated = true
		if cmd := m.handleSelection(); cmd != nil {
			return m, cmd
		}
		// handleSelection returned nil (separator or unknown value).
		// Reset navigated so the menu stays responsive.
		m.navigated = false
		return m, nil
	}

	return m, cmd
}

// View renders the form, converting the string output to tea.View.
func (m *mainMenuScreen) View() tea.View {
	return tea.NewView(m.form.View())
}

// handleSelection maps the selected menu choice to a navigation command.
func (m *mainMenuScreen) handleSelection() tea.Cmd {
	var screen Screen
	switch m.choice {
	case "deploy":
		screen = newDeployScreen(m.svc, m.styles, m.isDark)
	case "remove":
		screen = newRemoveScreen(m.svc, m.styles, m.isDark)
	case "status":
		screen = newStatusScreen(m.svc, m.styles, m.isDark)
	case "browse":
		screen = newBrowseScreen(m.svc, m.styles, m.isDark)
	case "doctor":
		screen = newDoctorScreen(m.svc, m.styles, m.isDark)
	case "profile":
		screen = newProfileScreen(m.svc, m.styles, m.isDark)
	case "snapshot":
		screen = newSnapshotScreen(m.svc, m.styles, m.isDark)
	case "pin":
		screen = newPinScreen(m.svc, m.styles, m.isDark)
	case "source":
		screen = newSourceScreen(m.svc, m.styles, m.isDark)
	case "scope":
		screen = newScopeScreen(m.svc, m.styles, m.isDark)
	case "settings":
		screen = newSettingsScreen(m.svc, m.styles, m.isDark)
	case "export":
		// Export has no TUI screen yet — return to the main menu.
		return func() tea.Msg { return BackMsg{} }
	case "quit":
		return tea.Quit
	default:
		return nil
	}
	return func() tea.Msg { return NavigateMsg{Screen: screen} }
}
