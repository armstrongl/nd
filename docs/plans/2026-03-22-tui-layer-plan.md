# TUI Layer Implementation Plan

> **For agentic workers:** REQUIRED: Use supapowers:subagent-driven-development (if subagents available) or supapowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the TUI layer for nd, providing menu-driven interactive access to all core operations via bare `nd` invocation.

**Design doc:** `docs/plans/2026-03-22-tui-layer-design.md`
**Brainstorm:** `docs/brainstorms/2026-03-22-tui-layer-brainstorm.md`

**Tech Stack:** Bubble Tea v2, huh v2, Lip Gloss v2, Bubbles v2 (all `charm.land/*`)

**Existing types used:**
- `cmd/app.go` — App struct (satisfies Services interface)
- `cmd/root.go` — RunE entry point modification
- `cmd/export.go` — huh v1 usage to migrate
- `internal/deploy/` — Engine, Request, Result, BulkResult
- `internal/state/` — Store, HealthStatus, HealthCheck, Deployment
- `internal/profile/` — Manager, Store, ProfileSummary, SnapshotSummary
- `internal/sourcemanager/` — SourceManager, ScanSummary
- `internal/agent/` — Registry, Agent
- `internal/asset/` — Index, Asset, Identity
- `internal/nd/` — Scope, AssetType, DeployOrigin

---

## Phase 0: Dependency migration (prerequisite)

Migrate from Charm v1 to v2 ecosystem. This must happen first because huh v1 and huh v2 use incompatible Bubble Tea types.

### Task 0.1: Migrate huh v1 to huh v2

**Files:**
- Modify: `go.mod`, `go.sum`
- Modify: `cmd/export.go`
- Modify: `cmd/export_test.go`

- [ ] **Step 1: Update go.mod**
  Replace `github.com/charmbracelet/huh v1.0.0` with `charm.land/huh/v2`. Add `charm.land/bubbletea/v2` and `charm.land/lipgloss/v2` as direct dependencies. Run `go mod tidy`.

- [ ] **Step 2: Update imports in cmd/export.go**
  Change `github.com/charmbracelet/huh` to `charm.land/huh/v2`. Check huh v2 upgrade guide for API changes (theme interface, spinner subpackage, method renames).

- [ ] **Step 3: Update cmd/export_test.go imports**
  Same import path changes.

- [ ] **Step 4: Verify tests pass**
  Run `go test ./cmd/ -run TestExport -v`. All existing export tests must pass.

- [ ] **Step 5: Verify build**
  Run `go build ./...`. Clean build with no v1 Charm imports remaining.

---

## Phase 1: Foundation (theme, services, shell)

Build the app shell that all screens plug into. At the end of this phase, `nd` launches a TUI with a header, help bar, and a placeholder main menu.

### Task 1.1: Theme and styles

**Files:**
- Create: `internal/tui/theme.go`
- Create: `internal/tui/theme_test.go`

- [ ] **Step 1: Write tests for NewStyles**
  Test that `NewStyles(true)` and `NewStyles(false)` return non-zero styles. Test glyph constants are non-empty. Test `NdTheme` returns non-nil `*huh.Styles`.

- [ ] **Step 2: Implement theme.go**
  Palette (5 AdaptiveColors), Styles struct (6 fields), NewStyles(), glyphs, NdTheme. As specified in design doc.

### Task 1.2: Services interface and mock

**Files:**
- Create: `internal/tui/services.go`
- Create: `internal/tui/services_test.go`
- Create: `internal/tui/testutil_test.go` (mockServices helper for all downstream tests)
- Modify: `cmd/app.go` (add Scope getter, DryRun getter, ResetForScope)
- Modify: `cmd/app_test.go` (test new methods)

- [ ] **Step 1: Define Services interface**
  As specified in design doc. Each method matches an existing `cmd.App` method.

- [ ] **Step 2: Create mockServices test helper**
  A `mockServices` struct satisfying `Services` with canned data (configurable profiles, assets, deployments). Used by all downstream screen tests.

- [ ] **Step 3: Add getter methods to cmd.App**
  Add `GetScope() nd.Scope`, `GetDryRun() bool`, `ResetForScope(scope, projectRoot)`. Use `GetScope`/`GetDryRun` to avoid field name collision. Update Services interface to match.

- [ ] **Step 4: Write tests for ResetForScope**
  Verify all cached services are nil after reset. Verify scope/projectRoot updated.

- [ ] **Step 5: Verify cmd.App satisfies Services**
  Add compile-time check in `services_test.go`: `var _ tui.Services = (*App)(nil)`.

### Task 1.3: Screen interface and navigation messages

**Files:**
- Create: `internal/tui/screens.go`
- Create: `internal/tui/screens_test.go`

- [ ] **Step 1: Define Screen interface, NavigateMsg, BackMsg, PopToRootMsg, RefreshHeaderMsg**
  As specified in design doc.

- [ ] **Step 2: Test that navigation messages are distinct types**
  Basic type assertion tests.

### Task 1.4: Header component

**Files:**
- Create: `internal/tui/header.go`
- Create: `internal/tui/header_test.go`

- [ ] **Step 1: Write tests for Header.View**
  Test left-aligned profile/scope/agent, right-aligned counts. Test issue count styled differently when > 0. Test width adaptation.

- [ ] **Step 2: Write tests for Header.Refresh**
  Test with mock services returning known profile, deployment counts.

- [ ] **Step 3: Implement header.go**

### Task 1.5: Help bar component

**Files:**
- Create: `internal/tui/helpbar.go`
- Create: `internal/tui/helpbar_test.go`

- [ ] **Step 1: Write tests for HelpBar.View**
  Test default items present. Test HelpProvider custom items appended.

- [ ] **Step 2: Implement helpbar.go**
  Includes `HelpProvider` interface (optional, screens implement to add custom help items) and `HelpItem` type. Both are part of the public API used by screen implementers.

### Task 1.6: Root model and Run()

**Files:**
- Create: `internal/tui/tui.go`
- Create: `internal/tui/tui_test.go`

- [ ] **Step 1: Write tests for screen stack navigation**
  Test NavigateMsg pushes, BackMsg pops, PopToRootMsg clears to root. Test BackMsg on single screen = quit.

- [ ] **Step 2: Write tests for input-aware key routing**
  Test q/esc suppressed when InputActive() is true. Test ctrl+c always works.

- [ ] **Step 3: Write tests for View composition**
  Test output contains header, content, and help bar sections.

- [ ] **Step 4: Implement tui.go**
  Model struct, Run(), Init(), Update(), View() as specified in design doc.

### Task 1.7: Wire into cmd/root.go

**Files:**
- Modify: `cmd/root.go`
- Create: `cmd/root_tui_test.go`

- [ ] **Step 1: Modify RunE**
  When `isTerminal()` and not verbose/quiet, call `tui.Run(app)`. Otherwise show help.

- [ ] **Step 2: Test non-terminal falls back to help**
  Verify piped stdin still shows help text.

---

## Phase 2: Main menu and empty states

### Task 2.1: Empty state messages

**Files:**
- Create: `internal/tui/empty.go`
- Create: `internal/tui/empty_test.go`

- [ ] **Step 1: Define empty state functions**
  `NoSources()`, `NoAssets()`, `NothingDeployed()`, `NoProfiles()`, `NoSnapshots()`, `AllDeployed(typeName)`. Each returns a styled string with an actionable hint.

- [ ] **Step 2: Test each returns non-empty string with hint text**

### Task 2.2: Main menu screen

**Files:**
- Create: `internal/tui/main_menu.go`
- Create: `internal/tui/main_menu_test.go`

- [ ] **Step 1: Write tests for menu selection**
  Test each menu item emits correct NavigateMsg or QuitMsg.

- [ ] **Step 2: Implement main_menu.go**
  huh Select with grouped options. InputActive() returns false.

- [ ] **Step 3: Manual smoke test**
  Run `nd` — verify header, menu, and help bar render. Verify q quits. Verify esc quits.

### Task 2.3: First-run init screen

**Files:**
- Create: `internal/tui/init.go`
- Create: `internal/tui/init_test.go`

- [ ] **Step 1: Write tests for init detection**
  Test that when config path doesn't exist, init screen is pushed.

- [ ] **Step 2: Write tests for init flow**
  Test source type selection, path/URL input, completion.

- [ ] **Step 3: Implement init.go**
  Guided: source type (local/git/skip) -> input -> creates config dir + minimal config.yaml. On completion, emits PopToRootMsg.

---

## Phase 3: Core operations (deploy, remove, status)

### Task 3.1: Progress component (build first — deploy and remove depend on it)

**Files:**
- Create: `internal/tui/progress.go`
- Create: `internal/tui/progress_test.go`

- [ ] **Step 1: Write tests for progress rendering**
  Test bar width adapts to terminal width. Test counter display ("8/20"). Test item name display.

- [ ] **Step 2: Implement progress.go**
  Wraps bubbles/progress. Accepts `progressMsg{completed, total, name}`. Renders: filled bar + counter + current item name.

### Task 3.2: Deploy flow

**Files:**
- Create: `internal/tui/deploy.go`
- Create: `internal/tui/deploy_test.go`

- [ ] **Step 1: Write tests for type selection step**
  Test selecting "Skills" filters assets. Test "Search all" enables filter input and InputActive() returns true. Test "All types" shows grouped list.

- [ ] **Step 2: Write tests for asset selection step**
  Test MultiSelect shows only undeployed assets. Test descriptions from _meta.yaml appear. Test `/` filter narrows results.

- [ ] **Step 3: Write tests for progress and result**
  Test progressMsg updates counter. Test deployDoneMsg shows result with paths. Test enter emits PopToRootMsg + RefreshHeaderMsg.

- [ ] **Step 4: Write tests for partial failure**
  Test failed items show asset, path, and reason (NFR-004). Test "Retry failed" re-enters progress with only failed items.

- [ ] **Step 5: Implement deploy.go**
  Multi-step screen with step enum. Per-item Deploy in tea.Cmd for progress.

### Task 3.3: Remove flow

**Files:**
- Create: `internal/tui/remove.go`
- Create: `internal/tui/remove_test.go`

- [ ] **Step 1: Write tests for remove flow**
  Test shows only deployed assets. Test confirmation uses huh Confirm with auto-snapshot mention. Test progress updates. Test result display.

- [ ] **Step 2: Implement remove.go**

### Task 3.4: Status screen

**Files:**
- Create: `internal/tui/status.go`
- Create: `internal/tui/status_test.go`

- [ ] **Step 1: Write tests for status grouping**
  Test assets grouped by type. Test health glyphs correct (`ok`/`!!`/`??`/`--`). Test issue count.

- [ ] **Step 2: Write tests for status actions**
  Test `f` emits NavigateMsg to doctor. Test `d` emits NavigateMsg to deploy. Verify these only work when InputActive() is false.

- [ ] **Step 3: Implement status.go**
  Viewport with grouped output. Custom HelpItems for f/d/r shortcuts. Implements HelpProvider interface.

---

## Phase 4: Health and browsing

### Task 4.1: Doctor screen

**Files:**
- Create: `internal/tui/doctor.go`
- Create: `internal/tui/doctor_test.go`

- [ ] **Step 1: Write tests for doctor flow**
  Test scanning shows issues. Test confirm uses huh Confirm. Test fix applies repairs.

- [ ] **Step 2: Implement doctor.go**

### Task 4.2: Browse screen

**Files:**
- Create: `internal/tui/browse.go`
- Create: `internal/tui/browse_test.go`

- [ ] **Step 1: Write tests for browse display**
  Test all assets shown with type, name, source, status. Test filter narrows results.

- [ ] **Step 2: Implement browse.go**
  Viewport-based scrollable list with / filter.

---

## Phase 5: Profiles, snapshots, sources

### Task 5.1: Profile submenu

**Files:**
- Create: `internal/tui/profile.go`
- Create: `internal/tui/profile_test.go`

- [ ] **Step 1: Write tests for profile flows**
  Test switch shows select, applies change, shows diff. Test create uses input. Test list displays profiles.

- [ ] **Step 2: Implement profile.go**

### Task 5.2: Snapshot submenu

**Files:**
- Create: `internal/tui/snapshot.go`
- Create: `internal/tui/snapshot_test.go`

- [ ] **Step 1: Write tests for snapshot flows**
  Test save uses huh Input for name, creates snapshot. Test restore shows huh Select of snapshots with timestamps and deployment counts. Test list displays all snapshots. Test back emits BackMsg.

- [ ] **Step 2: Implement snapshot.go**

### Task 5.3: Source submenu

**Files:**
- Create: `internal/tui/source.go`
- Create: `internal/tui/source_test.go`

- [ ] **Step 1: Write tests for source flows**
  Test add local with input. Test add git with input. Test remove with confirm. Test sync with progress.

- [ ] **Step 2: Implement source.go**
  InputActive() returns true during add flows.

---

## Phase 6: Remaining screens

### Task 6.1: Pin/Unpin screen

**Files:**
- Create: `internal/tui/pin.go`
- Create: `internal/tui/pin_test.go`

- [ ] **Step 1: Write tests for pin/unpin flow**
  Test shows deployed assets with current pin status. Test MultiSelect toggles pin state. Test confirmation before applying changes. Test result message shows what changed.

- [ ] **Step 2: Implement pin.go**
  Shows deployed assets with pin indicator. huh MultiSelect to toggle. Confirm changes. Calls `SetOrigin` to update pin state.

### Task 6.2: Export flow migration

**Files:**
- Create: `internal/tui/export.go`
- Create: `internal/tui/export_test.go`
- Modify: `cmd/export.go` (keep CLI path, extract shared logic)

- [ ] **Step 1: Extract interactive export logic into reusable functions**
  The huh forms in `cmd/export.go` `runExportInteractive()` should be callable from both the CLI path and the TUI screen.

- [ ] **Step 2: Verify existing export tests still pass after extraction**
  Run `go test ./cmd/ -run TestExport -v`.

- [ ] **Step 3: Write tests for TUI export screen**
  Test screen wraps shared logic. Test NavigateMsg/BackMsg navigation. Test result display.

- [ ] **Step 4: Implement TUI export screen wrapping the shared logic**

### Task 6.3: Settings submenu

**Files:**
- Create: `internal/tui/settings.go`
- Create: `internal/tui/settings_test.go`

- [ ] **Step 1: Write tests for settings flows**
  Test "Edit config" suspends TUI (emits tea.ExecProcess or equivalent). Test "Show config path" displays path inline. Test "Show version" displays version inline. Test "Switch scope/agent" calls ResetForScope and emits RefreshHeaderMsg.

- [ ] **Step 2: Implement settings.go**
  Edit config (suspend TUI, open $EDITOR, re-validate on return). Show path/version as inline text. Switch scope/agent via ResetForScope.

---

## Phase 7: Integration testing and polish

### Task 7.1: Integration tests

**Files:**
- Create: `internal/tui/integration_test.go`

- [ ] **Step 1: Test full deploy flow**
  Main menu -> Deploy -> type -> select -> progress -> result -> back to menu.

- [ ] **Step 2: Test navigation edge cases**
  Rapid esc presses. PopToRootMsg from deep stack. BackMsg from main menu.

- [ ] **Step 3: Test key routing with input screens**
  Source add flow: type q in path, verify no quit. Backspace deletes character, not navigate.

- [ ] **Step 4: Test empty states**
  Launch with no sources. Deploy with no assets of type. Status with nothing deployed.

### Task 7.2: Polish

- [ ] **Step 1: Test with NO_COLOR and --no-color**
  Verify all screens are readable without color.

- [ ] **Step 2: Test with narrow terminal (60 columns)**
  Verify header truncates, status wraps.

- [ ] **Step 3: Test --dry-run in TUI**
  Verify header shows indicator, results say "Would..."

- [ ] **Step 4: Manual UX walkthrough**
  Complete every operation from the TUI. Note friction points.

---

## Dependency graph

```
Phase 0 ─── Phase 1 ─── Phase 2 ─── Phase 3 ───┬─── Phase 5 ───┐
  (deps)      (shell)     (menu)     (core ops)  │   (profiles)  │
                                                  ├─── Phase 4   ├── Phase 7
                                                  │   (health)   │   (integration)
                                                  └─── Phase 6 ──┘
                                                      (remaining)
```

Phases 4, 5, and 6 can run in parallel after Phase 3.

## Summary

| Phase | Tasks | New files | Modified files |
| --- | --- | --- | --- |
| 0 | 1 | 0 | 4 (go.mod, go.sum, export.go, export_test.go) |
| 1 | 7 | 14 | 3 (app.go, app_test.go, root.go) |
| 2 | 3 | 6 | 0 |
| 3 | 4 | 8 | 0 |
| 4 | 2 | 4 | 0 |
| 5 | 3 | 6 | 0 |
| 6 | 3 | 6 | 1 (export.go) |
| 7 | 2 | 1 | 0 |
| **Total** | **25 tasks** | **45 files** | **8 modifications** |
