# TUI Layer design

| Field | Value |
| --- | --- |
| **Date** | 2026-03-22 |
| **Author** | Larah |
| **Status** | Audit-remediated |
| **Packages** | `internal/tui/` (new) |
| **Spec refs** | FR-001, FR-002, FR-016c, FR-016d, FR-017, FR-018, FR-019, FR-024a, FR-025, FR-028, FR-029, NFR-001, NFR-004 |
| **Brainstorm** | `docs/brainstorms/2026-03-22-tui-layer-brainstorm.md` (audit-remediated) |
| **Supersedes** | `.claude/archive/complete-plans/2026-03-15-tui-layer-design.md` (deprecated) |

## Overview

The TUI is a menu-driven, wizard-style interactive interface launched by bare `nd` (no args). It provides near-complete parity with the CLI through sequential screens and guided flows. All operations delegate to the existing service layer through a `Services` interface that breaks the `cmd` <-> `internal/tui` circular import.

**Design principles:**
- **Visual minimalism.** No borders, no box-drawing characters. Whitespace and sparse color communicate structure.
- **Menu-driven, not dashboard-centric.** Each screen does one thing. The previous TUI failed because a persistent multi-panel dashboard was too complex.
- **huh does the heavy lifting.** Select, MultiSelect, Input, and Confirm from huh v2 are the primary interactive components. Custom Bubble Tea code is limited to the app shell (screen routing, header, help bar).
- **Input-aware key routing.** Global shortcuts (`q`, `esc`) are only active when no text field has focus.

**Technology:**
- Bubble Tea v2 (`charm.land/bubbletea/v2`) — app lifecycle, screen management
- huh v2 (`charm.land/huh/v2`) — forms, selects, confirms
- Lip Gloss v2 (`charm.land/lipgloss/v2`) — styling, adaptive color
- Bubbles v2 (`charm.land/bubbles/v2`) — progress bar, viewport

## Architecture

### Package layout

```text
internal/tui/
  services.go       Services interface (consumed by TUI, satisfied by cmd.App)
  tui.go            Root model: Init, Update, View, Run()
  screens.go        Screen interface, navigation messages (NavigateMsg, BackMsg, PopToRootMsg)
  header.go         Persistent header: profile · scope · agent · counts
  helpbar.go        Context-sensitive bottom help bar
  theme.go          Palette (5 colors), Styles (6 styles), glyphs, huh theme
  main_menu.go      Main menu (huh Select, grouped by level)
  deploy.go         Deploy flow: search/type -> multi-select -> progress -> result
  remove.go         Remove flow: type -> multi-select -> confirm -> progress -> result
  browse.go         Browse all assets (filterable list, nd list equivalent)
  status.go         Deployment status grouped by type
  doctor.go         Doctor: show issues -> confirm fix -> result
  profile.go        Profile submenu: switch, create, list, deploy
  snapshot.go       Snapshot submenu: save, restore, list
  source.go         Source submenu: list, add local/git, remove, sync
  pin.go            Pin/unpin: select deployed assets -> toggle
  init.go           First-run guided setup (auto-detected)
  export.go         Export plugin flow (migrated from cmd/export.go)
  settings.go       Settings submenu: edit config, show path/version
  progress.go       Reusable progress bar for bulk operations
  empty.go          Empty state messages with actionable hints
```

21 files. No subpackages. Each screen file contains its own model, messages, and view — self-contained.

### Services interface

Breaks the circular import between `cmd/` and `internal/tui/`. Defined in the TUI package; satisfied by `cmd.App`.

```go
// internal/tui/services.go

package tui

import (
    "github.com/armstrongl/nd/internal/agent"
    "github.com/armstrongl/nd/internal/deploy"
    "github.com/armstrongl/nd/internal/nd"
    "github.com/armstrongl/nd/internal/oplog"
    "github.com/armstrongl/nd/internal/profile"
    "github.com/armstrongl/nd/internal/sourcemanager"
    "github.com/armstrongl/nd/internal/state"
)

// Services provides access to nd's service layer.
// cmd.App satisfies this interface with small additions (GetScope, IsDryRun, ResetForScope).
type Services interface {
    // Source management
    SourceManager() (*sourcemanager.SourceManager, error)
    ScanIndex() (*sourcemanager.ScanSummary, error)

    // Agent management
    AgentRegistry() (*agent.Registry, error)
    DefaultAgent() (*agent.Agent, error)

    // Deployment
    DeployEngine() (*deploy.Engine, error)
    StateStore() *state.Store

    // Profiles & snapshots
    ProfileManager() (*profile.Manager, error)
    ProfileStore() (*profile.Store, error)

    // Operation logging
    OpLog() *oplog.Writer

    // Display state — named to avoid collision with App.Scope/DryRun fields
    GetScope() nd.Scope
    ConfigPath() string
    IsDryRun() bool

    // Mid-session reset (scope/agent switching)
    // Nils all cached services so they reinitialize for the new scope.
    ResetForScope(scope nd.Scope, projectRoot string)
}
```

**Note:** `cmd.App` needs three new methods to satisfy this interface:
- `GetScope() nd.Scope` returns `a.Scope` (named `GetScope` to avoid collision with the `Scope` field)
- `IsDryRun() bool` returns `a.DryRun` (named `IsDryRun` to avoid collision with the `DryRun` field)
- `ResetForScope(scope nd.Scope, projectRoot string)` nils all cached services (`sm`, `reg`, `eng`, `profMgr`, `pstore`, `sstore`, `ol`) then sets `a.Scope` and `a.ProjectRoot` (see brainstorm A3)
- `OpLog()` already exists on `cmd.App`

### Screen interface

```go
// internal/tui/screens.go

package tui

import tea "charm.land/bubbletea/v2"

// Screen is a TUI view that the root model manages on a stack.
type Screen interface {
    tea.Model
    Title() string       // Displayed in breadcrumb context
    InputActive() bool   // True when a text field has focus (suppresses global keys)
}

// Navigation messages — screens emit these, root model handles them.
type NavigateMsg struct{ Screen Screen }  // Push a new screen
type BackMsg struct{}                      // Pop one level
type PopToRootMsg struct{}                 // Clear stack, return to main menu
type RefreshHeaderMsg struct{}             // Re-query state for header counts
```

### Root model

```go
// internal/tui/tui.go

package tui

import (
    tea "charm.land/bubbletea/v2"
    "charm.land/lipgloss/v2"
)

type Model struct {
    svc     Services
    styles  Styles
    screens []Screen   // Navigation stack. screens[0] = main menu.
    header  Header
    helpbar HelpBar
    width   int
    height  int
    isDark  bool
}

func Run(svc Services) error {
    isDark := lipgloss.HasDarkBackground(os.Stdin, os.Stderr)
    styles := NewStyles(isDark)

    // Determine initial screen before creating the program.
    // Init() returns tea.Cmd only (no model), so screen state must be set here.
    var initialScreen Screen
    if _, err := os.Stat(svc.ConfigPath()); os.IsNotExist(err) {
        initialScreen = newInitScreen(svc, styles, isDark)
    } else {
        initialScreen = newMainMenuScreen(styles, isDark)
    }

    m := Model{
        svc:     svc,
        styles:  styles,
        isDark:  isDark,
        screens: []Screen{initialScreen},
    }

    p := tea.NewProgram(m, tea.WithAltScreen())
    _, err := p.Run()
    return err
}
```

**Init:**

```go
func (m Model) Init() tea.Cmd {
    // Bootstrap header + async agent detection.
    // Initial screen is already set in Run() — Init only returns commands.
    return tea.Batch(
        func() tea.Msg { return RefreshHeaderMsg{} },
        func() tea.Msg {
            if reg, err := m.svc.AgentRegistry(); err == nil {
                reg.Detect()
            }
            return agentDetectedMsg{}
        },
    )
}
```

First-run detection (no config file) happens in `Run()` before the program starts. `Init()` only fires async commands: header refresh and agent detection. Agent detection runs in a background `tea.Cmd` — if >100ms, a spinner shows.

**Update — key routing:**

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height

    case NavigateMsg:
        m.screens = append(m.screens, msg.Screen)
        return m, msg.Screen.Init()

    case BackMsg:
        if len(m.screens) <= 1 {
            return m, tea.Quit  // Back from main menu = quit
        }
        m.screens = m.screens[:len(m.screens)-1]
        return m, nil

    case PopToRootMsg:
        m.screens = m.screens[:1]  // Keep only main menu
        return m, nil

    case RefreshHeaderMsg:
        m.header = m.header.Refresh(m.svc)
        return m, nil

    case tea.KeyPressMsg:
        current := m.screens[len(m.screens)-1]

        // When text input is active, only ctrl+c force-quits
        if current.InputActive() {
            if msg.String() == "ctrl+c" {
                return m, tea.Quit
            }
            // Fall through to delegate to screen
        } else {
            // Global keys only when no text input
            switch msg.String() {
            case "q", "ctrl+c":
                return m, tea.Quit
            case "esc":
                if len(m.screens) <= 1 {
                    return m, tea.Quit
                }
                m.screens = m.screens[:len(m.screens)-1]
                return m, nil
            }
        }
    }

    // Delegate to current screen
    if len(m.screens) > 0 {
        idx := len(m.screens) - 1
        updated, cmd := m.screens[idx].Update(msg)
        m.screens[idx] = updated.(Screen)
        return m, cmd
    }
    return m, nil
}
```

**View — vertical composition:**

```go
func (m Model) View() string {
    if len(m.screens) == 0 {
        return ""
    }

    header := m.header.View(m.styles, m.width)
    content := m.screens[len(m.screens)-1].View()
    helpbar := m.helpbar.View(m.styles, m.screens[len(m.screens)-1], m.width)

    return lipgloss.JoinVertical(lipgloss.Left, header, "", content, "", helpbar)
}
```

Three sections separated by blank lines: header, content, help bar. When `svc.IsDryRun()` is true, the header prepends `[DRY RUN]` to the left side.

**Screens that emit `RefreshHeaderMsg` after state changes:**
- `deploy.go` — after `deployDoneMsg` (deployment count changed)
- `remove.go` — after `removeDoneMsg` (deployment count changed)
- `profile.go` — after profile switch (active profile, counts changed)
- `snapshot.go` — after snapshot restore (all state changed)
- `source.go` — after source sync (asset counts may change)
- `doctor.go` — after fix applied (issue counts changed)
- `settings.go` — after scope/agent switch via `ResetForScope`

### Header

```go
// internal/tui/header.go

type Header struct {
    Profile    string  // "go-dev" or "no profile"
    Scope      string  // "global" or "project"
    Agent      string  // "claude"
    Deployed   int
    Issues     int
}

func (h Header) View(s Styles, width int) string {
    left := fmt.Sprintf("  %s · %s · %s", h.Profile, h.Scope, h.Agent)
    right := fmt.Sprintf("%d deployed  %d issues", h.Deployed, h.Issues)

    // Style: left default, right subtle (issues danger if > 0)
    leftStyled := left
    rightStyled := s.Subtle.Render(right)
    if h.Issues > 0 {
        rightStyled = fmt.Sprintf("%s  %s",
            s.Subtle.Render(fmt.Sprintf("%d deployed", h.Deployed)),
            s.Danger.Render(fmt.Sprintf("%d issues", h.Issues)))
    }

    gap := width - lipgloss.Width(left) - lipgloss.Width(right)
    if gap < 1 { gap = 1 }
    return leftStyled + strings.Repeat(" ", gap) + rightStyled
}

func (h Header) Refresh(svc Services) Header {
    // Re-query profile, agent, deployment counts
    if pm, err := svc.ProfileManager(); err == nil {
        h.Profile, _ = pm.ActiveProfile()
    }
    if h.Profile == "" { h.Profile = "no profile" }
    h.Scope = string(svc.GetScope())

    if ag, err := svc.DefaultAgent(); err == nil {
        h.Agent = ag.Name
    }

    if eng, err := svc.DeployEngine(); err == nil {
        if entries, err := eng.Status(); err == nil {
            h.Deployed = len(entries)
            h.Issues = 0
            for _, e := range entries {
                if e.Health != state.HealthOK { h.Issues++ }
            }
        }
    }
    return h
}
```

### Help bar

```go
// internal/tui/helpbar.go

type HelpBar struct{}

type HelpItem struct {
    Key  string
    Desc string
}

func (hb HelpBar) View(s Styles, screen Screen, width int) string {
    items := defaultHelp(screen)
    parts := make([]string, len(items))
    for i, item := range items {
        parts[i] = s.Subtle.Render(item.Key+" "+item.Desc)
    }
    return "  " + strings.Join(parts, "  ")
}

func defaultHelp(screen Screen) []HelpItem {
    // Base items present on all screens
    items := []HelpItem{
        {"esc", "back"},
        {"j/k", "navigate"},
        {"enter", "select"},
    }
    // Screens can implement HelpProvider to add custom items
    if hp, ok := screen.(HelpProvider); ok {
        items = append(items, hp.HelpItems()...)
    }
    items = append(items, HelpItem{"?", "help"}, HelpItem{"q", "quit"})
    return items
}

type HelpProvider interface {
    HelpItems() []HelpItem
}
```

### Theme

```go
// internal/tui/theme.go

package tui

import (
    "charm.land/lipgloss/v2"
    "charm.land/huh/v2"
)

// Palette — 5 semantic accent colors + adaptive dark/light
var (
    cSubtle  = lipgloss.AdaptiveColor{Dark: "#6c7086", Light: "#9ca0b0"}
    cPrimary = lipgloss.AdaptiveColor{Dark: "#89b4fa", Light: "#1e66f5"}
    cSuccess = lipgloss.AdaptiveColor{Dark: "#a6e3a1", Light: "#40a02b"}
    cWarning = lipgloss.AdaptiveColor{Dark: "#f9e2af", Light: "#df8e1d"}
    cDanger  = lipgloss.AdaptiveColor{Dark: "#f38ba8", Light: "#d20f39"}
)

// Styles — the complete set of reusable styles
type Styles struct {
    Subtle  lipgloss.Style
    Primary lipgloss.Style
    Success lipgloss.Style
    Warning lipgloss.Style
    Danger  lipgloss.Style
    Bold    lipgloss.Style
}

func NewStyles(isDark bool) Styles {
    return Styles{
        Subtle:  lipgloss.NewStyle().Foreground(cSubtle),
        Primary: lipgloss.NewStyle().Foreground(cPrimary),
        Success: lipgloss.NewStyle().Foreground(cSuccess),
        Warning: lipgloss.NewStyle().Foreground(cWarning),
        Danger:  lipgloss.NewStyle().Foreground(cDanger),
        Bold:    lipgloss.NewStyle().Bold(true),
    }
}

// Glyphs — text-based, readable without color
// One glyph per HealthStatus (OK, Broken, Drifted, Orphaned, Missing)
const (
    GlyphOK      = "ok"
    GlyphBroken  = "!!"
    GlyphDrifted = "??"
    GlyphOrphan  = "--"
    GlyphMissing = "xx"
    GlyphDot     = "·"
    GlyphArrow   = "->"
)

// NdTheme implements huh.Theme for consistent form styling
type NdTheme struct{}

func (NdTheme) Theme(isDark bool) *huh.Styles {
    return huh.ThemeCatppuccin().Theme(isDark)
}
```

## Screen Specifications

### Main Menu

**File:** `main_menu.go`
**Model state:** None (stateless — just a huh Select)
**Entry:** On TUI launch (or PopToRootMsg)

```go
type mainMenuScreen struct {
    form *huh.Form
}

func newMainMenuScreen(styles Styles, isDark bool) Screen {
    choices := []huh.Option[string]{
        huh.NewOption("Deploy assets", "deploy"),
        huh.NewOption("Remove assets", "remove"),
        huh.NewOption("Browse assets", "browse"),
        huh.NewOption("View status", "status"),
        huh.NewOption("Run doctor", "doctor"),
        // Blank separator handled via huh group or styled option
        huh.NewOption("Switch profile", "profile"),
        huh.NewOption("Manage snapshots", "snapshot"),
        huh.NewOption("Pin/Unpin assets", "pin"),
        huh.NewOption("Manage sources", "source"),
        // Blank separator
        huh.NewOption("Export plugin", "export"),
        huh.NewOption("Settings", "settings"),
        huh.NewOption("Quit", "quit"),
    }
    // ... build form with huh.NewSelect
}
```

**On selection:** Emits `NavigateMsg` with the appropriate screen. "Quit" emits `tea.Quit`.

### Deploy Flow

**File:** `deploy.go`
**Model state:** `step` enum (pickType, selectAssets, running, result), selected type, selected assets, result
**Entry:** From main menu "Deploy assets"

**Steps:**
1. `pickType` — huh Select: "Search all", Skills, Commands, Rules, Context, Agents, Output styles, Hooks, All types
2. `selectAssets` — huh MultiSelect filtered to undeployed assets of chosen type. Includes `/` filter. Shows description + source (styled Subtle).
3. `running` — Progress bar. Operation runs in `tea.Cmd`. Sends `progressMsg{completed, total, name}`.
4. `result` — Shows deployed count, per-item paths. Enter emits `PopToRootMsg`. On partial failure: per-item errors with asset/path/reason (NFR-004), "Retry failed" option.

**Goroutine safety (A5):** The deploy engine is fetched in Update (single-threaded), then passed to the `tea.Cmd` closure.

**Progress callback (A4):** `DeployBulk` doesn't support per-item callbacks today. Two options:
- Option A: Call `Deploy` in a loop within the `tea.Cmd`, sending progress messages between each.
- Option B: Add `DeployBulkWithProgress(reqs, func)` to the engine.

Option A is simpler and avoids lock contention. The `tea.Cmd` goroutine calls `eng.Deploy(req)` per item (where `req` is `deploy.DeployRequest`), sends a `progressMsg` via `p.Send()` after each, then sends `deployDoneMsg{succeeded []deploy.DeployResult, failed []deploy.DeployError}` with the full result.

```go
func deployCmd(eng *deploy.Engine, reqs []deploy.DeployRequest) tea.Cmd {
    return func() tea.Msg {
        var results []deploy.DeployResult
        var errors []deploy.DeployError
        for i, req := range reqs {
            result, err := eng.Deploy(req)
            if err != nil {
                errors = append(errors, deploy.DeployError{
                    AssetName: req.Asset.Name, AssetType: req.Asset.Type,
                    SourcePath: req.Asset.SourcePath, Err: err,
                })
            } else {
                results = append(results, *result)
            }
            // send progress update via p.Send(progressMsg{completed: i+1, total: len(reqs)})
        }
        return deployDoneMsg{succeeded: results, failed: errors}
    }
}
```

**InputActive:** Returns `true` when the "Search all" filter input has focus.

### Remove Flow

**File:** `remove.go`
**Model state:** `step` (pickType, selectAssets, confirm, running, result)
**Entry:** From main menu "Remove assets"

Same structure as deploy but:
- Asset list shows only deployed assets
- Confirmation step uses huh Confirm: "Remove N type(s)? An auto-snapshot will be saved first."
- Uses per-item `eng.Remove(req deploy.RemoveRequest)` in a `tea.Cmd` loop (same pattern as deploy) for progress callbacks. `RemoveBulk` doesn't support progress.
- Result: `removeDoneMsg{succeeded []state.Deployment, failed []deploy.RemoveError}`

### Browse Screen

**File:** `browse.go`
**Model state:** Filtered asset list, current filter text
**Entry:** From main menu "Browse assets"

Displays all assets from all sources with deployment status. Equivalent to `nd list`. Uses a viewport for scrolling with `/` filter support. Columns: status marker (`*` deployed, ` ` available), type, name, source, description.

### Status Screen

**File:** `status.go`
**Model state:** Cached status entries
**Entry:** From main menu "View status"

Groups deployed assets by type. Shows health glyph (ok/!!/??/--/xx mapping to OK/Broken/Drifted/Orphaned/Missing), name, scope, origin, source. Issues count at bottom. Uses `deploy.StatusEntry` which contains `state.Deployment` + `state.HealthStatus` + detail string.

**Custom help items:** `f` fix, `d` deploy, `r` remove, `/` filter. These shortcuts are active because `InputActive()` returns false on this screen.

### Doctor Screen

**File:** `doctor.go`
**Model state:** `step` (scanning, results, confirm, fixing, done)
**Entry:** From main menu "Run doctor" or `f` from status screen

Shows issues with description. huh Confirm to fix all. Applies fixes (remove broken records, remove orphaned symlinks). Shows result.

### Profile Submenu

**File:** `profile.go`
**Model state:** Submenu selection, inner flow state
**Entry:** From main menu "Switch profile"

Submenu: Switch, Create, List, Deploy, Back. "Switch" shows huh Select of profiles with asset counts. After switch, shows diff summary.

### Snapshot Submenu

**File:** `snapshot.go`
**Model state:** `step` (menu, saving, restoring, list, done), snapshot name, selected snapshot
**Entry:** From main menu "Manage snapshots"

Submenu (huh Select): Save, Restore, List, Back.

- **Save:** huh Input for snapshot name (validation: non-empty, no slashes). On submit, calls `ProfileStore().Save(name)`. Shows result message. Emits `RefreshHeaderMsg`.
- **Restore:** huh Select listing available snapshots (name + timestamp). On submit, huh Confirm "Restore snapshot X? This will overwrite current state." Calls `ProfileStore().Restore(name)`. Shows result. Emits `RefreshHeaderMsg`.
- **List:** Read-only viewport listing snapshots with name and creation timestamp.

**InputActive:** Returns `true` during Save name input.

### Source Submenu

**File:** `source.go`
**Entry:** From main menu "Manage sources"

Submenu: List, Add local (huh Input), Add Git (huh Input), Remove (huh Select + Confirm), Sync all (progress bar), Back.

**InputActive:** Returns `true` during Add local/Git flows.

### Pin/Unpin

**File:** `pin.go`
**Model state:** `step` (select, confirm, done), selected assets, results
**Entry:** From main menu "Pin/Unpin assets"

Shows deployed assets with current pin status (prefix `[pinned]` or `[unpinned]`). huh MultiSelect to toggle — pre-selected items are currently pinned. On submit, diff is computed (newly selected = pin, deselected = unpin). huh Confirm with summary of changes. Applies via `StateStore().SetPinned(id, bool)`. Shows result.

**Empty state:** "No deployed assets to pin." with hint to deploy first.

### Init (First-Run)

**File:** `init.go`
**Entry:** Auto-detected when no config file exists

Guided setup: Add source type (local/git/skip) -> path/URL input -> scope detection -> done. Creates config directory and minimal config.yaml.

### Export

**File:** `export.go`
**Model state:** `step` (selectAssets, metadata, running, result), config, results
**Entry:** From main menu "Export plugin"

Migrated from `cmd/export.go` `runExportInteractive()`. Same huh forms (MultiSelect for assets, Input for name/author/version/description, Confirm), same `export.PluginExporter` logic, rendered inside the TUI shell instead of standalone huh run.

**Steps:**
1. huh MultiSelect for assets to include (grouped by type)
2. huh form: name, author, version, description, output directory
3. huh Confirm with summary
4. Progress — runs `exporter.Export(config)` in `tea.Cmd`
5. Result — shows exported path, file count

**InputActive:** Returns `true` during metadata form inputs.

### Settings

**File:** `settings.go`
**Model state:** Submenu selection, inner flow state
**Entry:** From main menu "Settings"

Submenu (huh Select): Edit config, Show config path, Show version, Switch scope, Back.

- **Edit config:** Suspends the TUI with `tea.ExecProcess(os.Getenv("EDITOR"), configPath)`. On return, re-validates config. If invalid, shows warning.
- **Show config path:** Inline display of `svc.ConfigPath()`.
- **Show version:** Inline display of `nd.Version`.
- **Switch scope:** huh Select: global, project. If project, auto-detects or prompts for project root. Calls `svc.ResetForScope(scope, root)`. Emits `RefreshHeaderMsg`. Agent re-detection happens automatically when `DefaultAgent()` is called after reset (registry was niled).

**InputActive:** Returns `false` (no text inputs on this screen).

### Empty States

**File:** `empty.go`
**Entry:** Called by screens when they have no data to display

Returns styled messages with actionable hints:
- No sources: "No asset sources configured. Press enter to add one."
- No assets: "No assets found. Add a source first."
- Nothing deployed: "Nothing deployed yet."
- No profiles: "No profiles yet."
- All deployed: "All type assets are already deployed."

## Integration with cmd/

### root.go change

```go
// cmd/root.go — RunE modification
RunE: func(cmd *cobra.Command, args []string) error {
    if !isTerminal() {
        return cmd.Help()
    }
    // Flags incompatible with TUI — fall back to help text
    if app.Verbose || app.Quiet || app.JSON {
        return cmd.Help()
    }
    return tui.Run(app)
},
```

### App additions

```go
// cmd/app.go — new methods to satisfy tui.Services

func (a *App) GetScope() nd.Scope { return a.Scope }
func (a *App) IsDryRun() bool     { return a.DryRun }

func (a *App) ResetForScope(scope nd.Scope, projectRoot string) {
    a.Scope = scope
    a.ProjectRoot = projectRoot
    // Nil all cached services so they reinitialize for the new scope
    a.sm = nil
    a.reg = nil
    a.eng = nil
    a.profMgr = nil
    a.pstore = nil
    a.sstore = nil
    a.ol = nil
}
```

**Note:** The interface uses `GetScope()` and `IsDryRun()` to avoid collision with `App.Scope` and `App.DryRun` public fields. Go forbids a method and field with the same name on a struct.

### PersistentPreRunE interaction

For bare `nd` (no args, no explicit `--scope`), `PersistentPreRunE` skips scope validation. The TUI handles scope selection interactively. When `--scope` is explicitly passed, strict validation applies and the TUI respects it.

### huh v1 -> v2 migration

`cmd/export.go` currently imports `github.com/charmbracelet/huh`. This must be migrated to `charm.land/huh/v2` before or alongside the TUI implementation.

**Changes:**
- Import path: `github.com/charmbracelet/huh` -> `charm.land/huh/v2`
- API changes: Check huh v2 upgrade guide for breaking changes (theme interface, spinner subpackage, method renames)
- `go.mod`: Replace `github.com/charmbracelet/huh v1.0.0` with `charm.land/huh/v2`
- Transitive deps: BT v1 and BT v2 can coexist as separate modules, but once all huh usage is on v2, the v1 deps can be removed

## Testing Strategy

### Unit tests per screen

Each screen file gets a companion `_test.go`. Tests exercise the model via `Update()` with synthetic messages:

```go
func TestDeployScreen_TypeSelection(t *testing.T) {
    s := newDeployScreen(mockServices(), testStyles())
    // Simulate selecting "Skills"
    s, cmd := s.Update(selectMsg("skills"))
    assert.Equal(t, "selectAssets", s.(*deployScreen).step)
}
```

### Integration tests for root model

Test multi-screen navigation flows via the root model:

```go
func TestNavigationStack(t *testing.T) {
    m := newTestModel(mockServices())
    // Main menu -> Deploy -> Back -> Main menu
    m, _ = m.Update(NavigateMsg{newDeployScreen(...)})
    assert.Equal(t, 2, len(m.screens))
    m, _ = m.Update(BackMsg{})
    assert.Equal(t, 1, len(m.screens))
}
```

### Key routing tests

Verify that global keys are suppressed during input:

```go
func TestGlobalKeys_SuppressedDuringInput(t *testing.T) {
    m := newTestModel(mockServices())
    // Push a screen with InputActive() == true
    m, _ = m.Update(NavigateMsg{inputScreen{}})
    // Press 'q' — should NOT quit
    m, cmd = m.Update(tea.KeyPressMsg{...})
    assert.NotEqual(t, tea.Quit(), cmd)
}
```

### Mock services

A `mockServices` struct satisfies the `Services` interface with canned data for tests. No filesystem, no real agents.

## Non-Functional Requirements

| NFR | Approach |
| --- | --- |
| NFR-001 (500ms render) | Menu-driven = no data scan on launch. Agent detection is async. |
| NFR-004 (error detail) | Partial failure results include asset, path, and reason per item. |
| NFR-011 (file locking) | Lock failures show warning banner. Read-only mode when locked. |
| NO_COLOR | Glyphs are text-based. Styles degrade to unstyled. |
| Min terminal | 60x15. Below minimum: centered "Terminal too small" message. |
| --dry-run | Header shows "[DRY RUN]". Results say "Would deploy..." |
