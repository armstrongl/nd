package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"

	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/state"
)

type pinStep int

const (
	pinLoading pinStep = iota
	pinSelect
	pinConfirm
	pinRunning
	pinDone
)

// pinLoadedMsg carries the initial deployment list for pin selection.
type pinLoadedMsg struct {
	deployments []state.Deployment
	err         error
}

// pinDoneMsg carries the result of applying pin/unpin changes.
type pinDoneMsg struct {
	pinned   int
	unpinned int
	err      error
}

// pinScreen lets users toggle pin status on deployed assets.
type pinScreen struct {
	svc    Services
	styles Styles
	isDark bool
	step   pinStep

	deployments []state.Deployment
	selected    []string // identity strings of currently pinned/to-pin assets
	err         error

	// select
	assetForm *huh.Form
	applying  bool

	// confirm
	confirmForm *huh.Form
	confirmed   bool

	// done
	pinned   int
	unpinned int
}

func newPinScreen(svc Services, styles Styles, isDark bool) *pinScreen {
	return &pinScreen{svc: svc, styles: styles, isDark: isDark}
}

func (s *pinScreen) Title() string     { return "Pin/Unpin" }
func (s *pinScreen) InputActive() bool {
	return s.step == pinSelect || s.step == pinConfirm
}

func (s *pinScreen) Init() tea.Cmd {
	svc := s.svc
	return func() tea.Msg {
		store := svc.StateStore()
		if store == nil {
			return pinLoadedMsg{err: fmt.Errorf("state store not available")}
		}
		st, _, err := store.Load()
		if err != nil {
			return pinLoadedMsg{err: err}
		}
		return pinLoadedMsg{deployments: st.Deployments}
	}
}

func (s *pinScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case pinLoadedMsg:
		return s.handleLoaded(msg)

	case pinDoneMsg:
		s.step = pinDone
		s.pinned = msg.pinned
		s.unpinned = msg.unpinned
		s.err = msg.err
		return s, func() tea.Msg { return RefreshHeaderMsg{} }
	}

	switch s.step {
	case pinSelect:
		return s.updateSelect(msg)
	case pinConfirm:
		return s.updateConfirm(msg)
	case pinDone:
		return s.updateDone(msg)
	}
	return s, nil
}

func (s *pinScreen) View() tea.View {
	switch s.step {
	case pinLoading:
		return tea.NewView("  Loading deployed assets...")

	case pinSelect:
		if len(s.deployments) == 0 {
			return tea.NewView("  " + NothingDeployed() + "\n\n  " +
				s.styles.Subtle.Render("Press esc to go back."))
		}
		if s.assetForm != nil {
			return tea.NewView(s.assetForm.View())
		}

	case pinConfirm:
		if s.confirmForm != nil {
			return tea.NewView(s.confirmForm.View())
		}

	case pinRunning:
		return tea.NewView(fmt.Sprintf("  %s", s.styles.Primary.Render("Applying changes...")))

	case pinDone:
		return s.viewDone()
	}
	return tea.NewView("")
}

// --- Step handlers ---

func (s *pinScreen) handleLoaded(msg pinLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		s.err = msg.err
		s.step = pinDone
		return s, nil
	}
	s.deployments = msg.deployments
	s.step = pinSelect

	if len(s.deployments) == 0 {
		return s, nil
	}

	// Pre-select currently pinned assets.
	opts := make([]huh.Option[string], len(s.deployments))
	for i, d := range s.deployments {
		key := d.Identity().String()
		label := deploymentLabel(d)
		if d.Origin == nd.OriginPinned {
			label += " [pinned]"
			s.selected = append(s.selected, key)
		}
		opts[i] = huh.NewOption(label, key)
	}

	s.assetForm = huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select assets to pin (pinned assets are pre-selected)").
				Options(opts...).
				Value(&s.selected),
		),
	).WithTheme(huh.ThemeFunc(huh.ThemeCatppuccin))

	return s, s.assetForm.Init()
}

func (s *pinScreen) updateSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	if s.applying || s.assetForm == nil {
		return s, nil
	}
	model, cmd := s.assetForm.Update(msg)
	if f, ok := model.(*huh.Form); ok {
		s.assetForm = f
	}
	if s.assetForm.State == huh.StateCompleted {
		return s.buildConfirm()
	}
	if s.assetForm.State == huh.StateAborted {
		return s, func() tea.Msg { return BackMsg{} }
	}
	return s, cmd
}

func (s *pinScreen) buildConfirm() (tea.Model, tea.Cmd) {
	s.step = pinConfirm
	// Compute diff: which assets will be newly pinned / unpinned.
	newPins, newUnpins := s.computeDiff()
	title := fmt.Sprintf("Pin %d, unpin %d asset(s)?", newPins, newUnpins)
	if newPins == 0 && newUnpins == 0 {
		// No changes — go back directly.
		return s, func() tea.Msg { return BackMsg{} }
	}
	s.confirmed = false
	s.confirmForm = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(title).
				Affirmative("Apply").
				Negative("Cancel").
				Value(&s.confirmed),
		),
	).WithTheme(huh.ThemeFunc(huh.ThemeCatppuccin))
	return s, s.confirmForm.Init()
}

func (s *pinScreen) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if s.applying {
		return s, nil
	}
	model, cmd := s.confirmForm.Update(msg)
	if f, ok := model.(*huh.Form); ok {
		s.confirmForm = f
	}
	if s.confirmForm.State == huh.StateCompleted {
		if !s.confirmed {
			return s, func() tea.Msg { return BackMsg{} }
		}
		s.applying = true
		s.step = pinRunning
		return s, s.runApply()
	}
	if s.confirmForm.State == huh.StateAborted {
		return s, func() tea.Msg { return BackMsg{} }
	}
	return s, cmd
}

func (s *pinScreen) runApply() tea.Cmd {
	selectedSet := make(map[string]bool, len(s.selected))
	for _, k := range s.selected {
		selectedSet[k] = true
	}
	deployments := s.deployments
	svc := s.svc
	return func() tea.Msg {
		eng, err := svc.DeployEngine()
		if err != nil {
			return pinDoneMsg{err: err}
		}
		if eng == nil {
			return pinDoneMsg{err: fmt.Errorf("deploy engine not available")}
		}
		var pinned, unpinned int
		for _, d := range deployments {
			key := d.Identity().String()
			wasPinned := d.Origin == nd.OriginPinned
			nowPinned := selectedSet[key]

			if nowPinned == wasPinned {
				continue // no change
			}
			origin := nd.OriginManual
			if nowPinned {
				origin = nd.OriginPinned
			}
			if err := eng.SetOrigin(d.Identity(), d.Scope, d.ProjectPath, origin); err != nil {
				return pinDoneMsg{err: fmt.Errorf("set origin %s: %w", key, err)}
			}
			if nowPinned {
				pinned++
			} else {
				unpinned++
			}
		}
		return pinDoneMsg{pinned: pinned, unpinned: unpinned}
	}
}

func (s *pinScreen) updateDone(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok && keyMsg.String() == "enter" {
		return s, tea.Batch(
			func() tea.Msg { return PopToRootMsg{} },
			func() tea.Msg { return RefreshHeaderMsg{} },
		)
	}
	return s, nil
}

func (s *pinScreen) viewDone() tea.View {
	if s.err != nil {
		return tea.NewView(fmt.Sprintf("  %s\n\n  %s\n\n  %s",
			s.styles.Danger.Render("Error"),
			s.err.Error(),
			s.styles.Subtle.Render("Press esc to go back.")))
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("  %s Changes applied:\n\n", s.styles.Success.Render(GlyphOK)))
	if s.pinned > 0 {
		b.WriteString(fmt.Sprintf("  %s Pinned:   %d\n", s.styles.Primary.Render(GlyphArrow), s.pinned))
	}
	if s.unpinned > 0 {
		b.WriteString(fmt.Sprintf("  %s Unpinned: %d\n", s.styles.Subtle.Render(GlyphArrow), s.unpinned))
	}
	if s.pinned == 0 && s.unpinned == 0 {
		b.WriteString("  No changes made.\n")
	}
	b.WriteString(fmt.Sprintf("\n  %s", s.styles.Subtle.Render("Press enter to return.")))
	return tea.NewView(b.String())
}

// computeDiff returns how many assets will be newly pinned and newly unpinned.
func (s *pinScreen) computeDiff() (newPins, newUnpins int) {
	selectedSet := make(map[string]bool, len(s.selected))
	for _, k := range s.selected {
		selectedSet[k] = true
	}
	for _, d := range s.deployments {
		key := d.Identity().String()
		wasPinned := d.Origin == nd.OriginPinned
		nowPinned := selectedSet[key]
		if nowPinned && !wasPinned {
			newPins++
		} else if !nowPinned && wasPinned {
			newUnpins++
		}
	}
	return
}
