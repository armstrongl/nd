---
title: "Resolve project root on demand in TUI deploy/profile and sourcemanager"
id: "k63tsg"
status: pending
priority: high
type: bug
tags: ["tui", "scope"]
created_at: "2026-05-17"
dependencies: ["2r0nyd"]
---

## Resolve project root on demand in TUI deploy/profile and sourcemanager

### Objective

Pattern expansion of seed 2r0nyd. The root cause (a cached `App.ProjectRoot` populated only on the `Scope == ScopeProject` CLI branch) affects more than the three scope-switch handlers the seed covers. When the TUI launches in default global scope inside a real project, TUI deploy and profile-switch silently propagate an empty `ProjectRoot` into project-scoped operations, and `SourceManager` silently skips project config. Fix all call sites consistently and distinguish "unresolved" from "genuinely not in a project".

### Tasks

- [ ] Add a resolving accessor (delegating to `App.ResolveProjectRoot()` / `nd.FindProjectRoot`) exposed via the `internal/tui/services.go` interface
- [ ] `internal/tui/deploy.go:453` -- resolve project root before building `DeployRequest` when `scope == project` (currently sends empty `ProjectRoot`)
- [ ] `internal/tui/profile.go:319` -- resolve project root before `mgr.Switch(...)` when `scope == project`
- [ ] `cmd/app.go:52` -- ensure `SourceManager()` receives a resolved project root so `.nd/config.yaml` project config merges even when launched in global scope inside a project
- [ ] `cmd/app.go:199` (`GetProjectRoot`) -- make the accessor resolve on demand instead of returning a possibly-empty cached field
- [ ] Confirm consistency with seed-covered sites `scope.go:105,111`, `settings.go:102,107`, `tui.go:167,171` (do not duplicate 2r0nyd; this is the extension)
- [ ] Surface the underlying `nd.FindProjectRoot` message for the genuine non-project case
- [ ] Add regression tests: TUI deploy and profile-switch carry a resolved non-empty `ProjectRoot` in project scope; sourcemanager merges project config when launched global inside a project

### Acceptance criteria

- TUI project-scoped deploy and profile-switch send a non-empty resolved `ProjectRoot`
- `SourceManager` merges project config when cwd is in a project regardless of launch scope
- Genuine non-project directory still produces a clear `.git/`/`.claude/` error
- New tests pass; no regression with the seed 2r0nyd fix

### References

- Seed task: 2r0nyd -- `tasks/tui/2r0nyd-tui-project-scope-switch.md` (this extends, does not duplicate)
- `cmd/app.go:22,52,154-168,199`; `cmd/root.go:172-176`
- `internal/tui/deploy.go:453`, `profile.go:319`, `services.go`; `internal/nd/project.go:12-32`
