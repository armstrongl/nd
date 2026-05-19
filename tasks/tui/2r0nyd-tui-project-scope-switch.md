---
title: "TUI scope switch to project fails with 'no project root detected' when launched in global scope"
id: "2r0nyd"
status: pending
priority: high
type: bug
tags: ["tui", "scope", "bug"]
created_at: "2026-05-17"
context:
  - "cmd/app.go"
  - "cmd/root.go"
  - "internal/nd/project.go"
  - "internal/tui/services.go"
  - "internal/tui/scope.go"
  - "internal/tui/settings.go"
  - "internal/tui/tui.go"
  - "internal/tui/testutil_test.go"
  - "internal/tui/scope_test.go"
  - "internal/tui/settings_test.go"
  - "internal/tui/tui_test.go"
verify:
  - type: bash
    run: "go build -o nd ."
  - type: bash
    run: "go test ./internal/tui/... ./cmd/... ./internal/nd/..."
  - type: bash
    run: "go test -race ./internal/tui/..."
  - type: assert
    check: "Launching TUI in global scope from inside a project (.git/ or .claude/ present), then switching to project scope via the Switch scope screen, succeeds (calls ResetForScope with the discovered root) instead of showing 'no project root detected'."
  - type: assert
    check: "Switching to project scope from a directory with no .git/ or .claude/ marker still shows a clear error mentioning .git/ or .claude/."
  - type: assert
    check: "All three entry points (scope.go form, settings.go scope submenu, tui.go ctrl+s toggle) resolve the root on demand and behave consistently."
---

## TUI scope switch to project fails with 'no project root detected' when launched in global scope

### Objective

Make the TUI's "switch to project scope" action resolve the project root on
demand (from the current working directory) instead of relying on a value that
is only populated when nd is launched with `--scope project`. Today, launching
the TUI with no flags (global scope is the default) leaves `App.ProjectRoot`
empty, so every in-TUI attempt to switch to project scope fails even when the
cwd is inside a valid project. Fix all three switch entry points and keep a
clear error for the genuine "not in a project" case.

### Background / why the bug happens

- The default scope is `global`. `cmd/root.go:56` registers the `--scope` flag
  with default `string(nd.ScopeGlobal)`, and `cmd/root.go:44-50` launches the
  TUI (`tui.Run(app)`) for the bare `nd` invocation.
- `persistentPreRun` (`cmd/root.go:139-184`) only resolves the project root
  when scope is `project`: see `cmd/root.go:171-176`
  (`if app.Scope == nd.ScopeProject { app.ResolveProjectRoot() }`). With the
  default global scope this branch is skipped, so `App.ProjectRoot` stays `""`.
- `App.GetProjectRoot()` (`cmd/app.go:197-199`) just returns the cached
  `a.ProjectRoot` field; it does no discovery.
- `App.ResolveProjectRoot()` (`cmd/app.go:152-168`) is the on-demand resolver:
  if `a.ProjectRoot != ""` it returns it, otherwise it calls
  `nd.FindProjectRoot(cwd)` (`internal/nd/project.go:12-32`), caches the result
  in `a.ProjectRoot`, and returns it. It is currently only invoked from
  `persistentPreRun`.
- `nd.FindProjectRoot` (`internal/nd/project.go:12-32`) walks up from the start
  dir looking for a `.git/` or `.claude/` directory and, on failure, returns
  the error: `no project root found (looked for .git/ or .claude/ from <dir>)`
  (`internal/nd/project.go:28`).

### Root cause (exact locations)

Three TUI entry points gate the project-scope switch on
`GetProjectRoot() == ""` and never call `ResolveProjectRoot()`:

1. `internal/tui/scope.go:105` — `scopeScreen.handleScopeSelection()`. When the
   "Switch scope" screen completes with `project` and `s.svc.GetProjectRoot()
   == ""`, it sets `s.errorMsg = "Cannot switch to project scope: no project
   root detected."` (`scope.go:106`) and transitions to `scopeShowError`.
2. `internal/tui/settings.go:102` — `settingsScreen.Update` handling
   `settingsScopeSelectedMsg`. Same guard; sets the identical message into
   `s.result` (`settings.go:103`) and transitions to `settingsShowResult`.
3. `internal/tui/tui.go:167` — `Model.toggleScope()`, the inline `ctrl+s`
   keybinding handler (registered at `tui.go:138-139`). Same guard but it
   silently returns a no-op (`tui.go:168`, `return m, nil`) with **no error
   message at all** — pressing `ctrl+s` to go to project scope appears to do
   nothing.

Because `GetProjectRoot()` returns the never-resolved empty string, all three
treat a perfectly valid project directory as "no project".

The `Services` interface (`internal/tui/services.go:16-45`) currently exposes
`GetProjectRoot() string` but **not** `ResolveProjectRoot`. `cmd.App` already
implements `ResolveProjectRoot() (string, error)` (`cmd/app.go:152-168`), so the
fix is to add it to the interface and the test mock, then call it from the
three sites.

Note: the steps below say "Open Switch Context" loosely — the actual TUI menu
label is "Switch scope" (`internal/tui/main_menu.go:51`, also reachable via
Settings -> "Switch scope", `internal/tui/settings.go:68`).

### Steps to reproduce

1. `cd` into a valid project directory (one containing `.git/` or `.claude/`),
   e.g. this repo root `/Users/larah/Repos/Personal/nd`.
2. Run `nd` with no flags (defaults to global scope; launches the TUI).
3. From the main menu choose "Switch scope" (or Settings -> "Switch scope", or
   press `ctrl+s`).
4. Select scope -> `project`.

Expected: TUI discovers the project root from cwd and switches to project scope.
Actual: scope.go/settings.go show `Cannot switch to project scope: no project
root detected.`; the `ctrl+s` toggle silently does nothing — even though cwd is
a valid project.

### Tasks

- [ ] Add `ResolveProjectRoot() (string, error)` to the `Services` interface in
  `internal/tui/services.go` (after `GetProjectRoot() string`, ~line 39). It is
  already satisfied by `cmd.App.ResolveProjectRoot()` (`cmd/app.go:152-168`) so
  no production wiring change is needed in `cmd/`.
- [ ] Add the matching mock to `internal/tui/testutil_test.go`: a
  `resolveProjectRootFn func() (string, error)` field next to
  `getProjectRootFn` (line 28) and a `func (m *mockServices)
  ResolveProjectRoot() (string, error)` method mirroring the existing
  `GetProjectRoot()` pattern (`testutil_test.go:122-127`); default return
  should be `("", <error>)` so the genuine-failure tests still pass with zero
  config, OR keep it returning `m.GetProjectRoot()`-style fallthrough — pick
  one and make the new tests set `resolveProjectRootFn` explicitly.
- [ ] Fix `internal/tui/scope.go:105` (`handleScopeSelection`): when target is
  `nd.ScopeProject`, call `root, err := s.svc.ResolveProjectRoot()`. If `err !=
  nil` (or root is empty), set `s.errorMsg` to a message that includes the
  underlying `err` (so the `FindProjectRoot` "looked for .git/ or .claude/ from
  ..." text surfaces) and go to `scopeShowError`. Otherwise pass the resolved
  `root` to `s.svc.ResetForScope(newScope, root)` (replace the
  `s.svc.GetProjectRoot()` read at `scope.go:111`).
- [ ] Fix `internal/tui/settings.go:102` (`settingsScopeSelectedMsg` handler):
  same pattern — resolve on demand, include `err` text in `s.result` on
  failure, and pass the resolved root to `ResetForScope` (replace the
  `s.svc.GetProjectRoot()` at `settings.go:107`).
- [ ] Fix `internal/tui/tui.go:167` (`Model.toggleScope`): resolve on demand;
  pass the resolved root to `m.svc.ResetForScope` (replace the
  `m.svc.GetProjectRoot()` at `tui.go:171`). On failure return `m, nil` (this
  is a silent toggle with no error surface — keep that behavior; do not invent
  a new error screen here).
- [ ] Keep the global -> any and project -> global paths unchanged (no
  resolution needed when target scope is `global`).
- [ ] Update `internal/tui/scope_test.go`: add a test where
  `resolveProjectRootFn` returns `("/some/project", nil)` and `getProjectRootFn`
  returns `""` (simulates launched-in-global-from-inside-a-project), assert
  `ResetForScope` is called once with scope `project` and root
  `/some/project`. Update `TestScope_NoProjectRootShowsError`
  (`scope_test.go:148-173`) so the mock's `resolveProjectRootFn` returns an
  error, and assert the error path still triggers `scopeShowError` and that
  `m.errorMsg` mentions `.git`/`.claude` (or contains the resolver error).
- [ ] Update `internal/tui/settings_test.go`
  (`TestSettingsScreen_ScopeSwitch_ProjectWithNoRootShowsError`,
  ~`settings_test.go:111-131`) the same way, and add a success-from-global test.
- [ ] Update `internal/tui/tui_test.go` (the `ctrl+s`/`toggleScope` tests
  around `tui_test.go:399-470`): add a case where `getProjectRootFn` is `""`
  but `resolveProjectRootFn` returns `("/some/project", nil)` and assert
  `ResetForScope` is called once with the resolved root.

### Acceptance criteria

- `go build -o nd .` succeeds; `internal/tui` compiles with the new interface
  method (the mock in `testutil_test.go` satisfies it).
- Launching the TUI in global scope from inside a project (`.git/` or
  `.claude/` present), then switching to project scope via the Switch scope
  screen, the Settings submenu, or `ctrl+s`, succeeds: `ResetForScope` is
  called with the discovered project root (asserted in tests via mock
  `resetCalls`).
- Switching to project scope from a directory with no `.git/`/`.claude/`
  marker still surfaces a clear error in `scope.go` (`scopeShowError`) and
  `settings.go` (`settingsShowResult`) whose text references the missing
  `.git/`/`.claude/` markers (i.e. includes the `nd.FindProjectRoot` error).
- The `ctrl+s` toggle in `tui.go` resolves on demand and is a silent no-op
  only when resolution genuinely fails.
- `go test ./internal/tui/... ./cmd/... ./internal/nd/...` and
  `go test -race ./internal/tui/...` pass, including the new/updated tests in
  `scope_test.go`, `settings_test.go`, and `tui_test.go`.

### References

- GitHub issue: https://GitHub.com/armstrongl/nd/issues/107
- Close this issue when the task is completed.
- `cmd/app.go:152-168` — `App.ResolveProjectRoot()` (the on-demand resolver to
  expose through `Services`).
- `cmd/app.go:197-199` — `App.GetProjectRoot()` (returns cached value only).
- `cmd/app.go:203-213` — `App.ResetForScope(scope, projectRoot)` (already used
  by all three call sites; pass the freshly resolved root into it).
- `cmd/root.go:139-184` — `persistentPreRun`; the `cmd/root.go:171-176` block
  is the only current caller of `ResolveProjectRoot`, gated on project scope.
- `cmd/root.go:44-50`, `cmd/root.go:56` — bare `nd` launches the TUI; `--scope`
  defaults to `global`.
- `internal/nd/project.go:12-32` — `nd.FindProjectRoot`; error string at
  `internal/nd/project.go:28`: `no project root found (looked for .git/ or
  .claude/ from <dir>)`.
- `internal/tui/services.go:16-45` — `Services` interface (add the method here).
- `internal/tui/scope.go:101-119` — `handleScopeSelection`; guard at line 105,
  message at 106, `ResetForScope` at 112.
- `internal/tui/settings.go:100-114` — `settingsScopeSelectedMsg` handler;
  guard at 102, message at 103, `ResetForScope` at 107.
- `internal/tui/tui.go:156-176` — `Model.toggleScope`; guard at 167,
  `ResetForScope` at 171; bound to `ctrl+s` at `tui.go:138-139`.
- `internal/tui/testutil_test.go:16-145` — `mockServices`; mirror
  `GetProjectRoot` (`:122-127`) and `ResetForScope` (`:136-144`) when adding
  `ResolveProjectRoot`.
- `internal/tui/scope_test.go:101-173` — existing `ResetForScope` /
  no-root-error tests to extend.
- `internal/tui/settings_test.go:111-146` — existing settings scope tests.
- `internal/tui/tui_test.go:399-470` — existing `ctrl+s`/`toggleScope` tests.

### Environment

- OS: macOS (Darwin 25.3.0)
- Version: nd TUI on `main` (Go 1.25.8; module `github.com/armstrongl/nd`)
- Component: `internal/tui` (scope switching), `cmd/app.go` (project root
  resolution), `internal/nd/project.go` (root discovery)
