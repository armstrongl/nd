---
title: "Fix handleSelection missing export case"
id: "47kdob"
status: pending
priority: medium
type: bug
tags: ["tui"]
created_at: "2026-04-20"
---

## Fix handleSelection missing export case

### Objective

The `handleSelection` function in `internal/tui/main_menu.go` has a `case "export":` branch, but it only returns a `BackMsg{}`, silently bouncing the user back to the main menu with no feedback. When a user selects "Export plugin" from the TUI main menu, nothing visible happens. The export case needs either a dedicated TUI screen (`newExportScreen`) or, at minimum, a user-facing message explaining that export is a CLI-only command (`nd export`).

### Steps to reproduce

1. Launch the TUI with `nd` (bare command).
2. Navigate to "Export plugin" in the main menu and press Enter.
3. Observe that the menu silently resets with no navigation, no message, and no error.

### Expected behavior

Selecting "Export plugin" should either navigate to an export screen or display a notice that export is only available via `nd export` on the command line, then return to the main menu.

### Actual behavior

The `case "export":` branch at `internal/tui/main_menu.go:127-129` returns `BackMsg{}`, which resets the menu without any user-visible feedback. The test at `internal/tui/main_menu_test.go:126-139` confirms this: it only asserts a `BackMsg` is returned, not that a screen or notification is shown.

### Tasks

- [ ] Decide on approach: add a `newExportScreen` TUI screen, or show an inline notice/toast that redirects to `nd export`
- [ ] If adding a screen: create `internal/tui/export.go` with a `newExportScreen` constructor following the pattern of other screens
- [ ] If adding a notice: replace the `BackMsg` return with a `StatusMsg` or equivalent that displays "Export is available via `nd export` on the command line"
- [ ] Update the `case "export":` branch in `handleSelection` (`internal/tui/main_menu.go:127-129`)
- [ ] Update `TestMainMenu_HandleSelectionExport` in `internal/tui/main_menu_test.go` to assert the new behavior (screen navigation or status message)
- [ ] Run `go test ./internal/tui/...` and confirm all tests pass
- [ ] Manually verify the fix in the TUI

### Acceptance criteria

- Selecting "Export plugin" from the TUI main menu produces visible user feedback (either a screen or a notice)
- The silent `BackMsg` return is replaced with meaningful behavior
- `TestMainMenu_HandleSelectionExport` is updated to assert the new behavior
- All existing TUI tests continue to pass
- `go test ./...` passes with no regressions

### References

- Bug location: `internal/tui/main_menu.go:102-136` (`handleSelection` function)
- Current test: `internal/tui/main_menu_test.go:126-139` (`TestMainMenu_HandleSelectionExport`)
- Project memory note: "File ISSUE-007 for `handleSelection` missing `case "export":` bug"
