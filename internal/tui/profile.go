package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"

	"github.com/armstrongl/nd/internal/profile"
)

type profileStep int

const (
	profileLoading profileStep = iota
	profileMenu
	profileList
	profileSwitch
	profileCreateName
	profileDone
)

// profileLoadedMsg carries the initial profile list and active profile.
type profileLoadedMsg struct {
	profiles []profile.ProfileSummary
	active   string
	err      error
}

// profileSwitchedMsg carries the result of a profile switch.
type profileSwitchedMsg struct {
	result *profile.SwitchResult
	err    error
}

// profileCreatedMsg carries the result of creating a new profile.
type profileCreatedMsg struct {
	name string
	err  error
}

// profileScreen provides Switch, Create, and List flows for profiles.
type profileScreen struct {
	svc    Services
	styles Styles
	isDark bool
	step   profileStep

	profiles []profile.ProfileSummary
	active   string
	err      error

	// menu
	menuForm   *huh.Form
	menuChoice string
	navigated  bool

	// switch
	switchForm   *huh.Form
	switchChoice string
	switching    bool

	// create
	createForm *huh.Form
	createName string

	// done
	doneMsg string
}

func newProfileScreen(svc Services, styles Styles, isDark bool) *profileScreen {
	return &profileScreen{svc: svc, styles: styles, isDark: isDark}
}

func (s *profileScreen) Title() string { return "Profiles" }

func (s *profileScreen) InputActive() bool {
	return s.step == profileMenu || s.step == profileSwitch || s.step == profileCreateName
}

func (s *profileScreen) Init() tea.Cmd {
	svc := s.svc
	return func() tea.Msg {
		mgr, err := svc.ProfileManager()
		if err != nil {
			return profileLoadedMsg{err: err}
		}
		if mgr == nil {
			return profileLoadedMsg{err: fmt.Errorf("profile manager not available")}
		}
		profiles, err := mgr.ListProfiles()
		if err != nil {
			return profileLoadedMsg{err: err}
		}
		active, _ := mgr.ActiveProfile()
		return profileLoadedMsg{profiles: profiles, active: active}
	}
}

func (s *profileScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case profileLoadedMsg:
		if msg.err != nil {
			s.err = msg.err
			s.step = profileDone
			return s, nil
		}
		s.profiles = msg.profiles
		s.active = msg.active
		return s.buildMenu()

	case profileSwitchedMsg:
		if msg.err != nil {
			s.doneMsg = fmt.Sprintf("%s Error: %s", s.styles.Danger.Render(GlyphBroken), msg.err.Error())
		} else {
			target := ""
			if msg.result != nil {
				target = msg.result.ToProfile
			}
			s.doneMsg = s.formatSwitchResult(target, msg.result)
		}
		s.step = profileDone
		return s, func() tea.Msg { return RefreshHeaderMsg{} }

	case profileCreatedMsg:
		if msg.err != nil {
			s.doneMsg = fmt.Sprintf("%s Error: %s", s.styles.Danger.Render(GlyphBroken), msg.err.Error())
		} else {
			s.doneMsg = fmt.Sprintf("%s Profile %q created.", s.styles.Success.Render(GlyphOK), msg.name)
		}
		s.step = profileDone
		return s, nil
	}

	switch s.step {
	case profileMenu:
		return s.updateMenu(msg)
	case profileSwitch:
		return s.updateSwitchForm(msg)
	case profileCreateName:
		return s.updateCreateForm(msg)
	case profileDone:
		return s.updateDone(msg)
	}
	return s, nil
}

func (s *profileScreen) View() tea.View {
	if s.step == profileLoading {
		return tea.NewView("  Loading profiles...")
	}
	if s.err != nil && s.step == profileDone {
		return tea.NewView(fmt.Sprintf("  %s\n\n  %s",
			s.styles.Danger.Render(s.err.Error()),
			s.styles.Subtle.Render("Press esc to go back.")))
	}
	switch s.step {
	case profileMenu:
		if s.menuForm != nil {
			return tea.NewView(s.menuForm.View())
		}
	case profileList:
		return s.viewList()
	case profileSwitch:
		if s.switchForm != nil {
			return tea.NewView(s.switchForm.View())
		}
	case profileCreateName:
		if s.createForm != nil {
			return tea.NewView(s.createForm.View())
		}
	case profileDone:
		return tea.NewView(fmt.Sprintf("  %s\n\n  %s",
			s.doneMsg, s.styles.Subtle.Render("Press enter to return.")))
	}
	return tea.NewView("")
}

// --- Step builders ---

func (s *profileScreen) buildMenu() (tea.Model, tea.Cmd) {
	s.step = profileMenu
	s.menuChoice = ""
	s.navigated = false
	opts := []huh.Option[string]{
		huh.NewOption("Switch profile", "switch"),
		huh.NewOption("Create profile", "create"),
		huh.NewOption("List profiles", "list"),
		huh.NewOption("Back", "back"),
	}
	s.menuForm = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Profiles").
				Options(opts...).
				Value(&s.menuChoice),
		),
	).WithTheme(huh.ThemeFunc(huh.ThemeCatppuccin))
	return s, s.menuForm.Init()
}

func (s *profileScreen) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	if s.navigated {
		return s, nil
	}
	model, cmd := s.menuForm.Update(msg)
	if f, ok := model.(*huh.Form); ok {
		s.menuForm = f
	}
	if s.menuForm.State == huh.StateCompleted {
		s.navigated = true
		switch s.menuChoice {
		case "switch":
			return s.buildSwitchForm()
		case "create":
			return s.buildCreateForm()
		case "list":
			s.step = profileList
			return s, nil
		default:
			return s, func() tea.Msg { return BackMsg{} }
		}
	}
	if s.menuForm.State == huh.StateAborted {
		return s, func() tea.Msg { return BackMsg{} }
	}
	return s, cmd
}

func (s *profileScreen) buildSwitchForm() (tea.Model, tea.Cmd) {
	s.step = profileSwitch
	s.switchChoice = ""
	s.switching = false

	if len(s.profiles) == 0 {
		s.doneMsg = "No profiles to switch to. Create one first."
		s.step = profileDone
		return s, nil
	}

	opts := make([]huh.Option[string], 0, len(s.profiles))
	for _, p := range s.profiles {
		label := p.Name
		if p.Name == s.active {
			label += " (active)"
		}
		opts = append(opts, huh.NewOption(label, p.Name))
	}

	s.switchForm = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Switch to profile").
				Options(opts...).
				Value(&s.switchChoice),
		),
	).WithTheme(huh.ThemeFunc(huh.ThemeCatppuccin))
	return s, s.switchForm.Init()
}

func (s *profileScreen) updateSwitchForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if s.switching {
		return s, nil
	}
	model, cmd := s.switchForm.Update(msg)
	if f, ok := model.(*huh.Form); ok {
		s.switchForm = f
	}
	if s.switchForm.State == huh.StateCompleted {
		s.switching = true
		return s, s.runSwitch()
	}
	if s.switchForm.State == huh.StateAborted {
		return s.buildMenu()
	}
	return s, cmd
}

func (s *profileScreen) runSwitch() tea.Cmd {
	target := s.switchChoice
	current := s.active
	svc := s.svc
	return func() tea.Msg {
		mgr, err := svc.ProfileManager()
		if err != nil {
			return profileSwitchedMsg{err: err}
		}
		if mgr == nil {
			return profileSwitchedMsg{err: fmt.Errorf("profile manager not available")}
		}
		eng, err := svc.DeployEngine()
		if err != nil {
			return profileSwitchedMsg{err: err}
		}
		if eng == nil {
			return profileSwitchedMsg{err: fmt.Errorf("deploy engine not available")}
		}
		summary, err := svc.ScanIndex()
		if err != nil {
			return profileSwitchedMsg{err: err}
		}
		if summary == nil || summary.Index == nil {
			return profileSwitchedMsg{err: fmt.Errorf("no asset index available")}
		}
		result, err := mgr.Switch(current, target, eng, summary.Index, svc.GetProjectRoot())
		return profileSwitchedMsg{result: result, err: err}
	}
}

func (s *profileScreen) buildCreateForm() (tea.Model, tea.Cmd) {
	s.step = profileCreateName
	s.createName = ""
	s.createForm = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("New profile name").
				Placeholder("my-profile").
				Validate(func(v string) error {
					if v == "" {
						return fmt.Errorf("name cannot be empty")
					}
					return nil
				}).
				Value(&s.createName),
		),
	).WithTheme(huh.ThemeFunc(huh.ThemeCatppuccin))
	return s, s.createForm.Init()
}

func (s *profileScreen) updateCreateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	model, cmd := s.createForm.Update(msg)
	if f, ok := model.(*huh.Form); ok {
		s.createForm = f
	}
	if s.createForm.State == huh.StateCompleted {
		return s, s.runCreate()
	}
	if s.createForm.State == huh.StateAborted {
		return s.buildMenu()
	}
	return s, cmd
}

func (s *profileScreen) runCreate() tea.Cmd {
	name := s.createName
	svc := s.svc
	return func() tea.Msg {
		pstore, err := svc.ProfileStore()
		if err != nil {
			return profileCreatedMsg{err: err}
		}
		if pstore == nil {
			return profileCreatedMsg{err: fmt.Errorf("profile store not available")}
		}
		newProfile := profile.Profile{Name: name}
		return profileCreatedMsg{name: name, err: pstore.CreateProfile(newProfile)}
	}
}

func (s *profileScreen) updateDone(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok && keyMsg.String() == "enter" {
		return s.buildMenu()
	}
	return s, nil
}

func (s *profileScreen) viewList() tea.View {
	if len(s.profiles) == 0 {
		return tea.NewView("  " + NoProfiles())
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("  %s\n\n", s.styles.Bold.Render("Profiles")))
	for _, p := range s.profiles {
		marker := " "
		if p.Name == s.active {
			marker = "*"
		}
		b.WriteString(fmt.Sprintf("  %s  %-30s  %s\n",
			marker, p.Name,
			s.styles.Subtle.Render(fmt.Sprintf("%d assets", p.AssetCount))))
	}
	b.WriteString(fmt.Sprintf("\n  %s", s.styles.Subtle.Render("Press esc to go back.")))
	return tea.NewView(b.String())
}

func (s *profileScreen) formatSwitchResult(target string, result *profile.SwitchResult) string {
	if result == nil {
		return fmt.Sprintf("%s Switched to %q.", s.styles.Success.Render(GlyphOK), target)
	}
	to := result.ToProfile
	if to == "" {
		to = target
	}
	deployed := 0
	removed := 0
	if result.Deployed != nil {
		deployed = len(result.Deployed.Succeeded)
	}
	if result.Removed != nil {
		removed = len(result.Removed.Succeeded)
	}
	return fmt.Sprintf("%s Switched to %q. Deployed: %d, Removed: %d.",
		s.styles.Success.Render(GlyphOK), to, deployed, removed)
}
