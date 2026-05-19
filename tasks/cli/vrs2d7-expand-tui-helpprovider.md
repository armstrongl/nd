---
title: "Implement HelpProvider on TUI screens missing it"
id: "vrs2d7"
status: pending
priority: medium
type: feature
tags: ["tui", "ux"]
created_at: "2026-05-17"
dependencies: ["6bije3"]
context:
  - "internal/tui/helpbar.go"
  - "internal/tui/helpbar_test.go"
  - "internal/tui/screens.go"
  - "internal/tui/deploy.go"
  - "internal/tui/remove.go"
  - "internal/tui/browse.go"
  - "internal/tui/status.go"
  - "internal/tui/main_menu.go"
  - "internal/tui/firstrun.go"
  - "internal/tui/scope.go"
  - "internal/tui/settings.go"
  - "internal/tui/profile.go"
  - "internal/tui/snapshot.go"
  - "internal/tui/pin.go"
  - "internal/tui/source.go"
  - "internal/tui/doctor.go"
  - "internal/tui/testutil_test.go"
verify:
  - type: bash
    run: "go build -o nd ."
  - type: bash
    run: "go test ./internal/tui/..."
  - type: bash
    run: "go test -race ./internal/tui/..."
  - type: bash
    run: "go test ./internal/tui/... -run TestHelp"
  - type: assert
    check: "All 13 screen constructors (newBrowseScreen, newDeployScreen, newDoctorScreen, newFirstRunScreen, newMainMenuScreen, newProfileScreen, newPinScreen, newRemoveScreen, newScopeScreen, newSettingsScreen, newSnapshotScreen, newSourceScreen, newStatusScreen) return a value satisfying HelpProvider or FullHelpProvider"
  - type: assert
    check: "On a huh MultiSelect step the help bar shows 'x/space toggle' (not 'enter select'); on a huh Confirm step it shows 'h/l yes/no'; on a text-input step it does not show 'enter select'"
---

## Implement HelpProvider on TUI screens missing it

### Objective

Pattern expansion of seed task 6bije3 (`tasks/cli/6bije3-tui-help-instructions.md`,
"Add embedded help instructions to TUI").

The bottom help bar (`internal/tui/helpbar.go`) renders a one-line keybinding hint.
`defaultHelp` (`helpbar.go:36-52`) returns a screen's `FullHelpItems()` if it
implements `FullHelpProvider` (`helpbar.go:19-21`), otherwise the generic
`esc back / j-k navigate / enter select` set plus any `HelpItems()` from
`HelpProvider` (`helpbar.go:12-14`) plus `? help` and `q quit`.

There are exactly **13 `Screen` types** (one per `func new*Screen(svc Services,
styles Styles, isDark bool)` constructor — verified list below). Only **4**
implement a help interface:

- `internal/tui/deploy.go:143` — `(*deployScreen).FullHelpItems()`, `switch ds.step`
- `internal/tui/remove.go:95` — `(*removeScreen).FullHelpItems()`, `switch m.step`
- `internal/tui/browse.go:55` — `(*browseScreen).FullHelpItems()`, branches on `b.filter.active`
- `internal/tui/status.go:53` — `(*statusScreen).HelpItems()`, branches on `s.filter.active`

The other **9** fall back to the generic bar, which is often wrong: it advertises
`enter select` while the active step is a huh MultiSelect (toggle with x/space),
a huh Confirm (h/l for yes/no), or a text input. Implement step-aware
`FullHelpProvider` (or `HelpProvider`) on the 9 missing screens so the bar always
shows the keys that step's `Update` actually handles.

Why: GitHub issue #75 (https://GitHub.com/armstrongl/nd/issues/75) asks for
contextual per-screen help; seed 6bije3 establishes the interfaces and the
`?` overlay. This task closes the gap so every screen has accurate help text.

Note: an earlier draft claimed "the seed text wrongly lists `deploy` as missing".
That is false — 6bije3 correctly lists `deploy` as a reference implementation.
The accurate list of missing screens is the 9-item checklist below.

### Huh widget key conventions (verified, do not invent keys)

The 9 screens drive `charm.land/huh/v2` forms. Keys actually handled:

- huh **Select** (single-choice menu): `j/k` or `↑/↓` navigate, `enter` select,
  `esc` aborts the form. Used by main_menu, firstrun, scope, settings menu/scope,
  profile menu/switch, snapshot menu/restore-select, source menu/remove-select.
- huh **Input** (text field): typed text + `backspace`, `enter` submits, `esc`
  aborts. Used by profile create-name, snapshot save-name, source add inputs.
- huh **Confirm** (yes/no): `h/l` or `←/→` toggle, `enter` confirm, `esc` aborts.
  Used by snapshot restore-confirm, pin confirm, source remove-confirm, doctor confirm.
  Confirmed by the comment at `internal/tui/doctor.go:216-217` ("huh Confirm widget
  uses h/l (not j/k)") and deploy's `deployConflictConfirm` help at `deploy.go:160-165`.
- Custom `tea.KeyPressMsg` switches (not huh): `j/k`+`up/down` scroll for list
  views (e.g. `profile.go:385-394`, `snapshot.go:418-427`, `source.go:464-473`,
  `doctor.go:217-226`); `enter` to return on "done"/result/error views.
- A pin MultiSelect uses `x`/`space` toggle, `j/k` navigate, `enter` confirm
  (mirror deploy's `deploySelectAssets` help at `deploy.go:152-159`).

Mirror the existing exemplars exactly:

- Step-switch `FullHelpItems`: `internal/tui/deploy.go:143-173` and
  `internal/tui/remove.go:95-119` (each `switch <recv>.step { case ...: return
  []HelpItem{...} default: return []HelpItem{...} }`).
- State-branch `HelpItems`: `internal/tui/status.go:53-67`.

Always end an item set with `{"q", "quit"}` on full-replace screens (matches
deploy/remove). For `HelpProvider` (not full-replace) the bar already appends
`?`/`q`, so do not duplicate them — see status's `HelpItems`.

### Tasks

For each screen below, add a `FullHelpItems() []HelpItem` method (these screens
are multi-step / form-driven, so full-replace is correct) keyed off the screen's
`step` field (or, where there is no step, a flat list). Use the verified step
constants and `InputActive()` lines as the source of truth for which steps exist.

- [ ] `internal/tui/main_menu.go` — `mainMenuScreen` has no `step` (single huh
      Select; `InputActive()` always `false` at line 64). Add a flat
      `FullHelpItems()` returning `{"j/k","navigate"},{"enter","select"},
      {"?","help"},{"q","quit"}` (no `esc back` — main menu is the root).
- [ ] `internal/tui/firstrun.go` — `firstRunScreen`, single huh Select (Add a
      source / Quit), `InputActive()` at line 59. Flat `FullHelpItems()`:
      `{"j/k","navigate"},{"enter","select"},{"q","quit"}`.
- [ ] `internal/tui/scope.go` — `scopeScreen`. Steps: `scopeFormStep`,
      `scopeShowError` (`scope.go:13-16`). `Update` (`scope.go:61-88`): form step
      = huh Select; `scopeShowError` only handles `enter` (`scope.go:63-68`).
      `switch s.step`: `scopeShowError` → `{"enter","return"},{"q","quit"}`;
      default → `{"esc","back"},{"j/k","navigate"},{"enter","select"},{"q","quit"}`.
- [ ] `internal/tui/settings.go` — `settingsScreen`. Steps: `settingsMenu`,
      `settingsShowResult`, `settingsSwitchScope` (`settings.go:16-20`).
      `settingsMenu`/`settingsSwitchScope` are huh Selects; `settingsShowResult`
      handles only `enter` (`settings.go:228-238`). `switch s.step`:
      `settingsShowResult` → `{"enter","return"},{"q","quit"}`; default (menu /
      scope) → `{"esc","back"},{"j/k","navigate"},{"enter","select"},{"q","quit"}`.
- [ ] `internal/tui/profile.go` — `profileScreen`. Steps: `profileLoading`,
      `profileMenu`, `profileList`, `profileSwitch`, `profileCreateName`,
      `profileDone` (`profile.go:15-22`). Per `Update`/handlers:
      `profileMenu`/`profileSwitch` huh Select; `profileCreateName` huh Input
      (`profile.go:324-356`); `profileList` j/k scroll + esc back
      (`profile.go:385-395`); `profileDone` enter to return (`profile.go:374-379`).
      `switch s.step`: list → `{"esc","back"},{"j/k","scroll"},{"q","quit"}`;
      create → `{"esc","cancel"},{"enter","submit"},{"q","quit"}`;
      done/loading → `{"enter","return"},{"q","quit"}`;
      default (menu/switch) → `{"esc","back"},{"j/k","navigate"},
      {"enter","select"},{"q","quit"}`.
- [ ] `internal/tui/snapshot.go` — `snapshotScreen`. Steps: `snapshotLoading`,
      `snapshotMenu`, `snapshotSaveName`, `snapshotRestoreSelect`,
      `snapshotList`, `snapshotDone` (`snapshot.go:15-22`). Note
      `snapshotRestoreSelect` has two phases: huh Select then huh Confirm
      (`snapshot.go:321-360`) — distinguish via `s.confirmForm != nil`.
      `switch s.step`: list → `{"esc","back"},{"j/k","scroll"},{"q","quit"}`;
      saveName → `{"esc","cancel"},{"enter","submit"},{"q","quit"}`;
      restoreSelect with `s.confirmForm != nil` → `{"h/l","yes/no"},
      {"enter","confirm"},{"q","quit"}` else `{"esc","back"},{"j/k","navigate"},
      {"enter","select"},{"q","quit"}`; done/loading → `{"enter","return"},
      {"q","quit"}`; default (menu) → navigate/select set as above.
- [ ] `internal/tui/pin.go` — `pinScreen`. Steps: `pinLoading`, `pinSelect`,
      `pinConfirm`, `pinRunning`, `pinDone` (`pin.go:16-22`). `pinSelect` is a
      huh **MultiSelect** (`pin.go:164-173`); `pinConfirm` huh Confirm
      (`pin.go:193-213`); `pinDone` handles `enter` (`pin.go:278-286`).
      `switch s.step`: pinSelect → `{"esc","back"},{"j/k","navigate"},
      {"x/space","toggle"},{"enter","confirm"},{"q","quit"}`;
      pinConfirm → `{"h/l","yes/no"},{"enter","confirm"},{"q","quit"}`;
      default (loading/running/done) → `{"enter","return"},{"q","quit"}`.
- [ ] `internal/tui/source.go` — `sourceScreen`. Steps: `sourceLoading`,
      `sourceMenu`, `sourceList`, `sourceAddLocalInput`, `sourceAddGitInput`,
      `sourceRemoveSelect`, `sourceRemoveConfirm`, `sourceSyncing`,
      `sourceDone` (`source.go:15-25`). Add/inputs are huh Input
      (`source.go:259-303`); remove-select huh Select, remove-confirm huh
      Confirm (`source.go:356-410`); list j/k scroll (`source.go:464-474`);
      done enter (`source.go:452-457`). `switch s.step`:
      list → `{"esc","back"},{"j/k","scroll"},{"q","quit"}`;
      addLocal/addGit → `{"esc","cancel"},{"enter","submit"},{"q","quit"}`;
      removeSelect → `{"esc","back"},{"j/k","navigate"},{"enter","select"},
      {"q","quit"}`; removeConfirm → `{"h/l","yes/no"},{"enter","confirm"},
      {"q","quit"}`; done/loading/syncing → `{"enter","return"},{"q","quit"}`;
      default (menu) → `{"esc","back"},{"j/k","navigate"},{"enter","select"},
      {"q","quit"}`.
- [ ] `internal/tui/doctor.go` — `doctorScreen`. Steps: `doctorLoading`,
      `doctorConfirm`, `doctorFixing`, `doctorDone` (`doctor.go:16-21`).
      `doctorConfirm` intercepts `j/k`/`up`/`down` for issue-list scroll then
      delegates to a huh Confirm using `h/l` (`doctor.go:205-247`, comment at
      `:216-217`); `doctorDone` handles `enter` (`doctor.go:249-259`).
      `switch d.step`: doctorConfirm → `{"j/k","scroll"},{"h/l","yes/no"},
      {"enter","confirm"},{"q","quit"}`; default (loading/fixing/done) →
      `{"enter","return"},{"q","quit"}`.
- [ ] Add a coverage test in a new file `internal/tui/help_coverage_test.go`
      (package `tui`). Build each of the 13 screens with
      `svc := newMockServices()` (`internal/tui/testutil_test.go:41`) and
      `styles := NewStyles(true)`, then assert each result satisfies
      `HelpProvider` or `FullHelpProvider`. Table-drive over every constructor:
      `newBrowseScreen`, `newDeployScreen`, `newDoctorScreen`,
      `newFirstRunScreen`, `newMainMenuScreen`, `newProfileScreen`,
      `newPinScreen`, `newRemoveScreen`, `newScopeScreen`, `newSettingsScreen`,
      `newSnapshotScreen`, `newSourceScreen`, `newStatusScreen` — all have
      signature `(svc Services, styles Styles, isDark bool)`. For each: fail
      unless `_, okA := s.(HelpProvider); _, okB := s.(FullHelpProvider)` and
      `okA || okB`. Mirror the assertion style in
      `internal/tui/helpbar_test.go` and the table-driven style in
      `internal/tui/screens_test.go`.
- [ ] Optionally add per-screen tests mirroring
      `internal/tui/deploy_test.go:597-646` (`TestDeploy_FullHelpItems_*`):
      construct the screen, set its `step` field, call `FullHelpItems()`, and
      assert the expected `HelpItem` is present (e.g. MultiSelect step contains
      `{"x/space","toggle"}`, Confirm step contains `{"h/l","yes/no"}`).
- [ ] Run `go build -o nd .` then
      `go test ./internal/tui/... && go test -race ./internal/tui/... &&
      go test ./internal/tui/... -run TestHelp`.

### Acceptance criteria

- All 13 `Screen` types implement `HelpProvider` or `FullHelpProvider`,
  enforced by the new `internal/tui/help_coverage_test.go`.
- For every form step, the help bar text matches the keys that step's `Update`
  actually handles: huh MultiSelect → `x/space toggle` (never `enter select`);
  huh Confirm → `h/l yes/no`; text-input steps do not show `enter select`.
- The 4 already-implemented screens (deploy, remove, browse, status) and the
  bottom `HelpBar` are unchanged; `internal/tui/helpbar_test.go`
  (`TestHelpBarView_*`, `TestDefaultHelp_*`) still passes with no edits.
- `go build -o nd .` succeeds; `go test ./internal/tui/...`,
  `go test -race ./internal/tui/...`, and
  `go test ./internal/tui/... -run TestHelp` all pass.

### References

- GitHub issue: https://GitHub.com/armstrongl/nd/issues/75
- Close this issue when the task is completed.
- Seed task (defines `?` overlay + interfaces; status `pending`):
  `tasks/cli/6bije3-tui-help-instructions.md` (dependency id `6bije3`)
- Help interfaces + bar: `internal/tui/helpbar.go` (`HelpItem` 5-9,
  `HelpProvider` 12-14, `FullHelpProvider` 19-21, `HelpBar.View` 27-34,
  `defaultHelp` 36-52)
- `Screen` interface: `internal/tui/screens.go:6-10`
- FullHelpProvider exemplars (step-switch): `internal/tui/deploy.go:143-173`,
  `internal/tui/remove.go:95-119`
- FullHelpProvider exemplar (state-branch): `internal/tui/browse.go:55-70`
- HelpProvider exemplar (state-branch): `internal/tui/status.go:53-67`
- huh Confirm uses h/l, evidence: `internal/tui/doctor.go:216-217`,
  `internal/tui/deploy.go:160-165`
- Test scaffolding: `internal/tui/testutil_test.go` (`newMockServices` line 41),
  `internal/tui/screens_test.go` (table-driven stub pattern),
  `internal/tui/helpbar_test.go` (`defaultHelp`/interface assertions),
  `internal/tui/deploy_test.go:597-646` (`FullHelpItems` per-step tests)
