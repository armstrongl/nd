---
title: "Pre-filter assets by target agent supported types"
id: "rglxhu"
status: pending
priority: high
type: feature
tags: ["deploy", "multi-agent"]
created_at: "2026-05-17"
dependencies: ["wui1vo"]
---

## Pre-filter assets by target agent supported types

### Objective

Pattern expansion of seed y1i7w6 (sync must filter by `Agent.SupportedTypes`). Today the only `SupportsType` check is post-hoc inside `deployOne` for the single bound agent; no listing, profile, TUI, or bulk path pre-filters by target-agent type support, so cross-agent fan-out would attempt and fail N-1 times per asset instead of pre-filtering. Add per-target-agent type filtering before requests are built.

### Tasks

- [ ] `internal/deploy/sync.go` -- extend `SyncPlan` (or add `AgentSyncPlan`) with a Blocked/Skipped category for unsupported types
- [ ] `internal/deploy/deploy.go:464-507` -- add a per-agent `SupportsType` pre-filter (mirror `checkContextCollisions`) so unsupported (asset, agent) pairs are skipped-with-reason, not failed
- [ ] `cmd/deploy.go:165-174` -- pre-filter requests by target agent `SupportsType` when fanning out to multiple agents
- [ ] `internal/profile/manager.go:219-224,262-267,336-339` -- skip+report assets whose type the target agent does not support
- [ ] `internal/tui/deploy.go:354-360` -- add an `agent.SupportsType` filter alongside the `IsDeployable` filter in the asset picker
- [ ] `cmd/list.go:72-75` + `internal/tui/deploy.go:342-352` -- augment `FilterByAgent` (source-alias) with a `SupportedTypes` filter so unsupported types are not shown (e.g. `hooks` for copilot per `registry.go:53`)

### Acceptance criteria

- Sync/deploy plan lists unsupported (asset, agent) pairs as "skipped: <type> not supported by <agent>" before execution
- Profile deploy across agents does not produce hard failures for type-incompatible assets
- TUI deploy picker and `nd list` hide asset types the target/active agent does not support
- Existing single-agent deploy still rejects unsupported types via `deployOne` unchanged

### References

- Seed task: y1i7w6 -- `tasks/cli/y1i7w6-sync-assets-across-agents.md`
- `internal/agent/agent.go:25-33` (`SupportsType`), `internal/agent/registry.go:42,53`, `internal/deploy/sync.go`
- Related: wui1vo (single-agent assumptions)
