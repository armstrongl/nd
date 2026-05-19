---
title: "Pre-filter assets by target agent supported types"
id: "rglxhu"
status: pending
priority: high
type: feature
tags: ["deploy", "multi-agent"]
created_at: "2026-05-17"
dependencies: ["wui1vo"]
verify:
  - type: bash
    run: "go build -o nd ."
  - type: bash
    run: "go test ./internal/deploy/... ./internal/profile/... ./internal/agent/... ./internal/asset/... ./cmd/..."
  - type: bash
    run: "go test -race ./..."
  - type: bash
    run: "golangci-lint run"
  - type: bash
    run: "go test ./tests/integration/ -v -run Deploy"
  - type: assert
    check: "Deploying a hooks/commands/rules/output-styles asset to copilot is reported as skipped-with-reason (type not supported), not a hard failure"
  - type: assert
    check: "TUI deploy picker and `nd list` hide asset types the active agent does not support (e.g. hooks for copilot)"
  - type: assert
    check: "Single-agent deploy of an unsupported type via Engine.Deploy() still returns UnsupportedTypeError unchanged"
context:
  - "internal/agent/agent.go"
  - "internal/agent/registry.go"
  - "internal/deploy/deploy.go"
  - "internal/deploy/sync.go"
  - "internal/profile/manager.go"
  - "internal/tui/deploy.go"
  - "internal/asset/index.go"
  - "internal/nd/asset_type.go"
  - "cmd/deploy.go"
  - "cmd/list.go"
  - "cmd/app.go"
---

## Pre-filter assets by target agent supported types

### Objective

When deployment fans out to more than one coding agent (the multi-agent
capability delivered by dependency `wui1vo`), each agent supports a different
subset of asset types. Today `claude-code` supports all deployable types
(`internal/agent/registry.go:42`, `SupportedTypes: nd.DeployableAssetTypes()` =
skills, agents, commands, output-styles, rules, context, hooks) while `copilot`
supports only `{skills, agents, context}` (`internal/agent/registry.go:53`),
so deploying a `hooks`, `commands`, `rules`, or `output-styles` asset to
copilot is invalid.

The only type-support check today is post-hoc and single-agent: `deployOne`
at `internal/deploy/deploy.go:200-202` calls `e.agent.SupportsType(...)` on the
one `*agent.Agent` bound to the `Engine` (set in `cmd/app.go:104` via
`deploy.New(sstore, ag, ...)`), returning `*UnsupportedTypeError`. In bulk,
`DeployBulk` at `internal/deploy/deploy.go:491-494` catches that error and only
tags `DeployError.UnsupportedType = true`; the asset still counts as a *failure*
and `cmd/deploy.go:252-260` returns `nd.ExitPartialFailure`. No listing,
profile, TUI, or sync path filters by target-agent type support *before*
building requests, so once `wui1vo` adds N-agent fan-out, each cross-agent
asset would be attempted and fail N-1 times instead of being pre-filtered and
reported as a deliberate skip.

This task is the type-aware pre-filter layer requested by seed `y1i7w6`
(`tasks/cli/y1i7w6-sync-assets-across-agents.md`, subtasks at its lines 22-23:
"filter to only asset types supported by each target agent (check
`Agent.SupportedTypes`)" and "showing what will be created, skipped, or
**blocked** (type not supported)"). Add per-target-agent `SupportsType`
filtering before requests are built across deploy/profile/sync/TUI/list, and
turn "unsupported type" into an explicit skipped-with-reason category rather
than a failure.

Scope note: this builds on `wui1vo` (catalogs/refactors single-agent
assumption sites so `DeployRequest`/engine construction carries an explicit
target agent and deploy fans out per agent). Implement against the per-agent
target API that `wui1vo` introduces; do not re-implement the fan-out itself.
Do not change `internal/agent/agent.go:26-33` (`SupportsType`) — reuse it.

### Tasks

- [ ] `internal/deploy/sync.go` — this file currently only declares
      `SyncPlan` (lines 6-10: `Repairs`, `Removes`, `Healthy`) and `SyncAction`
      (lines 13-17); there is no agent-sync consumer yet. Add a `Skipped`
      category for `(asset, agent)` pairs whose type the target agent does not
      support. Add a `[]SyncSkip` field to `SyncPlan` (or a new `AgentSyncPlan`
      type), where `SyncSkip` records the asset identity, target agent name,
      asset type, and a reason string. Keep the existing `SyncPlan` fields
      unchanged so health-sync (`cmd/sync.go`) is unaffected.
- [ ] `internal/deploy/deploy.go:466-507` (`DeployBulk`) — add a per-request
      target-agent `SupportsType` pre-filter before the `deployOne` loop at
      `deploy.go:482-499`, mirroring the existing pre-scan pattern
      `checkContextCollisions` (`deploy.go:307-328`, called at
      `deploy.go:478-480`). Requests whose type is unsupported by the target
      agent must be diverted into a skipped-with-reason result, NOT passed to
      `deployOne` (which would produce a failure). Add a `Skipped []DeployError`
      (or a dedicated `Skipped []DeploySkip`) field to `BulkDeployResult`
      (`deploy.go:140-144`) and route unsupported pairs there with a reason
      like `<type> not supported by <agent>`. Resolve the target agent per
      request via the agent reference `wui1vo` adds to `DeployRequest`
      (`deploy.go:101-108`) — if `wui1vo` instead uses one-engine-per-agent,
      filter using `e.agent.SupportsType`.
- [ ] `cmd/deploy.go:165-174` — this block builds `[]deploy.DeployRequest`
      from resolved assets; today it is single-agent (no fan-out exists here
      yet — that arrives via `wui1vo`). After `wui1vo`'s per-agent fan-out is
      in place, skip building requests for `(asset, agent)` pairs where
      `agent.SupportsType(asset.Type)` is false, and surface them through the
      skipped path. Update the human output at `cmd/deploy.go:224-239` (which
      currently prints `Skipped N asset(s) (unsupported by agent X)` and counts
      `f.UnsupportedType` failures): unsupported pairs must be reported as
      intentional skips and must NOT trigger `nd.ExitPartialFailure` at
      `cmd/deploy.go:252-260` when they are the only non-success.
- [ ] `internal/profile/manager.go:219-224` (`Switch`),
      `internal/profile/manager.go:262-267` (`DeployProfile`),
      `internal/profile/manager.go:336-341` (`Restore`) — each of these is a
      `deployReqs = append(deployReqs, deploy.DeployRequest{...})` block that
      builds requests from profile/snapshot assets. Before appending, skip
      assets whose type the target agent does not support and record them on
      the result (`SwitchResult`/`RestoreResult`, `manager.go:26-44`) in a new
      `SkippedUnsupported []ProfileAsset` field, analogous to the existing
      `MissingAssets`/`SkippedPinned` fields. Profile deploy/restore across
      type-incompatible agents must not produce hard failures.
- [ ] `internal/tui/deploy.go:354-360` — the asset picker filters
      `allAssets` to deployable types via `a.Type.IsDeployable()`. Add an
      `agent.SupportsType(a.Type)` check alongside it, using the active agent
      resolved at `internal/tui/deploy.go:342-345` (`svc.ActiveAgent()`), so
      types the active agent does not support never appear in the picker.
- [ ] `cmd/list.go:71-75` and `internal/tui/deploy.go:342-352` — both call
      `index.FilterByAgent(agentAlias)` / `index.ByTypeFiltered(...)`
      (`internal/asset/index.go:98-124`), which filter only by source
      `GroupDir`/alias, not by supported types. Add a supported-types filter
      after `FilterByAgent`/`ByTypeFiltered` so unsupported types are not
      listed for the active agent (e.g. `hooks`, `commands`, `rules`,
      `output-styles` are hidden when the active agent is `copilot`). Prefer
      filtering in the caller using the resolved `*agent.Agent` (the index has
      no agent dependency); do not add an agent dependency to
      `internal/asset/index.go`.

### Acceptance criteria

- `BulkDeployResult` exposes a skipped-with-reason category; an
  `(asset, agent)` pair whose type is unsupported lands there with reason text
  containing the type and agent name, and is excluded from `Failed`.
- `nd deploy` of an unsupported-type asset to an agent that lacks that type
  (e.g. a `hooks` asset to `copilot`) prints an explicit "skipped: <type> not
  supported by <agent>" line and exits 0 when that skip is the only
  non-success (no `ExitPartialFailure`).
- Profile `Switch`/`DeployProfile`/`Restore` populate a
  `SkippedUnsupported` list for type-incompatible assets and produce no
  `DeployError` entries for them.
- TUI deploy picker and `nd list` omit asset types the active agent does not
  support (verifiable: with active agent `copilot`, no `hooks`/`commands`/
  `rules`/`output-styles` entries appear; with `claude-code`, all appear).
- Single-agent `Engine.Deploy()` of an unsupported type still returns
  `*deploy.UnsupportedTypeError` from `deployOne`
  (`internal/deploy/deploy.go:200-202`) — that path is unchanged.
- `go build -o nd .`, `go test ./...`, `go test -race ./...`, and
  `golangci-lint run` all pass; new pre-filter behavior has unit tests
  (deploy pre-filter, profile skip list, list/TUI hiding) covering both
  agents (`claude-code` = all types, `copilot` = skills/agents/context).

### References

- GitHub issue: https://GitHub.com/armstrongl/nd/issues/123
- Close this issue when the task is completed.
- Seed task: `y1i7w6` — `tasks/cli/y1i7w6-sync-assets-across-agents.md`
  (subtasks at lines 22-23 require the supported-type filter and a
  "blocked: type not supported" plan category this task delivers).
- Dependency: `wui1vo` — `tasks/cli/wui1vo-expand-single-agent-assumptions.md`
  (catalogs single-agent assumption sites; adds the per-agent target on
  `DeployRequest`/engine and the deploy fan-out this task filters within).
- Type-support API: `internal/agent/agent.go:11-33`
  (`Agent.SupportedTypes` field, `Agent.SupportsType` method — reuse, do not
  modify).
- Per-agent supported types: `internal/agent/registry.go:35-58`
  (`claude-code` → `nd.DeployableAssetTypes()` at line 42; `copilot` →
  `{nd.AssetSkill, nd.AssetAgent, nd.AssetContext}` at line 53).
- Asset type catalog: `internal/nd/asset_type.go:6-32`
  (`DeployableAssetTypes()` = skills, agents, commands, output-styles, rules,
  context, hooks; plugins excluded).
- Existing single-agent check (the only `SupportsType` call today):
  `internal/deploy/deploy.go:200-202` (`deployOne`); bulk classification at
  `internal/deploy/deploy.go:491-494`; CLI output at
  `cmd/deploy.go:224-239`, `cmd/deploy.go:252-260`.
- Pre-scan pattern to mirror: `internal/deploy/deploy.go:307-328`
  (`checkContextCollisions`), invoked at `internal/deploy/deploy.go:478-480`
  before the `deployOne` loop.
- Active-agent resolution: `cmd/app.go:75-92` (`App.ActiveAgent`),
  `cmd/app.go:95-114` (`App.DeployEngine`, binds one `*agent.Agent`).
- Index filtering (alias only, no type filter): `internal/asset/index.go:96-124`
  (`FilterByAgent`, `ByTypeFiltered`).
