package tui

import (
	"fmt"
	"os/exec"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"

	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/version"
)

type settingsStep int

const (
	settingsMenu settingsStep = iota
	settingsShowResult
	settingsSwitchScope
)

// settingsActionMsg is sent internally when the menu form completes.
type settingsActionMsg struct{ action string }

// settingsScopeSelectedMsg is sent when the scope form completes.
type settingsScopeSelectedMsg struct{ scope nd.Scope }

// editorFinishedMsg is sent when the external editor process exits.
type editorFinishedMsg struct{ err error }

// settingsScreen provides a submenu for editing config, showing info, and switching scope.
type settingsScreen struct {
	svc    Services
	styles Styles
	isDark bool
	step   settingsStep

	// menu
	form   *huh.Form
	choice string

	// show result
	result string

	// scope switch
	scopeForm  *huh.Form
	scopeValue string
	navigated  bool
}

func newSettingsScreen(svc Services, styles Styles, isDark bool) *settingsScreen {
	s := &settingsScreen{svc: svc, styles: styles, isDark: isDark}
	s.buildMenu()
	return s
}

func (s *settingsScreen) buildMenu() {
	s.choice = ""
	s.navigated = false
	s.form = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Settings").
				Options(
					huh.NewOption("Edit config", "edit"),
					huh.NewOption("Show config path", "path"),
					huh.NewOption("Show version", "version"),
					huh.NewOption("Switch scope", "scope"),
					huh.NewOption("Back", "back"),
				).
				Value(&s.choice),
		),
	).WithTheme(huh.ThemeFunc(huh.ThemeCatppuccin))
}

func (s *settingsScreen) Title() string    { return "Settings" }
func (s *settingsScreen) InputActive() bool {
	return s.step == settingsMenu || s.step == settingsSwitchScope
}

func (s *settingsScreen) Init() tea.Cmd {
	return s.form.Init()
}

func (s *settingsScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case settingsActionMsg:
		return s.handleAction(msg.action)

	case editorFinishedMsg:
		if msg.err != nil {
			s.result = fmt.Sprintf("Editor error: %s", msg.err)
			s.step = settingsShowResult
			return s, nil
		}
		s.step = settingsMenu
		s.buildMenu()
		return s, s.form.Init()

	case settingsScopeSelectedMsg:
		// Project scope requires a project root.
		if msg.scope == nd.ScopeProject && s.svc.GetProjectRoot() == "" {
			s.result = "Cannot switch to project scope: no project root detected."
			s.step = settingsShowResult
			return s, nil
		}
		s.svc.ResetForScope(msg.scope, s.svc.GetProjectRoot())
		s.result = fmt.Sprintf("Scope switched to %q.", msg.scope)
		s.step = settingsShowResult
		return s, tea.Batch(
			func() tea.Msg { return ScopeSwitchedMsg{} },
			func() tea.Msg { return RefreshHeaderMsg{} },
		)
	}

	switch s.step {
	case settingsMenu:
		return s.updateMenu(msg)
	case settingsSwitchScope:
		return s.updateScopeForm(msg)
	case settingsShowResult:
		return s.updateResult(msg)
	}
	return s, nil
}

func (s *settingsScreen) View() tea.View {
	switch s.step {
	case settingsMenu:
		if s.form != nil {
			return tea.NewView(s.form.View())
		}
	case settingsShowResult:
		return tea.NewView(fmt.Sprintf("  %s\n\n  %s",
			s.result,
			s.styles.Subtle.Render("Press enter to return.")))
	case settingsSwitchScope:
		if s.scopeForm != nil {
			return tea.NewView(s.scopeForm.View())
		}
	}
	return tea.NewView("")
}

// --- Step handlers ---

func (s *settingsScreen) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	if s.navigated {
		return s, nil
	}
	model, cmd := s.form.Update(msg)
	if f, ok := model.(*huh.Form); ok {
		s.form = f
	}
	if s.form.State == huh.StateCompleted {
		s.navigated = true
		return s.handleAction(s.choice)
	}
	if s.form.State == huh.StateAborted {
		return s, func() tea.Msg { return BackMsg{} }
	}
	return s, cmd
}

func (s *settingsScreen) handleAction(action string) (tea.Model, tea.Cmd) {
	switch action {
	case "edit":
		configPath := s.svc.GetConfigPath()
		editorEnv := "EDITOR"
		editorCmd := exec.Command("sh", "-c", fmt.Sprintf("${%s:-vi} %q", editorEnv, configPath))
		return s, tea.ExecProcess(editorCmd, func(err error) tea.Msg {
			return editorFinishedMsg{err: err}
		})
	case "path":
		s.result = fmt.Sprintf("Config path: %s", s.svc.GetConfigPath())
		s.step = settingsShowResult
	case "version":
		s.result = version.String()
		s.step = settingsShowResult
	case "scope":
		return s.buildScopeForm()
	case "back":
		return s, func() tea.Msg { return BackMsg{} }
	}
	return s, nil
}

func (s *settingsScreen) buildScopeForm() (tea.Model, tea.Cmd) {
	s.step = settingsSwitchScope
	s.scopeValue = string(s.svc.GetScope())

	s.scopeForm = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Switch scope").
				Options(
					huh.NewOption("Global", string(nd.ScopeGlobal)),
					huh.NewOption("Project", string(nd.ScopeProject)),
				).
				Value(&s.scopeValue),
		),
	).WithTheme(huh.ThemeFunc(huh.ThemeCatppuccin))

	return s, s.scopeForm.Init()
}

func (s *settingsScreen) updateScopeForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if s.scopeForm == nil {
		return s, nil
	}
	model, cmd := s.scopeForm.Update(msg)
	if f, ok := model.(*huh.Form); ok {
		s.scopeForm = f
	}
	if s.scopeForm.State == huh.StateCompleted {
		return s, func() tea.Msg {
			return settingsScopeSelectedMsg{scope: nd.Scope(s.scopeValue)}
		}
	}
	if s.scopeForm.State == huh.StateAborted {
		s.step = settingsMenu
		s.buildMenu()
		return s, s.form.Init()
	}
	return s, cmd
}

func (s *settingsScreen) updateResult(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		if keyMsg.String() == "enter" {
			// Return to menu.
			s.step = settingsMenu
			s.buildMenu()
			return s, s.form.Init()
		}
	}
	return s, nil
}
