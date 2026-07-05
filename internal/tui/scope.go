package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/armstrongl/nd/internal/nd"
)

type scopeStep int

const (
	scopeFormStep scopeStep = iota
	scopeShowError
)

type scopeScreen struct {
	svc       Services
	form      *huh.Form
	choice    string
	styles    Styles
	isDark    bool
	navigated bool
	step      scopeStep
	errorMsg  string
}

func newScopeScreen(svc Services, styles Styles, isDark bool) *scopeScreen {
	s := &scopeScreen{
		svc:    svc,
		styles: styles,
		isDark: isDark,
	}

	current := string(svc.GetScope())
	s.choice = current

	s.form = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Switch scope").
				Options(
					huh.NewOption("Global", "global"),
					huh.NewOption("Project", "project"),
				).
				Value(&s.choice),
		),
	).WithTheme(huh.ThemeFunc(huh.ThemeCatppuccin))

	return s
}

// newScopeErrorScreen builds a scopeScreen already parked on its error step,
// rendering msg with a "Press enter to return." hint. Used by the inline Ctrl+S
// toggle (Model.toggleScope) to surface a project-root resolution failure with a
// visible message instead of silently doing nothing.
func newScopeErrorScreen(svc Services, styles Styles, isDark bool, msg string) *scopeScreen {
	s := newScopeScreen(svc, styles, isDark)
	s.step = scopeShowError
	s.errorMsg = msg
	return s
}

func (s *scopeScreen) Title() string    { return "Switch Scope" }
func (s *scopeScreen) InputActive() bool { return s.step == scopeFormStep && !s.navigated }

// FullHelpItems returns step-specific keybindings for the help bar and overlay.
func (s *scopeScreen) FullHelpItems() []HelpItem {
	switch s.step {
	case scopeShowError:
		return []HelpItem{
			{"enter", "return"},
			{"q", "quit"},
		}
	default: // scopeFormStep
		return []HelpItem{
			{"esc", "back"},
			{"j/k", "navigate"},
			{"enter", "select"},
			{"q", "quit"},
		}
	}
}

func (s *scopeScreen) Init() tea.Cmd {
	return s.form.Init()
}

func (s *scopeScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch s.step {
	case scopeShowError:
		if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
			if keyMsg.String() == "enter" {
				return s, func() tea.Msg { return PopToRootMsg{} }
			}
		}
		return s, nil

	default: // scopeFormStep
		if s.navigated {
			return s, nil
		}

		model, cmd := s.form.Update(msg)
		if f, ok := model.(*huh.Form); ok {
			s.form = f
		}

		if s.form.State == huh.StateCompleted {
			s.navigated = true
			return s, s.handleScopeSelection()
		}

		return s, cmd
	}
}

func (s *scopeScreen) View() tea.View {
	switch s.step {
	case scopeShowError:
		return tea.NewView(fmt.Sprintf("  %s\n\n  %s",
			s.errorMsg,
			s.styles.Subtle.Render("Press enter to return.")))
	default:
		return tea.NewView(s.form.View())
	}
}

func (s *scopeScreen) handleScopeSelection() tea.Cmd {
	newScope := nd.Scope(s.choice)

	projectRoot := s.svc.GetProjectRoot()
	// Project scope requires a project root; resolve it on demand from cwd so
	// switching works even when the TUI was launched in global scope.
	if newScope == nd.ScopeProject {
		root, err := s.svc.ResolveProjectRoot()
		if err != nil {
			s.errorMsg = fmt.Sprintf("Cannot switch to project scope: %v", err)
			s.step = scopeShowError
			return nil
		}
		if root == "" {
			s.errorMsg = "Cannot switch to project scope: no project root detected."
			s.step = scopeShowError
			return nil
		}
		projectRoot = root
	}

	s.svc.ResetForScope(newScope, projectRoot)

	return tea.Batch(
		func() tea.Msg { return ScopeSwitchedMsg{} },
		func() tea.Msg { return RefreshHeaderMsg{} },
		func() tea.Msg { return PopToRootMsg{} },
	)
}
