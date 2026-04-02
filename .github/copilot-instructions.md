## nd — coding agent asset manager

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

- **Standard library `testing` only** — no testify or assertion libraries.
- **`t.Helper()` on all test helpers**.
- **`t.TempDir()` for filesystem isolation** — no shared temp dirs.
- **Table-driven tests** where appropriate, with `t.Run(name, ...)` subtests.
- **Command tests**: Create `cmd.App{}`, build root command via `NewRootCmd(app)`, set args with `rootCmd.SetArgs(...)`, capture output via `rootCmd.SetOut(&buf)`.
- **Integration tests** in `tests/integration/` exercise the real CLI binary and filesystem.
- **Race detection**: CI runs `go test ./... -race`.

### Approved dependencies

Additions need justification. This is the full approved set:

- `github.com/spf13/cobra`: CLI framework
- `gopkg.in/yaml.v3`: YAML parsing
- `charm.land/bubbletea/v2`, `charm.land/huh/v2`, `charm.land/lipgloss/v2`, `charm.land/bubbles/v2`: TUI stack
- `golang.org/x/term`: terminal detection
- `github.com/catppuccin/go`: TUI color palette

### CI pipeline

CI runs on every PR to `main`. These checks must pass:

1. **golangci-lint v2** (config: `.golangci.yml`): errcheck exclusions for test files, Cobra registration, and syscall.Flock
2. **`go test ./... -race`**: all tests with race detector
3. **`go build`**: compilation check
4. **goreleaser check**: release config validation

### Commit message format

Conventional Commits: `feat(scope):`, `fix(scope):`, `refactor:`, `docs:`, `ci:`, `test:`, `chore:`.

---

## PR review instructions

When reviewing pull requests, apply the following rules. Flag violations as review comments with the appropriate severity.

### Architecture boundaries (block if violated)

- Business logic must live in `internal/`, not `cmd/`. Command files in `cmd/` should only parse flags, call services, and format output. Flag any business logic (conditionals, loops over domain data, validation beyond flag parsing) in `cmd/` files.
- New packages belong under `internal/`. Nothing should be added to the top-level module API.
- TUI code stays flat in `internal/tui/` — no subpackages. The `tui.Services` interface is the only coupling point to `cmd.App`.
- Cross-package dependencies must flow downward: `cmd → internal/*`, `tui → services interface`, `deploy → state + agent + asset + nd`. Flag any import that creates a cycle or reverses the dependency direction.

### Filesystem safety (block if violated)

- All symlink and file operations must go through `deploy.Engine` (which uses injected OS functions for testability). Flag direct `os.Symlink`/`os.Remove`/`os.MkdirAll` calls in new code outside `internal/deploy/`.
- Path traversal: any path derived from user input or config must be validated. Flag paths that are not checked for `../` escapes. See `PathTraversalError` in `internal/nd/errors.go`.
- State mutations must use `store.WithLock(fn)` to prevent concurrent corruption. Flag state loads/saves that happen outside a lock scope.
- Destructive operations (bulk remove, profile switch, uninstall) must create auto-snapshots via `SnapshotSaver` before proceeding (FR-029a). Flag destructive paths without snapshot creation.

### Error handling (block if violated)

- Use domain error types (`ConflictError`, `LockError`, `PathTraversalError`) for actionable errors, `fmt.Errorf("context: %w", err)` for wrapping.
- Exit codes must follow `internal/nd/exit_codes.go`: 0 success, 1 error, 2 partial failure, 3 invalid usage. Flag commands that return the wrong code, especially for partial-success scenarios (e.g., bulk deploy where some assets fail).
- Errors must propagate up, not be swallowed. The only exception is `oplog` (best-effort logging that never blocks operations). Flag any `_ = err` or empty error handling outside oplog.

### Testing (block if missing)

- Every new exported function or method needs tests. Flag missing test files or untested exported symbols.
- Tests must use `t.TempDir()` for filesystem isolation — flag any writes to real config dirs, `/tmp` directly, or hardcoded paths.
- Test helpers must call `t.Helper()`. Flag helpers without it.
- No external test libraries (testify, gomock, etc.). Flag any new test dependency.
- Command-level tests must use the `NewRootCmd(app)` + `SetArgs` + `Execute()` pattern (see `cmd/deploy_test.go` for reference). Flag tests that shell out to the binary for unit-level tests.
- Flag any change that would break `go test ./... -race` (shared mutable state without synchronization).

### Symlink deployment correctness (block if wrong)

- New asset types must be added to `AllAssetTypes()` and conditionally to `DeployableAssetTypes()` in `internal/nd/asset_type.go`.
- Directory-based assets (skills, plugins, hooks) vs file-based assets (commands, rules, etc.) — verify `IsDirectory()` is correct for any new types.
- Relative vs absolute symlink strategy must be respected. Verify `nd.SymlinkStrategy` usage.
- Deploy/remove results must be logged to oplog via `oplog.Writer`. Flag deploy or remove paths that skip oplog logging.

### CLI conventions (block if wrong)

- New commands need: the command file in `cmd/`, a corresponding test file, registration in `root.go`, and `--dry-run`/`--json`/`--yes` support where applicable.
- Destructive commands must require `--yes` or interactive confirmation. Flag any unguarded destructive path (removes, deletes, uninstalls, profile switches).
- `--dry-run` must produce output describing what *would* happen without any side effects. Flag dry-run paths that perform mutations.
- Global flags (`--config`, `--scope`, `--dry-run`, `--verbose`, `--quiet`, `--json`, `--no-color`, `--yes`) are defined in `root.go`. New commands should respect these, not redefine them.

### TUI consistency (suggest if wrong)

- New screens must follow the existing pattern: model struct, `Init`/`Update`/`View` methods, integration with the screen stack in `tui.go`.
- Colors must come from the Catppuccin theme in `internal/tui/theme.go` — flag any hardcoded ANSI codes or hex colors.
- Key bindings must be input-aware (global keys suppressed when text inputs are focused).
- Empty states must be handled — flag potential panics on nil/empty data in view rendering.

### Dependencies (block if unjustified)

- New direct dependencies require justification. This project deliberately keeps deps minimal.
- Charm libraries (`charm.land/*`) are the approved TUI stack. Flag competing TUI frameworks.
- `gopkg.in/yaml.v3` for YAML. Flag alternative YAML libraries.
- `github.com/spf13/cobra` for CLI. Flag alternative CLI frameworks.
- Flag any new test-only dependencies (testify, gomock, etc.).

### Documentation style (suggest if wrong)

All documentation in `docs/` and `README.md` must follow these rules. Flag violations in docs changes:

- **Sentence case headings**: capitalize only the first word and proper nouns (`## Create profiles` not `## Creating Profiles`).
- **Base verb forms**: use imperative verbs in headings, not gerunds (`## Deploy assets` not `## Deploying Assets`).
- **`shell` code fences**: use ` ```shell `, never ` ```bash `.
- **Colon separators**: use `:` between a term and its description in lists, not `--`.
- **Standard tree notation**: use `├──`, `└──`, `│` for filesystem trees, not `+--` or `|`.
- **No forbidden words**: never use `simple`, `simply`, `easy`, `easily`, `just`, `obviously`, or `straightforward`.
- **Asset terminology**: use "source" or "asset", not "file", when describing nd deployments (both files and directories are valid targets).
- **Agent behavior hedging**: when describing what happens after deploy, use "typically" since pickup timing depends on the agent.

### Security (block if violated)

- No secrets, credentials, or tokens in committed code. Flag any hardcoded paths to user home directories, API keys, or tokens.
- Git clone URLs from user config must be validated before execution. Flag unvalidated URL usage in `internal/sourcemanager/git.go`.
- File permissions: assets copied during plugin export must preserve source permissions but not escalate them. Flag any `os.Chmod(0777, ...)` or overly permissive modes.
- Symlinks must not escape their intended scope boundary. The deploy engine validates this — flag any bypass of scope validation.

### What to approve without comment

- Straightforward dependency updates (patch/minor) in `go.mod`/`go.sum` where CI passes.
- Purely additive test coverage with no production code changes.
- Documentation-only changes that pass `scripts/lint-docs.sh` style checks.
- Refactors within a single `internal/` package that don't change the public API or behavior.

## Vexp context tools <!-- vexp v1.2.30 -->

**MANDATORY: use `run_pipeline` — do NOT grep, glob, or read files manually.**
vexp returns pre-indexed, graph-ranked context in a single call.

### Workflow

1. `run_pipeline` with your task description — ALWAYS FIRST (replaces all other tools)
2. Make targeted changes based on the context returned
3. `run_pipeline` again only if you need more context

### Available MCP tools

- `run_pipeline` — **PRIMARY TOOL**. Runs capsule + impact + memory in 1 call.
  Auto-detects intent. Includes file content. Example: `run_pipeline({ "task": "fix auth bug" })`
- `get_context_capsule` — lightweight, for simple questions only
- `get_impact_graph` — impact analysis of a specific symbol
- `search_logic_flow` — execution paths between functions
- `get_skeleton` — compact file structure
- `index_status` — indexing status
- `get_session_context` — recall observations from sessions
- `search_memory` — cross-session search
- `save_observation` — persist insights (prefer run_pipeline's observation param)

### Agentic search

- Do NOT use built-in file search, grep, or codebase indexing — always call `run_pipeline` first
- If you spawn sub-agents or background tasks, pass them the context from `run_pipeline`
  rather than letting them search the codebase independently

### Smart features

Intent auto-detection, hybrid ranking, session memory, auto-expanding budget.

### Multi-Repo

`run_pipeline` auto-queries all indexed repos. Use `repos: ["alias"]` to scope. Run `index_status` to see aliases.
<!-- /vexp -->
