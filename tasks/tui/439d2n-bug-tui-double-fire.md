---
title: "Add double-fire guard to TUI create/save/add forms"
id: "439d2n"
status: pending
priority: medium
type: bug
tags: ["tui"]
created_at: "2026-05-17"
---

## Add double-fire guard to TUI create/save/add forms

### Objective

Net-new bug (no seed; found during the codebase sweep). `updateCreateForm` (profile), `updateSaveForm` (snapshot), and `updateAddForm` (source) have no double-fire guard, unlike `updateSwitchForm`/`updateMenu`/deploy/remove which guard with a `switching`/`navigated` flag. On `StateCompleted` the huh form stays completed, so a second matching `Update` before the `*CreatedMsg`/`*SavedMsg` arrives can re-issue the create/save command (duplicate creation attempt).

### Steps to reproduce

1. Complete the profile-create (or snapshot-save / source-add) form.
2. A second `Update` arriving before the result message re-fires the command.

### Tasks

- [ ] `internal/tui/profile.go:344-356` -- add a guard flag to `updateCreateForm` (mirror `updateSwitchForm`'s `s.switching` at :276)
- [ ] `internal/tui/snapshot.go:263-275` -- add the same guard to `updateSaveForm`
- [ ] `internal/tui/source.go:287-303` -- add the same guard to `updateAddForm`
- [ ] Tests: a second `Update` after `StateCompleted` does not re-issue the command

### Acceptance criteria

- Create/save/add commands fire exactly once per form completion
- Tests assert no double-fire on a repeated post-completion `Update`
- No regression in existing TUI tests

### References

- net-new, no seed pattern
- `internal/tui/profile.go:344-356` (cf. guarded `:276`), `snapshot.go:263-275`, `source.go:287-303`
