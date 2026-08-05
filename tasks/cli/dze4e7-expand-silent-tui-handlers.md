---
title: "Fix silent TUI handlers across screens"
id: "dze4e7"
status: completed
priority: medium
type: bug
tags: ["tui"]
created_at: "2026-05-17"
dependencies: ["47kdob", "k63tsg"]
context:
  - "internal/tui/main_menu.go"
  - "internal/tui/main_menu_test.go"
  - "internal/tui/tui.go"
  - "internal/tui/pin.go"
  - "internal/tui/deploy.go"
  - "internal/tui/remove.go"
  - "internal/tui/firstrun.go"
  - "internal/tui/scope.go"
  - "internal/tui/screens.go"
  - "internal/tui/empty.go"
  - "internal/tui/theme.go"
verify:
  - type: bash
    run: "go build -o nd ."
  - type: bash
    run: "go test ./internal/tui/... -count=1"
  - type: bash
    run: "go test ./... -count=1"
  - type: bash
    run: "golangci-lint run ./internal/tui/..."
  - type: assert
    check: "Selecting \"Export plugin\" in the TUI main menu shows a visible notice (not a silent BackMsg reset)"
  - type: assert
    check: "Ctrl+S toggle to project scope with no resolvable project root shows a visible message instead of silently doing nothing"
  - type: assert
    check: "Pin no-change, deploy no-selection, and remove no-selection paths each render a visible message before returning"
completed_at: 2026-08-01
---

## Fix silent TUI handlers across screens

### Objective

Several TUI handlers terminate a user-initiated action by returning a bare
`BackMsg{}` / `PopToRootMsg{}` / `nil` with **zero user-visible feedback**, so
the user sees the screen silently snap back with no explanation. This is the
same anti-pattern fixed in seed task `47kdob`
(`tasks/cli/47kdob-fix-handle-selection-export.md`): the main-menu `case
"export":` returned a bare `BackMsg{}`. `47kdob` is a **dependency** — its fix
introduces the message-screen pattern this task reuses. The other dependency,
`k63tsg` (`tasks/tui/k63tsg-expand-project-root-resolution.md`, itself seeded by
`2r0nyd`), fixes on-demand project-root resolution; the `tui.go` `toggleScope`
fix here must layer on top of it (resolve the root, then show a message only if
resolution genuinely fails — do not duplicate `k63tsg`'s resolution work).

Goal: every silent dead-end path below produces a clear notice/result/error
message instead of a silent reset.

### Background — verified message/feedback patterns to mirror

There is **no** global `StatusMsg` / `ToastMsg` / `NoticeMsg` type. The only
navigation messages are in `internal/tui/screens.go:14-28`: `NavigateMsg`,
`BackMsg`, `PopToRootMsg`, `RefreshHeaderMsg`, `ScopeSwitchedMsg`. The `Model`
struct (`internal/tui/tui.go`, `type Model struct`) has **no** inline
status/notice field. Feedback must therefore be delivered by an in-screen step
or by navigating to a screen — not a new global message type.

Two proven in-repo patterns:

1. **In-screen "result/done/error" step that already renders text + a "Press
   enter/esc to return." hint.** Each affected screen already has such a step:
   - `deployScreen`: an `info string` field rendered at
     `internal/tui/deploy.go:270-274` ("`Press esc to go back.`"); set the
     example at `internal/tui/deploy.go:238` (`ds.info = AllDeployed(...)`).
   - `pinScreen`: the `pinDone` step. `viewDone()` (function near
     `internal/tui/pin.go:288`) already renders **"No changes made."** when
     `s.pinned == 0 && s.unpinned == 0`, with "Press enter to return.".
     `updateDone` (near `internal/tui/pin.go:278`) returns
     `PopToRootMsg`+`RefreshHeaderMsg` on Enter.
   - `removeScreen`: `m.err != nil` branch in `View()`
     (`internal/tui/remove.go`, top of `func (m *removeScreen) View()`) renders
     `m.styles.Danger.Render(m.err.Error())` + "Press esc to go back.".
2. **Two-step message screen (`scopeScreen`).** `internal/tui/scope.go:61-99`:
   `scopeShowError` step renders `fmt.Sprintf("  %s\n\n  %s", s.errorMsg,
   s.styles.Subtle.Render("Press enter to return."))` and emits
   `PopToRootMsg{}` on Enter (`internal/tui/scope.go:64-69`). `47kdob` adds an
   `exportScreen` modeled on this; reuse that same approach.

Copy convention: `internal/tui/empty.go` helpers (e.g. `NothingDeployed()` at
`internal/tui/empty.go:19-21`, `AllDeployed(typeName)` at
`internal/tui/empty.go:35-37`) — a short sentence + optional actionable hint.
Subtle style: `Styles.Subtle` declared `internal/tui/theme.go:29`, set
`internal/tui/theme.go:42`.

### Root causes (verified file:line on `main`)

1. **`internal/tui/main_menu.go:127-129`** — `(*mainMenuScreen).handleSelection`,
   `case "export":` returns `func() tea.Msg { return BackMsg{} }`. Main menu is
   the root screen, so popping it just re-renders the menu unchanged. **This is
   the exact site `47kdob` fixes** (replace with `newExportScreen(...)` +
   `NavigateMsg`). If `47kdob` is already merged this item is done — verify and
   tick it; otherwise it is owned by `47kdob`, not re-implemented here.
2. **`internal/tui/tui.go:166-169`** — `(Model).toggleScope()` (Ctrl+S binding
   dispatched at `internal/tui/tui.go:138-139`): when switching to project scope
   and `m.svc.GetProjectRoot() == ""` it does `return m, nil` — completely
   silent. `Model` has no inline message field, so feedback requires navigating
   to a message screen (the `47kdob` `exportScreen` / `scopeScreen` pattern) or
   reusing the existing `scopeScreen`. This site overlaps `2r0nyd`/`k63tsg`:
   first apply on-demand resolution per `k63tsg`, then only show a message when
   resolution truly fails (referencing the missing `.git/`/`.claude/` markers,
   matching `scope.go:106`'s error wording).
3. **`internal/tui/pin.go:198-201`** — `(*pinScreen).buildConfirm`: when
   `newPins == 0 && newUnpins == 0` it returns
   `func() tea.Msg { return BackMsg{} }` and never reaches the confirm form, so
   nothing is shown. The fix is trivial: set `s.step = pinDone` and return
   `s, nil` (or `s.Init`-equivalent) instead — `viewDone()` already renders
   "No changes made." + "Press enter to return.", and `updateDone` handles Enter.
4. **`internal/tui/deploy.go:431-434`** — `(*deployScreen).startDeploy`: when
   `len(ds.selected) == 0` it returns `func() tea.Msg { return BackMsg{} }`.
   The screen already has the `info` path; set
   `ds.info = "No assets selected."`, `ds.step = deployResult`, and return
   `ds, nil` so `View()` (`internal/tui/deploy.go:270-274`) shows the message.
5. **`internal/tui/remove.go:257-260`** — `(*removeScreen).updateSelectAssets`:
   when `len(m.selected) == 0` it returns
   `m, func() tea.Msg { return BackMsg{} }`. `removeScreen` has no `info` field
   (only `err error`). Either add an `info string` field rendered like the
   `m.err` branch in `View()`, or set
   `m.err = fmt.Errorf("No assets selected.")` + `m.step = removeResult` so the
   existing error branch renders it. Prefer a non-error `info` path to avoid the
   red "Error" styling for a benign no-op (mirror `deployScreen.info`).
6. **`internal/tui/firstrun.go:99-101`** and **`internal/tui/main_menu.go:132-133`**
   — `default:` branches returning `nil`. These are **intentional safe no-ops**:
   `main_menu.go` separator sentinels `menuSepManage`/`menuSepSystem`
   (`internal/tui/main_menu.go:10-13`) and unknown values are unreachable for
   real menu items; `mainMenuScreen.Update` resets `navigated=false` so the menu
   stays responsive (`internal/tui/main_menu.go:87-90`; covered by
   `TestMainMenu_SeparatorsAreNoOp`, `TestMainMenu_SeparatorDoesNotFreeze`,
   `TestMainMenu_UnknownChoiceDoesNotFreeze` in `main_menu_test.go:188-233`).
   `firstrun.go` has only `"add"`/`"quit"` options. Action: add a one-line
   comment documenting why each `default:` is a deliberate no-op (no behavior
   change). Do **not** add a message here.

### Stale test that must be updated

`internal/tui/main_menu_test.go:126-139`, `TestMainMenu_HandleSelectionExport`,
currently asserts the export cmd produces a `BackMsg`. **`47kdob` owns
rewriting this test** to assert a `NavigateMsg` to the export screen. If
`47kdob` is merged, just confirm it; do not re-edit. This task adds *new* tests
for sites 2–5 (see Tasks).

### Steps to reproduce (per path)

1. `go build -o nd .` then run `./nd` in a terminal inside a directory that is
   **not** a project (no `.git/` and no `.claude/`).
2. Export: arrow to "Export plugin" under "── Manage ──", Enter → menu silently
   re-renders (fixed by `47kdob`).
3. Scope: press `Ctrl+S` (toggles toward project scope) → nothing happens, no
   message (`tui.go:166-169`).
4. Pin: "Pin/Unpin assets", complete the selection form **without changing any
   pin state** → silently bounces back, no "no changes" message
   (`pin.go:198-201`).
5. Deploy: "Deploy assets", pick a type, complete the asset form selecting
   **nothing** → silently bounces back (`deploy.go:431-434`).
6. Remove: "Remove assets", complete the asset form selecting **nothing** →
   silently bounces back (`remove.go:257-260`).

### Tasks

- [ ] **Export (`internal/tui/main_menu.go:127-129`)** — owned by dependency
  `47kdob`. Verify it landed (export navigates to a visible notice screen, not
  `BackMsg`); if not yet merged, do not duplicate — leave to `47kdob`.
- [ ] **Scope (`internal/tui/tui.go:166-169`)** — depends on `k63tsg`. After
  `k63tsg`'s on-demand resolution, replace the silent `return m, nil`: if the
  root genuinely cannot be resolved, surface a visible message (navigate to a
  message screen mirroring `47kdob`'s `exportScreen` / `scopeScreen` at
  `internal/tui/scope.go:90-99`, or push a fresh `newScopeScreen` whose
  `scopeShowError` step already shows the error). Reuse `scope.go:106` wording
  referencing missing `.git/`/`.claude/`. Coordinate with `k63tsg`; do not
  re-implement project-root resolution.
- [ ] **Pin (`internal/tui/pin.go:198-201`)** — replace the
  `if newPins == 0 && newUnpins == 0 { return s, func() tea.Msg { return
  BackMsg{} } }` body with `s.step = pinDone; return s, nil`. `viewDone()`
  (near `internal/tui/pin.go:288`) already renders "No changes made." and
  `updateDone` (near `:278`) returns `PopToRootMsg`+`RefreshHeaderMsg` on Enter.
  Confirm `s.pinned`/`s.unpinned` are 0 in this branch so the existing message
  fires.
- [ ] **Deploy (`internal/tui/deploy.go:431-434`)** — replace the
  `len(ds.selected) == 0` body with `ds.info = "No assets selected."; ds.step =
  deployResult; return func() tea.Msg { return nil }` (or equivalent that lands
  on the info-rendering path at `internal/tui/deploy.go:270-274`). Mirror the
  existing `ds.info = AllDeployed(...)` use at `internal/tui/deploy.go:238`.
- [ ] **Remove (`internal/tui/remove.go:257-260`)** — add an `info string`
  field to `removeScreen` (struct near `internal/tui/remove.go:43-70`) and
  render it in `View()` like the existing `m.err` branch but with
  `m.styles.Subtle`/non-danger styling + "Press esc to go back." Then in
  `updateSelectAssets` set `m.info = "No assets selected."` and `m.step =
  removeResult` instead of returning `BackMsg`. (Alternative: reuse `m.err`,
  but prefer non-error styling for a benign no-op, matching `deployScreen.info`.)
- [ ] **Default branches** — add a clarifying comment above
  `internal/tui/firstrun.go:99` (`default:`) and confirm the existing comment
  intent at `internal/tui/main_menu.go:8-13`/`:132-133`. No behavior change; the
  separator/unknown no-op is already test-covered
  (`internal/tui/main_menu_test.go:188-233`).
- [ ] **Tests** — add focused tests in the relevant `_test.go` files:
  - Pin: assert `buildConfirm()` with zero diff transitions to `pinDone` and
    `viewDone()` content contains "No changes" (mirror existing pin tests in
    `internal/tui/pin_test.go` if present).
  - Deploy: assert `startDeploy()` with empty `ds.selected` sets `step ==
    deployResult` and `View().Content` contains "No assets selected".
  - Remove: assert `updateSelectAssets` (form completed, empty `m.selected`)
    sets `step == removeResult` and renders "No assets selected".
  - Export test (`internal/tui/main_menu_test.go:126-139`): only touch if
    `47kdob` has not already rewritten it.
- [ ] `go build -o nd .`; `go test ./internal/tui/... -count=1`;
  `go test ./... -count=1`; `golangci-lint run ./internal/tui/...`.
- [ ] Manual TUI verification of all six repro paths.

### Acceptance criteria

- Export "Export plugin" produces a visible notice screen, not a silent
  `BackMsg` reset (delivered by dependency `47kdob`; verified here).
- Ctrl+S toggle toward project scope with no resolvable project root shows a
  visible message referencing missing `.git/`/`.claude/`, not silence
  (layered on dependency `k63tsg`).
- Pin no-change path lands on `pinDone` and shows "No changes made."
- Deploy no-selection path lands on `deployResult` and shows "No assets
  selected."
- Remove no-selection path lands on `removeResult` and shows "No assets
  selected." (non-error styling).
- `firstrun.go` / `main_menu.go` `default:` branches each have a comment
  documenting the deliberate no-op; no behavior change; existing separator
  tests still pass.
- New tests for the pin/deploy/remove paths exist and pass; no new global
  message type was introduced.
- `go build -o nd .` succeeds; `go test ./... -count=1` and
  `golangci-lint run ./internal/tui/...` pass with no regressions.

### References

- GitHub issue: https://GitHub.com/armstrongl/nd/issues/131
- Close this issue when the task is completed.
- Seed/dependency `47kdob` (export fix + message-screen pattern):
  `tasks/cli/47kdob-fix-handle-selection-export.md`. Root site:
  `internal/tui/main_menu.go:127-129`.
- Dependency `k63tsg` (on-demand project-root resolution):
  `tasks/tui/k63tsg-expand-project-root-resolution.md`; itself seeded by
  `2r0nyd` (`tasks/tui/2r0nyd-tui-project-scope-switch.md`).
- Silent sites: `internal/tui/tui.go:166-169` (toggleScope; bound at
  `internal/tui/tui.go:138-139`); `internal/tui/pin.go:198-201`
  (`buildConfirm`); `internal/tui/deploy.go:431-434` (`startDeploy`);
  `internal/tui/remove.go:257-260` (`updateSelectAssets`).
- Patterns to mirror: in-screen result/info — `internal/tui/deploy.go:238`,
  `:270-274`; `internal/tui/pin.go:278`, `:288` (`viewDone`/`updateDone`);
  message screen — `internal/tui/scope.go:61-99`.
- Default no-ops: `internal/tui/firstrun.go:92-102`;
  `internal/tui/main_menu.go:8-13`, `:132-133`; tests
  `internal/tui/main_menu_test.go:188-233`.
- Navigation messages / no `StatusMsg`: `internal/tui/screens.go:14-28`.
- Copy + style conventions: `internal/tui/empty.go` (esp. `:19-21`, `:35-37`);
  `internal/tui/theme.go:29`, `:42` (`Styles.Subtle`).
