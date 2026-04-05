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

func (s *scopeScreen) Title() string    { return "Switch Scope" }
func (s *scopeScreen) InputActive() bool { return s.step == scopeFormStep && !s.navigated }

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

	// Project scope requires a project root.
	if newScope == nd.ScopeProject && s.svc.GetProjectRoot() == "" {
		s.errorMsg = "Cannot switch to project scope: no project root detected."
		s.step = scopeShowError
		return nil
	}

	projectRoot := s.svc.GetProjectRoot()
	s.svc.ResetForScope(newScope, projectRoot)

	return tea.Batch(
		func() tea.Msg { return ScopeSwitchedMsg{} },
		func() tea.Msg { return RefreshHeaderMsg{} },
		func() tea.Msg { return PopToRootMsg{} },
	)
}
