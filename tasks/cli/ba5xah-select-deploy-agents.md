---
title: "Let user select target agents for deploy"
id: "ba5xah"
status: pending
priority: high
type: feature
tags: ["deploy", "multi-agent"]
created_at: "2026-04-20"
---

## Let user select target agents for deploy

### Objective

Let users choose which detected coding agents to deploy assets to during the deploy flow. Currently, deployment targets the single default agent from `config.DefaultAgent`. This feature adds a multi-select agent picker so users can deploy to Claude Code, Copilot, or both in a single operation. It also adds a `default_deploy_agents` config field so users can persist their preferred targets and skip the picker when defaults are set.

### Tasks

- [ ] Add a `default_deploy_agents` field to `config.Config` (list of agent names); when set, skip the agent picker and deploy to all listed agents
- [ ] Add a new `deployPickAgents` step to `deployScreen` (between `deployPickType` and `deploySelectAssets`) that presents a multi-select of detected agents from the registry
- [ ] Filter the multi-select to only show agents where `Detected == true`; pre-select agents listed in `default_deploy_agents`
- [ ] Thread the selected agents through `startDeploy` so `DeployRequest` items are generated per agent; the deploy engine already accepts an `Agent` per request via the engine constructor, so either create one engine per agent or extend `DeployRequest` to carry an agent reference
- [ ] Update `deployBulkCmd` and result rendering to show which agent each asset was deployed to (e.g., "skill/foo -> claude-code", "skill/foo -> copilot")
- [ ] For the CLI path (`nd deploy`), add a `--agents` flag (comma-separated agent names) that overrides defaults; validate names against the registry
- [ ] Add unit tests: single agent selected, multiple agents selected, default agents config honored, unknown agent name rejected, no agents detected shows error
- [ ] Add TUI tests: agent picker renders only detected agents, pre-selects defaults, skips when only one agent detected

### Acceptance criteria

- When multiple agents are detected, the TUI deploy flow shows an agent picker step before asset selection
- Users can select one or more agents via multi-select (space/x to toggle, enter to confirm)
- When `default_deploy_agents` is configured, the picker is skipped and those agents are used automatically
- When only one agent is detected, the picker is skipped entirely (no unnecessary prompt)
- `nd deploy --agents claude-code,copilot` deploys to both agents in CLI mode
- Deploy results clearly indicate which agent each asset was deployed to
- An invalid agent name in `--agents` or `default_deploy_agents` produces a clear error
- Existing single-agent deploy behavior is preserved when no multi-agent config is set
- All new and existing deploy tests pass

### References

- https://GitHub.com/armstrongl/nd/issues/79
