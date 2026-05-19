---
title: "Offer shell completion install during init"
id: "ce0bw6"
status: pending
priority: medium
type: feature
tags: ["cli", "completions", "onboarding"]
created_at: "2026-04-20"
context:
  - "cmd/init_cmd.go"
  - "cmd/completion.go"
  - "cmd/init_cmd_test.go"
  - "cmd/completion_test.go"
  - "cmd/helpers.go"
  - "cmd/app.go"
  - "cmd/root.go"
  - "docs/guide/getting-started.md"
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
    check: "Interactive `nd init` with SHELL=/bin/zsh prints an 'Install shell completions for zsh? [y/N]' prompt after the 'Detected agents:' line and before the built-in deploy prompt; answering y writes ~/.zfunc/_nd and prints the install path; answering n/Enter skips with no error"
  - type: assert
    check: "`nd init --yes`, `nd init --json`, and `nd init --quiet` never prompt for or install completions; an unset or unsupported $SHELL silently skips the step; a filesystem write error during install prints a warning to stderr but `nd init` still exits 0"
---

## Offer shell completion install during init

### Objective

Add an **opt-in, interactive-only** shell completion install step to the `nd init`
command. Today shell completions are only reachable through the hidden
`nd completion <shell> --install` subcommand (`cmd/completion.go:24`,
`Hidden: true`), so first-time users never discover tab completion. This task
surfaces it during onboarding: after `nd init` writes the config and prints the
detected agents, detect the user's shell from `$SHELL`, ask whether to install
completions, and on yes call the existing `installCompletion` helper.

Source of truth: GitHub issue #78 "Offer to install shell completions during
init" (https://GitHub.com/armstrongl/nd/issues/78). Scope per the issue: detect
shell from `$SHELL`, support bash/zsh/fish, install to the standard per-shell
location, and make it an optional step.

Out of scope: the bare-`nd completion` `$SHELL` auto-detect described in sibling
task `tasks/cli/37uilg-expand-parent-cmd-rune.md` (issue #62). That task adds a
`RunE` to the `completion` parent command and is independent of this one — do
not implement it here, and do not depend on its `$SHELL`-mapping helper (it does
not exist yet). This task adds its own small `$SHELL` -> {bash,zsh,fish} mapping.

### Where this plugs in

`nd init`'s `RunE` is `newInitCmd` in `cmd/init_cmd.go:33-72`. The current flow is:

1. `cmd/init_cmd.go:37-39` — error out if config already exists.
2. `cmd/init_cmd.go:41-44` — `runInitSetup` creates dirs + writes default config.
3. `cmd/init_cmd.go:46-54` — detect agents, then `displayAgentDetection` (only when `!app.JSON && !app.Quiet`).
4. `cmd/init_cmd.go:56` — `deployBuiltinAssets` (prompts to deploy built-ins).
5. `cmd/init_cmd.go:58-68` — if `app.JSON`, build and print the JSON envelope, then `return` (this is an early return — any human-only step must run before it or be guarded by `!app.JSON`).
6. `cmd/init_cmd.go:70` — return `deployErr`.

Insert the completion step in `RunE` **after** `displayAgentDetection`
(`cmd/init_cmd.go:53`) and **before** `deployBuiltinAssets`
(`cmd/init_cmd.go:56`). It must run only on the human, interactive path.

Do NOT add this to `runInitSetup` (`cmd/init_cmd.go:78`): that helper is also
called from the first-run auto-init prompt `offerInit` in `cmd/root.go:228`, and
completions are out of scope for that path. Keeping the new logic in
`newInitCmd`'s `RunE` confines it to the explicit `nd init` command.

### Existing pieces to reuse (verified)

- `installCompletion(cmd *cobra.Command, genFn func(*bytes.Buffer) error, dir, filename string) error` — `cmd/completion.go:190`. Generates into a buffer, `MkdirAll`s the dir, writes the file, and on success prints `"Completion script installed to <path>\n"` to `cmd.OutOrStdout()`. Returns a non-nil error on any failure (gen/mkdir/write); callers must treat that error as non-fatal here.
- Per-shell default dirs: `defaultBashCompletionDir()` `cmd/completion.go:162`, `defaultZshCompletionDir()` `cmd/completion.go:174` (returns `~/.zfunc`), `defaultFishCompletionDir()` `cmd/completion.go:182` (returns `~/.config/fish/completions`). Each returns `""` if the home dir can't be resolved — treat `""` as "skip".
- Completion generators (call on `cmd.Root()`): bash `rootCmd.GenBashCompletionV2(buf, true)`, zsh `rootCmd.GenZshCompletion(buf)`, fish `rootCmd.GenFishCompletion(buf, true)` — see `cmd/completion.go:61,67-69` / `:107,113-115` / `:146,152-154`. Install filenames: bash `"nd"`, zsh `"_nd"`, fish `"nd.fish"` (`cmd/completion.go:69,115,154`).
- `confirm(r io.Reader, w io.Writer, prompt string, yesFlag bool) (bool, error)` — `cmd/helpers.go:58`. Prints `"<prompt> [y/N] "`, reads a line, returns true for `y`/`yes`. Returns an error if stdin is not a terminal. Pass `yesFlag=false` so `--yes` does NOT auto-accept (completions are opt-in only).
- `isTerminal()` — `cmd/helpers.go:115`. Use to skip the step entirely when not interactive.
- Flags live on `App`: `app.Yes` (`cmd/app.go:30`), `app.JSON` (`cmd/app.go:27`), `app.Quiet` (`cmd/app.go:26`). Registered as persistent flags in `cmd/root.go:60-63`.
- Writers/reader inside `RunE`: `cmd.OutOrStdout()`, `cmd.ErrOrStderr()`, `cmd.InOrStdin()` (already used in `deployBuiltinAssets`, `cmd/init_cmd.go:161,195,248`).

### Tasks

- [ ] Add a helper in `cmd/init_cmd.go` (or `cmd/completion.go`), e.g. `detectShellFromEnv() (shell string, ok bool)`, that reads `os.Getenv("SHELL")`, takes `filepath.Base` of the value, and maps `bash`->"bash", `zsh`->"zsh", `fish`->"fish". Return `("", false)` when `$SHELL` is empty or maps to none of these. Add a small unit test table for it.
- [ ] Add a helper, e.g. `offerCompletionInstall(cmd *cobra.Command, app *App)`, that: returns immediately if `app.JSON || app.Quiet || app.Yes || !isTerminal()`; calls `detectShellFromEnv` and returns if `!ok`; resolves the per-shell default dir via the matching `default*CompletionDir()` and returns (printing nothing) if it is `""`; prompts via `confirm(cmd.InOrStdin(), cmd.OutOrStdout(), fmt.Sprintf("Install shell completions for %s?", shell), false)`; on confirmed-true calls `installCompletion` with the shell's generator closure, default dir, and filename (`nd`/`_nd`/`nd.fish`); on a non-nil `installCompletion` error prints `"warning: could not install completions: %v\n"` to `cmd.ErrOrStderr()` and returns nil (never propagate).
- [ ] Call the new helper in `newInitCmd`'s `RunE` between `cmd/init_cmd.go:54` (after `displayAgentDetection`) and `cmd/init_cmd.go:56` (before `deployBuiltinAssets`). It must not return an error and must not affect `deployErr` / the JSON path.
- [ ] Verify the `--yes` carve-out: `confirm` is called with `yesFlag=false`, and the early-return guard also short-circuits on `app.Yes`, so `nd init --yes` neither prompts nor installs.
- [ ] Add unit tests in `cmd/init_cmd_test.go` mirroring the existing table style there (use `t.Setenv("SHELL", ...)`, `rootCmd.SetIn(...)` with a `strings.NewReader("y\n")`, and `bytes.Buffer` for out/err as in `TestInitCmd_WithYes`). Note: the existing tests run non-interactively so `isTerminal()` is false — add tests that exercise the helper directly (not only via `rootCmd.Execute`) so the prompt path is reachable, OR factor the interactive guard so tests can inject a fake "is interactive" + reader. Cover: each of bash/zsh/fish detected from `SHELL`; unsupported `SHELL` (e.g. `/bin/csh`) skips; unset `SHELL` skips; `--yes` skips; `--json` skips; `--quiet` skips; an `installCompletion` failure (point the dir at an unwritable path, e.g. a path under a file) prints a warning but `nd init` still returns nil and the config still exists.
- [ ] Update the `nd init` long description at `cmd/init_cmd.go:23` to mention the optional shell-completion step (one sentence, same tone as the existing copy).
- [ ] Update `docs/guide/getting-started.md`: in section "## 2. Initialize" (line 54, after the built-in-deploy paragraph at line 64) note that interactive `nd init` also offers to install shell completions for the detected shell; and add a cross-reference from the "### Shell completions" section (line 181) noting it can also be installed during `nd init`.

### Acceptance criteria

- Interactive `nd init` with `SHELL=/bin/zsh` prints `Install shell completions for zsh? [y/N]` AFTER the `Detected agents:` line and BEFORE the `Deploy N built-in asset(s)?` prompt.
- Answering `y` writes the zsh script to `~/.zfunc/_nd` and prints `Completion script installed to <path>` (the existing `installCompletion` output, `cmd/completion.go:211`).
- Answering `n` or pressing Enter skips installation, prints no error, and `nd init` continues to the built-in deploy step.
- `SHELL=/usr/bin/fish nd init` (interactive) offers fish completions and, on yes, writes `~/.config/fish/completions/nd.fish`.
- `SHELL=/bin/bash nd init` (interactive) offers bash completions and, on yes, writes to `defaultBashCompletionDir()` as `nd`.
- An unset `$SHELL` or an unsupported value (e.g. `/bin/csh`, `/bin/tcsh`) silently skips the completion step with no output.
- `nd init --yes` does NOT prompt for or install completions (opt-in only; `confirm` called with `yesFlag=false` and guard short-circuits on `app.Yes`).
- `nd init --json` and `nd init --quiet` skip the completion prompt entirely; JSON output envelope is unchanged.
- A filesystem error from `installCompletion` (e.g. dir not creatable) prints `warning: could not install completions: ...` to stderr but `nd init` still exits 0 and the config file still exists.
- `go build -o nd .`, `go test ./cmd/...`, `go test -race ./cmd/...`, and `golangci-lint run ./cmd/...` all pass; all existing `cmd/init_cmd_test.go` tests still pass alongside the new ones.

### References

- GitHub issue: https://GitHub.com/armstrongl/nd/issues/78
- Close this issue when the task is completed.
- Related (independent, do not depend on): `tasks/cli/37uilg-expand-parent-cmd-rune.md` — bare `nd completion` `$SHELL` auto-detect (issue #62)
- Insertion point: `cmd/init_cmd.go:33-72` (`newInitCmd` RunE); shared setup helper boundary `cmd/init_cmd.go:78` / `cmd/root.go:228`
- Reuse: `installCompletion` `cmd/completion.go:190`; default dirs `cmd/completion.go:162,174,182`; `confirm` `cmd/helpers.go:58`; `isTerminal` `cmd/helpers.go:115`
- Docs: `docs/guide/getting-started.md` lines 54 ("## 2. Initialize") and 181 ("### Shell completions")
