---
title: "Prevent state-file data loss on corrupt-rename and backup collision"
id: "fpff5s"
status: pending
priority: high
type: bug
tags: ["state", "data-loss"]
created_at: "2026-05-17"
context:
  - internal/state/store.go
  - internal/state/state.go
  - internal/deploy/deploy.go
  - internal/state/store_test.go
  - internal/deploy/deploy_test.go
  - internal/profile/profile.go
  - internal/nd/scope.go
verify:
  - type: bash
    run: "go build -o nd ."
  - type: bash
    run: "go test ./internal/state/... ./internal/deploy/..."
  - type: bash
    run: "go test -race ./..."
  - type: assert
    check: "When os.Rename in handleCorrupt fails, Load() returns an error (or a sentinel the caller honors) so Store.Save() never overwrites the still-present corrupt original with empty state"
  - type: assert
    check: "Two existing files with the same basename backed up within a single DeployBulk call produce two distinct backup paths and both survive on disk"
  - type: assert
    check: "DeploymentState.Validate() returns non-empty errors for duplicate identities or an invalid scope, and Store.Load() surfaces that as an error"
---

## Prevent state-file data loss on corrupt-rename and backup collision

### Objective

Three defects in the deployment-state layer can silently destroy user data. Fix all three so that a corrupt or partially-failed state operation never loses the user's only recovery source, concurrent same-name backups don't clobber each other, and structurally-invalid (but YAML-parseable) state is rejected instead of silently accepted.

Net-new bug found during a codebase sweep; there is no seed file or prior conversation context — everything needed is below.

This is a Go module (`github.com/armstrongl/nd`, Go 1.25). There is no Makefile; build with `go build -o nd .` and test with `go test ./...`.

### Background: how the data loss happens

`Store.Load()` (`internal/state/store.go:31-57`) reads `deployments.yaml`. If `yaml.Unmarshal` fails it calls `handleCorrupt` (`internal/state/store.go:74-84`):

```go
func (s *Store) handleCorrupt(_ error) (*DeploymentState, []string, error) {
	ts := time.Now().Format("2006-01-02T15-04-05")
	corruptPath := fmt.Sprintf("%s.corrupt.%s", s.path, ts)
	os.Rename(s.path, corruptPath)               // store.go:77 — error DISCARDED
	warning := fmt.Sprintf("Warning: deployments.yaml was corrupted and has been renamed to %s. ...", filepath.Base(corruptPath))
	return &DeploymentState{Version: nd.SchemaVersion}, []string{warning}, nil  // returns nil error
}
```

Every state-mutating caller does `st, _, err := e.store.Load()` then proceeds and calls `e.store.Save(st)` on success. Examples: `Engine.DeployBulk` (`internal/deploy/deploy.go:466-507`, Load at `:470`, Save at `:501`), `Engine.Deploy` (`:171-193`, Load at `:175`), `Engine.Remove` (`:510-523`, Load at `:512`, Save at `:521`), `Engine.RemoveBulk` (`:526`+), plus `internal/profile/manager.go` and `cmd/*.go`. `Save` calls `nd.AtomicWrite` (`internal/nd/atomic.go:11`) which write-temp-then-renames over `s.path`.

Failure sequence: if `os.Rename(s.path, corruptPath)` fails (e.g. corrupt-rename target dir not writable, cross-device, perms), `handleCorrupt` still returns empty state with a `nil` error and a warning that *claims* the file was renamed. The original corrupt-but-recoverable `deployments.yaml` is still on disk. The caller proceeds and `Save()` atomically overwrites it with empty state. The only recovery source is destroyed and the warning is a lie.

### Steps to reproduce

1. Write garbage into `deployments.yaml` so `yaml.Unmarshal` fails (e.g. `{{{{not yaml`).
2. Make the rename inside `handleCorrupt` fail. In a unit test this is done by injecting a failing rename; in the field this happens when the state directory is read-only to the process but the file content was already mangled.
3. Run any state-mutating command (e.g. a deploy). `Load()` returns empty state + the (false) "renamed" warning + `nil` error; the command succeeds and `Save()` overwrites the still-corrupt original with empty state. User's deployment state is permanently lost with no `.corrupt.*` copy.

Confirm the second-resolution backup collision separately:

4. In one `Engine.DeployBulk` call, deploy two context assets whose existing on-disk files share a basename (e.g. two `CLAUDE.md` files in different scopes/dirs). With `e.now()` returning the same wall-clock second for both, `backupExistingFile` builds the same `<base>.<ts>.bak` path twice; the second `e.rename` overwrites the first backup. One original file is unrecoverable.

### Root causes (verified file:line)

1. `internal/state/store.go:77` — `os.Rename(s.path, corruptPath)` return value is discarded. `handleCorrupt` (`:74-84`) returns `nil` error regardless, so callers `Save()` over the still-present original.
2. `internal/deploy/deploy.go:428` — in `backupExistingFile` (`:420-439`), `ts := e.now().Format("2006-01-02T15-04-05")` has 1-second resolution; `backupName := fmt.Sprintf("%s.%s.bak", base, ts)` (`:429`) + `backupPath := filepath.Join(e.backupDir, backupName)` (`:430`) collide for same-basename files backed up in the same second within one `DeployBulk` (`:466-507`). `e.rename(path, backupPath)` (`:432`) silently clobbers the prior backup. Note `pruneBackups` (`:442-462`) sorts by filename and relies on the timestamp for chronological ordering, so any uniqueness suffix must still sort chronologically (append after the timestamp, before `.bak`).
3. `internal/state/state.go:44-46` — `DeploymentState.Validate() []error` returns `nil` unconditionally and is **never called anywhere** (no callers in `internal/` or `cmd/`). Corrupt-but-parseable state (duplicate identities, invalid scope) is silently accepted by `Store.Load()`.

### Tasks

- [ ] `internal/state/store.go:77` and `:74-84` — capture the `os.Rename` error in `handleCorrupt`. If the rename fails, do NOT return empty state with a `nil` error; return a non-nil error (e.g. `fmt.Errorf("deployments.yaml is corrupt and could not be quarantined (rename to %s failed: %w); refusing to continue to avoid overwriting it", corruptPath, err)`) so every `st, _, err := store.Load()` caller aborts before `Save()`. Only return empty state + warning + `nil` error when the rename **succeeded**. Update the doc comment on `Load()` (`:28-30`) and `handleCorrupt` (`:73`) to match the new behavior.
- [ ] `internal/deploy/deploy.go:428-430` — make the backup path unique within a process. Add a uniqueness component after the timestamp and before `.bak` (so chronological filename sort in `pruneBackups` at `:442-462` still holds), e.g. an atomic per-`Engine` counter or `e.now().Format("...T15-04-05.000000000")` nanosecond precision, or a collision check that bumps a suffix while `backupPath` already exists (use `e.lstat`/`e.stat`, see `internal/deploy/deploy.go:71-74` for the injectable `lstat`/`stat` hooks). Ensure `pruneBackups`'s `prefix := baseName + "."` match (`:448-454`) still matches the new names.
- [ ] `internal/state/state.go:44-46` — implement `DeploymentState.Validate() []error` to return errors for: (a) duplicate `Deployment.Identity()` (`internal/state/state.go:35-41`, key on `{SourceID, AssetType, AssetName}`); (b) any `Deployment.Scope` not in `{nd.ScopeGlobal, nd.ScopeProject}` (`internal/nd/scope.go:7-8`). Mirror the existing pattern in `internal/profile/profile.go:43-51` (`func (p *Profile) Validate() []error { var errs []error; ...; return errs }`).
- [ ] `internal/state/store.go` — call `st.Validate()` inside `Store.Load()` after a successful unmarshal + version/migrate handling (after `:54`, before `return &st, nil, nil` at `:56`); if it returns errors, return them as a non-nil error so callers don't `Save()` over a now-emptied/invalid in-memory state. Mirror how `internal/profile/store.go:62` and `:164` consume `Validate()` (`if errs := p.Validate(); len(errs) > 0 { return ... errs[0] }`).
- [ ] Regression tests, following existing conventions:
  - `internal/state/store_test.go` (package `state_test`, uses `t.TempDir()`): add a test where the unmarshal fails AND the rename cannot succeed (e.g. point the store at a path whose parent is made read-only, or restructure so the rename target is unwritable) — assert `Load()` returns a non-nil error and the original file content is unchanged on disk. Existing `TestStoreLoadCorruptYAML` (`:64-96`) covers the success path; keep it green.
  - `internal/state/state_test.go` (package `state_test`): add tests asserting `DeploymentState.Validate()` returns errors for duplicate identities and for an invalid `Scope`, and returns no errors for a valid state.
  - `internal/deploy/deploy_test.go` (package `deploy_test`, uses `newMockStore()` at `:29`, `testAgent()`, `engine.SetNow`/`SetRename`/`SetLstat`): add a test that deploys two same-basename context files in one `DeployBulk` with `engine.SetNow` pinned to a constant time, and asserts two distinct `.bak` files exist in `backupDir` and neither original was lost. See `TestDeployContextWithExistingPlainFile` (`:377-429`) and the pruning test around `:763-815` for setup patterns.

### Acceptance criteria

- A failed corrupt-file rename never results in empty state being written over the original: `Load()` returns a non-nil error, callers abort, and the original on-disk bytes are unchanged.
- The corrupt path that previously returned `nil` error still returns empty state + warning + `nil` error **only when the rename succeeded** (existing `TestStoreLoadCorruptYAML` stays green).
- Two same-basename backups created within one `DeployBulk` call produce two distinct backup paths; both files survive on disk; `pruneBackups` still keeps the most recent 5 by chronological filename order.
- `DeploymentState.Validate()` returns non-empty `[]error` for duplicate identities and for an invalid scope, and is invoked by `Store.Load()` such that invalid-but-parseable state causes `Load()` to error rather than silently accepting it and risking a `Save()` overwrite.
- New regression tests cover all three error paths and pass; `go build -o nd .`, `go test ./...`, and `go test -race ./...` are green.

### References

- Root-cause sites: `internal/state/store.go:74-97` (`handleCorrupt`, `Save`), `internal/state/store.go:31-57` (`Load`), `internal/state/state.go:35-46` (`Identity`, `Validate`), `internal/deploy/deploy.go:420-462` (`backupExistingFile`, `pruneBackups`), `internal/deploy/deploy.go:466-507` (`DeployBulk`).
- Save/atomic-write path: `internal/nd/atomic.go:11` (`AtomicWrite`), called from `internal/state/store.go:96`.
- `Validate()` pattern to mirror: `internal/profile/profile.go:43-51`; consumed at `internal/profile/store.go:62` and `:164`.
- Scope enum: `internal/nd/scope.go:7-8` (`ScopeGlobal = "global"`, `ScopeProject = "project"`).
- Test conventions: `internal/state/store_test.go`, `internal/state/state_test.go`, `internal/deploy/deploy_test.go` (mock store at `:20-58`, helpers like `testAgent()`).
- `StateStore` interface (Load/Save/WithLock contract callers depend on): `internal/deploy/deploy.go:17-21`.
- Engine injectable fs hooks for tests/uniqueness checks: `internal/deploy/deploy.go:41-86` (`remove`, `mkdirAll`, `rename`, `now`, `lstat`, `stat`, and their `Set*` methods).
- Net-new, no seed pattern.
