---
title: "Sync deployed assets across agents"
id: "y1i7w6"
status: pending
priority: high
type: feature
tags: ["deploy", "multi-agent"]
created_at: "2026-04-20"
---

## Sync deployed assets across agents

### Objective

Provide a way to sync deployed assets between coding agents so that skills, hooks, and configs deployed to one agent (e.g., Claude Code) can be replicated to another (e.g., Copilot) without manual re-deployment. This supports one-to-many sync patterns and guards against conflicts and data loss. Builds on the multi-agent support already shipped in the `copilot` branch (PR #92).

### Tasks

- [ ] Add a new `nd sync agents` CLI subcommand that accepts `--from <agent>` and `--to <agent>[,<agent>]` flags
- [ ] Add a TUI sync screen accessible from the main menu; present a two-step flow: (1) select source agent, (2) multi-select target agents
- [ ] Query the state store for all deployments belonging to the source agent; filter to only asset types supported by each target agent (check `Agent.SupportedTypes`)
- [ ] Build a sync plan (extend the existing `deploy.SyncPlan` or create a new `AgentSyncPlan`) showing what will be created, skipped (already deployed), or blocked (type not supported)
- [ ] Show the sync plan as a preview before execution; require explicit confirmation (or `--yes` for CLI)
- [ ] Execute sync by creating `DeployRequest` items for each target agent, reusing the existing `deploy.Engine.DeployBulk` API
- [ ] Handle conflicts: if an asset is already deployed to a target agent with different content, show a conflict prompt (reuse the existing conflict resolution pattern from `deployScreen`)
- [ ] Add a `--dry-run` flag to the CLI command and a dry-run mode in the TUI that shows the plan without executing
- [ ] Add unit tests: sync plan generation with supported/unsupported type filtering, conflict detection, one-to-many fan-out, empty source (no deployments) case
- [ ] Add TUI tests: source agent picker, target agent multi-select, plan preview rendering, conflict resolution flow

### Acceptance criteria

- `nd sync agents --from claude-code --to copilot` syncs all compatible deployed assets from Claude Code to Copilot
- Assets with types not supported by the target agent are skipped with a clear message (e.g., "hooks not supported by copilot, skipping")
- Assets already deployed to the target agent are skipped (no duplicate deployments)
- Conflicting assets (same name, different content) trigger a conflict prompt before overwriting
- `--dry-run` shows the full sync plan without writing any files or updating state
- One-to-many works: `--to copilot,other-agent` syncs to both targets in a single run
- The TUI sync screen shows source and target selection, then a preview, then results
- No data loss: sync never removes or modifies deployments on the source agent
- All new and existing deploy/sync tests pass

### References

- https://GitHub.com/armstrongl/nd/issues/80
