---
title: "feat: TUI Layer (Bubble Tea) — Interactive Dashboard"
type: feat
status: active
date: 2026-03-15
origin: docs/plans/2026-03-15-tui-layer-design.md
---

## Overview

Wire all completed service layers into an interactive TUI via Bubble Tea v2 and Lip Gloss. This delivers nd's primary interactive experience: a dashboard-centric interface with tabbed asset views, fuzzy deploy, profile switching, snapshot management, and health monitoring. The TUI is a pure presentation layer — all business logic stays in the existing `internal/` packages.

## Problem Statement

nd has complete service layers and a full CLI, but `nd` with no arguments just prints help text (FR-001 requires TUI launch). Users cannot interactively browse assets, deploy via fuzzy search, switch profiles, or monitor deployment health. This plan delivers the TUI layer that makes nd fully interactive.

## Proposed Solution

Nested Bubble Tea models in `internal/tui/` and `internal/tui/components/`. A root `App` model composes components (header, menu, tabbar, table, helpbar) and routes messages. Modal overlays handle deploy (fuzzy finder), profile switch (list picker), and snapshot operations. The TUI depends on service interfaces (not concrete types) for testability.

(see design: `docs/plans/2026-03-15-tui-layer-design.md`)

## Technical Approach

### Architecture

```text
cmd/root.go
  └─ RunE (no args) → tui.Run(app)
       └─ tui.App (root model)
            ├─ components.Menu        (main menu)
            ├─ components.Header      (status bar)
            ├─ components.TabBar      (tab navigation)
            ├─ components.Table       (asset list)
            ├─ components.HelpBar     (context help)
            ├─ components.Picker      (scope/agent)
            ├─ components.FuzzyFinder (deploy)
            ├─ components.ListPicker  (profile/snapshot)
            ├─ components.Prompt      (snapshot name)
            └─ components.Toast       (status messages)
```

### Implementation Phases

#### Phase 0: Prerequisites (no TUI deps)

Small service layer additions needed before TUI work begins. All tasks are independent.

**Task 0.1: Add `ListProfiles()` to `profile.Manager`**

Delegation method wrapping the private `store` field.

```go
// internal/profile/manager.go
func (m *Manager) ListProfiles() ([]ProfileSummary, error) {
    return m.store.ListProfiles()
}
```

- Acceptance: `Manager.ListProfiles()` returns same result as `Store.ListProfiles()`
- Test: add to `internal/profile/manager_test.go`

**Task 0.2: Add `ListSnapshots()` to `profile.Manager`**

```go
func (m *Manager) ListSnapshots() ([]SnapshotSummary, error) {
    return m.store.ListSnapshots()
}
```

- Acceptance: `Manager.ListSnapshots()` returns same result as `Store.ListSnapshots()`
- Test: add to `internal/profile/manager_test.go`

#### Phase 1: Foundation (no component deps)

Core TUI infrastructure. All tasks in this phase are independent and can be parallelized.

**Task 1.1: `internal/tui/services.go` + test**

Consumer-defined interfaces for the service layer. Follows the pattern from `internal/profile/manager.go` (lines 12-23).

```go
package tui

// Deployer abstracts the deploy engine for TUI operations.
type Deployer interface {
    Deploy(deploy.DeployRequest) (*deploy.DeployResult, error)
    DeployBulk([]deploy.DeployRequest) (*deploy.BulkDeployResult, error)
    Remove(deploy.RemoveRequest) error
    RemoveBulk([]deploy.RemoveRequest) (*deploy.BulkRemoveResult, error)
    Status() ([]deploy.StatusEntry, error)
    Check() ([]state.HealthCheck, error)
    Sync() (*deploy.SyncResult, error)
    SetOrigin(asset.Identity, nd.Scope, string, nd.DeployOrigin) error  // F11: asset.Identity, not nd.AssetIdentity
}

// ProfileSwitcher abstracts profile operations for the TUI.
// Uses a simplified interface that captures engine/index at construction time (F12).
// An adapter in cmd/ or internal/tui/ wraps *profile.Manager with pre-bound deps.
type ProfileSwitcher interface {
    ActiveProfile() (string, error)
    Switch(current, target string) (*profile.SwitchResult, error)      // adapter binds engine, index, projectRoot
    Restore(name string) (*profile.RestoreResult, error)               // adapter binds engine, index
    SaveSnapshot(name string) error                                    // FR-020: save current state as snapshot
    ListProfiles() ([]profile.ProfileSummary, error)
    ListSnapshots() ([]profile.SnapshotSummary, error)
}

// SourceScanner abstracts source management for the TUI.
type SourceScanner interface {
    Sources() []source.Source
    Scan() (*sourcemanager.ScanSummary, error)
    SyncSource(sourceID string) error
}

// AgentDetector abstracts agent detection for the TUI.
type AgentDetector interface {
    Detect() agent.DetectionResult
    Default() (*agent.Agent, error)
    All() []agent.Agent
}
```

The `ProfileSwitcher` adapter pattern is needed because `profile.Manager.Switch()` takes `engine DeployEngine, index *asset.Index, projectRoot string` — parameters that the TUI should not manage directly. The adapter captures these at construction time:

```go
// internal/tui/profileadapter.go
type profileAdapter struct {
    mgr         *profile.Manager
    engine      profile.DeployEngine
    indexFn     func() *asset.Index  // lazy: re-scans on each call
    projectRoot string
}

func (a *profileAdapter) Switch(current, target string) (*profile.SwitchResult, error) {
    return a.mgr.Switch(current, target, a.engine, a.indexFn(), a.projectRoot)
}
// ... etc
```

- Acceptance: interfaces compile; adapter satisfies `ProfileSwitcher`
- Test: compile-time assertion test + adapter unit test

**Task 1.2: `internal/tui/styles.go`**

Lip Gloss style constants. Adaptive colors for light/dark terminals.

```go
package tui

import "charm.land/lipgloss/v2"

var (
    StyleHeader     lipgloss.Style
    StyleTabActive  lipgloss.Style
    StyleTabInactive lipgloss.Style
    StyleStatusOK   lipgloss.Style
    StyleStatusBad  lipgloss.Style
    // ... etc per design doc styling table
)
```

- Acceptance: all styles from design doc's styling table are defined
- No test file needed (pure style constants)

**Task 1.3: `internal/tui/keys.go`**

Keybinding definitions using Bubble Tea's `key.Binding` type.

```go
package tui

type KeyMap struct {
    Up, Down, Left, Right key.Binding
    Enter, Esc, Backspace key.Binding
    Deploy, Remove, Sync, Fix key.Binding
    Pin, Profile, Snapshot key.Binding
    Quit key.Binding
    // Deferred to future iteration: Search (/), Help (?)
    Yes, No, Retry key.Binding
}
```

- Acceptance: all keys from design doc's key ownership matrix are defined
- No test file needed (keybinding declarations)

**Task 1.4: `internal/tui/states.go`** (F1/F9 fix: shared types for `tui` and `components`)

App state enum and shared types that both `tui` and `tui/components` packages need. Defined in the `tui` package so the dependency direction is `components` -> `tui` (unidirectional). Components import `tui` for these types; `tui` imports `components` for component structs. No cycle.

```go
package tui

// AppState represents the current state of the TUI application.
type AppState int

const (
    StatePicker AppState = iota
    StateMenu
    StateDashboard
    StateDetail
    StateFuzzy
    StateListPicker
    StatePrompt
    StateConfirm
    StateLoading
)

// ToastLevel indicates the severity of a toast notification.
type ToastLevel int

const (
    ToastInfo ToastLevel = iota
    ToastSuccess
    ToastWarning
    ToastError
)
```

- Acceptance: types defined and importable by `components` package without cycle
- No test file needed (type definitions)

**Task 1.5: `internal/tui/messages.go`**

Custom message types for async operations and cross-component communication.

```go
package tui

// Service result messages
type DeployResultMsg struct { ... }
type SyncResultMsg struct { ... }
type HealthCheckMsg struct { ... }
type ScanCompleteMsg struct { ... }
type ProfileSwitchMsg struct { ... }
type SnapshotSaveMsg struct { ... }
type SnapshotRestoreMsg struct { ... }

// UI state messages
type ToastMsg struct { Message string; Level ToastLevel }
type ErrorMsg struct { Err error }
```

- Acceptance: all async operations from the design have corresponding message types
- No test file needed (type definitions)

**Task 1.6: Update `go.mod` — add Bubble Tea v2, Lip Gloss v2, Bubbles v2**

Bubble Tea v2 uses vanity import paths (not `github.com/charmbracelet/*`). Requires Go 1.24.2+ (project has 1.25.1).

```text
require (
    charm.land/bubbletea/v2  (latest stable, currently v2.0.2+)
    charm.land/lipgloss/v2   (latest stable)
    charm.land/bubbles/v2    (latest stable)
)
```

**v2 API differences from v1 (affects all components):**

- `Init()` returns `(tea.Model, tea.Cmd)` (not just `tea.Cmd`)
- `View()` returns `tea.View` (not `string`) — use `tea.NewView(content)`
- Key events use `tea.KeyMsg` with codes like `tea.KeyUp`, `tea.KeyDown`, `tea.KeyEnter`
- `tea.WindowSizeMsg` sent automatically on startup and resize
- Run `go mod tidy` after adding

#### Phase 2: Basic Components (depends on Phase 1)

Individual UI components, tested in isolation. All tasks are independent and can be parallelized.

**Task 2.1: `internal/tui/components/header.go` + test**

Single-line header bar rendering scope, agent, profile, issue count.

```go
type Header struct {
    Scope      nd.Scope
    Agent      string
    Profile    string
    IssueCount int
    SourceWarn int  // unavailable sources
    Width      int  // terminal width for layout
}

func (h Header) View() string
func (h Header) Update(msg tea.Msg) (Header, tea.Cmd)
```

- Renders: `nd - Global | Claude Code | profile: default | 2 issues`
- Source warning appended when > 0: `| 1 source unavailable`
- Truncates gracefully when terminal is narrow
- Test: verify render output for various configurations, narrow widths

**Task 2.2: `internal/tui/components/tabbar.go` + test**

Horizontal tab bar with issue badges and responsive overflow.

```go
type TabBar struct {
    Tabs      []Tab
    Active    int
    Width     int
}

type Tab struct {
    Label      string
    IssueCount int
}

func (t *TabBar) Update(msg tea.Msg) (*TabBar, tea.Cmd)
func (t TabBar) View() string
```

- Tab order (matches spec A5 asset type order): Overview, Skills, Agents, Commands, Output Styles, Rules, Context, Plugins, Hooks
- Active tab highlighted, issue badges `(N!)` on tabs with issues
- Width >= 95: full names; 60-94: abbreviated; < 60: `< Skills (3/9) >` style
- Test: verify rendering at various widths, tab switching, badge display

**Task 2.3: `internal/tui/components/helpbar.go` + test**

Context-sensitive single-line help bar.

```go
type HelpBar struct {
    Bindings []HelpBinding
    Width    int
}

type HelpBinding struct {
    Key  string
    Help string
}

func (h HelpBar) View() string
func HelpForState(state AppState) []HelpBinding
```

- Different binding sets per app state (from key ownership matrix in design)
- Truncates bindings from right if terminal is too narrow
- Test: verify each state produces correct bindings, truncation behavior

**Task 2.4: `internal/tui/components/toast.go` + test**

Temporary status message that auto-dismisses after 3 seconds.

```go
type Toast struct {
    Message string
    Level   ToastLevel  // info, success, warning, error
    timer   *time.Timer
}

func NewToast(msg string, level ToastLevel) Toast
func (t Toast) View() string
func (t Toast) Update(msg tea.Msg) (Toast, tea.Cmd)
```

- Returns `tea.Tick` command for auto-dismiss
- Renders as inverse bar below header
- Test: verify render, auto-dismiss timing, level styling

**Task 2.5: `internal/tui/components/picker.go` + test**

Scope/agent selection widget for TUI launch.

```go
type Picker struct {
    Scopes    []nd.Scope
    Agents    []agent.Agent
    ScopeIdx  int
    AgentIdx  int
    Field     int  // 0=scope, 1=agent
    Done      bool
}

func NewPicker(agents []agent.Agent, hasProjectDir bool) Picker
func (p *Picker) Update(msg tea.Msg) (*Picker, tea.Cmd)
func (p Picker) View() string
func (p Picker) Selected() (nd.Scope, agent.Agent)
```

- If only one agent, auto-select and show only scope choice
- Default scope: Project if `.claude/` exists in cwd, else Global
- Arrow keys toggle, Enter confirms
- Test: verify navigation, auto-select, default scope logic

#### Phase 3: Table Component (depends on Phase 1)

The table is the most complex component — its own phase.

**Task 3.1: `internal/tui/components/table.go` + test**

Asset table with sortable rows, status/origin icons, and inline expand.

```go
type Table struct {
    Rows       []TableRow
    Selected   int
    Expanded   int       // -1 if none expanded
    Width      int
    Height     int
    Offset     int       // scroll offset
}

type TableRow struct {
    Origin     nd.DeployOrigin
    Health     state.HealthStatus
    Name       string
    Source     string
    Scope      nd.Scope
    StatusText string
    Detail     *RowDetail  // populated on expand
    IsFailed   bool        // for retry-failed tracking
}

type RowDetail struct {
    SourcePath  string
    TargetPath  string
    Scope       string
    Origin      string
    Pinned      bool
    Profile     string
    ErrorDetail string  // for failed/broken assets
    ShadowedBy  string  // FR-016a duplicate warning
    // Context-specific fields from _meta.yaml (FR-016c)
    Description    string  // also shown as trailing column in Context tab
    Tags           []string
    TargetLanguage string
    TargetProject  string
    TargetAgent    string
}

func (t *Table) Update(msg tea.Msg) (*Table, tea.Cmd)
func (t Table) View() string
func (t *Table) SetRows(rows []TableRow)
```

- Columns: origin icon (1), status icon (1), name (flex), source (flex), scope (7), status text (flex)
- Width < 80: hide Source; < 60: hide Source + Scope
- Issues sort to top, then alphabetical
- Enter expands/collapses inline detail
- Scrolling with viewport when rows exceed height
- Context tab: extra description column from `_meta.yaml`
- Test: verify rendering, sorting, expand/collapse, scroll, responsive columns

**Task 3.2: Empty state rendering** (in `table.go`)

When the table has zero rows, render guided onboarding messages.

```go
func (t Table) emptyView() string
```

- Per-tab empty messages from design doc (no sources, no assets, no assets of type)
- Centered, dimmed text with actionable guidance
- Test: verify each empty state message renders correctly

#### Phase 4: Menu Component (depends on Phase 2)

**Task 4.1: `internal/tui/components/menu.go` + test**

Main menu with status summary (FR-028).

```go
type Menu struct {
    Items      []MenuItem
    Selected   int
    Summary    MenuSummary
    Width      int
    Height     int
}

type MenuItem struct {
    Label  string
    Target AppState  // which state to transition to
    TabIdx int       // for asset type items, which tab to select
}

type MenuSummary struct {
    Sources  int
    Deployed int
    Issues   int
    Loading  bool
}

func NewMenu() Menu
func (m *Menu) Update(msg tea.Msg) (*Menu, tea.Cmd)
func (m Menu) View() string
```

- Items: Dashboard, Skills, Agents, Commands, Output Styles, Rules, Context, Plugins, Hooks, Settings, Quit (matches spec A5 order)
- Status summary at bottom loads asynchronously (spinner while loading)
- Up/Down navigate, Enter selects, `q` quits
- Test: verify rendering, navigation, item selection, loading state

#### Phase 5: Modal Components (depends on Phase 1)

Modal overlays. All tasks are independent.

**Task 5.1: `internal/tui/components/fuzzy.go` + test**

Fuzzy finder for deploying assets.

```go
type FuzzyFinder struct {
    Input     textinput.Model  // from bubbles
    Items     []FuzzyItem
    Filtered  []FuzzyItem
    Selected  int
    Width     int
    Height    int
    PreFilter nd.AssetType  // empty = all types
    Loading   bool
}

type FuzzyItem struct {
    Name     string
    Type     nd.AssetType
    Source   string
}

func NewFuzzyFinder(items []FuzzyItem, preFilter nd.AssetType) FuzzyFinder
func (f *FuzzyFinder) Update(msg tea.Msg) (*FuzzyFinder, tea.Cmd)
func (f FuzzyFinder) View() string
func (f FuzzyFinder) SelectedItem() *FuzzyItem
```

- Uses `bubbles/textinput` for the search input
- Fuzzy matching on item names
- Pre-filtered by type when opened from a type-specific tab
- Shows "Scanning sources..." when index is not yet available
- Match count display: `3/47 matches`
- **Context file backup warning (FR-016b):** after user selects a context asset and presses Enter, check if target location has an existing file. If it does, transition to `StateConfirm` with a message that varies: nd-managed symlink gets "Replace deployed context 'X' with 'Y'?"; non-nd file gets "CLAUDE.md exists at target and is NOT managed by nd. Back up and replace?". The deploy engine's `deploy.DeployResult` should indicate whether a backup was created.
- Test: verify filtering, pre-filter, selection, empty states, loading, context backup warning flow

**Task 5.2: `internal/tui/components/listpicker.go` + test**

Generic list picker for profile switch and snapshot restore.

```go
type ListPicker struct {
    Title    string
    Items    []ListPickerItem
    Selected int
    Width    int
    Height   int
}

type ListPickerItem struct {
    Label       string
    Description string
    Active      bool  // for current profile indicator
}

func NewListPicker(title string, items []ListPickerItem) ListPicker
func (l *ListPicker) Update(msg tea.Msg) (*ListPicker, tea.Cmd)
func (l ListPicker) View() string
func (l ListPicker) SelectedItem() *ListPickerItem
```

- Centered modal overlay
- Up/Down navigate, Enter selects, Esc cancels
- Active item marked with `(active)` suffix
- Test: verify rendering, navigation, selection, empty state

**Task 5.3: `internal/tui/components/prompt.go` + test**

Text input modal for snapshot naming.

```go
type Prompt struct {
    Title string
    Input textinput.Model  // from bubbles
    Width int
}

func NewPrompt(title, placeholder string) Prompt
func (p *Prompt) Update(msg tea.Msg) (*Prompt, tea.Cmd)
func (p Prompt) View() string
func (p Prompt) Value() string
```

- Uses `bubbles/textinput`
- Enter submits, Esc cancels
- Test: verify rendering, input, submit, cancel

#### Phase 6: Root App Model (depends on Phases 2-5)

The root model that composes everything.

**Task 6.1: `internal/tui/app.go` + test**

Root Bubble Tea model with state machine and component composition.

```go
// AppState and ToastLevel defined in states.go (Task 1.4)

type ConfirmState struct {
    Active  bool
    Message string
    OnYes   func() tea.Cmd
}

type App struct {
    // Services (interfaces)
    deployer   Deployer
    profiles   ProfileSwitcher
    sources    SourceScanner
    agents     AgentDetector

    // State
    state      AppState
    prevState  AppState
    confirm    ConfirmState
    scope      nd.Scope
    agent      *agent.Agent
    projectRoot string

    // Cached data
    assetIndex *asset.Index
    indexReady bool

    // Components
    header     components.Header
    tabbar     components.TabBar
    table      components.Table
    helpbar    components.HelpBar
    menu       components.Menu
    picker     components.Picker
    fuzzy      components.FuzzyFinder
    listPicker components.ListPicker
    prompt     components.Prompt
    toast      components.Toast

    // Terminal
    width  int
    height int
}

func New(d Deployer, p ProfileSwitcher, s SourceScanner, a AgentDetector, hasProjectDir bool, resolveProjectRoot func() (string, error)) App
func (a App) Init() (tea.Model, tea.Cmd)
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd)
func (a App) View() tea.View
```

- `Init()`: starts in `StatePicker`, returns `Detect()` command
- `Update()`: routes messages per state (key ownership matrix)
- `View()`: composes components vertically; modal overlays render on top
- Handles `tea.WindowSizeMsg` for responsive layout (propagates to all components)
- State transitions follow the navigation state diagram from design
- **Pinned asset removal warning (FR-024a):** when user presses `r` on a row where `RowDetail.Pinned == true`, the confirm message must include a stronger warning: "Asset 'X' is PINNED. Remove anyway?" vs. the regular "Remove 'X'?"
- **Post-deploy reminder (Q13):** when `DeployResultMsg` returns for a hook or output-style asset, display a persistent toast (ToastWarning, no auto-dismiss) with the `settings.json` registration instruction. The deploy engine's result should indicate whether manual registration is needed.
- **Profile switch conflict warning (FR-023):** `profile.SwitchResult.Conflicts` is already populated by `profile.Manager.Switch()`. When conflicts are non-empty, the confirm message must include: "WARNING: N conflicts with pinned/manual assets." after the deploy/remove counts.
- **Plugin tab action restrictions:** when the active tab is Plugins, the `d` (deploy), `r` (remove), and `s` (sync) keys are disabled (not shown in HelpBar). Pressing them shows a toast: "Plugins use `nd export` workflow." Fuzzy finder excludes plugin-type assets from its item list.

Key routing logic:

```go
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch a.state {
    case StatePicker:
        return a.updatePicker(msg)
    case StateMenu:
        return a.updateMenu(msg)
    case StateDashboard:
        return a.updateDashboard(msg)
    case StateConfirm:
        return a.updateConfirm(msg)
    // ... etc
    }
}
```

- Test: state transitions, message routing, component coordination, window resize

**Task 6.2: App async commands** (in `app.go`)

Implement `tea.Cmd` functions for all async service operations.

```go
func (a App) cmdLoadStatus() tea.Cmd      // -> StatusResultMsg
func (a App) cmdRunHealthChecks() tea.Cmd  // -> HealthCheckMsg (batched)
func (a App) cmdScanSources() tea.Cmd      // -> ScanCompleteMsg
func (a App) cmdSyncSources() tea.Cmd      // -> SyncResultMsg
func (a App) cmdDeploy(item) tea.Cmd       // -> DeployResultMsg
func (a App) cmdRemove(row) tea.Cmd        // -> RemoveResultMsg
func (a App) cmdLoadRowDetail(row) tea.Cmd // -> RowDetailMsg (populates RowDetail on expand)
func (a App) cmdProfileSwitch(name) tea.Cmd // -> ProfileSwitchMsg (auto-snapshot handled by profile.Manager internally; retention: last 5 per FR-029a)
func (a App) cmdSnapshotSave(name) tea.Cmd  // -> SnapshotSaveMsg
func (a App) cmdSnapshotRestore(name) tea.Cmd // -> SnapshotRestoreMsg (pre-compute diff: compare current deployments vs snapshot entries to show "Deploys N, removes M" in confirm dialog before execution)
func (a App) cmdRepair() tea.Cmd            // -> RepairResultMsg
```

- Each returns a `tea.Cmd` that runs the service call in a goroutine and returns a typed message
- Health checks return batched results (every 100ms) to reduce render churn
- RowDetail population (F4): when user presses Enter on a row, `cmdLoadRowDetail` constructs `RowDetail` from the `deploy.StatusEntry` and state data (source/target paths, origin, scope, pinned status, error detail, shadowed-by info from FR-016a). This is synchronous (data already cached from `cmdLoadStatus`), so it can be set directly in `updateDashboard` rather than as an async command.
- Test: verify command functions return correct message types (use mock services)

**Task 6.3: App view composition and responsive layout** (in `app.go`)

```go
func (a App) View() string {
    if a.width < 40 || a.height < 10 {
        return tooSmallView(a.width, a.height)
    }
    // Compose: header + tabbar + table + helpbar
    // Overlay modals when active
}
```

- Minimum terminal: 40w x 10h (show "too small" message)
- Modal overlays: centered, dimmed background
- Table gets remaining height after header + tabbar + helpbar
- **Loading state (StateLoading):** renders a centered spinner (from bubbles) with descriptive text ("Loading deployment state...", "Scanning sources..."). Shown during Init before data is available.
- Test: verify layout at various terminal sizes, loading state renders spinner

#### Phase 7: CLI Wiring (depends on Phase 6)

**Task 7.1: Update `cmd/root.go` — launch TUI on no args + `internal/tui/run.go`**

The `tui.Run()` function accepts service interfaces, NOT `*cmd.App` (F6: avoids import cycle between `internal/tui` and `cmd`). Service resolution and error handling happen in `cmd/root.go`.

```go
// cmd/root.go — updated RunE
RunE: func(cmd *cobra.Command, args []string) error {
    // Resolve services with proper error handling (F7)
    eng, err := app.DeployEngine()
    if err != nil {
        return fmt.Errorf("init deploy engine: %w", err)
    }
    prof, err := app.ProfileManager()
    if err != nil {
        return fmt.Errorf("init profile manager: %w", err)
    }
    src, err := app.SourceManager()
    if err != nil {
        return fmt.Errorf("init source manager: %w", err)
    }
    reg, err := app.AgentRegistry()
    if err != nil {
        return fmt.Errorf("init agent registry: %w", err)
    }

    // Build profile adapter with pre-bound deps (F12)
    scanIndex, _ := app.ScanIndex()
    var idx *asset.Index
    if scanIndex != nil {
        idx = scanIndex.Index
    }
    adapter := tui.NewProfileAdapter(prof, eng, idx, app.ProjectRoot)

    // Resolve project root if scope is project (F8/F20)
    // The TUI picker may change scope at runtime; pass a resolver function
    resolver := func() (string, error) { return app.ResolveProjectRoot() }

    return tui.Run(eng, adapter, src, reg, app.Scope == nd.ScopeProject, resolver)
}
```

```go
// internal/tui/run.go — accepts interfaces, no cmd dependency
func Run(
    deployer Deployer,
    profiles ProfileSwitcher,
    sources SourceScanner,
    agents AgentDetector,
    hasProjectDir bool,
    resolveProjectRoot func() (string, error),
) error {
    model := New(deployer, profiles, sources, agents, hasProjectDir, resolveProjectRoot)
    p := tea.NewProgram(model, tea.WithAltScreen())
    _, err := p.Run()
    return err
}
```

- Uses `tea.WithAltScreen()` for full terminal takeover
- Service errors propagate to CLI error handling (exit code 1)
- Project root resolver passed as callback for runtime scope changes (F8/F20)
- Test: verify Run creates program correctly (mock services)

**Task 7.2: Update `go.mod` — verify deps resolve**

Run `go mod tidy` and verify all Bubble Tea ecosystem deps resolve.

#### Phase 8: Integration Tests (depends on all phases)

**Task 8.1: `internal/tui/app_integration_test.go`**

Integration tests using Bubble Tea's test facilities (`tea.Send`, programmatic message injection).

```go
func TestAppLaunchFlow(t *testing.T)       // picker -> menu -> dashboard
func TestAppDeployFlow(t *testing.T)       // dashboard -> fuzzy -> deploy -> table refresh
func TestAppRemoveFlow(t *testing.T)       // dashboard -> select -> remove -> confirm -> table refresh
func TestAppProfileSwitch(t *testing.T)    // dashboard -> P -> list picker -> confirm -> refresh
func TestAppSnapshotSave(t *testing.T)     // dashboard -> W -> s -> prompt -> save
func TestAppSnapshotRestore(t *testing.T)  // dashboard -> W -> r -> list picker -> confirm -> restore
func TestAppSyncFlow(t *testing.T)         // dashboard -> s -> sync -> toast -> table refresh
func TestAppFixFlow(t *testing.T)          // dashboard -> f -> repair -> toast -> table refresh
func TestAppEmptyStates(t *testing.T)      // no sources, no assets, no profiles
func TestAppResponsiveLayout(t *testing.T) // various terminal sizes
func TestAppRemovePinnedAsset(t *testing.T)  // pinned asset -> stronger confirm warning (FR-024a)
func TestAppDeployHookReminder(t *testing.T) // deploy hook -> persistent toast with settings.json reminder (Q13)
func TestAppProfileSwitchConflict(t *testing.T) // profile switch with conflicts -> warning in confirm (FR-023)
func TestAppPluginTabRestrictions(t *testing.T) // d/r/s disabled on Plugins tab
func TestAppContextBackupWarning(t *testing.T)  // deploy context over existing file -> FR-016b confirm
func TestAppRunEntryPoint(t *testing.T)     // verify Run() creates program correctly (covers run.go)
```

- Use mock services (from interfaces in `services.go`)
- Verify state transitions, component updates, and view output
- Test: each user flow from the design doc

**Task 8.2: `internal/tui/components/component_test.go` + `internal/tui/testutil_test.go`**

Shared test helpers for component testing (F16/F17). Mock implementations of service interfaces live in `internal/tui/testutil_test.go` (same package as `app_integration_test.go`). Component-level helpers live in `internal/tui/components/component_test.go`.

```go
// internal/tui/components/component_test.go
func sendKeys(m tea.Model, keys ...tea.KeyMsg) tea.Model
func viewContains(t *testing.T, m tea.Model, substr string)
func viewNotContains(t *testing.T, m tea.Model, substr string)

// internal/tui/testutil_test.go
type mockDeployer struct { ... }      // implements Deployer
type mockProfileSwitcher struct { ... } // implements ProfileSwitcher
type mockSourceScanner struct { ... }  // implements SourceScanner
type mockAgentDetector struct { ... }  // implements AgentDetector
```

#### Phase 9: Audit and Polish (depends on Phase 8)

**Task 9.1: Full flow walkthrough** (manual)

Manual testing of all user flows:

- [ ] Launch: picker -> menu -> dashboard
- [ ] Tab navigation: all 9 tabs render correctly
- [ ] Deploy: `d` -> fuzzy search -> select -> deploy -> table updates
- [ ] Remove: select asset -> `r` -> confirm -> table updates
- [ ] Fix: `f` -> repair -> toast with results
- [ ] Sync: `s` -> sync -> toast with results
- [ ] Profile switch: `P` -> list -> select -> confirm -> refresh
- [ ] Snapshot save: `W` -> `s` -> name prompt -> save -> toast
- [ ] Snapshot restore: `W` -> `r` -> list -> select -> confirm -> refresh
- [ ] Empty states: no sources, no assets, no profiles
- [ ] Responsive: resize terminal through all breakpoints
- [ ] Esc/back: verify navigation from every state

**Task 9.2: Performance validation** (NFR-001)

- Measure time from `nd` to first rendered frame
- Target: < 500ms with 500+ assets
- Profile with `go test -bench` if needed

**Task 9.3: Code review and cleanup** (all files)

- Run `go vet ./...`
- Run `golangci-lint run`
- Run `go test ./... -race`
- Verify test coverage > 80% on `internal/tui/`

## Known Limitations and Deferred Features

- **`/` search filter:** defined in design key ownership matrix but deferred to a future iteration. Dashboard filtering is not in the v1 TUI scope.
- **`?` help overlay:** defined in design but deferred. The context-sensitive HelpBar provides keybinding information per state.
- **Bulk remove from dashboard:** TUI v1 supports single-asset remove only. Bulk remove is available via CLI (`nd remove --all`, `nd remove --type skills`). Multi-select (Space to toggle) deferred per design TQ-2.
- **Screen reader support:** deferred. Keyboard-only TUI is inherently accessible to keyboard users. Lip Gloss adaptive colors provide baseline light/dark terminal support.
- **Deployer interface:** plan uses `asset.Identity` (corrected per audit F11); design document still has `nd.AssetIdentity`. Design update deferred to design-doc-refresh pass.

## Acceptance Criteria

### Functional Requirements

- [ ] `nd` with no args launches TUI (FR-001)
- [ ] Picker prompts for scope and agent on launch (FR-029)
- [ ] Main menu shows dashboard, asset types, settings, quit (FR-028)
- [ ] Dashboard shows tabbed interface with overview + per-type tabs (FR-017)
- [ ] Header displays profile, scope, agent, issue count (FR-018)
- [ ] Inline actions: deploy, remove, sync, fix, profile switch, snapshots (FR-019)
- [ ] Profile switch flow with confirmation and diff summary (FR-023)
- [ ] Snapshot save and restore flows (FR-020, FR-021)
- [ ] Bulk operation results with retry-failed option (FR-010)
- [ ] Context file metadata display in Context tab (FR-016c)
- [ ] Duplicate asset warnings (FR-016a)
- [ ] Auto-snapshots before bulk operations (FR-029a)
- [ ] Empty states with guided onboarding

### Non-Functional Requirements

- [ ] Initial menu renders within 500ms with 500+ assets (NFR-001)
- [ ] Graceful degradation when sources unavailable (NFR-006)
- [ ] Responsive layout at 40-200 char terminal widths
- [ ] Unit test coverage > 80% on `internal/tui/`
- [ ] No race conditions (`go test -race`)

### Quality Gates

- [ ] `go vet ./...` passes
- [ ] `go test ./...` passes
- [ ] `golangci-lint run` passes
- [ ] `rumdl check` passes on any new `.md` files

## Dependencies and Prerequisites

- All 6 layers complete and passing (confirmed: PR #1-#5 merged)
- Go 1.25.1+ (confirmed in `go.mod`)
- Phase 0 prerequisite: `ListProfiles()` and `ListSnapshots()` added to `profile.Manager`
- New deps: Bubble Tea v2, Lip Gloss v2, Bubbles

## Risk Analysis and Mitigation

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|-----------|
| Bubble Tea v2 API instability | Low | High | Pin exact version, check release notes |
| Terminal rendering inconsistencies across OS | Medium | Medium | Use Lip Gloss adaptive colors, test on macOS + Linux |
| Complex nested model state management | Medium | Medium | Key ownership matrix prevents conflicts, thorough state tests |
| Performance with large asset sets (500+) | Low | Medium | Lazy index, batched health checks, async scanning |
| Fuzzy matching performance | Low | Low | Simple substring match first, upgrade to fuzzy if needed |
| Modal overlay z-index rendering | Low | Low | Bubble Tea handles alt screen; modals render last in View() |

## File Inventory

### New files (34)

| File | Phase | Purpose |
|------|-------|---------|
| `internal/tui/services.go` | 1 | Consumer-defined service interfaces |
| `internal/tui/services_test.go` | 1 | Compile-time interface assertions |
| `internal/tui/styles.go` | 1 | Lip Gloss style constants |
| `internal/tui/keys.go` | 1 | Keybinding definitions |
| `internal/tui/states.go` | 1 | AppState, ToastLevel shared types (F9 fix) |
| `internal/tui/messages.go` | 1 | Custom message types |
| `internal/tui/profileadapter.go` | 1 | ProfileSwitcher adapter for profile.Manager (F12 fix) |
| `internal/tui/profileadapter_test.go` | 1 | Adapter unit tests |
| `internal/tui/app.go` | 6 | Root App model |
| `internal/tui/app_test.go` | 6 | Root model unit tests |
| `internal/tui/app_integration_test.go` | 8 | Integration tests |
| `internal/tui/run.go` | 7 | `Run()` entry point for CLI wiring |
| `internal/tui/components/header.go` | 2 | Header bar component |
| `internal/tui/components/header_test.go` | 2 | Tests |
| `internal/tui/components/tabbar.go` | 2 | Tab bar component |
| `internal/tui/components/tabbar_test.go` | 2 | Tests |
| `internal/tui/components/helpbar.go` | 2 | Help bar component |
| `internal/tui/components/helpbar_test.go` | 2 | Tests |
| `internal/tui/components/toast.go` | 2 | Toast notification component |
| `internal/tui/components/toast_test.go` | 2 | Tests |
| `internal/tui/components/picker.go` | 2 | Scope/agent picker component |
| `internal/tui/components/picker_test.go` | 2 | Tests |
| `internal/tui/components/table.go` | 3 | Asset table component |
| `internal/tui/components/table_test.go` | 3 | Tests |
| `internal/tui/components/menu.go` | 4 | Main menu component |
| `internal/tui/components/menu_test.go` | 4 | Tests |
| `internal/tui/components/fuzzy.go` | 5 | Fuzzy finder component |
| `internal/tui/components/fuzzy_test.go` | 5 | Tests |
| `internal/tui/components/listpicker.go` | 5 | List picker component |
| `internal/tui/components/listpicker_test.go` | 5 | Tests |
| `internal/tui/components/prompt.go` | 5 | Text prompt component |
| `internal/tui/components/prompt_test.go` | 5 | Tests |
| `internal/tui/testutil_test.go` | 8 | Mock service implementations for integration tests |
| `internal/tui/components/component_test.go` | 8 | Shared test helpers |

### Modified files (4)

| File | Phase | Change |
|------|-------|--------|
| `internal/profile/manager.go` | 0 | Add `ListProfiles()` and `ListSnapshots()` delegation methods |
| `internal/profile/manager_test.go` | 0 | Tests for new methods |
| `cmd/root.go` | 7 | Change `RunE` to launch TUI when no args |
| `go.mod` | 1 | Add Bubble Tea, Lip Gloss, Bubbles deps |

## Sources and References

### Origin

- **Design document:** [docs/plans/2026-03-15-tui-layer-design.md](2026-03-15-tui-layer-design.md) — Key decisions: dashboard-centric UX, arrow key navigation, nested models, service interfaces, responsive layout, main menu, profile/snapshot flows.
- **Brainstorm document:** [docs/brainstorms/2026-03-15-tui-layer-brainstorm.md](../brainstorms/2026-03-15-tui-layer-brainstorm.md)

### Internal References

- Spec TUI requirements: `docs/specs/nd-go-spec.md` — FR-001, FR-017-019, FR-028-029
- CLI wiring pattern: `cmd/root.go`, `cmd/app.go` — lazy init, service creation
- Profile Manager API: `internal/profile/manager.go:45-327`
- Deploy Engine API: `internal/deploy/deploy.go`
- Source Manager API: `internal/sourcemanager/sourcemanager.go`
- Agent Registry API: `internal/agent/registry.go`
- Architecture audit: `.claude/docs/reports/2026-03-15-tui-layer-design-architecture-review.md`
- Spec gap audit: `.claude/docs/reports/2026-03-15-tui-design-vs-spec-audit.md`

### External References

- Bubble Tea v2 documentation: `charm.land/bubbletea/v2` (vanity domain; source: `github.com/charmbracelet/bubbletea`)
- Lip Gloss v2 documentation: `charm.land/lipgloss/v2` (vanity domain; source: `github.com/charmbracelet/lipgloss`)
- Bubbles v2 component library: `charm.land/bubbles/v2` (vanity domain; source: `github.com/charmbracelet/bubbles`)
- Bubble Tea v2 upgrade guide: `github.com/charmbracelet/bubbletea/blob/v2.0.2/UPGRADE_GUIDE_V2.md`
