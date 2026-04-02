package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"

	"github.com/armstrongl/nd/internal/profile"
)

type snapshotStep int

const (
	snapshotLoading snapshotStep = iota
	snapshotMenu
	snapshotSaveName
	snapshotRestoreSelect
	snapshotList
	snapshotDone
)

// snapshotLoadedMsg carries the initial snapshot list.
type snapshotLoadedMsg struct {
	snapshots []profile.SnapshotSummary
	err       error
}

// snapshotSavedMsg carries the result of saving a snapshot.
type snapshotSavedMsg struct {
	name string
	err  error
}

// snapshotRestoredMsg carries the result of restoring a snapshot.
type snapshotRestoredMsg struct {
	result *profile.RestoreResult
	err    error
}

// snapshotScreen provides Save, Restore, and List flows for snapshots.
type snapshotScreen struct {
	svc    Services
	styles Styles
	isDark bool
	step   snapshotStep

	snapshots []profile.SnapshotSummary
	err       error

	// menu
	menuForm   *huh.Form
	menuChoice string
	navigated  bool

	// save
	saveForm *huh.Form
	saveName string

	// restore
	restoreForm   *huh.Form
	restoreChoice string
	confirmForm   *huh.Form
	confirmed     bool
	fixing        bool

	// result
	doneMsg string

	// snapshotList scrolling
	listLines []string
	height    int
	scroll    listScroll
}

func newSnapshotScreen(svc Services, styles Styles, isDark bool) *snapshotScreen {
	return &snapshotScreen{svc: svc, styles: styles, isDark: isDark}
}

func (s *snapshotScreen) Title() string { return "Snapshots" }

// InputActive returns true only during the snapshot name input.
func (s *snapshotScreen) InputActive() bool {
	return s.step == snapshotMenu || s.step == snapshotSaveName || s.step == snapshotRestoreSelect
}

func (s *snapshotScreen) Init() tea.Cmd {
	svc := s.svc
	return func() tea.Msg {
		mgr, err := svc.ProfileManager()
		if err != nil {
			return snapshotLoadedMsg{err: err}
		}
		if mgr == nil {
			return snapshotLoadedMsg{err: fmt.Errorf("profile manager not available")}
		}
		snaps, err := mgr.ListSnapshots()
		return snapshotLoadedMsg{snapshots: snaps, err: err}
	}
}

func (s *snapshotScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.height = msg.Height
		return s, nil

	case snapshotLoadedMsg:
		if msg.err != nil {
			s.err = msg.err
			s.step = snapshotDone
			return s, nil
		}
		s.snapshots = msg.snapshots
		return s.buildMenu()

	case snapshotSavedMsg:
		if msg.err != nil {
			s.doneMsg = fmt.Sprintf("%s Error: %s", s.styles.Danger.Render(GlyphBroken), msg.err.Error())
		} else {
			s.doneMsg = fmt.Sprintf("%s Snapshot %q saved.", s.styles.Success.Render(GlyphOK), msg.name)
		}
		s.step = snapshotDone
		return s, nil

	case snapshotRestoredMsg:
		if msg.err != nil {
			s.doneMsg = fmt.Sprintf("%s Error: %s", s.styles.Danger.Render(GlyphBroken), msg.err.Error())
			s.step = snapshotDone
			return s, nil
		}
		name := ""
		if msg.result != nil {
			name = msg.result.SnapshotName
		}
		s.doneMsg = fmt.Sprintf("%s Snapshot %q restored.", s.styles.Success.Render(GlyphOK), name)
		s.step = snapshotDone
		return s, func() tea.Msg { return RefreshHeaderMsg{} }
	}

	switch s.step {
	case snapshotMenu:
		return s.updateMenu(msg)
	case snapshotList:
		return s.updateList(msg)
	case snapshotSaveName:
		return s.updateSaveForm(msg)
	case snapshotRestoreSelect:
		return s.updateRestoreSelect(msg)
	case snapshotDone:
		return s.updateDone(msg)
	}
	return s, nil
}

func (s *snapshotScreen) View() tea.View {
	if s.step == snapshotLoading {
		return tea.NewView("  Loading snapshots...")
	}
	if s.err != nil && s.step == snapshotDone {
		return tea.NewView(fmt.Sprintf("  %s\n\n  %s",
			s.styles.Danger.Render(s.err.Error()),
			s.styles.Subtle.Render("Press esc to go back.")))
	}
	switch s.step {
	case snapshotMenu:
		if s.menuForm != nil {
			return tea.NewView(s.menuForm.View())
		}
	case snapshotSaveName:
		if s.saveForm != nil {
			return tea.NewView(s.saveForm.View())
		}
	case snapshotRestoreSelect:
		if s.restoreForm != nil {
			return tea.NewView(s.restoreForm.View())
		}
		if s.confirmForm != nil {
			return tea.NewView(s.confirmForm.View())
		}
	case snapshotList:
		return s.viewList()
	case snapshotDone:
		return tea.NewView(fmt.Sprintf("  %s\n\n  %s",
			s.doneMsg, s.styles.Subtle.Render("Press enter to return.")))
	}
	return tea.NewView("")
}

// --- Step builders ---

func (s *snapshotScreen) buildMenu() (tea.Model, tea.Cmd) {
	s.step = snapshotMenu
	s.menuChoice = ""
	s.navigated = false
	s.menuForm = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Snapshots").
				Options(
					huh.NewOption("Save snapshot", "save"),
					huh.NewOption("Restore snapshot", "restore"),
					huh.NewOption("List snapshots", "list"),
					huh.NewOption("Back", "back"),
				).
				Value(&s.menuChoice),
		),
	).WithTheme(huh.ThemeFunc(huh.ThemeCatppuccin))
	return s, s.menuForm.Init()
}

func (s *snapshotScreen) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		case "save":
			return s.buildSaveForm()
		case "restore":
			return s.buildRestoreForm()
		case "list":
			s.step = snapshotList
			s.scroll = listScroll{}
			s.listLines = splitLines(s.buildListContent())
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

func (s *snapshotScreen) buildSaveForm() (tea.Model, tea.Cmd) {
	s.step = snapshotSaveName
	s.saveName = ""
	s.saveForm = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Snapshot name").
				Placeholder("my-snapshot").
				Validate(func(v string) error {
					if v == "" {
						return fmt.Errorf("name cannot be empty")
					}
					return nil
				}).
				Value(&s.saveName),
		),
	).WithTheme(huh.ThemeFunc(huh.ThemeCatppuccin))
	return s, s.saveForm.Init()
}

func (s *snapshotScreen) updateSaveForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	model, cmd := s.saveForm.Update(msg)
	if f, ok := model.(*huh.Form); ok {
		s.saveForm = f
	}
	if s.saveForm.State == huh.StateCompleted {
		return s, s.runSave()
	}
	if s.saveForm.State == huh.StateAborted {
		return s.buildMenu()
	}
	return s, cmd
}

func (s *snapshotScreen) runSave() tea.Cmd {
	name := s.saveName
	svc := s.svc
	return func() tea.Msg {
		mgr, err := svc.ProfileManager()
		if err != nil {
			return snapshotSavedMsg{err: err}
		}
		if mgr == nil {
			return snapshotSavedMsg{err: fmt.Errorf("profile manager not available")}
		}
		return snapshotSavedMsg{name: name, err: mgr.SaveSnapshot(name)}
	}
}

func (s *snapshotScreen) buildRestoreForm() (tea.Model, tea.Cmd) {
	s.step = snapshotRestoreSelect
	s.restoreChoice = ""
	s.fixing = false
	s.confirmForm = nil

	if len(s.snapshots) == 0 {
		s.doneMsg = "No snapshots available."
		s.step = snapshotDone
		return s, nil
	}

	opts := make([]huh.Option[string], len(s.snapshots))
	for i, snap := range s.snapshots {
		label := fmt.Sprintf("%s  (%d deployments)", snap.Name, snap.DeploymentCount)
		opts[i] = huh.NewOption(label, snap.Name)
	}

	s.restoreForm = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select snapshot to restore").
				Options(opts...).
				Value(&s.restoreChoice),
		),
	).WithTheme(huh.ThemeFunc(huh.ThemeCatppuccin))
	return s, s.restoreForm.Init()
}

func (s *snapshotScreen) updateRestoreSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	if s.fixing {
		return s, nil
	}

	// Phase 1: selecting snapshot.
	if s.confirmForm == nil {
		if s.restoreForm == nil {
			return s, nil
		}
		model, cmd := s.restoreForm.Update(msg)
		if f, ok := model.(*huh.Form); ok {
			s.restoreForm = f
		}
		if s.restoreForm.State == huh.StateCompleted {
			return s.buildRestoreConfirm()
		}
		if s.restoreForm.State == huh.StateAborted {
			return s.buildMenu()
		}
		return s, cmd
	}

	// Phase 2: confirming.
	model, cmd := s.confirmForm.Update(msg)
	if f, ok := model.(*huh.Form); ok {
		s.confirmForm = f
	}
	if s.confirmForm.State == huh.StateCompleted {
		if !s.confirmed {
			return s.buildMenu()
		}
		s.fixing = true
		return s, s.runRestore()
	}
	if s.confirmForm.State == huh.StateAborted {
		return s.buildMenu()
	}
	return s, cmd
}

func (s *snapshotScreen) buildRestoreConfirm() (tea.Model, tea.Cmd) {
	s.confirmed = false
	title := fmt.Sprintf("Restore %q? This will overwrite current deployments.", s.restoreChoice)
	s.confirmForm = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(title).
				Affirmative("Restore").
				Negative("Cancel").
				Value(&s.confirmed),
		),
	).WithTheme(huh.ThemeFunc(huh.ThemeCatppuccin))
	return s, s.confirmForm.Init()
}

func (s *snapshotScreen) runRestore() tea.Cmd {
	snapName := s.restoreChoice
	svc := s.svc
	return func() tea.Msg {
		mgr, err := svc.ProfileManager()
		if err != nil {
			return snapshotRestoredMsg{err: err}
		}
		if mgr == nil {
			return snapshotRestoredMsg{err: fmt.Errorf("profile manager not available")}
		}
		eng, err := svc.DeployEngine()
		if err != nil {
			return snapshotRestoredMsg{err: err}
		}
		if eng == nil {
			return snapshotRestoredMsg{err: fmt.Errorf("deploy engine not available")}
		}
		summary, err := svc.ScanIndex()
		if err != nil {
			return snapshotRestoredMsg{err: err}
		}
		if summary == nil || summary.Index == nil {
			return snapshotRestoredMsg{err: fmt.Errorf("no asset index available")}
		}
		result, err := mgr.Restore(snapName, eng, summary.Index)
		return snapshotRestoredMsg{result: result, err: err}
	}
}

func (s *snapshotScreen) updateDone(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok && keyMsg.String() == "enter" {
		return s.buildMenu()
	}
	return s, nil
}

func (s *snapshotScreen) contentHeight() int {
	if s.height == 0 {
		return listScrollUnlimited
	}
	h := s.height - 4
	if h < 3 {
		h = 3
	}
	return h
}

func (s *snapshotScreen) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyMsg.String() {
		case "j", "down":
			s.scroll.ScrollDown(len(s.listLines), s.contentHeight())
		case "k", "up":
			s.scroll.ScrollUp()
		}
	}
	return s, nil
}

func (s *snapshotScreen) buildListContent() string {
	if len(s.snapshots) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "  %s\n\n", s.styles.Bold.Render("Snapshots"))
	for _, snap := range s.snapshots {
		fmt.Fprintf(&b, "  %-30s  %s deployments\n",
			snap.Name,
			s.styles.Subtle.Render(fmt.Sprintf("%d", snap.DeploymentCount)))
	}
	fmt.Fprintf(&b, "\n  %s", s.styles.Subtle.Render("Press esc to go back."))
	return b.String()
}

func (s *snapshotScreen) viewList() tea.View {
	if len(s.snapshots) == 0 {
		return tea.NewView("  No snapshots saved yet.\n\n  " +
			s.styles.Subtle.Render("Press esc to go back."))
	}
	if len(s.listLines) == 0 {
		s.listLines = splitLines(s.buildListContent())
	}

	lines := s.listLines
	pageSize := s.contentHeight()
	start, end := s.scroll.Window(len(lines), pageSize)

	var b strings.Builder
	if above := s.scroll.MoreAbove(); above > 0 {
		fmt.Fprintf(&b, "%s\n", scrollIndicatorLine(s.styles, "↑", above))
	}
	b.WriteString(strings.Join(lines[start:end], "\n"))
	if below := s.scroll.MoreBelow(len(lines), pageSize); below > 0 {
		fmt.Fprintf(&b, "\n%s", scrollIndicatorLine(s.styles, "↓", below))
	}
	return tea.NewView(b.String())
}
