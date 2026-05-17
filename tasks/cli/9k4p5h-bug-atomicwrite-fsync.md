---
title: "Fsync parent directory in AtomicWrite for crash safety"
id: "9k4p5h"
status: pending
priority: medium
type: bug
tags: ["core", "durability"]
created_at: "2026-05-17"
---

## Fsync parent directory in AtomicWrite for crash safety

### Objective

Net-new bug (no seed; found during the codebase sweep). `AtomicWrite` syncs the temp file but never fsyncs the containing directory after `os.Rename`. On a crash immediately after the rename, the rename itself may not be durable, so the "atomic" write can be lost despite the NFR-010 comment claiming crash safety. This affects every state/profile/config/snapshot write.

### Steps to reproduce

1. `AtomicWrite` a file; crash/power-loss immediately after the rename returns.
2. On recovery the rename may not have persisted; the file is missing or stale.

### Tasks

- [ ] `internal/nd/atomic.go:45-50` -- after `os.Rename`, open the parent directory and `fsync` it (handle platforms where directory fsync is a no-op) before returning
- [ ] Verify no measurable regression on hot write paths (state save)
- [ ] Add a test asserting the parent directory is fsynced (or the call path is exercised)

### Acceptance criteria

- `AtomicWrite` fsyncs the parent directory after rename so the rename is durable
- The NFR-010 crash-safety claim holds
- No measurable slowdown on the state-save hot path

### References

- net-new, no seed pattern
- `internal/nd/atomic.go:25-50`
