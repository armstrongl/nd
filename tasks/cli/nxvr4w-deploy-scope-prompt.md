---
title: "Offer global or project scope on deploy"
id: "nxvr4w"
status: pending
priority: medium
type: feature
tags: ["deploy", "ux"]
created_at: "2026-04-20"
---

## Offer global or project scope on deploy

### Objective

Prompt users to choose between global and project scope every time they perform a deploy action. Present a clear menu distinguishing global (system-wide, deploys to `~/.claude/`) from project (local, deploys to `.agents/` in the current project). Remember the user's preference for the current session to avoid repeated prompting. Support both single-asset and batch deployments.

### Tasks

- [ ] **Design scope selection UX**
  - Define a scope picker menu with two options: "Global (system-wide, ~/.claude/)" and "Project (local, .agents/)"
  - Include a brief explanation of each scope's effect on first display
  - Add a "remember for this session" toggle or auto-remember after first selection

- [ ] **Implement scope prompt in CLI deploy flow**
  - Add scope prompt to `cmd/deploy.go` when `--scope` flag is not provided
  - Skip the prompt when `--scope global` or `--scope project` is explicitly passed
  - Store session preference in a package-level variable (reset on process exit)
  - Apply selected scope to the `deploy.Engine` call

- [ ] **Implement scope prompt in TUI deploy flow**
  - Add scope selection step to the TUI deploy screen (`internal/tui/deploy.go`)
  - Show scope picker before confirming the deploy action
  - Remember selection for the TUI session (persist in the App model state)
  - Apply selected scope to the deploy message/command

- [ ] **Support batch deployments**
  - Apply the session scope preference to `nd deploy --all` and profile deploy operations
  - Allow overriding per-batch with an explicit `--scope` flag
  - Show the active scope in batch deploy confirmation output

- [ ] **Handle edge cases**
  - When not inside a git repo or project directory, default to global and skip the prompt
  - When `--yes` is passed, require `--scope` or default to global with a warning
  - When in JSON output mode, require `--scope` flag (no interactive prompt)

- [ ] **Write tests**
  - Test CLI prompt appears when `--scope` is omitted
  - Test CLI prompt is skipped when `--scope` is explicit
  - Test session preference is remembered across multiple deploys in the same process
  - Test TUI scope picker renders correctly and propagates selection
  - Test batch deploy uses session scope
  - Test non-project directory defaults to global
  - Test `--yes` without `--scope` defaults to global

### Acceptance criteria

- Running `nd deploy <asset>` without `--scope` prompts the user to choose global or project
- Passing `--scope global` or `--scope project` skips the prompt entirely
- After choosing a scope, subsequent deploys in the same session use that scope without re-prompting
- Batch operations (`--all`, profile deploy) respect the session scope preference
- The TUI deploy screen includes a scope selection step before confirming
- Outside a project directory, global is used automatically without prompting
- `--yes` mode without `--scope` defaults to global scope
- All existing deploy tests continue to pass

### References

- GitHub issue: https://GitHub.com/armstrongl/nd/issues/82 (closed; originally ISSUE-018)
- Related: TUI Phase 7 scope switching feature (`ScopeSwitchedMsg`)
