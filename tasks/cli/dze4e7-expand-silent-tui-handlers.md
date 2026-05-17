---
title: "Fix silent TUI handlers across screens"
id: "dze4e7"
status: pending
priority: medium
type: bug
tags: ["tui"]
created_at: "2026-05-17"
dependencies: ["47kdob", "k63tsg"]
---

## Fix silent TUI handlers across screens

### Objective

Pattern expansion of seed 47kdob (`case "export":` returns bare `BackMsg{}` with no feedback). The same anti-pattern -- a handler returning `BackMsg{}`/`PopToRootMsg{}`/`nil` with zero user-visible feedback when the user expected an action -- occurs in several other TUI screens. Each path should produce a clear notice, result, or status message.

### Tasks

- [ ] `internal/tui/main_menu.go:127-129` -- `case "export":` bare `BackMsg`: show a notice ("Export is available via `nd export`") instead of silent reset (the seed)
- [ ] `internal/tui/tui.go:166-169` -- `toggleScope()` Ctrl+S: when project root empty it `return m, nil` silently; surface a message (coordinate with seed 2r0nyd / task k63tsg)
- [ ] `internal/tui/pin.go:198-201` -- `buildConfirm`: when `newPins == 0 && newUnpins == 0` returns `BackMsg` with no message; show "No pin changes detected"
- [ ] `internal/tui/deploy.go:431-434` -- `startDeploy`: `len(ds.selected) == 0` silently bounces; show "No assets selected"
- [ ] `internal/tui/remove.go:257-260` -- `updateSelectAssets`: `len(m.selected) == 0` silently bounces; show "No assets selected"
- [ ] Review `internal/tui/firstrun.go:99-101` and `internal/tui/main_menu.go:132-133` default branches; guard or document why they are safe no-ops (separator sentinels)
- [ ] Extend `internal/tui/main_menu_test.go` `TestMainMenu_HandleSelectionExport` and add tests asserting a notice/message for each path
- [ ] `go test ./internal/tui/...` and manual TUI verification

### Acceptance criteria

- Selecting "Export plugin" produces visible feedback (screen or notice)
- Ctrl+S toggle to project scope without a root shows a message, not silence
- Pin / deploy / remove "no selection / no change" paths show a clear message
- Default branches are guarded or documented
- All existing TUI tests pass with no regressions

### References

- Seed task: 47kdob -- `tasks/cli/47kdob-fix-handle-selection-export.md`
- `internal/tui/main_menu.go`, `tui.go`, `pin.go`, `deploy.go`, `remove.go`
- Related: k63tsg (project-root resolution)
