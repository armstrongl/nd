---
title: "Fix SourceManager Remove rollback slice corruption"
id: "6irbna"
status: completed
priority: medium
type: bug
tags: ["source"]
created_at: "2026-05-17"
verify:
  - type: bash
    run: "go build -o nd ."
  - type: bash
    run: "go test ./internal/sourcemanager/..."
  - type: bash
    run: "go test -race ./internal/sourcemanager/..."
  - type: bash
    run: "go vet ./internal/sourcemanager/..."
  - type: assert
    check: "A simulated WriteConfig failure during Remove leaves sm.Config().Sources byte-identical (reflect.DeepEqual) to its pre-call value, including order, for a non-last source removal"
context:
  - internal/sourcemanager/register.go
  - internal/sourcemanager/register_test.go
  - internal/sourcemanager/config.go
  - internal/sourcemanager/sourcemanager.go
  - internal/nd/atomic.go
completed_at: 2026-08-01
---

## Fix SourceManager.Remove rollback slice corruption

### Objective

`(*SourceManager).Remove` in `internal/sourcemanager/register.go` deletes a
source entry by compacting `sm.cfg.Sources` **in place**, which mutates the
shared backing array. If the subsequent `WriteConfig` call fails, the rollback
code reconstructs the slice from the already-corrupted backing array, so
`sm.cfg.Sources` is left in a wrong order/contents state instead of being
restored. The two sibling methods (`AddLocal`, `AddGit`) already use the
correct snapshot-and-restore pattern; `Remove` must be brought in line.

Why it matters: a transient config-write failure (e.g. disk full, read-only
config dir) during `nd source remove` would silently corrupt the in-memory
source list for the rest of the process lifetime, and any later successful
write would persist the corruption to disk.

This is a net-new bug found during a codebase sweep — there is no seed/spec
document; the only source of truth is the code itself.

### Root cause (verified)

`internal/sourcemanager/register.go`, function `Remove` (lines 100-126):

```go
removed := sm.cfg.Sources[idx]                                              // line 116
sm.cfg.Sources = append(sm.cfg.Sources[:idx], sm.cfg.Sources[idx+1:]...)     // line 117  <-- in-place compaction
if err := WriteConfig(sm.configPath, sm.cfg); err != nil {                   // line 119
    // Roll back
    sm.cfg.Sources = append(sm.cfg.Sources[:idx], append([]config.SourceEntry{removed}, sm.cfg.Sources[idx:]...)...) // line 121  <-- reads corrupted data
    return fmt.Errorf("save config: %w", err)                                // line 122
}
```

- Line 117 `append(sm.cfg.Sources[:idx], sm.cfg.Sources[idx+1:]...)` shifts
  every element after `idx` left by one **within the same backing array** and
  shortens the slice by one. The slot that previously held the
  originally-last element now holds a stale duplicate, and that original
  last element's value is gone from the addressable region.
- Line 121 attempts to rebuild the slice, but `sm.cfg.Sources[idx:]` now
  refers to the already-shifted (corrupted) data, so the restored slice has
  the wrong contents/order — the originally-last element is lost.

Contrast with the correct pattern, `AddLocal` (lines 80-87) and `AddGit`
(lines 167-174):

```go
oldSources := sm.cfg.Sources                       // snapshot BEFORE mutating
sm.cfg.Sources, _ = insertBeforeBuiltin(sm.cfg.Sources, entry)
if err := WriteConfig(sm.configPath, sm.cfg); err != nil {
    sm.cfg.Sources = oldSources                    // restore verbatim
    return ...
}
```

Note: `oldSources := sm.cfg.Sources` works for the Add path because
`insertBeforeBuiltin` (lines 16-27) allocates a *new* slice via
`make(...)`/`append(append, ...)` and never writes into the original backing
array. `Remove` must adopt the same discipline: take a snapshot that the
mutation cannot corrupt, and on failure restore it.

### How `WriteConfig` failure happens (for the regression test)

`WriteConfig` (`internal/sourcemanager/config.go:127`) calls
`nd.AtomicWrite` (`internal/nd/atomic.go:11`), whose first step is
`os.CreateTemp(filepath.Dir(path), ...)` (atomic.go:14). That fails
deterministically when the directory containing `sm.configPath` does not
exist. In tests, `newTestManager` builds the manager with
`configPath = filepath.Join(t.TempDir(), "config.yaml")`; deleting that
TempDir (`os.RemoveAll(dir)`) after populating sources makes the next
`Remove` -> `WriteConfig` call fail without any chmod tricks.

### Steps to reproduce

1. Construct a `SourceManager` via `sourcemanager.New(filepath.Join(dir, "config.yaml"), "")`.
2. Register at least two local sources with `AddLocal` (plus the implicit
   builtin entry appended by `New`, so `sm.Config().Sources` has >= 3 entries
   with the builtin last).
3. `os.RemoveAll(dir)` so the config directory no longer exists.
4. Snapshot `before := append([]config.SourceEntry(nil), sm.Config().Sources...)`.
5. Call `sm.Remove(<id of the FIRST, i.e. non-last, source>)` — it returns a
   `save config:` error (expected).
6. Compare `sm.Config().Sources` to `before` with `reflect.DeepEqual`.
   Currently they differ (order/contents corrupted); after the fix they are
   equal.

### Tasks

- [ ] In `internal/sourcemanager/register.go`, function `Remove` (lines
  100-126): replace the in-place compaction + reconstruction rollback
  (lines 116-122) with a snapshot-then-restore approach mirroring
  `AddLocal` (lines 80-87):
  - Snapshot the original slice before any mutation:
    `oldSources := sm.cfg.Sources`.
  - Build the new slice **without aliasing the original backing array**,
    e.g. `newSources := make([]config.SourceEntry, 0, len(oldSources)-1)`
    then append `oldSources[:idx]` and `oldSources[idx+1:]`, assigning to
    `sm.cfg.Sources`. (Do not reuse `oldSources[:idx]` as the append target.)
  - On `WriteConfig` failure, restore verbatim: `sm.cfg.Sources = oldSources`
    and return the existing `fmt.Errorf("save config: %w", err)`.
  - Remove the now-unneeded `removed` variable / reconstruction expression.
- [ ] Add a regression test in
  `internal/sourcemanager/register_test.go` (external test package
  `sourcemanager_test`, alongside `TestRemove` at lines 169-191). It must:
  - Use a manually-built manager so the temp dir is known: replicate
    `newTestManager` inline (`dir := t.TempDir()`,
    `configPath := filepath.Join(dir, "config.yaml")`,
    `sm, _ := sourcemanager.New(configPath, "")`) so the dir can be deleted.
  - Register two local sources via `sm.AddLocal(t.TempDir(), "")` (call
    twice with two different `t.TempDir()` dirs).
  - Snapshot `before := append([]config.SourceEntry(nil), sm.Config().Sources...)`.
  - `os.RemoveAll(dir)` to force the next `WriteConfig` to fail.
  - Capture the ID of the first (non-last) source via `sm.Sources()[0].ID`.
  - Assert `sm.Remove(id)` returns a non-nil error.
  - Assert `reflect.DeepEqual(sm.Config().Sources, before)` is true (use
    `reflect`; add the import).
- [ ] Confirm existing tests still pass, especially `TestRemove`,
  `TestRemoveNotFound`, `TestRemovePersists`,
  `TestRemoveBuiltinReturnsError` (register_test.go:169-231).

### Acceptance criteria

- After the fix, the reproduction sequence above leaves
  `sm.Config().Sources` `reflect.DeepEqual` to its pre-`Remove` snapshot,
  including element order, for a non-last source removal with a write
  failure.
- A new regression test in `internal/sourcemanager/register_test.go`
  exercises a non-last source removal under a forced `WriteConfig` failure
  and fails on the current (buggy) code, passes after the fix.
- `go build -o nd .` succeeds.
- `go test ./internal/sourcemanager/...` and
  `go test -race ./internal/sourcemanager/...` pass.
- `go vet ./internal/sourcemanager/...` reports nothing.
- No behavioral change on the happy path: a successful `Remove` still drops
  exactly the targeted entry and persists the result (existing
  `TestRemove`/`TestRemovePersists` keep passing unchanged).

### References

- GitHub issue: https://GitHub.com/armstrongl/nd/issues/113
- Close this issue when the task is completed.
- Bug site: `internal/sourcemanager/register.go:100-126` (`Remove`),
  specifically lines 116-122.
- Pattern to mirror: `internal/sourcemanager/register.go:80-87` (`AddLocal`
  `oldSources` snapshot/restore) and `:167-174` (`AddGit`). Allocation
  helper that keeps the backing array un-aliased:
  `insertBeforeBuiltin` at `internal/sourcemanager/register.go:16-27`.
- `WriteConfig`: `internal/sourcemanager/config.go:127-134`.
- Failure mechanism: `internal/nd/atomic.go:11-51` (`os.CreateTemp` at
  line 14 fails when the target directory is absent).
- Existing test scaffolding: `internal/sourcemanager/register_test.go:28-37`
  (`newTestManager`), `:169-231` (Remove tests).
- Accessors: `(*SourceManager).Config()` returns `*config.Config`
  (`internal/sourcemanager/sourcemanager.go:52-55`); `Sources()` returns a
  copy with `Order` set (`sourcemanager.go:57-71`). `config.SourceEntry`
  is a flat value struct (`internal/config/config.go:20-26`), so
  `reflect.DeepEqual` and slice copy via `append([]config.SourceEntry(nil), ...)`
  compare correctly.
- net-new, no seed pattern.
