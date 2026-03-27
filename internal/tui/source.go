package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"

	"github.com/armstrongl/nd/internal/source"
)

type sourceStep int

const (
	sourceLoading sourceStep = iota
	sourceMenu
	sourceList
	sourceAddLocalInput
	sourceAddGitInput
	sourceRemoveSelect
	sourceRemoveConfirm
	sourceSyncing
	sourceDone
)

// sourceLoadedMsg carries the initial source list.
type sourceLoadedMsg struct {
	sources []source.Source
	err     error
}

// sourceAddedMsg carries the result of adding a source.
type sourceAddedMsg struct {
	src *source.Source
	err error
}

// sourceRemovedMsg carries the result of removing a source.
type sourceRemovedMsg struct {
	id  string
	err error
}

// sourceSyncedMsg carries the result of syncing all sources.
type sourceSyncedMsg struct {
	synced int
	errors []error
}

// sourceScreen provides List, Add, Remove, and Sync flows for sources.
type sourceScreen struct {
	svc    Services
	styles Styles
	isDark bool
	step   sourceStep

	sources []source.Source
	err     error

	// menu
	menuForm   *huh.Form
	menuChoice string
	navigated  bool

	// add forms
	addForm    *huh.Form
	addInput   string

	// remove
	removeForm    *huh.Form
	removeChoice  string
	confirmForm   *huh.Form
	confirmed     bool
	removing      bool

	// done
	doneMsg string
}

func newSourceScreen(svc Services, styles Styles, isDark bool) *sourceScreen {
	return &sourceScreen{svc: svc, styles: styles, isDark: isDark}
}

func (s *sourceScreen) Title() string { return "Sources" }

func (s *sourceScreen) InputActive() bool {
	return s.step == sourceMenu || s.step == sourceAddLocalInput || s.step == sourceAddGitInput || s.step == sourceRemoveSelect || s.step == sourceRemoveConfirm
}

func (s *sourceScreen) Init() tea.Cmd {
	svc := s.svc
	return func() tea.Msg {
		sm, err := svc.SourceManager()
		if err != nil {
			return sourceLoadedMsg{err: err}
		}
		if sm == nil {
			return sourceLoadedMsg{err: fmt.Errorf("source manager not available")}
		}
		return sourceLoadedMsg{sources: sm.Sources()}
	}
}

func (s *sourceScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case sourceLoadedMsg:
		if msg.err != nil {
			s.err = msg.err
			s.step = sourceDone
			return s, nil
		}
		s.sources = msg.sources
		return s.buildMenu()

	case sourceAddedMsg:
		if msg.err != nil {
			s.doneMsg = fmt.Sprintf("%s Error: %s", s.styles.Danger.Render(GlyphBroken), msg.err.Error())
		} else {
			s.doneMsg = fmt.Sprintf("%s Source %q added.", s.styles.Success.Render(GlyphOK), msg.src.ID)
		}
		s.step = sourceDone
		return s, func() tea.Msg { return RefreshHeaderMsg{} }

	case sourceRemovedMsg:
		if msg.err != nil {
			s.doneMsg = fmt.Sprintf("%s Error: %s", s.styles.Danger.Render(GlyphBroken), msg.err.Error())
		} else {
			s.doneMsg = fmt.Sprintf("%s Source %q removed.", s.styles.Success.Render(GlyphOK), msg.id)
		}
		s.step = sourceDone
		return s, func() tea.Msg { return RefreshHeaderMsg{} }

	case sourceSyncedMsg:
		s.step = sourceDone
		s.doneMsg = s.formatSyncResult(msg)
		return s, func() tea.Msg { return RefreshHeaderMsg{} }
	}

	switch s.step {
	case sourceMenu:
		return s.updateMenu(msg)
	case sourceAddLocalInput, sourceAddGitInput:
		return s.updateAddForm(msg)
	case sourceRemoveSelect, sourceRemoveConfirm:
		return s.updateRemove(msg)
	case sourceDone:
		return s.updateDone(msg)
	}
	return s, nil
}

func (s *sourceScreen) View() tea.View {
	if s.step == sourceLoading {
		return tea.NewView("  Loading sources...")
	}
	if s.err != nil && s.step == sourceDone {
		return tea.NewView(fmt.Sprintf("  %s\n\n  %s",
			s.styles.Danger.Render(s.err.Error()),
			s.styles.Subtle.Render("Press esc to go back.")))
	}
	switch s.step {
	case sourceMenu:
		if s.menuForm != nil {
			return tea.NewView(s.menuForm.View())
		}
	case sourceList:
		return s.viewList()
	case sourceAddLocalInput, sourceAddGitInput:
		if s.addForm != nil {
			return tea.NewView(s.addForm.View())
		}
	case sourceRemoveSelect:
		if s.removeForm != nil {
			return tea.NewView(s.removeForm.View())
		}
	case sourceRemoveConfirm:
		if s.confirmForm != nil {
			return tea.NewView(s.confirmForm.View())
		}
	case sourceSyncing:
		return tea.NewView(fmt.Sprintf("  %s", s.styles.Primary.Render("Syncing sources...")))
	case sourceDone:
		return tea.NewView(fmt.Sprintf("  %s\n\n  %s",
			s.doneMsg, s.styles.Subtle.Render("Press enter to return.")))
	}
	return tea.NewView("")
}

// --- Step builders ---

func (s *sourceScreen) buildMenu() (tea.Model, tea.Cmd) {
	s.step = sourceMenu
	s.menuChoice = ""
	s.navigated = false
	s.menuForm = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Sources").
				Options(
					huh.NewOption("List sources", "list"),
					huh.NewOption("Add local source", "add-local"),
					huh.NewOption("Add Git source", "add-git"),
					huh.NewOption("Remove source", "remove"),
					huh.NewOption("Sync all sources", "sync"),
					huh.NewOption("Back", "back"),
				).
				Value(&s.menuChoice),
		),
	).WithTheme(huh.ThemeFunc(huh.ThemeCatppuccin))
	return s, s.menuForm.Init()
}

func (s *sourceScreen) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		case "list":
			s.step = sourceList
			return s, nil
		case "add-local":
			return s.buildAddForm("local")
		case "add-git":
			return s.buildAddForm("git")
		case "remove":
			return s.buildRemoveForm()
		case "sync":
			return s.startSync()
		default:
			return s, func() tea.Msg { return BackMsg{} }
		}
	}
	if s.menuForm.State == huh.StateAborted {
		return s, func() tea.Msg { return BackMsg{} }
	}
	return s, cmd
}

func (s *sourceScreen) buildAddForm(kind string) (tea.Model, tea.Cmd) {
	s.addInput = ""
	title := "Local source path"
	placeholder := "/path/to/assets"
	if kind == "git" {
		s.step = sourceAddGitInput
		title = "Git repository URL"
		placeholder = "https://github.com/org/nd-assets"
	} else {
		s.step = sourceAddLocalInput
	}
	s.addForm = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(title).
				Placeholder(placeholder).
				Validate(func(v string) error {
					if v == "" {
						return fmt.Errorf("cannot be empty")
					}
					return nil
				}).
				Value(&s.addInput),
		),
	).WithTheme(huh.ThemeFunc(huh.ThemeCatppuccin))
	return s, s.addForm.Init()
}

func (s *sourceScreen) updateAddForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	model, cmd := s.addForm.Update(msg)
	if f, ok := model.(*huh.Form); ok {
		s.addForm = f
	}
	if s.addForm.State == huh.StateCompleted {
		kind := "local"
		if s.step == sourceAddGitInput {
			kind = "git"
		}
		return s, s.runAdd(kind, s.addInput)
	}
	if s.addForm.State == huh.StateAborted {
		return s.buildMenu()
	}
	return s, cmd
}

func (s *sourceScreen) runAdd(kind, input string) tea.Cmd {
	svc := s.svc
	return func() tea.Msg {
		sm, err := svc.SourceManager()
		if err != nil {
			return sourceAddedMsg{err: err}
		}
		if sm == nil {
			return sourceAddedMsg{err: fmt.Errorf("source manager not available")}
		}
		if kind == "git" {
			src, err := sm.AddGit(input, "")
			return sourceAddedMsg{src: src, err: err}
		}
		src, err := sm.AddLocal(input, "")
		return sourceAddedMsg{src: src, err: err}
	}
}

func (s *sourceScreen) buildRemoveForm() (tea.Model, tea.Cmd) {
	s.step = sourceRemoveSelect
	s.removeChoice = ""
	s.removing = false

	if len(s.sources) == 0 {
		s.doneMsg = "No sources to remove."
		s.step = sourceDone
		return s, nil
	}

	opts := make([]huh.Option[string], len(s.sources))
	for i, src := range s.sources {
		label := src.ID
		if src.URL != "" {
			label = fmt.Sprintf("%s  (%s)", src.ID, src.URL)
		} else if src.Path != "" {
			label = fmt.Sprintf("%s  (%s)", src.ID, src.Path)
		}
		opts[i] = huh.NewOption(label, src.ID)
	}
	s.removeForm = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select source to remove").
				Options(opts...).
				Value(&s.removeChoice),
		),
	).WithTheme(huh.ThemeFunc(huh.ThemeCatppuccin))
	return s, s.removeForm.Init()
}

func (s *sourceScreen) updateRemove(msg tea.Msg) (tea.Model, tea.Cmd) {
	if s.removing {
		return s, nil
	}

	if s.step == sourceRemoveSelect {
		model, cmd := s.removeForm.Update(msg)
		if f, ok := model.(*huh.Form); ok {
			s.removeForm = f
		}
		if s.removeForm.State == huh.StateCompleted {
			return s.buildRemoveConfirm()
		}
		if s.removeForm.State == huh.StateAborted {
			return s.buildMenu()
		}
		return s, cmd
	}

	// sourceRemoveConfirm
	if s.confirmForm == nil {
		return s, nil
	}
	model, cmd := s.confirmForm.Update(msg)
	if f, ok := model.(*huh.Form); ok {
		s.confirmForm = f
	}
	if s.confirmForm.State == huh.StateCompleted {
		if !s.confirmed {
			return s.buildMenu()
		}
		s.removing = true
		return s, s.runRemove()
	}
	if s.confirmForm.State == huh.StateAborted {
		return s.buildMenu()
	}
	return s, cmd
}

func (s *sourceScreen) buildRemoveConfirm() (tea.Model, tea.Cmd) {
	s.step = sourceRemoveConfirm
	s.confirmed = false
	title := fmt.Sprintf("Remove source %q?", s.removeChoice)
	s.confirmForm = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(title).
				Affirmative("Remove").
				Negative("Cancel").
				Value(&s.confirmed),
		),
	).WithTheme(huh.ThemeFunc(huh.ThemeCatppuccin))
	return s, s.confirmForm.Init()
}

func (s *sourceScreen) runRemove() tea.Cmd {
	id := s.removeChoice
	svc := s.svc
	return func() tea.Msg {
		sm, err := svc.SourceManager()
		if err != nil {
			return sourceRemovedMsg{err: err}
		}
		if sm == nil {
			return sourceRemovedMsg{err: fmt.Errorf("source manager not available")}
		}
		return sourceRemovedMsg{id: id, err: sm.Remove(id)}
	}
}

func (s *sourceScreen) startSync() (tea.Model, tea.Cmd) {
	s.step = sourceSyncing
	sources := s.sources
	svc := s.svc
	return s, func() tea.Msg {
		sm, err := svc.SourceManager()
		if err != nil {
			return sourceSyncedMsg{errors: []error{err}}
		}
		if sm == nil {
			return sourceSyncedMsg{errors: []error{fmt.Errorf("source manager not available")}}
		}
		var errs []error
		synced := 0
		for _, src := range sources {
			if err := sm.SyncSource(src.ID); err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", src.ID, err))
			} else {
				synced++
			}
		}
		return sourceSyncedMsg{synced: synced, errors: errs}
	}
}

func (s *sourceScreen) updateDone(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok && keyMsg.String() == "enter" {
		return s.buildMenu()
	}
	return s, nil
}

func (s *sourceScreen) viewList() tea.View {
	if len(s.sources) == 0 {
		return tea.NewView("  " + NoSources())
	}
	var b strings.Builder
	fmt.Fprintf(&b, "  %s\n\n", s.styles.Bold.Render("Sources"))
	for _, src := range s.sources {
		loc := src.Path
		if src.URL != "" {
			loc = src.URL
		}
		fmt.Fprintf(&b, "  %-20s  %s\n",
			src.ID, s.styles.Subtle.Render(loc))
	}
	fmt.Fprintf(&b, "\n  %s", s.styles.Subtle.Render("Press esc to go back."))
	return tea.NewView(b.String())
}

func (s *sourceScreen) formatSyncResult(msg sourceSyncedMsg) string {
	if len(msg.errors) == 0 {
		return fmt.Sprintf("%s Synced %d source(s).", s.styles.Success.Render(GlyphOK), msg.synced)
	}
	var b strings.Builder
	if msg.synced > 0 {
		fmt.Fprintf(&b, "%s Synced %d source(s).\n", s.styles.Success.Render(GlyphOK), msg.synced)
	}
	for _, err := range msg.errors {
		fmt.Fprintf(&b, "  %s %s\n", s.styles.Danger.Render(GlyphBroken), err.Error())
	}
	return strings.TrimRight(b.String(), "\n")
}
