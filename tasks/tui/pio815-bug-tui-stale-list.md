---
title: "Refresh TUI screen list after create/save/add mutation"
id: "pio815"
status: pending
priority: medium
type: bug
tags: ["tui"]
created_at: "2026-05-17"
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
    check: "After profileCreatedMsg success, s.profiles is reloaded before the user re-opens List/Switch (new profile appears without leaving the screen)"
  - type: assert
    check: "After snapshotSavedMsg success, s.snapshots is reloaded (new snapshot appears in List/Restore)"
  - type: assert
    check: "After sourceAddedMsg / sourceRemovedMsg / sourceSyncedMsg success, s.sources is reloaded (changes appear in List/Remove)"
context:
  - internal/tui/profile.go
  - internal/tui/snapshot.go
  - internal/tui/source.go
  - internal/tui/profile_test.go
  - internal/tui/snapshot_test.go
  - internal/tui/source_test.go
  - internal/tui/testutil_test.go
  - internal/tui/services.go
  - internal/tui/screens.go
---

## Refresh TUI screen list after create/save/add mutation

### Objective

In the `nd` TUI, the Profiles, Snapshots, and Sources screens cache their data
in an in-memory slice (`s.profiles`, `s.snapshots`, `s.sources`). That slice is
populated **only once**, by the screen's `Init()` load command, when its
`*LoadedMsg` is first handled. After a successful mutation (create profile /
save snapshot / add/remove/sync source), each screen sets `s.step = *Done`,
shows a confirmation, and on `enter` calls `buildMenu()` — which only rebuilds
the menu `huh.Form` and never re-runs the load command. Result: the just-created
profile / saved snapshot / added/removed/synced source is **missing** from the
List/Switch/Restore/Remove views until the screen is exited and re-entered
(which triggers a fresh `Init()`).

Fix: after each successful mutation, re-run the same load logic that `Init()`
uses so the in-memory slice reflects the change, while keeping the existing
"done" confirmation screen and any `RefreshHeaderMsg` behavior intact.

This is a net-new bug found during a codebase sweep. There is no seed/spec
reference and no related task dependency.

### Steps to reproduce

1. Run `go build -o nd . && ./nd` (or `./nd tui`) and open the Profiles screen.
2. Choose "Create profile", enter a name, confirm. The success screen shows
   `Profile "<name>" created.` Press `enter` to return to the menu.
3. Choose "List profiles" (or "Switch profile"). The just-created profile is
   **absent** from the list.
4. Exit the screen (Back/esc) and re-enter Profiles → the profile now appears
   (because `Init()` re-ran). Same defect repeats on Snapshots (Save → List/
   Restore) and Sources (Add/Remove/Sync → List/Remove).

### Root cause (verified file:line)

The data slice is set only inside the initial `*LoadedMsg` handler; the
mutation-result handlers never reload it:

- `internal/tui/profile.go:118` — `s.profiles = msg.profiles` lives only in the
  `profileLoadedMsg` case (lines 112-120). The `profileCreatedMsg` handler is
  `internal/tui/profile.go:135-142` and only sets `s.doneMsg` + `s.step =
  profileDone`; it never reloads `s.profiles`. The load logic to reuse is the
  closure in `Init()` at `internal/tui/profile.go:87-104` (calls
  `mgr.ListProfiles()` and `mgr.ActiveProfile()`).
- `internal/tui/snapshot.go:115` — `s.snapshots = msg.snapshots` lives only in
  the `snapshotLoadedMsg` case (lines 109-116). The `snapshotSavedMsg` handler
  is `internal/tui/snapshot.go:118-125` (NOTE: the old task said 115-116; that
  is the `snapshotLoadedMsg` line, not the save handler). It only sets
  `s.doneMsg` + `s.step = snapshotDone`. The load logic to reuse is the closure
  in `Init()` at `internal/tui/snapshot.go:88-101` (calls `mgr.ListSnapshots()`).
- `internal/tui/source.go:122` — `s.sources = msg.sources` lives only in the
  `sourceLoadedMsg` case (lines 116-123). The mutation handlers are
  `sourceAddedMsg` at `internal/tui/source.go:125-132`, `sourceRemovedMsg` at
  `internal/tui/source.go:134-141`, and `sourceSyncedMsg` at
  `internal/tui/source.go:143-146`. Each sets `s.step = sourceDone` and returns
  `func() tea.Msg { return RefreshHeaderMsg{} }` but never reloads `s.sources`.
  The load logic to reuse is the closure in `Init()` at
  `internal/tui/source.go:96-108` (calls `sm.Sources()`).

(NOTE: the old task referenced `source.go:122-132` as the combined target; the
mutation handlers actually span lines 125-146 as listed above.)

### Tasks

- [ ] **profile.go**: Extract the load closure currently inline in
  `(*profileScreen).Init()` (`internal/tui/profile.go:87-104`, the
  `func() tea.Msg { ... profileLoadedMsg{...} }` body) into a method, e.g.
  `func (s *profileScreen) reload() tea.Cmd` returning that command. Have
  `Init()` return `s.reload()`. In the `profileCreatedMsg` success branch
  (`internal/tui/profile.go:135-142`), after setting `s.doneMsg`/`s.step =
  profileDone`, return `s, s.reload()` on success (keep returning `s, nil` on
  `msg.err != nil`). The reload's `profileLoadedMsg` will flow back through the
  existing `profileLoadedMsg` case (lines 112-120) — but that case calls
  `s.buildMenu()`, which would yank the user off the "done" confirmation. To
  preserve the confirmation screen, guard the `profileLoadedMsg` handler so that
  if `s.step == profileDone` it only refreshes `s.profiles`/`s.active` and
  returns `s, nil` (does NOT call `buildMenu()`). Verify the existing
  `updateDone` handler (`internal/tui/profile.go:374-379`) still calls
  `buildMenu()` on `enter`.
- [ ] **snapshot.go**: Mirror the same change. Extract `Init()`'s closure
  (`internal/tui/snapshot.go:88-101`) into `func (s *snapshotScreen) reload()
  tea.Cmd`. In the `snapshotSavedMsg` success branch
  (`internal/tui/snapshot.go:118-125`) return `s, s.reload()` on success. Guard
  the `snapshotLoadedMsg` case (`internal/tui/snapshot.go:109-116`) so that when
  `s.step == snapshotDone` it only updates `s.snapshots` and returns `s, nil`
  instead of calling `buildMenu()`.
- [ ] **source.go**: Extract `Init()`'s closure
  (`internal/tui/source.go:96-108`) into `func (s *sourceScreen) reload()
  tea.Cmd`. In each of the three success branches — `sourceAddedMsg`
  (`internal/tui/source.go:125-132`), `sourceRemovedMsg`
  (`internal/tui/source.go:134-141`), `sourceSyncedMsg`
  (`internal/tui/source.go:143-146`) — return both the existing
  `RefreshHeaderMsg` cmd AND the reload cmd, batched via
  `tea.Batch(s.reload(), func() tea.Msg { return RefreshHeaderMsg{} })`
  (`tea` is the `charm.land/bubbletea/v2` alias already imported). Keep the
  error path returning only `RefreshHeaderMsg` as today (or `s, nil`). Guard the
  `sourceLoadedMsg` case (`internal/tui/source.go:116-123`) so that when
  `s.step == sourceDone` it only updates `s.sources` and returns `s, nil`
  instead of `buildMenu()`.
- [ ] **Tests** (table/idiom mirrors existing tests in
  `internal/tui/profile_test.go`, `snapshot_test.go`, `source_test.go`; mock
  via `newMockServices()` from `internal/tui/testutil_test.go`, overriding
  `profileManagerFn` / `profileStoreFn` / `profileManagerFn` /
  `sourceManagerFn` to return managers whose list grows after the mutation, or
  assert the emitted reload cmd is non-nil and produces a `*LoadedMsg`):
  - Profiles: drive `profileCreatedMsg{name, err: nil}` and assert the returned
    `tea.Cmd` is non-nil and, when invoked, yields a `profileLoadedMsg` (the
    reload). Then assert that feeding that `profileLoadedMsg` while
    `s.step == profileDone` updates `s.profiles` without leaving the done view
    (`s.step` stays `profileDone`, `View()` still contains the success text).
  - Snapshots: same shape with `snapshotSavedMsg` → `snapshotLoadedMsg`.
  - Sources: `sourceAddedMsg` / `sourceRemovedMsg` / `sourceSyncedMsg` each
    return a batched cmd; assert invoking it yields both a `sourceLoadedMsg`
    and a `RefreshHeaderMsg` (order-independent — `tea.Batch` cmds may need to
    be unwrapped; follow how `TestSourceScreen_RefreshHeaderAfterSync` at
    `internal/tui/source_test.go:150` inspects the cmd).
- [ ] Run `go build -o nd .`, `go test ./internal/tui/...`,
  `go test -race ./internal/tui/...`, `golangci-lint run ./internal/tui/...`
  and confirm no regressions in the existing TUI tests (notably
  `TestProfileScreen_*`, `TestSnapshotScreen_*`, `TestSourceScreen_*`).

### Acceptance criteria

- After a successful create profile / save snapshot / add/remove/sync source,
  the in-memory slice (`s.profiles` / `s.snapshots` / `s.sources`) is reloaded
  via the same logic `Init()` uses, so the change appears in the subsequent
  List/Switch/Restore/Remove views **without exiting and re-entering the
  screen**.
- The post-mutation confirmation ("done") screen is preserved: the reload does
  NOT bounce the user back to the menu; pressing `enter` still returns to the
  menu via the existing `updateDone` → `buildMenu()` path.
- Existing `RefreshHeaderMsg` emission on the Sources mutation handlers is
  preserved (header counts still refresh).
- New tests cover all three screens (profile create, snapshot save, source
  add/remove/sync) and assert the reload cmd is emitted.
- `go build -o nd .`, `go test ./internal/tui/...`,
  `go test -race ./internal/tui/...`, and `golangci-lint run ./internal/tui/...`
  all pass; no regression in existing TUI tests.

### References

- Bug location: `internal/tui/profile.go:118` (slice set) vs.
  `internal/tui/profile.go:135-142` (`profileCreatedMsg`, no reload);
  `internal/tui/snapshot.go:115` vs. `internal/tui/snapshot.go:118-125`
  (`snapshotSavedMsg`); `internal/tui/source.go:122` vs.
  `internal/tui/source.go:125-146` (`sourceAddedMsg`/`sourceRemovedMsg`/
  `sourceSyncedMsg`).
- Load logic to extract/reuse: `(*profileScreen).Init()`
  `internal/tui/profile.go:87-104`; `(*snapshotScreen).Init()`
  `internal/tui/snapshot.go:88-101`; `(*sourceScreen).Init()`
  `internal/tui/source.go:96-108`.
- Underlying data sources: `(*profile.Manager).ListProfiles` /
  `ActiveProfile` (`internal/profile/manager.go:95`, `:62`);
  `(*profile.Manager).ListSnapshots` (`internal/profile/manager.go:100`);
  `(*sourcemanager.SourceManager).Sources` (`internal/sourcemanager/sourcemanager.go:58`).
- Messages/return convention: `BackMsg` / `RefreshHeaderMsg` defined in
  `internal/tui/screens.go:17-24`; `tea.Batch` from the
  `charm.land/bubbletea/v2` alias (`tea`) already imported in all three files.
- Test harness: `newMockServices()` and overridable `*Fn` fields in
  `internal/tui/testutil_test.go`; analogous assertion of an emitted cmd in
  `TestSourceScreen_RefreshHeaderAfterSync`
  (`internal/tui/source_test.go:150`) and
  `TestProfileScreen_RefreshHeaderAfterSwitch`
  (`internal/tui/profile_test.go:124`).
- Net-new bug; no seed pattern, no spec/NFR reference, no task dependency.
