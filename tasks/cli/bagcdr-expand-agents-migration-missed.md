---
title: "Cover missed files in .agents type migration"
id: "bagcdr"
status: cancelled
priority: high
type: feature
tags: ["deploy", "breaking-change"]
created_at: "2026-05-17"
dependencies: []
cancelled_at: 2026-05-17
---

## Cover missed files in the .agents/<type> migration

> **⚠ Superseded by bbd9oy** — folded into `tasks/cli/bbd9oy-update-nd-to-deploy-to-agents-type-by-default.md` so the migration lands atomically. This task is cancelled; do not action it.

### Objective

Pattern expansion of seed bbd9oy. The seed enumerates the registry, `agent_test.go`, and integration helpers but misses several production and test sites -- most critically `internal/nd/project.go`, where `.claude/` is a project-root marker. Migrating deploy to `.agents/` without touching `project.go` means a project with only `.agents/` and no `.git` will fail project-root detection (a behavioral regression, not just a string rename). Scope is the NEW gap occurrences not already in bbd9oy.

### Tasks

- [ ] `internal/nd/project.go:19` -- add `.agents` to the project-root marker list (decide: replace `.claude` or keep both)
- [ ] `internal/nd/project.go:28` -- update the "looked for .git/ or .claude/" error message accordingly
- [ ] `internal/nd/project_test.go:30` -- update the fixture to match the new marker behavior
- [ ] `internal/export/plugin.go:318` -- update generated `.claude/rules/` install-doc text
- [ ] `internal/export/plugin.go:326` -- update generated `~/.claude/` install-doc text (decide project-scope half)
- [ ] `cmd/init_cmd_test.go:21,26,65,268,307,502` -- update `.claude` path fixtures + `ProjectDir` override
- [ ] `cmd/deploy_test.go:30,31,34,204,205,234,235` -- update agentDir/linkPath/config-string fixtures
- [ ] `cmd/list_test.go:175,213` -- update config-string `global_dir` references (`:172,:210` already in seed)
- [ ] `internal/deploy/result_test.go:17` -- update `LinkPath` literal
- [ ] `internal/deploy/health_test.go` -- decide scope (these are global-scope `~/.claude`; update only if global path also changes, likely NO per bbd9oy AC "global unchanged")

### Acceptance criteria

- `FindProjectRoot` detects a project containing only `.agents/` (no `.git`)
- The project-root not-found error lists the correct marker(s)
- `nd export` generated install docs reference the correct project-scope dir
- All listed test fixtures pass with updated expectations
- Global-scope (`~/.claude`) behavior and its tests are intentionally left unchanged per bbd9oy AC

### References

- GitHub issue: https://GitHub.com/armstrongl/nd/issues/124
- Close this issue when the task is completed.
- Seed task: bbd9oy -- `tasks/cli/bbd9oy-update-nd-to-deploy-to-agents-type-by-default.md`
- `internal/nd/project.go` (highest-risk missed file), `internal/export/plugin.go`, `internal/agent/registry.go` (already in seed)
