---
title: "Catalog and refactor single-agent assumption sites"
id: "wui1vo"
status: pending
priority: high
type: feature
tags: ["deploy", "multi-agent"]
created_at: "2026-05-17"
dependencies: ["ba5xah"]
---

## Catalog and refactor single-agent assumption sites

### Objective

Pattern expansion of seeds ba5xah and y1i7w6, which describe multi-agent deploy and cross-agent sync but enumerate almost none of the concrete single-agent assumption sites. The deploy/remove/status/list/profile pipeline is wired to exactly one agent via `App.DeployEngine()` -> `ActiveAgent()` -> `Registry.Default()`, and `deploy.Engine` holds a single `*agent.Agent`. This catalogs and refactors those sites so deployment can target N agents. `cmd/doctor.go:91-108` already iterates all agents correctly -- use it as the model.

### Tasks

- [ ] `internal/deploy/deploy.go:101-108` -- add an `Agent` reference to `DeployRequest` (or document one-engine-per-agent); thread through `deployOne:200,274`
- [ ] `internal/config/config.go:11,42` + `internal/sourcemanager/config.go:88-89` + `internal/config/validation.go:52` -- add `default_deploy_agents []string`, merge (project over global), validate
- [ ] `cmd/app.go:91,95-114` -- add a multi-agent resolution path alongside `ActiveAgent()`/`DeployEngine()`
- [ ] `internal/agent/registry.go:188-208` -- add `SelectedAgents()`/`DefaultAgents()` returning `[]Agent`
- [ ] `cmd/deploy.go:113,165-174,176-261,233-238` -- fan out `DeployRequest`s per selected agent; fix the single-agent "unsupported by agent X" message
- [ ] `internal/tui/deploy.go:454-466,482,653` -- add a `deployPickAgents` step + per-agent request fan-out
- [ ] `cmd/init_cmd.go:213,226,231-237` -- built-in deploy honors `default_deploy_agents`
- [ ] `internal/profile/manager.go:219-224,262-267,336-339` -- make profile deploy/restore agent-aware
- [ ] `cmd/status.go:31,45` / `cmd/list.go:59-75` / `cmd/sync.go:61,85` / `cmd/remove.go:59-121` / `cmd/uninstall.go:71-83` -- iterate agents (model on `cmd/doctor.go:91-108`) instead of a single `DeployEngine()`
- [ ] `cmd/remove.go:113-121` + `cmd/uninstall.go:76-83` -- populate `RemoveRequest.Agent` so removal is not agent-blind (empty `Agent` matches any agent per `deploy.go:594-604` -- latent multi-agent data-loss)

### Acceptance criteria

- `DeployRequest` (or engine construction) carries an explicit target agent
- `config.default_deploy_agents` accepted, merged, validated
- `nd deploy --agents a,b` deploys to both; results show per-agent target
- `nd status`/`nd list` reflect deployments across all detected agents
- `nd remove`/`nd uninstall` scope removal to the correct agent (no cross-agent collateral)
- Existing single-agent behavior preserved when no multi-agent config is set; `cmd/doctor.go` unchanged

### References

- Seed tasks: ba5xah -- `tasks/cli/ba5xah-select-deploy-agents.md`; y1i7w6 -- `tasks/cli/y1i7w6-sync-assets-across-agents.md`
- `internal/deploy/deploy.go`, `cmd/app.go`, `cmd/doctor.go:91-108` (reference)
