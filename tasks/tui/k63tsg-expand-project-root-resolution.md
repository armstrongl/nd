---
title: "Resolve project root on demand in TUI deploy/profile and sourcemanager"
id: "k63tsg"
status: pending
priority: high
type: bug
tags: ["tui", "scope"]
created_at: "2026-05-17"
dependencies: ["2r0nyd"]
context:
  - "cmd/app.go"
  - "cmd/root.go"
  - "internal/nd/project.go"
  - "internal/tui/services.go"
  - "internal/tui/deploy.go"
  - "internal/tui/profile.go"
  - "internal/tui/scope.go"
  - "internal/tui/settings.go"
  - "internal/tui/tui.go"
  - "internal/tui/testutil_test.go"
  - "internal/tui/deploy_test.go"
  - "internal/tui/profile_test.go"
  - "internal/sourcemanager/sourcemanager.go"
  - "tasks/tui/2r0nyd-tui-project-scope-switch.md"
verify:
  - type: bash
    run: "go build -o nd ."
  - type: bash
    run: "go test ./internal/tui/... ./cmd/... ./internal/sourcemanager/... -count=1"
  - type: bash
    run: "go test ./... -count=1"
  - type: bash
    run: "golangci-lint run ./internal/tui/... ./cmd/... ./internal/sourcemanager/..."
  - type: assert
    check: "TUI deploy in project scope builds DeployRequest with a non-empty resolved ProjectRoot even when the TUI was launched in default global scope inside a project"
  - type: assert
    check: "TUI profile switch in project scope passes a non-empty resolved project root to mgr.Switch even when launched in global scope inside a project"
  - type: assert
    check: "SourceManager merges .nd/config.yaml project config when cwd is inside a project, regardless of launch scope"
  - type: assert
    check: "A cwd that is not inside any project still produces the FindProjectRoot error ('looked for .git/ or .claude/ from ...')"
---

## Resolve project root on demand in TUI deploy/profile and sourcemanager

### Objective

Make project-root resolution lazy and consistent so that running the nd TUI in
its default global scope from inside a real project does not silently degrade
project-scoped operations.

**Why:** `App.ProjectRoot` (`cmd/app.go:22`) is only populated by
`App.ResolveProjectRoot()` (`cmd/app.go:154-168`), which `PersistentPreRunE`
calls **only** when `app.Scope == nd.ScopeProject` (`cmd/root.go:171-176`). The
TUI launches in the default global scope, so `a.ProjectRoot` stays `""`. Seed
task `2r0nyd` (`tasks/tui/2r0nyd-tui-project-scope-switch.md`) fixes the three
scope-*switch* gates that read `GetProjectRoot() == ""`
(`scope.go:105`, `settings.go:102`, `tui.go:167`). This task is the **extension**:
the same empty-root defect also reaches:

1. **TUI deploy** — `deploy.go:453` reads `ds.svc.GetProjectRoot()` and puts it
   into every `deploy.DeployRequest.ProjectRoot`. In project scope after a
   switch, or if the user is in project scope but the root was never cached,
   this can be `""`, so project-scoped deploys target the wrong location.
2. **TUI profile switch** — `profile.go:319` passes `svc.GetProjectRoot()` as
   the last arg to `mgr.Switch(...)`, with the same defect.
3. **SourceManager construction** — `cmd/app.go:52` passes `a.ProjectRoot` to
   `sourcemanager.New(...)`. When it is `""`, `sourcemanager.New`
   (`internal/sourcemanager/sourcemanager.go:33-40`) skips the
   `projectDir/.nd/config.yaml` `LoadProjectConfig` + `MergeConfigs` step
   entirely, so project config is silently ignored when the TUI is launched in
   global scope inside a project.

The fix: make `GetProjectRoot()` resolve on demand (delegating to
`App.ResolveProjectRoot()` → `nd.FindProjectRoot`), apply it consistently at the
deploy/profile/sourcemanager sites, and keep the genuine "not in a project"
error (`internal/nd/project.go:28`:
`no project root found (looked for .git/ or .claude/ from %s)`) intact.

### Root cause (verified)

- `App.GetProjectRoot()` at `cmd/app.go:199` is `return a.ProjectRoot` — a bare
  read of the cached field, no resolution.
- `App.ResolveProjectRoot()` at `cmd/app.go:154-168` is the only writer of
  `a.ProjectRoot`; it returns the cached value if set, else `os.Getwd()` →
  `nd.FindProjectRoot(cwd)` and caches the result.
- `cmd/root.go:171-176` only calls `app.ResolveProjectRoot()` when
  `app.Scope == nd.ScopeProject`. The TUI default scope is `nd.ScopeGlobal`
  (`internal/nd/scope.go:7-8`), so resolution never runs for the TUI's default
  launch.
- Downstream consumers of the unresolved empty value:
  - `internal/tui/deploy.go:453` — `projectRoot := ds.svc.GetProjectRoot()`,
    used at `deploy.go:462` (`ProjectRoot: projectRoot`).
  - `internal/tui/profile.go:319` —
    `mgr.Switch(current, target, eng, summary.Index, svc.GetProjectRoot())`.
  - `cmd/app.go:52` — `sourcemanager.New(a.ConfigPath, a.ProjectRoot)`.

### Repro

1. `cd` into a directory containing `.git/` or `.claude/`.
2. `go build -o nd . && ./nd` (no scope flags → global scope).
3. In the TUI, switch scope to project (works via the seed `2r0nyd` fix).
4. Deploy an asset; inspect the resulting `state/deployments.yaml` (path under
   `filepath.Dir(ConfigPath)/state/deployments.yaml`, see
   `cmd/app.go:147-149`): the deployment's project root is empty / not the
   project dir. Equivalent failure for profile switch and for `.nd/config.yaml`
   not being merged.

### Tasks

- [ ] **`cmd/app.go:197-199`** — Change `GetProjectRoot()` to resolve on demand:
      call `a.ResolveProjectRoot()` and on success return the resolved path; on
      error return `""` (the existing callers treat `""` as "no project").
      Mirror the doc-comment style of the other accessors (`cmd/app.go:185-199`).
      Note: `ResolveProjectRoot()` caches into `a.ProjectRoot`, so this stays
      cheap after the first call and is safe with `ResetForScope`
      (`cmd/app.go:201-213`) which nils `ProjectRoot`.
- [ ] **`cmd/app.go:47-58` (`SourceManager`)** — Before calling
      `sourcemanager.New(a.ConfigPath, a.ProjectRoot)` at `cmd/app.go:52`,
      resolve the project root (best-effort: ignore the error, fall back to the
      current `a.ProjectRoot`) so `.nd/config.yaml` merges
      (`internal/sourcemanager/sourcemanager.go:33-40`) when cwd is inside a
      project even if launched global. Prefer reusing the resolving accessor
      from the previous item rather than duplicating `ResolveProjectRoot`
      handling. Do not error the TUI when not in a project — pass `""` through
      as today (the `projectDir == ""` branch is the supported non-project path).
- [ ] **`internal/tui/services.go:39`** — The `Services` interface already
      declares `GetProjectRoot() string`. No signature change is required; the
      resolving behavior comes from `cmd.App` (which satisfies this interface
      per `services.go:14-16`). Confirm the interface comment still reads
      accurately; update only if behavior wording changed.
- [ ] **`internal/tui/deploy.go:451-453`** — With `GetProjectRoot()` now
      resolving, no code change is required here unless resolution must be
      surfaced as an error to the user; verify `projectRoot` is non-empty for a
      project-scope deploy and add a comment noting the resolving accessor is
      relied upon. If a clearer error is wanted for project scope with no
      resolvable root, set `ds.err` and `ds.step = deployResult` (mirror the
      pattern at `deploy.go:482-490`).
- [ ] **`internal/tui/profile.go:319`** — Same: `svc.GetProjectRoot()` now
      resolves; verify non-empty for project-scope switch. No structural change
      expected beyond the resolving accessor.
- [ ] **Consistency check (do NOT modify — owned by seed `2r0nyd`)** — Confirm
      the resolving accessor makes the seed-covered gates at `scope.go:105`,
      `settings.go:102`, and `tui.go:167` behave correctly together; if `2r0nyd`
      is already merged, ensure no double-resolution or behavior conflict. These
      sites read `s.svc.GetProjectRoot() == ""` /
      `m.svc.GetProjectRoot() == ""`.
- [ ] **Surface the genuine non-project message** — When `FindProjectRoot`
      fails for a project-scoped action, ensure the user sees the underlying
      message `no project root found (looked for .git/ or .claude/ from %s)`
      (`internal/nd/project.go:28`, wrapped by `cmd/app.go:163-164`). Decide
      whether to thread this string through or keep the existing static
      "no project root detected" copy in the seed-covered gates; for
      deploy/profile, prefer surfacing the real error since those paths show
      `err.Error()` (`deploy.go:264-269`).
- [ ] **Regression tests** — Add to `internal/tui/deploy_test.go` and
      `internal/tui/profile_test.go`, driving the screens via the `mockServices`
      double in `internal/tui/testutil_test.go` (override `getProjectRootFn`,
      `getScopeFn`, `stateStoreFn`, `scanIndexFn`, `deployEngineFn` /
      `profileManagerFn` as the existing tests do):
  - deploy in project scope builds a `deploy.DeployRequest` whose
    `ProjectRoot` equals the resolved project dir (assert non-empty / expected
    value). Inspect `ds.reqs` after `startDeploy`.
  - profile switch in project scope passes the resolved root to `mgr.Switch`
    (capture via a `profileManagerFn` stub or assert on the value handed to the
    switch). Mirror existing assertions in `profile_test.go`.
  - Add a `cmd` or `internal/sourcemanager` test (e.g. in `cmd/app_test.go` if
    present, else `internal/sourcemanager/sourcemanager_test.go`) that, with a
    temp dir containing `.git/` and `.nd/config.yaml`, constructs the manager
    via `App.SourceManager()` in global scope and asserts the project config was
    merged (a project-only `config.yaml` key is present in `sm.Config()`).
  - Negative case: a cwd with no `.git/`/`.claude/` ancestor still yields the
    `FindProjectRoot` error and the project-scope action surfaces it.

### Acceptance criteria

- TUI project-scoped deploy builds `DeployRequest`s with a non-empty resolved
  `ProjectRoot` (`internal/tui/deploy.go:453,462`), even when the TUI was
  launched in default global scope inside a project.
- TUI profile switch in project scope passes a non-empty resolved project root
  to `mgr.Switch` (`internal/tui/profile.go:319`) under the same conditions.
- `App.SourceManager()` (`cmd/app.go:47-58`) merges `.nd/config.yaml` when cwd
  is inside a project, regardless of launch scope.
- A cwd not inside any project still produces a clear error referencing the
  missing `.git/`/`.claude/` markers (`internal/nd/project.go:28`).
- New tests pass; no regression with the seed `2r0nyd` scope-switch fix; all
  `verify` commands pass.

### References

- GitHub issue: https://GitHub.com/armstrongl/nd/issues/110
- Close this issue when the task is completed.
- Seed task (extends, does not duplicate):
  `tasks/tui/2r0nyd-tui-project-scope-switch.md` (id `2r0nyd`, this task's
  `dependencies`).
- Root-cause sites: `cmd/app.go:22` (`ProjectRoot` field),
  `cmd/app.go:47-58` (`SourceManager`), `cmd/app.go:154-168`
  (`ResolveProjectRoot`), `cmd/app.go:197-199` (`GetProjectRoot`),
  `cmd/app.go:201-213` (`ResetForScope`); `cmd/root.go:171-176`
  (scope-gated resolution); `internal/nd/project.go:12-32`
  (`FindProjectRoot`, error string at line 28); `internal/nd/scope.go:7-8`
  (`ScopeGlobal`/`ScopeProject`).
- Consumer sites: `internal/tui/deploy.go:451-466`,
  `internal/tui/profile.go:293-322`, `internal/tui/services.go:13-45`
  (interface), `internal/sourcemanager/sourcemanager.go:27-50`
  (project-config merge).
- Seed-covered gates (do not modify here): `internal/tui/scope.go:101-119`,
  `internal/tui/settings.go:100-114`, `internal/tui/tui.go:156-176`.
- Test double: `internal/tui/testutil_test.go` (`mockServices`,
  `getProjectRootFn` at line 28, `resetCalls` tracking).
