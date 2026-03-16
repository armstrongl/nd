package tuiapp

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/armstrongl/nd/internal/agent"
	"github.com/armstrongl/nd/internal/deploy"
	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/state"
	"github.com/armstrongl/nd/internal/tui"
	"github.com/armstrongl/nd/internal/tui/components"
)

// ConfirmState holds the data for a yes/no confirmation dialog.
type ConfirmState struct {
	Active  bool
	Message string
	OnYes   func() tea.Cmd
}

// App is the root Bubble Tea model. It composes all TUI components
// and routes messages through a state machine.
type App struct {
	// Services
	deployer tui.Deployer
	profiles tui.ProfileSwitcher
	sources  tui.SourceScanner
	agents   tui.AgentDetector

	// State
	state     tui.AppState
	prevState tui.AppState
	confirm   ConfirmState
	scope     nd.Scope
	agent     *agent.Agent

	// Config
	hasProjectDir      bool
	resolveProjectRoot func() (string, error)

	// Cached data
	statusEntries []deploy.StatusEntry
	indexReady    bool

	// Components
	header     components.Header
	tabbar     components.TabBar
	table      *components.Table
	helpbar    components.HelpBar
	menu       *components.Menu
	picker     components.Picker
	fuzzy      components.FuzzyFinder
	listPicker components.ListPicker
	prompt     components.Prompt
	toast      components.Toast

	// Terminal
	width  int
	height int
	styles tui.Styles
	keys   tui.KeyMap
}

// New creates a root App model with the given services.
func New(
	d tui.Deployer,
	p tui.ProfileSwitcher,
	s tui.SourceScanner,
	a tui.AgentDetector,
	hasProjectDir bool,
	resolveProjectRoot func() (string, error),
) App {
	styles := tui.DefaultStyles()

	detectedAgents := a.All()
	picker := components.NewPicker(detectedAgents, hasProjectDir)

	menu := components.NewMenu()
	menu.Styles = styles
	menu.Summary.Loading = true

	table := components.NewTable()
	table.Styles = styles

	tabbar := components.TabBar{
		Tabs:   components.DefaultTabs(),
		Styles: styles,
	}

	return App{
		deployer:           d,
		profiles:           p,
		sources:            s,
		agents:             a,
		state:              tui.StatePicker,
		hasProjectDir:      hasProjectDir,
		resolveProjectRoot: resolveProjectRoot,
		styles:             styles,
		keys:               tui.DefaultKeyMap(),
		picker:             picker,
		menu:               menu,
		table:              table,
		tabbar:             tabbar,
		header: components.Header{
			Styles: styles,
		},
		helpbar: components.HelpBar{
			Bindings: components.HelpForState(tui.StatePicker),
			Styles:   styles,
		},
	}
}

// Init returns the initial command (agent detection).
func (a App) Init() tea.Cmd {
	return a.cmdDetectAgents()
}

// Update routes messages through the state machine.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Global message handling (regardless of state)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.propagateSize()
		return a, nil

	case tea.BackgroundColorMsg:
		ld := lipgloss.LightDark(msg.IsDark())
		a.styles = tui.NewStyles(ld)
		a.propagateStyles()
		return a, nil

	case tui.ToastMsg:
		t, cmd := components.NewToast(msg.Message, msg.Level)
		a.toast = t
		return a, cmd

	case tui.ToastDismissMsg:
		a.toast.Dismiss()
		return a, nil

	case tui.StatusResultMsg:
		return a.handleStatusResult(msg)

	case tui.HealthCheckMsg:
		return a.handleHealthCheck(msg)

	case tui.ScanCompleteMsg:
		return a.handleScanComplete(msg)

	case tui.SyncResultMsg:
		return a.handleSyncResult(msg)

	case tui.DeployResultMsg:
		return a.handleDeployResult(msg)

	case tui.RemoveResultMsg:
		return a.handleRemoveResult(msg)

	case tui.ProfileSwitchMsg:
		return a.handleProfileSwitch(msg)

	case tui.SnapshotSaveMsg:
		return a.handleSnapshotSave(msg)

	case tui.SnapshotRestoreMsg:
		return a.handleSnapshotRestore(msg)

	case tui.ErrorMsg:
		t, cmd := components.NewToast(msg.Err.Error(), tui.ToastError)
		a.toast = t
		return a, cmd
	}

	// State-specific routing
	switch a.state {
	case tui.StatePicker:
		return a.updatePicker(msg)
	case tui.StateMenu:
		return a.updateMenu(msg)
	case tui.StateDashboard:
		return a.updateDashboard(msg)
	case tui.StateConfirm:
		return a.updateConfirm(msg)
	case tui.StateFuzzy:
		return a.updateFuzzy(msg)
	case tui.StateListPicker:
		return a.updateListPicker(msg)
	case tui.StatePrompt:
		return a.updatePrompt(msg)
	}

	return a, nil
}

// View renders the TUI.
func (a App) View() tea.View {
	if a.width < 40 || a.height < 10 {
		v := tea.NewView(a.tooSmallView())
		v.AltScreen = true
		return v
	}

	var content string
	switch a.state {
	case tui.StatePicker:
		content = a.pickerView()
	case tui.StateMenu:
		content = a.menuView()
	case tui.StateDashboard, tui.StateDetail:
		content = a.dashboardView()
	case tui.StateConfirm:
		content = a.confirmView()
	case tui.StateFuzzy:
		content = a.fuzzyView()
	case tui.StateListPicker:
		content = a.listPickerView()
	case tui.StatePrompt:
		content = a.promptView()
	case tui.StateLoading:
		content = a.loadingView()
	default:
		content = a.dashboardView()
	}

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

// --- State update handlers ---

func (a App) updatePicker(msg tea.Msg) (tea.Model, tea.Cmd) {
	p, cmd := a.picker.Update(msg)
	a.picker = p

	if a.picker.Done {
		scope, ag := a.picker.Selected()
		a.scope = scope
		a.agent = ag
		a.transitionTo(tui.StateMenu)
		a.header.Scope = scope
		if ag != nil {
			a.header.Agent = ag.Name
		}
		if prof, err := a.profiles.ActiveProfile(); err == nil {
			a.header.Profile = prof
		}
		return a, tea.Batch(cmd, a.cmdLoadStatus())
	}

	return a, cmd
}

func (a App) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, a.keys.Quit):
			return a, tea.Quit
		case key.Matches(msg, a.keys.Enter):
			item := a.menu.SelectedItem()
			a.tabbar.Active = item.TabIdx
			a.transitionTo(tui.StateDashboard)
			return a, nil
		}
	}

	m, cmd := a.menu.Update(msg)
	a.menu = m
	return a, cmd
}

func (a App) updateDashboard(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, a.keys.Esc):
			a.transitionTo(tui.StateMenu)
			return a, nil
		case key.Matches(msg, a.keys.Left):
			a.tabbar.Prev()
			a.refreshTable()
			return a, nil
		case key.Matches(msg, a.keys.Right):
			a.tabbar.Next()
			a.refreshTable()
			return a, nil
		case key.Matches(msg, a.keys.Deploy):
			return a.openFuzzyFinder()
		case key.Matches(msg, a.keys.Remove):
			return a.confirmRemove()
		case key.Matches(msg, a.keys.Sync):
			return a, a.cmdSync()
		case key.Matches(msg, a.keys.Fix):
			return a, a.cmdFix()
		case key.Matches(msg, a.keys.Profile):
			return a.openProfilePicker()
		case key.Matches(msg, a.keys.Snapshot):
			return a.openSnapshotMenu()
		}
	}

	t, cmd := a.table.Update(msg)
	a.table = t
	return a, cmd
}

func (a App) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, a.keys.Yes):
			var cmd tea.Cmd
			if a.confirm.OnYes != nil {
				cmd = a.confirm.OnYes()
			}
			a.confirm = ConfirmState{}
			a.transitionTo(a.prevState)
			return a, cmd
		case key.Matches(msg, a.keys.No), key.Matches(msg, a.keys.Esc):
			a.confirm = ConfirmState{}
			a.transitionTo(a.prevState)
			return a, nil
		}
	}
	return a, nil
}

func (a App) updateFuzzy(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, a.keys.Esc):
			a.transitionTo(tui.StateDashboard)
			return a, nil
		case key.Matches(msg, a.keys.Enter):
			if item := a.fuzzy.SelectedItem(); item != nil {
				a.transitionTo(tui.StateDashboard)
				return a, a.cmdDeploy(item)
			}
			return a, nil
		}
	}

	f, cmd := a.fuzzy.Update(msg)
	a.fuzzy = f
	return a, cmd
}

func (a App) updateListPicker(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, a.keys.Esc):
			a.transitionTo(a.prevState)
			return a, nil
		case key.Matches(msg, a.keys.Enter):
			if item := a.listPicker.SelectedItem(); item != nil {
				return a.handleListPickerSelect(item)
			}
			return a, nil
		}
	}

	lp, cmd := a.listPicker.Update(msg)
	a.listPicker = lp
	return a, cmd
}

func (a App) updatePrompt(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, a.keys.Esc):
			a.transitionTo(tui.StateDashboard)
			return a, nil
		case key.Matches(msg, a.keys.Enter):
			name := a.prompt.Value()
			if name != "" {
				a.transitionTo(tui.StateDashboard)
				return a, a.cmdSnapshotSave(name)
			}
			return a, nil
		}
	}

	p, cmd := a.prompt.Update(msg)
	a.prompt = p
	return a, cmd
}

// --- Async command helpers ---

func (a App) cmdDetectAgents() tea.Cmd {
	return func() tea.Msg {
		a.agents.Detect()
		return nil
	}
}

func (a App) cmdLoadStatus() tea.Cmd {
	return func() tea.Msg {
		entries, err := a.deployer.Status()
		return tui.StatusResultMsg{Entries: entries, Err: err}
	}
}

func (a App) cmdSync() tea.Cmd {
	return func() tea.Msg {
		result, err := a.deployer.Sync()
		return tui.SyncResultMsg{Result: result, Err: err}
	}
}

func (a App) cmdFix() tea.Cmd {
	return func() tea.Msg {
		checks, err := a.deployer.Check()
		return tui.HealthCheckMsg{Checks: checks, Err: err}
	}
}

func (a App) cmdDeploy(item *components.FuzzyItem) tea.Cmd {
	return func() tea.Msg {
		req := deploy.DeployRequest{
			Scope: a.scope,
		}
		if a.scope == nd.ScopeProject {
			if root, err := a.resolveProjectRoot(); err == nil {
				req.ProjectRoot = root
			}
		}
		result, err := a.deployer.Deploy(req)
		return tui.DeployResultMsg{Result: result, Err: err}
	}
}

func (a App) cmdRemove(row components.TableRow) tea.Cmd {
	return func() tea.Msg {
		req := deploy.RemoveRequest{
			Scope: row.Scope,
		}
		if row.Scope == nd.ScopeProject {
			if root, err := a.resolveProjectRoot(); err == nil {
				req.ProjectRoot = root
			}
		}
		err := a.deployer.Remove(req)
		return tui.RemoveResultMsg{Err: err}
	}
}

func (a App) cmdSnapshotSave(name string) tea.Cmd {
	return func() tea.Msg {
		err := a.profiles.SaveSnapshot(name)
		return tui.SnapshotSaveMsg{Name: name, Err: err}
	}
}

func (a App) cmdProfileSwitch(target string) tea.Cmd {
	return func() tea.Msg {
		current, _ := a.profiles.ActiveProfile()
		result, err := a.profiles.Switch(current, target)
		return tui.ProfileSwitchMsg{Result: result, Err: err}
	}
}

func (a App) cmdSnapshotRestore(name string) tea.Cmd {
	return func() tea.Msg {
		result, err := a.profiles.Restore(name)
		return tui.SnapshotRestoreMsg{Result: result, Err: err}
	}
}

// --- Message result handlers ---

func (a App) handleStatusResult(msg tui.StatusResultMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		t, cmd := components.NewToast("Failed to load status: "+msg.Err.Error(), tui.ToastError)
		a.toast = t
		return a, cmd
	}

	a.statusEntries = msg.Entries
	a.indexReady = true

	a.menu.Summary.Loading = false
	a.menu.Summary.Sources = len(a.sources.Sources())
	a.menu.Summary.Deployed = len(msg.Entries)
	issueCount := 0
	for _, e := range msg.Entries {
		if e.Health != state.HealthOK {
			issueCount++
		}
	}
	a.menu.Summary.Issues = issueCount
	a.header.IssueCount = issueCount

	a.refreshTable()
	return a, nil
}

func (a App) handleHealthCheck(msg tui.HealthCheckMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		t, cmd := components.NewToast("Health check failed: "+msg.Err.Error(), tui.ToastError)
		a.toast = t
		return a, cmd
	}

	count := len(msg.Checks)
	message := "All healthy"
	level := tui.ToastSuccess
	if count > 0 {
		message = fmt.Sprintf("%d issues found", count)
		level = tui.ToastWarning
	}
	t, cmd := components.NewToast(message, level)
	a.toast = t
	return a, tea.Batch(cmd, a.cmdLoadStatus())
}

func (a App) handleScanComplete(msg tui.ScanCompleteMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		t, cmd := components.NewToast("Scan failed: "+msg.Err.Error(), tui.ToastError)
		a.toast = t
		return a, cmd
	}
	t, cmd := components.NewToast("Scan complete", tui.ToastSuccess)
	a.toast = t
	return a, cmd
}

func (a App) handleSyncResult(msg tui.SyncResultMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		t, cmd := components.NewToast("Sync failed: "+msg.Err.Error(), tui.ToastError)
		a.toast = t
		return a, cmd
	}
	repaired := 0
	removed := 0
	if msg.Result != nil {
		repaired = len(msg.Result.Repaired)
		removed = len(msg.Result.Removed)
	}
	message := fmt.Sprintf("Sync: %d repaired, %d removed", repaired, removed)
	t, cmd := components.NewToast(message, tui.ToastSuccess)
	a.toast = t
	return a, tea.Batch(cmd, a.cmdLoadStatus())
}

func (a App) handleDeployResult(msg tui.DeployResultMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		t, cmd := components.NewToast("Deploy failed: "+msg.Err.Error(), tui.ToastError)
		a.toast = t
		return a, cmd
	}
	message := "Deployed successfully"
	level := tui.ToastSuccess
	if msg.RequiresManual {
		message += " (manual registration needed)"
		level = tui.ToastWarning
	}
	t, cmd := components.NewToast(message, level)
	a.toast = t
	return a, tea.Batch(cmd, a.cmdLoadStatus())
}

func (a App) handleRemoveResult(msg tui.RemoveResultMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		t, cmd := components.NewToast("Remove failed: "+msg.Err.Error(), tui.ToastError)
		a.toast = t
		return a, cmd
	}
	t, cmd := components.NewToast("Asset removed", tui.ToastSuccess)
	a.toast = t
	return a, tea.Batch(cmd, a.cmdLoadStatus())
}

func (a App) handleProfileSwitch(msg tui.ProfileSwitchMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		t, cmd := components.NewToast("Profile switch failed: "+msg.Err.Error(), tui.ToastError)
		a.toast = t
		return a, cmd
	}
	if prof, err := a.profiles.ActiveProfile(); err == nil {
		a.header.Profile = prof
	}
	t, cmd := components.NewToast("Profile switched", tui.ToastSuccess)
	a.toast = t
	return a, tea.Batch(cmd, a.cmdLoadStatus())
}

func (a App) handleSnapshotSave(msg tui.SnapshotSaveMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		t, cmd := components.NewToast("Snapshot save failed: "+msg.Err.Error(), tui.ToastError)
		a.toast = t
		return a, cmd
	}
	t, cmd := components.NewToast(fmt.Sprintf("Snapshot '%s' saved", msg.Name), tui.ToastSuccess)
	a.toast = t
	return a, cmd
}

func (a App) handleSnapshotRestore(msg tui.SnapshotRestoreMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		t, cmd := components.NewToast("Restore failed: "+msg.Err.Error(), tui.ToastError)
		a.toast = t
		return a, cmd
	}
	t, cmd := components.NewToast("Snapshot restored", tui.ToastSuccess)
	a.toast = t
	return a, tea.Batch(cmd, a.cmdLoadStatus())
}

// --- State transition helpers ---

func (a *App) transitionTo(newState tui.AppState) {
	a.prevState = a.state
	a.state = newState
	a.helpbar.Bindings = components.HelpForState(newState)
}

func (a *App) propagateSize() {
	a.header.Width = a.width
	a.tabbar.Width = a.width
	a.table.Width = a.width
	a.helpbar.Width = a.width
	a.menu.Width = a.width
	a.menu.Height = a.height
	a.fuzzy.Width = a.width
	a.fuzzy.Height = a.height - 4

	// Table height: total - header(1) - tabbar(1) - helpbar(1) - toast(1) - padding(2)
	a.table.Height = a.height - 6
	if a.table.Height < 3 {
		a.table.Height = 3
	}
}

func (a *App) propagateStyles() {
	a.header.Styles = a.styles
	a.tabbar.Styles = a.styles
	a.table.Styles = a.styles
	a.helpbar.Styles = a.styles
	a.menu.Styles = a.styles
}

// --- Action openers ---

func (a App) openFuzzyFinder() (tea.Model, tea.Cmd) {
	var items []components.FuzzyItem
	for _, e := range a.statusEntries {
		items = append(items, components.FuzzyItem{
			Name:   e.Deployment.AssetName,
			Type:   e.Deployment.AssetType,
			Source: e.Deployment.SourceID,
		})
	}
	a.fuzzy = components.NewFuzzyFinder(items, "")
	a.fuzzy.Width = a.width
	a.fuzzy.Height = a.height - 4
	a.transitionTo(tui.StateFuzzy)
	return a, nil
}

func (a App) confirmRemove() (tea.Model, tea.Cmd) {
	if len(a.table.Rows) == 0 || a.table.Selected >= len(a.table.Rows) {
		return a, nil
	}
	row := a.table.Rows[a.table.Selected]
	message := fmt.Sprintf("Remove '%s'?", row.Name)
	if row.Detail != nil && row.Detail.Pinned {
		message = fmt.Sprintf("Asset '%s' is PINNED. Remove anyway?", row.Name)
	}
	a.confirm = ConfirmState{
		Active:  true,
		Message: message,
		OnYes: func() tea.Cmd {
			return a.cmdRemove(row)
		},
	}
	a.transitionTo(tui.StateConfirm)
	return a, nil
}

func (a App) openProfilePicker() (tea.Model, tea.Cmd) {
	profiles, err := a.profiles.ListProfiles()
	if err != nil {
		t, cmd := components.NewToast("Failed to load profiles: "+err.Error(), tui.ToastError)
		a.toast = t
		return a, cmd
	}
	active, _ := a.profiles.ActiveProfile()
	var items []components.ListPickerItem
	for _, p := range profiles {
		items = append(items, components.ListPickerItem{
			Label:       p.Name,
			Description: fmt.Sprintf("%d assets", p.AssetCount),
			Active:      p.Name == active,
		})
	}
	a.listPicker = components.NewListPicker("Switch Profile", items)
	a.transitionTo(tui.StateListPicker)
	return a, nil
}

func (a App) openSnapshotMenu() (tea.Model, tea.Cmd) {
	a.prompt = components.NewPrompt("Save Snapshot", "snapshot name")
	a.transitionTo(tui.StatePrompt)
	return a, nil
}

func (a App) handleListPickerSelect(item *components.ListPickerItem) (tea.Model, tea.Cmd) {
	if a.listPicker.Title == "Switch Profile" {
		if item.Active {
			a.transitionTo(tui.StateDashboard)
			return a, nil
		}
		a.transitionTo(tui.StateDashboard)
		return a, a.cmdProfileSwitch(item.Label)
	}
	a.transitionTo(tui.StateDashboard)
	return a, a.cmdSnapshotRestore(item.Label)
}

// --- Table refresh ---

func (a *App) refreshTable() {
	rows := a.buildTableRows()
	a.table.SetRows(rows)
	a.updateTabBadges()
}

func (a *App) buildTableRows() []components.TableRow {
	activeTab := a.tabbar.Active

	var rows []components.TableRow
	for _, e := range a.statusEntries {
		// Filter by tab: tab 0 = overview (all), tabs 1-8 = specific type
		if activeTab > 0 && activeTab-1 < len(tabAssetTypes) {
			if e.Deployment.AssetType != tabAssetTypes[activeTab-1] {
				continue
			}
		}
		rows = append(rows, components.TableRow{
			Origin:     e.Deployment.Origin,
			Health:     e.Health,
			Name:       e.Deployment.AssetName,
			Source:     e.Deployment.SourceID,
			Scope:      e.Deployment.Scope,
			StatusText: healthText(e.Health),
			IsFailed:   e.Health == state.HealthBroken,
		})
	}
	return rows
}

// tabAssetTypes maps tab indices (1-8) to asset types.
var tabAssetTypes = []nd.AssetType{
	nd.AssetSkill,
	nd.AssetAgent,
	nd.AssetCommand,
	nd.AssetOutputStyle,
	nd.AssetRule,
	nd.AssetContext,
	nd.AssetPlugin,
	nd.AssetHook,
}

func (a *App) updateTabBadges() {
	issueCounts := make(map[nd.AssetType]int)
	for _, e := range a.statusEntries {
		if e.Health != state.HealthOK {
			issueCounts[e.Deployment.AssetType]++
		}
	}

	for i, tab := range a.tabbar.Tabs {
		if i == 0 {
			total := 0
			for _, c := range issueCounts {
				total += c
			}
			tab.IssueCount = total
		} else if i-1 < len(tabAssetTypes) {
			tab.IssueCount = issueCounts[tabAssetTypes[i-1]]
		}
		a.tabbar.Tabs[i] = tab
	}
}

func healthText(h state.HealthStatus) string {
	switch h {
	case state.HealthOK:
		return "ok"
	case state.HealthBroken:
		return "broken"
	case state.HealthDrifted:
		return "drifted"
	default:
		return "unknown"
	}
}

// --- View builders ---

func (a App) tooSmallView() string {
	return fmt.Sprintf("Terminal too small (%dx%d).\nMinimum: 40x10", a.width, a.height)
}

func (a App) pickerView() string {
	var b strings.Builder
	b.WriteString(a.picker.View())
	b.WriteString("\n\n")
	b.WriteString(a.helpbar.View())
	return b.String()
}

func (a App) menuView() string {
	var b strings.Builder
	b.WriteString(a.header.View())
	b.WriteString("\n")
	b.WriteString(a.menu.View())
	b.WriteString("\n")
	b.WriteString(a.helpbar.View())
	return b.String()
}

func (a App) dashboardView() string {
	var b strings.Builder
	b.WriteString(a.header.View())
	b.WriteString("\n")
	if a.toast.Visible {
		b.WriteString(a.toast.View())
		b.WriteString("\n")
	}
	b.WriteString(a.tabbar.View())
	b.WriteString("\n")
	b.WriteString(a.table.View())
	b.WriteString("\n")
	b.WriteString(a.helpbar.View())
	return b.String()
}

func (a App) confirmView() string {
	var b strings.Builder
	b.WriteString(a.dashboardView())
	b.WriteString("\n\n")
	b.WriteString(tui.StyleModal.Render(fmt.Sprintf("  %s\n\n  y/n?", a.confirm.Message)))
	return b.String()
}

func (a App) fuzzyView() string {
	return a.fuzzy.View()
}

func (a App) listPickerView() string {
	return a.listPicker.View()
}

func (a App) promptView() string {
	return a.prompt.View()
}

func (a App) loadingView() string {
	return a.styles.Loading.Render("Loading...")
}
