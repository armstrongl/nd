---
title: "Fix handleSelection missing export case"
id: "47kdob"
status: pending
priority: medium
type: bug
tags: ["tui"]
created_at: "2026-04-20"
context:
  - "internal/tui/main_menu.go"
  - "internal/tui/main_menu_test.go"
  - "internal/tui/scope.go"
  - "internal/tui/screens.go"
  - "internal/tui/browse.go"
  - "internal/tui/theme.go"
  - "cmd/export.go"
  - "cmd/root.go"
verify:
  - type: bash
    run: "go build -o nd ."
  - type: bash
    run: "go test ./internal/tui/... -count=1"
  - type: bash
    run: "go test ./... -count=1"
  - type: assert
    check: "Selecting \"Export plugin\" in the TUI main menu shows a visible notice telling the user to run `nd export` on the command line, then returns to the main menu on Enter (no silent reset)"
---

## Fix handleSelection missing export case

### Objective

When a user picks "Export plugin" from the `nd` TUI main menu, nothing visible
happens — the menu silently snaps back to itself with no screen, message, or
error. Export has no TUI screen (it is a CLI-only command, `nd export`), so the
fix is to give the user clear feedback: show a notice explaining that export is
available via `nd export` on the command line, then return to the main menu.
This removes a dead-end interaction and is the seed pattern for follow-up task
`dze4e7` (expand silent TUI handlers), which depends on this task.

### Background (verified against the codebase)

- The TUI launches from the bare `nd` command: `cmd/root.go:47` calls
  `tui.Run(app)` when `isTerminal()` is true and no conflicting flags are set.
- The main menu offers "Export plugin" → choice value `"export"`:
  `internal/tui/main_menu.go:48` (`huh.NewOption("Export plugin", "export")`).
- `nd export` is a real Cobra command (`cmd/export.go:20`, `newExportCmd`). It
  requires `--name` and `--assets` for non-interactive use, or runs an
  interactive huh flow when invoked in a terminal. It has no in-TUI equivalent.

### Root cause (exact location)

`internal/tui/main_menu.go:127-129`, inside `(*mainMenuScreen).handleSelection`:

```go
case "export":
	// Export has no TUI screen yet — return to the main menu.
	return func() tea.Msg { return BackMsg{} }
```

`BackMsg{}` pops the navigation stack (`internal/tui/screens.go:18`). Because
the main menu is the root screen, popping it just re-renders the menu with no
user-visible change. `internal/tui/main_menu_test.go:126-139`
(`TestMainMenu_HandleSelectionExport`) currently asserts exactly this
(`BackMsg`), so it locks in the bad behavior.

Note: there is **no** `StatusMsg` / `ToastMsg` type in the codebase. The only
navigation messages are in `internal/tui/screens.go:15-28`: `NavigateMsg`,
`BackMsg`, `PopToRootMsg`, `RefreshHeaderMsg`, `ScopeSwitchedMsg`. Feedback must
be delivered via a screen, not a new global message type.

### Steps to reproduce

1. Build: `go build -o nd .`
2. Run the bare TUI in a terminal: `./nd`
3. Arrow down to "Export plugin" under the "── Manage ──" group, press Enter.
4. Observe: the menu silently re-renders. No screen, no message, no error.

### Chosen approach

Add a minimal notice screen and navigate to it (do **not** invent a new global
message type). Mirror the proven two-step "show message, return on Enter"
pattern in `internal/tui/scope.go` (the `scopeShowError` step at
`internal/tui/scope.go:62-69` and `internal/tui/scope.go:90-99`), which renders
a message and emits `PopToRootMsg{}` on Enter to return to the main menu.

Reuse the existing empty-state helper convention in `internal/tui/empty.go`
(e.g. `NoSources()`, `NothingDeployed()` — short string + actionable hint, two
spaces of indent) for the notice copy. Style the message body with
`s.styles.Subtle` (defined at `internal/tui/theme.go:29` / set at
`internal/tui/theme.go:42`), matching how `scope.go` renders its
"Press enter to return." hint.

### Tasks

- [ ] Add an export-notice helper to `internal/tui/empty.go` (follow the
  existing function style, e.g. `NothingDeployed()` at
  `internal/tui/empty.go:19-21`):
  `func ExportCLIOnly() string` returning roughly
  `"Export is a command-line action.\n\n  Run nd export --name <plugin> --assets <type>/<name> to export assets as a Claude Code plugin."`
  (keep wording consistent with `cmd/export.go` usage/`Example` at
  `cmd/export.go:41-45`).
- [ ] Create `internal/tui/export.go` defining an `exportScreen` that satisfies
  the `Screen` interface (`internal/tui/screens.go:6-10`: `tea.Model` +
  `Title() string` + `InputActive() bool`). Model it on the no-form variant of
  `scope.go`'s error step:
  - `newExportScreen(svc Services, styles Styles, isDark bool) *exportScreen`
    constructor (same signature as `newScopeScreen`,
    `internal/tui/scope.go:29`).
  - `Title()` returns `"Export"`.
  - `InputActive()` returns `false` (no text input on this screen).
  - `Init()` returns `nil`.
  - `Update(msg)`: on `tea.KeyPressMsg` with `String() == "enter"` (or
    `"esc"`), return `func() tea.Msg { return PopToRootMsg{} }`; otherwise
    return `s, nil`. Mirror `internal/tui/scope.go:62-69`.
  - `View()` returns
    `tea.NewView(fmt.Sprintf("  %s\n\n  %s", ExportCLIOnly(), s.styles.Subtle.Render("Press enter to return.")))`,
    mirroring `internal/tui/scope.go:90-99`.
- [ ] Replace the `case "export":` branch in
  `internal/tui/main_menu.go:127-129` so it sets
  `screen = newExportScreen(m.svc, m.styles, m.isDark)` (drop the early
  `return func() tea.Msg { return BackMsg{} }`), letting the existing
  `return func() tea.Msg { return NavigateMsg{Screen: screen} }` at
  `internal/tui/main_menu.go:135` fire — exactly like the other wired cases
  (`internal/tui/main_menu.go:105-126`).
- [ ] Rewrite `TestMainMenu_HandleSelectionExport` in
  `internal/tui/main_menu_test.go:126-139` to assert `handleSelection()` for
  `choice == "export"` returns a non-nil cmd whose message is a `NavigateMsg`
  carrying an `*exportScreen` (assert the message type is `NavigateMsg` and
  `msg.Screen.Title() == "Export"`). Mirror `TestMainMenu_HandleSelectionWiredScreens`
  at `internal/tui/main_menu_test.go:103-124`.
- [ ] Optionally also add `"export"` to the `wiredChoices` slice in
  `TestMainMenu_HandleSelectionWiredScreens`
  (`internal/tui/main_menu_test.go:108-111`) since it is now a wired screen.
- [ ] Add a focused test for `exportScreen` in a new
  `internal/tui/export_test.go` (mirror `internal/tui/scope_test.go` if
  present): construct via `newExportScreen`, assert `Title() == "Export"`,
  `InputActive() == false`, `View().Content` is non-empty and contains
  `"nd export"`, and that `Update` with an Enter `tea.KeyPressMsg` returns a
  cmd producing `PopToRootMsg{}`.
- [ ] `go build -o nd .`
- [ ] `go test ./internal/tui/... -count=1` and `go test ./... -count=1`
- [ ] Manually verify: `./nd` → "Export plugin" → notice screen appears →
  Enter returns to the main menu (menu stays responsive).

### Acceptance criteria

- Selecting "Export plugin" in the TUI main menu navigates to a visible notice
  screen (no silent `BackMsg` reset). The notice tells the user to run
  `nd export` on the command line and mentions the `--name` / `--assets` flags.
- Pressing Enter (or Esc) on the notice screen returns to the main menu and the
  menu remains responsive (`navigated` does not stay stuck — `PopToRootMsg`
  resets the stack to root).
- `internal/tui/export.go` defines `exportScreen` satisfying the `Screen`
  interface; `Title()` returns `"Export"`.
- `TestMainMenu_HandleSelectionExport` asserts a `NavigateMsg` to the export
  screen (not `BackMsg`); a dedicated `exportScreen` test exists.
- `go build -o nd .` succeeds; `go test ./internal/tui/... -count=1` and
  `go test ./... -count=1` pass with no regressions.

### References

- Root cause: `internal/tui/main_menu.go:127-129`
  (`(*mainMenuScreen).handleSelection`, full func `internal/tui/main_menu.go:102-136`).
- Stale test to rewrite: `internal/tui/main_menu_test.go:126-139`
  (`TestMainMenu_HandleSelectionExport`); wired-screen test to mirror:
  `internal/tui/main_menu_test.go:103-124`.
- Pattern to mirror (message screen + return on Enter): `internal/tui/scope.go`
  (`scopeShowError` step, `internal/tui/scope.go:62-69` and `:90-99`).
- Screen interface + nav messages: `internal/tui/screens.go:6-28`
  (no `StatusMsg` exists — use a screen).
- Empty-state copy convention: `internal/tui/empty.go` (e.g. `:19-21`).
- Notice styling: `internal/tui/theme.go:29` / `:42` (`Styles.Subtle`).
- CLI command this notice points users to: `cmd/export.go:20`
  (`newExportCmd`), usage/example `cmd/export.go:37-45`.
- TUI entry point: `cmd/root.go:47` (`tui.Run(app)`).
- Follow-up that depends on this seed: `tasks/cli/dze4e7-expand-silent-tui-handlers.md`
  (the "Project memory note: File ISSUE-007 …" referenced previously resolves
  to this pattern-expansion task; `dze4e7` lists `47kdob` as a dependency).
