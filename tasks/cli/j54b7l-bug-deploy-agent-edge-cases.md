---
title: "Fix deploy/agent edge cases in backup target default detection and init symlink strategy"
id: "j54b7l"
status: pending
priority: medium
type: bug
tags: ["deploy"]
created_at: "2026-05-17"
---

## Fix deploy/agent edge cases

### Objective

Net-new bugs (no seed; found during the codebase sweep). Three independent edge-case defects in the deploy/agent path: a foreign-symlink conflict warning drops the old target, `Registry.Default()` can silently pick a stale empty agent dir, and `nd init` built-in deploy ignores the configured symlink strategy.

### Tasks

- [ ] `internal/deploy/deploy.go:403-417` -- `backupAndWarn` accepts `target` but does `_ = target`; for a foreign-symlink conflict on a context file the warning never says where the old symlink pointed, losing recovery info. Include the target in the message
- [ ] `internal/agent/registry.go:201-205` -- `Default()` returns the first agent with `Detected == true`, and `Detected` is true if the global dir merely exists (`registry.go:142`); a stale empty `~/.copilot` with no binary can silently become the default. Prefer binary-in-PATH detection over bare-dir existence when choosing the default
- [ ] `cmd/init_cmd.go:231-236` -- built-in deploy hardcodes `nd.ScopeGlobal` + `SymlinkAbsolute` and ignores `cfg.SymlinkStrategy`; respect the configured strategy like `cmd/deploy.go:150-162`
- [ ] Tests for each: backup warning includes target; `Default()` does not pick a binary-less stale dir over a real agent; init honors `symlink_strategy: relative`

### Acceptance criteria

- Foreign-symlink conflict warnings include the previous target path
- `Registry.Default()` does not select a binary-less stale directory when a real agent is available
- `nd init` built-in deploy honors the configured symlink strategy
- Regression tests for all three

### References

- net-new, no seed pattern
- `internal/deploy/deploy.go:403-417`, `internal/agent/registry.go:142,201-205`, `cmd/init_cmd.go:231-236`
