---
title: "Implement HelpProvider on TUI screens missing it"
id: "vrs2d7"
status: pending
priority: medium
type: feature
tags: ["tui", "ux"]
created_at: "2026-05-17"
dependencies: ["6bije3"]
---

## Implement HelpProvider on TUI screens missing it

### Objective

Pattern expansion of seed 6bije3. Only 4 of 13 `Screen` types (`deploy`, `browse`, `remove`, `status`) implement `HelpProvider`/`FullHelpProvider` (`internal/tui/helpbar.go`). The other 9 fall back to a generic, often-wrong help bar -- e.g. "enter select" shown on MultiSelect/Confirm and text-input steps. Implement step-aware `FullHelpProvider` on the 9 missing screens. Note the seed text wrongly lists `deploy` as missing; the real gaps are below.

### Tasks

- [ ] `internal/tui/main_menu.go` -- add `FullHelpItems` (j/k navigate, enter select, q quit)
- [ ] `internal/tui/firstrun.go` -- add `FullHelpItems` for the welcome form
- [ ] `internal/tui/scope.go` -- add `FullHelpItems`; distinguish `scopeFormStep` vs `scopeShowError` (enter to return)
- [ ] `internal/tui/settings.go` -- add `FullHelpItems` per step (menu / scope form / result)
- [ ] `internal/tui/profile.go` -- add `FullHelpItems` per step (menu / list scroll / switch / create input / done)
- [ ] `internal/tui/snapshot.go` -- add `FullHelpItems` per step (menu / save input / restore select / confirm / list / done)
- [ ] `internal/tui/pin.go` -- add `FullHelpItems` per step (select x/space toggle, confirm h/l, done enter)
- [ ] `internal/tui/source.go` -- add `FullHelpItems` per step (menu / list / add input / remove select / confirm / done)
- [ ] `internal/tui/doctor.go` -- add `FullHelpItems` per step (confirm h/l + j/k scroll, done enter)
- [ ] Add a test asserting every constructed `Screen` implements `HelpProvider` or `FullHelpProvider`
- [ ] `go test ./internal/tui/... -run TestHelp`

### Acceptance criteria

- All 13 `Screen` types implement `HelpProvider` or `FullHelpProvider`
- Help bar text matches the actual valid keys for the current step (no "enter select" on MultiSelect/Confirm/input)
- `helpbar_test.go` and existing tests pass

### References

- Seed task: 6bije3 -- `tasks/cli/6bije3-tui-help-instructions.md`
- `internal/tui/helpbar.go` (interfaces); `internal/tui/deploy.go`, `status.go` (reference implementations)
