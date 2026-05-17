---
title: "Refresh TUI screen list after create/save/add mutation"
id: "pio815"
status: pending
priority: medium
type: bug
tags: ["tui"]
created_at: "2026-05-17"
---

## Refresh TUI screen list after create/save/add mutation

### Objective

Net-new bug (no seed; found during the codebase sweep). After a successful create/save/add in the profile, snapshot, and source TUI screens, the in-memory list is never reloaded. `buildMenu` only rebuilds the menu form; the underlying slice is set only in the initial `*LoadedMsg` handler. So a just-created profile / saved snapshot / added source does not appear in the subsequent List/Switch/Restore/Remove views until the screen is fully re-entered.

### Steps to reproduce

1. In the TUI, create a profile (or save a snapshot, or add a source).
2. Without leaving the screen, open "List" / "Switch" / "Restore" / "Remove".
3. The new item is missing.

### Tasks

- [ ] `internal/tui/profile.go:135-142` -- after `profileCreatedMsg` success, reload `s.profiles` (re-run the load command) before rebuilding the menu
- [ ] `internal/tui/snapshot.go:115-116` -- after `snapshotSavedMsg` success, reload `s.snapshots`
- [ ] `internal/tui/source.go:122-132` -- after `sourceAddedMsg`/`sourceRemovedMsg`/`sourceSyncedMsg`, reload `s.sources`
- [ ] Tests: create/save/add then assert the new item appears in the subsequent list view within the same screen session

### Acceptance criteria

- A just-created profile / saved snapshot / added source appears in the List/Switch/Restore/Remove views without re-entering the screen
- Tests cover each of the three screens
- No regression in existing TUI tests

### References

- net-new, no seed pattern
- `internal/tui/profile.go:118,135-142`, `snapshot.go:115-116`, `source.go:122-132`
