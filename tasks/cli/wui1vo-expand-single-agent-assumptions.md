---
title: "Catalog and refactor single-agent assumption sites"
id: "wui1vo"
status: pending
priority: high
type: feature
tags: ["deploy", "multi-agent"]
created_at: "2026-05-17"
dependencies: ["ba5xah"]
verify:
  - type: bash
    run: "go build -o nd ."
  - type: bash
    run: "go test ./..."
  - type: bash
    run: "go test -race ./internal/tui/... ./internal/deploy/..."
  - type: bash
    run: "go test ./tests/integration/ -v"
  - type: bash
    run: "golangci-lint run"
  - type: assert
    check: "`nd deploy --agents claude-code,copilot greeting` deploys greeting once per agent; state records one state.Deployment per (asset, agent) with the correct Agent field; results label each line with its target agent."
  - type: assert
    check: "config.default_deploy_agents round-trips through YAML, merges project-over-global, and an unknown agent name yields a ValidationError with Field == \"default_deploy_agents\"."
  - type: assert
    check: "`nd status` and `nd list` reflect deployments across all detected agents, not just ActiveAgent()."
  - type: assert
    check: "`nd remove greeting` and `nd uninstall` only delete deployments for the resolved agent (RemoveRequest.Agent populated); a deployment of the same asset on a different agent is untouched."
  - type: assert
    check: "With no --agents flag and no default_deploy_agents config, behavior is byte-for-byte identical to today (single ActiveAgent); cmd/doctor.go is unchanged."
context:
  - "internal/deploy/deploy.go"
  - "internal/config/config.go"
  - "internal/config/validation.go"
  - "internal/sourcemanager/config.go"
  - "internal/agent/registry.go"
  - "internal/agent/agent.go"
  - "cmd/app.go"
  - "cmd/doctor.go"
  - "cmd/deploy.go"
  - "cmd/status.go"
  - "cmd/list.go"
  - "cmd/sync.go"
  - "cmd/remove.go"
  - "cmd/uninstall.go"
  - "cmd/init_cmd.go"
  - "internal/tui/deploy.go"
  - "internal/profile/manager.go"
  - "tasks/cli/ba5xah-select-deploy-agents.md"
---

## Catalog and refactor single-agent assumption sites

### Objective

The entire deploy/remove/status/list/sync/profile pipeline assumes exactly one
target agent. The single-agent binding is created in `cmd/app.go`:
`App.DeployEngine()` (`cmd/app.go:94-114`) calls `App.ActiveAgent()`
(`cmd/app.go:73-92`) -> `Registry.Default()` (`internal/agent/registry.go:188-208`),
then builds one `deploy.New(sstore, ag, a.BackupDir)` engine that stores that
single `*agent.Agent` for its lifetime (`internal/deploy/deploy.go:30-62`; the
agent is read in `deployOne` at `internal/deploy/deploy.go:200-201` and written
into `state.Deployment.Agent` at `internal/deploy/deploy.go:274`). Every command
that calls `app.DeployEngine()` therefore silently scopes to one agent.

Sibling seed `ba5xah` (`tasks/cli/ba5xah-select-deploy-agents.md`, this task's
only dependency) delivers the user-facing multi-agent *picker* and the
`default_deploy_agents` config field, the `DeployEngineFor(*agent.Agent)` engine
factory, and per-agent fan-out for the `deploy` path (CLI + TUI). `ba5xah` is a
hard prerequisite: this task assumes `App.DeployEngineFor(ag *agent.Agent)
(*deploy.Engine, error)` and `config.Config.DefaultDeployAgents []string`
already exist. If they do not yet exist when this task starts, `ba5xah` is
incomplete — do not duplicate that work here.

This task **catalogs and refactors the remaining single-agent assumption sites**
that `ba5xah` does not touch: `status`, `list`, `sync`, `remove`, `uninstall`,
`init` built-in deploy, and the profile manager. Each must iterate the detected
agents instead of a single `DeployEngine()`. **`cmd/doctor.go:91-109` already
iterates all known agents correctly via `reg.All()` — use it as the canonical
model. Do not modify `cmd/doctor.go`.**

#### Verified single-agent assumption sites (the catalog)

| Site | What's wrong | Fix direction |
|------|--------------|---------------|
| `internal/deploy/deploy.go:30-62`, `:48-62` | `Engine` holds one `*agent.Agent`; `New` binds it | One engine per agent (use `DeployEngineFor` from `ba5xah`). No struct change needed. |
| `internal/deploy/deploy.go:583-620` (`removeOne`) | Empty `RemoveRequest.Agent` matches **any** agent (`:594-604`) — latent multi-agent data loss | Callers must populate `RemoveRequest.Agent` (see below) |
| `cmd/status.go:31` | `eng, _ := app.DeployEngine()` then `eng.Status()` (`internal/deploy/health.go:234`) only sees the active agent's engine | Iterate `reg.All()` detected agents; build a per-agent engine via `app.DeployEngineFor(ag)`; merge `Status()` entries; add an agent column/field |
| `cmd/list.go:59-75` | `eng, _ := app.DeployEngine()` builds `deployedSet`; `app.ActiveAgent()` (`:71-74`) sets `agentAlias` for `index.FilterByAgent` | Cross-reference deployments across all detected agents |
| `cmd/sync.go:61` | `eng, _ := app.DeployEngine()`; `eng.Check()`/`eng.Sync()` (`internal/deploy/health.go:44,168`) only repair the active agent's symlinks | Per-agent engine; sync each detected agent |
| `cmd/remove.go:59` + `:113-122` | `RemoveRequest` built **without** `Agent` (`:113-122`); resolves against one engine's `Status()` | Populate `RemoveRequest.Agent` from the resolved deployment's `Deployment.Agent` |
| `cmd/uninstall.go:71` + `:76-83` | `RemoveRequest` built without `Agent` (`:78-82`) from `st.Deployments` | Set `RemoveRequest.Agent = d.Agent` for each deployment |
| `cmd/init_cmd.go:212-226` + `:229-237` | `reg.Default()` then `deploy.New(sstore, ag, backupDir)`; built-in deploy ignores `default_deploy_agents` | Honor `config.DefaultDeployAgents`; fan out per agent |
| `internal/profile/manager.go:187-191`, `:219-224`, `:262-267`, `:310-314`, `:336-342` | Profile `Switch`/`DeployProfile`/`Restore` build `DeployRequest`/`RemoveRequest` with no agent and run through one injected `engine` | Make profile deploy/restore agent-aware (see Tasks) |

### Tasks

- [ ] **Confirm prerequisites from `ba5xah` exist.** Verify
  `func (a *App) DeployEngineFor(ag *agent.Agent) (*deploy.Engine, error)` is
  defined in `cmd/app.go` (next to `DeployEngine()` at `cmd/app.go:94-114`) and
  that `config.Config` has `DefaultDeployAgents []string`
  (`internal/config/config.go:8-16`). If absent, stop and finish `ba5xah` first.
- [ ] **Shared resolution helper.** Add one helper that the refactored commands
  reuse, e.g. `func (a *App) DeployAgents() ([]*agent.Agent, error)` on `*App`
  in `cmd/app.go` (next to `ActiveAgent()` / `DeployEngineFor()`). Precedence,
  mirroring `ActiveAgent()` (`cmd/app.go:73-92`): (1) `--agent` flag (single,
  via `reg.Get`+`reg.Detect()`+`ag.Detected`, same checks as
  `cmd/app.go:80-90`); else (2) `config.DefaultDeployAgents` from
  `app.SourceManager().Config()` (read pattern: `cmd/deploy.go:152-157`),
  each validated with `reg.Get(name)` + `ag.Detected`; else (3) **all detected
  agents** — `reg.Detect()` then iterate `reg.All()` keeping `ag.Detected`
  (`internal/agent/agent.go`: `Detected` bool; iteration model is
  `cmd/doctor.go:91-109`). Returns an error if no agents are detected (reuse
  the message style from `Registry.Default()`,
  `internal/agent/registry.go:207`).
- [ ] **`cmd/status.go`.** Replace the single `eng, err := app.DeployEngine()`
  (`cmd/status.go:31-34`) with: for each agent from `app.DeployAgents()`, build
  `eng, _ := app.DeployEngineFor(ag)`, call `eng.PruneAll()` (currently
  `cmd/status.go:37`) and `eng.Status()` (`cmd/status.go:45`,
  `internal/deploy/health.go:234`), and accumulate `[]StatusEntry`. Add an
  `Agent` field to `statusDisplay` (`cmd/status.go:122-130`) populated from
  `e.Deployment.Agent` (treat empty as `"claude-code"` per the v1→v2 migration
  rule documented at `internal/deploy/deploy.go:595-600`). Render it in the
  human path (the `printHuman` row at `cmd/status.go:112-114`) and include it
  in the JSON struct (`cmd/status.go:76-82`). Since per-agent state lives in one
  shared `deployments.yaml` (`cmd/app.go:148`), an alternative is a single
  `eng.Status()` call followed by *not* filtering by agent — but you must still
  surface the `Agent` field; prefer whichever keeps the agent label correct.
- [ ] **`cmd/list.go`.** The cross-reference `deployedSet` built from one
  `eng.Status()` (`cmd/list.go:59-69`) must reflect every detected agent.
  Iterate `app.DeployAgents()` and union the `Status()` entries into
  `deployedSet`. The `agentAlias` used for `index.FilterByAgent`
  (`cmd/list.go:71-75`) currently comes from `app.ActiveAgent()`; when multiple
  agents are in play, union the assets visible to each agent's `SourceAlias`
  (dedupe by `type/name`) so the listing isn't narrowed to one agent.
- [ ] **`cmd/sync.go`.** Replace `eng, err := app.DeployEngine()`
  (`cmd/sync.go:61-64`) with a loop over `app.DeployAgents()`: per agent build
  `app.DeployEngineFor(ag)`, then call `eng.Check()` (dry-run branch,
  `cmd/sync.go:67`) or `eng.Sync()` (`cmd/sync.go:85`,
  `internal/deploy/health.go:168`). Merge `SyncResult.Repaired`/`.Removed`
  across agents before the reporting/oplog block (`cmd/sync.go:90-99`).
- [ ] **`cmd/remove.go` — populate `RemoveRequest.Agent`.** At
  `cmd/remove.go:113-122` the `deploy.RemoveRequest` is built with no `Agent`.
  Set `Agent: d.Agent` where `d := dep.Deployment` (the resolved deployment
  found by `findDeployedAsset`, `cmd/remove.go:72-77`). Treat an empty
  `d.Agent` as `"claude-code"` to match the migration rule in
  `removeOne` (`internal/deploy/deploy.go:595-604`). This closes the
  data-loss path where an empty `RemoveRequest.Agent` matches *any* agent's
  deployment (`internal/deploy/deploy.go:594-604`).
- [ ] **`cmd/uninstall.go` — populate `RemoveRequest.Agent`.** At
  `cmd/uninstall.go:76-83` the loop builds `RemoveRequest` from
  `st.Deployments` with no `Agent`. Add `Agent: d.Agent` (the per-deployment
  agent already recorded in state). Uninstall removes everything, but the field
  must be set so removal targets the exact recorded deployment rather than
  relying on the any-agent fallback.
- [ ] **`cmd/init_cmd.go` — built-in deploy honors `default_deploy_agents`.**
  In `deployBuiltinAssets` (`cmd/init_cmd.go:160-253`): instead of
  `reg.Default()` (`cmd/init_cmd.go:212-221`) + `deploy.New(sstore, ag,
  backupDir)` (`cmd/init_cmd.go:224-226`), resolve the target agent set the
  same way as the shared helper (config `DefaultDeployAgents` if set, else
  `reg.Default()` for back-compat — do **not** fan out to every detected agent
  at init unless `default_deploy_agents` explicitly lists them). For each
  resolved agent build a fresh `deploy.New(sstore, ag, backupDir)` (or the
  `DeployEngineFor` helper) and run the request set
  (`cmd/init_cmd.go:229-242`) per agent; sum `bulkResult.Succeeded` across
  agents for the reported count (`cmd/init_cmd.go:244`).
- [ ] **`internal/profile/manager.go` — agent-aware deploy/restore.** The
  `Manager` methods take a single injected `engine DeployEngine`. The
  agent-blind `RemoveRequest`s are at `internal/profile/manager.go:187-191`
  (`Switch`) and `:310-314` (`Restore`); agent-blind `DeployRequest`s at
  `:219-224` (`Switch`), `:262-267` (`DeployProfile`), `:336-342` (`Restore`).
  Restore's intent is to recreate the *snapshot's* exact deployments — the
  snapshot entries carry `Agent` (`state.Deployment.Agent`), so when rebuilding
  remove/deploy requests in `Restore` (`internal/profile/manager.go:307-342`)
  thread `entry.Agent` / `d.Agent` through, and route each request through an
  engine bound to that agent. Choose the lowest-churn approach: either (a)
  change the profile API to accept an `engineFor func(*agent.Agent)
  (*deploy.Engine, error)` instead of a single `engine`, or (b) group requests
  by agent and call the matching engine per group. Document the chosen approach
  in the worklog (`taskmd worklog wui1vo --add ...`). Caller wiring is in
  `cmd/profile*.go` / `cmd/snapshot*.go` — update those call sites to pass
  `app.DeployEngineFor` instead of `app.DeployEngine()`.
- [ ] **Unit tests.** Add tests proving each refactored site is agent-aware:
  - `internal/deploy` already exercises `removeOne` agent filtering
    (`internal/deploy/deploy.go:594-604`); add a test asserting a populated
    `RemoveRequest.Agent` does NOT remove a same-identity deployment whose
    `Deployment.Agent` differs.
  - `cmd/status_test.go` / `cmd/remove_test.go` / `cmd/uninstall_test.go`:
    deployments for two agents -> status lists both with correct `Agent`;
    remove of agent A's deployment leaves agent B's intact.
  - `internal/profile`: restore of a snapshot containing per-agent deployments
    recreates each on its recorded agent.
- [ ] **Integration test.** Add to `tests/integration/status_sync_test.go`
  (use `setupIntegrationEnv` + `runND`, pattern at
  `tests/integration/deploy_test.go:9-37`): deploy `greeting`, then
  `nd status` shows it; `nd remove greeting --yes` followed by `nd status`
  shows no deployment; assert exit codes. (The sandboxed env typically only
  detects `claude-code`; assert behavior accordingly rather than requiring
  `copilot`.)

### Acceptance criteria

- A single shared resolver (`App.DeployAgents()` or equivalent) returns the
  target agent list with precedence: `--agent` > `config.default_deploy_agents`
  > all detected agents; no detected agents yields a clear error.
- `nd status` and `nd list` reflect deployments across **all** detected agents,
  and `nd status` surfaces the target agent per deployment (human + `--json`).
- `nd sync` repairs symlinks for every detected agent, not just the active one.
- `nd remove` and `nd uninstall` populate `RemoveRequest.Agent`; removing an
  asset deployed to agent A never deletes the same-named deployment on agent B
  (verified against `internal/deploy/deploy.go:594-604`).
- `nd init` built-in deploy honors `config.default_deploy_agents` when set,
  otherwise preserves single-default behavior.
- Profile `Switch`/`DeployProfile`/`Restore` deploy/remove against the correct
  per-agent engine; restore recreates each snapshot deployment on its recorded
  `Agent`.
- With no `--agent` flag and no `default_deploy_agents`, every command behaves
  byte-for-byte as today (single `ActiveAgent`).
- `cmd/doctor.go` is unchanged.
- `go build -o nd .`, `go test ./...`, `go test -race ./internal/tui/...
  ./internal/deploy/...`, `go test ./tests/integration/ -v`, and
  `golangci-lint run` all pass.

### References

- GitHub issue: https://GitHub.com/armstrongl/nd/issues/132
- Close this issue when the task is completed.
- Dependency seed (prerequisite, defines `DeployEngineFor` +
  `DefaultDeployAgents` + the deploy picker):
  `tasks/cli/ba5xah-select-deploy-agents.md`
- Downstream task that depends on this one: `y1i7w6`
  (`tasks/cli/y1i7w6-sync-assets-across-agents.md`)
- Canonical multi-agent iteration model (do NOT modify):
  `cmd/doctor.go:91-109`
- Single-agent engine binding: `internal/deploy/deploy.go:30-62`
  (`Engine`/`New`), `:100-108` (`DeployRequest`), `:146-152`
  (`RemoveRequest.Agent`), `:274` (`Agent` written to state), `:583-620`
  (`removeOne` agent filter, the data-loss site at `:594-604`)
- App service wiring: `cmd/app.go:73-92` (`ActiveAgent`), `:94-114`
  (`DeployEngine`), `:143-150` (shared `StateStore`)
- Engine query API: `internal/deploy/health.go:44` (`Check`), `:117`
  (`PruneAll`), `:168` (`Sync`), `:234` (`Status`)
- Registry: `internal/agent/registry.go:102-106` (`All`), `:117-155`
  (`Detect`), `:176-183` (`Get`), `:188-208` (`Default`);
  `internal/agent/agent.go` (`Detected`, `SupportsType`)
- Config + merge + validate (from `ba5xah`):
  `internal/config/config.go:8-16,39-46`,
  `internal/sourcemanager/config.go:78-122` (`MergeConfigs`, `DefaultAgent`
  block at `:88-90`), `internal/config/validation.go:26-101` (`DefaultAgent`
  check at `:52-56`)
- Refactor sites: `cmd/status.go:31,45,122-130`, `cmd/list.go:59-75`,
  `cmd/sync.go:61,67,85`, `cmd/remove.go:113-122`, `cmd/uninstall.go:76-83`,
  `cmd/init_cmd.go:212-237`, `internal/profile/manager.go:187-191,219-224,
  262-267,310-314,336-342`
- Test patterns: `internal/config/config_test.go:11-40`,
  `tests/integration/deploy_test.go:9-37`,
  `tests/integration/status_sync_test.go`
