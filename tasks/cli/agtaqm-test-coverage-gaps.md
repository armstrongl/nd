---
title: "Address test coverage gaps from TUI audit"
id: "agtaqm"
status: pending
priority: low
type: chore
tags: ["testing"]
created_at: "2026-04-20"
---

## Address test coverage gaps from TUI audit

### Objective

Address 32 test coverage recommendations identified during the TUI Phase 4-6 audit. Key gaps span `StateAborted` handling, OpLog integration, dry-run behavior, nil `DeployEngine` safety, and symlink strategy verification across most TUI screens.

### Tasks

#### StateAborted tests

- [ ] Add `StateAborted` transition tests for the deploy screen (`internal/tui/deploy.go`)
- [ ] Add `StateAborted` transition tests for the remove screen (`internal/tui/remove.go`)
- [ ] Add `StateAborted` transition tests for the status screen (`internal/tui/status.go`)
- [ ] Add `StateAborted` transition tests for the doctor screen (`internal/tui/doctor.go`)
- [ ] Add `StateAborted` transition tests for the browse screen (`internal/tui/browse.go`)
- [ ] Add `StateAborted` transition tests for the profile screen (`internal/tui/profile.go`)
- [ ] Add `StateAborted` transition tests for the snapshot screen (`internal/tui/snapshot.go`)
- [ ] Add `StateAborted` transition tests for the source screen (`internal/tui/source.go`)
- [ ] Add `StateAborted` transition tests for the pin screen (`internal/tui/pin.go`)
- [ ] Add `StateAborted` transition tests for the settings screen (`internal/tui/settings.go`)

#### OpLog integration tests

- [ ] Add OpLog recording tests for deploy operations
- [ ] Add OpLog recording tests for remove operations
- [ ] Add OpLog recording tests for profile deploy operations
- [ ] Add OpLog recording tests for snapshot restore operations
- [ ] Verify OpLog entries contain correct timestamps and operation metadata

#### Dry-run behavior tests

- [ ] Add dry-run mode tests for the deploy screen (no filesystem changes)
- [ ] Add dry-run mode tests for the remove screen (no filesystem changes)
- [ ] Add dry-run mode tests for bulk operations (profile deploy, snapshot restore)
- [ ] Verify dry-run output matches expected preview format

#### Nil DeployEngine safety tests

- [ ] Add nil `DeployEngine` guard tests for deploy screen initialization
- [ ] Add nil `DeployEngine` guard tests for remove screen initialization
- [ ] Add nil `DeployEngine` guard tests for profile deploy flow
- [ ] Verify graceful error messages when `DeployEngine` is nil

#### Symlink strategy tests

- [ ] Add symlink creation verification tests for single-asset deploy
- [ ] Add symlink target resolution tests (correct source path)
- [ ] Add stale symlink detection tests for doctor screen
- [ ] Add symlink conflict handling tests (existing file at target path)

#### Miscellaneous coverage gaps

- [ ] Add filter input edge case tests for browse and status screens
- [ ] Add scroll boundary tests for screens using `RenderScrolledLines`
- [ ] Add empty state rendering tests for screens with no data

### Acceptance criteria

- All 32 identified test gaps have corresponding test functions
- All new tests follow existing patterns in `internal/tui/*_test.go` (use `testutil_test.go` helpers)
- `go test ./internal/tui/...` passes with all new tests
- No existing tests broken by additions
- Test coverage for `internal/tui/` increases measurably (run `go test -cover` before and after)

### References

- Audit report: `.claude/reports/2026-03-23-tui-phases4-6-audit.md`
- TUI test utilities: `internal/tui/testutil_test.go`
- Shared list rendering: `internal/tui/listview.go` and `internal/tui/listview_test.go`
