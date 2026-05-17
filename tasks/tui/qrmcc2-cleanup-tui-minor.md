---
title: "TUI minor cleanups for dead code and unsurfaced sync error"
id: "qrmcc2"
status: pending
priority: low
type: chore
tags: ["tui"]
created_at: "2026-05-17"
---

## TUI minor cleanups

### Objective

Net-new low-severity quality issues (no seed; found during the codebase sweep). Dead/misleading code and one inconsistent error-surfacing path in the TUI.

### Tasks

- [ ] `internal/tui/header.go:34` -- `leftStyled := left` is assigned the raw string and never styled (only `rightStyled` is); remove the misleading `_Styled` naming/dead assignment or actually style it
- [ ] `internal/tui/source.go:434,437,507-519` -- `startSync` errors are surfaced only via `doneMsg` and never set the screen-level `s.err`, so the `View()` `s.err != nil` branch is dead; make sync errors surface consistently with how other screens show load errors

### Acceptance criteria

- No dead/misleading `leftStyled` assignment in `header.go`
- Source sync errors surface consistently with other screens' error display
- No regression in existing TUI tests

### References

- net-new, no seed pattern
- `internal/tui/header.go:28-48`, `internal/tui/source.go:434-519`
