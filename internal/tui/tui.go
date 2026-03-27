// Package tui provides an interactive terminal UI for managing nd assets.
package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Model is the root Bubble Tea model. It manages a stack of screens,
// a persistent header, and a context-sensitive help bar.
type Model struct {
	svc     Services
	styles  Styles
	screens []Screen
	header  Header
	helpbar HelpBar
	width   int
	height  int
	isDark  bool
}

// Run launches the TUI. It detects the terminal color scheme, determines the
// initial screen (first-run init or main menu), and starts the Bubble Tea program.
func Run(svc Services) error {
	// Default to dark; updated by tea.BackgroundColorMsg when the terminal responds.
	isDark := true
	styles := NewStyles(isDark)

	initial := Screen(newMainMenuScreen(svc, styles, isDark))

	m := Model{
		svc:     svc,
		styles:  styles,
		isDark:  isDark,
		screens: []Screen{initial},
	}

	p := tea.NewProgram(m)
	_, err := p.Run()
	return err
}

// Init bootstraps the header with current state.
func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd

	// Initialize the first screen.
	if len(m.screens) > 0 {
		cmds = append(cmds, m.screens[0].Init())
	}

	// Refresh header asynchronously.
	cmds = append(cmds, func() tea.Msg {
		return RefreshHeaderMsg{}
	})

	return tea.Batch(cmds...)
}

// Update handles navigation messages, global key bindings, and delegates
// everything else to the current screen.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.BackgroundColorMsg:
		isDark := msg.IsDark()
		if isDark != m.isDark {
			m.isDark = isDark
			m.styles = NewStyles(isDark)
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case NavigateMsg:
		m.screens = append(m.screens, msg.Screen)
		return m, msg.Screen.Init()

	case BackMsg:
		if len(m.screens) <= 1 {
			return m, tea.Quit
		}
		n := len(m.screens)
		m.screens[n-1] = nil // release for GC
		m.screens = m.screens[:n-1]
		return m, nil

	case PopToRootMsg:
		for i := 1; i < len(m.screens); i++ {
			m.screens[i] = nil // release for GC
		}
		m.screens = m.screens[:1]
		return m, nil

	case RefreshHeaderMsg:
		m.header = m.header.Refresh(m.svc)
		return m, nil

	case tea.KeyPressMsg:
		if len(m.screens) == 0 {
			return m, nil
		}
		current := m.screens[len(m.screens)-1]

		// When text input is active, only ctrl+c force-quits.
		if current.InputActive() {
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
			// Fall through to delegate to screen.
		} else {
			// Global keys only when no text input is active.
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "esc":
				if len(m.screens) <= 1 {
					return m, tea.Quit
				}
				n := len(m.screens)
				m.screens[n-1] = nil // release for GC
				m.screens = m.screens[:n-1]
				return m, nil
			}
		}
	}

	// Delegate to current screen.
	if len(m.screens) > 0 {
		idx := len(m.screens) - 1
		updated, cmd := m.screens[idx].Update(msg)
		if scr, ok := updated.(Screen); ok {
			m.screens[idx] = scr
		}
		return m, cmd
	}
	return m, nil
}

// View composes header, current screen content, and help bar vertically.
func (m Model) View() tea.View {
	if len(m.screens) == 0 {
		return tea.NewView("")
	}

	header := m.header.View(m.styles, m.width)
	content := m.screens[len(m.screens)-1].View().Content
	helpbar := m.helpbar.View(m.styles, m.screens[len(m.screens)-1], m.width)

	v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, header, "", content, "", helpbar))
	v.AltScreen = true
	return v
}
