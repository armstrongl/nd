package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"

	"github.com/armstrongl/nd/internal/asset"
	"github.com/armstrongl/nd/internal/deploy"
	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/oplog"
	"github.com/armstrongl/nd/internal/state"
)

type deployStep int

const (
	deployPickType deployStep = iota
	deploySelectAssets
	deployRunning
	deployResult
)

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

	// selectAssets step
	assetForm *huh.Form
	selected  []string       // "sourceID:type/name" keys
	assets    []*asset.Asset // available (undeployed) assets
	deploying bool           // H1: guards against double-fire after asset form completion

	// running step
	progress progressBar

	// result step
	succeeded []deploy.DeployResult
	failed    []deploy.DeployError
	dryRun    bool                 // true when result is a dry-run preview
	dryReqs   []deploy.DeployRequest // populated for dry-run display

	err  error
	info string // non-error informational message (e.g. "all deployed")
}

// deployDoneMsg signals that the background deploy goroutine completed.
type deployDoneMsg struct {
	succeeded []deploy.DeployResult
	failed    []deploy.DeployError
}

// scanDoneMsg signals that the background scan+filter completed.
type scanDoneMsg struct {
	assets []*asset.Asset
	err    error
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
	return ds.step == deployPickType || ds.step == deploySelectAssets
}

// Init initializes the type picker form.
func (ds *deployScreen) Init() tea.Cmd {
	return ds.typeForm.Init()
}

// Update handles messages for each step of the deploy flow.
func (ds *deployScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case deployDoneMsg:
		ds.step = deployResult
		ds.succeeded = msg.succeeded
		ds.failed = msg.failed
		// M5: Log operation to oplog
		if ol := ds.svc.OpLog(); ol != nil {
			var identities []asset.Identity
			for _, r := range msg.succeeded {
				identities = append(identities, r.Deployment.Identity())
			}
			_ = ol.Log(oplog.LogEntry{
				Timestamp: time.Now(),
				Operation: oplog.OpDeploy,
				Assets:    identities,
				Scope:     ds.svc.GetScope(),
				Succeeded: len(msg.succeeded),
				Failed:    len(msg.failed),
			})
		}
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
		ds.step = deploySelectAssets
		ds.buildAssetForm()
		return ds, ds.assetForm.Init()
	}

	switch ds.step {
	case deployPickType:
		return ds.updatePickType(msg)
	case deploySelectAssets:
		return ds.updateSelectAssets(msg)
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

	case deploySelectAssets:
		if ds.assetForm != nil {
			return tea.NewView(ds.assetForm.View())
		}
		return tea.NewView("  Loading assets...")

	case deployRunning:
		return tea.NewView(fmt.Sprintf("  %s",
			ds.styles.Primary.Render("Deploying...")))

	case deployResult:
		return tea.NewView(ds.viewResult())
	}

	return tea.NewView("")
}

// updatePickType delegates to the type picker form and transitions on completion.
func (ds *deployScreen) updatePickType(msg tea.Msg) (tea.Model, tea.Cmd) {
	// H1: guard against double-fire after form completion
	if ds.scanning {
		return ds, nil
	}

	model, cmd := ds.typeForm.Update(msg)
	if f, ok := model.(*huh.Form); ok {
		ds.typeForm = f
	}

	if ds.typeForm.State == huh.StateCompleted {
		ds.scanning = true
		return ds, ds.startScan()
	}

	if ds.typeForm.State == huh.StateAborted {
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

		var allAssets []*asset.Asset
		if typeFilter == "" {
			allAssets = summary.Index.All()
		} else {
			allAssets = summary.Index.ByType(typeFilter)
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

		undeployed := filterUndeployed(deployable, deployed)
		return scanDoneMsg{assets: undeployed}
	}
}

// buildAssetForm creates the multi-select form from the available (undeployed) assets.
func (ds *deployScreen) buildAssetForm() {
	opts := make([]huh.Option[string], len(ds.assets))
	for i, a := range ds.assets {
		label := fmt.Sprintf("%s  %s", a.Name, a.SourceID)
		if a.Meta != nil && a.Meta.Description != "" {
			label = fmt.Sprintf("%s  %s  %s", a.Name, a.SourceID, a.Meta.Description)
		}
		opts[i] = huh.NewOption(label, assetKey(a))
	}

	ds.assetForm = huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select assets to deploy").
				Options(opts...).
				Value(&ds.selected),
		),
	).WithTheme(huh.ThemeFunc(huh.ThemeCatppuccin))
}

// updateSelectAssets delegates to the asset selection form and starts deployment.
func (ds *deployScreen) updateSelectAssets(msg tea.Msg) (tea.Model, tea.Cmd) {
	// H1: guard against double-fire after form completion
	if ds.deploying || ds.assetForm == nil {
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

	// Build a set of selected keys for lookup
	selectedSet := make(map[string]bool, len(ds.selected))
	for _, key := range ds.selected {
		selectedSet[key] = true
	}

	// M4: Read symlink strategy from config (flag > config > default)
	strategy := nd.SymlinkAbsolute
	if sm, err := ds.svc.SourceManager(); err == nil && sm != nil {
		cfg := sm.Config()
		if cfg.SymlinkStrategy != "" {
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

	// H2: Dry-run mode — show preview without executing
	if ds.svc.IsDryRun() {
		ds.step = deployResult
		ds.dryRun = true
		ds.dryReqs = reqs
		return func() tea.Msg { return RefreshHeaderMsg{} }
	}

	ds.step = deployRunning
	ds.progress = newProgressBar(40)

	eng, err := ds.svc.DeployEngine()
	if err != nil {
		ds.err = fmt.Errorf("deploy engine: %w", err)
		return nil
	}
	if eng == nil {
		ds.err = fmt.Errorf("deploy engine not available")
		return nil
	}

	// M3: Use bulk API for single lock cycle + auto-snapshot
	return deployBulkCmd(eng.DeployBulk, reqs)
}

// deployBulkCmd creates a tea.Cmd that deploys all requests via the bulk API.
func deployBulkCmd(deployer func([]deploy.DeployRequest) (*deploy.BulkDeployResult, error), reqs []deploy.DeployRequest) tea.Cmd {
	return func() tea.Msg {
		result, err := deployer(reqs)
		if err != nil {
			// Total failure — report all as failed
			var failed []deploy.DeployError
			for _, req := range reqs {
				failed = append(failed, deploy.DeployError{
					AssetName:  req.Asset.Name,
					AssetType:  req.Asset.Type,
					SourcePath: req.Asset.SourcePath,
					Err:        err,
				})
			}
			return deployDoneMsg{failed: failed}
		}
		return deployDoneMsg{succeeded: result.Succeeded, failed: result.Failed}
	}
}

// updateResult handles key presses at the result step.
// H4: Only "enter" reaches here — esc/q are intercepted by root model.
func (ds *deployScreen) updateResult(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		if keyMsg.String() == "enter" {
			// M7: PopToRootMsg (matching remove screen behavior)
			return ds, tea.Batch(
				func() tea.Msg { return PopToRootMsg{} },
				func() tea.Msg { return RefreshHeaderMsg{} },
			)
		}
	}
	return ds, nil
}

// viewResult renders the deployment results.
func (ds *deployScreen) viewResult() string {
	var b strings.Builder

	// H2: Dry-run preview
	if ds.dryRun {
		fmt.Fprintf(&b, "  %s Would deploy %d asset(s):\n\n",
			ds.styles.Warning.Render("[DRY RUN]"), len(ds.dryReqs))
		for _, req := range ds.dryReqs {
			fmt.Fprintf(&b, "    %s %s/%s from %s\n",
				GlyphArrow, req.Asset.Type, req.Asset.Name, req.Asset.SourceID)
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
			fmt.Fprintf(&b, "    %s %s/%s\n",
				GlyphOK, r.Deployment.AssetType, r.Deployment.AssetName)
		}
		b.WriteString("\n")
	}

	if len(ds.failed) > 0 {
		fmt.Fprintf(&b, "  %s\n", ds.styles.Danger.Render(
			fmt.Sprintf("%d failed", len(ds.failed))))
		for _, f := range ds.failed {
			fmt.Fprintf(&b, "    %s %s/%s: %v\n",
				GlyphBroken, f.AssetType, f.AssetName, f.Err)
		}
		b.WriteString("\n")
	}

	if len(ds.succeeded) == 0 && len(ds.failed) == 0 {
		b.WriteString("  No assets were deployed.\n\n")
	}

	fmt.Fprintf(&b, "  %s", ds.styles.Subtle.Render("Press enter to return."))

	return b.String()
}

// --- helpers ---

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
