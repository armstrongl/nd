---
title: "Update nd to deploy to .agents/<type> by default"
id: "bbd9oy"
status: pending
priority: high
type: feature
tags: ["deploy", "breaking-change"]
created_at: "2026-04-20"
context:
  - internal/agent/registry.go
  - internal/agent/agent.go
  - internal/agent/agent_test.go
  - internal/agent/registry_test.go
  - internal/nd/project.go
  - internal/nd/project_test.go
  - internal/export/plugin.go
  - cmd/root.go
  - cmd/init_cmd_test.go
  - internal/deploy/deploy.go
  - internal/deploy/uninstall.go
  - cmd/remove.go
  - docs/guide/how-nd-works.md
  - docs/guide/getting-started.md
verify:
  - type: bash
    run: "go build -o nd ."
  - type: bash
    run: "go test ./..."
  - type: bash
    run: "go test -race ./..."
  - type: bash
    run: "go test ./tests/integration/ -v"
  - type: bash
    run: "scripts/lint-docs.sh"
  - type: assert
    check: "claude-code Agent.ProjectDir resolves to \".agents\" by default (registry.go:39), so DeployPath at project scope produces <project>/.agents/<type>/<name>"
  - type: assert
    check: "FindProjectRoot detects a directory containing only .agents/ (no .git, no .claude); the not-found error message lists the correct marker(s)"
  - type: assert
    check: "Global scope (~/.claude/...) deploy paths are byte-for-byte unchanged; no global-scope test fixture was modified"
---

## Update nd to deploy to .agents/<type> by default

### Objective

Change nd's default **project-scope** deploy target for the `claude-code`
agent from `<project>/.claude/<type>/` to `<project>/.agents/<type>/` (e.g.
`.agents/skills/`, `.agents/agents/`, `.agents/commands/`). This aligns with
the emerging `.agents/` convention for portable agent configuration, keeping
`.claude/` reserved for Claude Code's own runtime state.

**Global-scope paths (`~/.claude/...`) MUST remain unchanged.** This is the
single most important invariant in this task: most `.claude` strings in the
codebase and tests are global-scope and must not be touched.

The only field that drives project-scope deployment is
`Agent.ProjectDir`. Today it is hard-coded to `".claude"` for the
`claude-code` agent. Changing that one default cascades through
`Agent.DeployPath` / `Agent.configDir` automatically — there is no separate
"DeployPath" or "configDir" constant to edit despite what an older draft of
this task implied.

This task also absorbs the cancelled task `bagcdr`
(`tasks/cli/bagcdr-expand-agents-migration-missed.md`, status `cancelled`),
which exists to ensure the highest-risk missed file (`internal/nd/project.go`,
which uses `.claude/` as a project-root marker) lands atomically with the
deploy change. Do not action `bagcdr` separately.

### Bug / regression this prevents

`internal/nd/project.go` `FindProjectRoot` (lines 18-31) walks up looking for
a directory containing `.git/` **or** `.claude/`:

```go
for _, marker := range []string{".git", ".claude"}{ ... }
```

If we migrate deploy to `.agents/` without updating this marker list, a
project that has only `.agents/` (no `.git`, no `.claude/`) will fail
project-root detection — a behavioral regression, not just a cosmetic
rename. The error message at `internal/nd/project.go:28` also hard-codes
`looked for .git/ or .claude/` and would then be wrong.

### Key code map (verified line numbers)

- `internal/agent/registry.go:39` — `ProjectDir: ".claude"` for the
  `claude-code` agent. **Single source of the default.**
- `internal/agent/registry.go:60-74` — config-override loop; an explicit
  `agents[].project_dir` in config still overrides the new default
  (preserve this).
- `internal/agent/agent.go:84-89` — `configDir(scope, projectRoot)` returns
  `filepath.Join(projectRoot, a.ProjectDir)` for project scope. No change
  needed; it reads `ProjectDir`.
- `internal/agent/agent.go:46-60` — `DeployPath`; non-context assets go to
  `<configDir>/<assetType.DeploySubdir()>/<assetName>`. No change needed.
- `internal/agent/agent.go:63-81` — `contextDeployPath`. For `claude-code`
  `ContextInProjectDir` is `false`, so project-scope context (CLAUDE.md)
  deploys to the **project root**, NOT inside `.agents/`. Confirm this stays
  true (do not move CLAUDE.md under `.agents/`).
- `internal/nd/project.go:19` — marker slice `[]string{".git", ".claude"}`.
- `internal/nd/project.go:28` — not-found error string
  `looked for .git/ or .claude/ from %s`.
- `cmd/root.go:68` — scope flag completion:
  `"project\tDeploy to .claude/ in project"` (note: same line also contains
  the global completion `"global\tDeploy to ~/.claude/"` which must stay).
- `internal/export/plugin.go:318` — generated install-doc text
  `Copy to your`.claude/rules/`directory:`.
- `internal/export/plugin.go:326` — generated install-doc text
  `Copy to your project root or`~/.claude/`:` (the `~/.claude/` half is
  global; only adjust project-scope wording if you change it at all).
- Symlink creation: `internal/deploy/deploy.go:222` computes `linkPath` via
  `e.agent.DeployPath(...)`; `:248-263` `mkdirAll(parentDir)` then
  `e.symlink(target, linkPath)`. This already works for any `ProjectDir`.

### Decisions to make explicitly (document the choice in the worklog)

1. **Marker behavior** (`project.go:19`): replace `.claude` with `.agents`,
   or keep BOTH (`{".git", ".agents", ".claude"}`). Recommended: keep both
   so existing `.claude/`-only projects keep working during transition.
   Whatever you choose, the error message at `:28` must match.
2. **Compat symlinks / migration**: the original seed proposed creating
   `.claude/<type> -> .agents/<type>` compat symlinks and auto-migrating
   existing `.claude/<type>/` asset symlinks. This is large and optional.
   If you implement it, do it in `internal/deploy/deploy.go` (after the
   `e.symlink` call at `:261-263`) and clean it up in the uninstall/remove
   path (`internal/deploy/uninstall.go`, `cmd/remove.go`). If you defer it,
   say so in the worklog and update the acceptance criteria below to drop
   the compat-symlink lines. The marker/registry change is the
   non-negotiable core; compat symlinks are nice-to-have.

### Tasks

Core (non-negotiable):

- [ ] `internal/agent/registry.go:39` — change `ProjectDir: ".claude"` to
  `ProjectDir: ".agents"` for the `claude-code` agent only. Leave `copilot`
  (`.github`) and all `GlobalDir` values untouched.
- [ ] `internal/nd/project.go:19` — add `.agents` to the marker slice (per
  decision #1 above).
- [ ] `internal/nd/project.go:28` — update the not-found error string to
  list the chosen marker(s) accurately.
- [ ] `cmd/root.go:68` — change the project half of the completion to
  `"project\tDeploy to .agents/ in project"`; keep the global half
  (`"global\tDeploy to ~/.claude/"`) unchanged.
- [ ] `internal/export/plugin.go:318` — update the generated install-doc
  text for project-scope rules to `.agents/rules/`.
- [ ] `internal/export/plugin.go:326` — review the
  `Copy to your project root or`~/.claude/`:` string; the `~/.claude/`
  part is global and stays. Only reword if it implies a project-scope
  `.claude/` path.

Tests (verified — only project-scope assertions change):

- [ ] `internal/agent/agent_test.go:14` — `claudeCode()` helper sets
  `ProjectDir: ".claude"`; change to `".agents"`.
- [ ] `internal/agent/agent_test.go:60` — assertion
  `want := "/Users/dev/myapp/.claude/skills/review"` (project scope, set up
  at `:56` with `nd.ScopeProject`); change to
  `"/Users/dev/myapp/.agents/skills/review"`. Leave the global-scope
  assertions (`:48` `/Users/dev/.claude/skills/review`, `:105`
  `/Users/dev/.claude/agents/go-specialist.md`, `:72`
  `/Users/dev/.claude/CLAUDE.md`) UNCHANGED — they are `nd.ScopeGlobal`.
  Leave `:85` `/Users/dev/myapp/CLAUDE.md` UNCHANGED (project context still
  goes to project root).
- [ ] `internal/agent/registry_test.go` — there is currently NO test
  asserting the default `claude-code` `ProjectDir`. ADD one (mirror
  `TestNewRegistryAppliesProjectDirOverride` at `:69-79` and
  `TestNewRegistryHasBothAgents` at `:40-53`) asserting
  `agents[0].ProjectDir == ".agents"`. The existing override test at
  `:69-79` (uses `.custom-claude`) needs no change.
- [ ] `internal/nd/project_test.go:27-40` — `TestFindProjectRoot_ClaudeMarker`
  creates a `.claude/` dir and expects detection. Per decision #1: if you
  keep `.claude` as a marker, this test still passes as-is; ADD a sibling
  `TestFindProjectRoot_AgentsMarker` creating only `.agents/` and asserting
  detection. If you replace `.claude` with `.agents`, rename/rewrite this
  test to use `.agents/`.
- [ ] `cmd/init_cmd_test.go:26` — `testInitAgent` helper sets
  `ProjectDir: ".claude"`; change to `".agents"`. **Do NOT change** the
  `agentDir := filepath.Join(tmp, ".claude")` lines at `:21, :65, :268,
  :307, :502` — those are `GlobalDir` redirects (global scope) and must
  stay. (An earlier draft incorrectly flagged all of these.)
- [ ] `internal/export/plugin_test.go` (if it exists) — search for
  `.claude/rules` assertions and update only project-scope ones to match
  the new `plugin.go` output.

Explicitly DO NOT touch (verified global-scope `.claude`, invariant says
global is unchanged) — listed so a future agent does not "fix" them:

- `cmd/deploy_test.go:30, :204, :234` (`global_dir` overrides via config)
- `cmd/list_test.go:172, :175, :210, :213` (`global_dir` overrides)
- `internal/deploy/result_test.go:17` (`LinkPath: "/Users/dev/.claude/..."`,
  a global path literal)
- `internal/deploy/health_test.go` (global `~/.claude` health checks)
- `tests/integration/deploy_test.go:83`,
  `tests/integration/helpers_test.go:79`,
  `tests/integration/status_sync_test.go:21` (all `global_dir` redirects)
- All `~/.claude/...` strings in `docs/` and `internal/export/plugin.go:326`

Optional (only if implementing compat symlinks per decision #2):

- [ ] Add compat symlinks: after `e.symlink(target, linkPath)` in
  `internal/deploy/deploy.go:261-263`, for project scope + claude-code,
  create `<project>/.claude/<type> -> <project>/.agents/<type>` directory
  symlink (idempotent; skip if it exists or is a real dir).
- [ ] Migration: detect existing `<project>/.claude/<type>/` asset symlinks,
  move them to `.agents/<type>/`, then create the compat symlink.
- [ ] `nd remove` (`cmd/remove.go` + `internal/deploy/uninstall.go`): when
  the last asset in a `.agents/<type>/` dir is removed, also remove the
  `.claude/<type>` compat symlink and the empty `.agents/<type>/` dir.

Docs:

- [ ] `docs/guide/how-nd-works.md:82` — table row
  `| Claude Code (default) |`~/.claude/` | `<project>/.claude/`|`: change
  only the project column to `<project>/.agents/`.
- [ ] `docs/guide/how-nd-works.md:99` — example
  `~/myproject/.claude/skills/greeting -> ...`: change to
  `~/myproject/.agents/skills/greeting -> ...`.
- [ ] `docs/guide/how-nd-works.md:134,140` — context-file section says
  project context deploys to project root NOT inside `.claude/`. Reword
  the "NOT inside `.claude/`" phrasing to "NOT inside `.agents/`" so it
  stays accurate; do NOT move CLAUDE.md.
- [ ] `docs/guide/getting-started.md:136` — bullet
  `**Project** (...): ... (e.g.,`.claude/`for Claude Code, ...)`: change
  the Claude Code project example to `.agents/`. Leave the global bullet at
  `:135` and the `~/.claude/` example at `:125` unchanged.
- [ ] Sweep remaining docs for project-scope `.claude/<type>` examples:
  `grep -rn '\.claude/skills\|\.claude/agents\|\.claude/commands\|\.claude/rules' docs/ README.md`
  and update only project-scope occurrences (leave `~/.claude/...`).

Final checks:

- [ ] `go build -o nd .` succeeds.
- [ ] `go test ./...` and `go test -race ./...` pass.
- [ ] `go test ./tests/integration/ -v` passes.
- [ ] `scripts/lint-docs.sh` passes (it lints `docs/`; run from repo root).
- [ ] `golangci-lint run` if available (binary may not be installed in this
  env; `.golangci.yml` exists at repo root — run if you have the tool,
  otherwise note it was skipped).

### Acceptance criteria

- `nd deploy --scope project` for `claude-code` creates symlinks under
  `<project>/.agents/<type>/` by default (verifiable: deploy a skill at
  project scope and `ls -la <project>/.agents/skills/`).
- Global deploy (`--scope global`) still targets `~/.claude/<type>/` —
  unchanged, and no global-scope test fixture was modified.
- Project-scope context files (CLAUDE.md) still deploy to the **project
  root**, not inside `.agents/` (`agent_test.go:85` still expects
  `/Users/dev/myapp/CLAUDE.md` and passes).
- `.local.md` context still project-only; global scope still rejected.
- Copilot agent paths (`.github/`) are unaffected (`registry_test.go`
  copilot assertions still pass).
- `FindProjectRoot` detects a project containing only `.agents/` with no
  `.git` and no `.claude/`; the not-found error message lists the correct
  marker(s).
- `nd export` generated install docs reference `.agents/rules/` for
  project scope; the `~/.claude/` global wording is unchanged.
- CLI `--scope` completion shows `Deploy to .agents/ in project` for the
  `project` value and still `Deploy to ~/.claude/` for `global`.
- `go test ./...`, `go test -race ./...`, and
  `go test ./tests/integration/ -v` all pass.
- Docs no longer show a project-scope `.claude/<type>` example for Claude
  Code; `scripts/lint-docs.sh` passes.
- (If compat symlinks implemented) `.claude/<type>` resolves to
  `.agents/<type>`, and `nd remove` cleans up both when the last asset in a
  type dir is removed. (If deferred, this criterion is dropped — note the
  deferral in the worklog.)

### References

- GitHub issue: https://GitHub.com/armstrongl/nd/issues/127
- Close this issue when the task is completed.
- Cancelled, merged-in task: `bagcdr` —
  `tasks/cli/bagcdr-expand-agents-migration-missed.md` (status `cancelled`;
  do not action separately; its scope is folded here).
- Single source of the project default: `internal/agent/registry.go:39`.
- Path computation: `internal/agent/agent.go` `DeployPath` (`:46-60`) and
  `configDir` (`:84-89`).
- Project-root marker (regression risk): `internal/nd/project.go:18-31`.
- Symlink creation flow: `internal/deploy/deploy.go:222, :248-263`.
- Doc lint script: `scripts/lint-docs.sh`; Go lint config: `.golangci.yml`.
</content>
</invoke>
