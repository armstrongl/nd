---
title: "Additional test coverage gaps beyond TUI audit"
id: "wughc0"
status: pending
priority: low
type: chore
tags: ["testing"]
created_at: "2026-05-17"
dependencies: ["agtaqm"]
---

## Additional test coverage gaps beyond TUI audit

### Objective

Pattern expansion of seed agtaqm, which is scoped exclusively to `internal/tui/` screens (32 enumerated gaps). This task covers the SAME gap classes (error paths, nil/empty-input guards, dry-run no-side-effect, OpLog side effects, symlink target/cleanup) in NON-TUI packages so the codebase coverage is consistent. None of the below duplicates agtaqm's TUI-only items.

### Tasks

#### Error-path gaps (engine / store / git / source-purge)

- [ ] `internal/deploy/deploy.go:186,501,521,548` -- Save-error returns; set the unused `mockStore.saveErr` (`internal/deploy/deploy_test.go:25`)
- [ ] `internal/state/store.go:88-90,92-95,102-104` -- `Save` MkdirAll/Marshal failures and `WithLock` acquire-failure branch
- [ ] `internal/sourcemanager/git.go:62-64,72-74` -- `gitClone`/`gitPull` error branches (only happy paths tested)
- [ ] `internal/sourcemanager/register.go:154-156,170-174` -- `AddGit` clone-failure cleanup and WriteConfig-failure rollback
- [ ] `cmd/source.go:342-374` -- `removeSourceDeployments` (0% covered): status-error, `RemoveBulk` error, partial-failure branches

#### Nil/empty-input guard gaps

- [ ] `internal/deploy/deploy.go:558-578` -- `SetOrigin` is 0% covered (not-found, scope-mismatch, match+Save, load-error)
- [ ] `internal/deploy/deploy.go:596-602` -- `removeOne` agent-filtering branch (`req.Agent != ""`)
- [ ] `internal/config/validation.go:71-74` -- empty source `ID` branch; `:21` -- `ValidationError.Error()` no-file branch

#### Dry-run no-side-effect gaps (cmd layer)

- [ ] `cmd/deploy_test.go:96`, `remove_test.go:60`, `snapshot_test.go:250`, `sync_test.go:33`, `uninstall_test.go:12`, `profile_test.go:273` -- assert no symlink/state/config change after dry-run (not just output text); add a profile-switch dry-run test (`cmd/profile.go:474-484`)

#### OpLog side-effect gaps (cmd layer)

- [ ] Assert oplog entries for `OpProfileSwitch` (`cmd/profile.go:548`), profile-deploy `OpDeploy` (`:373`), `OpSnapshotRestore` (`cmd/snapshot.go:196`), `OpSourceAdd` (`cmd/source.go:82,105`), `OpSourceRemove` (`:239`), `OpSourceSync` (`cmd/sync.go:50`), `OpUninstall` (`cmd/uninstall.go:92`); add dry-run no-log assertions for remove/snapshot-restore/profile-deploy/sync/uninstall

#### Symlink target/cleanup gaps

- [ ] `internal/deploy/deploy.go:256-258` -- relative-symlink `filepath.Rel` error branch
- [ ] `cmd/source.go:201,213,342` -- deploy then `source remove --purge`: verify symlinks removed and state entries gone
- [ ] `internal/deploy/deploy.go:372-374,390-392` -- `handleConflict` ForceReplace remove-error branches

### Acceptance criteria

- Each listed file:line is exercised by a new failing/erroring or side-effect-asserting test
- `mockStore.saveErr` is set by at least one test per Save call site; `SetOrigin` coverage rises from 0%
- `go test ./internal/deploy/... ./internal/state/... ./internal/sourcemanager/... ./internal/config/... ./cmd/...` passes
- No agtaqm TUI item is duplicated

### References

- Seed task: agtaqm -- `tasks/cli/agtaqm-test-coverage-gaps.md` (extends, does not duplicate; agtaqm is TUI-only)
- `internal/deploy/deploy.go`, `internal/state/store.go`, `internal/sourcemanager/`, `cmd/oplog_integration_test.go`
