---
title: "Fix profile/snapshot create TOCTOU silent overwrite"
id: "2zlf37"
status: pending
priority: medium
type: bug
tags: ["profile", "concurrency"]
created_at: "2026-05-17"
verify:
  - type: bash
    run: "go build -o nd ."
  - type: bash
    run: "go test ./internal/profile/..."
  - type: bash
    run: "go test -race ./internal/profile/..."
  - type: bash
    run: "go test ./..."
  - type: assert
    check: "Concurrent same-name CreateProfile / SaveSnapshot yields exactly one success; every loser gets a non-nil 'already exists' error and no goroutine silently overwrites another's file."
  - type: assert
    check: "AutoSnapshot returns an error instead of silently overwriting when a snapshot file with the generated name already exists."
context:
  - "internal/profile/store.go"
  - "internal/nd/atomic.go"
  - "internal/state/lock.go"
  - "internal/state/store.go"
  - "internal/profile/store_test.go"
---

## Fix profile/snapshot create TOCTOU silent overwrite

### Objective

`profile.Store.CreateProfile`, `SaveSnapshot`, and `AutoSnapshot`
(`internal/profile/store.go`) persist YAML by calling `nd.AtomicWrite`
(`internal/nd/atomic.go`). `AtomicWrite` ends with an **unconditional**
`os.Rename(tmpPath, path)` (`internal/nd/atomic.go:45`) which silently
overwrites any pre-existing file at `path`. `CreateProfile` and
`SaveSnapshot` try to prevent overwrites with an `os.Stat` "already exists"
check *before* the write, but nothing serializes the gap between that
`Stat` and the `Rename` — this is a time-of-check/time-of-use (TOCTOU)
race. Two concurrent `nd profile create foo` (or `nd snapshot save foo`)
invocations can both pass the `Stat` (file absent), then both `Rename`;
the second clobbers the first with no "already exists" error returned to
the loser. `AutoSnapshot` skips the existence check entirely, so a
name collision (e.g. clock not strictly monotonic, or two auto-snapshots
in the same nanosecond) silently overwrites.

Unlike deployment-state writes, profile/snapshot writes take no lock. The
deployment `state.Store` already serializes its read-modify-write cycles
via `state.Store.WithLock` (`internal/state/store.go:99-107`), which wraps
an advisory `flock`-based `state.FileLock` (`internal/state/lock.go`). The
fix should give profile/snapshot writes equivalent serialization plus a
real exclusive-create guarantee so the loser of a race reliably gets the
existing "already exists" error and `AutoSnapshot` never silently
overwrites.

Why it matters: profiles/snapshots are user-authored config. A silent
overwrite loses data with no error, no warning, and no recovery path.

### Steps to reproduce

There is no existing repro test. To reproduce in a new test (see Tasks):

1. Create one `profile.Store` via `profile.NewStore(profilesDir, snapshotsDir)`
   (mirror the `tempDirs(t)` helper at `internal/profile/store_test.go:16-20`).
2. Launch N (e.g. 8) goroutines that each call `store.CreateProfile(p)` with
   the **same** `p.Name` (build a valid `profile.Profile` like
   `TestStoreCreateProfile` at `internal/profile/store_test.go:22-45`).
   Collect every returned error.
3. Observed (bug): more than one goroutine returns `nil` (multiple
   "successes"), and/or some racing writer's file content is lost.
   Expected (fixed): exactly one `nil`; every other call returns a non-nil
   error whose message contains `already exists`.
4. For `AutoSnapshot`: pre-write a snapshot file at the path
   `s.snapshotPath(name, true)` would resolve to, then call `AutoSnapshot`
   with that same name forced — it currently overwrites silently.

### Root cause locations (verified against current code)

- `internal/profile/store.go:72-74` — `CreateProfile`: `os.Stat(path)`
  existence check, followed by `nd.AtomicWrite(path, data)` at
  `internal/profile/store.go:80`. Gap between the two is unguarded.
- `internal/profile/store.go:219-221` — `SaveSnapshot`: identical
  `os.Stat(path)` check, followed by `nd.AtomicWrite(path, data)` at
  `internal/profile/store.go:227`.
- `internal/profile/store.go:325-333` — `AutoSnapshot`: computes `path`
  via `s.snapshotPath(name, true)` then `nd.AtomicWrite` with **no**
  existence check at all.
- `internal/nd/atomic.go:45` — `os.Rename(tmpPath, path)` is
  unconditional, so it overwrites whatever is at `path`.

### Tasks

- [ ] Add cross-process serialization for profile/snapshot writes,
  mirroring the existing `state` lock pattern. Reuse
  `state.NewFileLock` + `state.FileLock.Acquire(5*time.Second)` /
  `.Release()` (`internal/state/lock.go:21-111`), the same primitive
  `state.Store.WithLock` uses (`internal/state/store.go:99-107`).
  Add a lock-file path to `profile.Store` (e.g. a `lockPath` field set in
  `NewStore` at `internal/profile/store.go:44-49`, such as
  `filepath.Join(profilesDir, ".profile.lock")`), and a private
  `withLock(fn func() error) error` helper on `*Store` that acquires,
  defers release, and runs `fn`. Note: the `profile` package importing
  `internal/state` would create an import cycle only if `state` imports
  `profile` — it does not (verified: `internal/state/*.go` imports
  `internal/nd` only). Importing `state.FileLock` from `profile` is safe.
- [ ] `internal/profile/store.go:58-81` — `CreateProfile`: wrap the
  `os.Stat` existence check + marshal + `nd.AtomicWrite` (currently lines
  72-80) inside `withLock`, so the check-then-write is atomic w.r.t. other
  `nd` processes. Keep the existing error string
  `fmt.Errorf("profile %q already exists", p.Name)` (line 73) so existing
  tests like `TestStoreCreateProfileDuplicate`
  (`internal/profile/store_test.go:46-58`) still pass.
- [ ] `internal/profile/store.go:202-228` — `SaveSnapshot`: wrap the
  `os.Stat` check (line 219) + marshal + `nd.AtomicWrite` (line 227) in
  the same `withLock`. Preserve the existing
  `fmt.Errorf("snapshot %q already exists", snap.Name)` message (line 220);
  `TestStoreSaveSnapshotDuplicate` (`internal/profile/store_test.go:271-280`)
  must still pass.
- [ ] `internal/profile/store.go:304-336` — `AutoSnapshot`: under the same
  lock, add an existence check before `nd.AtomicWrite` (line 331). If
  `os.Stat(path)` succeeds (file exists), return a non-nil error
  (e.g. `fmt.Errorf("auto snapshot %q already exists", name)`) instead of
  overwriting. `AutoSave` (`internal/profile/store.go:413-420`) calls
  `AutoSnapshot` then `PruneAutoSnapshots(5)`; surface the error up through
  `AutoSave` (it already returns the `AutoSnapshot` error at line 416-418).
- [ ] Optional hardening (recommended): in
  `internal/nd/atomic.go`, do not weaken the existing atomic-overwrite
  contract used by `state.Store.Save` (`internal/state/store.go:96`, which
  *intends* to overwrite). Do NOT change `AtomicWrite` to fail on existing
  targets — `state.Store.Save` and `UpdateProfile`
  (`internal/profile/store.go:160-178`) legitimately overwrite. The guard
  must live in the profile-store callers, not in `AtomicWrite`.
- [ ] Add a regression test in `internal/profile/store_test.go` (package
  `profile_test`). Pattern: build a valid `profile.Profile` (copy from
  `TestStoreCreateProfile`, `internal/profile/store_test.go:22-45`), spawn
  N goroutines calling `store.CreateProfile` with the same name, use
  `sync.WaitGroup`, collect errors into a slice guarded by a `sync.Mutex`,
  assert exactly one `nil` and that every other error string contains
  `already exists`. Add an analogous test for `SaveSnapshot`. Add a test
  asserting `AutoSnapshot` returns a non-nil error when its target path
  already has a file. The test must pass under `go test -race`.

### Acceptance criteria

- `go build -o nd .` succeeds.
- `go test ./internal/profile/...` and `go test -race ./internal/profile/...`
  pass, including the new concurrency regression tests.
- `go test ./...` passes (no regressions; existing duplicate-name tests
  `TestStoreCreateProfileDuplicate` and `TestStoreSaveSnapshotDuplicate`
  still pass with their original error messages).
- A concurrency test with ≥8 goroutines doing same-name `CreateProfile`
  (and separately `SaveSnapshot`) yields exactly one `nil` result; every
  other result is a non-nil error containing `already exists`; no file
  content is lost.
- `AutoSnapshot` returns a non-nil error (does not overwrite) when a file
  already exists at its computed snapshot path.
- No change to `internal/nd/atomic.go`'s overwrite semantics; the guard is
  implemented in the `profile.Store` callers.

### References

- Bug type: net-new, found during codebase sweep — no seed file or
  numbered seed pattern exists; do not search for one.
- `internal/profile/store.go:58-81` — `CreateProfile` (TOCTOU site).
- `internal/profile/store.go:202-228` — `SaveSnapshot` (TOCTOU site).
- `internal/profile/store.go:304-336` — `AutoSnapshot` (no check at all).
- `internal/profile/store.go:44-49` — `NewStore` (add `lockPath` here).
- `internal/nd/atomic.go:11-51` — `AtomicWrite`; unconditional
  `os.Rename` at line 45. Header comment references NFR-010 (the
  crash-safety non-functional requirement: writes must be atomic so a
  crash mid-write never leaves a partial file). Do not break this.
- `internal/state/lock.go:12-111` — `state.FileLock`: `NewFileLock`,
  `Acquire(timeout)` (advisory `flock` via `syscall.Flock`, 5s default,
  60s stale-lock break), `Release()`. Reuse this; do not write a new lock.
- `internal/state/store.go:99-107` — `state.Store.WithLock`: the exact
  acquire/defer-release/run-fn pattern to mirror.
- `internal/nd/errors.go:16-28` — `nd.LockError` (NFR-011: returned when a
  lock cannot be acquired within the timeout). `Acquire` returns this; the
  helper should propagate it unchanged.
- `internal/profile/store_test.go:16-20` — `tempDirs(t)` helper.
- `internal/profile/store_test.go:22-58` — `TestStoreCreateProfile` /
  `TestStoreCreateProfileDuplicate` (valid `Profile` construction +
  duplicate-error expectation to keep green).
- `internal/profile/store_test.go:238-280` — snapshot save + duplicate
  tests to keep green.
- `cmd/app.go:130-140` — `App.ProfileStore()` shows the real lock-dir
  location at runtime (`<configDir>/profiles`, `<configDir>/snapshots`);
  pick a lock path that lives under one of these existing dirs.
