---
title: "Add useful default RunE to parent commands"
id: "37uilg"
status: pending
priority: medium
type: bug
tags: ["cli"]
created_at: "2026-05-17"
dependencies: []
context:
  - "cmd/completion.go"
  - "cmd/source.go"
  - "cmd/profile.go"
  - "cmd/snapshot.go"
  - "cmd/settings.go"
  - "cmd/root.go"
  - "cmd/app.go"
  - "cmd/completion_test.go"
  - "cmd/helpers.go"
  - "tasks/cli/9awxuj-fix-completion-shell.md"
verify:
  - type: bash
    run: "go build -o nd ."
  - type: bash
    run: "go test ./cmd/..."
  - type: bash
    run: "go test -race ./cmd/..."
  - type: bash
    run: "golangci-lint run ./cmd/..."
  - type: assert
    check: "SHELL=/bin/zsh nd completion produces byte-identical output to nd completion zsh; SHELL unset or SHELL=/bin/csh exits non-zero listing bash, zsh, fish"
  - type: assert
    check: "Bare nd source / nd profile / nd snapshot print their list output (not Cobra usage); nd settings opens $EDITOR on the config (or prints the resolved config path)"
  - type: assert
    check: "All nd completion bash|zsh|fish, nd source/profile/snapshot/settings subcommands behave unchanged"
---

## Add useful default RunE to parent commands

### Objective

Several parent/group commands have no `RunE`, so bare invocation falls through to
Cobra's generic usage text instead of doing something useful. This is the
generalization of cancelled seed `9awxuj` (`tasks/cli/9awxuj-fix-completion-shell.md`,
GitHub issue https://GitHub.com/armstrongl/nd/issues/62), whose sole deliverable
was the `nd completion` auto-detect case below.

Make bare invocation of each parent command do the obvious thing:

- `nd completion` -> auto-detect the shell from `$SHELL` and generate that script.
- `nd source` / `nd profile` / `nd snapshot` -> print the `list` output.
- `nd settings` -> open the editor on the config (its only subcommand is `edit`).

`sync` and `export` are leaf commands with their own `RunE` and are out of scope.
The only existing precedent for a parent `RunE` is the root command
(`cmd/root.go:44`), which falls back to `cmd.Help()`; there is no
parent-delegates-to-subcommand pattern in the repo yet, so pick one of the
delegation approaches described under Tasks and apply it consistently.

### Root cause

These parent commands are constructed with only `Use`/`Short`/`Long` and
`AddCommand(...)`; none sets `RunE`, so Cobra prints usage when invoked with no
subcommand:

- `cmd/completion.go:12` `newCompletionCmd` — command literal at `cmd/completion.go:13`; subcommands `newCompletionBashCmd` (`:36`), `newCompletionZshCmd` (`:77`), `newCompletionFishCmd` (`:123`); shared install helpers `defaultBashCompletionDir` (`:162`), `defaultZshCompletionDir` (`:174`), `defaultFishCompletionDir` (`:182`), `installCompletion` (`:190`).
- `cmd/source.go:15` `newSourceCmd` — literal at `:16`; list subcommand `newSourceListCmd` at `cmd/source.go:263`.
- `cmd/profile.go:14` `newProfileCmd` — literal at `:15`; list subcommand `newProfileListCmd` at `cmd/profile.go:213`.
- `cmd/snapshot.go:14` `newSnapshotCmd` — literal at `:15`; list subcommand `newSnapshotListCmd` at `cmd/snapshot.go:232`.
- `cmd/settings.go:11` `newSettingsCmd` — literal at `:12`; only subcommand `newSettingsEditCmd` at `cmd/settings.go:25`.

`App.JSON`, `App.Quiet`, `App.DryRun` are fields on `App` (`cmd/app.go:26-28`);
`settings edit` reads the config path from `app.ConfigPath` (`cmd/settings.go:37`).

### Repro

```
go build -o nd .
./nd completion        # prints usage listing bash/zsh/fish instead of detecting $SHELL
./nd source            # prints Cobra usage instead of the source list
./nd profile           # prints Cobra usage instead of the profile list
./nd snapshot          # prints Cobra usage instead of the snapshot list
./nd settings          # prints Cobra usage instead of opening the editor
```

### Delegation approach (choose one, apply uniformly)

The `RunE` of each list subcommand is a closure built in its `newXListCmd`
constructor (e.g. `cmd/source.go:276`). Two clean options:

1. Look up the subcommand on the parent and invoke it: in the parent `RunE`,
   find the child with `for _, c := range cmd.Commands() { if c.Name() == "list" { return c.RunE(c, nil) } }`.
   Note the child must run with the parent's IO streams — Cobra propagates
   `OutOrStdout`/`ErrOrStderr`/`InOrStdin` to children, so calling
   `c.RunE(c, nil)` reuses them correctly.
2. Extract each list subcommand's body into a package-level helper
   (e.g. `func runSourceList(app *App, cmd *cobra.Command) error`) and call it
   from both the subcommand `RunE` and the new parent `RunE`. Preferred if you
   want the parent path independently testable.

Either way, all `app.JSON` / `app.Quiet` / `app.DryRun` branching already lives
inside the list/edit subcommand bodies, so delegating reuses it unchanged — do
not duplicate that logic in the parent.

### Tasks

- [ ] `cmd/completion.go` — add `RunE` to `newCompletionCmd` (command literal at `:13`, after `Hidden: true` / before `cmd.AddCommand`). Read `os.Getenv("SHELL")`, take `filepath.Base` of it (`path/filepath` and `os` are already imported, `:6-7`), switch on `"bash"`/`"zsh"`/`"fish"` and delegate to the matching subcommand; forward `--install` and `--install-dir`. When `$SHELL` is empty or not one of the three, return `fmt.Errorf` listing the supported shells (`bash, zsh, fish`). To forward the install flags through option 1, define them on the parent too, or via option 2 thread them into the extracted helper. Keep the parent `Hidden: true`.
- [ ] `cmd/source.go` — add `RunE` to `newSourceCmd` (literal at `:16`) delegating to the `list` path (`newSourceListCmd` body, `cmd/source.go:276-322`).
- [ ] `cmd/profile.go` — add `RunE` to `newProfileCmd` (literal at `:15`) delegating to the `list` path (`newProfileListCmd` body, `cmd/profile.go:226-282`).
- [ ] `cmd/snapshot.go` — add `RunE` to `newSnapshotCmd` (literal at `:15`) delegating to the `list` path (`newSnapshotListCmd` body, `cmd/snapshot.go:245-277`).
- [ ] `cmd/settings.go` — add `RunE` to `newSettingsCmd` (literal at `:12`) delegating to the `edit` path (`newSettingsEditCmd` body, `cmd/settings.go:35-62`), so bare `nd settings` opens the editor. Acceptable fallback: print the resolved config path (`app.ConfigPath`) when not a TTY, mirroring the existing dry-run branch at `cmd/settings.go:44-47`.
- [ ] Do NOT add `Args` constraints to the parent commands; they currently have none, and an unrecognized subcommand must still surface Cobra's "unknown command" error rather than being swallowed by the new `RunE`. Verify `nd source bogus` still errors after the change.
- [ ] Update `cmd/completion_test.go` — `TestCompletionNoSubcommand` (`cmd/completion_test.go:81-96`) currently asserts bare `nd completion` prints usage listing bash/zsh/fish; this assertion breaks once the `RunE` is added. Rewrite it as table-driven shell-detection tests: cases for `SHELL=/bin/zsh`, `SHELL=/bin/bash`, `SHELL=/usr/local/bin/fish`, `SHELL` unset, `SHELL=/bin/csh`. Set the env with `t.Setenv("SHELL", …)`. Mirror the existing harness in this file: `app := &App{}; rootCmd := NewRootCmd(app); rootCmd.SetOut(&out); rootCmd.SetErr(&out); rootCmd.SetArgs([]string{"completion"}); rootCmd.Execute()`. Assert detected output equals `nd completion <shell>` output (e.g. zsh contains `#compdef`, bash contains `__nd`, fish contains `complete`); assert unset/csh return a non-nil error whose message contains `bash`, `zsh`, `fish`.
- [ ] Add bare-invocation tests for the other parents: extend `cmd/source_test.go`, `cmd/profile_test.go`, `cmd/snapshot_test.go`, `cmd/settings_test.go`. Mirror existing patterns there — e.g. `TestSourceList_Empty` (`cmd/source_test.go:186`) for an empty `nd source`, and for settings reuse `setupDeployEnv(t)` + `--dry-run` as in `TestSettingsEditCmd_DryRun` (`cmd/settings_test.go:31`). Assert bare-parent output matches the corresponding `list`/`edit` output and is NOT the Cobra usage text (does not contain `"Usage:"`/`"Available Commands:"`).

### Acceptance criteria

- `SHELL=/bin/zsh ./nd completion` produces byte-identical output to `./nd completion zsh` (same for bash/fish). `SHELL` unset or `SHELL=/bin/csh ./nd completion` exits non-zero with an error listing `bash, zsh, fish`. `SHELL=/usr/local/bin/fish ./nd completion --install` installs fish completions to the resolved dir.
- `./nd source`, `./nd profile`, `./nd snapshot` print their `list` output instead of Cobra usage; `--json` and `--quiet` behave exactly as the explicit `list` subcommand.
- `./nd settings` opens `$EDITOR`/`$VISUAL`/`vi` on `app.ConfigPath` (or prints the resolved config path when non-interactive) instead of Cobra usage.
- Unknown subcommands still error (`./nd source bogus` returns Cobra's unknown-command error).
- All existing explicit subcommands (`completion bash|zsh|fish`, `source|profile|snapshot list`, `settings edit`) behave unchanged.
- `go build -o nd .` succeeds; `go test ./cmd/...`, `go test -race ./cmd/...`, and `golangci-lint run ./cmd/...` pass.

### References

- Cancelled seed (folded in here): `tasks/cli/9awxuj-fix-completion-shell.md`; GitHub issue https://GitHub.com/armstrongl/nd/issues/62
- Parent commands: `cmd/completion.go:12`, `cmd/source.go:15`, `cmd/profile.go:14`, `cmd/snapshot.go:14`, `cmd/settings.go:11`
- List/edit subcommands to delegate to: `cmd/source.go:263`, `cmd/profile.go:213`, `cmd/snapshot.go:232`, `cmd/settings.go:25`; completion subcommands `cmd/completion.go:36/77/123`
- Only existing parent `RunE` precedent: `cmd/root.go:44-50` (`cmd.Help()` fallback)
- `App` flags: `cmd/app.go:26-28` (`Quiet`, `JSON`, `DryRun`); test harness helpers: `cmd/helpers_test.go`, `setupDeployEnv` used in `cmd/settings_test.go:32`
- Test files to update/extend: `cmd/completion_test.go:81`, `cmd/source_test.go`, `cmd/profile_test.go`, `cmd/snapshot_test.go`, `cmd/settings_test.go`

### Merged scope (supersedes 9awxuj)

This task supersedes cancelled seed `9awxuj`, whose entire deliverable is the
`cmd/completion.go` `RunE` item above (auto-detect `$SHELL`, forward
`--install`/`--install-dir`, error listing supported shells for unset/unknown,
table-driven `t.Setenv` tests in `cmd/completion_test.go`). Do not action
`9awxuj` separately.
