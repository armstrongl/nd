---
title: "Let user select target agents for deploy"
id: "ba5xah"
status: completed
priority: high
type: feature
tags: ["deploy", "multi-agent"]
created_at: "2026-04-20"
verify:
  - type: bash
    run: "go build -o nd ."
  - type: bash
    run: "go test ./internal/config/... ./internal/sourcemanager/... ./internal/tui/... ./cmd/..."
  - type: bash
    run: "go test -race ./internal/tui/..."
  - type: bash
    run: "go test ./..."
  - type: bash
    run: "go test ./tests/integration/ -v -run Deploy"
  - type: bash
    run: "golangci-lint run"
  - type: assert
    check: "With two agents detected and no default_deploy_agents set, the TUI deploy flow shows a multi-select agent picker between the type picker and the asset picker; selecting both agents deploys every chosen asset once per agent."
  - type: assert
    check: "When default_deploy_agents is set (e.g. [claude-code, copilot]), the agent picker is skipped and those agents are used; when only one agent is detected the picker is skipped regardless of config."
  - type: assert
    check: "`nd deploy --agents claude-code,copilot greeting` deploys greeting to both agents; an unknown name in --agents or default_deploy_agents produces a clear error and a non-zero exit code."
  - type: assert
    check: "Existing single-agent behavior is byte-for-byte preserved when no --agents flag and no default_deploy_agents config are set."
context:
  - "internal/config/config.go"
  - "internal/config/validation.go"
  - "internal/sourcemanager/config.go"
  - "internal/agent/agent.go"
  - "internal/agent/registry.go"
  - "internal/deploy/deploy.go"
  - "cmd/app.go"
  - "cmd/deploy.go"
  - "internal/tui/deploy.go"
  - "internal/tui/services.go"
  - "internal/tui/testutil_test.go"
completed_at: 2026-05-20
---

## Let user select target agents for deploy

### Objective

Today every deploy targets exactly one agent. Both the CLI and TUI paths
build a single `deploy.Engine` bound to one `*agent.Agent` resolved by
`App.ActiveAgent()` (`cmd/app.go:75-92`, `cmd/app.go:95-114`), and the engine
stores that agent for its lifetime (`deploy.New(store, ag, backupDir)` —
`internal/deploy/deploy.go:48-62`; the agent is read in `deployOne` at
`internal/deploy/deploy.go:200-222`).

This task lets a user deploy the *same* selected assets to **multiple**
detected agents (Claude Code, Copilot, or both) in one operation, plus a
`default_deploy_agents` config field to persist preferred targets and skip the
picker. Implements GitHub issue #79 (https://GitHub.com/armstrongl/nd/issues/79
— "Allow users to select which of their detected and configured coding agents
to deploy assets to … single and multi-select … provide option to set default
agents").

Known agents are hard-coded in `agent.New` (`internal/agent/registry.go:35-58`):
`claude-code` (supports all deployable types) and `copilot` (supports only
`skill`, `agent`, `context`). An agent is "detected" when its binary is on PATH
*or* its global config dir exists (`Registry.Detect`,
`internal/agent/registry.go:117-155`); the result is the `Agent.Detected` bool
(`internal/agent/agent.go:21`).

#### Key constraint: engine is single-agent

`deploy.Engine` cannot deploy to multiple agents. The cleanest approach is to
build **one engine per selected agent** and run each agent's requests through
it. `App.DeployEngine()` caches a single engine for `ActiveAgent()`
(`cmd/app.go:95-114`); do **not** reuse that cached engine for other agents —
construct fresh engines via `deploy.New(app.StateStore(), ag, app.BackupDir)`
and wire the snapshot saver the same way `DeployEngine()` does
(`cmd/app.go:104-110`). Per-agent deployments are distinguished in state by
`state.Deployment.Agent` (set from `e.agent.Name` in
`internal/deploy/deploy.go:265-277`), and `removeOne` already filters by agent
(`internal/deploy/deploy.go:593-604`), so per-agent state is already supported.

### Tasks

- [x] **Config field.** Add `DefaultDeployAgents []string` to `config.Config`
  with tag `` `yaml:"default_deploy_agents,omitempty" json:"default_deploy_agents,omitempty"` ``
  in `internal/config/config.go:8-16` (alongside `DefaultAgent` at line 11).
  Add the matching pointer-free slice to `ProjectConfig`
  (`internal/config/config.go:39-46`): `DefaultDeployAgents []string` with
  `` `yaml:"default_deploy_agents,omitempty"` `` (slices use len>0 to detect
  "set", consistent with `Sources`/`Agents` there).
- [x] **Merge.** In `MergeConfigs` (`internal/sourcemanager/config.go:78-122`)
  add: `if len(project.DefaultDeployAgents) > 0 { merged.DefaultDeployAgents = project.DefaultDeployAgents }`
  near the `project.DefaultAgent` block (lines 88-90). Leave `DefaultConfig`
  (`internal/sourcemanager/config.go:18-26`) unchanged (empty slice = no default).
- [x] **Validate names.** In `Config.Validate`
  (`internal/config/validation.go:26-101`) add a block after the
  `DefaultAgent` check (lines 52-56): for each name in
  `c.DefaultDeployAgents`, verify it is one of the known agent names.
  Known names are `"claude-code"` and `"copilot"` per
  `internal/agent/registry.go:37,48`. To avoid an import cycle
  (`agent` imports `config`), either (a) add an exported
  `agent.KnownAgentNames() []string` in `internal/agent/registry.go` and call
  it from a non-`config` caller, or (b) validate against a small local
  allowlist constant in the `config` package — prefer (a) and validate at the
  CLI/sourcemanager boundary if an import cycle would result. Emit a
  `ValidationError{Field: "default_deploy_agents", Message: ...}` for unknown
  names (mirror the existing error style at lines 52-56).
- [x] **Engine factory helper.** Add a helper on `*App` in `cmd/app.go`
  (next to `DeployEngine`, `cmd/app.go:95-114`), e.g.
  `func (a *App) DeployEngineFor(ag *agent.Agent) (*deploy.Engine, error)`:
  build `deploy.New(a.StateStore(), ag, a.BackupDir)` and attach the snapshot
  saver exactly as lines 104-110 do. Refactor `DeployEngine()` to delegate to
  it for `ActiveAgent()` so the wiring stays single-sourced.
- [x] **TUI: new picker step.** In `internal/tui/deploy.go`:
  - Add `deployPickAgents` to the `deployStep` enum
    (`internal/tui/deploy.go:21-27`) ordered **between** `deployPickType` and
    `deploySelectAssets`. Updating iota will renumber later constants — that is
    fine, all comparisons use the named constants.
  - Add a `*huh.Form` agent multi-select field + a `selectedAgents []string`
    field to `deployScreen` (`internal/tui/deploy.go:51-94`), mirroring the
    asset multi-select built in `buildAssetForm`
    (`internal/tui/deploy.go:378-397`: `huh.NewMultiSelect[string]()` …
    `.WithTheme(huh.ThemeFunc(huh.ThemeCatppuccin))`).
  - Populate options from the registry: `reg, _ := ds.svc.AgentRegistry()`
    then `reg.Detect()` then iterate `reg.All()` keeping only entries where
    `a.Detected` is true (`internal/agent/agent.go:21`). The `Services`
    interface already exposes `AgentRegistry()`
    (`internal/tui/services.go:22`).
  - Wire the new step into `Update`'s step switch
    (`internal/tui/deploy.go:248-257`), `View`'s switch
    (`internal/tui/deploy.go:276-296`), `InputActive()`
    (`internal/tui/deploy.go:137-139`), and `FullHelpItems()`
    (`internal/tui/deploy.go:143-173` — reuse the `deploySelectAssets`
    multi-select help: `x/space toggle`, `enter confirm`, `esc back`).
  - **Skip logic:** after the type form completes
    (`updatePickType`, `internal/tui/deploy.go:316-318`, currently calls
    `ds.startScan()`), branch: if `config.DefaultDeployAgents` is non-empty
    use those names; else if exactly one agent is detected use it; otherwise
    enter `deployPickAgents`. When skipping the picker, proceed straight to
    the existing scan/`deploySelectAssets` flow with the resolved agents
    stored on `ds`. Read config via
    `ds.svc.SourceManager()` then `sm.Config()` (this is how the symlink
    strategy is already read at `internal/tui/deploy.go:444-449`).
  - Add an `updatePickAgents(msg)` handler mirroring `updateSelectAssets`
    (`internal/tui/deploy.go:400-427`): delegate to the form, on
    `huh.StateCompleted` store `ds.selectedAgents` and advance to the scan
    step; on `huh.StateAborted` or `esc` emit `BackMsg{}`; reject an empty
    selection (do not advance).
- [x] **TUI: multi-agent deploy.** In `startDeploy`
  (`internal/tui/deploy.go:430-494`) the request loop at lines 454-468 builds
  `[]deploy.DeployRequest` for one agent. Resolve `ds.selectedAgents` to
  `[]*agent.Agent` via `reg.Get(name)` (`internal/agent/registry.go:176-183`).
  Build, per selected agent, the same request set, run each agent's batch
  through that agent's engine (the `Services` interface only exposes a single
  `DeployEngine()` — add `DeployEngineFor(*agent.Agent)` to the `Services`
  interface in `internal/tui/services.go:16-45` and implement it on the mock
  at `internal/tui/testutil_test.go:16-44,73-78`), and merge all
  `BulkDeployResult`s before transitioning to `deployResult`. The dry-run
  branch (`internal/tui/deploy.go:471-477`) and the bulk command
  `deployBulkCmd` (`internal/tui/deploy.go:497-515`) must be extended to carry
  the target agent name so results can be labeled per agent.
- [x] **Result rendering.** In `buildResultContent`
  (`internal/tui/deploy.go:542-588`) the success/fail lines
  (lines 564-567, 574-577) print `type/name`. Append the target agent, e.g.
  `skill/foo -> claude-code`. Use `state.Deployment.Agent`
  (set in `internal/deploy/deploy.go:274`) for succeeded entries; for failed
  entries thread the agent name through `deploy.DeployError` /
  `deployBulkCmd`. Apply the same to the dry-run branch
  (`internal/tui/deploy.go:546-555`).
- [x] **CLI `--agents` flag.** In `newDeployCmd`
  (`cmd/deploy.go:15-295`) register `cmd.Flags().StringSliceVar(&agents,
  "agents", nil, "comma-separated target agents (overrides config default_deploy_agents)")`
  next to the existing flags (`cmd/deploy.go:283-293`). Resolution precedence:
  `--agents` flag > `config.DefaultDeployAgents` (read from
  `app.SourceManager().Config()`, see `cmd/deploy.go:152-157` for the existing
  config-read pattern) > current single `ActiveAgent()` behavior. Validate
  every name with `reg.Get(name)` and require `ag.Detected` after
  `reg.Detect()` (same checks `ActiveAgent` does at `cmd/app.go:80-90`);
  return `withExitCode(nd.ExitInvalidUsage, err)` on an unknown/undetected
  name (mirror the existing usage-error pattern at `cmd/deploy.go:108`).
  Build and run requests per agent via the new `DeployEngineFor` helper, then
  merge results before the existing reporting block (`cmd/deploy.go:198-261`).
  Add a `cmd.RegisterFlagCompletionFunc("agents", ...)` returning known agent
  names (analogous to the `type` completion at `cmd/deploy.go:284-290`).
- [x] **Unit tests — config:** in `internal/config/config_test.go` (follow the
  YAML round-trip pattern at `internal/config/config_test.go:11-40`) assert
  `default_deploy_agents` round-trips; in `internal/config/validation_test.go`
  (or `config_test.go`) assert an unknown agent name yields a
  `ValidationError` with `Field == "default_deploy_agents"` and a valid set
  yields none.
- [x] **Unit tests — merge:** in `internal/sourcemanager/` assert
  `MergeConfigs` replaces `DefaultDeployAgents` from a non-empty project value
  and preserves the global value when the project slice is empty.
- [x] **TUI tests:** in `internal/tui/deploy_test.go` (use `newMockServices()`
  and the existing `newTestDeployScreen` helper at
  `internal/tui/deploy_test.go:50-69`; override `agentRegistryFn` /
  `defaultAgentFn` on the mock — see `internal/tui/testutil_test.go:59-71`):
  picker renders only `Detected==true` agents; picker is skipped when exactly
  one agent is detected; picker is skipped and config agents used when
  `DefaultDeployAgents` is set; single-agent selection and two-agent selection
  both produce the correct per-agent request sets.
- [x] **Integration test:** add to `tests/integration/deploy_test.go` (use
  `setupIntegrationEnv` + `runND`, see `tests/integration/deploy_test.go:9-37`):
  `nd deploy --agents <name> greeting` succeeds and `nd deploy --agents bogus
  greeting` exits non-zero with a clear message. (Detection in the sandboxed
  env may only find `claude-code`; assert behavior accordingly rather than
  asserting copilot is present.)

### Acceptance criteria

- When ≥2 agents are detected and `default_deploy_agents` is unset, the TUI
  deploy flow inserts a multi-select agent picker between the type picker and
  the asset picker; space/x toggles, enter confirms.
- When `default_deploy_agents` is configured, the picker is skipped and those
  agents are used. When exactly one agent is detected, the picker is skipped
  regardless of config.
- Selecting N agents deploys each chosen asset once per agent; state records
  one `state.Deployment` per (asset, agent) with the correct `Agent` field.
- `nd deploy --agents claude-code,copilot greeting` deploys to both agents in
  CLI mode; `--agents` overrides `default_deploy_agents`.
- Deploy results (TUI and CLI, including dry-run) show the target agent for
  each asset (e.g. `skill/foo -> copilot`).
- An invalid agent name in `--agents` or `default_deploy_agents` produces a
  clear error; `--agents` with a bad name exits non-zero
  (`nd.ExitInvalidUsage`).
- With no `--agents` flag and no `default_deploy_agents`, behavior is identical
  to today (single `ActiveAgent`).
- `go build -o nd .`, `go test ./...`, `go test -race ./internal/tui/...`,
  `golangci-lint run`, and the integration deploy tests all pass.

### References

- GitHub issue: https://GitHub.com/armstrongl/nd/issues/79
- Close this issue when the task is completed.
- Config struct + merge: `internal/config/config.go:8-46`,
  `internal/config/validation.go:26-101`,
  `internal/sourcemanager/config.go:78-122`
- Agent model/registry: `internal/agent/agent.go:11-33` (`Detected`,
  `SupportsType`), `internal/agent/registry.go:35-58` (known agents),
  `:102-106` (`All`), `:117-155` (`Detect`), `:176-183` (`Get`)
- Single-agent engine binding: `internal/deploy/deploy.go:30-62`,
  `:100-108` (`DeployRequest`), `:265-277` (`Agent` written to state),
  `:466-507` (`DeployBulk`)
- App service wiring: `cmd/app.go:75-114` (`ActiveAgent`, `DeployEngine`)
- TUI deploy flow: `internal/tui/deploy.go:21-27` (steps),
  `:301-326` (`updatePickType`), `:378-397` (`buildAssetForm` multi-select
  pattern), `:400-427` (`updateSelectAssets`), `:430-515` (`startDeploy`,
  `deployBulkCmd`), `:542-588` (`buildResultContent`)
- CLI deploy: `cmd/deploy.go:15-295` (flags at `:283-293`, request build at
  `:165-174`, reporting at `:198-261`)
- Services interface + mock: `internal/tui/services.go:16-45`,
  `internal/tui/testutil_test.go:16-78`
- Test patterns: `internal/config/config_test.go:11-40`,
  `internal/tui/deploy_test.go:50-69`,
  `tests/integration/deploy_test.go:9-37`
