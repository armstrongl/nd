---
title: "Update nd to deploy to .agents/<type> by default"
id: "bbd9oy"
status: pending
priority: high
type: feature
tags: ["deploy", "breaking-change"]
created_at: "2026-04-20"
---

## Update nd to deploy to .agents/<type> by default

### Objective

Change nd's default project-scope deploy target from `.claude/<type>/` to `.agents/<type>/` (e.g., `.agents/skills/`, `.agents/agents/`, `.agents/commands/`). This aligns with the emerging `.agents/` convention for agent configuration, keeping `.claude/` reserved for Claude Code's own state. Global-scope paths (`~/.claude/`) remain unchanged for now.

### Tasks

- [ ] Update `Agent.ProjectDir` default for claude-code from `".claude"` to `".agents"` in the registry
- [ ] Update `DeployPath` and `configDir` to use the new project dir
- [ ] Handle context files: determine whether context assets also move under `.agents/` or remain at project root (CLAUDE.md stays at root)
- [ ] Update all test expectations in `agent_test.go` (e.g., `.claude/skills/review` → `.agents/skills/review`)
- [ ] Update integration test helpers (`deploy_test.go`, `helpers_test.go`, `status_sync_test.go`, `list_test.go`) that hardcode `.claude` paths
- [ ] Update CLI completion text in `root.go` (`"Deploy to .claude/ in project"` → `"Deploy to .agents/ in project"`)
- [ ] Update registry tests that assert `ProjectDir` defaults and overrides
- [ ] Add compat symlinks: after deploying to `.agents/<type>/`, create `.claude/<type>` → `.agents/<type>` directory symlinks so tools expecting `.claude/` still resolve
- [ ] Add migration logic: detect existing `.claude/<type>/` asset symlinks, move them to `.agents/<type>/`, then create compat symlinks in `.claude/<type>/`
- [ ] Ensure `nd remove` cleans up both the `.agents/` asset and the `.claude/` compat symlink when the last asset in a type dir is removed
- [ ] Update documentation (`docs/`, `README.md`) to reference `.agents/` paths
- [ ] Run `scripts/lint-docs.sh` and `golangci-lint run` before pushing

### Acceptance criteria

- `nd deploy --scope project` creates symlinks under `.agents/<type>/` by default
- Global deploy (`--scope global`) still targets `~/.claude/`
- Context files (CLAUDE.md, .local.md) deploy to their existing locations (project root or global dir)
- Copilot agent paths (`.github/`) are unaffected
- All existing tests pass with updated expectations
- `nd status` and `nd list` correctly show `.agents/`-based deployments
- `nd remove` can clean up `.agents/`-based symlinks
- CLI `--scope project` completion shows `.agents/` description

### Merged scope (from bagcdr)

Absorbed from cancelled task `bagcdr` so the migration lands atomically. The seed list above misses several production and test sites; most critically `internal/nd/project.go` uses `.claude/` as a project-root marker, so migrating deploy paths without this is a behavioral regression (a project with only `.agents/` and no `.git` fails project-root detection).

- [ ] `internal/nd/project.go:19` -- add `.agents` to the project-root marker list (decide: replace `.claude` or keep both)
- [ ] `internal/nd/project.go:28` -- update the "looked for .git/ or .claude/" error message accordingly
- [ ] `internal/nd/project_test.go:30` -- update the fixture to match the new marker behavior
- [ ] `internal/export/plugin.go:318` -- update generated `.claude/rules/` install-doc text
- [ ] `internal/export/plugin.go:326` -- update generated `~/.claude/` install-doc text (decide project-scope half)
- [ ] `cmd/init_cmd_test.go:21,26,65,268,307,502` -- update `.claude` path fixtures + `ProjectDir` override
- [ ] `cmd/deploy_test.go:30,31,34,204,205,234,235` -- update agentDir/linkPath/config-string fixtures
- [ ] `cmd/list_test.go:175,213` -- update config-string `global_dir` references (`:172,:210` already above)
- [ ] `internal/deploy/result_test.go:17` -- update `LinkPath` literal
- [ ] `internal/deploy/health_test.go` -- decide scope (global-scope `~/.claude`; update only if global path also changes, likely NO)

Added acceptance: `FindProjectRoot` detects a project containing only `.agents/` (no `.git`); the not-found error lists the correct marker(s); `nd export` generated install docs reference the correct project-scope dir.
