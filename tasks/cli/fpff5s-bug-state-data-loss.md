---
title: "Prevent state-file data loss on corrupt-rename and backup collision"
id: "fpff5s"
status: pending
priority: high
type: bug
tags: ["state", "data-loss"]
created_at: "2026-05-17"
---

## Prevent state-file data loss on corrupt-rename and backup collision

### Objective

Net-new bug (no seed; found during the codebase sweep). Two error paths in the deployment-state layer silently destroy user data: `handleCorrupt` ignores the rename error and then lets the next `Save` overwrite the still-present corrupt file with empty state, and `backupExistingFile` uses second-precision timestamps so two same-basename backups within one `DeployBulk` call clobber each other. `DeploymentState.Validate()` is also a no-op, so corrupt-but-parseable state is never rejected.

### Steps to reproduce

1. Corrupt `deployments.yaml` so it fails to parse, then run any state-mutating command on a path where the rename target is not writable.
2. Observe the warning says the file was renamed to `*.corrupt.*`, but it was not; the next `Save()` overwrites it with empty state.

### Tasks

- [ ] `internal/state/store.go:77` -- `os.Rename(s.path, corruptPath)` error is ignored; propagate it (or refuse to return empty state if the original could not be preserved) so `Store.Save()` does not overwrite the only recovery source
- [ ] `internal/deploy/deploy.go:422-439` -- `backupExistingFile` builds the path from `e.now().Format("2006-01-02T15-04-05")` (1s resolution); two same-basename foreign context files backed up within one `DeployBulk` second produce an identical path and `e.rename` clobbers the first. Add a uniqueness suffix (counter/nanoseconds) or a backupPath collision check
- [ ] `internal/state/state.go:44-46` -- `DeploymentState.Validate()` returns nil unconditionally; reject duplicate identities / invalid scope on `Load()`
- [ ] Regression tests for the failed-corrupt-rename path and the same-basename backup collision

### Acceptance criteria

- A failed corrupt-file rename never results in empty state being written over the original
- Two same-basename backups in one `DeployBulk` call both survive (distinct backup paths)
- Corrupt-but-parseable state is rejected by `Validate()` on load
- Regression tests cover both error paths

### References

- net-new, no seed pattern
- `internal/state/store.go:74-97`, `internal/state/state.go:44-46`, `internal/deploy/deploy.go:422-507`
