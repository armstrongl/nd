package tui

import (
	"errors"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"

	"github.com/armstrongl/nd/internal/agent"
	"github.com/armstrongl/nd/internal/asset"
	"github.com/armstrongl/nd/internal/deploy"
	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/oplog"
	"github.com/armstrongl/nd/internal/state"
)

type deployStep int

const (
	deployPickType deployStep = iota
	deployPickAgents
	deploySelectAssets
	deployRunning
	deployConflictConfirm
	deployResult
)

// agentDeploy pairs a target agent (by name) and its engine with the deploy
// requests to run against that agent.
type agentDeploy struct {
	name   string
	engine *deploy.Engine
	reqs   []deploy.DeployRequest
}

// typeEntry pairs a display label with an optional asset type filter.
type typeEntry struct {
	label     string
	assetType nd.AssetType // empty string means "all types"
}

// typeDisplayNames returns the list of type choices for the picker.
func typeDisplayNames() []typeEntry {
	return []typeEntry{
		{label: "All types", assetType: ""},
		{label: "Skills", assetType: nd.AssetSkill},
		{label: "Commands", assetType: nd.AssetCommand},
		{label: "Rules", assetType: nd.AssetRule},
		{label: "Context", assetType: nd.AssetContext},
		{label: "Agents", assetType: nd.AssetAgent},
		{label: "Output styles", assetType: nd.AssetOutputStyle},
		{label: "Hooks", assetType: nd.AssetHook},
	}
}

// deployScreen implements the 4-step deploy flow:
// pickType -> selectAssets -> running -> result.
type deployScreen struct {
	svc    Services
	styles Styles
	isDark bool
	step   deployStep

	// pickType step
	typeForm   *huh.Form
	typeChoice string
	scanning   bool // H1: guards against double-fire after type form completion

	// pickAgents step
	agentForm      *huh.Form
	selectedAgents []string // resolved target agent names (picker, config, or single detected)

	// selectAssets step
	assetForm   *huh.Form
	selected    []string       // "sourceID:type/name" keys
	assets      []*asset.Asset // available (undeployed) assets
	recencyDays int            // recency_days from config (0 = 7-day default)
	deploying   bool           // H1: guards against double-fire after asset form completion

	// running step
	progress progressBar

	// result step
	succeeded []deploy.DeployResult
	failed    []deploy.DeployError
	dryRun    bool                   // true when result is a dry-run preview
	dryReqs   []deploy.DeployRequest // populated for dry-run display

	// conflict resolution (deployConflictConfirm step)
	reqs              []deploy.DeployRequest // all original requests from startDeploy
	firstSucceeded    []deploy.DeployResult  // succeeded before conflict resolution
	firstFailed       []deploy.DeployError   // non-conflict failures before resolution
	conflictFails     []deploy.DeployError   // failures with ConflictError
	conflictReqs      []deploy.DeployRequest // same requests re-built with ForceReplace=true
	conflictAgents    []string               // target agent name per conflictReqs entry (parallel)
	conflictForm      *huh.Form
	conflictConfirmed bool // captured by huh.Confirm
	conflictAnswered  bool // guards against double-fire

	err  error
	info string // non-error informational message (e.g. "all deployed")

	// result scrolling
	height      int
	scroll      listScroll
	resultLines []string
}

// deployDoneMsg signals that the background deploy goroutine completed.
type deployDoneMsg struct {
	succeeded []deploy.DeployResult
	failed    []deploy.DeployError
}

// scanDoneMsg signals that the background scan+filter completed.
type scanDoneMsg struct {
	assets      []*asset.Asset
	recencyDays int // resolved recency_days from config (0 = 7-day default)
	err         error
}

func newDeployScreen(svc Services, styles Styles, isDark bool) *deployScreen {
	ds := &deployScreen{
		svc:    svc,
		styles: styles,
		isDark: isDark,
		step:   deployPickType,
	}

	entries := typeDisplayNames()
	opts := make([]huh.Option[string], len(entries))
	for i, e := range entries {
		opts[i] = huh.NewOption(e.label, string(e.assetType))
	}

	ds.typeForm = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Asset type").
				Options(opts...).
				Value(&ds.typeChoice),
		),
	).WithTheme(huh.ThemeFunc(huh.ThemeCatppuccin))

	return ds
}

// Screen interface
func (ds *deployScreen) Title() string { return "Deploy" }

func (ds *deployScreen) InputActive() bool {
	return ds.step == deployPickType || ds.step == deployPickAgents ||
		ds.step == deploySelectAssets || ds.step == deployConflictConfirm
}

// FullHelpItems returns step-specific help items for the deploy screen.
// MultiSelect steps show "x/space toggle" instead of the default "enter select".
func (ds *deployScreen) FullHelpItems() []HelpItem {
	switch ds.step {
	case deployPickType:
		return []HelpItem{
			{"esc", "back"},
			{"j/k", "navigate"},
			{"enter", "select"},
			{"q", "quit"},
		}
	case deployPickAgents:
		return []HelpItem{
			{"esc", "back"},
			{"j/k", "navigate"},
			{"x/space", "toggle"},
			{"enter", "confirm"},
			{"q", "quit"},
		}
	case deploySelectAssets:
		return []HelpItem{
			{"esc", "back"},
			{"j/k", "navigate"},
			{"x/space", "toggle"},
			{"enter", "confirm"},
			{"q", "quit"},
		}
	case deployConflictConfirm:
		return []HelpItem{
			{"h/l", "yes/no"},
			{"enter", "confirm"},
			{"q", "quit"},
		}
	default:
		return []HelpItem{
			{"esc", "back"},
			{"enter", "return"},
			{"q", "quit"},
		}
	}
}

// HelpSections groups the deploy keybindings under headings for the '?' overlay,
// including step-specific tips (e.g. the multiselect toggle).
func (ds *deployScreen) HelpSections() []HelpSection {
	switch ds.step {
	case deploySelectAssets:
		return []HelpSection{
			{Title: "Navigation", Items: []HelpItem{
				{"j/k", "navigate"},
				{"esc", "back"},
				{"q", "quit"},
			}},
			{Title: "Selection", Items: []HelpItem{
				{"x/space", "toggle asset"},
				{"enter", "confirm"},
			}},
			{Title: "Tips", Items: []HelpItem{
				{GlyphDot, "Toggle assets with x or space, then press enter to deploy."},
			}},
		}
	case deployConflictConfirm:
		return []HelpSection{
			{Title: "Confirm", Items: []HelpItem{
				{"h/l", "yes/no"},
				{"enter", "confirm"},
				{"q", "quit"},
			}},
		}
	case deployPickType:
		return []HelpSection{
			{Title: "Navigation", Items: []HelpItem{
				{"j/k", "navigate"},
				{"enter", "select"},
				{"esc", "back"},
				{"q", "quit"},
			}},
		}
	default:
		return []HelpSection{
			{Title: "Result", Items: []HelpItem{
				{"enter", "return"},
				{"esc", "back"},
				{"q", "quit"},
			}},
		}
	}
}

// Init initializes the type picker form.
func (ds *deployScreen) Init() tea.Cmd {
	return ds.typeForm.Init()
}

// Update handles messages for each step of the deploy flow.
func (ds *deployScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		ds.height = msg.Height
		return ds, nil

	case deployDoneMsg:
		// Second pass (force-replace run): merge with first-run results.
		if ds.conflictReqs != nil {
			ds.step = deployResult
			ds.succeeded = append(ds.firstSucceeded, msg.succeeded...)
			ds.failed = append(ds.firstFailed, msg.failed...)
			ds.resultLines = nil
			ds.logOplog()
			return ds, func() tea.Msg { return RefreshHeaderMsg{} }
		}

		// First pass: partition failures into conflicts vs others.
		var conflictFails, otherFails []deploy.DeployError
		for _, f := range msg.failed {
			var ce *nd.ConflictError
			if errors.As(f.Err, &ce) {
				conflictFails = append(conflictFails, f)
			} else {
				otherFails = append(otherFails, f)
			}
		}

		if len(conflictFails) > 0 {
			// Offer conflict resolution before showing the result.
			ds.firstSucceeded = msg.succeeded
			ds.firstFailed = otherFails
			ds.conflictFails = conflictFails
			ds.conflictReqs, ds.conflictAgents = ds.buildForceRequests(conflictFails)
			ds.step = deployConflictConfirm
			return ds, ds.buildConflictForm()
		}

		// No conflicts — go straight to result.
		ds.step = deployResult
		ds.succeeded = msg.succeeded
		ds.failed = msg.failed
		ds.resultLines = nil
		ds.logOplog()
		return ds, func() tea.Msg { return RefreshHeaderMsg{} }

	case scanDoneMsg:
		if msg.err != nil {
			ds.err = msg.err
			ds.step = deployResult // M6: avoid dead-end re-triggering
			return ds, nil
		}
		if len(msg.assets) == 0 {
			typeName := "all"
			if ds.typeChoice != "" {
				typeName = ds.typeChoice
			}
			ds.info = AllDeployed(typeName) // L7: not an error
			ds.step = deployResult          // M6: avoid dead-end re-triggering
			return ds, nil
		}
		ds.assets = msg.assets
		ds.recencyDays = msg.recencyDays
		ds.step = deploySelectAssets
		ds.buildAssetForm()
		return ds, ds.assetForm.Init()
	}

	switch ds.step {
	case deployPickType:
		return ds.updatePickType(msg)
	case deployPickAgents:
		return ds.updatePickAgents(msg)
	case deploySelectAssets:
		return ds.updateSelectAssets(msg)
	case deployConflictConfirm:
		return ds.updateConflictConfirm(msg)
	case deployResult:
		return ds.updateResult(msg)
	}

	return ds, nil
}

// View renders the current step.
func (ds *deployScreen) View() tea.View {
	if ds.err != nil {
		return tea.NewView(fmt.Sprintf("  %s\n\n  %s\n\n  %s",
			ds.styles.Danger.Render("Error"),
			ds.err.Error(),
			ds.styles.Subtle.Render("Press esc to go back.")))
	}
	if ds.info != "" {
		return tea.NewView(fmt.Sprintf("  %s\n\n  %s",
			ds.info,
			ds.styles.Subtle.Render("Press esc to go back.")))
	}

	switch ds.step {
	case deployPickType:
		return tea.NewView(ds.typeForm.View())

	case deployPickAgents:
		if ds.agentForm != nil {
			return tea.NewView(ds.agentForm.View())
		}
		return tea.NewView("  Loading agents...")

	case deploySelectAssets:
		if ds.assetForm != nil {
			return tea.NewView(ds.assetForm.View())
		}
		return tea.NewView("  Loading assets...")

	case deployRunning:
		msg := fmt.Sprintf("Deploying %d asset(s)...", len(ds.reqs))
		return tea.NewView(fmt.Sprintf("  %s",
			ds.styles.Primary.Render(msg)))

	case deployConflictConfirm:
		return ds.viewConflictConfirm()

	case deployResult:
		return ds.viewResult()
	}

	return tea.NewView("")
}

// updatePickType delegates to the type picker form and transitions on completion.
func (ds *deployScreen) updatePickType(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok && keyMsg.String() == "esc" {
		return ds, func() tea.Msg { return BackMsg{} }
	}
	// H1: guard against double-fire after form completion
	if ds.scanning {
		return ds, nil
	}

	model, cmd := ds.typeForm.Update(msg)
	if f, ok := model.(*huh.Form); ok {
		ds.typeForm = f
	}

	if ds.typeForm.State == huh.StateCompleted {
		return ds.afterPickType()
	}

	if ds.typeForm.State == huh.StateAborted {
		return ds, func() tea.Msg { return BackMsg{} }
	}

	return ds, cmd
}

// afterPickType resolves the target agents once the type form completes.
// Resolution order: config default_deploy_agents > single detected agent >
// interactive multi-select picker (when 2+ agents are detected).
func (ds *deployScreen) afterPickType() (tea.Model, tea.Cmd) {
	if configured := ds.configuredAgents(); len(configured) > 0 {
		ds.selectedAgents = configured
		ds.scanning = true
		return ds, ds.startScan()
	}

	detected := ds.detectedAgents()
	if len(detected) >= 2 {
		ds.buildAgentForm(detected)
		ds.step = deployPickAgents
		return ds, ds.agentForm.Init()
	}
	if len(detected) == 1 {
		ds.selectedAgents = []string{detected[0].Name}
	}
	// Zero detected: leave selectedAgents empty; startDeploy falls back to the
	// active agent (preserving the existing single-agent behavior).
	ds.scanning = true
	return ds, ds.startScan()
}

// configuredAgents returns the config's default_deploy_agents, or nil.
func (ds *deployScreen) configuredAgents() []string {
	sm, err := ds.svc.SourceManager()
	if err != nil || sm == nil {
		return nil
	}
	return sm.Config().DefaultDeployAgents
}

// detectedAgents returns the agents currently detected on this system.
func (ds *deployScreen) detectedAgents() []agent.Agent {
	reg, err := ds.svc.AgentRegistry()
	if err != nil || reg == nil {
		return nil
	}
	reg.Detect()
	var out []agent.Agent
	for _, a := range reg.All() {
		if a.Detected {
			out = append(out, a)
		}
	}
	return out
}

// buildAgentForm creates the multi-select form from the detected agents.
func (ds *deployScreen) buildAgentForm(detected []agent.Agent) {
	opts := make([]huh.Option[string], len(detected))
	for i, a := range detected {
		opts[i] = huh.NewOption(a.Name, a.Name)
	}
	ds.agentForm = huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select target agents").
				Options(opts...).
				Value(&ds.selectedAgents),
		),
	).WithTheme(huh.ThemeFunc(huh.ThemeCatppuccin))
}

// updatePickAgents delegates to the agent multi-select form and, on completion,
// advances to the scan step with the chosen agents.
func (ds *deployScreen) updatePickAgents(msg tea.Msg) (tea.Model, tea.Cmd) {
	// H1: guard against double-fire after the picker completes and a scan starts.
	if ds.scanning {
		return ds, nil
	}
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok && keyMsg.String() == "esc" {
		return ds, func() tea.Msg { return BackMsg{} }
	}
	if ds.agentForm == nil {
		return ds, nil
	}

	model, cmd := ds.agentForm.Update(msg)
	if f, ok := model.(*huh.Form); ok {
		ds.agentForm = f
	}

	if ds.agentForm.State == huh.StateCompleted {
		if len(ds.selectedAgents) == 0 {
			// Reject an empty selection: rebuild the form so the user can retry.
			ds.buildAgentForm(ds.detectedAgents())
			return ds, ds.agentForm.Init()
		}
		ds.scanning = true
		return ds, ds.startScan()
	}

	if ds.agentForm.State == huh.StateAborted {
		return ds, func() tea.Msg { return BackMsg{} }
	}

	return ds, cmd
}

// startScan kicks off an async scan to find undeployed assets.
func (ds *deployScreen) startScan() tea.Cmd {
	svc := ds.svc
	typeFilter := nd.AssetType(ds.typeChoice)

	return func() tea.Msg {
		summary, err := svc.ScanIndex()
		if err != nil {
			return scanDoneMsg{err: err}
		}
		if summary == nil || summary.Index == nil {
			return scanDoneMsg{err: fmt.Errorf("no asset index available")}
		}

		agentAlias := ""
		if ag, err := svc.ActiveAgent(); err == nil {
			agentAlias = ag.SourceAlias
		}

		var allAssets []*asset.Asset
		if typeFilter == "" {
			allAssets = summary.Index.FilterByAgent(agentAlias)
		} else {
			allAssets = summary.Index.ByTypeFiltered(typeFilter, agentAlias)
		}

		// Filter to only deployable types
		var deployable []*asset.Asset
		for _, a := range allAssets {
			if a.Type.IsDeployable() {
				deployable = append(deployable, a)
			}
		}

		// Get deployed assets to filter them out
		store := svc.StateStore()
		var deployed []state.Deployment
		if store != nil {
			st, _, err := store.Load()
			if err == nil && st != nil {
				deployed = st.Deployments
			}
		}

		// Resolve the recency window from config so the picker can flag newly
		// modified assets (mirrors browseScreen.Init). 0 = 7-day default.
		recencyDays := 0
		if sm, err := svc.SourceManager(); err == nil && sm != nil {
			if cfg := sm.Config(); cfg != nil {
				recencyDays = cfg.RecencyDays
			}
		}

		undeployed := filterUndeployed(deployable, deployed)
		return scanDoneMsg{assets: undeployed, recencyDays: recencyDays}
	}
}

// assetOptionLabel builds a deploy-picker label for an asset: "name  source",
// with the description appended when present and a styled "new" badge when the
// asset's source file was modified within window. Deployed-badge logic is
// available via the shared helper, but ds.assets here is the undeployed set so
// it is a no-op in this picker today.
func assetOptionLabel(styles Styles, a *asset.Asset, window time.Duration) string {
	label := fmt.Sprintf("%s  %s", a.Name, a.SourceID)
	if a.Meta != nil && a.Meta.Description != "" {
		label = fmt.Sprintf("%s  %s  %s", a.Name, a.SourceID, a.Meta.Description)
	}
	if isNew(a, window) {
		label += "  " + styles.NewBadge()
	}
	return label
}

// buildAssetForm creates the multi-select form from the available (undeployed) assets.
func (ds *deployScreen) buildAssetForm() {
	window := recencyWindow(ds.recencyDays)
	opts := make([]huh.Option[string], len(ds.assets))
	for i, a := range ds.assets {
		opts[i] = huh.NewOption(assetOptionLabel(ds.styles, a, window), assetKey(a))
	}

	ds.assetForm = huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select assets to deploy").
				Options(opts...).
				Height(10).
				Value(&ds.selected),
		),
	).WithTheme(huh.ThemeFunc(huh.ThemeCatppuccin))
}

// updateSelectAssets delegates to the asset selection form and starts deployment.
func (ds *deployScreen) updateSelectAssets(msg tea.Msg) (tea.Model, tea.Cmd) {
	// H1: guard against double-fire after form completion; deploying blocks all input.
	if ds.deploying {
		return ds, nil
	}
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok && keyMsg.String() == "esc" {
		return ds, func() tea.Msg { return BackMsg{} }
	}
	if ds.assetForm == nil {
		return ds, nil
	}

	model, cmd := ds.assetForm.Update(msg)
	if f, ok := model.(*huh.Form); ok {
		ds.assetForm = f
	}

	if ds.assetForm.State == huh.StateCompleted {
		ds.deploying = true
		return ds, ds.startDeploy()
	}

	if ds.assetForm.State == huh.StateAborted {
		return ds, func() tea.Msg { return BackMsg{} }
	}

	return ds, cmd
}

// startDeploy transitions to the running step and kicks off the deploy goroutine.
func (ds *deployScreen) startDeploy() tea.Cmd {
	if len(ds.selected) == 0 {
		// Nothing selected — go back
		return func() tea.Msg { return BackMsg{} }
	}

	reqs := ds.buildBaseRequests()
	ds.reqs = reqs

	// H2: Dry-run mode — show preview without executing
	if ds.svc.IsDryRun() {
		ds.step = deployResult
		ds.dryRun = true
		ds.dryReqs = reqs
		ds.resultLines = nil
		return func() tea.Msg { return RefreshHeaderMsg{} }
	}

	// Build one engine per selected agent and run the same requests through each.
	batches, err := ds.firstPassBatches(reqs)
	if err != nil {
		ds.err = fmt.Errorf("deploy engine: %w", err)
		return nil
	}
	if len(batches) == 0 {
		ds.err = fmt.Errorf("deploy engine not available")
		return nil
	}

	ds.step = deployRunning
	ds.progress = newProgressBar(40)

	// M3: Use bulk API for single lock cycle + auto-snapshot (per agent).
	return ds.runBatchesCmd(batches)
}

// buildBaseRequests builds the per-asset deploy requests (one set, agent-agnostic).
func (ds *deployScreen) buildBaseRequests() []deploy.DeployRequest {
	selectedSet := make(map[string]bool, len(ds.selected))
	for _, key := range ds.selected {
		selectedSet[key] = true
	}

	// M4: Read symlink strategy from config (flag > config > default)
	strategy := nd.SymlinkAbsolute
	if sm, err := ds.svc.SourceManager(); err == nil && sm != nil {
		if cfg := sm.Config(); cfg.SymlinkStrategy != "" {
			strategy = cfg.SymlinkStrategy
		}
	}

	// Build deploy requests (C1: include ProjectRoot)
	scope := ds.svc.GetScope()
	projectRoot := ds.svc.GetProjectRoot()
	var reqs []deploy.DeployRequest
	for _, a := range ds.assets {
		if !selectedSet[assetKey(a)] {
			continue
		}
		reqs = append(reqs, deploy.DeployRequest{
			Asset:       *a,
			Scope:       scope,
			ProjectRoot: projectRoot,
			Origin:      nd.OriginManual,
			Strategy:    strategy,
		})
	}
	return reqs
}

// firstPassBatches builds one deploy batch per selected agent, each running the
// same base requests. When no agent was resolved (none detected), it falls back
// to a single batch bound to the active agent.
func (ds *deployScreen) firstPassBatches(base []deploy.DeployRequest) ([]agentDeploy, error) {
	names := ds.selectedAgents
	if len(names) == 0 {
		names = []string{""} // "" => active-agent fallback
	}
	batches := make([]agentDeploy, 0, len(names))
	for _, name := range names {
		eng, err := ds.engineFor(name)
		if err != nil {
			return nil, err
		}
		batches = append(batches, agentDeploy{name: name, engine: eng, reqs: base})
	}
	return batches, nil
}

// engineFor resolves a deploy engine for the named agent. An empty name resolves
// the active-agent engine (single-agent fallback).
func (ds *deployScreen) engineFor(name string) (*deploy.Engine, error) {
	if name == "" {
		eng, err := ds.svc.DeployEngine()
		if err != nil {
			return nil, err
		}
		if eng == nil {
			return nil, fmt.Errorf("deploy engine not available")
		}
		return eng, nil
	}
	reg, err := ds.svc.AgentRegistry()
	if err != nil {
		return nil, err
	}
	if reg == nil {
		return nil, fmt.Errorf("agent registry not available")
	}
	ag, err := reg.Get(name)
	if err != nil {
		return nil, err
	}
	eng, err := ds.svc.DeployEngineFor(ag)
	if err != nil {
		return nil, err
	}
	if eng == nil {
		return nil, fmt.Errorf("deploy engine not available")
	}
	return eng, nil
}

// runBatchesCmd returns the command that runs the batches, using the single-agent
// path when there is exactly one batch.
func (ds *deployScreen) runBatchesCmd(batches []agentDeploy) tea.Cmd {
	if len(batches) == 1 {
		b := batches[0]
		return deployBulkCmd(b.engine.DeployBulk, b.reqs, b.name)
	}
	return deployBatchesCmd(batches)
}

// runBulk executes one agent's bulk deploy. On a total failure it tags every
// request's error with the agent name so results can be labeled per agent.
func runBulk(deployer func([]deploy.DeployRequest) (*deploy.BulkDeployResult, error), reqs []deploy.DeployRequest, agentName string) ([]deploy.DeployResult, []deploy.DeployError) {
	result, err := deployer(reqs)
	if err != nil {
		var failed []deploy.DeployError
		for _, req := range reqs {
			failed = append(failed, deploy.DeployError{
				AssetName:  req.Asset.Name,
				AssetType:  req.Asset.Type,
				SourcePath: req.Asset.SourcePath,
				Agent:      agentName,
				Err:        err,
			})
		}
		return nil, failed
	}
	return result.Succeeded, result.Failed
}

// deployBulkCmd creates a tea.Cmd that deploys all requests to a single agent
// via the bulk API. agentName labels any total-failure errors.
func deployBulkCmd(deployer func([]deploy.DeployRequest) (*deploy.BulkDeployResult, error), reqs []deploy.DeployRequest, agentName string) tea.Cmd {
	return func() tea.Msg {
		succeeded, failed := runBulk(deployer, reqs, agentName)
		return deployDoneMsg{succeeded: succeeded, failed: failed}
	}
}

// deployBatchesCmd runs each agent's batch and merges the results into a single
// deployDoneMsg. Successes and failures carry their target agent name.
func deployBatchesCmd(batches []agentDeploy) tea.Cmd {
	return func() tea.Msg {
		var msg deployDoneMsg
		for _, b := range batches {
			succeeded, failed := runBulk(b.engine.DeployBulk, b.reqs, b.name)
			msg.succeeded = append(msg.succeeded, succeeded...)
			msg.failed = append(msg.failed, failed...)
		}
		return msg
	}
}

// updateResult handles key presses at the result step.
// H4: Only "enter" reaches here — esc/q are intercepted by root model.
func (ds *deployScreen) updateResult(msg tea.Msg) (tea.Model, tea.Cmd) {
	if len(ds.resultLines) == 0 {
		ds.resultLines = splitLines(ds.buildResultContent())
	}
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch HandleScrollKeys(keyMsg, &ds.scroll, len(ds.resultLines), ds.contentHeight()) {
		case scrollKeyHandled:
			return ds, nil
		case scrollKeyPopToRoot:
			return ds, tea.Batch(
				func() tea.Msg { return PopToRootMsg{} },
				func() tea.Msg { return RefreshHeaderMsg{} },
			)
		}
	}
	return ds, nil
}

func (ds *deployScreen) contentHeight() int {
	return ContentHeight(ds.height, 4)
}

// buildResultContent renders the full deployment result as a string.
func (ds *deployScreen) buildResultContent() string {
	var b strings.Builder

	// H2: Dry-run preview
	if ds.dryRun {
		agents := ds.selectedAgents
		labeled := len(agents) > 0
		if !labeled {
			agents = []string{""}
		}
		fmt.Fprintf(&b, "  %s Would deploy %d asset(s):\n\n",
			ds.styles.Warning.Render(GlyphDryRun), len(ds.dryReqs)*len(agents))
		for _, name := range agents {
			for _, req := range ds.dryReqs {
				if name != "" {
					fmt.Fprintf(&b, "    %s %s/%s from %s %s %s\n",
						GlyphArrow, req.Asset.Type, req.Asset.Name, req.Asset.SourceID, GlyphArrow, name)
				} else {
					fmt.Fprintf(&b, "    %s %s/%s from %s\n",
						GlyphArrow, req.Asset.Type, req.Asset.Name, req.Asset.SourceID)
				}
			}
		}
		fmt.Fprintf(&b, "\n  %s", ds.styles.Subtle.Render("Press enter to return."))
		return b.String()
	}

	// M12: Summary shows succeeded count, not total/total
	total := len(ds.succeeded) + len(ds.failed)
	fmt.Fprintf(&b, "  Deployment complete: %d of %d succeeded\n\n", len(ds.succeeded), total)

	if len(ds.succeeded) > 0 {
		fmt.Fprintf(&b, "  %s\n", ds.styles.Success.Render(
			fmt.Sprintf("%d succeeded", len(ds.succeeded))))
		for _, r := range ds.succeeded {
			fmt.Fprintf(&b, "    %s %s%s\n",
				GlyphOK, fmt.Sprintf("%s/%s", r.Deployment.AssetType, r.Deployment.AssetName),
				agentSuffix(r.Deployment.Agent))
		}
		b.WriteString("\n")
	}

	if len(ds.failed) > 0 {
		fmt.Fprintf(&b, "  %s\n", ds.styles.Danger.Render(
			fmt.Sprintf("%d failed", len(ds.failed))))
		for _, f := range ds.failed {
			fmt.Fprintf(&b, "    %s %s%s: %v\n",
				GlyphBroken, fmt.Sprintf("%s/%s", f.AssetType, f.AssetName),
				agentSuffix(f.Agent), f.Err)
		}
		b.WriteString("\n")
	}

	if len(ds.succeeded) == 0 && len(ds.failed) == 0 {
		b.WriteString("  No assets were deployed.\n\n")
	}

	fmt.Fprintf(&b, "  %s", ds.styles.Subtle.Render("Press enter to return."))

	return b.String()
}

// viewResult renders the result step with j/k scrolling when the list exceeds the terminal height.
func (ds *deployScreen) viewResult() tea.View {
	if len(ds.resultLines) == 0 {
		ds.resultLines = splitLines(ds.buildResultContent())
	}
	return tea.NewView(RenderScrolledLines(ds.styles, &ds.scroll, ds.resultLines, ds.contentHeight()))
}

// --- Conflict resolution ---

// buildForceRequests rebuilds the requests for failed assets with
// ForceReplace=true, returning a parallel slice of the target agent name for
// each request so the re-run can be routed to the correct engine.
func (ds *deployScreen) buildForceRequests(conflictFails []deploy.DeployError) ([]deploy.DeployRequest, []string) {
	lookup := make(map[string]deploy.DeployRequest, len(ds.reqs))
	for _, req := range ds.reqs {
		lookup[fmt.Sprintf("%s/%s", req.Asset.Type, req.Asset.Name)] = req
	}
	reqs := make([]deploy.DeployRequest, 0, len(conflictFails))
	agents := make([]string, 0, len(conflictFails))
	for _, f := range conflictFails {
		key := fmt.Sprintf("%s/%s", f.AssetType, f.AssetName)
		if req, ok := lookup[key]; ok {
			req.ForceReplace = true
			reqs = append(reqs, req)
			agents = append(agents, f.Agent)
		}
	}
	return reqs, agents
}

// conflictBatches groups the force-replace requests by their target agent and
// builds a deploy engine for each group.
func (ds *deployScreen) conflictBatches() ([]agentDeploy, error) {
	order := make([]string, 0)
	grouped := make(map[string][]deploy.DeployRequest)
	for i, req := range ds.conflictReqs {
		name := ""
		if i < len(ds.conflictAgents) {
			name = ds.conflictAgents[i]
		}
		if _, ok := grouped[name]; !ok {
			order = append(order, name)
		}
		grouped[name] = append(grouped[name], req)
	}
	batches := make([]agentDeploy, 0, len(order))
	for _, name := range order {
		eng, err := ds.engineFor(name)
		if err != nil {
			return nil, err
		}
		batches = append(batches, agentDeploy{name: name, engine: eng, reqs: grouped[name]})
	}
	return batches, nil
}

// buildConflictForm initialises the yes/no confirmation form for force-replace.
func (ds *deployScreen) buildConflictForm() tea.Cmd {
	ds.conflictAnswered = false
	ds.conflictConfirmed = false
	n := len(ds.conflictFails)
	title := fmt.Sprintf("%d asset(s) conflict with existing files. Replace them?", n)
	ds.conflictForm = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(title).
				Affirmative("Replace").
				Negative("Cancel").
				Value(&ds.conflictConfirmed),
		),
	).WithTheme(huh.ThemeFunc(huh.ThemeCatppuccin))
	return ds.conflictForm.Init()
}

// updateConflictConfirm handles input during the conflict resolution step.
func (ds *deployScreen) updateConflictConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ds.conflictAnswered || ds.conflictForm == nil {
		return ds, nil
	}

	model, cmd := ds.conflictForm.Update(msg)
	if f, ok := model.(*huh.Form); ok {
		ds.conflictForm = f
	}

	if ds.conflictForm.State == huh.StateCompleted {
		ds.conflictAnswered = true
		if !ds.conflictConfirmed {
			return ds.cancelConflictResolution()
		}
		// User said replace: re-run with ForceReplace=true, per target agent.
		ds.step = deployRunning
		batches, err := ds.conflictBatches()
		if err != nil || len(batches) == 0 {
			ds.err = fmt.Errorf("deploy engine not available")
			return ds, nil
		}
		return ds, ds.runBatchesCmd(batches)
	}

	if ds.conflictForm.State == huh.StateAborted {
		return ds.cancelConflictResolution()
	}

	return ds, cmd
}

// cancelConflictResolution moves to the result step with the first-run outcomes.
func (ds *deployScreen) cancelConflictResolution() (tea.Model, tea.Cmd) {
	ds.step = deployResult
	ds.succeeded = ds.firstSucceeded
	ds.failed = append(ds.firstFailed, ds.conflictFails...)
	ds.resultLines = nil
	ds.logOplog()
	return ds, func() tea.Msg { return RefreshHeaderMsg{} }
}

// logOplog writes the completed deploy operation to the oplog using ds.succeeded/failed.
func (ds *deployScreen) logOplog() {
	if ol := ds.svc.OpLog(); ol != nil {
		var identities []asset.Identity
		for _, r := range ds.succeeded {
			identities = append(identities, r.Deployment.Identity())
		}
		_ = ol.Log(oplog.LogEntry{
			Timestamp: time.Now(),
			Operation: oplog.OpDeploy,
			Assets:    identities,
			Scope:     ds.svc.GetScope(),
			Succeeded: len(ds.succeeded),
			Failed:    len(ds.failed),
		})
	}
}

// viewConflictConfirm renders the conflict list and the replace/cancel form.
func (ds *deployScreen) viewConflictConfirm() tea.View {
	var b strings.Builder
	fmt.Fprintf(&b, "  %s %d asset(s) already exist at the destination:\n\n",
		ds.styles.Warning.Render(GlyphBroken), len(ds.conflictFails))
	for _, f := range ds.conflictFails {
		fmt.Fprintf(&b, "    %s  %s/%s\n",
			ds.styles.Warning.Render(GlyphArrow), f.AssetType, f.AssetName)
	}
	if ds.conflictForm != nil {
		b.WriteString("\n")
		b.WriteString(ds.conflictForm.View())
	}
	return tea.NewView(b.String())
}

// --- helpers ---

// agentSuffix renders " -> <agent>" when an agent name is set, else "".
func agentSuffix(name string) string {
	if name == "" {
		return ""
	}
	return fmt.Sprintf(" %s %s", GlyphArrow, name)
}

// assetKey returns a unique key for an asset: "sourceID:type/name".
func assetKey(a *asset.Asset) string {
	return fmt.Sprintf("%s:%s/%s", a.SourceID, a.Type, a.Name)
}

// deploymentKey returns a unique key for a deployment: "sourceID:type/name".
func deploymentKey(d state.Deployment) string {
	return fmt.Sprintf("%s:%s/%s", d.SourceID, d.AssetType, d.AssetName)
}

// filterUndeployed returns only assets that are not already deployed.
func filterUndeployed(all []*asset.Asset, deployed []state.Deployment) []*asset.Asset {
	deployedSet := make(map[string]bool, len(deployed))
	for _, d := range deployed {
		deployedSet[deploymentKey(d)] = true
	}

	var available []*asset.Asset
	for _, a := range all {
		if !deployedSet[assetKey(a)] {
			available = append(available, a)
		}
	}
	return available
}
