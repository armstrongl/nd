---
title: "Fix profile/snapshot create TOCTOU silent overwrite"
id: "2zlf37"
status: pending
priority: medium
type: bug
tags: ["profile", "concurrency"]
created_at: "2026-05-17"
---

## Fix profile/snapshot create TOCTOU silent overwrite

### Objective

Net-new bug (no seed; found during the codebase sweep). `CreateProfile` and `SaveSnapshot` do an `os.Stat` "already exists" check then `nd.AtomicWrite`, whose unconditional `os.Rename` overwrites an existing file. Between the stat and the rename, a concurrent `nd profile create <samename>` silently clobbers the other writer -- profile/snapshot writes are not guarded by any lock (unlike `state.Store.WithLock`). `AutoSnapshot` skips the existence check entirely.

### Steps to reproduce

1. Run two concurrent `nd profile create foo` (or snapshot save) for the same name.
2. One silently overwrites the other; no "already exists" error for the loser.

### Tasks

- [ ] `internal/profile/store.go:72-80` -- `CreateProfile` TOCTOU: guard with a lock or use `O_CREATE|O_EXCL` create semantics instead of stat-then-AtomicWrite
- [ ] `internal/profile/store.go:219-227` -- `SaveSnapshot` same TOCTOU
- [ ] `internal/profile/store.go:325-333` -- `AutoSnapshot` skips the existence check and `AtomicWrite`s directly; collision on a non-monotonic clock silently overwrites -- add a guard
- [ ] Regression test: concurrent same-name create yields one success and one clear "already exists" error

### Acceptance criteria

- Concurrent same-name profile/snapshot create produces exactly one success; the loser gets a clear error, no silent overwrite
- `AutoSnapshot` cannot silently overwrite an existing snapshot
- Regression test covers the concurrent case

### References

- net-new, no seed pattern
- `internal/profile/store.go:72-80,219-227,325-333`, `internal/nd/atomic.go:45`
