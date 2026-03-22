---
date: 2026-03-22
topic: tui-layer-redesign
---

# TUI Layer Redesign

## What We're Building

An interactive terminal interface launched by bare `nd` (no args), providing full parity with the CLI through a menu-driven, wizard-style experience. The TUI is a thin presentation layer over the same service APIs the CLI uses.

The previous TUI attempt was deprecated (2026-03-18) and removed (2026-03-20) because it was too complex and had poor UX. This redesign starts from scratch with a simpler mental model: sequential screens with guided flows, not a multi-panel dashboard.

## Why This Approach

**Menu-driven wizard-style** was chosen over dashboard, tab-focused, or hybrid approaches for these reasons:

1. **The previous TUI failed because of complexity.** A full dashboard with panels and persistent views was too much surface area to build and maintain well. Wizard-style is inherently simpler — each screen does one thing.

2. **Progressive disclosure aligns with nd's design philosophy.** The spec defines three user levels (Level 0-2). A menu naturally presents Level 0 actions first, with advanced options further down.

3. **huh v2 provides 80% of the UI components for free.** Select, MultiSelect, Input, Confirm, and Spinner are all built into huh. The TUI shell (header, navigation, screen routing) is the only custom Bubble Tea code needed.

4. **Full CLI parity is achievable.** Every CLI command maps to a menu path. The flat action menu provides a 1:1 mapping to the command tree.

## Technology Stack

| Component | Library | Module Path | Purpose |
|-----------|---------|-------------|---------|
| App shell | Bubble Tea v2 | `charm.land/bubbletea/v2` | Lifecycle, alt screen, key routing, screen management |
| Forms | huh v2 | `charm.land/huh/v2` | Select, MultiSelect, Input, Confirm fields |
| Styling | Lip Gloss v2 | `charm.land/lipgloss/v2` | Header bar, borders, colors, layout |
| Components | Bubbles v2 | `charm.land/bubbles/v2` | Progress bar, spinner, list, viewport |
| CLI | Cobra | `github.com/spf13/cobra` | Unchanged — TUI is an alternative entry point |

**Migration note:** The existing `cmd/export.go` uses huh v1 (`github.com/charmbracelet/huh`). This will be migrated to huh v2 as part of TUI implementation since both can't coexist in the same binary.

### Why Bubble Tea v2

- 10x faster rendering (ncurses-based "Cursed Renderer")
- Declarative API (declare terminal features, don't manage them)
- Better composability with Lip Gloss v2 (pure styling, BT manages I/O)
- Released Feb 2026 — targeting the latest for a new feature is low-risk
- huh v2 requires BT v2 anyway

## Design System (Lip Gloss v2)

nd's TUI follows a **visually minimal** aesthetic: no borders, no box-drawing characters, no decorative chrome. Structure is communicated through whitespace, indentation, and sparse color. The design system lives in `internal/tui/theme/`.

### Design Principles

1. **Whitespace over borders.** Sections are separated by blank lines, not box-drawing characters.
2. **Color is semantic, not decorative.** Color appears only on status indicators, the active selection, and errors. Everything else is the terminal's default foreground.
3. **Text conveys meaning independently of color.** Every colored element has a text-only fallback (`OK`, `!!`, `??`).
4. **Let huh do the heavy lifting.** huh's built-in form rendering is already clean and minimal. Don't fight it.

### Color Palette

Minimal palette — just 5 semantic accent colors plus neutral tones. Uses **AdaptiveColor** for auto dark/light switching.

```go
// internal/tui/theme/palette.go

var (
    Subtle   = adaptive("#6c7086", "#9ca0b0")  // Secondary text, metadata
    Primary  = adaptive("#89b4fa", "#1e66f5")  // Active selection, prompts
    Success  = adaptive("#a6e3a1", "#40a02b")  // OK, deployed
    Warning  = adaptive("#f9e2af", "#df8e1d")  // Drifted, orphaned
    Danger   = adaptive("#f38ba8", "#d20f39")  // Broken, errors
)
```

Based on Catppuccin (Mocha dark, Latte light). Most text uses the terminal's default foreground — the palette only appears where it carries meaning.

### Style Registry

Small and flat. No deeply nested struct hierarchy.

```go
// internal/tui/theme/styles.go

type Styles struct {
    Subtle    lipgloss.Style  // Dimmed text (metadata, paths, help keys)
    Primary   lipgloss.Style  // Active items, prompts
    Success   lipgloss.Style  // OK indicators
    Warning   lipgloss.Style  // Drift/orphan indicators
    Danger    lipgloss.Style  // Errors, broken status
    Bold      lipgloss.Style  // Section headers, counts
}
```

Six styles. That's it. The header, help bar, status view, and results all compose from these six. No per-component style types.

### Glyph System

```go
var (
    OK      = "ok"   // Styled Success
    Broken  = "!!"   // Styled Danger
    Drifted = "??"   // Styled Warning
    Orphan  = "--"   // Styled Subtle
    Dot     = "·"    // Header separator (Subtle)
    Arrow   = "->"   // Symlink paths (Subtle)
)
```

### huh v2 Theme

Inherit Catppuccin, no overrides unless something clashes. Catppuccin is already visually minimal.

```go
type NdTheme struct{}

func (NdTheme) Theme(isDark bool) *huh.Styles {
    return huh.ThemeCatppuccin().Theme(isDark)
}
```

### Dark/Light Detection

Detected once on launch, passed to all screens:

```go
func Run(svc Services) error {
    isDark := lipgloss.HasDarkBackground(os.Stdin, os.Stderr)
    styles := theme.NewStyles(isDark)
    p := tea.NewProgram(newModel(svc, styles))
    _, err := p.Run()
    return err
}
```

## Architecture

### Root Container Pattern

The TUI uses the standard Bubble Tea multi-view pattern: a single root model that owns a stack of screens and routes messages.

```
internal/tui/
  tui.go          # Root model: Init, Update, View. Screen routing, header, help bar.
  header.go       # Persistent header component (profile, scope, agent, counts)
  helpbar.go      # Bottom help bar component (context-sensitive shortcuts)
  screens.go      # Screen enum/interface
  main_menu.go    # Main menu screen (huh Select)
  deploy.go       # Deploy flow: type picker -> asset multi-select -> confirm
  remove.go       # Remove flow: asset picker -> confirm
  status.go       # Status view: grouped asset list with health indicators
  doctor.go       # Doctor flow: run checks -> show results -> offer fixes
  profile.go      # Profile menu: switch, create, list, deploy
  snapshot.go     # Snapshot menu: save, restore, list
  source.go       # Source menu: list, add, remove, sync
  settings.go     # Settings menu: edit config, show info
  progress.go     # Reusable progress bar for bulk operations
  export.go       # Export flow (migrated from cmd/export.go interactive mode)
```

### Screen Interface

Every screen implements the same interface, composable with the root model:

```go
type Screen interface {
    tea.Model
    Title() string    // Used by header breadcrumb
    // Screens return NavigateMsg to signal transitions
}

// Navigation messages
type NavigateMsg struct {
    Screen Screen  // nil = go back
}

type BackMsg struct{}   // Go back one level
type QuitMsg struct{}   // Exit the TUI
```

### Message Routing

The root model handles global concerns, then delegates to the active screen:

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyPressMsg:
        switch msg.String() {
        case "q", "ctrl+c":
            return m, tea.Quit
        case "esc":
            return m.goBack()
        // backspace is NOT intercepted — screens handle it (e.g., text input deletion)
        }
    case NavigateMsg:
        return m.pushScreen(msg.Screen)
    case BackMsg:
        return m.goBack()
    }

    // Delegate to current screen
    updated, cmd := m.currentScreen().Update(msg)
    m.screens[len(m.screens)-1] = updated.(Screen)
    return m, cmd
}
```

### Integration with cmd/

The TUI is launched from `cmd/root.go` when `nd` is invoked with no args:

```go
RunE: func(cmd *cobra.Command, args []string) error {
    if !isTerminal() {
        return cmd.Help()  // Non-interactive: show help
    }
    return tui.Run(app)    // Interactive: launch TUI
}
```

The `tui.Run(app)` function receives the existing `*cmd.App` struct, giving the TUI access to all lazily-initialized services (SourceManager, DeployEngine, ProfileManager, etc.) without duplicating initialization logic.

## Screen Designs

### Persistent Header (all screens)

```
  go-dev · global · claude                     12 deployed  2 issues
```

One line, no borders. Profile, scope, agent left-aligned. Counts right-aligned. The `·` separator and counts are styled Subtle. Issue count is styled Danger when > 0. Updates after every operation.

**Smart defaults on launch (FR-029):**
- If only one agent detected, skip agent selection
- If inside a project with `.nd/`, default to project scope
- If no project detected, default to global
- When ambiguous (multiple agents, unclear scope), prompt with huh Select

### Help Bar (all screens)

```
  esc back  j/k navigate  enter select  ? help  q quit
```

Styled Subtle. Context-sensitive. No colons, no brackets — just key and action separated by a space.

### Main Menu

```
  go-dev · global · claude                     12 deployed  2 issues

  What would you like to do?

  > Deploy assets
    Remove assets
    Browse assets
    View status
    Run doctor

    Switch profile
    Manage snapshots
    Pin/Unpin assets
    Manage sources

    Export plugin
    Settings
    Quit

  esc quit  j/k navigate  enter select  ? help
```

Blank lines separate groups (Level 0 / Level 1 / Level 2+System). No divider characters. Implemented as a single huh Select.

### Deploy Flow

**Step 1: Pick asset type**
```
  go-dev · global · claude                     12 deployed  2 issues

  Deploy

  > Search all
    Skills
    Commands
    Rules
    Context
    Agents
    Output styles
    Hooks
    All types

  esc back
```

**Step 2: Select assets** (huh MultiSelect, filtered to undeployed assets of chosen type)
```
  go-dev · global · claude                     12 deployed  2 issues

  Deploy skills

  [x] greeting                                          local-skills
  [ ] code-review          review helper                 community
  [x] debugging                                          local-skills
  [ ] refactoring          refactor patterns             community

  2 selected

  esc back  space toggle  enter confirm  / filter  a all
```

Assets show name, description (from `_meta.yaml`, styled Subtle), and source (right-aligned, styled Subtle).

**Step 3: Execute** (with progress bar)
```
  go-dev · global · claude                     12 deployed  2 issues

  Deploying 2 skills...

  ████████████████████░░░░░░░░░░  1/2  greeting
```

**Step 4: Result**
```
  go-dev · global · claude                     14 deployed  2 issues

  Deployed 2/2 skills

    greeting       -> ~/.claude/skills/greeting
    debugging      -> ~/.claude/skills/debugging

  enter menu
```

Paths styled Subtle. Header count already updated. On partial failure, show failures with per-item error details (asset, path, reason per NFR-004) and offer "Retry failed" option.

### Remove Flow

Same pattern as deploy but shows only deployed assets. Confirmation uses huh Confirm.

```
  Remove 3 skills?
  An auto-snapshot will be saved first.

  > Yes, remove
    No, go back
```

### Status View

```
  go-dev · global · claude                     12 deployed  2 issues

  Status

  skills
    ok  greeting           global   profile    local-skills
    ok  debugging          global   profile    local-skills
    !!  old-skill          global   manual     local-skills   broken

  commands
    ok  hello.md           global   profile    local-skills

  rules
    ok  no-emoji.md        global   pinned     community

  2 issues found

  esc back  f fix  d deploy  r remove  / filter
```

No decorative borders. Type headers are just bold text. `ok` styled Success, `!!` styled Danger. Metadata columns styled Subtle.

### Doctor Flow

```
  go-dev · global · claude                     12 deployed  2 issues

  Doctor found 2 issues

  skills/old-skill         broken     target missing
  context/legacy           orphaned   source deleted

  Fix all 2 issues?

  > Yes, fix
    No, go back
```

Uses huh Confirm (consistent with all other confirmations per A19).

### Profile Submenu

```
  go-dev · global · claude                     12 deployed  2 issues

  Profiles                                     active: go-dev

  > Switch profile
    Create profile
    List profiles
    Deploy profile
    Back
```

Active profile shown in header area, styled Subtle.

### Snapshot Submenu

```
  Snapshots

  > Save snapshot
    Restore snapshot
    List snapshots
    Back
```

### Source Submenu

```
  Sources

  > List sources
    Add local source
    Add Git source
    Remove source
    Sync all
    Back
```

### Sync Flow

```
  go-dev · global · claude                     12 deployed  2 issues

  Syncing sources...

  ████████████████████████████░░  2/3  community

  Done. 3/3 sources synced.

  enter menu
```

### Export Flow

Migrated from the existing `cmd/export.go` `runExportInteractive()`. Same huh forms (asset multi-select, metadata inputs, confirm) but rendered inside the TUI shell with header and navigation.

### Settings Submenu

```
  Settings

  > Edit config
    Show config path
    Show version
    Back
```

## Navigation

### Key Bindings

| Key | Context | Action |
|-----|---------|--------|
| `j` / `down` | Lists, menus | Move selection down |
| `k` / `up` | Lists, menus | Move selection up |
| `enter` | Lists, menus | Select / confirm |
| `space` | MultiSelect | Toggle item |
| `esc` | Any screen (no text input) | Go back one level |
| `q` / `ctrl+c` | Any screen | Quit (confirm if mid-operation) |
| `?` | Any screen | Show help overlay |
| `/` | Status, lists | Filter / search |
| `a` | MultiSelect | Select all / deselect all |

### Vim-Style Navigation

`h`/`l` for back/forward are NOT used to avoid conflicts with text input fields. `j`/`k` for up/down work because huh selects use these natively.

### Screen Stack

Navigation uses a stack (like a mobile navigation controller):

```
Main Menu → Deploy → Pick Type → Select Assets → Confirm → Result
                                                            ↓ (enter)
Main Menu ← ← ← ← ← ← ← ← ← ← ← ← ← ← ← ← ← ← ← ←
```

After a completed action (deploy, remove, etc.), pressing enter returns directly to the main menu (not back through every step). The `esc` key during a flow goes back one step.

## Progress UX

Bulk operations (deploy, remove, sync) show a progress bar with item count:

```
  ████████████████░░░░░░░░░░░░░░  8/20  deploying skills/debugging
```

Implemented using bubbles/progress + a custom wrapper that updates on each item completion. The operation runs in a goroutine, sending progress messages to the Bubble Tea program via `p.Send()`.

## Error Handling

- **Operational errors** (deploy failure, missing source): Shown inline on the result screen with red styling. Offer "Retry" or "Back to menu" options.
- **Fatal errors** (can't read config, state locked): Exit the TUI with a clear error message printed to stderr, same as CLI behavior.
- **Partial failures**: Show summary (e.g., "Deployed 18/20. 2 failed:") with per-item error details. Offer "Retry failed" option.

## Relationship to CLI

The TUI and CLI share all business logic via `cmd.App`:

```
           ┌──────────────┐
           │   cmd.App    │  (lazy service initialization)
           └──────┬───────┘
                  │
        ┌─────────┴─────────┐
        │                   │
  ┌─────┴──────┐    ┌──────┴──────┐
  │  cmd/*.go  │    │ tui/*.go    │
  │  (Cobra)   │    │ (Bubble Tea)│
  └────────────┘    └─────────────┘
```

Both layers call the same service methods. The TUI doesn't bypass the CLI — it's a parallel presentation layer.

**Testing strategy:** TUI screens are tested via Bubble Tea's programmatic message sending (tea.Send / model.Update). No need for terminal emulation. Service layer is already tested at 80%+ coverage.

## FR Coverage

| FR | Description | TUI Implementation |
|----|-------------|-------------------|
| FR-001 | `nd` bare invocation launches TUI | `root.go` RunE detects terminal, calls `tui.Run()` |
| FR-002 | All operations available without TUI | Unchanged — CLI commands remain |
| FR-017 | Dashboard with tabs per asset type | Replaced by: Status view grouped by type, type-first drill-downs |
| FR-018 | Header shows profile/scope/agent/health | Persistent header bar on all screens |
| FR-019 | Inline actions | Main menu actions + context-sensitive hotkeys |
| FR-028 | Main menu with back navigation | Flat action menu with esc/backspace stack navigation |
| FR-029 | Scope/agent selection on launch | Smart defaults with prompt-when-ambiguous |
| FR-016c | `_meta.yaml` display | Shown in asset selection lists (deploy, remove) |
| NFR-001 | Initial render within 500ms | Menu-driven approach is inherently fast (no data scan on launch) |

**Spec deviation:** FR-017 specified a "tabbed interface" dashboard. This redesign replaces tabs with a menu-driven flow. The flat action menu + type-first drill-down achieves the same goal (access all asset types) without the complexity that caused the previous TUI to fail. The spec should be updated to reflect this change.

## Open Questions

1. **Should `nd` without args show a "loading..." state while detecting agents/scope?** Smart defaults require probing the filesystem. If detection takes >100ms, a brief spinner may be needed before the menu appears.

2. **Should the TUI support mouse interaction?** Bubble Tea v2 supports mouse events. Enabling mouse click on menu items would improve accessibility but adds complexity.

3. **Should the help overlay (`?`) be a full-screen help page or a floating panel?** Full-screen is simpler to implement. Floating panel preserves context but requires z-order management.

4. **Should the TUI support `--dry-run`?** The CLI has `--dry-run` as a global flag. Should the TUI expose this as a toggle in the header or per-operation?

5. **Should operations that modify state (deploy, remove) auto-refresh the header counts?** This is the expected behavior but requires the header to re-query the state store after each operation returns.

## Estimated Scope

| Component | Files | Complexity |
|-----------|-------|------------|
| Root model + screen routing | 2-3 | Medium |
| Header + help bar | 2 | Low |
| Main menu | 1 | Low (huh Select wrapper) |
| Deploy flow | 1 | Medium (3-step wizard) |
| Remove flow | 1 | Low (2-step wizard) |
| Status view | 1 | Medium (lipgloss table formatting) |
| Doctor flow | 1 | Low-Medium |
| Profile submenu + flows | 1 | Medium |
| Snapshot submenu + flows | 1 | Low |
| Source submenu + flows | 1 | Medium (huh Input for add) |
| Sync progress | 1 | Low |
| Export migration | 1 | Low (port existing code) |
| Settings | 1 | Low |
| Progress component | 1 | Low (bubbles/progress wrapper) |
| huh v1 -> v2 migration | 0 (edit) | Low (import path + minor API changes) |
| **Total** | ~16 files | Moderate overall |

All new code goes in `internal/tui/`. The `cmd/` layer gets one change: `root.go` RunE calls `tui.Run()` when invoked interactively with no args.

---

## Audit Remediation

Three parallel audits (spec compliance, architecture, UX) identified issues. This section documents all validated findings and their resolutions. The original design above should be read in conjunction with these remediations.

### [A1] Circular import: `cmd` <-> `internal/tui` (Critical, Architecture #1)

**Problem:** `tui.Run(app)` takes `*cmd.App`, but `cmd/root.go` imports `internal/tui`. Go forbids circular imports.

**Resolution:** Extract a `Services` interface in a shared package (`internal/nd/services.go` or `internal/tui/services.go`). The TUI accepts this interface, not the concrete `*cmd.App`.

```go
// internal/tui/services.go
type Services interface {
    SourceManager() (*sourcemanager.SourceManager, error)
    AgentRegistry() (*agent.Registry, error)
    DefaultAgent() (*agent.Agent, error)
    DeployEngine() (*deploy.Engine, error)
    ProfileManager() (*profile.Manager, error)
    ProfileStore() (*profile.Store, error)
    StateStore() *state.Store
    OpLog() *oplog.Writer
    ScanIndex() (*sourcemanager.ScanSummary, error)
    // Getters for display
    Scope() nd.Scope
    ConfigPath() string
    DryRun() bool
}
```

`cmd.App` already satisfies this interface — no changes to App needed. `tui.Run()` becomes `tui.Run(svc Services)`.

### [A2] Global key interception breaks text input (Critical, Architecture #7, UX #2/#3)

**Problem:** The root model intercepts `q`, `esc`, and `backspace` before delegating to the current screen. This makes it impossible to type `q` in a path, press backspace to delete characters, or press esc to cancel a field.

**Resolution:** Screens report whether they are in "input mode" (accepting text). The root model only intercepts global keys when the current screen is NOT in input mode.

```go
type Screen interface {
    tea.Model
    Title() string
    InputActive() bool  // True when a text field has focus
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyPressMsg:
        // Only handle global keys when no text input is active
        if !m.currentScreen().InputActive() {
            switch msg.String() {
            case "q", "ctrl+c":
                return m, tea.Quit
            case "esc":
                return m.goBack()
            }
        } else {
            // In input mode, only ctrl+c force-quits
            if msg.String() == "ctrl+c" {
                return m, tea.Quit
            }
        }
    // ... navigation messages unchanged
    }

    // Always delegate to current screen
    updated, cmd := m.currentScreen().Update(msg)
    m.screens[len(m.screens)-1] = updated.(Screen)
    return m, cmd
}
```

`backspace` is NEVER intercepted by the root — screens handle their own back-navigation by emitting `BackMsg` when appropriate (e.g., when a huh Select receives esc with no input active).

### [A3] App caches single agent/scope — no mid-session switching (Critical, Architecture #3/#9, UX #10/#11, Spec #24)

**Problem:** `App.DeployEngine()` caches a single agent. `App.SourceManager()` is built with a fixed project root. There's no way to switch agent or scope without restarting.

**Resolution:** Add `Reset()` methods to the Services interface and App:

```go
func (a *App) ResetForScope(scope nd.Scope, projectRoot string) {
    a.Scope = scope
    a.ProjectRoot = projectRoot
    // Clear cached services that depend on scope/agent
    a.sm = nil
    a.reg = nil
    a.eng = nil
    a.profMgr = nil
    a.pstore = nil
    a.sstore = nil
}
```

The TUI's main menu gains a "Switch scope/agent" option that calls `Reset()` and re-detects. The header updates after the switch.

### [A4] DeployBulk lock + progress deadlock (Important, Architecture #6)

**Problem:** `DeployBulk` holds the file lock for the entire batch. If the progress goroutine tries to read state, it deadlocks.

**Resolution:** The progress callback does NOT read state. Instead, `DeployBulk` is modified to accept an optional progress callback:

```go
type ProgressFunc func(completed int, total int, current string)

func (e *Engine) DeployBulkWithProgress(reqs []Request, fn ProgressFunc) (*BulkResult, error)
```

The callback is invoked inside the lock, on each item completion, and sends a Bubble Tea message via `p.Send()`. The progress display only reads the message data, never the state store.

### [A5] App lazy init is not goroutine-safe (Important, Architecture #4)

**Problem:** `App`'s check-and-set pattern for lazy initialization has no mutex protection.

**Resolution:** All service operations in the TUI run sequentially in the Bubble Tea Update loop (single-threaded by design). Goroutines are only used for long-running operations, which receive a pre-initialized service reference before the goroutine starts — they never call `App` accessors.

```go
// In a screen's Update:
case startDeployMsg:
    eng, err := m.services.DeployEngine()  // Called in Update (single-threaded)
    if err != nil { ... }
    return m, func() tea.Msg {
        result, _ := eng.DeployBulk(reqs)  // Goroutine uses pre-fetched engine
        return deployDoneMsg{result}
    }
```

This avoids the need for mutexes while keeping the goroutine pattern safe. Document this as a constraint: **never call Services methods from inside a goroutine.**

### [A6] huh v1/v2 coexistence claim is imprecise (Important, Architecture #5)

**Problem:** The brainstorm says huh v1 and v2 "can't coexist." Technically, different Go module paths CAN coexist. The real issue is type incompatibility — huh v1's `tea.Model` (BT v1) and TUI code's `tea.Model` (BT v2) are different types.

**Correction:** Replace "can't coexist in the same binary" with: "While Go allows different major versions as separate modules, huh v1 depends on Bubble Tea v1 types (`tea.Model`, `tea.Msg`) that are incompatible with BT v2's types. Embedding a huh v1 form inside a BT v2 program would not compile. Therefore, the existing huh v1 usage in `cmd/export.go` must be migrated to huh v2 before the TUI can be built."

### [A7] Screen stack has no "pop to root" mechanism (Important, Architecture #8, UX #12)

**Problem:** After a completed action, pressing enter should return to the main menu. But `BackMsg` only pops one level, and `NavigateMsg{nil}` means "go back one."

**Resolution:** Add a `PopToRootMsg`:

```go
type PopToRootMsg struct{}  // Clear stack, return to main menu
```

Result screens emit `PopToRootMsg` on enter. The root model handles this by resetting the screen stack to just the main menu. The result screen mockup is updated to say "press enter to return to menu" (not just "press enter to continue").

### [A8] Missing screens: list, init, pin/unpin (Important, Spec #1/#2/#3/#8)

**Problem:** The brainstorm claims "full parity" but omits `list`, `init`, `pin`/`unpin`, and `uninstall` from the TUI.

**Resolution:** Add these to the main menu and file list:

**Updated main menu** (grouped by progressive disclosure level):

```
  What would you like to do?

  > Deploy assets
    Remove assets
    Browse assets          <- NEW (nd list equivalent, FR-016d)
    View status
    Run doctor
  ─────────────────────
    Switch profile
    Manage snapshots
    Pin/Unpin assets       <- NEW (FR-024a)
    Manage sources
  ─────────────────────
    Export plugin
    Settings
    Quit
```

**Grouping with separators** addresses UX finding #7 (11 items, no progressive disclosure). Level 0 actions are above the first separator. Level 1 below. Level 2 (export) below the second.

**Additional screens:**

| Screen | File | Notes |
|--------|------|-------|
| Browse assets | `browse.go` | Filterable list of all assets with status, type, source. FR-016d. |
| Pin/Unpin | `pin.go` | Select deployed assets -> toggle pin. FR-024a. |
| Init (first-run) | `init.go` | Guided setup: add source, pick scope, detect agent. FR-025. Shown automatically on first run. |

**Explicit omissions** (not in TUI, by design):
- `nd uninstall` — destructive, better as explicit CLI command
- `nd completion` — shell-specific, not meaningful in TUI

The "full parity" claim is reworded to "near-complete parity — all operations except uninstall and shell completion."

### [A9] First-run / empty state UX (Critical, UX #1, Spec #16/#20)

**Problem:** No design for what happens when a new user launches `nd` with no config, sources, or assets.

**Resolution:** The TUI detects first-run state (no config file exists) and automatically launches the Init screen before showing the main menu:

```
  Welcome to nd!

  Let's set up your first asset source.

  Where are your assets stored?

  > Local directory (I have a folder of skills/commands)
    Git repository (I want to clone a repo)
    Skip for now

  ...
```

For empty states within individual screens:
- **Deploy with no assets:** "No assets found. Add a source first. [press s to manage sources]"
- **Status with nothing deployed:** "Nothing deployed yet. [press d to deploy assets]"
- **Profiles with none created:** "No profiles yet. [press c to create one]"

Each empty state includes an actionable hint pointing to the relevant menu path.

### [A10] Remove confirmation is misleading (Important, UX #13, Spec #9)

**Problem:** "This cannot be undone" but auto-snapshots make it recoverable.

**Resolution:** Change confirmation text:

```
  Remove 3 skill(s)?
  (An auto-snapshot will be saved before removal. You can restore it later.)

  > Yes, remove
    No, go back
```

### [A11] "Sync sources" is duplicated in main menu and Sources submenu (Important, UX #6)

**Resolution:** Remove "Sync sources" from the main menu. It's accessible via "Manage sources" -> "Sync all". The main menu item was added for convenience but the duplication causes confusion.

### [A12] Deploy flow is too many steps for common case (Important, UX #4)

**Resolution:** Add search/filter to the deploy flow. The type picker gains a "Search all" option at the top:

```
  Deploy — select asset type:

  > Search all (type to filter)    <- NEW
    Skills
    Commands
    ...
```

Selecting "Search all" opens a fuzzy-filter text input over all undeployed assets. This allows deploying a known asset in 3 interactions instead of 6: Menu -> Deploy -> type name -> enter.

### [A13] No search/filter on asset lists (Important, UX #5, Spec #23)

**Resolution:** All asset lists (deploy MultiSelect, remove picker, browse screen, status view) support `/` to activate a filter. huh's MultiSelect and list components support filtering natively — this just needs to be enabled and documented in the help bar.

### [A14] `--verbose`/`--quiet`/`--dry-run` in TUI context (Important, Spec #10; Minor, Architecture #13)

**Resolution:**
- `--verbose` and `--quiet` prevent TUI launch — fall back to `cmd.Help()`. These flags are for scripting, not interactive use.
- `--dry-run` is respected by the TUI. Operations show "[DRY RUN]" prefix in the header and results show "Would deploy..." instead of "Deployed...". This is useful for exploring what a profile switch would do.

### [A15] PersistentPreRunE interaction with TUI launch (Minor, Architecture #12)

**Problem:** Cobra's PersistentPreRunE runs before RunE and may fail (e.g., `--scope project` with no project root), preventing the TUI from launching gracefully.

**Resolution:** For bare `nd` invocation (no args, no scope flag explicitly set), the PersistentPreRunE skips scope validation. The TUI handles scope selection interactively via smart defaults. Only when `--scope` is explicitly passed does strict validation apply.

### [A16] State file locking and concurrent access (Minor, Spec #11/#17)

**Resolution:** If a lock acquisition fails (another nd process holds it), the TUI shows a non-blocking warning banner: "State locked by another process. Read-only mode." Operations that need the lock show an error: "Cannot deploy — state file locked by another process. Close other nd instances and try again."

### [A17] Terminal resize and minimum dimensions (Minor, UX #17/#21)

**Resolution:** Minimum terminal size: 60 columns x 15 rows. The root model handles `tea.WindowSizeMsg` and stores current dimensions. If below minimum, show a centered message: "Terminal too small (need 60x15)". The header and help bar adapt to width — the header truncates the profile name and the status view wraps to fewer columns.

### [A18] Accessibility: NO_COLOR and text-only indicators (Minor, UX #18)

**Resolution:** All status indicators use text as the primary differentiator, with color as enhancement:
- `OK` (green) / `!!` (red) / `??` (yellow) / `--` (dim)

When `NO_COLOR` is set or `--no-color` is passed, colors are stripped but the text markers remain readable. This is already the pattern in the status view mockup.

### [A19] Doctor flow uses inline `[Y/n]` instead of huh Confirm (Minor, UX #14)

**Resolution:** Change to huh Confirm for consistency with all other confirmation flows.

### [A20] "All types" in deploy type picker undefined (Minor, UX #20)

**Resolution:** "All types" shows a MultiSelect with assets grouped by type using section headers. huh MultiSelect doesn't support section headers natively, so use a styled separator string between groups:

```
  ── Skills ──────────────
  [ ] greeting         (local-skills)
  [ ] debugging        (local-skills)
  ── Commands ───────────
  [ ] hello.md         (local-skills)
```

### [A21] Partial failure "Retry failed" flow undefined (Minor, UX #21)

**Resolution:** "Retry failed" re-enters the progress screen with only the failed items. No re-confirmation — the user already confirmed the full set. Progress shows "Retrying 2 failed asset(s)..." with the same progress bar.

### [A22] Settings "Edit config" behavior when $EDITOR returns (Minor, UX #15)

**Resolution:** The TUI suspends (exits alt screen), opens `$EDITOR`, waits for it to close, then re-enters alt screen. After returning, the TUI re-validates the config. If invalid, shows a warning banner: "Config has validation errors: [details]. Some operations may fail."

### [A23] Settings "Show config path" / "Show version" UX (Minor, UX #16)

**Resolution:** These are inline displays, not full screens. Selecting them shows the value on the same screen momentarily (like a toast), then returns focus to the Settings menu. No screen push needed.

### [A24] Back menu item redundant with esc (Minor, UX #19)

**Resolution:** Keep "Back" as a menu item. It serves as discoverability for users who don't know about esc. The implementation is identical (both emit BackMsg). Low cost, improves accessibility.

### [A25] Testing strategy gaps for integration (Important, Architecture #10)

**Resolution:** Add integration tests for the root model that exercise multi-screen flows:

```go
func TestDeployFlowNavigation(t *testing.T) {
    m := newTestModel(mockServices)
    // Simulate: menu -> deploy -> pick type -> select assets -> confirm -> result -> back to menu
    m, _ = m.Update(selectMenuItemMsg("Deploy assets"))
    assert(m.currentScreen().Title() == "Deploy")
    m, _ = m.Update(selectMenuItemMsg("Skills"))
    // ... exercise full flow including BackMsg, PopToRootMsg
}
```

These tests validate stack management, global key routing, and screen transitions — the integration-level concerns that per-screen unit tests miss.

### Resolved Open Questions

| # | Question | Resolution |
|---|----------|------------|
| 1 | Loading state on launch? | Yes. Show a single-frame spinner ("Detecting environment...") if agent detection takes >100ms. |
| 2 | Mouse interaction? | Deferred to v2. Keyboard-only for initial implementation. |
| 3 | Help overlay style? | Full-screen help page. Simpler to implement, pushed as a screen on the stack. |
| 4 | `--dry-run` in TUI? | Yes. Respected via header indicator and "Would..." result text. See [A14]. |
| 5 | Auto-refresh header? | Yes. Required by FR-018 ("always displays"). The header re-queries after each operation completes. See [A11] perf note — acceptable for typical deployment counts. |

### Updated File List

```
internal/tui/
  services.go       # Services interface (resolves circular import)
  tui.go            # Root model, Run(), screen stack, global key routing
  header.go         # Persistent header component
  helpbar.go        # Context-sensitive help bar
  screens.go        # Screen interface, navigation messages
  main_menu.go      # Main menu (huh Select, grouped with separators)
  deploy.go         # Deploy flow (search-all + type-first + multi-select + confirm)
  remove.go         # Remove flow
  browse.go         # Browse all assets (nd list equivalent)
  status.go         # Deployment status view
  doctor.go         # Doctor + fix flow
  profile.go        # Profile submenu (switch, create, list, deploy)
  snapshot.go       # Snapshot submenu (save, restore, list)
  source.go         # Source submenu (list, add local/git, remove, sync)
  pin.go            # Pin/unpin flow
  init.go           # First-run guided setup
  export.go         # Export plugin flow
  settings.go       # Settings submenu (edit config, show info)
  progress.go       # Reusable progress bar component
  empty.go          # Empty state messages and hints
  theme.go          # Palette, Styles, glyphs, huh theme — all in one file
```

21 files. The theme is intentionally a single file — 6 colors, 6 styles, 6 glyphs doesn't warrant a subpackage.
