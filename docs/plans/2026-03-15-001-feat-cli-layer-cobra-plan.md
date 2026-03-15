---
title: "feat: CLI Layer (Cobra) — Full Command Tree"
type: feat
status: active
date: 2026-03-15
origin: docs/brainstorms/2026-03-15-cli-layer-brainstorm.md
---

## Overview

Wire all 5 completed service layers into user-facing commands via Cobra. This delivers nd as a usable CLI tool with 23 commands, 8 global flags, output formatting (human/JSON/quiet), and both unit and integration tests. All business logic stays in `internal/`; the CLI is a thin orchestration layer translating user input into service calls.

## Problem Statement

nd has complete service layers (Source Manager, Deploy Engine, Agent Registry, Profile/Snapshot Store, State Store) but no user interface. `main.go` is a stub. Users cannot interact with the tool. This plan delivers the CLI layer that makes nd functional.

## Proposed Solution

Flat `cmd/` package with one file per command group, an `App` struct for lazy service wiring, and helper functions for output formatting. All commands ship in a single cycle to ensure consistent patterns.

(see brainstorm: `docs/brainstorms/2026-03-15-cli-layer-brainstorm.md`)

## Technical Approach

### Architecture

```text
main.go
  └─ cmd.Execute()
       └─ rootCmd (global flags, PersistentPreRunE)
            ├─ App struct (lazy service init)
            ├─ helpers (printJSON/printHuman/confirm)
            └─ subcommands (deploy, remove, source, profile, ...)
                 └─ internal/* service calls
```

### Complete Command Tree

| Command | File | Service calls | FR |
|---------|------|--------------|-----|
| `nd` (no args) | `root.go` | Print help, exit 0 (TUI deferred) | FR-001 |
| `nd init` | `init_cmd.go` | Interactive walkthrough -> write config | FR-025 |
| `nd deploy <asset> [assets...]` | `deploy.go` | `sm.Scan()` -> index search -> `eng.Deploy()`/`DeployBulk()` | FR-009, FR-010 |
| `nd remove <asset> [assets...]` | `remove.go` | state lookup -> `eng.Remove()`/`RemoveBulk()` | FR-012 |
| `nd list` | `list.go` | `sm.Scan()` -> index query with filters | FR-016d |
| `nd status` | `status.go` | `eng.Status()`, `profMgr.ActiveProfile()` | FR-015 |
| `nd sync` | `sync.go` | `eng.Sync()`, optionally `sm.SyncSource()` | FR-013, FR-014, FR-027 |
| `nd pin <asset>` | `pin.go` | state update: set origin to `pinned` | FR-024a |
| `nd unpin <asset>` | `pin.go` | state update: set origin to `manual` | FR-024a |
| `nd source add <path\|url>` | `source.go` | `sm.AddLocal()` or `sm.AddGit()` | FR-005, FR-006 |
| `nd source remove <id>` | `source.go` | `sm.Remove()` (with 3-way choice) | FR-043 |
| `nd source list` | `source.go` | `sm.Sources()`, per-source scan for counts | FR-046 |
| `nd profile create <name>` | `profile.go` | `pstore.CreateProfile()` | FR-022 |
| `nd profile delete <name>` | `profile.go` | `profMgr.DeleteProfile()` | — |
| `nd profile list` | `profile.go` | `pstore.ListProfiles()` | — |
| `nd profile deploy <name>` | `profile.go` | `sm.Scan()` -> `profMgr.DeployProfile()` | FR-024 |
| `nd profile switch <name>` | `profile.go` | `sm.Scan()` -> `profMgr.Switch()` | FR-023 |
| `nd profile add-asset <profile> <asset>` | `profile.go` | `pstore.GetProfile()` -> update -> save | — |
| `nd snapshot save <name>` | `snapshot.go` | `sstore.Load()` -> `pstore.SaveSnapshot()` | FR-020 |
| `nd snapshot restore <name>` | `snapshot.go` | `sm.Scan()` -> `profMgr.Restore()` | FR-021 |
| `nd snapshot list` | `snapshot.go` | `pstore.ListSnapshots()` | — |
| `nd snapshot delete <name>` | `snapshot.go` | `pstore.DeleteSnapshot()` | — |
| `nd settings edit` | `settings.go` | `$EDITOR` on config path | FR-026 |
| `nd doctor` | `doctor.go` | Multi-check orchestration | FR-045 |
| `nd version` | `version.go` | Build-time ldflags | FR-044 |
| `nd uninstall` | `uninstall.go` | Remove all symlinks, optionally dirs | FR-036a |

### Global Flags

| Flag | Short | Type | Default | Mutual exclusion |
|------|-------|------|---------|-----------------|
| `--scope` | `-s` | `global\|project` | `global` | — (ignored by: version, doctor, init, settings, uninstall) |
| `--dry-run` | — | bool | `false` | — |
| `--verbose` | `-v` | bool | `false` | `--quiet` |
| `--quiet` | `-q` | bool | `false` | `--verbose` |
| `--json` | — | bool | `false` | — |
| `--no-color` | — | bool | `false` | — |
| `--config` | — | string | `~/.config/nd/config.yaml` | — |
| `--yes` | `-y` | bool | `false` | — |

### Implementation Phases

#### Phase 1: Foundation (no command deps)

New packages and utilities that commands depend on. All tasks in this phase are independent and can be parallelized.

**Task 1.1: `internal/version/version.go` + test**

Version info package with ldflags injection.

```go
// internal/version/version.go
package version

var (
    Version = "dev"
    Commit  = "none"
    Date    = "unknown"
)

func String() string { ... }
```

- Acceptance: `version.String()` returns formatted string
- Test: `internal/version/version_test.go`

**Task 1.2: `internal/nd/project.go` + test**

Project root discovery: walk up from cwd looking for `.git/` or `.claude/`.

```go
func FindProjectRoot(startDir string) (string, error)
```

- Walks parent directories from `startDir`
- Returns first directory containing `.git/` or `.claude/`
- Returns error if neither found (reaches filesystem root)
- Test: `internal/nd/project_test.go` — test with temp dirs containing `.git/`, `.claude/`, neither

**Task 1.3: `internal/asset/search.go` + test**

Name-based search methods on `Index` for CLI asset resolution.

```go
func (idx *Index) SearchByName(name string) []*Asset
func (idx *Index) SearchByTypeAndName(assetType nd.AssetType, name string) *Asset
```

- `SearchByName`: returns all assets matching name (case-insensitive) across all types
- `SearchByTypeAndName`: exact type+name lookup (no SourceID required)
- Note: `nd list --pattern` will use inline `strings.Contains` filtering instead of a dedicated search method
- Test: `internal/asset/search_test.go`

**Task 1.4: `internal/deploy/deploy.go` — add `Engine.SetOrigin()` method + test**

Add `SetOrigin()` to the deploy engine for updating deploy origin within a lock.

```go
func (e *Engine) SetOrigin(identity asset.Identity, scope nd.Scope, projectRoot string, origin nd.DeployOrigin) error
```

- Updates the origin field for a deployed asset within a lock
- Used by `cmd/pin.go` for `nd pin` / `nd unpin` instead of directly mutating state
- Test: update existing engine tests

**Task 1.5: `output.JSONResponse` update + `DeploymentsToEntries` helper**

Add `DryRun` field to the JSON envelope.

```go
type JSONResponse struct {
    Status string      `json:"status"`
    DryRun bool        `json:"dry_run,omitempty"`
    Data   interface{} `json:"data,omitempty"`
    Errors []JSONError `json:"errors,omitempty"`
}
```

Also extract `DeploymentsToEntries([]state.Deployment) []profile.SnapshotEntry` as a public function in `internal/profile/store.go`. This is needed by `cmd/snapshot.go` for `nd snapshot save` to convert current state deployments into snapshot entries.

- Test: update existing `json_test.go`; add test for `DeploymentsToEntries` in `internal/profile/store_test.go`

#### Phase 2: CLI Scaffold (depends on Phase 1)

The core `cmd/` package structure. Tasks 2.1-2.3 must be sequential (root depends on app, main depends on root).

**Task 2.1: `cmd/app.go` + test**

App struct with lazy service initialization.

```go
type App struct {
    ConfigPath  string
    Scope       nd.Scope
    ProjectRoot string
    BackupDir   string // derived as ~/.config/nd/backups/
    Verbose     bool
    Quiet       bool
    JSON        bool
    DryRun      bool
    NoColor     bool
    Yes         bool

    // lazily initialized
    sm     *sourcemanager.SourceManager
    reg    *agent.Registry
    eng    *deploy.Engine
    profMgr *profile.Manager
    pstore  *profile.Store
    sstore  *state.Store
}

func (a *App) SourceManager() (*sourcemanager.SourceManager, error)
func (a *App) AgentRegistry() (*agent.Registry, error)
func (a *App) DefaultAgent() (*agent.Agent, error)
func (a *App) DeployEngine() (*deploy.Engine, error)
func (a *App) ProfileManager() (*profile.Manager, error)
func (a *App) ProfileStore() (*profile.Store, error)
func (a *App) StateStore() *state.Store
func (a *App) ResolveProjectRoot() (string, error)
func (a *App) ScanIndex() (*sourcemanager.ScanSummary, error)
```

- Each accessor creates and caches the service on first call
- `DeployEngine()` internally calls `StateStore()` and `DefaultAgent()`; passes `a.BackupDir` to `deploy.New()`
- `ProfileManager()` internally calls `ProfileStore()` and `StateStore()`
- `AgentRegistry()` internally calls `SourceManager()` to get config via `sm.Config()`, then passes it to `agent.New()`
- `ScanIndex()` calls `SourceManager()` then `sm.Scan()`; returns `(*sourcemanager.ScanSummary, error)` — commands access `.Index`, `.Warnings`, and `.Errors` from the summary
- `ResolveProjectRoot()` uses `nd.FindProjectRoot(cwd)` when scope is project
- Test: `cmd/app_test.go` — verify lazy init, verify caching, verify error propagation

**Task 2.2: `cmd/helpers.go` + test**

Output formatting and confirmation helper functions (no struct).

```go
func printJSON(w io.Writer, data interface{}, dryRun bool) error
func confirm(r io.Reader, w io.Writer, prompt string, yesFlag bool) (bool, error)
func promptChoice(r io.Reader, w io.Writer, prompt string, choices []string) (string, error)
func printHuman(w io.Writer, format string, args ...interface{})
```

- `printJSON`: marshals `JSONResponse` with data and optional dry-run flag to writer
- `confirm`: if `yesFlag`, return true; detects TTY via `golang.org/x/term.IsTerminal(int(os.Stdin.Fd()))`; if not TTY, return error; else prompt y/n on reader/writer
- `promptChoice`: presents numbered choices, reads selection from reader; used by `nd source remove` for 3-way remove/orphan/cancel
- `printHuman`: formatted print to writer; callers add `[dry-run]` prefix when `app.DryRun` is true
- Reader/writer params enable testing without real stdin/stdout
- Test: `cmd/helpers_test.go`

**Task 2.3: `cmd/root.go`**

Root command with global flags and PersistentPreRunE.

```go
func NewRootCmd(app *App) *cobra.Command
func Execute()
```

- Defines all global persistent flags
- `MarkFlagsMutuallyExclusive("verbose", "quiet")`
- `PersistentPreRunE`: reads flags into `App` fields, resolves config path (tilde expansion)
- `RunE` (no subcommand): prints help text and exits 0 (TUI launch deferred to future phase)
- `SilenceUsage = true`, `SilenceErrors = true`
- Registers all subcommands
- Registers Cobra's built-in `completion` command for bash/zsh/fish shell completions (FR-035, Could Have)
- Test: `cmd/root_test.go` — verify flag parsing, mutual exclusion, help output

**Task 2.4: Update `main.go`**

Replace stub with Cobra entry point.

```go
package main

import (
    "os"
    "github.com/larah/nd/cmd"
)

func main() {
    code := cmd.Execute()
    os.Exit(code)
}
```

- `Execute()` returns exit code, `main()` calls `os.Exit()`
- Never call `os.Exit()` inside commands

**Task 2.5: Update `go.mod`**

Add Cobra dependency.

```text
require (
    github.com/spf13/cobra v1.9.1
    golang.org/x/term v0.28.0
)
```

#### Phase 3: Source Management Commands (depends on Phase 2)

**Task 3.1: `cmd/source.go` + test**

`nd source add`, `nd source remove`, `nd source list`.

- `add`: detect local vs git (URL detection), call `sm.AddLocal()` or `sm.AddGit()`
- `remove`: look up source, warn about deployed assets (FR-043), 3-way choice via `promptChoice()`:
  - **remove**: remove source + all deployed symlinks + state entries
  - **orphan**: remove source from config only; leave existing symlinks and state entries in place (symlinks become "foreign" — still tracked in state but source no longer registered)
  - **cancel**: abort
  - Call `sm.Remove()` for the source config; for "remove" mode, also call `eng.RemoveBulk()` on all deployments from that source
- `list`: call `sm.Sources()`, for each source call `ScanSource()` for asset count, format table
- Flags: `--alias` on add, `--force` on remove
- Test: `cmd/source_test.go`

#### Phase 4: Core Deploy/Remove/Status/List (depends on Phase 2)

All tasks in this phase are independent.

**Task 4.1: `cmd/deploy.go` + test**

`nd deploy <asset> [assets...]`

- Parse asset references (name, type/name, source:type/name)
- Scan sources, resolve assets via `Index.SearchByName()` / `SearchByTypeAndName()`
- After scanning, check `index.Conflicts()` and print warnings for any duplicate assets across sources
- If ambiguous, print candidates and exit with code 3
- Single asset: `eng.Deploy()`; multiple: `eng.DeployBulk()`
- Handle `--dry-run`: report what would happen without executing
- Print post-deploy reminders for hooks/output-styles (FR-016b settings.json)
- Origin: `nd.OriginManual` by default
- Exit code 2 for partial bulk failures
- Flags: `--type`
- Test: `cmd/deploy_test.go`

**Task 4.2: `cmd/remove.go` + test**

`nd remove <asset> [assets...]`

- Look up deployed assets in state
- If pinned: warn and require confirmation (FR-024a)
- Single: `eng.Remove()`; multiple: `eng.RemoveBulk()`
- Handle `--dry-run`
- Exit code 2 for partial bulk failures
- Test: `cmd/remove_test.go`

**Task 4.3: `cmd/list.go` + test**

`nd list [--type TYPE] [--source SOURCE] [--pattern PATTERN]`

- Scan sources, build index
- After scanning, check `index.Conflicts()` and print warnings for any duplicate assets across sources
- Filter by type, source, and/or name pattern
- Cross-reference with deployment state to show deploy status
- Format: table with name, type, source, status (deployed/available/broken)
- Support `--json` output
- Test: `cmd/list_test.go`

**Task 4.4: `cmd/status.go` + test**

`nd status`

- Call `eng.Status()` for all deployments with health
- Call `profMgr.ActiveProfile()` for active profile
- Group by asset type
- Show scope, origin, health status
- Filter by current project when `--scope project`
- Support `--json` output
- Test: `cmd/status_test.go`

**Task 4.5: `cmd/pin.go` + test**

`nd pin <asset>`, `nd unpin <asset>`

- Look up deployed asset in state
- Call `eng.SetOrigin()` to update origin field: `pinned` or `manual` (uses engine lock internally)
- Test: `cmd/pin_test.go`

#### Phase 5: Health & Sync (depends on Phase 2)

**Task 5.1: `cmd/sync.go` + test**

`nd sync [--source SOURCE]`

- If `--source` specified: call `sm.SyncSource()` (git pull) then `eng.Sync()`
- Otherwise: `eng.Sync()` only (repair symlinks)
- Report repaired, removed, warnings
- Handle `--dry-run`
- Test: `cmd/sync_test.go`

**Task 5.2: `cmd/doctor.go` + test**

`nd doctor`

- Orchestration logic is inline in the command (no separate `internal/doctor/run.go`):
  - Validates config, checks sources, checks deployments, checks agents, checks git
  - Populates `doctor.Report` struct (defined in `internal/doctor/report.go`)
- Format report with pass/warn/fail indicators
- Exit code 1 if any failures, 0 if all pass/warn
- Support `--json` output
- Test: `cmd/doctor_test.go`

#### Phase 6: Profile & Snapshot Commands (depends on Phase 2)

**Task 6.1: `cmd/profile.go` + test**

`nd profile create`, `delete`, `list`, `deploy`, `switch`, `add-asset`.

- `create <name>`: optional `--assets type/name,type/name` and `--from-current`
  - `--assets`: parse refs, resolve via index, build `Profile` struct
  - `--from-current`: read deployment state, convert to profile assets
  - `--description` flag for profile description
  - Note: `CreateProfile` takes a full `Profile` struct that the command must assemble (Version, timestamps, Assets)
- `delete <name>`: call `profMgr.DeleteProfile()`, confirm if has deployed assets
- `list`: call `pstore.ListProfiles()`, format table (name, description, asset count, active indicator)
- `deploy <name>`: scan index, call `profMgr.DeployProfile()`, report results
- `switch <name>`: must call `profMgr.ActiveProfile()` first to get current profile name; if no profile is active, show error: "No active profile. Use `nd profile deploy <name>` instead." Then scan index, call `profMgr.Switch()`, show diff summary, require confirmation (skippable with `--yes`)
- `add-asset <profile> <asset>`: resolve asset, add to existing profile, save
- Test: `cmd/profile_test.go`

**Task 6.2: `cmd/snapshot.go` + test**

`nd snapshot save`, `restore`, `list`, `delete`.

- `save <name>`: load state, call `pstore.SaveSnapshot()`
- `restore <name>`: scan index, call `profMgr.Restore()`, require confirmation
- `list`: call `pstore.ListSnapshots()`, format table (name, auto, deployment count, date)
- `delete <name>`: uses try-user-then-auto fallback (try `pstore.DeleteSnapshot(name, false)`, if not found try `pstore.DeleteSnapshot(name, true)`), require confirmation
- Test: `cmd/snapshot_test.go`

#### Phase 7: Settings, Init, Version, Utilities (depends on Phase 2)

All tasks are independent.

**Task 7.1: `cmd/version.go` + test**

`nd version`

- Print `nd version <Version> (commit: <Commit>, built: <Date>)`
- Also register with Cobra's `cmd.Version` for `--version` flag
- Support `--json` output
- Test: `cmd/version_test.go`

**Task 7.2: `cmd/init_cmd.go` + test**

`nd init`

- Interactive walkthrough: detect agents, ask scope, ask for first source
- Create `~/.config/nd/config.yaml` with defaults
- Create directory structure (`profiles/`, `snapshots/user/`, `snapshots/auto/`, `state/`, `sources/`)
- Respect `--yes` for non-interactive defaults
- Test: `cmd/init_cmd_test.go`

**Task 7.3: `cmd/settings.go` + test**

`nd settings edit`

- Open `$EDITOR` (fallback `$VISUAL`, fallback `vi`) on config path
- Check config exists first; if not, suggest `nd init`
- Test: `cmd/settings_test.go`

**Task 7.4: `cmd/uninstall.go` + test**

`nd uninstall`

- Load state via `app.StateStore().Load()`, show all deployments as an uninstall plan
- Show summary, require confirmation (respect `--yes`)
- If confirmed: call `eng.RemoveBulk()` on all deployments, optionally remove `~/.config/nd/`
- Support `--dry-run` (show plan only — load state, display symlinks)
- The existing `UninstallPlan` type in `internal/deploy/uninstall.go` is used as a data structure only
- Test: `cmd/uninstall_test.go`

#### Phase 8: Integration Tests (depends on all phases)

**Task 8.1: `tests/integration/helpers_test.go`**

Test harness: build binary, `runND()` helper, temp dir setup/teardown.

```go
func buildBinary(t *testing.T) string  // builds nd binary, returns path
func runND(t *testing.T, bin string, args ...string) (stdout, stderr string, exitCode int)
func setupTestSource(t *testing.T) string  // creates a source dir with test assets
```

**Task 8.2: `tests/integration/source_test.go`**

- Test US-005: add source, list sources, verify asset discovery
- Test source remove with deployed assets

**Task 8.3: `tests/integration/deploy_test.go`**

- Test US-001: deploy skills to project, verify symlinks
- Test bulk deploy with partial failures (exit code 2)
- Test deploy with conflict handling

**Task 8.4: `tests/integration/status_sync_test.go`**

- Test US-002: break symlinks externally, run sync, verify repair
- Test status output reflects health

**Task 8.5: `tests/integration/profile_test.go`**

- Test US-003: create profile, deploy, switch, verify pin persistence
- Test snapshot save/restore round-trip

**Task 8.6: `tests/integration/flags_test.go`**

- Test `--json` output envelope structure
- Test `--dry-run` outputs without side effects
- Test `--scope project` with project root detection
- Test `--yes` bypasses confirmation

### Dry-Run Implementation

- `nd deploy --dry-run`: resolve assets and compute deploy paths via `agent.DeployPath()`, show plan without calling engine
- `nd remove --dry-run`: look up deployments in state, show what would be removed
- `nd sync --dry-run`: use `eng.Check()` (returns issues that Sync would fix)
- `nd uninstall --dry-run`: load state, show all symlinks
- No new engine methods needed; CLI computes the preview using existing APIs

## System-Wide Impact

### Interaction Graph

CLI commands -> App struct -> service layers -> filesystem/state. No callbacks, no observers. Linear call chains.

### Error Propagation

Service layer errors (typed: `LockError`, `ConflictError`, `PathTraversalError`) propagate to commands. Commands map them:

| Error type | Exit code | User message |
|-----------|-----------|-------------|
| `LockError` | 1 | "Another nd process is running. Try again." |
| `ConflictError` | 1 | "Conflict at {path}: {detail}" |
| `PathTraversalError` | 1 | "Security: path escapes source root" |
| Bulk partial failure | 2 | "Deployed N/M assets. K failed: {details}" |
| Invalid args/flags | 3 | Cobra auto-generates usage |
| All other errors | 1 | Error message as-is |

### State Lifecycle Risks

All state mutations pass through `store.WithLock()`. CLI never modifies state directly. No partial-failure state corruption risk — the engine handles transactional writes.

### API Surface Parity

CLI covers all service layer operations. TUI (future) will call the same service layer methods, not CLI commands.

## Acceptance Criteria

### Functional Requirements

- [ ] All 23 commands from the command tree are implemented and respond to `--help`
- [ ] All 8 global flags work as documented with mutual exclusion enforced
- [ ] `nd deploy my-skill` resolves assets by name with disambiguation
- [ ] `nd deploy my-skill --dry-run` shows what would happen without side effects
- [ ] `nd status --json` produces valid JSON matching `output.JSONResponse` envelope
- [ ] `nd source add ~/my-skills && nd deploy skill-name` works end-to-end (US-001 level 0)
- [ ] `nd profile create go-dev --from-current` captures current deployments
- [ ] `nd profile switch` shows diff and requires confirmation (bypass with `--yes`)
- [ ] Exit codes: 0 (success), 1 (error), 2 (partial), 3 (invalid usage)
- [ ] `nd version` completes in <100ms (no service layer init)
- [ ] `nd doctor` validates config, sources, deployments, agents, git

### Non-Functional Requirements

- [ ] Unit test coverage >80% on `cmd/` package
- [ ] Integration tests cover US-001 through US-005
- [ ] No `os.Exit()` calls inside commands — only in `main()`
- [ ] `--json` output goes to stdout; all warnings/errors to stderr
- [ ] Cobra and `golang.org/x/term` are the only new external dependencies added

### Quality Gates

- [ ] `go vet ./...` passes
- [ ] `go test ./...` passes
- [ ] `rumdl check` passes on any new `.md` files

## Dependencies and Prerequisites

- All 5 service layers must be complete and passing (confirmed: PR #1-#4 merged)
- Go 1.25.1+ (confirmed in `go.mod`)
- Cobra v1.9.x (to be added)
- `golang.org/x/term` (to be added, for TTY detection in `confirm()`)

## Risk Analysis and Mitigation

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|-----------|
| Asset resolution ambiguity confuses users | Medium | Medium | Clear error messages listing candidates; require `type/name` syntax for disambiguation |
| Service layer API gaps discovered during wiring | Low | Medium | Phase 1 fills known gaps; any new gaps addressed as they surface |
| Integration test flakiness from filesystem timing | Low | Low | Use `t.TempDir()`, avoid parallel tests touching same state files |
| Cobra flag state leakage between tests | Medium | Low | Fresh command instance per test case (no shared rootCmd) |

## File Inventory

### New files (48)

| File | Phase | Purpose |
|------|-------|---------|
| `internal/version/version.go` | 1 | Version info with ldflags |
| `internal/version/version_test.go` | 1 | Tests |
| `internal/nd/project.go` | 1 | FindProjectRoot |
| `internal/nd/project_test.go` | 1 | Tests |
| `internal/asset/search.go` | 1 | Name-based search on Index |
| `internal/asset/search_test.go` | 1 | Tests |
| `cmd/app.go` | 2 | App struct, lazy init |
| `cmd/app_test.go` | 2 | Tests |
| `cmd/helpers.go` | 2 | Output formatting and confirmation helpers |
| `cmd/helpers_test.go` | 2 | Tests |
| `cmd/root.go` | 2 | Root command, global flags |
| `cmd/root_test.go` | 2 | Tests |
| `cmd/source.go` | 3 | Source add/remove/list |
| `cmd/source_test.go` | 3 | Tests |
| `cmd/deploy.go` | 4 | Deploy command |
| `cmd/deploy_test.go` | 4 | Tests |
| `cmd/remove.go` | 4 | Remove command |
| `cmd/remove_test.go` | 4 | Tests |
| `cmd/list.go` | 4 | List command |
| `cmd/list_test.go` | 4 | Tests |
| `cmd/status.go` | 4 | Status command |
| `cmd/status_test.go` | 4 | Tests |
| `cmd/pin.go` | 4 | Pin/unpin commands |
| `cmd/pin_test.go` | 4 | Tests |
| `cmd/sync.go` | 5 | Sync command |
| `cmd/sync_test.go` | 5 | Tests |
| `internal/doctor/report.go` | 5 | Doctor Report struct |
| `internal/doctor/report_test.go` | 5 | Tests |
| `cmd/doctor.go` | 5 | Doctor command |
| `cmd/doctor_test.go` | 5 | Tests |
| `cmd/profile.go` | 6 | Profile commands |
| `cmd/profile_test.go` | 6 | Tests |
| `cmd/snapshot.go` | 6 | Snapshot commands |
| `cmd/snapshot_test.go` | 6 | Tests |
| `cmd/version.go` | 7 | Version command |
| `cmd/version_test.go` | 7 | Tests |
| `cmd/init_cmd.go` | 7 | Init command |
| `cmd/init_cmd_test.go` | 7 | Tests |
| `cmd/settings.go` | 7 | Settings edit |
| `cmd/settings_test.go` | 7 | Tests |
| `cmd/uninstall.go` | 7 | Uninstall command |
| `cmd/uninstall_test.go` | 7 | Tests |
| `tests/integration/helpers_test.go` | 8 | Test harness |
| `tests/integration/source_test.go` | 8 | Source integration tests |
| `tests/integration/deploy_test.go` | 8 | Deploy integration tests |
| `tests/integration/status_sync_test.go` | 8 | Status/sync integration tests |
| `tests/integration/profile_test.go` | 8 | Profile integration tests |
| `tests/integration/flags_test.go` | 8 | Flag integration tests |

### Modified files (5)

| File | Phase | Change |
|------|-------|--------|
| `main.go` | 2 | Replace stub with `cmd.Execute()` |
| `go.mod` | 2 | Add Cobra and `golang.org/x/term` dependencies |
| `internal/output/json.go` | 1 | Add `DryRun` field |
| `internal/deploy/deploy.go` | 1 | Add `Engine.SetOrigin()` method |
| `internal/profile/store.go` | 1 | Extract `DeploymentsToEntries()` public function |

## Sources and References

### Origin

- **Brainstorm document:** [docs/brainstorms/2026-03-15-cli-layer-brainstorm.md](../brainstorms/2026-03-15-cli-layer-brainstorm.md) — Key decisions: flat cmd/ layout, App struct with lazy init, all commands in one phase, same-format dry-run with `[dry-run]` prefix, inline `--assets` flag for profile create.

### Internal References

- Spec CLI command reference: `docs/specs/nd-go-spec.md:425-475`
- Spec error behavior: `docs/specs/nd-go-spec.md:570-607`
- Exit codes: `internal/nd/exit_codes.go:1`
- JSON envelope: `internal/output/json.go:1`
- Deploy engine API: `internal/deploy/deploy.go:156-483`
- Profile manager API: `internal/profile/manager.go:45-327`
- Source manager API: `internal/sourcemanager/sourcemanager.go:25-116`
- Agent registry API: `internal/agent/registry.go`

### External References

- Cobra CLI best practices: `.claude/docs/reference/cobra-cli-best-practices.md`
- Go project layout research: `.claude/docs/reference/go-cli-project-layout-research.md`
- SpecFlow analysis: `.claude/docs/reports/2026-03-15-cli-layer-flow-analysis.md`
