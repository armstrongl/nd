---
title: "Fix unsafe state lock-break race"
id: "kn08l6"
status: completed
priority: high
type: bug
tags: ["state", "concurrency"]
created_at: "2026-05-17"
verify:
  - type: bash
    run: "go build -o nd ."
  - type: bash
    run: "go test ./internal/state/..."
  - type: bash
    run: "go test -race ./internal/state/..."
  - type: bash
    run: "go test ./..."
  - type: assert
    check: "Two FileLock holders that both observe the same stale (>60s old) lock file and both invoke the stale-break path cannot both succeed Acquire() and enter WithLock's fn() at the same time — exactly one wins; the other blocks or returns *nd.LockError."
  - type: assert
    check: "FileLock.Release() no longer silently discards the syscall.Flock(LOCK_UN) error; the unlock error is returned (joined with the Close error if both fail)."
context:
  - "internal/state/lock.go"
  - "internal/state/store.go"
  - "internal/state/lock_test.go"
  - "internal/nd/errors.go"
  - "internal/nd/atomic.go"
  - "internal/deploy/deploy.go"
completed_at: 2026-08-01
---

## Fix unsafe state lock-break race

### Objective

`state.FileLock` (`internal/state/lock.go`) is the advisory lock that
serializes every read-modify-write of `deployments.yaml`. It is acquired
through `state.Store.WithLock` (`internal/state/store.go:100-107`), which
wraps `lock.Acquire(5 * time.Second)` + `defer lock.Release()`. Every
deployment-state mutation goes through this path (e.g.
`internal/deploy/deploy.go:174-187` runs `Load` → mutate → `Save` inside
`WithLock`; `Save` calls `nd.AtomicWrite`, an unconditional
write-temp-then-`os.Rename`, at `internal/nd/atomic.go:45`).

Locking uses `syscall.Flock(fd, LOCK_EX)`. **`flock(2)` advisory locks are
associated with the open file description / inode, not with the
pathname.** The stale-lock recovery in `breakAndRetry`
(`internal/state/lock.go:74-99`) does `os.Remove(l.Path)` and then
`os.OpenFile(l.Path, O_CREATE|O_RDWR, 0o644)`. After the unlink, the next
`OpenFile` creates a **brand-new inode**. Two concurrent `nd` processes
that both time out on the *same* stale lock file each run
`breakAndRetry`: each `os.Remove`s the path, each creates its own new
file/inode, and each `syscall.Flock(LOCK_EX|LOCK_NB)` succeeds because the
locks are taken on *different inodes*. Both processes return from
`Acquire`, both enter `WithLock`'s `fn()` critical section, and both run
`Load` → mutate → `Save`. The two `AtomicWrite` `os.Rename`s race and one
silently clobbers the other's deployment state — the lock provides no
mutual exclusion in exactly the situation it exists to handle.

Secondary defect: `Release` (`internal/state/lock.go:102-111`) calls
`syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)` and **ignores its
return value**, only propagating the `l.file.Close()` error. An unlock
failure is silently swallowed, and (depending on the chosen scheme) the
lock file may need cleanup that `Release` never performs.

Jargon resolved (no spec document exists; these are informal internal
non-functional requirements referenced only in code comments and the
CHANGELOG):

- **NFR-011** — "the state file is guarded by an advisory file lock"
  (`internal/state/state.go:12`, `internal/nd/errors.go:16`). `nd.LockError`
  is the error returned when the lock cannot be acquired within the
  timeout.
- **NFR-010** — "state files are written atomically (write-to-temp,
  fsync, rename) so a crash mid-write never leaves a partial file"
  (`internal/nd/atomic.go:10`, `internal/state/store.go:86`). Do not
  weaken `AtomicWrite`; the fix is in the locking layer only.

Why it matters: `deployments.yaml` is the source of truth for what `nd`
has deployed. A lost write under stale-lock contention corrupts that
state with no error and no recovery path.

### Steps to reproduce

There is no existing repro test that fails on the bug. Existing
`internal/state/lock_test.go` actually *asserts the buggy behavior*:
`TestFileLockStaleBreakFailsWhenStillHeld`
(`internal/state/lock_test.go:150-182`) deliberately expects the
remove-and-retry to succeed on a new inode even while another holder
still has the flock — that test must be rewritten as part of this fix
(see Tasks), not left green.

To reproduce the race in a new in-process test (two goroutines, one
process — `flock` semantics are per-inode regardless of process, so two
goroutines are sufficient and deterministic):

1. `dir := t.TempDir()`; `lockPath := filepath.Join(dir, "test.lock")`.
2. Create the lock file and set its mod time >60s in the past so
   `isStale()` (`internal/state/lock.go:64-70`,
   `staleLockThreshold = 60s` at line 26) returns true. Mirror the setup
   in `TestFileLockStaleDetectionSucceeds`
   (`internal/state/lock_test.go:84-120`): `os.OpenFile(..., O_CREATE|
   O_RDWR, 0o644)`, then `os.Chtimes(lockPath, oldTime, oldTime)` with
   `oldTime := time.Now().Add(-2 * time.Minute)`. Leave the file present
   but with NO active flock holder (so both racers reach the stale-break
   path on timeout rather than acquiring immediately, or hold an
   *unlocked* fd open to keep the old mod time).
3. Launch 2 goroutines, each: `lock := state.NewFileLock(lockPath)`;
   `err := lock.Acquire(200 * time.Millisecond)`; if `err == nil`,
   enter a guarded critical section (increment a shared counter under a
   `sync.Mutex`, sleep briefly, decrement), then `lock.Release()`.
4. Observed (bug): both `Acquire` calls return `nil` and the max
   observed concurrent-in-critical-section count is 2.
   Expected (fixed): the in-critical-section count never exceeds 1
   (the loser either blocks until the winner releases, or returns a
   non-nil `*nd.LockError`).

### Root cause locations (verified against current code)

- `internal/state/lock.go:74-99` — `breakAndRetry`: `os.Remove(l.Path)`
  (line 75) then `os.OpenFile(l.Path, O_CREATE|O_RDWR, 0o644)` (line 77)
  creates a new inode; `syscall.Flock(LOCK_EX|LOCK_NB)` (line 86) on that
  new inode succeeds even if another process holds the lock on the old
  (now-unlinked) inode. This is the unsafe unlink-then-relock.
- `internal/state/lock.go:50-52` — `Acquire`'s timeout branch calls
  `l.isStale()` then `l.breakAndRetry(timeout)`; the staleness check at
  line 50 and the recovery at line 51 are themselves a TOCTOU window
  across processes.
- `internal/state/lock.go:106` — `Release` calls
  `syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)` and discards the
  returned error (only `l.file.Close()`'s error at line 107 is returned).

### Tasks

- [ ] `internal/state/lock.go:74-99` — replace the
  unlink-then-recreate-then-relock in `breakAndRetry` with a path-stable
  scheme so the lock identity survives stale recovery. Pick ONE:
  - **(Recommended) Lock a stable, never-unlinked file.** Keep
    `os.OpenFile(l.Path, O_CREATE|O_RDWR, 0o644)` but never `os.Remove`
    the lock file. To recover from a stale holder, do NOT delete the
    inode; instead, after detecting staleness, keep retrying
    `Flock(LOCK_EX|LOCK_NB)` on the *same* open fd/inode within an
    extended deadline (a dead process's flock is released by the kernel
    on exit, so the existing inode becomes lockable without unlinking).
    Optionally `os.Truncate`/rewrite metadata on the same inode rather
    than recreating it. Because every contender opens the same path →
    same inode → same flock, mutual exclusion holds.
  - **(Alternative) `O_CREATE|O_EXCL` pidfile with ownership +
    staleness validation.** The lock owner writes its PID; a contender
    that times out reads the PID, verifies the process is dead
    (`syscall.Kill(pid, 0)` → `ESRCH`) AND the mtime is >60s old, and
    only then breaks it — and the break must itself be serialized (e.g.
    rename-into-place / `O_EXCL` create of the new pidfile so only one
    breaker wins). Do not unlink-then-relock on a fresh inode.
- [ ] `internal/state/lock.go:32-60` — update `Acquire` so the stale
  path it dispatches to (line 50-52) uses the chosen path-stable scheme.
  Preserve the public contract: signature
  `Acquire(timeout time.Duration) error`, sets `l.file`, `l.AcquiredAt`
  on success, returns `*nd.LockError` (`internal/nd/errors.go:16-28`) on
  failure with `Path` set, `Timeout = timeout.String()`, and
  `Stale = true` when a stale lock was detected. Existing tests depend
  on this: `TestFileLockBlocksConcurrent`
  (`internal/state/lock_test.go:31-52`) asserts `errors.As(err,
  &*nd.LockError)`; `TestFileLockNoStaleBreakForRecentLock`
  (`internal/state/lock_test.go:122-148`) asserts `lockErr.Stale ==
  false` for a recent lock; `TestFileLockStaleDetectionSucceeds`
  (`internal/state/lock_test.go:84-120`) asserts a released stale lock
  is re-acquirable.
- [ ] `internal/state/lock.go:102-111` — `Release`: capture the
  `syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)` return value and do
  not drop it. Return it (use `errors.Join(unlockErr, closeErr)` or
  return the unlock error if non-nil, else the close error). Keep
  `Release` idempotent — the `if l.file == nil { return nil }` early
  return (lines 103-105) must stay so `TestFileLockDoubleRelease`
  (`internal/state/lock_test.go:71-82`) still passes. If the chosen
  scheme requires lock-file cleanup, do it here consistently with the
  scheme (do NOT introduce an unlink that reopens the same race).
- [ ] `internal/state/lock_test.go:150-182` — rewrite
  `TestFileLockStaleBreakFailsWhenStillHeld`. It currently encodes the
  bug ("After removing the stale file, the retry opens a new file (new
  inode), so flock should succeed" — `lock_test.go:171-180`). Replace it
  with a test asserting the *correct* invariant: when a stale-aged lock
  file is still actively flocked by another fd, a second `Acquire`
  within a short timeout must NOT enter the critical section — it must
  fail with `*nd.LockError` (or block until the holder releases). Keep
  `TestFileLockStaleDetectionSucceeds` passing (a *released* stale lock
  must still be re-acquirable).
- [ ] Add a concurrency regression test in
  `internal/state/lock_test.go` (package `state_test`, mirror existing
  imports at `internal/state/lock_test.go:1-13`). Two goroutines, forced
  >60s-stale lock file (per "Steps to reproduce"), `sync.WaitGroup`,
  shared counter guarded by `sync.Mutex`; assert the max concurrent
  count inside the post-`Acquire` critical section is exactly 1 and that
  at most one `Acquire` returns `nil` at a time. Must pass under
  `go test -race ./internal/state/...`.

### Acceptance criteria

- `go build -o nd .` succeeds.
- `go test ./internal/state/...` and
  `go test -race ./internal/state/...` pass, including the new
  concurrency regression test and the rewritten
  `TestFileLockStaleBreakFailsWhenStillHeld`.
- `go test ./...` passes with no regressions; existing lock tests
  `TestFileLockAcquireRelease`, `TestFileLockBlocksConcurrent`,
  `TestFileLockReleaseUnlocks`, `TestFileLockDoubleRelease`,
  `TestFileLockStaleDetectionSucceeds`,
  `TestFileLockNoStaleBreakForRecentLock`
  (`internal/state/lock_test.go`) still pass (their public expectations
  on `*nd.LockError` and `Stale` are preserved).
- Two concurrent holders that both observe the same stale (>60s) lock
  and both take the stale-break path can never both be inside
  `WithLock`'s `fn()` at the same time (verified by the new test).
- `FileLock.Release()` returns the `syscall.Flock(LOCK_UN)` error
  instead of silently discarding it; `Release` remains idempotent.
- No change to `internal/nd/atomic.go` (NFR-010 atomic-overwrite
  semantics for `state.Store.Save` must be preserved); the fix is
  confined to the locking layer (`internal/state/lock.go` and its
  tests).

### References

- GitHub issue: https://GitHub.com/armstrongl/nd/issues/135
- Close this issue when the task is completed.
- Bug type: net-new, found during a codebase sweep — there is no seed
  file and no numbered seed pattern; do not search for one.
- `internal/state/lock.go:12-111` — `state.FileLock`: `NewFileLock`
  (21-23), `staleLockThreshold = 60s` (26), `Acquire` (32-60),
  `isStale` (64-70), `breakAndRetry` (74-99, the unsafe unlink+relock),
  `Release` (102-111, drops the unlock error).
- `internal/state/store.go:100-107` — `state.Store.WithLock`: the
  acquire / `defer Release` / run-`fn` wrapper. Its callers are the
  read-modify-write cycles this lock protects.
- `internal/deploy/deploy.go:174-187` — representative critical section
  (`Load` → mutate → `Save` inside `WithLock`); shows what double entry
  corrupts. Similar blocks at `deploy.go:469-501`, `511-521`, `529-548`,
  `559-574`, and `internal/deploy/health.go:47-`, `125-158`.
- `internal/nd/atomic.go:11-51` — `nd.AtomicWrite`; unconditional
  `os.Rename(tmpPath, path)` at line 45 means concurrent `Save`s
  silently clobber. Do not modify (NFR-010).
- `internal/state/store.go:87-97` — `state.Store.Save` (calls
  `nd.AtomicWrite`); it legitimately overwrites — the serialization
  guarantee must come from the lock, not from `AtomicWrite`.
- `internal/nd/errors.go:16-28` — `nd.LockError` (NFR-011). `Acquire`
  must keep returning this with `Path`, `Timeout`, and `Stale` set as
  today.
- `internal/state/state.go:11-12` — comment documenting the NFR-010
  (atomic write) + NFR-011 (advisory lock) contract for the state file.
- `internal/state/lock_test.go:1-182` — full existing test suite;
  patterns to reuse: stale-lock simulation via `os.Chtimes` with
  `time.Now().Add(-2*time.Minute)` (lines 98-102, 166-169), held-lock
  simulation via a second fd + `syscall.Flock` (lines 90-96, 155-162),
  `errors.As(err, &*nd.LockError)` assertions (lines 48-51, 141-144).
- Go module path: `github.com/armstrongl/nd` (import the package under
  test as `github.com/armstrongl/nd/internal/state`, package
  `state_test`).
- `flock(2)` semantics reference: advisory locks created with `flock()`
  are associated with the open file description and the underlying
  inode; unlinking the path and recreating it yields a new inode that is
  independently lockable — this is the core mechanism of the bug.
