---
title: "Listing profiles fails with invalid name after creating a profile and restarting nd"
id: "p18du1"
status: pending
priority: high
type: bug
tags: ["profile", "config", "tui"]
created_at: "2026-08-21"
---

## Listing profiles fails with invalid name after creating a profile and restarting nd

### Steps to reproduce

1. Launch `nd` with no active profile set (fresh state, or after `nd profile` clears it).
2. Create a profile from the TUI profile screen (`internal/tui/profile.go:423` → `Store.CreateProfile`).
3. Quit `nd`.
4. Relaunch `nd` and open the profile screen, then act on a profile (switch).

### Expected behavior

With no active profile, the profile screen works: switching to a profile deploys it from an empty baseline (the same outcome as `nd profile deploy <name>`), or the UI reports a clear, actionable message instead of a validation error. Creating a profile should either make it active or leave a state the profile screen can act on.

### Actual behavior

The header renders `no profile · global · claude-code` and acting on a profile fails with:

```
!! Error: load current profile: get profile: invalid name "": must match [a-zA-Z0-9][a-zA-Z0-9_-]*
```

Cause chain:

- `Store.CreateProfile` never calls `SetActiveProfile`, so `DeploymentState.ActiveProfile` stays unset. It is `yaml:"active_profile,omitempty"` (`internal/state/state.go:16`), so it is omitted from `deployments.yaml` and reads back as `""`.
- `Manager.ActiveProfile()` (`internal/profile/manager.go:69`) returns `""` with no fallback.
- `profileScreen.runSwitch()` (`internal/tui/profile.go:338`) copies that `""` into `current` and passes it to `Manager.Switch` unguarded.
- `Manager.Switch` (`internal/profile/manager.go:148-152`) hands `currentName` straight to `Store.GetProfile`, which validates the name (`internal/profile/store.go:107-110`) against `^[a-zA-Z0-9][a-zA-Z0-9_-]*$` (`internal/profile/validate.go:9`) and fails.

The CLI already handles this: `cmd/profile.go:447-452` checks `currentName == ""` and returns a directed message.

### Environment

- OS: macOS 26.5.2
- Version: nd v0.7.0 (`d9d1110`)

### Tasks

- [ ] Guard the empty current profile in `Manager.Switch` (`internal/profile/manager.go:148`): treat `currentName == ""` as an empty baseline profile rather than calling the validating `Store.GetProfile`, so `ComputeSwitchDiff` sees no assets to remove.
- [ ] Guard `profileScreen.runSwitch()` (`internal/tui/profile.go:338`) so the TUI never sends `""` as `currentName`, and surface a readable message if the deploy path is unavailable.
- [ ] Decide and implement create semantics: either set the new profile active in `internal/tui/profile.go:423` and `cmd/profile.go:120`, or document that creation leaves no active profile. Keep TUI and CLI consistent.
- [ ] Add unit tests in `internal/profile/manager_test.go` for `Switch("", target, ...)`.
- [ ] Add a TUI test in `internal/tui/profile_test.go` covering a switch with an empty active profile.
- [ ] Verify the header rendering path (`internal/tui/header.go:57`) still shows `no profile` correctly after the fix.

### Acceptance criteria

- [ ] Create a profile, quit `nd`, relaunch, and act on the profile screen: no `invalid name ""` error.
- [ ] `Manager.Switch("", target, ...)` returns a valid `*SwitchResult` (or a clear domain error), never a name validation error.
- [ ] TUI and CLI agree on whether creating a profile makes it active.
- [ ] New tests cover the empty-active-profile switch at both the manager and TUI layer, and `go test ./...` passes.

### Context

Key files:

- `internal/profile/manager.go` — `Switch` (148), `ActiveProfile` (69), `SetActiveProfile` (84, validates only when non-empty; cleared with `""` at 394)
- `internal/profile/store.go` — `GetProfile` (107), `profilePath` (71)
- `internal/profile/validate.go` — `namePattern` (9), `ValidateName` (12)
- `internal/state/state.go` — `DeploymentState.ActiveProfile` (16), `omitempty`
- `internal/tui/profile.go` — `runSwitch` (338), error render (165), create (423)
- `internal/tui/header.go` — `no profile` label (57)
- `cmd/profile.go` — create (120), empty-active guard precedent (447-452)

Coverage gap: no existing test exercises `Switch("")` or an empty active profile end to end. `manager_test.go` only asserts `ActiveProfile()` return values; `internal/tui/profile_test.go` has no empty-active case.

#### References

- GitHub issue: https://GitHub.com/armstrongl/nd/issues/154
- Close this issue when the task is completed.
