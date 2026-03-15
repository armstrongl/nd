---
date: 2026-03-15
topic: cli-layer-cobra
---

# CLI Layer (Cobra)

## What We're Building

The CLI layer wires all 5 completed service layers (Source Manager, Deploy Engine, Agent Registry, Profile/Snapshot Store, State Store) into user-facing commands via Cobra. This is the `cmd/` package — 18 commands, 8 global flags, 4 exit codes — that turns nd from a library into a usable tool.

The CLI layer is a thin orchestration shell. All business logic lives in `internal/`. Commands translate user input (args, flags) into service calls, then format the output (human-readable or `--json`).

## Why This Approach

All service layers are complete with high test coverage. The APIs are clean and well-defined:

- `sourcemanager.New()` / `.Scan()` / `.SyncSource()` / `.AddLocal()` / `.AddGit()` / `.Remove()`
- `deploy.Engine.Deploy()` / `.DeployBulk()` / `.Remove()` / `.RemoveBulk()` / `.Check()` / `.Sync()` / `.Status()`
- `agent.Registry.New()` / `.Detect()` / `.Default()` / `.Get()` / `.All()`
- `profile.Manager.Switch()` / `.DeployProfile()` / `.Restore()` / `.ActiveProfile()`
- `profile.Store.CreateProfile()` / `.GetProfile()` / `.ListProfiles()` / `.SaveSnapshot()` / `.ListSnapshots()`
- `state.Store.Load()` / `.Save()` / `.WithLock()`

The CLI is a direct mapping from spec command tree (lines 429-475) to these APIs.

## Key Decisions

### 1. Flat `cmd/` layout

Standard cobra-cli convention. One file per top-level command; subcommands in the same file as their parent.

```text
cmd/
  root.go         # rootCmd, global flags, Execute()
  app.go          # App struct, lazy service initialization
  deploy.go       # nd deploy
  remove.go       # nd remove
  status.go       # nd status
  sync.go         # nd sync
  source.go       # nd source {add,remove,list}
  profile.go      # nd profile {create,switch,list,deploy}
  snapshot.go     # nd snapshot {save,restore,list}
  settings.go     # nd settings edit
  init_cmd.go     # nd init (init.go conflicts with Go's init())
  doctor.go       # nd doctor
  version.go      # nd version
  export.go       # nd export (Could Have — stub only)
  uninstall.go    # nd uninstall
```

Note: `init.go` is renamed to `init_cmd.go` to avoid collision with Go's `func init()` convention.

### 2. App struct with lazy initialization

A single `App` struct in `cmd/app.go` holds config paths derived from flags and lazily creates service instances. This avoids constructing unused services (e.g., `nd version` doesn't need the deploy engine).

```go
type App struct {
    // Set from flags/env in root PersistentPreRunE
    configPath  string
    scope       nd.Scope
    projectRoot string
    verbose     bool
    quiet       bool
    jsonOutput  bool
    dryRun      bool
    noColor     bool

    // Lazily initialized
    sm    *sourcemanager.SourceManager
    reg   *agent.Registry
    eng   *deploy.Engine
    prof  *profile.Manager
    pstore *profile.Store
    sstore *state.Store
}
```

Each accessor method (e.g., `app.SourceManager()`) creates and caches the instance on first call. Dependencies flow naturally: `DeployEngine()` calls `StateStore()` and `AgentRegistry()` internally.

### 3. All commands in one phase

The full command tree ships in a single design/plan/implementation cycle. This avoids revisiting the scaffold and ensures the wiring patterns established for early commands carry through to all commands.

### 4. Output formatting

Three output modes, controlled by global flags:

| Mode | Flag | Behavior |
|------|------|----------|
| Normal | (default) | Colored human-readable output to stdout, warnings/errors to stderr |
| Quiet | `--quiet` | Errors only to stderr, no stdout except `--json` |
| JSON | `--json` | Machine-readable JSON to stdout using `output.JSONResponse` envelope |

`--no-color` disables ANSI codes. `--verbose` adds detail lines to stderr.

The existing `internal/output/json.go` (`JSONResponse`, `JSONError`) is the JSON envelope. Each command defines its own `Data` payload struct.

### 5. Exit code mapping

Uses existing `internal/nd/exit_codes.go`:

| Code | Constant | When |
|------|----------|------|
| 0 | `ExitSuccess` | All operations succeeded |
| 1 | `ExitError` | General failure |
| 2 | `ExitPartialFailure` | Bulk ops: some succeeded, some failed |
| 3 | `ExitInvalidUsage` | Bad args/flags (Cobra handles most of this) |

### 6. Testing strategy

**Unit tests** (`cmd/*_test.go`): Test each command via `cmd.Execute()` with injected App. Mock service layers where needed. Fast, run on every CI push.

**Integration tests** (`tests/integration/`): Build the real binary, run via `os/exec` with real filesystem, real symlinks. Cover key user stories (US-001 through US-008). Run in CI on merge to main.

### 7. Cobra dependency

`go.mod` currently only has `gopkg.in/yaml.v3`. This will add:

- `github.com/spf13/cobra` (CLI framework)
- `github.com/spf13/pflag` (pulled in by Cobra)

No other new dependencies. Bubble Tea / Lip Gloss come later with the TUI layer.

## Command-to-API Mapping

| Command | Service calls |
|---------|--------------|
| `nd source add <path\|url>` | `sm.AddLocal()` or `sm.AddGit()` |
| `nd source remove <id>` | `sm.Remove()` |
| `nd source list` | `sm.Sources()`, per-source `ScanSource()` for counts |
| `nd deploy <asset> [assets...]` | `sm.Scan()` -> index lookup -> `eng.Deploy()` or `eng.DeployBulk()` |
| `nd remove <asset> [assets...]` | `sstore.Load()` -> find deployments -> `eng.Remove()` or `eng.RemoveBulk()` |
| `nd status` | `eng.Status()`, `profMgr.ActiveProfile()` |
| `nd sync` | `eng.Sync()`, optionally `sm.SyncSource()` for git sources |
| `nd profile create <name>` | `pstore.CreateProfile()` |
| `nd profile list` | `pstore.ListProfiles()` |
| `nd profile deploy <name>` | `sm.Scan()` -> `profMgr.DeployProfile()` |
| `nd profile switch <name>` | `sm.Scan()` -> `profMgr.Switch()` |
| `nd snapshot save <name>` | `sstore.Load()` -> `pstore.SaveSnapshot()` |
| `nd snapshot restore <name>` | `sm.Scan()` -> `profMgr.Restore()` |
| `nd snapshot list` | `pstore.ListSnapshots()` |
| `nd doctor` | `sm.Sources()`, `eng.Check()`, `reg.Detect()`, git version check |
| `nd init` | Interactive walkthrough -> write `config.yaml` |
| `nd settings edit` | `$EDITOR ~/.config/nd/config.yaml` |
| `nd version` | Build-time `ldflags` (version, commit, date) |
| `nd uninstall` | `sstore.Load()` -> remove all symlinks -> optionally remove `~/.config/nd/` |
| `nd export` | Stub: prints "not yet implemented" (Could Have) |

## Asset Resolution in CLI

When users type `nd deploy my-skill`, the CLI needs to resolve `my-skill` to an `asset.Asset`. Resolution flow:

1. `sm.Scan()` builds `asset.Index`
2. Try exact match: `index.Lookup(Identity{SourceID: "*", Type: "skills", Name: "my-skill"})`
3. If ambiguous (matches multiple types), require `--type` flag or `type/name` syntax
4. If no match, search across all types and suggest closest matches

The `nd deploy` command accepts multiple forms:

- `nd deploy my-skill` (search by name across types)
- `nd deploy skills/my-skill` (type-qualified)
- `nd deploy source-id:skills/my-skill` (fully qualified)
- `nd deploy --type skills my-skill` (flag-qualified)

## Resolved Questions

- **Profile creation UX**: `nd profile create` supports `--assets` for inline asset lists, `--from-current` to snapshot current deployments, and a `nd profile add-asset` subcommand for incremental building.
- **Dry-run format**: Same output format as real operations with `[dry-run]` prefix on action lines. JSON mode adds `"dry_run": true` to the envelope.

## Next Steps

Proceed to planning (`/ce:plan`) for the implementation plan with file-level task breakdown.
