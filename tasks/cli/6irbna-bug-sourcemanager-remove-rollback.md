---
title: "Fix SourceManager Remove rollback slice corruption"
id: "6irbna"
status: pending
priority: medium
type: bug
tags: ["source"]
created_at: "2026-05-17"
---

## Fix SourceManager.Remove rollback slice corruption

### Objective

Net-new bug (no seed; found during the codebase sweep). `SourceManager.Remove` compacts its source slice in place, mutating the shared backing array. If `WriteConfig` fails, the rollback reads from the already-compacted slice and re-appends over the same backing array, producing wrong contents/order (the originally-last element is lost). `AddLocal`/`AddGit` use the correct `oldSources` snapshot pattern; `Remove` is the odd one out.

### Steps to reproduce

1. Register multiple sources.
2. Make `WriteConfig` fail (e.g. read-only config path) and call `Remove` on a non-last source.
3. Inspect `sm.cfg.Sources`: order/contents are corrupted instead of restored.

### Tasks

- [ ] `internal/sourcemanager/register.go:117-121` -- snapshot the original slice before mutating and restore it verbatim on `WriteConfig` failure (apply the `oldSources` pattern from `AddLocal:80-87` / `AddGit:167-174`)
- [ ] Regression test: simulate a `WriteConfig` failure during `Remove` and assert `sm.cfg.Sources` is byte-identical to its pre-call state

### Acceptance criteria

- A simulated `WriteConfig` failure leaves `sm.cfg.Sources` identical to its pre-call state
- Regression test covers a non-last source removal with write failure

### References

- net-new, no seed pattern
- `internal/sourcemanager/register.go:100-126` (compare `AddLocal:80-87`)
