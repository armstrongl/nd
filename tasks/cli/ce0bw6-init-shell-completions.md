---
title: "Offer shell completion install during init"
id: "ce0bw6"
status: pending
priority: medium
type: feature
tags: ["cli", "completions", "onboarding"]
created_at: "2026-04-20"
---

## Offer shell completion install during init

### Objective

Add an optional shell completion installation step to `nd init`. After creating the config directory and detecting agents, detect the user's shell from `$SHELL` and offer to install completions using the existing `nd completion <shell> --install` infrastructure. This improves onboarding by making tab completion available without the user needing to discover the hidden `completion` subcommand.

### Tasks

- [ ] Read `$SHELL` environment variable and map it to a supported shell (bash, zsh, fish); fall back gracefully if `$SHELL` is unset or maps to an unsupported shell
- [ ] After agent detection in `runInitSetup` or `newInitCmd.RunE`, prompt the user: "Install shell completions for <shell>? [y/N]"
- [ ] Skip the prompt entirely in non-interactive mode (`--yes` should NOT auto-install completions; completions are opt-in only)
- [ ] Skip the prompt when `--json` or `--quiet` flags are set
- [ ] On acceptance, call the existing `installCompletion` helper from `cmd/completion.go` with the detected shell's default directory and filename
- [ ] Print the installed path on success (reuse the existing completion install output format)
- [ ] Print a non-fatal warning on failure (do not abort `nd init` if completion install fails)
- [ ] Add unit tests: mock `$SHELL` to each supported value, verify prompt appears; verify unsupported shell skips prompt; verify `--yes` skips prompt; verify install failure does not propagate as init error
- [ ] Update `nd init` long description and `docs/` getting-started guide to mention the completion step

### Acceptance criteria

- Running `nd init` in a zsh shell shows "Install shell completions for zsh? [y/N]" after agent detection
- Answering "y" writes the completion script to `~/.zfunc/_nd` (or the shell-appropriate default path)
- Answering "n" or pressing Enter skips installation with no error
- `$SHELL=/usr/bin/fish nd init` detects fish and offers fish completions
- Unsupported or missing `$SHELL` silently skips the completion step
- `nd init --yes` does NOT install completions (opt-in only)
- `nd init --json` and `nd init --quiet` skip the completion prompt
- A filesystem permission error during install prints a warning but `nd init` still succeeds
- All new and existing `cmd/init_cmd_test.go` tests pass

### References

- https://GitHub.com/armstrongl/nd/issues/78
