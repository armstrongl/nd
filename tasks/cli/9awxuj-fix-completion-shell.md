---
title: "Fix nd completion shell detection"
id: "9awxuj"
status: pending
priority: medium
type: bug
tags: ["cli", "completions"]
created_at: "2026-04-20"
---

## Fix nd completion shell detection

### Objective

Running `nd completion` with no subcommand should auto-detect the user's shell from `$SHELL` and generate the appropriate completion script, removing the need to manually specify `bash`, `zsh`, or `fish`.

### Steps to reproduce

1. Run `nd completion` (no subcommand)
2. Observe that it prints usage help instead of generating completions for the current shell

### Expected behavior

`nd completion` reads `$SHELL`, extracts the shell name (e.g. `/bin/zsh` -> `zsh`), and generates the matching completion script to stdout. If `$SHELL` is unset or unrecognized, it returns a clear error message listing the supported shells.

### Actual behavior

`nd completion` requires an explicit `bash`, `zsh`, or `fish` subcommand. There is no auto-detection from `$SHELL`.

### Tasks

- [ ] Add a `RunE` handler to the parent `completion` command in `cmd/completion.go` that reads `os.Getenv("SHELL")` and delegates to the appropriate subcommand generator
- [ ] Use `filepath.Base()` on `$SHELL` to extract the shell name and match against `bash`, `zsh`, `fish`
- [ ] Return a descriptive error when `$SHELL` is empty or not one of the supported shells
- [ ] Support the `--install` and `--install-dir` flags on the auto-detected path (forward them to the resolved subcommand)
- [ ] Add unit tests covering: zsh detection, bash detection, fish detection, empty `$SHELL`, unrecognized shell value
- [ ] Update `cmd/completion_test.go` with table-driven tests that set `$SHELL` via `t.Setenv`

### Acceptance criteria

- `SHELL=/bin/zsh nd completion` produces the same output as `nd completion zsh`
- `SHELL=/usr/local/bin/fish nd completion --install` installs fish completions
- `nd completion` with `$SHELL` unset returns exit code 1 and a helpful error
- `nd completion` with `$SHELL=/bin/csh` returns an error listing supported shells
- All existing `nd completion bash|zsh|fish` subcommands continue to work unchanged
- Tests pass: `go test ./cmd/... -run TestCompletion`

### References

- GitHub issue: https://GitHub.com/armstrongl/nd/issues/62
- Source: `cmd/completion.go`
