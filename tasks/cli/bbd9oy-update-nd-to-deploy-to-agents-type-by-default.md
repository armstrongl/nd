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
