---
title: "Add embedded help instructions to TUI"
id: "6bije3"
status: pending
priority: medium
type: feature
tags: ["tui", "ux"]
created_at: "2026-04-20"
context:
  - "internal/tui/helpbar.go"
  - "internal/tui/helpbar_test.go"
  - "internal/tui/tui.go"
  - "internal/tui/screens.go"
  - "internal/tui/status.go"
  - "internal/tui/deploy.go"
  - "internal/tui/firstrun.go"
  - "cmd/app.go"
  - "cmd/root.go"
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
    check: "Pressing '?' on every TUI screen opens a help overlay listing that screen's keybindings; pressing '?' or 'esc' closes it"
  - type: assert
    check: "On first TUI launch a dismissible 'Press ? for help' tip appears and never reappears after the help_seen flag is written"
  - type: assert
    check: "Every screen registered in the navigation stack implements HelpProvider or FullHelpProvider"
---

## Add embedded help instructions to TUI

### Objective

Make the TUI's keybindings discoverable from inside the interface. Today the bottom
`HelpBar` (`internal/tui/helpbar.go:23-34`) renders a one-line hint that *advertises*
a `?` key (`internal/tui/helpbar.go:50` appends `HelpItem{"?", "help"}`), but **no
code anywhere handles the `?` key** — the root model's global key switch
(`internal/tui/tui.go:124-141`) only handles `q`/`ctrl+c`/`esc`/`ctrl+s`, and no
screen handles `?` in its own `Update`. So pressing `?` currently does nothing: the
affordance is a lie.

Deliver: (1) a `?`-triggered, per-screen help overlay that lists every keybinding
for the current screen, grouped under section headings; (2) `HelpProvider`
implementations on the screens that still lack one; (3) a one-time first-run tip
("Press ? for help at any time") gated by a persisted `help_seen` flag.

Why: GitHub issue #75 (https://GitHub.com/armstrongl/nd/issues/75, "Add embedded
help instructions", labels enhancement/tui) asks for first-run help text, command
hints, and contextual per-screen help. The hint bar and `HelpProvider` interface
already exist as the foundation — this task completes the loop.

### Current state (verified)

- Help interfaces live in `internal/tui/helpbar.go`:
  - `HelpItem{Key, Desc string}` — `helpbar.go:5-9`
  - `HelpProvider interface { HelpItems() []HelpItem }` — `helpbar.go:11-14` (adds
    items to the defaults)
  - `FullHelpProvider interface { FullHelpItems() []HelpItem }` — `helpbar.go:16-21`
    (replaces the default item set entirely)
  - `defaultHelp(screen Screen) []HelpItem` — `helpbar.go:36-52`: returns
    `FullHelpItems()` if the screen is a `FullHelpProvider`, otherwise the base
    `esc/j-k/enter` items plus `HelpItems()` plus `?`/`q`.
- `Screen` interface — `internal/tui/screens.go:6-10`: embeds `tea.Model` and adds
  `Title() string` and `InputActive() bool`.
- Root model `Model` — `internal/tui/tui.go:12-21`. Global key handling is in
  `Model.Update` at `tui.go:110-142` (the `tea.KeyPressMsg` case). `Model.View`
  (`tui.go:194-206`) composes `header`, the current screen's `View().Content`, and
  `helpbar` with `lipgloss.JoinVertical`, then sets `v.AltScreen = true`. The screen
  stack is `m.screens []Screen`; the active screen is `m.screens[len(m.screens)-1]`.
- Screens that **already** implement a help interface (do not duplicate work; mirror
  their pattern):
  - `internal/tui/status.go:52-67` — `(*statusScreen).HelpItems()` (state-dependent:
    returns different items when `s.filter.active`)
  - `internal/tui/deploy.go:141-170+` — `(*deployScreen).FullHelpItems()`
    (step-dependent `switch ds.step`)
  - `internal/tui/browse.go:54-` — `(*browseScreen).FullHelpItems()`
  - `internal/tui/remove.go:93-` — `(*removeScreen).FullHelpItems()`
- Screens that **lack** any help interface (need `HelpProvider`/`FullHelpProvider`),
  each verified to define only `Title()`/`InputActive()`:
  - `internal/tui/snapshot.go` (`snapshotScreen`, `Title()` at line 81)
  - `internal/tui/profile.go` (`profileScreen`, `Title()` at line 81)
  - `internal/tui/pin.go` (`pinScreen`, `Title()` at line 65)
  - `internal/tui/doctor.go` (`doctorScreen`, `Title()` at line 69)
  - `internal/tui/source.go` (`sourceScreen`, `Title()` at line 90)
  - `internal/tui/main_menu.go` (`mainMenuScreen`, `Title()` at line 63)
  - `internal/tui/settings.go` (`settingsScreen`, `Title()` at line 76)
  - `internal/tui/scope.go` (`scopeScreen`, `Title()` at line 54)
- First-run gating: `internal/tui/firstrun.go`. `tui.Run` (`tui.go:25-47`) shows
  `newFirstRunScreen` vs `newMainMenuScreen` based on `hasUserSources(svc)`
  (`firstrun.go:10-22`). **There is no `help_seen` flag anywhere today** — `grep -r
  help_seen internal/` returns nothing; it must be created.
- nd config/state directory (verified): the config file path comes from
  `svc.GetConfigPath()` (`internal/tui/services.go:38`); default is
  `~/.config/nd/config.yaml` (`cmd/root.go:245-250`). nd derives sibling state
  directories as `filepath.Dir(a.ConfigPath)` + subdir, e.g. the state store uses
  `<configDir>/state/deployments.yaml` (`cmd/app.go:147-149`) and the op log uses
  `<configDir>/logs` (`cmd/app.go:175-177`). Put the help flag at
  `filepath.Join(filepath.Dir(svc.GetConfigPath()), "state", "help_seen")` so it
  lives beside existing state. In tests `mockServices.GetConfigPath()` returns
  `/tmp/nd-test/config.yaml` unless overridden via `getConfigPathFn`
  (`internal/tui/testutil_test.go:115-120`).

### Tasks

- [ ] Add a `HelpSection` type to `internal/tui/helpbar.go` (next to `HelpItem`,
      after line 9): `type HelpSection struct { Title string; Items []HelpItem }`.
      Add an optional interface
      `type SectionedHelpProvider interface { HelpSections() []HelpSection }` so a
      screen can group its overlay help under headings (e.g. "Navigation",
      "Actions", "Filters"). Keep `HelpProvider`/`FullHelpProvider` untouched — the
      one-line bar still uses them.
- [ ] Create `internal/tui/helpoverlay.go` with a `HelpOverlay` component. API:
      `func (HelpOverlay) View(s Styles, screen Screen, width, height int) string`.
      It must derive content from the screen: if the screen implements
      `SectionedHelpProvider`, render `HelpSections()` with styled headings
      (use `s.Bold` / `s.Subtle`, mirroring `statusScreen.buildContent`
      `internal/tui/status.go:177-212`); otherwise fall back to the flat list from
      `defaultHelp(screen)` (`helpbar.go:36`). Respect `width` — truncate or wrap
      `Key + "  " + Desc` lines so nothing exceeds `width`; this satisfies the
      narrow-terminal requirement. Title the overlay with `screen.Title()`.
- [ ] Wire the `?` key in the root model. In `Model` add a `bool` field
      `helpOpen` (`internal/tui/tui.go:12-21`) and a `helpOverlay HelpOverlay`
      field next to `helpbar HelpBar`. In `Model.Update`'s `tea.KeyPressMsg` block
      (`tui.go:110-142`): when `helpOpen` is true, `?`/`esc` close it (set
      `helpOpen=false`, return early, do **not** delegate to the screen so esc does
      not also pop the nav stack); when the overlay is closed and the current
      screen's `InputActive()` is false, `?` opens it (`helpOpen=true`). Do not let
      `?` reach the screen delegate path (`tui.go:144-153`) while text input is
      active — match the existing `current.InputActive()` guard at `tui.go:117`.
- [ ] Render the overlay in `Model.View` (`tui.go:194-206`): when
      `m.helpOpen`, replace the `content` segment passed to
      `lipgloss.JoinVertical` with `m.helpOverlay.View(m.styles, currentScreen,
      m.width, m.height)` (keep the header and help bar so the layout/`AltScreen`
      behavior is unchanged). The active screen is
      `m.screens[len(m.screens)-1]`.
- [ ] Implement help on every screen currently missing it (see "Current state"
      list). For each — `snapshot.go`, `profile.go`, `pin.go`, `doctor.go`,
      `source.go`, `main_menu.go`, `settings.go`, `scope.go` — add a `HelpItems()`
      method (mirror `status.go:52-67`) or, if the screen is multi-step / uses a
      huh form, a `FullHelpItems()` method (mirror `deploy.go:141-170`). List the
      keys each screen's `Update` actually handles (read each file's
      `tea.KeyPressMsg` switch — do not invent keys). Where a heading grouping adds
      value (e.g. deploy/status with filters), also implement `HelpSections()`.
- [ ] Add screen-specific contextual hints to the overlay text where the screen
      already exposes the behavior, e.g. status screen note "Press / to filter by
      name" (it handles `/` at `internal/tui/status.go:134-136`) and deploy note
      about its multiselect toggle. Put these as a "Tips" section via
      `HelpSections()` — do not add fake keybindings.
- [ ] Create `internal/tui/helpseen.go` with two helpers:
      `helpSeenPath(svc Services) string` returning
      `filepath.Join(filepath.Dir(svc.GetConfigPath()), "state", "help_seen")`;
      `helpSeen(svc Services) bool` (file exists check); `markHelpSeen(svc
      Services) error` (MkdirAll the state dir then write the flag file — reuse the
      atomic write helper in `internal/nd/atomic.go` if convenient, else a plain
      `os.WriteFile` with `0o644`).
- [ ] Add the first-run tip. In `Model` add a `bool firstRunTip` field; in
      `tui.Run` (`tui.go:25-47`) set it to `!helpSeen(svc)`. In `Model.View`
      (`tui.go:194-206`), when `firstRunTip` is true and `helpOpen` is false,
      render a single dismissible line (e.g. styled with `m.styles.Subtle`)
      "Press ? for help at any time" above or below the content. In
      `Model.Update`'s `tea.KeyPressMsg` block: on the first key press while
      `firstRunTip` is true, set `firstRunTip=false` and call `markHelpSeen(m.svc)`
      (ignore the error best-effort, matching `App.LogOp` in `cmd/app.go:181-183`)
      so the tip never reappears.
- [ ] Tests in `internal/tui/helpoverlay_test.go` (package `tui`; mirror the style
      of `internal/tui/helpbar_test.go` including its `helpTestScreen` /
      `helpTestScreenWithItems` mocks): overlay renders flat items for a basic
      screen; renders section headings when the screen implements
      `SectionedHelpProvider`; truncates/wraps for a narrow width (e.g. width 20);
      `?` opens and `?`/`esc` close it via the root `Model.Update`.
- [ ] Test in `internal/tui/screens_test.go` (or a new `help_coverage_test.go`):
      table-drive every screen constructor (`newMainMenuScreen`,
      `newDeployScreen`, `newBrowseScreen`, `newPinScreen`, `newRemoveScreen`,
      `newProfileScreen`, `newSettingsScreen`, `newSnapshotScreen`,
      `newScopeScreen`, `newStatusScreen`, `newDoctorScreen`, `newSourceScreen`,
      `newFirstRunScreen` — all take `(svc Services, styles Styles, isDark bool)`,
      verified at the constructor lines listed above) and assert each result
      satisfies `HelpProvider` or `FullHelpProvider`.
- [ ] Tests in `internal/tui/helpseen_test.go`: with a `mockServices` whose
      `getConfigPathFn` points at a `t.TempDir()`, assert `helpSeen` is false
      initially, `markHelpSeen` creates the flag, `helpSeen` is then true; and that
      the first key press in `Model.Update` flips `firstRunTip` and persists the
      flag.

### Acceptance criteria

- Pressing `?` on any TUI screen (when no text input is focused) opens a help
  overlay titled with the screen name and listing every keybinding that screen's
  `Update` handles.
- Pressing `?` or `esc` while the overlay is open closes it and does **not** also
  pop the navigation stack or quit.
- When a screen implements `SectionedHelpProvider`, the overlay renders its items
  under section headings; otherwise it renders the flat `defaultHelp` list.
- The overlay never renders a line wider than the terminal width (truncated or
  wrapped) at narrow widths (verified at width 20 in tests).
- On first TUI launch (no `help_seen` flag at
  `<configDir>/state/help_seen`), a dismissible "Press ? for help at any time"
  line appears; the first key press dismisses it and writes the flag.
- The tip does not reappear on subsequent launches (flag present → not shown).
- Every screen constructor's result satisfies `HelpProvider` or
  `FullHelpProvider` (enforced by the coverage test).
- The bottom `HelpBar` is unchanged — existing tests in
  `internal/tui/helpbar_test.go` (`TestHelpBarView_*`, `TestDefaultHelp_*`) still
  pass with no edits.
- `go build -o nd .` succeeds; `go test ./internal/tui/...`,
  `go test -race ./internal/tui/...`, and
  `go test ./internal/tui/... -run TestHelp` all pass.

### References

- GitHub issue: https://GitHub.com/armstrongl/nd/issues/75
- Close this issue when the task is completed.
- Help interfaces + bar: `internal/tui/helpbar.go` (`HelpItem` 5-9,
  `HelpProvider` 11-14, `FullHelpProvider` 16-21, `defaultHelp` 36-52,
  `HelpBar.View` 27-34)
- Help bar tests (must keep passing): `internal/tui/helpbar_test.go`
- Root model / global keys / view composition: `internal/tui/tui.go`
  (`Model` 12-21, `Update` key block 110-153, `View` 194-206)
- `Screen` interface: `internal/tui/screens.go:6-10`
- HelpProvider exemplar (state-dependent): `internal/tui/status.go:52-67`
- FullHelpProvider exemplar (step-dependent): `internal/tui/deploy.go:141-170`
- First-run flow: `internal/tui/firstrun.go` (`hasUserSources` 10-22),
  `internal/tui/tui.go:25-47`
- Config/state dir derivation: `cmd/root.go:245-250` (default config path),
  `cmd/app.go:147-149` & `:175-177` (`filepath.Dir(ConfigPath)` + subdir pattern),
  `internal/tui/services.go:38` (`GetConfigPath()`)
- Atomic write helper (optional for the flag file): `internal/nd/atomic.go`
- Test scaffolding: `internal/tui/testutil_test.go` (`newMockServices`,
  `getConfigPathFn`), `internal/tui/screens_test.go` (stub screen pattern)
