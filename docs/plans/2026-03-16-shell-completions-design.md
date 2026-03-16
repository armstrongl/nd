# Shell Completions Design (FR-035)

| Field | Value |
|---|---|
| **Date** | 2026-03-16 |
| **Author** | Larah |
| **Status** | Draft |
| **Spec ref** | FR-035 |
| **Priority** | Could Have |

## Overview

Add shell completion support to nd for Bash, Zsh, and Fish. Completions include both static (commands, flags) and dynamic (asset names, profile names, snapshot names, source IDs) suggestions.

## Command structure

A hidden `completion` parent command with three subcommands:

```text
nd completion bash [--install]
nd completion zsh  [--install]
nd completion fish [--install]
```

- **Hidden**: `completion` command has `Hidden: true` — does not appear in `nd help` or `nd --help`
- **Stdout default**: Each subcommand prints the completion script to stdout
- **`--install` flag**: Writes to the conventional location and prints sourcing instructions

### Install locations

| Shell | `--install` path | Notes |
|-------|-----------------|-------|
| Bash | `~/.local/share/bash-completion/completions/nd` (XDG) | Falls back to `~/.bash_completion.d/nd` if XDG dir missing |
| Zsh | `~/.zfunc/_nd` | Creates dir if needed |
| Fish | `~/.config/fish/completions/nd.fish` | Standard Fish completions dir |

### Help text

Each subcommand's `Long` description includes manual install steps:

- Bash: `nd completion bash > ~/.local/share/bash-completion/completions/nd`
- Zsh: `nd completion zsh > ~/.zfunc/_nd` (with note to add `fpath+=~/.zfunc` and `autoload -Uz compinit && compinit` to `~/.zshrc` if not already present)
- Fish: `nd completion fish > ~/.config/fish/completions/nd.fish`

The parent `nd completion` with no subcommand prints usage showing `bash`, `zsh`, and `fish` options.

## Dynamic completions

Cobra's `ValidArgsFunction` provides context-aware suggestions at tab-completion time. The `cobra.ShellCompDirectiveNoFileComp` directive is used for all dynamic completions to prevent shells from falling back to filename completion when no matches are found.

### Positional argument completions

| Command | Completes | Source |
|---------|-----------|--------|
| `nd deploy <TAB>` | Available (undeployed) asset names | Index scan |
| `nd remove <TAB>` | Deployed asset names | State file |
| `nd pin <TAB>` / `nd unpin <TAB>` | Deployed asset names | State file |
| `nd profile switch <TAB>` | Profile names | Profile store |
| `nd profile deploy <TAB>` | Profile names | Profile store |
| `nd profile delete <TAB>` | Profile names | Profile store |
| `nd profile add-asset <TAB>` | Profile names (1st arg), asset names (2nd arg) | Profile store, index scan |
| `nd snapshot restore <TAB>` | Snapshot names | Snapshot store |
| `nd snapshot delete <TAB>` | Snapshot names | Snapshot store |
| `nd source remove <TAB>` | Source IDs | Config file |

### Flag value completions

Registered via `cmd.RegisterFlagCompletionFunc()`:

| Flag | Completes | Source |
|------|-----------|--------|
| `--scope <TAB>` | `global`, `project` | Static list (registered on `rootCmd` since it is a persistent flag) |
| `--type <TAB>` (deploy, list) | Asset type names | `nd.AllAssetTypes()` static list |
| `--source <TAB>` (list, sync) | Source IDs | Config file |

### Implementation pattern

Each command's `ValidArgsFunction` uses the `*App` instance already available via closure capture from the command constructor (`newDeployCmd(app *App)` etc.). Since `PersistentPreRunE` is not guaranteed to have run during completion, each `ValidArgsFunction` calls a lightweight init helper that expands `ConfigPath` and sets `BackupDir` without full validation. This helper is idempotent and safe to call multiple times.

Errors in completion functions are silently swallowed — completion must never fail visibly or produce error output. On failure, return an empty list with `cobra.ShellCompDirectiveNoFileComp`.

### Performance constraint

Completion functions must respond within 200ms. The index scan is the heaviest operation. If it exceeds the budget, fall back to completing from the state file (deployed assets only) rather than scanning all sources.

## Files

| File | Purpose |
|------|---------|
| `cmd/completion.go` | Hidden `completion` parent + `bash`/`zsh`/`fish` subcommands, `--install` flag logic |
| `cmd/completion_test.go` | Unit tests for completion output and install behavior |

Dynamic `ValidArgsFunction` implementations are added directly to existing command files (`deploy.go`, `remove.go`, `pin.go`, `profile.go`, `snapshot.go`, `source.go`, `list.go`, `sync.go`).

A shared `completionInitApp(app *App)` helper in `cmd/helpers.go` handles lightweight App initialization for completion contexts.

## Testing strategy

### Unit tests for completion command

- Verify `nd completion bash` output contains expected Bash markers (e.g., `__nd_` function prefix)
- Verify `nd completion zsh` output contains `#compdef nd` header
- Verify `nd completion fish` output contains Fish completion markers
- Verify `--install` writes to temp dir path with correct content for each shell
- Verify `--install` on unwritable path returns clear error with manual alternative
- Verify `nd completion` with no subcommand prints usage
- Verify Bash completions include descriptions (`includeDesc: true`)

### Unit tests for ValidArgsFunction

- Mock the App services (source manager, state store, profile store)
- Verify each function returns expected suggestions matching available assets/profiles/snapshots
- Verify empty list returned on service initialization failure
- Verify `RegisterFlagCompletionFunc` for `--type`, `--scope`, `--source` flags

### Not tested

Actual shell interpretation of generated scripts (invoking `bash --rcfile` or `zsh -c 'compinit'`) is fragile in CI and not worth the maintenance cost. We test the generation output correctness.

## Error handling

| Scenario | Behavior |
|----------|----------|
| `--install` on unwritable path | Clear error message with the manual `>` redirect alternative |
| Dynamic completion failure | Return empty list silently (Cobra convention) |
| `nd completion` with no subcommand | Print usage showing `bash`, `zsh`, and `fish` options |
| Unknown shell argument | Cobra's built-in unknown-subcommand error |

## Cobra API details

| Shell | Generator method | Key parameter |
|-------|-----------------|---------------|
| Bash | `GenBashCompletionV2(w, includeDesc)` | `includeDesc: true` — includes descriptions in completion output for better discoverability |
| Zsh | `GenZshCompletion(w)` | N/A |
| Fish | `GenFishCompletion(w, includeDesc)` | `includeDesc: true` |

## Dependencies

- Cobra's built-in completion generation: `GenBashCompletionV2()`, `GenZshCompletion()`, `GenFishCompletion()`
- Existing services: source manager (index scan), state store (deployed assets), profile store (profiles/snapshots)
- No new external dependencies required
