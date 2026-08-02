---
title: "Offer global or project scope on deploy"
id: "nxvr4w"
status: completed
priority: medium
type: feature
tags: ["deploy", "ux"]
created_at: "2026-04-20"
context:
  - "cmd/root.go"
  - "cmd/deploy.go"
  - "cmd/profile.go"
  - "cmd/app.go"
  - "cmd/helpers.go"
  - "internal/nd/scope.go"
  - "internal/nd/project.go"
  - "internal/agent/registry.go"
  - "internal/tui/deploy.go"
  - "internal/tui/scope.go"
  - "internal/tui/services.go"
  - "cmd/deploy_test.go"
verify:
  - type: bash
    run: "go build -o nd ."
  - type: bash
    run: "go test ./cmd/... ./internal/tui/... ./internal/deploy/..."
  - type: bash
    run: "go test -race ./..."
  - type: bash
    run: "golangci-lint run"
  - type: assert
    check: "Running `nd deploy <asset>` in a TTY without an explicit `--scope` flag prompts the user to choose global or project; passing `--scope global` or `--scope project` skips the prompt."
  - type: assert
    check: "After the first prompt answer, subsequent deploys in the same process do not re-prompt (session preference is reused)."
  - type: assert
    check: "`--json` or non-TTY stdin with no explicit `--scope` does not block on a prompt; it proceeds with the resolved default scope (no hang, no error introduced for existing scripted callers)."
completed_at: 2026-08-01
---

## Offer global or project scope on deploy

### Objective

Today every deploy silently uses **global** scope unless the user remembers to
pass `--scope project`. The `--scope` persistent flag is registered with a
default value of `string(nd.ScopeGlobal)` (`cmd/root.go:56`), so a user who
omits the flag is indistinguishable, at the value level, from one who typed
`--scope global`. New users routinely deploy system-wide by accident.

Goal: when a deploy is initiated **interactively** (TTY stdin, no explicit
`--scope`, not `--json`), prompt the user once to pick global vs project,
explain the difference, remember the choice for the rest of the process, and
apply it to single, multi-asset, and profile deploys. Non-interactive and
explicit-flag invocations must keep their current behavior exactly (no new
prompts, no new errors for existing scripted callers).

Path facts to state correctly in any user-facing copy (verified against
`internal/agent/registry.go:35-39` for the default `claude-code` agent):

- **Global** = `~/.claude/` (agent `GlobalDir`, `filepath.Join(homeDir, ".claude")`).
- **Project** = `.claude/` in the project root (agent `ProjectDir`, `".claude"`).
  Do **not** describe project scope as `.agents/` — that is not the default
  destination. (Other agents can override these dirs via config; copy may say
  "the agent's project directory" if you want to be agent-agnostic, but the
  default is `.claude/`.)

### Key facts the implementer must know

- **The flag has a non-empty default.** `cmd/root.go:56`:
  `pf.StringVarP((*string)(&app.Scope), "scope", "s", string(nd.ScopeGlobal), ...)`.
  To detect whether the user explicitly passed `--scope`, you cannot compare
  `app.Scope` to a value. Use Cobra's flag-changed API on the persistent flag:
  `cmd.Flags().Changed("scope")` (Cobra merges persistent flags into the
  subcommand's flag set, so this works from inside the `deploy` `RunE`). No
  code in the repo uses `.Changed()` yet, so this is a new pattern — verify it
  in a test (see Tasks).
- **`nd deploy --all` does not exist.** The task title's "batch deployments"
  means: (a) multiple asset args to `nd deploy a b c` (handled by
  `eng.DeployBulk`, `cmd/deploy.go:198`) and (b) `nd profile deploy <name>`
  (`cmd/profile.go`, scope read from `app.Scope` at lines ~374). There is no
  `--all` flag to add; do not invent one.
- **Scope is consumed in three request-builder sites** that all read
  `app.Scope` / `app.ProjectRoot`:
  - `cmd/deploy.go:165-174` — builds `deploy.DeployRequest{Scope: app.Scope, ProjectRoot: app.ProjectRoot, ...}`.
  - `cmd/profile.go:~374` (and similar entries near 549/627) — `Scope: app.Scope`.
  - `internal/tui/deploy.go:452-453` — `scope := ds.svc.GetScope()` / `projectRoot := ds.svc.GetProjectRoot()`.
  Resolving scope **before** these sites (so `app.Scope` already holds the
  chosen value) keeps the change small and avoids touching `deploy.Engine`.
- **Project root resolution is scope-gated.** `cmd/root.go:171-176` only calls
  `app.ResolveProjectRoot()` when `app.Scope == nd.ScopeProject`. If the prompt
  yields `project`, you must call `app.ResolveProjectRoot()` (which falls back
  to `nd.FindProjectRoot(cwd)`, `internal/nd/project.go:12`) before building
  requests, and surface its error (`"no project root found (looked for .git/ or
  .claude/ from ...)"`) if it fails.
- **`FindProjectRoot` is the "are we in a project?" check.** It walks up from
  cwd looking for `.git/` or `.claude/` (`internal/nd/project.go:18-31`). Use a
  non-fatal probe (call it, discard the path, check `err == nil`) to decide
  whether to even offer "project" as an option / whether to auto-default to
  global outside a project.
- **A reusable prompt primitive already exists.**
  `promptChoice(r io.Reader, w io.Writer, prompt string, choices []string)`
  in `cmd/helpers.go:77-97` prints numbered choices, reads a line, returns the
  chosen string, and **already errors when stdin is not a terminal**
  (`!isTerminal()`). `isTerminal()` is `cmd/helpers.go:115-117`. The existing
  "pick an asset" flow in `cmd/deploy.go:82` calls `promptChoice`; mirror that
  call site exactly for the scope prompt.
- **`config.DefaultScope` exists but is currently ignored by deploy.** It is
  parsed (`internal/config/config.go:10`) and defaulted to `nd.ScopeGlobal`
  (`internal/sourcemanager/config.go:21`), but `cmd/deploy.go` never reads it —
  it uses `app.Scope` straight from the flag. Treat `config.DefaultScope` as
  the *fallback default* for the prompt's pre-selection only; do not change how
  the `--scope` flag itself is parsed.
- **TUI already has a standalone scope switcher** (`internal/tui/scope.go`,
  `newScopeScreen`, reachable from the main menu `case "scope"` at
  `internal/tui/main_menu.go:123-124`) that emits `ScopeSwitchedMsg`
  (`internal/tui/screens.go:26-28`) and calls `svc.ResetForScope(...)`
  (`cmd/app.go:203-213`). The TUI deploy screen (`internal/tui/deploy.go`)
  currently reads scope at `internal/tui/deploy.go:452` but has **no** scope
  step in its `deployStep` flow (`deployPickType -> deploySelectAssets ->
  deployRunning -> deployConflictConfirm -> deployResult`,
  `internal/tui/deploy.go:21-27`). Reuse `huh.NewSelect[string]` exactly as
  `internal/tui/scope.go:41-48` does (Title, two `huh.NewOption`s, Catppuccin
  theme) for a new pre-pick step.

### Tasks

- [ ] **Add a scope-resolution helper in the `cmd` package.**
  - In `cmd/deploy.go` (or a new small `cmd/scope_prompt.go`), add
    `resolveDeployScope(cmd *cobra.Command, app *App) (nd.Scope, error)`:
    1. If `cmd.Flags().Changed("scope")` is true → return `app.Scope` unchanged
       (explicit flag wins, no prompt).
    2. If a session preference was already set (package-level
       `var deployScopePref *nd.Scope` in package `cmd`, nil = unset) → return
       it without prompting.
    3. If `app.JSON` or `!isTerminal()` → do **not** prompt; return the current
       default (`app.Scope`, i.e. global, possibly informed by
       `config.DefaultScope` if you wire that fallback) with no error so
       existing scripts keep working.
    4. Probe `nd.FindProjectRoot(cwd)`; if it errors (not in a project) →
       return `nd.ScopeGlobal` without prompting (record it as the session
       pref so later deploys in the same process stay consistent).
    5. Otherwise call `promptChoice(cmd.InOrStdin(), cmd.OutOrStdout(),
       "Deploy scope:", []string{"Global (system-wide, ~/.claude/)",
       "Project (this project, .claude/)"})` (mirror `cmd/deploy.go:82`),
       map the chosen label to `nd.ScopeGlobal`/`nd.ScopeProject`, store it in
       `deployScopePref`, and return it.
  - The package-level `deployScopePref` is the "remember for this session"
    store; it naturally resets on process exit. Add an unexported reset helper
    `resetDeployScopePref()` so tests can clear it between cases.

- [ ] **Wire the helper into the CLI deploy flow (`cmd/deploy.go`).**
  - In the `RunE` (`cmd/deploy.go:54`), after sources scan / before building
    requests (i.e. before `cmd/deploy.go:165` where `reqs` is built), call
    `resolveDeployScope(cmd, app)`. On non-nil error, return it.
  - When the resolved scope is `nd.ScopeProject`, call
    `app.ResolveProjectRoot()` and return its error if any (the
    `cmd/root.go:171-176` path will NOT have run because the flag default kept
    `app.Scope == global` at PreRun time). Then set `app.Scope = resolved`
    before the request loop so `cmd/deploy.go:169` and the oplog `Scope:`
    fields (`cmd/deploy.go:185`, `cmd/deploy.go:211`) pick it up.
  - Do not change the `--dry-run` path's existing output; just ensure scope is
    resolved before the dry-run branch (`cmd/deploy.go:127`) so the preview
    reflects the chosen scope if you print it (optional, see Acceptance).

- [ ] **Apply the session preference to profile / multi-asset deploys.**
  - Multi-asset (`nd deploy a b c`) already flows through the same `RunE`, so
    fixing `cmd/deploy.go` covers it once scope is resolved before
    `eng.DeployBulk` (`cmd/deploy.go:198`). No extra work beyond the wiring
    above; just confirm with a test.
  - In `cmd/profile.go` `profile deploy` `RunE` (the block around
    `cmd/profile.go:340-385`, before `profMgr.DeployProfile(... app.ProjectRoot)`
    at ~line 360), call the same `resolveDeployScope(cmd, app)` and apply it
    (including `app.ResolveProjectRoot()` when project) so profile deploys honor
    the session preference and prompt consistently. Reuse the helper; do not
    duplicate the prompt logic.

- [ ] **Add a scope step to the TUI deploy screen (`internal/tui/deploy.go`).**
  - Add a new `deployStep` constant (e.g. `deployPickScope`) to the iota block
    at `internal/tui/deploy.go:21-27`, ordered **before** `deployPickType`, and
    set it as the initial `step` in `newDeployScreen` (`internal/tui/deploy.go:108-114`).
  - Build a two-option `huh.NewSelect[string]` (Global/Project) exactly like
    `internal/tui/scope.go:39-49` (Catppuccin theme, `huh.NewOption("Global",
    "global")`, `huh.NewOption("Project", "project")`); pre-select
    `string(svc.GetScope())`.
  - Add an `updatePickScope` handler analogous to `updatePickType`
    (`internal/tui/deploy.go:302-326`): on `esc` emit `BackMsg{}`; on form
    completion, validate project scope via `svc.GetProjectRoot()` /
    `svc.ResetForScope(...)` the way `internal/tui/scope.go:101-119` does
    (reuse the same `nd.ScopeProject && GetProjectRoot()==""` guard and error
    presentation), then advance `ds.step = deployPickType` and return
    `ds.typeForm.Init()`.
  - Add the step to the `View()` switch (`internal/tui/deploy.go:276-298`),
    `InputActive()` (`internal/tui/deploy.go:137-139`), and `FullHelpItems()`
    (`internal/tui/deploy.go:143-173`) so the help bar / focus behave correctly.
  - Remember the selection for the TUI session: because `ResetForScope`
    persists scope on the shared `App` (`cmd/app.go:203-213`) which the TUI
    `Services` wraps, selecting once updates `svc.GetScope()` for the rest of
    the TUI session — no extra App-model state field is needed. Confirm
    `startDeploy` (`internal/tui/deploy.go:452-453`) reads the updated scope.

- [ ] **Handle edge cases.**
  - Outside a project (`nd.FindProjectRoot` errors): CLI auto-uses global with
    no prompt (helper step 4). TUI: keep the existing
    `internal/tui/scope.go:104-109` style guard so picking "Project" with no
    root shows the standard "no project root detected" error rather than
    deploying to a bad path.
  - `--json`: helper step 3 returns the default with no prompt (no behavior
    change for `TestDeployCmd_JSON`, `cmd/deploy_test.go:118`).
  - Non-TTY stdin (piped): helper step 3 (`!isTerminal()`) returns the default
    with no prompt and no error (no behavior change for
    `TestDeployCmd_NoArgs_NonTTY`, `cmd/deploy_test.go:165`).
  - `--yes`: `--yes` (`app.Yes`, `cmd/root.go:63`) is not a TTY signal; rely on
    the non-TTY/JSON guards above. Do not add a `--yes`-specific scope error.

- [ ] **Tests (table/CLI style mirroring `cmd/deploy_test.go`).**
  - In `cmd/deploy_test.go` (reuse `setupDeployEnv`, `cmd/deploy_test.go:14-38`,
    which already writes a `claude-code` agent with a `global_dir` override):
    - Test `--scope project` is honored and skips any prompt (assert no prompt
      text in output, deployment recorded under project). Note non-TTY: the
      explicit-flag path (helper step 1) returns before the TTY check, so this
      works without a PTY.
    - Test that with explicit `--scope global` the prompt is skipped.
    - Test that piped stdin + no `--scope` does NOT error and does NOT print the
      scope prompt (defaults to global), preserving
      `TestDeployCmd_NoArgs_NonTTY` behavior.
    - Test the session-preference reset helper: call `resolveDeployScope`-level
      logic twice in one process with the pref set, assert the second call does
      not consult input. Add a direct unit test for `cmd.Flags().Changed("scope")`
      returning true only when the flag is passed (`SetArgs` with vs without
      `--scope`).
  - In `internal/tui/deploy_test.go`: assert the deploy screen starts on the
    new scope step, that selecting "Global" advances to the type picker, and
    that the screen's `InputActive()`/`FullHelpItems()` cover the new step.
    Follow existing patterns in `internal/tui/deploy_test.go` and
    `internal/tui/scope_test.go`.
  - Ensure all pre-existing deploy tests still pass unchanged
    (`go test ./cmd/... ./internal/tui/... ./internal/deploy/...`).

### Acceptance criteria

- `nd deploy <asset>` with a TTY stdin and **no** `--scope` flag prompts
  "Deploy scope:" with Global/Project options before deploying; the prompt
  copy correctly says `~/.claude/` (global) and `.claude/` in the project
  (project).
- `nd deploy <asset> --scope global` and `--scope project` deploy with no
  prompt (verified via `cmd.Flags().Changed("scope")`).
- After answering the prompt once, a second `nd deploy ...` in the same process
  reuses the choice without re-prompting (session preference).
- `nd deploy a b c` (multi-asset) and `nd profile deploy <name>` use the same
  resolved/session scope and the same prompt behavior.
- The TUI deploy screen presents a Global/Project step before the asset-type
  picker; choosing Project with no detectable project root shows the existing
  "no project root detected" message instead of deploying.
- Outside any project (`nd.FindProjectRoot` fails from cwd), CLI deploy uses
  global with no prompt.
- `--json` and piped/non-TTY stdin without `--scope` deploy with no prompt and
  no new error (existing scripted behavior preserved).
- `go build -o nd .`, `go test -race ./...`, and `golangci-lint run` all pass;
  pre-existing tests in `cmd/deploy_test.go` are unchanged and green.

### References

- GitHub issue: https://GitHub.com/armstrongl/nd/issues/82
- Close this issue when the task is completed.
- `--scope` flag with global default: `cmd/root.go:56`; scope validation /
  project-root gating: `cmd/root.go:163-176`.
- Scope type: `internal/nd/scope.go` (`ScopeGlobal`, `ScopeProject`).
- Project detection: `internal/nd/project.go:12-32` (`FindProjectRoot`);
  resolver `App.ResolveProjectRoot` `cmd/app.go:152-168`.
- Default agent dirs (`~/.claude` global, `.claude` project):
  `internal/agent/registry.go:35-39`.
- Prompt primitive: `promptChoice` `cmd/helpers.go:77-97`; `isTerminal`
  `cmd/helpers.go:115-117`; existing call site `cmd/deploy.go:82`.
- CLI request builders that read scope: `cmd/deploy.go:165-174`,
  `cmd/deploy.go:198`; profile: `cmd/profile.go:~340-385`.
- TUI scope picker to mirror: `internal/tui/scope.go:29-119`; deploy screen
  steps: `internal/tui/deploy.go:19-27`, `108-114`, `137-173`, `276-326`,
  `452-453`; `ScopeSwitchedMsg`/`ResetForScope`: `internal/tui/screens.go:26-28`,
  `cmd/app.go:203-213`; Services interface: `internal/tui/services.go:16-45`.
- Test scaffold to reuse: `setupDeployEnv` `cmd/deploy_test.go:14-38`;
  existing cases `cmd/deploy_test.go:40-247`.
- Related (do not depend on, separate bug): TUI project-scope-switch task
  `tasks/tui/2r0nyd-tui-project-scope-switch.md` (same `GetProjectRoot()==""`
  guard pitfall when launched in global scope).
- Origin: GitHub issue https://GitHub.com/armstrongl/nd/issues/82 (closed;
  formerly tracked as ISSUE-018).
