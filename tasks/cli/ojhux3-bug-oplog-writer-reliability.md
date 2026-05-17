---
title: "Fix oplog writer ignored Close and swallowed rotation error"
id: "ojhux3"
status: pending
priority: medium
type: bug
tags: ["state"]
created_at: "2026-05-17"
---

## Fix oplog writer ignored close and swallowed rotation error

### Objective

Net-new bug (no seed; found during the codebase sweep). The oplog writer discards the `Close` error on an append file it just wrote (a deferred-close failure can mean the final write was not durably flushed), and `rotateIfNeeded` returns nil if `os.Stat` fails for any reason (not just `IsNotExist`), so a permission error on the log path silently disables rotation and the log grows unbounded past `maxSize`.

### Steps to reproduce

1. Make the oplog path's `os.Stat` fail with a non-NotExist error (e.g. permission).
2. Observe rotation never happens; the log exceeds `maxSize` indefinitely.

### Tasks

- [ ] `internal/oplog/writer.go:58` -- capture and return the `Close` error when the preceding `Write` succeeded (named return + deferred assignment)
- [ ] `internal/oplog/writer.go:74,82` -- distinguish the `os.Stat` error: only treat `errors.Is(err, fs.ErrNotExist)` as "no file yet"; surface other errors instead of silently skipping rotation
- [ ] Regression tests: `Log` returns a non-nil error if `Close` fails after a successful write; a non-NotExist stat error does not silently skip rotation

### Acceptance criteria

- `oplog.Writer` returns a non-nil error if `Close` fails after a successful write
- A non-NotExist stat error is surfaced, not swallowed; rotation is not silently disabled
- Regression tests for both

### References

- net-new, no seed pattern
- `internal/oplog/writer.go:45-68,74-82`
