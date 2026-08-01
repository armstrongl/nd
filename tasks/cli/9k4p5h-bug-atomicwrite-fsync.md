---
title: "Fsync parent directory in AtomicWrite for crash safety"
id: "9k4p5h"
status: completed
priority: medium
type: bug
tags: ["core", "durability"]
created_at: "2026-05-17"
context:
  - "internal/nd/atomic.go"
  - "internal/nd/atomic_test.go"
  - "internal/state/store.go"
  - "internal/profile/store.go"
  - "internal/sourcemanager/config.go"
verify:
  - type: bash
    run: "go build -o nd ."
  - type: bash
    run: "go test ./internal/nd/..."
  - type: bash
    run: "go test -race ./..."
  - type: bash
    run: "golangci-lint run"
  - type: assert
    check: "AtomicWrite opens the parent directory after os.Rename and calls Sync() on it (errors other than unsupported-platform are surfaced); existing temp-file fsync, chmod, and rename behavior is unchanged"
  - type: assert
    check: "A test in internal/nd/atomic_test.go exercises the post-rename directory-fsync path and still passes (no temp files left, content correct)"
completed_at: 2026-08-01
---

## Fsync parent directory in AtomicWrite for crash safety

### Objective

`AtomicWrite` in `internal/nd/atomic.go` is the single durability primitive for
every state/profile/config/snapshot write in the codebase. It fsyncs the temp
file before rename but never fsyncs the **parent directory** after
`os.Rename`. On POSIX filesystems the directory entry created/updated by
`rename(2)` is not guaranteed durable until the directory itself is fsynced;
a crash or power loss immediately after `os.Rename` returns can leave the
target file missing or pointing at stale content even though `AtomicWrite`
returned `nil`. This contradicts the function's own doc comment
(`internal/nd/atomic.go:9-10`) and the NFR-010 crash-safety claim.

Fix: after a successful `os.Rename`, open the parent directory and `fsync` it
before returning, tolerating platforms where directory fsync is unsupported
(some non-POSIX/network filesystems return `EINVAL`/`ENOTSUP`).

NFR-010 is an internal durability requirement: "state files are written
atomically (write-to-temp-then-rename) so a crash mid-write never corrupts or
loses committed data." It is not in a formal spec doc — it appears only as
inline references in `internal/nd/atomic.go:10`, `internal/state/store.go:86`,
`internal/state/state.go:11`, `internal/sourcemanager/config.go:124`, and
`CHANGELOG.md:131`. The directory-fsync gap is the part of that guarantee
currently unmet.

Affected write paths (all route through `nd.AtomicWrite`):

- `internal/state/store.go:96` — `deployments.yaml` (hot path; `Store.Save`)
- `internal/sourcemanager/config.go:133` — source config
- `internal/profile/store.go:80,177,227` — profile files
- `internal/profile/store.go:331` — auto snapshot files

### Steps to reproduce

1. Call `nd.AtomicWrite(path, data)`; it fsyncs the temp file then `os.Rename`s
   it over `path` and returns `nil` (`internal/nd/atomic.go:45-50`).
2. Simulate crash/power loss immediately after the rename (before the
   filesystem flushes the parent directory metadata).
3. On recovery the directory entry for `path` may not have persisted: the file
   is missing or still references the pre-rename inode/content, despite
   `AtomicWrite` having returned success. No automated repro is feasible
   without fault injection; the defect is verified by code inspection
   (no `Open(dir)` + `Sync()` exists after line 45) and fixed by code review +
   the existing/new tests confirming behavior is otherwise preserved.

### Tasks

- [ ] In `internal/nd/atomic.go`, after the successful `os.Rename` block
  (currently `internal/nd/atomic.go:45-48`) and before `return nil`
  (currently line 50): open the parent directory
  (`dir := filepath.Dir(path)`, already computed at
  `internal/nd/atomic.go:12`) with `os.Open(dir)`, call `Sync()` on the
  directory handle, then `Close()` it. Wrap errors with `fmt.Errorf("fsync
  parent directory: %w", err)` to match the existing error-wrapping style
  in this file (e.g. lines 16, 27, 32, 37, 42, 47).
- [ ] Tolerate platforms where directory fsync is unsupported: treat a `Sync()`
  error of `syscall.EINVAL` or `syscall.ENOTSUP` (use `errors.Is`) as a
  no-op success rather than failing the write; surface any other error.
  Do not let a directory `Open`/`Sync` failure leave a leaked fd or undo
  the completed rename (the file is already in place — only report the
  durability error).
- [ ] Add a test to `internal/nd/atomic_test.go` (package `nd_test`,
  mirror existing style: `t.TempDir()`, focused `TestAtomicWrite*`
  function, table-free) that calls `nd.AtomicWrite` into a temp dir and
  asserts it succeeds, the file content is correct, and no `.nd-*.tmp`
  files remain (same assertions as `TestAtomicWriteNoTempFilesAfterSuccess`
  at `internal/nd/atomic_test.go:80-99`). This exercises the new
  directory-fsync code path without crash injection. Name it e.g.
  `TestAtomicWriteFsyncsParentDir`.
- [ ] Confirm no measurable regression on the hot write path
  (`internal/state/store.go:87` `Store.Save`): one extra directory
  open+fsync+close per write is expected and acceptable; just verify
  `go test -race ./...` and the integration tests still pass and runtime
  is unchanged in practice.

### Acceptance criteria

- After a successful `os.Rename`, `AtomicWrite` opens the parent directory and
  calls `Sync()` on it before returning, so the rename is durable.
- Directory-fsync failures with `EINVAL`/`ENOTSUP` are tolerated (no-op);
  any other directory fsync error is returned wrapped with context.
- Existing `AtomicWrite` behavior is unchanged: temp file fsync, `chmod 0o644`,
  rename, no temp files left on success or failure (all tests in
  `internal/nd/atomic_test.go` still pass).
- A new test in `internal/nd/atomic_test.go` exercises the post-rename
  directory-fsync path.
- `go build -o nd .`, `go test -race ./...`, and `golangci-lint run` all pass.

### References

- GitHub issue: https://GitHub.com/armstrongl/nd/issues/117
- Close this issue when the task is completed.
- Bug site: `internal/nd/atomic.go:45-50` (`os.Rename` then `return nil`, no
  directory fsync); parent dir already computed at `internal/nd/atomic.go:12`.
- Doc comment making the crash-safety claim: `internal/nd/atomic.go:9-10`.
- Error-wrapping pattern to mirror: `internal/nd/atomic.go:16,27,32,37,42,47`.
- Test patterns to mirror: `internal/nd/atomic_test.go:80-99`
  (`TestAtomicWriteNoTempFilesAfterSuccess`) and the file's overall structure
  (package `nd_test`, `t.TempDir()`, per-behavior `TestAtomicWrite*` funcs).
- Callers depending on this durability:
  `internal/state/store.go:96`, `internal/sourcemanager/config.go:133`,
  `internal/profile/store.go:80,177,227,331`.
- NFR-010 references (informal; no spec doc): `internal/nd/atomic.go:10`,
  `internal/state/store.go:86`, `internal/state/state.go:11`,
  `internal/sourcemanager/config.go:124`, `CHANGELOG.md:131`.
- net-new, no seed pattern.
