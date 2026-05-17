---
title: "Fix unsafe state lock-break race"
id: "kn08l6"
status: pending
priority: high
type: bug
tags: ["state", "concurrency"]
created_at: "2026-05-17"
---

## Fix unsafe state lock-break race

### Objective

Net-new bug (no seed; found during the codebase sweep). The deployment-state lock's stale-lock recovery uses an unlink-then-recreate pattern. `flock` is bound to the open file description/inode, not the path: two concurrent `nd` processes that both time out on the same stale lock each `os.Remove` then create their own lock file and both succeed `flock(LOCK_EX)` on distinct inodes, both entering the `WithLock` critical section and corrupting `deployments.yaml` despite the lock.

### Steps to reproduce

1. Create a stale lock file older than 60s.
2. Start two `nd` processes that both contend on it.
3. Both break the lock, acquire flock on different inodes, and write `deployments.yaml` concurrently.

### Tasks

- [ ] `internal/state/lock.go:74-99` -- replace the unlink-recovery in `breakAndRetry` with a path-stable scheme (flock the state file itself, or an `O_CREATE|O_EXCL` pidfile with ownership/staleness validation and no unlink-before-relock)
- [ ] `internal/state/lock.go:106` -- `Release` ignores `syscall.Flock(LOCK_UN)` error and never removes the lock file; reconcile with the chosen scheme
- [ ] Add a concurrency test (two goroutines/processes, forced-stale lock) asserting serialized state writes

### Acceptance criteria

- Two concurrent processes contending on a stale lock cannot both enter the critical section
- A concurrency test shows serialized `deployments.yaml` writes under forced-stale-lock contention
- `Release` errors are handled, not silently dropped

### References

- net-new, no seed pattern
- `internal/state/lock.go:32-111`, `internal/state/store.go:99-107`
