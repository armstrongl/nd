---
title: "TUI scope switch to project fails with 'no project root detected' when launched in global scope"
id: "2r0nyd"
status: pending
priority: high
type: bug
tags: ["tui", "scope", "bug"]
created_at: "2026-05-17"
---

## TUI scope switch to project fails with 'no project root detected' when launched in global scope

### Steps to reproduce

1. `cd` into a valid project directory (one containing `.git/` or `.claude/`).
2. Launch the nd TUI with no scope flags (defaults to global scope).
3. Open Switch Context.
4. Select scope -> project.

### Expected behavior

The TUI detects the project root from the current working directory and switches to project scope.

### Actual behavior

The TUI shows: `Cannot switch to project scope: no project root detected.` even though the cwd is inside a valid project.

### Root cause

`App.GetProjectRoot()` (`cmd/app.go:199`) returns the cached `a.ProjectRoot`, which is only populated by `App.ResolveProjectRoot()`. That resolver is called from `PersistentPreRun` (`cmd/root.go:172-176`) **only when `app.Scope == nd.ScopeProject`**. The TUI launches in the default global scope, so `ResolveProjectRoot()` never runs and `a.ProjectRoot` stays empty.

The TUI scope-switch handlers gate on `s.svc.GetProjectRoot() == ""`:

- `internal/tui/scope.go:105`
- `internal/tui/settings.go:102`
- `internal/tui/tui.go:167`

Because the root was never resolved, this check is true even when cwd is a valid project, producing the false "no project root detected" error.

### Proposed fix

In the TUI scope-switch path, resolve the project root on demand instead of relying solely on the pre-cached value:

- Call `App.ResolveProjectRoot()` (which falls back to `nd.FindProjectRoot(cwd)`) when switching to project scope, and only show the error if resolution actually fails.
- Apply the fix consistently across all three call sites (`scope.go`, `settings.go`, `tui.go`).
- Surface the underlying `FindProjectRoot` error message ("looked for .git/ or .claude/ from ...") so a genuine non-project directory still gives a clear message.

### Tasks

- [ ] Add on-demand project root resolution to the TUI project-scope switch path (likely a service method exposed via the `services.go` interface).
- [ ] Update `internal/tui/scope.go` to resolve the root before reporting the error.
- [ ] Update `internal/tui/settings.go` scope handler the same way.
- [ ] Update `internal/tui/tui.go:167` scope handler the same way.
- [ ] Distinguish "genuinely not in a project" (keep a clear error) from "root not yet resolved" (resolve and proceed).
- [ ] Update/extend `internal/tui/scope_test.go` to cover launching in global scope from within a project, then switching to project scope.
- [ ] Add a regression test for the non-project-directory case to confirm the error is still shown with a useful message.

### Acceptance criteria

- Launching the TUI in global scope from inside a project (`.git/` or `.claude/` present), then switching context to project scope, succeeds and switches scope.
- Switching to project scope from a directory that is not inside any project still shows a clear error referencing the missing `.git/`/`.claude/` markers.
- All three scope-switch entry points (`scope.go`, `settings.go`, `tui.go`) behave consistently.
- New/updated tests in `internal/tui/scope_test.go` cover both the success and the genuine-failure cases and pass.

### Environment

- OS: macOS (Darwin 25.3.0)
- Version: nd TUI on `main`
- Component: `internal/tui` (scope switching), `cmd/app.go` (project root resolution)
