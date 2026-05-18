---
title: "Sync deployed assets across agents"
id: "y1i7w6"
status: pending
priority: high
type: feature
tags: ["deploy", "multi-agent"]
created_at: "2026-04-20"
dependencies: ["wui1vo"]
verify:
  - type: bash
    run: "go build -o nd ."
  - type: bash
    run: "go test ./internal/deploy/... ./internal/tui/... ./cmd/..."
  - type: bash
    run: "go test -race ./internal/tui/..."
  - type: bash
    run: "go test ./..."
  - type: bash
    run: "go test ./tests/integration/ -v -run Sync"
  - type: bash
    run: "golangci-lint run"
  - type: assert
    check: "`nd sync agents --from claude-code --to copilot --yes` deploys every claude-code deployment whose asset type is supported by copilot, skips types copilot does not support with a clear per-asset message, and skips assets already deployed to copilot."
  - type: assert
    check: "`nd sync agents --from claude-code --to copilot --dry-run` prints the full plan (create / skip-already / skip-unsupported) and writes no symlinks and no state changes (state file unchanged)."
  - type: assert
    check: "`nd sync agents --from claude-code --to copilot,other` fans out to both targets in one run; results are labelled per target agent."
  - type: assert
    check: "When an asset is already deployed to a target with a different SourcePath, the run prompts for confirmation (CLI: blocked unless --force/--yes; TUI: a huh.Confirm) before overwriting, mirroring deployScreen conflict resolution."
  - type: assert
    check: "No source-agent deployment is ever removed or modified by sync (verified by comparing source-agent deployments in state before/after)."
context:
  - "internal/deploy/deploy.go"
  - "internal/deploy/sync.go"
  - "internal/agent/agent.go"
  - "internal/agent/registry.go"
  - "internal/state/state.go"
  - "internal/state/store.go"
  - "cmd/app.go"
  - "cmd/sync.go"
  - "cmd/deploy.go"
  - "cmd/root.go"
  - "internal/tui/deploy.go"
  - "internal/tui/main_menu.go"
  - "internal/tui/services.go"
  - "internal/tui/screens.go"
  - "tests/integration/deploy_test.go"
---

## Sync deployed assets across agents

### Objective

Let a user replicate deployments from one coding agent to one or more other
agents in a single operation, without re-running `nd deploy` per agent.
Concretely: read every `state.Deployment` whose `Agent` field equals the
source agent, and for each target agent re-deploy the same asset (same
`SourcePath`, `Scope`, `Origin`, `Strategy`) to that target — skipping asset
types the target does not support and assets already present on the target,
and prompting before overwriting a differing existing deployment. Implements
GitHub issue #80 ("Sync deployed assets and configs across all agents":
one-to-many and many-to-many source/target selection, prevent conflicts and
data loss).

This builds on the multi-agent deploy plumbing delivered by dependency
`wui1vo` (`tasks/cli/wui1vo-expand-single-agent-assumptions.md`) and its
dependency `ba5xah` (`tasks/cli/ba5xah-select-deploy-agents.md`). In
particular `wui1vo` adds a per-agent engine factory
`App.DeployEngineFor(*agent.Agent)` and threads an explicit target agent
through deploy. **This task must use that per-agent engine path; do not
deploy via the single cached `App.DeployEngine()`** (it is bound to
`ActiveAgent()` only — `cmd/app.go:94-114`).

#### Naming conflict — read before starting

`nd sync` **already exists** and is a leaf command meaning "repair broken
symlinks / pull a git source" (`cmd/sync.go:11-124`, registered at
`cmd/root.go:84` via `newSyncCmd(app)`; declared `Args: cobra.NoArgs`). It is
unrelated to cross-agent sync. The original task text said "add `nd sync
agents`", but a Cobra command with `cobra.NoArgs` cannot also host
subcommands. Resolve this by **converting `sync` into a parent command**:
keep the current repair/pull behaviour as the default `RunE` on the parent
(so `nd sync` with no args is byte-for-byte unchanged) and `AddCommand` a new
`sync agents` subcommand. Do not change the existing repair/pull semantics or
flags (`--source`, `--dry-run` inherited from the persistent flag).

#### `deploy.SyncPlan` does NOT fit — define a new type

`deploy.SyncPlan` / `SyncAction` (`internal/deploy/sync.go:5-17`) model
symlink-repair actions (`Repairs`, `Removes`, `Healthy` over
`state.HealthCheck`). They are the wrong shape for cross-agent sync. Define a
**new** type for this feature (suggested: `AgentSyncPlan` in a new file
`internal/deploy/agent_sync.go`, package `deploy`) with per-target entries
classifying each source deployment as: `Create` (will deploy),
`SkipUnsupported` (target agent's `SupportsType` is false),
`SkipAlreadyDeployed` (a target deployment already exists with the same
`SourcePath`), or `Conflict` (a target deployment exists with a *different*
`SourcePath`). Do not extend or repurpose `SyncPlan`.

### Tasks

- [ ] **Plan type.** Add `internal/deploy/agent_sync.go` (package `deploy`)
  with `type AgentSyncPlan` holding, per target agent name, a slice of
  entries `{Deployment state.Deployment; Action <enum>}` where the action is
  one of create / skip-unsupported / skip-already-deployed / conflict.
  Add a pure function
  `BuildAgentSyncPlan(src []state.Deployment, targets []*agent.Agent, existing []state.Deployment) AgentSyncPlan`
  that: (a) for each target, iterates `src`; (b) skips an entry when
  `!target.SupportsType(d.AssetType)` (record `SkipUnsupported`) — uses
  `(*agent.Agent).SupportsType`, `internal/agent/agent.go:25-33`; (c) looks
  up an `existing` deployment for that target via the same identity match
  `removeOne` uses (`SourceID` + `AssetType` + `AssetName` + `Scope`, and
  `ProjectPath` when `Scope == nd.ScopeProject`) plus
  `existing.Agent == target.Name` (apply the empty-Agent→"claude-code"
  normalisation from `internal/deploy/deploy.go:594-604`); if found and
  `SourcePath` equal → `SkipAlreadyDeployed`, if found and `SourcePath`
  differs → `Conflict`, else → `Create`. No I/O in this function (unit
  testable).
- [ ] **Query source deployments.** In the new `sync agents` command, load
  state once via `app.StateStore().Load()`
  (`internal/state/store.go:31`, returns `(*DeploymentState, []string,
  error)`) and filter `st.Deployments` to those whose normalised `Agent`
  equals the `--from` agent name (empty `Agent` ⇒ `"claude-code"`, per
  `internal/deploy/deploy.go:594-604`). Pass that slice plus the full
  `st.Deployments` (as `existing`) into `BuildAgentSyncPlan`.
- [ ] **CLI: make `sync` a parent + add `sync agents`.** In `cmd/sync.go`
  (`newSyncCmd`, currently `cmd/sync.go:11-124`): keep the existing repair/
  pull logic as the parent's `RunE` and its `Args: cobra.NoArgs`-equivalent
  default behaviour intact; then `cmd.AddCommand(newSyncAgentsCmd(app))`.
  Add `newSyncAgentsCmd(app *App) *cobra.Command` (new function, may live in
  a new file `cmd/sync_agents.go`) with:
  - `Use: "agents"`, a `Short`, and an `Example` block (mirror the style at
    `cmd/sync.go:14-28`).
  - Flags: `--from <agent>` (required, string),
    `--to <agent>[,<agent>]` (required, `StringSliceVar`),
    `--dry-run` is already a persistent/global flag exposed as `app.DryRun`
    (see how `cmd/sync.go:42` / `cmd/deploy.go` read `app.DryRun`); add a
    local `--yes`/`-y` bool to bypass the conflict/overwrite confirmation in
    CLI mode (CLI has no interactive prompt — without `--yes`, conflicts must
    be reported and the run must exit non-zero rather than overwrite).
  - Validate `--from` and every `--to` name with
    `reg.Get(name)` then `reg.Detect()` and require `ag.Detected` (same
    checks `ActiveAgent` performs at `cmd/app.go:75-91`); on an unknown or
    undetected name return `withExitCode(nd.ExitInvalidUsage, err)` (pattern:
    `cmd/deploy.go:108`, `internal/nd/exit_codes.go:8` defines
    `ExitInvalidUsage = 3`). Reject `--from` appearing in `--to`.
  - Register shell completion for `--from`/`--to` returning known agent
    names (mirror `RegisterFlagCompletionFunc` usage in
    `cmd/sync.go:120-122`).
- [ ] **CLI: execute the plan.** For each target agent, build
  `deploy.DeployRequest` items from the plan's `Create` entries — set
  `Asset` from the source deployment (reconstruct via the asset layer or
  carry enough fields; mirror the request build at `cmd/deploy.go:165-174`),
  `Scope`/`ProjectRoot`/`Origin`/`Strategy` copied from the source
  `state.Deployment` (fields: `Scope`, `ProjectPath`, `Origin`, `Strategy` —
  `internal/state/state.go:20-32`). Get a per-target engine via
  `app.DeployEngineFor(ag)` (added by `wui1vo`) and run
  `engine.DeployBulk(reqs)` (`internal/deploy/deploy.go:464-507`). Merge
  per-target `BulkDeployResult`s and report per target (succeeded/failed/
  skipped with reason). Honour `app.JSON`/`app.Quiet` like `cmd/sync.go`.
- [ ] **CLI: conflicts.** For `Conflict` entries: without `--yes`, do **not**
  deploy them — print a clear message per asset
  (e.g. `conflict: skill/foo already deployed to copilot with different
  source; rerun with --yes to overwrite`) and exit
  `withExitCode(nd.ExitPartialFailure, ...)` if other work succeeded
  (pattern: `cmd/deploy.go:258`). With `--yes`, set `ForceReplace: true` on
  the request (the engine then removes the conflicting symlink/state and
  re-deploys — `internal/deploy/deploy.go:107,371-376,389-394`).
- [ ] **CLI: dry-run.** When `app.DryRun` is true, print the full
  `AgentSyncPlan` (every target: create / skip-unsupported / skip-already /
  conflict) and return **before** constructing any engine or calling
  `DeployBulk` — no symlink writes, no `state.Store.Save`. Mirror the
  dry-run early-return shape at `cmd/sync.go:66-83`.
- [ ] **TUI: menu entry.** In `internal/tui/main_menu.go` add a selectable
  option under the "── Manage ──" group (after the `huh.NewOption` list at
  `internal/tui/main_menu.go:43-48`), e.g.
  `huh.NewOption("Sync across agents", "syncagents")`, and a matching
  `case "syncagents":` in `handleSelection`
  (`internal/tui/main_menu.go:102-135`) that builds the new screen and emits
  `NavigateMsg{Screen: ...}` (pattern: the other cases at
  `internal/tui/main_menu.go:105-126`).
- [ ] **TUI: sync screen.** Add `internal/tui/sync_agents.go` implementing
  the `Screen` interface (`internal/tui/screens.go:6-10`:
  `tea.Model` + `Title() string` + `InputActive() bool`). Model a multi-step
  flow with a `step` enum exactly like `deployScreen`
  (`internal/tui/deploy.go:17-25`):
  1. pick source agent — `huh.NewSelect[string]()` over detected agents
     (`reg, _ := svc.AgentRegistry(); reg.Detect()`; iterate `reg.All()`
     keeping `a.Detected`, `internal/agent/agent.go:21`).
  2. pick target agents — `huh.NewMultiSelect[string]()` excluding the
     chosen source (mirror the multi-select + theme pattern used in the
     deploy flow, `huh.ThemeCatppuccin`, as in
     `internal/tui/deploy.go` form builders and
     `internal/tui/main_menu.go:31-57`).
  3. plan preview — render `BuildAgentSyncPlan(...)` grouped by target.
  4. running → result, reusing `deployBulkCmd`-style command dispatch (see
     `internal/tui/deploy.go` `startDeploy`/`deployBulkCmd`).
  Get per-target engines via `svc.DeployEngineFor(ag)` — `wui1vo` adds this
  to the `Services` interface (`internal/tui/services.go:13-45`) and the
  test mock; this task only consumes it. Emit `BackMsg{}` on abort/esc
  (pattern: `internal/tui/deploy.go:304,322,406,423,433`).
- [ ] **TUI: conflict resolution.** Reuse the deployScreen conflict pattern
  verbatim in shape: a `huh.NewConfirm()` step
  ("N asset(s) already deployed to <agent> with different source. Replace
  them?", Affirmative "Replace" / Negative "Cancel"), and on confirm rebuild
  the conflicting requests with `ForceReplace = true` and re-run
  `DeployBulk`. The reference implementation is
  `internal/tui/deploy.go:598-680` (`buildForceRequests`,
  `buildConflictForm`, `updateConflictConfirm`, `cancelConflictResolution`)
  — follow it; do not invent a different UX.
- [ ] **No source mutation.** Sync must only ever *add* deployments for
  target agents. Never call `Remove`/`RemoveBulk` on source-agent
  deployments and never include the source agent in the target list. Add a
  guard rejecting `--from` ∈ `--to`.
- [ ] **Unit tests — plan.** New `internal/deploy/agent_sync_test.go`:
  table-driven `BuildAgentSyncPlan` cases — unsupported-type filtering
  (copilot has `SupportedTypes = {skill, agent, context}`,
  `internal/agent/registry.go:53`; claude-code supports all,
  `internal/agent/registry.go:42` via `nd.DeployableAssetTypes()`);
  already-deployed skip (same SourcePath); conflict (different SourcePath);
  empty source list (no actions); one-to-many fan-out (two targets);
  empty-`Agent` normalised to `claude-code`.
- [ ] **TUI tests.** Add `internal/tui/sync_agents_test.go` using the
  existing mock-services + screen-construction helpers (see
  `internal/tui/deploy_test.go` for `newMockServices()` /
  `newTestDeployScreen` patterns and
  `internal/tui/testutil_test.go`): source picker shows only
  `Detected==true` agents and excludes nothing; target multi-select excludes
  the chosen source; plan preview renders create/skip/conflict groupings;
  conflict confirm flow re-runs with force.
- [ ] **Integration test.** Add to `tests/integration/` (new test, or extend
  `tests/integration/deploy_test.go`; use `setupIntegrationEnv` + `runND`,
  `tests/integration/deploy_test.go:9-37`): deploy `greeting` to a source
  agent, run `nd --config <p> sync agents --from <src> --to <tgt> --yes`,
  assert exit 0 and that the asset is reported synced; run with `--dry-run`
  and assert no state change; run with a bogus `--from`/`--to` and assert
  non-zero exit. (The sandbox may only detect `claude-code`; assert
  behaviour against whatever is detectable rather than requiring copilot.)

### Acceptance criteria

- `nd sync` with no subcommand keeps its current repair/pull behaviour
  unchanged (parent command default `RunE`); `nd sync agents` is a new
  subcommand.
- `nd sync agents --from claude-code --to copilot --yes` deploys every
  claude-code deployment whose type copilot supports; types copilot does not
  support are skipped with a clear per-asset message
  (e.g. `skip: hook/foo — not supported by copilot`).
- Assets already deployed to a target with the same `SourcePath` are skipped
  (no duplicate `state.Deployment`).
- A target deployment with the same identity but a different `SourcePath`
  is a conflict: CLI without `--yes` refuses to overwrite (clear message,
  non-zero exit if partial); CLI with `--yes` and the TUI confirm flow
  overwrite via `ForceReplace`.
- `--dry-run` prints the full plan and performs zero filesystem and state
  writes (state file byte-identical before/after).
- One-to-many works: `--to copilot,other` syncs to both in one run; results
  labelled per target agent.
- The TUI sync screen flows source → targets → plan preview → result, and
  is reachable from the main menu's "── Manage ──" group.
- No data loss: no source-agent deployment is removed or modified; `--from`
  may not appear in `--to`.
- `go build -o nd .`, `go test ./...`, `go test -race ./internal/tui/...`,
  `golangci-lint run`, and the integration sync tests pass.

### References

- Issue: https://GitHub.com/armstrongl/nd/issues/80 (title: "Sync deployed
  assets and configs across all agents")
- Dependency tasks (provide the multi-agent engine plumbing this builds on):
  `tasks/cli/wui1vo-expand-single-agent-assumptions.md` (adds
  `App.DeployEngineFor`, threads target agent through deploy/state),
  `tasks/cli/ba5xah-select-deploy-agents.md` (multi-agent picker + config)
- Existing single-purpose `sync` command (must become a parent):
  `cmd/sync.go:11-124`, registered `cmd/root.go:84`
- Wrong-shape plan type — do NOT reuse: `internal/deploy/sync.go:5-17`
  (`SyncPlan`/`SyncAction`)
- Deploy engine + request/result types:
  `internal/deploy/deploy.go:30-62` (one engine ↔ one agent),
  `:100-108` (`DeployRequest`, incl. `ForceReplace`),
  `:140-144` (`BulkDeployResult`), `:464-507` (`DeployBulk`),
  `:371-394` (force-replace path), `:583-620` (`removeOne` identity match +
  empty-`Agent`→`claude-code` normalisation, `:594-604`)
- Agent model: `internal/agent/agent.go:11-33`
  (`SupportedTypes`, `SupportsType`, `Detected`),
  `internal/agent/registry.go:35-58` (known agents + their
  `SupportedTypes`), `:101-106` (`All`), `:117-155` (`Detect`),
  `:176-183` (`Get`)
- State: `internal/state/state.go:13-41`
  (`DeploymentState`, `Deployment` fields incl. `Agent`),
  `internal/state/store.go:31` (`Store.Load`)
- App service wiring: `cmd/app.go:58-114` (`AgentRegistry`, `ActiveAgent`,
  `DeployEngine`) — note `DeployEngineFor` is added by `wui1vo`
- CLI patterns to mirror: `cmd/deploy.go:108,165-174,198,258`
  (usage error, request build, `DeployBulk`, partial-failure exit),
  `internal/nd/exit_codes.go:8` (`ExitInvalidUsage`)
- TUI patterns to mirror: `internal/tui/main_menu.go:31-135`
  (menu options + `handleSelection`),
  `internal/tui/screens.go:6-18` (`Screen` interface, `NavigateMsg`,
  `BackMsg`), `internal/tui/deploy.go:17-25` (step enum),
  `:598-680` (conflict-resolution reference: `buildForceRequests`,
  `buildConflictForm`, `updateConflictConfirm`,
  `cancelConflictResolution`), `internal/tui/services.go:13-45`
  (`Services` interface)
- Test patterns: `internal/tui/deploy_test.go`,
  `internal/tui/testutil_test.go`,
  `tests/integration/deploy_test.go:9-37`
