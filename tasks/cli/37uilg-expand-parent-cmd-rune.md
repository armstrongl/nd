---
title: "Add useful default RunE to parent commands"
id: "37uilg"
status: pending
priority: medium
type: bug
tags: ["cli"]
created_at: "2026-05-17"
dependencies: []
---

## Add useful default RunE to parent commands

### Objective

Pattern expansion of seed 9awxuj (`nd completion` has no parent `RunE`, so bare invocation prints generic usage instead of auto-detecting `$SHELL`). Several other parent/group commands share this: bare invocation prints Cobra usage where a sensible default or a clear "pick one of" message would help. `sync` and `export` are leaf commands with their own `RunE` and are excluded.

### Tasks

- [ ] `cmd/completion.go:12` -- add `RunE` that auto-detects `$SHELL` (`filepath.Base`), dispatches to bash/zsh/fish, forwards `--install`/`--install-dir`; error listing supported shells if unset/unknown (the seed)
- [ ] `cmd/source.go:15` -- add `RunE` defaulting to `source list` output
- [ ] `cmd/profile.go:14` -- add `RunE` defaulting to `profile list` output
- [ ] `cmd/snapshot.go:14` -- add `RunE` defaulting to `snapshot list` output
- [ ] `cmd/settings.go:11` -- add `RunE` defaulting to `settings edit` (single subcommand) or printing the resolved config path
- [ ] Ensure each new `RunE` respects `--json`/`--quiet` consistently with the delegated subcommand
- [ ] Tests: bare-invocation tests for each command; completion shell-detection table tests (zsh/bash/fish/unset/unknown) in `cmd/completion_test.go`

### Acceptance criteria

- `SHELL=/bin/zsh nd completion` == `nd completion zsh`; unset/unknown `$SHELL` -> exit 1 with supported-shell list
- `nd source`, `nd profile`, `nd snapshot` print their list output instead of usage
- `nd settings` opens the editor (or prints config path) instead of usage
- Existing subcommands unchanged; all `cmd/` tests pass

### References

- Seed task: 9awxuj -- `tasks/cli/9awxuj-fix-completion-shell.md`
- `cmd/completion.go`, `cmd/source.go`, `cmd/profile.go`, `cmd/snapshot.go`, `cmd/settings.go`

### Merged scope (supersedes 9awxuj)

This task supersedes cancelled seed `9awxuj`, whose entire deliverable is the `cmd/completion.go:12` item above. Reproduction (from 9awxuj): run `nd completion` with no subcommand; it prints usage instead of generating completions for `$SHELL`. Added detail/acceptance from 9awxuj: forward `--install`/`--install-dir` on the auto-detected path; `SHELL=/usr/local/bin/fish nd completion --install` installs fish completions; `SHELL=/bin/csh` returns an error listing supported shells; table-driven tests set `$SHELL` via `t.Setenv` in `cmd/completion_test.go`.
