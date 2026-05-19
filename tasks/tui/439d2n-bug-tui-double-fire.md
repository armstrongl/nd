---
title: "Add double-fire guard to TUI create/save/add forms"
id: "439d2n"
status: pending
priority: medium
type: bug
tags: ["tui"]
created_at: "2026-05-17"
context:
  - internal/tui/profile.go
  - internal/tui/snapshot.go
  - internal/tui/source.go
  - internal/tui/deploy.go
  - internal/tui/deploy_test.go
  - internal/tui/profile_test.go
  - internal/tui/snapshot_test.go
  - internal/tui/source_test.go
verify:
  - type: bash
    run: "go build -o nd ."
  - type: bash
    run: "go test ./internal/tui/..."
  - type: bash
    run: "go test -race ./internal/tui/..."
  - type: bash
    run: "golangci-lint run ./internal/tui/..."
  - type: assert
    check: "runCreate/runSave/runAdd fire exactly once per form completion even when Update is called again after huh.StateCompleted"
---

## Add double-fire guard to TUI create/save/add forms

### Objective

Three Bubble Tea TUI screens issue their "create"/"save"/"add" command without a
re-entrancy guard. Once the underlying `huh.Form` reaches `huh.StateCompleted`,
its `State` stays `StateCompleted`. Bubble Tea can deliver another `tea.Msg`
(another keypress, a `tea.WindowSizeMsg`, etc.) to the screen's `Update` before
the async result message (`profileCreatedMsg` / `snapshotSavedMsg` /
`sourceAddedMsg`) arrives and flips `s.step` to the done state. Each such
intervening `Update` re-enters the form handler, re-observes
`State == huh.StateCompleted`, and returns the command **again** — causing a
duplicate create/save/add attempt (e.g. a second "profile already exists" error,
a duplicate snapshot write, or a duplicate source clone).

This is a net-new bug found during a codebase sweep (no seed pattern / no
upstream ticket). The fix is to mirror the existing, proven double-fire guard
already used by the sibling handlers in the same package.

### Root cause (verified)

The three completed-state branches fire their command unconditionally, with no
guard flag checked at function entry:

- `internal/tui/profile.go:344-356` — `func (s *profileScreen) updateCreateForm`.
  At `profile.go:349-351`, `if s.createForm.State == huh.StateCompleted { return s, s.runCreate() }`. No guard at function entry (compare `updateSwitchForm` at `profile.go:275-291`, which checks `if s.switching { return s, nil }` at `profile.go:276` and sets `s.switching = true` at `profile.go:284` before calling `s.runSwitch()`).
- `internal/tui/snapshot.go:263-275` — `func (s *snapshotScreen) updateSaveForm`.
  At `snapshot.go:268-270`, `if s.saveForm.State == huh.StateCompleted { return s, s.runSave() }`. No guard. (Compare `updateRestoreSelect` at `snapshot.go:321-360`, guarded by `if s.fixing { return s, nil }` at `snapshot.go:322`, with `s.fixing = true` set at `snapshot.go:353` before `s.runRestore()`.)
- `internal/tui/source.go:287-303` — `func (s *sourceScreen) updateAddForm`.
  At `source.go:292-298`, `if s.addForm.State == huh.StateCompleted { ... return s, s.runAdd(kind, s.addInput) }`. No guard. (Compare `updateRemove` at `source.go:356-394`, guarded by `if s.removing { return s, nil }` at `source.go:357`, with `s.removing = true` set at `source.go:387` before `s.runRemove()`.)

Canonical reference implementation of the pattern to copy: `deployScreen.updateSelectAssets` in `internal/tui/deploy.go:399-427` — note the field declared at `deploy.go:66` (`deploying bool // H1: guards against double-fire after asset form completion`), the entry guard `if ds.deploying { return ds, nil }` at `deploy.go:401-404`, and `ds.deploying = true` set at `deploy.go:418` immediately before `return ds, ds.startDeploy()`.

### Steps to reproduce

1. Open the TUI, go to Profiles → Create profile, type a name, press enter to
   complete the form (`s.createForm.State` becomes `huh.StateCompleted`).
2. Before `profileCreatedMsg` is delivered, another `tea.Msg` reaches
   `profileScreen.Update`, which routes to `updateCreateForm` again
   (`profile.go:153`).
3. `updateCreateForm` re-observes `huh.StateCompleted` and returns
   `s.runCreate()` a second time → `pstore.CreateProfile` is invoked twice for
   the same name. The same race exists for snapshot save and source add.

Deterministic test-level repro (no real terminal needed): construct the screen,
set its step + a completed form, call the update handler twice, and assert the
second call returns a nil command.

### Tasks

- [ ] `internal/tui/profile.go`: add field `creating bool` to the `// create`
      group of the `profileScreen` struct (struct ends at `profile.go:75`;
      `// create` group is `profile.go:64-66`). Reset it to `false` in
      `buildCreateForm` (`profile.go:324-342`, alongside `s.createName = ""` at
      `profile.go:326`). In `updateCreateForm` (`profile.go:344-356`) add
      `if s.creating { return s, nil }` as the first statement, and set
      `s.creating = true` immediately before `return s, s.runCreate()` at
      `profile.go:350`. Mirror `updateSwitchForm` (`profile.go:275-291`)
      exactly.
- [ ] `internal/tui/snapshot.go`: add field `saving bool` to the `// save`
      group of `snapshotScreen` (`snapshot.go:57-59`; struct ends at
      `snapshot.go:75`). Reset it to `false` in `buildSaveForm`
      (`snapshot.go:243-261`, alongside `s.saveName = ""` at `snapshot.go:245`).
      In `updateSaveForm` (`snapshot.go:263-275`) add
      `if s.saving { return s, nil }` as the first statement and set
      `s.saving = true` immediately before `return s, s.runSave()` at
      `snapshot.go:269`.
- [ ] `internal/tui/source.go`: add field `adding bool` to the `// add forms`
      group of `sourceScreen` (`source.go:66-68`; struct ends at
      `source.go:84`). Reset it to `false` in `buildAddForm`
      (`source.go:259-285`, alongside `s.addInput = ""` at `source.go:260`).
      In `updateAddForm` (`source.go:287-303`) add
      `if s.adding { return s, nil }` as the first statement and set
      `s.adding = true` immediately before `return s, s.runAdd(kind, s.addInput)`
      at `source.go:297`.
- [ ] Add a regression test to `internal/tui/profile_test.go` named
      `TestProfileScreen_DoubleFireGuard_Create`: build the screen with
      `newProfileScreen(newMockServices(), NewStyles(true), true)` (constructor
      at `profile.go:77`; `newMockServices` at
      `internal/tui/testutil_test.go:41`), set `s.step = profileCreateName`,
      set `s.creating = true`, call
      `s.updateCreateForm(tea.KeyPressMsg{Code: tea.KeyEnter})`, and assert the
      returned `cmd` is `nil`. Mirror `TestDeploy_DoubleFireGuard_SelectAssets`
      at `internal/tui/deploy_test.go:530-539`.
- [ ] Add `TestSnapshotScreen_DoubleFireGuard_Save` to
      `internal/tui/snapshot_test.go` and
      `TestSourceScreen_DoubleFireGuard_Add` to
      `internal/tui/source_test.go`, following the same template (set the
      guard flag, call the update handler with a key message, assert nil cmd).
- [ ] Run the `verify` commands; confirm `go test -race ./internal/tui/...`
      passes with no new failures.

### Acceptance criteria

- After the form reaches `huh.StateCompleted`, a second (or further) call into
  `updateCreateForm` / `updateSaveForm` / `updateAddForm` before the result
  message arrives returns `(s, nil)` and does **not** re-invoke
  `runCreate` / `runSave` / `runAdd`.
- The guard flag (`creating` / `saving` / `adding`) is reset to `false` in the
  corresponding `build*Form` builder so re-entering the flow after returning to
  the menu works normally (no permanently-stuck form).
- New tests `TestProfileScreen_DoubleFireGuard_Create`,
  `TestSnapshotScreen_DoubleFireGuard_Save`,
  `TestSourceScreen_DoubleFireGuard_Add` exist and assert a nil command on the
  post-completion repeated `Update`.
- `go build -o nd .`, `go test ./internal/tui/...`,
  `go test -race ./internal/tui/...`, and `golangci-lint run ./internal/tui/...`
  all pass with no new failures (no regression in existing TUI tests).

### References

- GitHub issue: https://GitHub.com/armstrongl/nd/issues/108
- Close this issue when the task is completed.
- Bug location: `internal/tui/profile.go:344-356` (`updateCreateForm`),
  `internal/tui/snapshot.go:263-275` (`updateSaveForm`),
  `internal/tui/source.go:287-303` (`updateAddForm`).
- Guard pattern to mirror in the same package:
  `internal/tui/profile.go:275-291` (`updateSwitchForm`, field `s.switching`
  at `profile.go:62`), `internal/tui/snapshot.go:321-360` (`updateRestoreSelect`,
  field `s.fixing` at `snapshot.go:66`), `internal/tui/source.go:356-394`
  (`updateRemove`, field `s.removing` at `source.go:75`).
- Canonical implementation + comment style:
  `internal/tui/deploy.go:66` and `internal/tui/deploy.go:399-427`
  (`deployScreen.updateSelectAssets`, field `deploying`).
- Test template: `internal/tui/deploy_test.go:530-539`
  (`TestDeploy_DoubleFireGuard_SelectAssets`); mock helper
  `internal/tui/testutil_test.go:41` (`newMockServices`).
- net-new, no seed pattern.
