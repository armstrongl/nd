# Deploy Engine design

| Field | Value |
| --- | --- |
| **Date** | 2026-03-15 |
| **Author** | Larah |
| **Status** | Draft |
| **Packages** | `internal/state/` (Store addition), `internal/deploy/` |
| **Spec refs** | FR-009, FR-010, FR-011, FR-012, FR-013, FR-014, FR-015, FR-016b, NFR-010, NFR-011, NFR-014 |
| **Audit** | `.claude/docs/reports/deploy-engine-design-audit.md` |

## Overview

The Deploy Engine is the central orchestrator that creates and manages symlinks between source assets and agent configuration directories. It handles single and bulk deploys, removal, health checking (broken/drifted symlinks), and repair (sync). State persistence (Store) is added to the existing `internal/state/` package, co-located with `DeploymentState`, `FileLock`, and `HealthCheck` types already defined there.

This design covers core operations only: deploy, remove, status, health check, and sync (FR-009 through FR-015). Profile switch logic (FR-022 through FR-024) and auto-snapshots (FR-029a) are deferred until the Profile/Snapshot Store is designed.

## Architecture

```text
internal/state/       EXISTING package — add Store (load, save, lock) alongside existing types
internal/deploy/      NEW package — Engine: symlink lifecycle, health checks, context backups
```

### Why not a separate statestore package

The `internal/state/` package already contains `DeploymentState`, `Deployment`, `FileLock`, `HealthStatus`, `HealthCheck`, and query methods. Adding `Store` here avoids a confusing split where types live in one package and their I/O lives in another. It also lets `Store` use `FileLock` without cross-package coupling.

### Dependencies

```text
deploy.Engine
  |-- state.StateStore     (interface — load, save, lock)
  |-- agent.Agent          (deploy path computation)
  |-- asset.Asset          (discovered assets from Index)
  |-- state.Deployment     (existing data type)
  |-- state.HealthCheck    (existing health type)
  |-- nd.ConflictError     (existing error type)
  |-- nd.SchemaVersion     (existing version constant)
```

The caller (CLI/TUI) resolves the agent from the Registry and passes it to the engine:

```go
reg := agent.New(cfg)
ag, _ := reg.Default()
store := state.NewStore(statePath)
engine := deploy.New(store, ag, backupDir)
```

## state package additions

### Store struct

Added to `internal/state/store.go`:

```go
// Store manages the deployment state file on disk.
type Store struct {
    path string
    lock *FileLock
}

// NewStore creates a Store targeting the given deployments.yaml path.
func NewStore(path string) *Store
```

### StateStore interface

Defined in `internal/deploy/` for engine testability (accept interfaces, return structs):

```go
// StateStore abstracts state persistence for testing.
type StateStore interface {
    Load() (*state.DeploymentState, []string, error)
    Save(st *state.DeploymentState) error
    WithLock(fn func() error) error
}
```

`*state.Store` satisfies this interface. Engine tests use an in-memory mock.

### Load (NFR-014)

```go
func (s *Store) Load() (*state.DeploymentState, []string, error)
```

1. File missing: return `DeploymentState{Version: nd.SchemaVersion}`, no warnings
2. Parse YAML
3. **Schema version check (NFR-014)**:
   - `file.Version < nd.SchemaVersion`: migrate automatically, back up original first
   - `file.Version > nd.SchemaVersion`: refuse to load, return error explaining version mismatch
   - `file.Version == nd.SchemaVersion`: proceed normally
4. YAML parse failure: rename to `deployments.yaml.corrupt.<timestamp>` (format: `2026-03-14T10-30-00`), return empty state with warning: `"Warning: deployments.yaml was corrupted and has been renamed to deployments.yaml.corrupt.<timestamp>. Run nd sync to rebuild deployment state from the filesystem."`

### Save (NFR-010)

```go
func (s *Store) Save(st *state.DeploymentState) error
```

Marshals to YAML, delegates to `nd.AtomicWrite(s.path, data)`. Reuses the existing atomic write utility (write-to-temp, fsync, rename) already used by `sourcemanager.WriteConfig`.

### WithLock (NFR-011)

```go
func (s *Store) WithLock(fn func() error) error
```

Uses the existing `state.FileLock` type with `flock(2)` (via `syscall.Flock` or `golang.org/x/sys/unix`). The `FileLock.fd` field was designed for this.

- Acquire: open lock file, call `flock(fd, LOCK_EX)` with a 5-second timeout
- Stale detection: if timeout expires, check lock file age; if >60s, break and retry
- Release: `flock(fd, LOCK_UN)`, close fd

This replaces the `O_CREATE|O_EXCL` + PID polling approach, which has race conditions and NFS issues.

### Files added to state package

| File | Purpose |
| --- | --- |
| `internal/state/store.go` | Store struct, NewStore, Load, Save, WithLock |
| `internal/state/store_test.go` | Unit tests |

Existing files (`state.go`, `queries.go`, `health.go`, `lock.go`) unchanged.

## deploy package

### Engine struct

```go
type Engine struct {
    store     StateStore     // interface for testability
    agent     *agent.Agent
    backupDir string         // e.g., ~/.config/nd/backups/

    // Injected for testing (default to os.*)
    symlink  func(oldname, newname string) error
    readlink func(name string) (string, error)
    lstat    func(name string) (os.FileInfo, error)
    stat     func(name string) (os.FileInfo, error)
    remove   func(name string) error
    mkdirAll func(path string, perm os.FileMode) error
    rename   func(oldpath, newpath string) error
}

func New(store StateStore, agent *agent.Agent, backupDir string) *Engine
```

Both `lstat` (for checking symlink nodes) and `stat` (for following symlinks to check source existence) are injected.

### Locking contract

**Every engine operation that reads or writes state wraps the entire Load-mutate-Save cycle inside `store.WithLock()`:**

- `Deploy`: `WithLock { Load -> create symlink -> add entry -> Save }`
- `Remove`: `WithLock { Load -> remove symlink -> remove entry -> Save }`
- `DeployBulk`: `WithLock { Load -> iterate all (create symlinks, accumulate entries) -> Save once }`
- `RemoveBulk`: `WithLock { Load -> iterate all (remove symlinks, remove entries) -> Save once }`
- `Check`: `WithLock { Load -> check each entry }` (read-only, no Save)
- `Sync`: `WithLock { Load -> check -> repair -> Save }`
- `Status`: `WithLock { Load -> check each entry }` (read-only, no Save)

Bulk operations acquire the lock once and do a single Load/Save cycle for efficiency and consistency.

### Core operations

#### Deploy (FR-009, FR-011)

```go
type DeployRequest struct {
    Asset       asset.Asset
    Scope       nd.Scope
    ProjectRoot string         // required when Scope == ScopeProject
    Origin      nd.DeployOrigin
}

type DeployResult struct {
    Deployment state.Deployment
    Warnings   []string
    BackedUp   string // non-empty if an existing file was backed up
}

func (e *Engine) Deploy(req DeployRequest) (*DeployResult, error)
```

Algorithm:

1. Validate: asset must be deployable (`AssetType.IsDeployable()`)
2. Compute link path via `agent.DeployPath()`. For context assets, extract `contextFile` from `req.Asset.ContextFile.FileName`. For non-context assets, pass `""`.
3. **Conflict check (all asset types)**: `lstat(linkPath)` to check for existing file/symlink
   - Nothing exists: proceed
   - nd-managed symlink (same asset): update timestamp, return early
   - nd-managed symlink (different asset): remove old symlink, update state
   - Foreign symlink or plain file — context assets: back up existing file (FR-016b), warn with strong language for plain files. Non-context assets: return `nd.ConflictError` (spec Boundaries: "report conflicts rather than silently overwriting").
4. **Writability check**: verify parent directory is writable before creating symlink
5. Create parent directories (`mkdirAll`) if needed
6. Create symlink (`os.Symlink(sourcePath, linkPath)`)
7. Add `Deployment` entry to state, save via store
8. If `AssetType.RequiresSettingsRegistration()`, append warning about manual settings step

**Context file extraction**: when `req.Asset.Type == nd.AssetContext`, the engine reads `req.Asset.ContextFile.FileName` (e.g., `"CLAUDE.md"`) and passes it to `agent.DeployPath()`. This is the existing `ContextInfo` struct from `internal/asset/context.go`.

#### DeployBulk (FR-010)

```go
type BulkDeployResult struct {
    Succeeded []DeployResult
    Failed    []DeployError
}

type DeployError struct {
    AssetName string
    AssetType nd.AssetType
    SourcePath string
    Err       error
}

func (e *Engine) DeployBulk(reqs []DeployRequest) (*BulkDeployResult, error)
```

Fail-open: acquires lock once, loads state once, iterates through all requests (creating symlinks, accumulating state changes), saves state once at the end. Individual failures are collected, not fatal.

#### Remove (FR-012)

```go
type RemoveRequest struct {
    Identity    asset.Identity
    Scope       nd.Scope
    ProjectRoot string
}

func (e *Engine) Remove(req RemoveRequest) error
```

1. Find deployment in state by identity + scope (+ projectRoot for project scope)
2. Remove symlink at `LinkPath` (ignore "not exists" — already gone)
3. Remove deployment entry from state, save

#### RemoveBulk (FR-012)

```go
type BulkRemoveResult struct {
    Succeeded []RemoveRequest
    Failed    []RemoveError
}

type RemoveError struct {
    Identity asset.Identity
    Err      error
}

func (e *Engine) RemoveBulk(reqs []RemoveRequest) (*BulkRemoveResult, error)
```

Same single-lock, single-Save pattern as DeployBulk.

#### Status (FR-015)

```go
type StatusEntry struct {
    Deployment state.Deployment
    Health     state.HealthStatus
    Detail     string
}

func (e *Engine) Status() ([]StatusEntry, error)
```

Loads state, runs health check on each entry, returns a **flat list**. Grouping by asset type is the caller's responsibility (CLI/TUI presentation layer).

#### Check (FR-013)

Reuses the existing `state.HealthCheck` and `state.HealthStatus` types.

```go
func (e *Engine) Check() ([]state.HealthCheck, error)
```

For each deployment in state:

1. `lstat(linkPath)` — if error → `HealthMissing` (symlink was deleted externally)
2. `readlink(linkPath)` — if target != `sourcePath` → `HealthDrifted`
3. `stat(linkPath)` (follows symlinks) — if error → `HealthBroken` (symlink exists but target gone)
4. All pass → `HealthOK`

**Known limitation (FR-013 partial)**: the spec says "detects symlinks that have been renamed outside of nd." The engine detects the *absence* of the original symlink (`HealthMissing`) but does not scan config directories for renamed versions. This would require a filesystem scan of every agent config subdirectory and fuzzy matching, which is deferred. The `nd sync` command repairs `HealthMissing` entries by re-creating the symlink if the source still exists.

#### Sync (FR-014)

```go
type SyncResult struct {
    Repaired []state.Deployment
    Removed  []state.Deployment
    Warnings []string
}

func (e *Engine) Sync() (*SyncResult, error)
```

1. Call `Check()` to get all issues
2. For each issue:
   - `HealthBroken` / `HealthOrphaned`: source gone → remove symlink + remove from state
   - `HealthMissing`: source exists → re-create symlink
   - `HealthDrifted`: re-create symlink to correct target
3. Save updated state

## Context file backup (FR-016b)

When deploying a context asset and a non-symlink file exists at the target:

1. Generate backup path: `<backupDir>/<filename>.<timestamp>.bak`
   - Timestamp format: `2026-03-14T10-30-00` (hyphens for colons, filesystem-safe)
2. Rename existing file to backup path
3. Prune old backups: keep last 5 per target filename, delete older ones
4. Append warning: for plain files, use stronger language: "Backed up existing manually created file"

Backup logic lives in an unexported function `backupExistingFile` in `internal/deploy/` (testable from `_test.go` in the same package).

## Error handling

Uses existing typed errors from `internal/nd/errors.go`:

| Scenario | Behavior |
| --- | --- |
| Asset not deployable (plugins) | `fmt.Errorf("asset type %q is not deployable via symlink; use nd export", type)` |
| Conflict at target path (non-context) | `nd.ConflictError{TargetPath, ExistingKind, AssetName}` |
| Permission denied on symlink/mkdir | Wrap with actionable message per spec: `"Permission denied: cannot write to <path>..."` |
| Context + global + .local.md | `agent.DeployPath()` returns error (existing behavior) |
| Bulk partial failure | Fail-open, `BulkDeployResult.Failed` populated, CLI uses exit code 2 |
| State file locked | `nd.LockError{Path, Timeout, Stale}` after 5s timeout |
| State file corrupted | Store renames, returns empty state + warning string |
| Schema version mismatch (NFR-014) | Newer version → refuse to load with explanatory error |

## Deferred items

| Item | Reason |
| --- | --- |
| Profile switch logic (FR-022-024) | Requires Profile/Snapshot Store design |
| Auto-snapshots (FR-029a) | Requires Snapshot Store; bulk operations will gain auto-snapshot hooks later |
| `ActiveProfile` management | Deferred with profile switch logic |
| Rename detection in Check (FR-013 partial) | Would require config dir scanning; `HealthMissing` + Sync covers the repair path |

## Testing strategy

Both additions use injected dependencies — no real filesystem needed.

### state.Store tests

| Scenario | Validates |
| --- | --- |
| Load from missing file | Returns empty state with Version: nd.SchemaVersion |
| Load from valid YAML | Parses correctly |
| Load from corrupt YAML | Renames file, returns empty state + warning with prescribed format |
| Load older schema version | Migrates and backs up original |
| Load newer schema version | Refuses to load with version mismatch error |
| Save + Load round-trip | Uses nd.AtomicWrite, preserves data |
| WithLock basic | Lock acquired and released via flock |
| WithLock timeout | nd.LockError after 5s when lock held |
| WithLock stale break | Stale lock (>60s) is broken |

### deploy.Engine tests

| Scenario | Validates |
| --- | --- |
| Deploy single asset | Symlink created, state updated within WithLock |
| Deploy creates parent dirs | mkdirAll called for missing subdirs |
| Deploy with writability check failure | Error before symlink creation |
| Deploy context (no conflict) | contextFile extracted from Asset.ContextFile.FileName |
| Deploy context (existing nd symlink) | Old symlink removed, new created |
| Deploy context (existing manual file) | File backed up, warning with strong language |
| Deploy .local.md at global scope | Error from DeployPath |
| Deploy non-context (existing file) | nd.ConflictError returned |
| Deploy hook/output-style | Warning about settings registration |
| DeployBulk partial failure | Fail-open, single lock, single Save |
| DeployBulk all succeed | Single lock, single Load/Save cycle |
| Remove asset | Symlink removed, state updated |
| Remove already-gone symlink | No error, state still cleaned |
| RemoveBulk | Single lock, results separated |
| Check: healthy deployment | HealthOK |
| Check: broken link | HealthBroken (stat follows symlink, target gone) |
| Check: missing link | HealthMissing |
| Check: drifted link | HealthDrifted |
| Check: orphaned source | HealthOrphaned |
| Sync: broken link (source gone) | Symlink + state entry removed |
| Sync: missing link (source exists) | Symlink re-created |
| Sync: drifted link | Symlink corrected |
| Status | Flat list with HealthStatus per entry |
| Backup retention | Only last 5 backups kept per target filename |
| StateStore mock | Engine works with in-memory mock |

Target: >85% test coverage for both state.Store and deploy.Engine.

## Files

### state package additions

| File | Purpose |
| --- | --- |
| `internal/state/store.go` | Store struct, NewStore, Load, Save, WithLock |
| `internal/state/store_test.go` | Unit tests |

Existing files unchanged: `state.go`, `queries.go`, `health.go`, `lock.go`.

### deploy package (new)

| File | Purpose |
| --- | --- |
| `internal/deploy/deploy.go` | StateStore interface, Engine struct, New, Deploy, DeployBulk, Remove, RemoveBulk |
| `internal/deploy/health.go` | Check, Sync, Status |
| `internal/deploy/deploy_test.go` | Tests for deploy/remove operations |
| `internal/deploy/health_test.go` | Tests for check/sync/status operations |
