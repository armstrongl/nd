## nd — Coding Agent Asset Manager

nd is a Go CLI tool that manages coding agent assets (skills, agents, commands, output-styles, rules, context files, plugins, hooks) via symlink-based deployment. Think of it as a package manager for AI coding assistant configurations.

### Architecture

```
main.go              → Entry point, calls cmd.Execute()
cmd/                 → Cobra CLI commands + App struct (lazy service initialization)
internal/            → All business logic (domain-based packages, not exported)
  nd/                → Core domain types: AssetType, Scope, errors, exit codes, symlink strategy
  asset/             → Asset discovery, identity, indexing, search, context metadata
  config/            → Config loading and validation (config.yaml)
  sourcemanager/     → Source registration (local/git), scanning, sync, config merging
  agent/             → Agent registry with detection (probes PATH + config dirs)
  deploy/            → Symlink deployment engine: deploy, remove, health check, repair, bulk ops
  profile/           → Profile CRUD + snapshot save/restore + Manager orchestration
  state/             → Deployment state persistence with file locking
  export/            → Plugin export: asset copying, hooks merging, marketplace generation
  oplog/             → JSONL operation log with rotation
  tui/               → Bubble Tea v2 interactive TUI (menu-driven, wizard-style)
  output/            → Structured output formatting (text/JSON)
  doctor/            → Health check diagnostics
  source/            → Source type definitions
  backup/            → Backup utilities
  version/           → Build version info (injected via ldflags)
tests/integration/   → Integration tests (full CLI exercising real filesystem)
```

### Key domain concepts

- **Asset types**: skills, agents, commands, output-styles, rules, context, plugins, hooks. Defined in `internal/nd/asset_type.go`. Plugins use export workflow (not symlinks).
- **Scope**: `global` (user-wide `~/.claude/`) or `project` (per-repo `.claude/`). Defined in `internal/nd/scope.go`.
- **Sources**: Local directories or git repos registered in config. Each source contains assets in convention-based subdirectories (e.g., `skills/`, `commands/`).
- **Deployment**: Symlinks from an agent's config directory to source asset files/dirs. Tracked in deployment state with file locking.
- **Profiles**: Named groups of deployed assets. Switch between profiles instantly.
- **Snapshots**: Point-in-time captures of deployment state for backup/restore.

### Coding conventions

- **Go 1.25+** with modules. Module path: `github.com/armstrongl/nd`
- **`main.go` at repo root**, Cobra commands in `cmd/`, all logic in `internal/`
- **Constructor pattern**: `New(...)` functions return `(*Type, error)` or `*Type`. No `init()` functions.
- **Dependency injection for testability**: OS functions (`os.Symlink`, `os.Stat`, etc.) injected as struct fields with `Set*` methods for test stubs. See `internal/deploy/deploy.go` and `internal/agent/registry.go`.
- **Error types**: Domain-specific error structs in `internal/nd/errors.go` (e.g., `PathTraversalError`, `ConflictError`, `LockError`). Use `fmt.Errorf("context: %w", err)` for wrapping.
- **Lazy initialization**: `cmd.App` fields are initialized on first access via accessor methods (e.g., `SourceManager()`, `DeployEngine()`).
- **TUI interface boundary**: `tui.Services` interface (in `internal/tui/services.go`) decouples TUI from `cmd.App`. Uses `GetScope()`/`IsDryRun()` (not `Scope()`/`DryRun()`) to avoid Go field/method name collisions.

### Testing patterns

- **Test-driven development** (RED/GREEN workflow)
- **Standard library `testing` only** — no testify or assertion libraries
- **`t.Helper()` on all test helpers**
- **`t.TempDir()` for filesystem isolation** — no shared temp dirs
- **Table-driven tests** where appropriate, with `t.Run(name, ...)` subtests
- **Command tests**: Create `cmd.App{}`, build root command via `NewRootCmd(app)`, set args with `rootCmd.SetArgs(...)`, capture output via `rootCmd.SetOut(&buf)`
- **Integration tests** in `tests/integration/` exercise the real CLI binary and filesystem
- **Race detection**: CI runs `go test ./... -race`

### CLI patterns (Cobra)

- Global flags: `--config`, `--scope` (`-s`), `--dry-run`, `--verbose` (`-v`), `--quiet` (`-q`), `--json`, `--no-color`, `--yes` (`-y`)
- Commands: deploy, remove, list, status, sync, source (add/remove/list), profile (create/delete/list/switch), snapshot (save/restore/list/delete), doctor, pin, settings, export, init, uninstall, completion, version
- Running `nd` with no args in a terminal launches the TUI

### TUI patterns (Bubble Tea v2)

- Uses `charm.land/bubbletea/v2`, `charm.land/huh/v2`, `charm.land/lipgloss/v2`
- Catppuccin-based color palette via `catppuccin/go`
- Menu-driven wizard-style navigation (not tab-based)
- All TUI code lives in `internal/tui/` — flat package, no subpackages
- Input-aware key routing (global keys suppressed when text inputs are focused)
- Visual minimalism: no borders/chrome, whitespace-driven layout

### Tooling

- **Linter**: golangci-lint v2 (`.golangci.yaml`)
- **Formatter**: gofumpt
- **Releaser**: goreleaser v2 (`.goreleaser.yaml`) — uses `homebrew_casks:` (not deprecated `brews:`)
- **CI**: GitHub Actions — lint, test (with race detector), build, goreleaser validation
- **Distribution**: Homebrew cask via `armstrongl/homebrew-tap`, GitHub Releases, `go install`

### Config file format

nd uses YAML config at `~/.config/nd/config.yaml`:

```yaml
version: 1
default_scope: global
default_agent: claude-code
symlink_strategy: absolute    # or "relative"
sources:
  - id: my-source
    type: local
    path: /path/to/source
agents:
  - name: claude-code
    global_dir: ~/.claude
```

### PR review checklist

When reviewing pull requests, check for these project-specific concerns:

**Architecture boundaries**
- Business logic must live in `internal/`, not `cmd/`. Command files in `cmd/` should only parse flags, call services, and format output.
- New packages belong under `internal/`. Nothing should be added to the top-level module API.
- TUI code stays flat in `internal/tui/` — no subpackages. The `tui.Services` interface is the only coupling point to `cmd.App`.
- Cross-package dependencies should flow downward: `cmd → internal/*`, `tui → services interface`, `deploy → state + agent + asset + nd`. Circular imports are a hard failure.

**Filesystem safety**
- All symlink and file operations must go through the `deploy.Engine` (which uses injected OS functions for testability). Direct `os.Symlink`/`os.Remove` calls in new code outside `deploy/` are a red flag.
- Path traversal: any path derived from user input or config must be validated. Check for `../` escapes. See `PathTraversalError` in `internal/nd/errors.go`.
- State mutations must use `store.WithLock(fn)` to prevent concurrent corruption. Look for state loads/saves outside a lock scope.
- Destructive operations (bulk remove, profile switch, uninstall) must create auto-snapshots via `SnapshotSaver` before proceeding (FR-029a).

**Error handling**
- Use domain error types (`ConflictError`, `LockError`, `PathTraversalError`) for actionable errors, `fmt.Errorf("context: %w", err)` for wrapping.
- Exit codes must follow `internal/nd/exit_codes.go`: 0 success, 1 error, 2 partial failure, 3 invalid usage. Check that commands return the right code for partial-success scenarios (e.g., bulk deploy where some assets fail).
- Errors should propagate up, not be swallowed. The only exception is `oplog` (best-effort logging that never blocks operations).

**Testing**
- Every new exported function or method needs tests. Check for missing test files.
- Tests must use `t.TempDir()` for filesystem isolation — never write to real config dirs or `/tmp` directly.
- Test helpers must call `t.Helper()`.
- No external test libraries (testify, gomock, etc.). Standard `testing` package only.
- Command-level tests should use the `NewRootCmd(app)` + `SetArgs` + `Execute()` pattern (see `cmd/deploy_test.go`).
- Check that new code doesn't break `go test ./... -race`.

**Symlink deployment correctness**
- New asset types must be added to `AllAssetTypes()` and conditionally to `DeployableAssetTypes()` in `internal/nd/asset_type.go`.
- Directory-based assets (skills, plugins, hooks) vs file-based assets (commands, rules, etc.) — check `IsDirectory()` is correct for new types.
- Relative vs absolute symlink strategy must be respected. Check `nd.SymlinkStrategy` usage.
- Deploy/remove results must be logged to oplog via `oplog.Writer`.

**CLI conventions**
- Conventional Commits format: `feat(scope):`, `fix(scope):`, `refactor:`, `docs:`, `ci:`, `test:`.
- New commands need: the command file, a test file, registration in `root.go`, and `--dry-run`/`--json`/`--yes` support where applicable.
- Destructive commands must require `--yes` or interactive confirmation. Check for unguarded destructive paths.
- `--dry-run` must produce output describing what _would_ happen without side effects.

**TUI consistency**
- New screens must follow the existing pattern: model struct, `Init`/`Update`/`View` methods, integration with the screen stack in `tui.go`.
- Colors must come from the Catppuccin theme in `internal/tui/theme.go` — no hardcoded ANSI codes or hex colors.
- Key bindings must be input-aware (suppressed when text inputs are focused).
- Empty states must be handled (no panics on nil/empty data).

**Dependencies**
- New dependencies need justification. This project deliberately keeps deps minimal.
- Charm libraries (`charm.land/*`) are the approved TUI stack. No competing TUI frameworks.
- `gopkg.in/yaml.v3` for YAML. No alternative YAML libraries.
- `github.com/spf13/cobra` for CLI. No alternative CLI frameworks.

### What NOT to do

- Don't put business logic in `cmd/` — it belongs in `internal/`
- Don't export anything from `internal/` packages (Go enforces this, but keep APIs minimal)
- Don't use `init()` functions — prefer explicit constructors
- Don't add external test dependencies (testify, gomock) — use standard library
- Don't use border-heavy UI in TUI — the design system is whitespace-driven
