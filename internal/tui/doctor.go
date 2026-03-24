package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"

	"github.com/armstrongl/nd/internal/deploy"
	"github.com/armstrongl/nd/internal/state"
)

type doctorStep int

const (
	doctorLoading doctorStep = iota
	doctorConfirm
	doctorFixing
	doctorDone
)

// doctorCheckedMsg carries the results of the async health check.
type doctorCheckedMsg struct {
	issues []state.HealthCheck
	err    error
}

// doctorSyncedMsg carries the results of the async sync/repair operation.
type doctorSyncedMsg struct {
	result *deploy.SyncResult
	err    error
}

// doctorScreen implements the doctor flow: scan -> confirm -> fix -> result.
type doctorScreen struct {
	svc    Services
	styles Styles
	isDark bool
	step   doctorStep

	// confirm step
	issues      []state.HealthCheck
	confirmForm *huh.Form
	confirmed   bool
	fixing      bool // guard against double-fire

	// done step
	syncResult *deploy.SyncResult
	err        error
}

func newDoctorScreen(svc Services, styles Styles, isDark bool) *doctorScreen {
	return &doctorScreen{svc: svc, styles: styles, isDark: isDark}
}

func (d *doctorScreen) Title() string { return "Doctor" }

// InputActive returns true during the confirm form to suppress global keys.
func (d *doctorScreen) InputActive() bool {
	return d.step == doctorConfirm
}

// Init starts an async health check.
func (d *doctorScreen) Init() tea.Cmd {
	svc := d.svc
	return func() tea.Msg {
		eng, err := svc.DeployEngine()
		if err != nil {
			return doctorCheckedMsg{err: err}
		}
		if eng == nil {
			return doctorCheckedMsg{err: fmt.Errorf("deploy engine not available")}
		}
		issues, err := eng.Check()
		return doctorCheckedMsg{issues: issues, err: err}
	}
}

// Update handles messages for each step of the doctor flow.
func (d *doctorScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case doctorCheckedMsg:
		return d.handleChecked(msg)

	case doctorSyncedMsg:
		d.step = doctorDone
		d.syncResult = msg.result
		d.err = msg.err
		return d, func() tea.Msg { return RefreshHeaderMsg{} }
	}

	switch d.step {
	case doctorConfirm:
		return d.updateConfirm(msg)
	case doctorDone:
		return d.updateDone(msg)
	}

	return d, nil
}

// View renders the current step.
func (d *doctorScreen) View() tea.View {
	switch d.step {
	case doctorLoading:
		return tea.NewView("  Scanning deployments...")

	case doctorConfirm:
		return d.viewConfirm()

	case doctorFixing:
		return tea.NewView(fmt.Sprintf("  %s", d.styles.Primary.Render("Applying fixes...")))

	case doctorDone:
		return d.viewDone()
	}

	return tea.NewView("")
}

// --- Step handlers ---

func (d *doctorScreen) handleChecked(msg doctorCheckedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		d.err = msg.err
		d.step = doctorDone
		return d, nil
	}

	if len(msg.issues) == 0 {
		// All healthy — skip confirm, go straight to done.
		d.step = doctorDone
		return d, nil
	}

	d.issues = msg.issues
	d.step = doctorConfirm

	title := fmt.Sprintf("Found %d issue(s). Fix all?", len(d.issues))
	d.confirmForm = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(title).
				Affirmative("Fix").
				Negative("Cancel").
				Value(&d.confirmed),
		),
	).WithTheme(huh.ThemeFunc(huh.ThemeCatppuccin))

	return d, d.confirmForm.Init()
}

func (d *doctorScreen) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if d.confirmForm == nil {
		return d, nil
	}

	// Guard against double-fire.
	if d.fixing {
		return d, nil
	}

	model, cmd := d.confirmForm.Update(msg)
	if f, ok := model.(*huh.Form); ok {
		d.confirmForm = f
	}

	if d.confirmForm.State == huh.StateCompleted {
		if !d.confirmed {
			return d, func() tea.Msg { return BackMsg{} }
		}
		d.fixing = true
		d.step = doctorFixing
		return d, d.runSync()
	}

	if d.confirmForm.State == huh.StateAborted {
		return d, func() tea.Msg { return BackMsg{} }
	}

	return d, cmd
}

func (d *doctorScreen) updateDone(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		if keyMsg.String() == "enter" {
			return d, func() tea.Msg { return PopToRootMsg{} }
		}
	}
	return d, nil
}

// runSync fires the async Sync operation.
func (d *doctorScreen) runSync() tea.Cmd {
	svc := d.svc
	return func() tea.Msg {
		eng, err := svc.DeployEngine()
		if err != nil {
			return doctorSyncedMsg{err: err}
		}
		if eng == nil {
			return doctorSyncedMsg{err: fmt.Errorf("deploy engine not available")}
		}
		result, err := eng.Sync()
		return doctorSyncedMsg{result: result, err: err}
	}
}

// --- Views ---

func (d *doctorScreen) viewConfirm() tea.View {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("  %s %d issue(s) found:\n\n",
		d.styles.Warning.Render(GlyphBroken), len(d.issues)))

	for _, issue := range d.issues {
		glyph := healthGlyph(issue.Status)
		styled := styleGlyphWith(d.styles, glyph, issue.Status)
		b.WriteString(fmt.Sprintf("    %s  %-20s  %s\n",
			styled, issue.Deployment.AssetName,
			d.styles.Subtle.Render(issue.Detail)))
	}

	if d.confirmForm != nil {
		b.WriteString("\n")
		b.WriteString(d.confirmForm.View())
	}

	return tea.NewView(b.String())
}

func (d *doctorScreen) viewDone() tea.View {
	if d.err != nil {
		return tea.NewView(fmt.Sprintf("  %s\n\n  %s\n\n  %s",
			d.styles.Danger.Render("Error"),
			d.err.Error(),
			d.styles.Subtle.Render("Press esc to go back.")))
	}

	if d.syncResult == nil {
		// All healthy — no sync was run.
		return tea.NewView(fmt.Sprintf("  %s All deployments are healthy.\n\n  %s",
			d.styles.Success.Render(GlyphOK),
			d.styles.Subtle.Render("Press enter to return.")))
	}

	var b strings.Builder
	repaired := len(d.syncResult.Repaired)
	removed := len(d.syncResult.Removed)
	warnings := d.syncResult.Warnings

	b.WriteString(fmt.Sprintf("  %s Fixes applied:\n\n",
		d.styles.Success.Render(GlyphOK)))

	if repaired > 0 {
		b.WriteString(fmt.Sprintf("  %s Repaired: %d\n",
			d.styles.Success.Render(GlyphArrow), repaired))
	}
	if removed > 0 {
		b.WriteString(fmt.Sprintf("  %s Removed:  %d\n",
			d.styles.Warning.Render(GlyphArrow), removed))
	}
	if repaired == 0 && removed == 0 {
		b.WriteString("  No changes made.\n")
	}

	for _, w := range warnings {
		b.WriteString(fmt.Sprintf("  %s %s\n",
			d.styles.Warning.Render("!"), w))
	}

	b.WriteString(fmt.Sprintf("\n  %s", d.styles.Subtle.Render("Press enter to return.")))

	return tea.NewView(b.String())
}

// styleGlyphWith applies color to a health glyph using the provided Styles.
// Exported as a package-level function so doctor.go can use statusScreen's logic
// without calling a method on statusScreen.
func styleGlyphWith(s Styles, glyph string, h state.HealthStatus) string {
	switch h {
	case state.HealthOK:
		return s.Success.Render(glyph)
	case state.HealthBroken, state.HealthMissing:
		return s.Danger.Render(glyph)
	case state.HealthDrifted:
		return s.Warning.Render(glyph)
	case state.HealthOrphaned:
		return s.Subtle.Render(glyph)
	default:
		return glyph
	}
}
