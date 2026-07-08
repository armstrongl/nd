---
title: "Fix oplog writer ignored Close and swallowed rotation error"
id: "ojhux3"
status: pending
priority: medium
type: bug
tags: ["state"]
created_at: "2026-05-17"
context:
  - "internal/oplog/writer.go"
  - "internal/oplog/writer_test.go"
  - "internal/oplog/oplog.go"
verify:
  - type: bash
    run: "go build -o nd ."
  - type: bash
    run: "go test ./internal/oplog/..."
  - type: bash
    run: "go test -race ./internal/oplog/..."
  - type: bash
    run: "golangci-lint run ./internal/oplog/..."
  - type: assert
    check: "oplog.Writer.Log returns a non-nil error if f.Close() fails after a successful Write"
  - type: assert
    check: "rotateIfNeeded only treats fs.ErrNotExist as 'no file yet'; any other os.Stat error is returned, not swallowed"
---

## Fix oplog writer ignored close and swallowed rotation error

### Objective

`internal/oplog/writer.go` is the only operation-log writer in the codebase
(`oplog.NewWriter` -> `Writer.Log`). It appends JSONL entries to
`<logDir>/operations.log` and rotates to `operations.log.1` when the file
exceeds `maxSize` (default 1 MB, `defaultMaxSize` at `writer.go:12`).

Two reliability bugs make the writer silently lose data and grow unbounded:

1. **Ignored `Close` error.** `Writer.Log` opens the append file with
   `os.OpenFile` and defers `f.Close()` at `internal/oplog/writer.go:58`. The
   deferred close error is discarded. On some filesystems the final buffered
   write is only flushed at `Close`, so `Log` can return `nil` even though the
   entry was never durably written — a silent audit-log loss.

2. **Swallowed rotation error.** `rotateIfNeeded` at
   `internal/oplog/writer.go:71-83` calls `os.Stat(w.path)` and, on **any**
   error, returns `nil` with the comment "file doesn't exist yet — nothing to
   rotate" (`writer.go:73-75`). This is correct only for a not-exist error. If
   `os.Stat` fails for another reason (e.g. EACCES on the log directory, a
   broken symlink, EIO), rotation is silently skipped and the log grows past
   `maxSize` forever, with no error surfaced to the caller of `Log`.

Fix both so failures are surfaced to the `Log` caller.

### Steps to reproduce

Bug 2 (deterministic, no root needed):

1. `mkdir -p /tmp/oplogrepro && touch /tmp/oplogrepro/operations.log`
2. `chmod 000 /tmp/oplogrepro` (remove traverse/exec on the dir so
   `os.Stat("/tmp/oplogrepro/operations.log")` returns EACCES, not
   `fs.ErrNotExist`).
3. Call `oplog.NewWriter("/tmp/oplogrepro").Log(entry)`. Current code:
   `os.MkdirAll` is a no-op on the existing dir, `rotateIfNeeded` swallows the
   EACCES from `os.Stat` and returns `nil`, so rotation never runs even once
   the file is over `maxSize`. Restore with `chmod 755 /tmp/oplogrepro`.

Bug 1 is reproduced via the regression test below (a `Close` failure is hard to
force without an injectable file handle; assert the behavior at the unit level —
see Tasks).

### Root cause (verified file:line)

- `internal/oplog/writer.go:58` — `defer f.Close()` discards the close error.
  `Writer.Log` (lines 45-68) uses a plain `func (w *Writer) Log(...) error`
  signature with `return err` at line 67 after `f.Write`.
- `internal/oplog/writer.go:71-75` — `rotateIfNeeded`:

  ```go
  info, err := os.Stat(w.path)
  if err != nil {
      return nil // file doesn't exist yet — nothing to rotate
  }
  ```

  treats every stat error as not-exist.
- `internal/oplog/writer.go:3-7` — current imports are only `encoding/json`,
  `os`, `path/filepath`. The fix must add `errors` and `io/fs`.

### Tasks

- [ ] `internal/oplog/writer.go:45` — change the signature to a named return:
  `func (w *Writer) Log(entry LogEntry) (err error)`.
- [ ] `internal/oplog/writer.go:58` — replace `defer f.Close()` with a deferred
  closure that closes `f` and, only if the function is not already
  returning an error, assigns the close error to the named return, e.g.:

  ```go
  defer func() {
  if cerr := f.Close(); cerr != nil && err == nil {
  err = cerr
  }
  }()```
  so a `Close` failure after a successful `f.Write` is surfaced. Keep the
  `os.OpenFile` error path (line 55-57) returning early as-is (no `f` to
  close there).
- [ ] `internal/oplog/writer.go:72-75` — in `rotateIfNeeded`, distinguish the
  `os.Stat` error: `if errors.Is(err, fs.ErrNotExist) { return nil }` then
  `return err` for any other error (do not swallow). Keep the
  `info.Size() < w.maxSize` short-circuit (line 77-79) and the
  `os.Rename` rotation (line 81-82) unchanged.
- [ ] `internal/oplog/writer.go:3-7` — add `"errors"` and `"io/fs"` to the
  import block; keep the block goimports-ordered (stdlib group, no blank
  lines between stdlib imports — match existing style).
- [ ] Add regression tests to `internal/oplog/writer_test.go` (package
  `oplog_test`, mirror the existing `TestWriter*` table/temp-dir style at
  `writer_test.go:17-260`, using `t.TempDir()` and `oplog.NewWriter`):
  - [ ] A stat error that is **not** `fs.ErrNotExist` is surfaced and rotation
    is not silently skipped. Reproduce via the directory-permission trick
    from "Steps to reproduce": create the temp log dir, write one entry,
    then `os.Chmod(logDir, 0o000)`; assert `w.Log(entry)` now returns a
    non-nil error (use `t.Cleanup` to `os.Chmod(logDir, 0o755)` so
    `t.TempDir` cleanup succeeds; `t.Skip` if running as root since root
    bypasses the permission check).
  - [ ] `Log` returns a non-nil error when `Close` fails after a successful
    `Write`. Since `Close` cannot be forced through the public API, assert
    the contract directly: after a normal successful `Log`, the error is
    `nil` and the entry is on disk (guards against the named-return change
    regressing the happy path). If feasible, add a same-package
    (`package oplog`) white-box test file
    (`internal/oplog/writer_internal_test.go`) that exercises the deferred
    close path, or document why the public-API assertion is sufficient.
- [ ] Run the `verify:` commands; ensure all existing `TestWriter*` tests in
  `internal/oplog/writer_test.go` still pass (rotation, append, JSONL,
  backup-overwrite, all operation types, partial-failure).

### Existing patterns to follow

- Named-return + deferred-close error capture is the standard Go idiom; there
  is no existing example in this package, but the change is local to
  `Writer.Log`.
- Test conventions: `internal/oplog/writer_test.go` uses external test package
  `oplog_test`, `t.TempDir()`, `oplog.NewWriter(dir, oplog.WithMaxSize(n))`,
  and reads `filepath.Join(dir, "operations.log")` / `operations.log.1`.
  `TestWriterRotatesAtMaxSize` (`writer_test.go:94-143`) and
  `TestWriterRotationOverwritesPreviousBackup` (`writer_test.go:145-180`) are
  the closest analogues for the rotation-error test.
- `LogEntry` shape and `OperationType` constants are in
  `internal/oplog/oplog.go` (e.g. `oplog.OpDeploy`, fields `Timestamp`,
  `Operation`, `Assets`, `Scope`, `Succeeded`, `Failed`, `Detail`).

### Acceptance criteria

- `oplog.Writer.Log` returns a non-nil error if `f.Close()` fails after a
  successful `f.Write` (no silent loss of the final entry).
- `rotateIfNeeded` returns `nil` only when `os.Stat` fails with
  `fs.ErrNotExist`; any other stat error is returned from `rotateIfNeeded` and
  propagated out of `Log` (rotation is not silently disabled).
- New regression tests cover both behaviors and live in
  `internal/oplog/writer_test.go` (and/or a white-box
  `internal/oplog/writer_internal_test.go`).
- `go build -o nd .`, `go test ./internal/oplog/...`,
  `go test -race ./internal/oplog/...`, and
  `golangci-lint run ./internal/oplog/...` all pass; existing oplog tests
  remain green.

### References

- GitHub issue: https://GitHub.com/armstrongl/nd/issues/119
- Close this issue when the task is completed.
- Net-new bug; no seed pattern (found during codebase sweep).
- `internal/oplog/writer.go:45-68` (`Writer.Log`), `:58` (`defer f.Close()`),
  `:71-83` (`rotateIfNeeded`), `:72-75` (swallowed stat error), `:3-7`
  (imports).
- `internal/oplog/writer_test.go:17-260` (existing test patterns to mirror).
- `internal/oplog/oplog.go:11-37` (`LogEntry`, `OperationType`).
