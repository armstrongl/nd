# TUI Layer design

> **DEPRECATED (2026-03-18):** This design and its implementation (PR #6) did not meet expectations. The TUI will be reworked from scratch with a new design. The spec requirements (FR-001, FR-017, FR-028, FR-029, etc.) still define the desired behavior — this design's approach to fulfilling them is what's being replaced. See also: `docs/plans/2026-03-15-002-feat-tui-layer-bubbletea-plan.md`.

| Field | Value |
| --- | --- |
| **Date** | 2026-03-15 |
| **Author** | Larah |
| **Status** | Deprecated |
| **Version** | 0.2 (audit-remediated) |
| **Deprecated** | 2026-03-18 |
| **Packages** | `internal/tui/` (new), `internal/tui/components/` (new) |
| **Spec refs** | FR-001, FR-010, FR-016a, FR-016b, FR-016c, FR-017, FR-018, FR-019, FR-020, FR-021, FR-023, FR-024a, FR-028, FR-029, FR-029a, NFR-001, NFR-006 |
| **Framework** | Bubble Tea v2, Lip Gloss |
| **Brainstorm** | `docs/brainstorms/2026-03-15-tui-layer-brainstorm.md` |
| **Audit reports** | `.claude/docs/reports/2026-03-15-tui-design-vs-spec-audit.md`, `.claude/docs/reports/2026-03-15-tui-layer-design-architecture-review.md` |

## Overview

The TUI layer is a dashboard-centric interactive interface for nd. It launches when `nd` is invoked with no arguments (FR-001) and provides a main menu, a persistent tabbed dashboard for managing assets, and inline flows for profiles and snapshots. The TUI is a pure presentation layer — all operations delegate to the existing service layer via consumer-defined interfaces.

The design prioritizes:

- **Dashboard-centric UX**: A persistent view the user lives in. Tabs switch content in-place. Actions happen via keyboard without leaving the view. Inspired by lazygit and k9s.
- **Clean density**: Show essentials with room to breathe. One-line-per-asset with name, source, status. Details expand inline on selection.
- **Progressive disclosure**: Level 0 concepts (sources, assets, deploy) are prominent. Profiles, snapshots, and advanced features are accessible but not in the way.
- **Keyboard-only**: Arrow keys + Enter. No mouse interaction. Context-sensitive help bar guides available actions.

## Architecture

### Package layout

```text
internal/tui/
  app.go            Root App model — composes all components, routes messages
  app_test.go       Root model tests (message routing, state transitions)
  services.go       Consumer-defined interfaces for service layer (F-01)
  keys.go           Keybinding definitions, key ownership matrix
  styles.go         Lip Gloss style constants (colors, borders, spacing)
  messages.go       Custom message types (deploy result, sync result, etc.)

internal/tui/components/
  header.go         Header bar: scope, agent, profile, issue badge
  header_test.go
  tabbar.go         Tab bar: overview + per-asset-type tabs
  tabbar_test.go
  table.go          Asset table: sortable rows, inline expand, status icons
  table_test.go
  helpbar.go        Context-sensitive bottom bar: available actions
  helpbar_test.go
  menu.go           Main menu: dashboard, asset types, settings, quit (FR-028)
  menu_test.go
  picker.go         Scope/agent inline picker (launch flow)
  picker_test.go
  fuzzy.go          Fuzzy finder modal (deploy action)
  fuzzy_test.go
  listpicker.go     Generic list picker (profile switch, snapshot restore)
  listpicker_test.go
  prompt.go         Text input prompt (snapshot name)
  prompt_test.go
  toast.go          Temporary status messages and warnings
  toast_test.go
```

### Service interfaces (F-01 remediation)

The TUI depends on interfaces, not concrete service types. This follows the pattern already established in `internal/profile/manager.go` (lines 13-23).

```go
// services.go — consumer-defined interfaces

type Deployer interface {
    Deploy(deploy.DeployRequest) (*deploy.DeployResult, error)
    DeployBulk([]deploy.DeployRequest) (*deploy.BulkDeployResult, error)
    Remove(deploy.RemoveRequest) error
    RemoveBulk([]deploy.RemoveRequest) (*deploy.BulkRemoveResult, error)
    Status() ([]deploy.StatusEntry, error)
    Check() ([]state.HealthCheck, error)
    Sync() (*deploy.SyncResult, error)
    SetOrigin(nd.AssetIdentity, nd.Scope, string, nd.DeployOrigin) error
}

type ProfileManager interface {
    ActiveProfile() (string, error)
    Switch(current, target string, engine deploy.Engine, index *asset.Index, projectRoot string) (*profile.SwitchResult, error)
    Restore(name string, engine deploy.Engine, index *asset.Index) (*profile.RestoreResult, error)
    ListProfiles() ([]profile.ProfileSummary, error)   // NEW: delegation method needed
    ListSnapshots() ([]profile.SnapshotSummary, error)  // NEW: delegation method needed
}

type SourceScanner interface {
    Sources() []source.Source
    Scan() (*sourcemanager.ScanSummary, error)
    SyncSource(sourceID string) error
}

type AgentDetector interface {
    Detect() agent.DetectionResult
    Default() (*agent.Agent, error)
    All() []agent.Agent
}
```

**Pre-TUI prerequisite**: Add `ListProfiles()` and `ListSnapshots()` delegation methods to `profile.Manager` (wrapping the private `store` field).

### Dependencies

```text
tui.App (root)
  |-- components.Menu          (main menu)
  |-- components.Header
  |-- components.TabBar
  |-- components.Table
  |-- components.HelpBar
  |-- components.Picker        (launch: scope/agent)
  |-- components.FuzzyFinder   (modal: deploy)
  |-- components.ListPicker    (modal: profile switch, snapshot restore)
  |-- components.Prompt        (modal: snapshot name)
  |-- components.Toast         (temporary messages)
  |-- ConfirmState             (app-level, not a nested model) (F-05)
  |
  |-- tui.Deployer             (interface -> deploy.Engine)
  |-- tui.ProfileManager       (interface -> profile.Manager)
  |-- tui.SourceScanner        (interface -> sourcemanager.Manager)
  |-- tui.AgentDetector        (interface -> agent.Registry)
```

The root App model owns the service interface references and passes results down to components via messages. Components never call services directly.

### Nested model pattern

Each component implements the Bubble Tea `Model` interface (`Init`, `Update`, `View`). The root `App` model:

1. Receives all `tea.Msg` from Bubble Tea
2. Routes messages to the active component based on app state (see key ownership matrix)
3. Handles cross-component coordination (e.g., deploy result updates the table)
4. Composes all component views in a vertical layout

```text
+--------------------------------------------------+
|  Header  (scope | agent | profile | issues)      |  <- components.Header
+--------------------------------------------------+
|  TabBar  (Overview | Skills | Agents | ...)       |  <- components.TabBar
+--------------------------------------------------+
|                                                    |
|  Table   (asset rows, inline expand)               |  <- components.Table
|                                                    |
|                                                    |
+--------------------------------------------------+
|  HelpBar (context-sensitive keybindings)           |  <- components.HelpBar
+--------------------------------------------------+
```

Modal overlays (fuzzy finder, list picker, prompt, picker) render on top of this layout when active.

### App states

The root App tracks the current state as an enum. This determines message routing and key ownership.

```text
statePicker       Launch-time scope/agent selection
stateMenu         Main menu
stateDashboard    Normal dashboard operation
stateDetail       Asset detail expanded
stateFuzzy        Fuzzy finder overlay (deploy)
stateListPicker   List picker overlay (profile switch, snapshot restore)
statePrompt       Text input overlay (snapshot name)
stateConfirm      Inline confirmation in help bar
stateLoading      Initial scan in progress (spinner)
```

## Screen flow

### Launch sequence

```text
nd (no args)
  |
  v
[Picker: scope + agent selection]   <- FR-029
  |  Arrow keys to toggle, Enter to confirm
  v
[Main Menu]                         <- FR-028
  |  Arrow keys to select, Enter to navigate
  v
[Dashboard: Overview tab]           <- FR-017, FR-018
  |  Scope/agent shown in header bar
  v
[Normal operation]
```

If only one agent is detected (v1: Claude Code only), the agent picker auto-selects and shows only scope selection.

### Navigation state diagram

```text
                    +------------+
                    |   Picker   |  (launch only)
                    +-----+------+
                          |
                          v
                    +-----+------+
              +---->|    Menu    |<---------+
              |     +-----+------+         |
              |           |                |
              |  Esc      v     Esc        |
              |     +-----+------+    +----+-------+
              +-----|  Dashboard |    | ListPicker  |
              |     +--+-+--+---+    +----+--------+
              |        | |  |             ^
              |        | |  +--P/W------->+
              |        | |  |
              |        | |  +--d-->+------------+
              |        | |        | FuzzyFinder |
              |        | |        +-----+-------+
              |        | |  Esc         |
              |        | |  <-----------+
              |        | |
              |        | +--Enter--+-----------+
              |        |           |  Detail   |
              |        |  Esc      +-----+-----+
              |        |  <--------------+
              |        |
              |        +--r/bulk-->+-----------+
              |                   |  Confirm   |
              |        y/n        +-----+------+
              +<------------------------|
```

- **Picker** appears only on launch.
- **Menu** is the hub. Esc from dashboard returns to menu. `q` from menu quits.
- **Dashboard** is the primary operating view.
- **FuzzyFinder**, **ListPicker**, **Prompt** are modal overlays from dashboard.
- **Confirm** is an inline state (in help bar), not a modal.
- **Detail** is an in-place table expansion state.

## Component specifications

### Menu (FR-028)

Full-screen main menu. Centered vertically with a status summary at the bottom.

```text
+------------------------------------------+
|                                           |
|         Napoleon Dynamite                 |
|         Asset Manager for Claude Code     |
|                                           |
|         > Dashboard                       |
|           Skills                          |
|           Agents                          |
|           Commands                        |
|           Rules                           |
|           Output Styles                   |
|           Context                         |
|           Plugins                         |
|           Hooks                           |
|           Settings                        |
|           Quit                            |
|                                           |
|         3 sources | 23 deployed | 2 !     |
+------------------------------------------+
```

- Up/Down to navigate, Enter to select
- **Dashboard** goes to Overview tab
- **Asset type items** navigate to dashboard with that tab pre-selected
- **Settings** runs `nd init` flow (interactive config setup)
- **Quit** exits
- Status summary at the bottom shows source count, deployed count, issue count
- Status summary loads asynchronously — show spinner until ready

### Header

Renders a single-line bar at the top of the terminal.

```text
 nd - Global | Claude Code | profile: default | 2 issues
```

| Element | Source | Update trigger |
| --- | --- | --- |
| Scope | Picker selection | Scope change |
| Agent | Picker selection / registry | Agent change |
| Profile | ProfileManager.ActiveProfile() | Profile switch |
| Issue count | Deployer.Check() | Any deploy/sync/remove |

When a source is unavailable (NFR-006), append a warning indicator:

```text
 nd - Global | Claude Code | profile: default | 2 issues | 1 source unavailable
```

### TabBar

Horizontal tab labels. Active tab is highlighted. Issue badges on tabs with problems.

```text
 Overview | Skills (1!) | Agents | Commands | Rules | Output Styles | Context | Plugins | Hooks
```

- **Arrow left/right** switches tabs (only in `stateDashboard`, not in other states)
- Each tab label shows an issue badge `(N!)` if assets of that type have health issues
- Overview tab shows all asset types combined
- Tab order: Overview, then asset types in spec order (skills, agents, commands, output-styles, rules, context, plugins, hooks)

**Tab bar overflow (F-04):** If the terminal is narrower than the full tab bar (~95 chars), truncate tab labels to abbreviations (e.g., `Ovr | Ski | Agt | Cmd | Rul | Out | Ctx | Plg | Hk`). Below 60 chars, show only the active tab name with left/right arrows: `< Skills (3/9) >`.

### Table

The primary content area. Shows a list of assets as rows.

**Columns:**

| Column | Content | Width |
| --- | --- | --- |
| Origin icon | `*` manual, `P` pinned, `@` profile-managed | 1 char fixed |
| Status icon | checkmark ok, `x` broken, `~` drifted | 1 char fixed |
| Name | Asset name | flex |
| Source | Source name (not full path) | flex |
| Scope | `global` or `project` | 7 char fixed |
| Status text | `ok`, `broken`, `missing`, `drifted` | flex |

**Column truncation (F-04):** When terminal width is less than 80 chars, hide the Source column. Below 60 chars, hide both Source and Scope columns (only show origin, status, name, status text).

**Sorting:** Issues sort to the top. Within each group, alphabetical by name.

**Inline expand:** Pressing Enter on a selected row expands it in-place to show detail:

```text
  * checkmark my-tdd-skill     local       global   ok
  +--------------------------------------------+
  | Source: ~/skills/my-tdd/SKILL.md           |
  | Target: ~/.claude/skills/my-tdd            |
  | Scope:  global                             |
  | Origin: manual                             |
  | Pinned: no                                 |
  | Profile: default                           |
  +--------------------------------------------+
  P checkmark go-reviewer       dotfiles    global   ok
```

Enter again or Esc collapses the detail.

**Overview tab** shows all deployed assets across all types. Per-type tabs filter to that type only. Same table structure, same columns.

**Duplicate asset warnings (FR-016a):** When two sources contain an asset with the same type and name, the lower-priority duplicate shows a warning icon and tooltip in the detail expand: `Shadowed by: <source-name> (higher priority)`.

### HelpBar

Single-line bar at the bottom. Content changes based on current app state.

**Menu:**

```text
 Up/Down select  Enter open  q quit
```

**Dashboard (normal):**

```text
 Up/Down navigate  Left/Right tabs  Enter detail  d deploy  r remove  s sync  f fix  P profiles  W snapshots  ? help  q menu
```

**Detail expanded:**

```text
 Esc close  r remove  p pin/unpin  s sync
```

**Confirm mode:**

```text
 Remove my-tdd-skill? y confirm  n cancel
```

**Fuzzy finder active:**

```text
 Type to filter  Enter deploy  Esc cancel
```

**List picker active:**

```text
 Up/Down select  Enter confirm  Esc cancel
```

**Prompt active:**

```text
 Type name  Enter save  Esc cancel
```

### Picker

Inline selection widget shown on launch (FR-029). Two fields stacked vertically.

```text
  Scope:   [> Global]  [ Project]
  Agent:   [> Claude Code]

  Enter to continue
```

- Arrow keys move between options
- Enter confirms and transitions to main menu
- If in a directory with `.claude/`, default to Project scope; otherwise Global
- In v1, agent list is just Claude Code (auto-selected if only one)

### FuzzyFinder

Modal overlay for deploying assets. Triggered by `d` from the dashboard.

```text
+-- Deploy asset ---------------------------+
| > search text                             |
|-------------------------------------------|
| go-reviewer        agents/    dotfiles    |
| go-tdd-skill       skills/    local       |
| go-rules           rules/     dotfiles    |
|                                           |
| 3/47 matches                              |
| Enter: deploy  Esc: cancel               |
+-------------------------------------------+
```

- Shows only undeployed assets (available from sources but not currently deployed)
- If triggered from a specific asset-type tab, pre-filters to that type
- If triggered from Overview tab, shows all types
- Type to filter by name (fuzzy match)
- Enter deploys the selected asset to the current scope
- After deploy, modal closes, table refreshes with the new asset

**Index lifecycle (F-06):** The asset index is built lazily on first `d` press by calling `SourceScanner.Scan()`. The index is cached on the root App model and invalidated after any sync operation. If `d` is pressed before scanning completes (stateLoading), show a brief "Scanning sources..." message in the fuzzy finder instead of results, then populate when ready.

**Context file backup (FR-016b):** When deploying a context asset and a non-symlink file already exists at the target location, the fuzzy finder shows an inline warning before confirming: `CLAUDE.md exists at target (not managed by nd). Back up and replace? y/n`. Backup is handled by the deploy engine.

### ListPicker

Generic modal list picker. Used for profile switching and snapshot operations.

**Profile switch (`P` from dashboard):**

```text
+-- Switch profile -------------------------+
|                                           |
|   > default       (active)               |
|     go-dev        12 assets              |
|     web-frontend  8 assets               |
|                                           |
| Enter: switch  Esc: cancel               |
+-------------------------------------------+
```

**Snapshot restore (`W` then `r` from dashboard):**

```text
+-- Restore snapshot -----------------------+
|                                           |
|   > morning-backup    2026-03-15 09:00   |
|     pre-refactor      2026-03-14 16:30   |
|     auto-1            2026-03-14 12:00   |
|                                           |
| Enter: restore  Esc: cancel              |
+-------------------------------------------+
```

### Prompt

Text input modal for naming things. Used for snapshot save.

**Snapshot save (`W` then `s` from dashboard):**

```text
+-- Save snapshot --------------------------+
|                                           |
|   Name: my-snapshot-name                  |
|                                           |
| Enter: save  Esc: cancel                 |
+-------------------------------------------+
```

### Toast

Temporary status messages that appear below the header bar and auto-dismiss after 3 seconds. Used for:

- Operation results: `Deployed go-reviewer to global scope`
- Sync results: `Synced 3 sources. 2 new assets found.`
- Repair results: `Fixed 3/5 issues. 2 remaining.`
- Warnings: `Source 'dotfiles' unavailable (network error)`
- Duplicate warnings (FR-016a): `Asset 'my-skill' in 'repo-b' shadowed by 'repo-a'`

### Confirm (app-level state, not a component) (F-05)

Confirmation is handled as a `ConfirmState` field on the root App model, not as a nested `tea.Model`. When active, it:

1. Changes the HelpBar content to show the confirmation prompt
2. Intercepts the next keypress (`y`, `n`, or Esc)
3. On `y`: executes the stored action callback, refreshes the table
4. On `n` or Esc: restores normal state

```go
type ConfirmState struct {
    Active  bool
    Message string           // "Remove my-tdd-skill?"
    OnYes   func() tea.Cmd   // action to execute
}
```

For single-asset operations: `Remove my-tdd-skill? y confirm  n cancel`

For bulk operations (FR-010, FR-029a): `Switch to profile 'go-dev'? Deploys 12, removes 5. y confirm  n cancel`

## Profile and snapshot flows (FR-019, FR-020, FR-021, FR-023)

### Profile switch

1. User presses `P` on dashboard
2. ListPicker opens with available profiles from `ProfileManager.ListProfiles()`
3. User selects a profile, presses Enter
4. Confirm state activates: `Switch to 'go-dev'? Deploys 12, removes 5. y confirm  n cancel`
5. On `y`: auto-snapshot is saved (FR-029a), profile switch executes via `ProfileManager.Switch()`
6. Table refreshes. Toast shows result. Header updates active profile name.
7. If partial failure: toast shows `Partial switch: 10/12 deployed. Restore previous state? y/n`

### Snapshot save

1. User presses `W` on dashboard, then `s` (or a sub-menu: `W` opens snapshot actions)
2. Prompt opens for snapshot name
3. User types name, presses Enter
4. Snapshot saved. Toast confirms: `Snapshot 'my-backup' saved (23 assets)`

### Snapshot restore

1. User presses `W` on dashboard, then `r`
2. ListPicker opens with available snapshots from `ProfileManager.ListSnapshots()`
3. User selects a snapshot, presses Enter
4. Confirm state activates: `Restore 'my-backup'? Deploys 5, removes 3. y confirm  n cancel`
5. On `y`: auto-snapshot saved first (FR-029a), then restore executes
6. Table refreshes. Toast shows result.

### Snapshot/profile keybinding flow

`W` opens a mini-menu in the help bar rather than a modal:

```text
 W pressed: [s]ave snapshot  [r]estore snapshot  Esc cancel
```

This avoids a dedicated overlay for a two-option choice.

## Bulk operations and error recovery (FR-010)

### Bulk deploy (from profile switch or snapshot restore)

When a bulk operation produces partial failure:

1. Toast shows summary: `Deployed 10/12 assets. 2 failed.`
2. Failed assets appear in the table with status `failed` (red)
3. Help bar shows: `R retry failed  Esc dismiss`
4. `R` retries only the failed assets
5. After retry, toast updates with new results

### Error detail

When fix (`f`) or sync (`s`) produces partial results, the user can press Enter on a failed asset to see the error detail in the inline expand view:

```text
  x broken-link    local    global   broken
  +--------------------------------------------+
  | Source: ~/skills/broken-link/SKILL.md      |
  | Error:  source file not found              |
  | Action: removed orphaned symlink           |
  +--------------------------------------------+
```

## Key ownership matrix (F-02 remediation)

Each key is owned by exactly one app state. Keys not listed for a state are ignored.

| Key | Picker | Menu | Dashboard | Detail | Fuzzy | ListPicker | Prompt | Confirm |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Up/Down | toggle | navigate | navigate rows | - | navigate list | navigate list | - | - |
| Left/Right | toggle | - | switch tabs | - | - | - | - | - |
| Enter | confirm | select | expand detail | collapse | deploy | select | submit | - |
| Esc | - | quit app | back to menu | collapse | close | close | close | cancel |
| `d` | - | - | fuzzy finder | - | - | - | - | - |
| `r` | - | - | remove (confirm) | - | - | - | - | - |
| `s` | - | - | sync all | - | - | - | - | - |
| `f` | - | - | fix all | - | - | - | - | - |
| `p` | - | - | - | pin/unpin | - | - | - | - |
| `P` | - | - | profile switch | - | - | - | - | - |
| `W` | - | - | snapshot menu | - | - | - | - | - |
| `/` | - | - | search filter | - | - | - | - | - |
| `?` | - | - | help overlay | help overlay | - | - | - | - |
| `q` | - | quit | back to menu | - | - | - | - | - |
| `y`/`n` | - | - | - | - | - | - | - | yes/no |
| `R` | - | - | retry failed | - | - | - | - | - |
| Typing | - | - | - | - | filter input | - | text input | - |
| Backspace | - | - | back to menu | collapse | edit input | close | edit input | cancel |

## Empty states (guided onboarding)

### No sources registered

Shown on menu and dashboard when `SourceScanner.Sources()` returns empty.

```text
+------------------------------------------+
|                                           |
|  No sources registered.                  |
|                                           |
|  Add a source to get started:            |
|    nd source add ~/my-skills             |
|    nd source add owner/repo              |
|                                           |
|  Press q to quit and add a source.       |
+------------------------------------------+
```

### No assets deployed (sources exist but nothing deployed)

```text
  No assets deployed.
  Press d to deploy assets from your sources.
  3 sources registered | 47 assets available
```

### No assets of type (per-type tab is empty)

```text
  No skills deployed.
  Press d to deploy a skill.
```

### No issues

When the user presses `f` with zero issues:

```text
Toast: No issues found. All deployments healthy.
```

### No profiles

When the user presses `P` with no profiles saved:

```text
  No profiles saved.
  Create a profile: nd profile save <name>
```

## Loading states (F-03 remediation)

### Initial launch

After picker and menu, the dashboard shows a loading indicator while deployment state is read and the first scan completes:

```text
 nd - Global | Claude Code | loading...
 Overview | Skills | Agents | ...
  Loading deployment state...  [spinner]
```

The loading state transitions to the populated table once `Deployer.Status()` returns. This avoids the "0 assets then populated" flash.

### Health check progression

After the table populates from deployment state, health checks run asynchronously. During this period:

- Status icons show a neutral indicator (`-`) instead of checkmark or x
- As health results arrive, icons update in batches (every 100ms) to reduce render churn
- Once all checks complete, the header issue count updates

### Source sync

When `s` (sync) is pressed, a toast shows `Syncing sources...` with a spinner. The table remains interactive. When sync completes, the table refreshes and toast shows results.

## Terminal size handling (F-04 remediation)

### Minimum dimensions

- **Minimum width:** 40 characters. Below this, show a centered message: `Terminal too narrow. Resize to at least 40 chars wide.`
- **Minimum height:** 10 rows. Below this, show: `Terminal too small. Resize to at least 10 rows tall.`

### Responsive layout

| Width | Behavior |
| --- | --- |
| 95+ chars | Full tab names, all columns |
| 80-94 chars | Full tab names, all columns (Source column narrower) |
| 60-79 chars | Abbreviated tab names, hide Source column |
| 40-59 chars | Active tab only with arrows, hide Source + Scope columns |
| Below 40 | "Too narrow" message |

| Height | Behavior |
| --- | --- |
| 20+ rows | Full layout (header + tabs + table + help) |
| 10-19 rows | Reduce table visible rows, keep header + help |
| Below 10 | "Too small" message |

### Resize handling

The app listens for `tea.WindowSizeMsg` and reflows all components. No scroll state is lost on resize.

## Issue surfacing (FR-013)

Issues are surfaced at three levels:

1. **Header bar**: Total issue count badge (e.g., `2 issues`)
2. **Tab labels**: Per-type issue count badge (e.g., `Skills (1!)`)
3. **Table rows**: Affected rows use distinct status icons (`x` broken, `~` drifted) and status text. Issues sort to the top of the table.

When the user presses `f` (fix), the deploy engine's `Sync` method is called. A toast shows progress. After completion:

- Success: `Fixed 5/5 issues.`
- Partial: `Fixed 3/5 issues. 2 remaining.` (Enter on failed rows shows error detail)
- Nothing to fix: `No issues found. All deployments healthy.`

## Auto-snapshots (FR-029a)

Before bulk operations (profile switch, snapshot restore, bulk remove), the TUI:

1. Calls auto-snapshot via the profile manager to save current state
2. Executes the operation
3. If the operation fails midway, shows confirm: `Partial failure. Restore previous state? y/n`

Auto-snapshots are transparent — no separate UI. They're a safety net surfaced only on failure.

## Context file metadata (FR-016c)

When the active tab is Context, the table includes an extra detail from `_meta.yaml`:

```text
  * checkmark go-project-rules    local    global   ok     Go project rules and conventions
```

The description field (from `_meta.yaml`) is appended as a trailing flex column, truncated to fit.

## Performance (NFR-001)

Target: TUI initial menu renders within 500ms with 500+ assets.

Strategy:

- **Menu render**: Synchronous — only reads source count and cached deployment state summary. Renders immediately.
- **Dashboard first paint**: Load deployment state (YAML file) synchronously — this is the only blocking I/O. Show loading spinner until ready.
- **Source scanning**: Asynchronous after dashboard renders. Assets already in state file are shown immediately.
- **Health checks**: Run in background after table populates. Status icons start neutral (`-`), update in batches as results arrive (F-03).
- **Fuzzy finder index**: Built lazily on first `d` press, cached on root model, invalidated after sync (F-06).

## Styling (Lip Gloss)

Minimal, terminal-friendly palette:

| Element | Style |
| --- | --- |
| Header bar | Bold, subtle background |
| Active tab | Bold + underline |
| Inactive tab | Dim |
| Issue badge | Red/yellow |
| Status ok | Green icon |
| Status broken | Red icon + red text |
| Status drifted | Yellow icon |
| Origin pinned | Blue `P` |
| Origin profile | Cyan `@` |
| Table selection | Reverse video (highlighted row) |
| Detail expand | Bordered box, slightly indented |
| Modal overlays | Bordered, dimmed background |
| Help bar | Dim text, action keys bold |
| Toast | Inverse bar below header, auto-fading |
| Empty state | Centered, dim, slightly larger text |
| Loading spinner | Animated dots, dim |

No hardcoded colors — use Lip Gloss adaptive colors that work on both light and dark terminals.

## Open questions

| # | Question | Impact |
| --- | --- | --- |
| TQ-1 | Should the picker be re-accessible via a hotkey after launch, or is scope/agent fixed for the session? | Low — can add later |
| TQ-2 | Should the table support multi-select (Space to toggle) for bulk remove/deploy? | Medium — affects confirm flow |
| TQ-3 | Should there be a Sources tab/view for managing sources from the TUI, or is that CLI-only? | Low — not in spec FRs |
| TQ-4 | What Bubble Tea v2 specific APIs should we target vs. v1 compatibility? | Medium — affects dependency version |
| TQ-5 | Should the TUI detect external state changes (another terminal running `nd deploy`) and auto-refresh? | Medium — adds complexity |
| TQ-6 | How should the Settings menu item work? Run `nd init` inline or open config in `$EDITOR`? | Low — defer to implementation |
| TQ-7 | Should the Plugins tab disable deploy/remove actions since plugins have a different lifecycle? | Low — spec is ambiguous |

## Requirement traceability

| FR | Design element |
| --- | --- |
| FR-001 | Root `nd` command launches `tui.App` |
| FR-010 | Bulk deploy reporting, "R retry failed" action, partial failure handling |
| FR-016a | Duplicate asset warning in toast + detail expand "Shadowed by" text |
| FR-016b | Context file backup offer in fuzzy finder deploy flow |
| FR-016c | Context tab table includes `_meta.yaml` description column |
| FR-017 | TabBar with Overview + per-type tabs; Table component |
| FR-018 | Header bar with profile, scope, agent, issue count |
| FR-019 | Keyboard actions: `d` deploy, `f` fix, `s` sync, `r` remove, `P` profile switch, `W` snapshots |
| FR-020 | Snapshot save via `W` then `s`, Prompt component for name |
| FR-021 | Snapshot restore via `W` then `r`, ListPicker component |
| FR-023 | Profile switch via `P`, ListPicker component, confirm with diff summary |
| FR-024a | Pinned asset indicator (`P` icon) in table, preserved during profile switch |
| FR-028 | Menu component: dashboard, asset types, settings, quit. Esc/Backspace for back. |
| FR-029 | Picker component on launch for scope/agent selection |
| FR-029a | Auto-snapshot before bulk ops, restore offer on failure |
| NFR-001 | Async loading strategy, lazy fuzzy index, menu-first for fast initial render |
| NFR-006 | Source-unavailable warning in header bar and toast |

## Audit findings addressed

| Finding | Source | Resolution |
| --- | --- | --- |
| F-01: Concrete service types | Architecture review | Consumer-defined interfaces in `services.go` |
| F-02: Key ownership ambiguity | Architecture review | Key ownership matrix per app state |
| F-03: Loading flash/flicker | Architecture review | Loading state, neutral status icons, batched updates |
| F-04: Terminal size handling | Architecture review | Responsive layout rules, minimum dimensions |
| F-05: Confirm as nested model | Architecture review | Changed to `ConfirmState` app-level field |
| F-06: Fuzzy index lifecycle | Architecture review | Lazy build, cache on root model, invalidate on sync |
| Missing main menu (FR-028) | Spec gap audit | Menu component added |
| Missing profile/snapshot flows | Spec gap audit | Full flows designed with keybindings and components |
| Missing bulk error recovery | Spec gap audit | "R retry failed" action, partial failure handling |
| Missing empty states | Spec gap audit | Guided onboarding messages for every empty view |
| Missing origin indicator | Spec gap audit | Origin icon column (`*`, `P`, `@`) |
| Missing source-unavailable | Spec gap audit | Header warning indicator, toast messages |
| Missing duplicate warnings | Spec gap audit | Toast + detail expand "Shadowed by" |
| Missing context backup offer | Spec gap audit | Inline warning in fuzzy finder for FR-016b |

## Pre-TUI prerequisites

1. Add `ListProfiles()` delegation method to `profile.Manager`
2. Add `ListSnapshots()` delegation method to `profile.Manager`

## Next steps

1. Create implementation plan with phased delivery
2. Research Bubble Tea v2 API specifics (component model, message routing)
