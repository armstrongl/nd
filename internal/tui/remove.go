package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/viewport"
	"charm.land/huh/v2"

	"github.com/armstrongl/nd/internal/deploy"
	"github.com/armstrongl/nd/internal/oplog"
	"github.com/armstrongl/nd/internal/state"
)

type removeStep int

const (
	removeSelectAssets removeStep = iota
	removeConfirm
	removeRunning
	removeResult
)

// removeDoneMsg is sent when the bulk remove operation completes.
type removeDoneMsg struct {
	succeeded int
	failed    []deploy.RemoveError
}

// deploymentsLoadedMsg is sent when the async deployment load completes.
type deploymentsLoadedMsg struct {
	deployments []state.Deployment
	err         error
}

// bulkRemover is the interface used by removeBulkCmd to perform removals.
// deploy.Engine satisfies this. Tests provide a mock.
type bulkRemover interface {
	RemoveBulk([]deploy.RemoveRequest) (*deploy.BulkRemoveResult, error)
}

type removeScreen struct {
	svc    Services
	styles Styles
	isDark bool
	step   removeStep

	// selectAssets
	assetForm   *huh.Form
	selected    []string           // "sourceID:type/name" keys
	deployments []state.Deployment // all deployed assets

	// confirm
	confirmForm *huh.Form
	confirmed   bool

	// running
	progress progressBar

	// result
	succeeded int
	failed    []deploy.RemoveError
	dryRun    bool                   // true when result is a dry-run preview
	dryReqs   []deploy.RemoveRequest // populated for dry-run display

	// viewport for scrollable result step (created when entering removeResult, not at construction).
	// Two-phase init: ScreenSizeMsg may arrive before the result step; pending dimensions are stored
	// and applied when the viewport is created.
	vp            *viewport.Model
	pendingWidth  int
	pendingHeight int

	err error
}

func newRemoveScreen(svc Services, styles Styles, isDark bool) *removeScreen {
	return &removeScreen{
		svc:    svc,
		styles: styles,
		isDark: isDark,
		step:   removeSelectAssets,
	}
}

// Screen interface
func (m *removeScreen) Title() string { return "Remove" }

// H5: InputActive returns true during form steps to prevent q/esc from quitting.
func (m *removeScreen) InputActive() bool {
	return m.step == removeSelectAssets || m.step == removeConfirm
}

// Init starts async loading of deployed assets.
func (m *removeScreen) Init() tea.Cmd {
	svc := m.svc
	return func() tea.Msg {
		store := svc.StateStore()
		if store == nil {
			return deploymentsLoadedMsg{err: fmt.Errorf("state store not available")}
		}
		st, _, err := store.Load()
		if err != nil {
			return deploymentsLoadedMsg{err: err}
		}
		return deploymentsLoadedMsg{deployments: st.Deployments}
	}
}

// Update handles messages for each step of the remove flow.
func (m *removeScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ScreenSizeMsg:
		if m.vp != nil {
			m.vp.SetWidth(msg.Width)
			m.vp.SetHeight(msg.Height)
		} else {
			m.pendingWidth = msg.Width
			m.pendingHeight = msg.Height
		}
		return m, nil

	case deploymentsLoadedMsg:
		return m.handleDeploymentsLoaded(msg)

	case removeDoneMsg:
		m.step = removeResult
		m.succeeded = msg.succeeded
		m.failed = msg.failed
		m.initViewport()
		// M5: Log operation to oplog
		if ol := m.svc.OpLog(); ol != nil {
			_ = ol.Log(oplog.LogEntry{
				Timestamp: time.Now(),
				Operation: oplog.OpRemove,
				Scope:     m.svc.GetScope(),
				Succeeded: msg.succeeded,
				Failed:    len(msg.failed),
			})
		}
		return m, func() tea.Msg { return RefreshHeaderMsg{} }
	}

	switch m.step {
	case removeSelectAssets:
		return m.updateSelectAssets(msg)
	case removeConfirm:
		return m.updateConfirm(msg)
	case removeResult:
		return m.updateResult(msg)
	}

	return m, nil
}

// View renders the current step.
func (m *removeScreen) View() tea.View {
	if m.err != nil {
		return tea.NewView(fmt.Sprintf("  %s\n\n  %s",
			m.styles.Danger.Render(m.err.Error()),
			m.styles.Subtle.Render("Press esc to go back.")))
	}

	switch m.step {
	case removeSelectAssets:
		return m.viewSelectAssets()
	case removeConfirm:
		return m.viewConfirm()
	case removeRunning:
		return m.viewRunning()
	case removeResult:
		content := m.viewResultContent()
		if m.vp != nil && m.vp.Width() > 0 && m.vp.Height() > 0 {
			m.vp.SetContent(content)
			return tea.NewView(m.vp.View())
		}
		return tea.NewView(content)
	}

	return tea.NewView("")
}

// --- Step handlers ---

func (m *removeScreen) handleDeploymentsLoaded(msg deploymentsLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.err = msg.err
		m.step = removeResult
		m.initViewport()
		return m, nil
	}

	m.deployments = msg.deployments

	if len(m.deployments) == 0 {
		return m, nil
	}

	// Build the MultiSelect form.
	options := make([]huh.Option[string], len(m.deployments))
	for i, d := range m.deployments {
		key := d.Identity().String()
		options[i] = huh.NewOption(deploymentLabel(d), key)
	}

	m.assetForm = huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select assets to remove").
				Options(options...).
				Value(&m.selected),
		),
	).WithTheme(huh.ThemeFunc(huh.ThemeCatppuccin))

	return m, m.assetForm.Init()
}

func (m *removeScreen) updateSelectAssets(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.assetForm == nil {
		return m, nil
	}

	model, cmd := m.assetForm.Update(msg)
	if f, ok := model.(*huh.Form); ok {
		m.assetForm = f
	}

	if m.assetForm.State == huh.StateCompleted {
		if len(m.selected) == 0 {
			// Nothing selected — go back.
			return m, func() tea.Msg { return BackMsg{} }
		}
		return m.transitionToConfirm()
	}

	if m.assetForm.State == huh.StateAborted {
		return m, func() tea.Msg { return BackMsg{} }
	}

	return m, cmd
}

func (m *removeScreen) transitionToConfirm() (tea.Model, tea.Cmd) {
	m.step = removeConfirm

	title := fmt.Sprintf("Remove %d asset(s)? An auto-snapshot will be saved first.", len(m.selected))

	m.confirmForm = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(title).
				Affirmative("Yes").
				Negative("No").
				Value(&m.confirmed),
		),
	).WithTheme(huh.ThemeFunc(huh.ThemeCatppuccin))

	return m, m.confirmForm.Init()
}

func (m *removeScreen) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.confirmForm == nil {
		return m, nil
	}

	model, cmd := m.confirmForm.Update(msg)
	if f, ok := model.(*huh.Form); ok {
		m.confirmForm = f
	}

	if m.confirmForm.State == huh.StateCompleted {
		if !m.confirmed {
			return m, func() tea.Msg { return BackMsg{} }
		}
		return m.transitionToRunning()
	}

	if m.confirmForm.State == huh.StateAborted {
		return m, func() tea.Msg { return BackMsg{} }
	}

	return m, cmd
}

func (m *removeScreen) transitionToRunning() (tea.Model, tea.Cmd) {
	// Build remove requests from selected keys.
	reqs := m.buildRemoveRequests()

	// H2: Dry-run mode — show preview without executing
	if m.svc.IsDryRun() {
		m.step = removeResult
		m.dryRun = true
		m.dryReqs = reqs
		m.initViewport()
		return m, func() tea.Msg { return RefreshHeaderMsg{} }
	}

	m.step = removeRunning
	m.progress = newProgressBar(40)

	eng, err := m.svc.DeployEngine()
	if err != nil {
		m.err = fmt.Errorf("deploy engine: %w", err)
		return m, nil
	}
	if eng == nil {
		m.err = fmt.Errorf("deploy engine not available")
		return m, nil
	}

	// M3: Use bulk API for single lock cycle + auto-snapshot
	return m, removeBulkCmd(eng, reqs)
}

func (m *removeScreen) buildRemoveRequests() []deploy.RemoveRequest {
	// Build a lookup from identity string to deployment.
	lookup := make(map[string]state.Deployment, len(m.deployments))
	for _, d := range m.deployments {
		lookup[d.Identity().String()] = d
	}

	reqs := make([]deploy.RemoveRequest, 0, len(m.selected))
	for _, key := range m.selected {
		if d, ok := lookup[key]; ok {
			reqs = append(reqs, deploy.RemoveRequest{
				Identity:    d.Identity(),
				Scope:       d.Scope,       // H3: use deployment's recorded scope
				ProjectRoot: d.ProjectPath,
			})
		}
	}
	return reqs
}

func (m *removeScreen) updateResult(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		if keyMsg.String() == "enter" {
			return m, tea.Batch(
				func() tea.Msg { return PopToRootMsg{} },
				func() tea.Msg { return RefreshHeaderMsg{} },
			)
		}
	}
	// Forward remaining messages to viewport for scrolling.
	if m.vp != nil && m.vp.Width() > 0 && m.vp.Height() > 0 {
		updated, cmd := m.vp.Update(msg)
		m.vp = &updated
		return m, cmd
	}
	return m, nil
}

// --- Views ---

func (m *removeScreen) viewSelectAssets() tea.View {
	if len(m.deployments) == 0 {
		return tea.NewView("  " + NothingDeployed())
	}
	if m.assetForm == nil {
		return tea.NewView("  Loading deployed assets...")
	}
	return tea.NewView(m.assetForm.View())
}

func (m *removeScreen) viewConfirm() tea.View {
	if m.confirmForm == nil {
		return tea.NewView("")
	}
	return tea.NewView(m.confirmForm.View())
}

func (m *removeScreen) viewRunning() tea.View {
	return tea.NewView(fmt.Sprintf("  %s",
		m.styles.Primary.Render("Removing...")))
}

// viewResultContent renders the remove results as a string for viewport wrapping.
func (m *removeScreen) viewResultContent() string {
	var b strings.Builder

	// H2: Dry-run preview
	if m.dryRun {
		fmt.Fprintf(&b, "  %s Would remove %d asset(s):\n\n",
			m.styles.Warning.Render("[DRY RUN]"), len(m.dryReqs))
		for _, req := range m.dryReqs {
			fmt.Fprintf(&b, "    %s %s/%s\n",
				GlyphArrow, req.Identity.Type, req.Identity.Name)
		}
		fmt.Fprintf(&b, "\n  %s", m.styles.Subtle.Render("Press enter to return."))
		return b.String()
	}

	if m.succeeded > 0 {
		fmt.Fprintf(&b, "  %s %d asset(s) removed successfully.\n",
			m.styles.Success.Render(GlyphOK), m.succeeded)
	}

	if len(m.failed) > 0 {
		fmt.Fprintf(&b, "\n  %s %d failed:\n",
			m.styles.Danger.Render(GlyphBroken), len(m.failed))
		for _, f := range m.failed {
			fmt.Fprintf(&b, "    %s  %s\n",
				f.Identity.String(), m.styles.Subtle.Render(f.Err.Error()))
		}
	}

	fmt.Fprintf(&b, "\n  %s", m.styles.Subtle.Render("Press enter to return."))

	return b.String()
}

// initViewport creates the viewport for the result step and applies any
// pending dimensions stored from earlier ScreenSizeMsg deliveries.
func (m *removeScreen) initViewport() {
	vp := viewport.New(
		viewport.WithWidth(m.pendingWidth),
		viewport.WithHeight(m.pendingHeight),
	)
	m.vp = &vp
}

// --- Helpers ---

// deploymentLabel formats a deployment for display in lists.
// Returns "type/name (source)" or "name (source)" for context assets.
func deploymentLabel(d state.Deployment) string {
	subdir := d.AssetType.DeploySubdir()
	if subdir == "" {
		return fmt.Sprintf("%s (%s)", d.AssetName, d.SourceID)
	}
	return fmt.Sprintf("%s/%s (%s)", subdir, d.AssetName, d.SourceID)
}

// removeBulkCmd builds a tea.Cmd that removes assets via the bulk API and returns a removeDoneMsg.
func removeBulkCmd(eng bulkRemover, reqs []deploy.RemoveRequest) tea.Cmd {
	return func() tea.Msg {
		result, err := eng.RemoveBulk(reqs)
		if err != nil {
			// Total failure — report all as failed
			var failed []deploy.RemoveError
			for _, req := range reqs {
				failed = append(failed, deploy.RemoveError{
					Identity: req.Identity,
					Err:      err,
				})
			}
			return removeDoneMsg{failed: failed}
		}
		return removeDoneMsg{
			succeeded: len(result.Succeeded),
			failed:    result.Failed,
		}
	}
}
