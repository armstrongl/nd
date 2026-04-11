---
title: "Troubleshoot"
description: "Load when debugging nd issues: broken symlinks, missing assets, config errors, profile switching problems, or context file conflicts."
lastValidated: "2026-04-04"
maxAgeDays: 90
weight: 70
paths:
  - "cmd/doctor.go"
  - "cmd/sync.go"
  - "internal/deploy/**"
tags:
  - troubleshooting
  - debugging
  - doctor
---

Start with [`nd doctor`](../reference/nd_doctor.md) to identify the category of problem, then find the matching section below.

## Run nd doctor

`nd doctor` runs five checks and reports pass, warn, or fail for each:

| Check | What it validates |
|-------|-------------------|
| Config | Global config file (`~/.config/nd/config.yaml`) parses and passes schema validation |
| Sources | Each registered source path exists and is accessible; reports asset count per source |
| Deployments | All deployed symlinks point to valid targets; flags broken links |
| Agents | Each configured agent is detected on the system; global config directory exists |
| Git | `git` binary is available in `$PATH` |

```shell
# Run all checks
nd doctor

# Machine-readable output for CI
nd doctor --json
```

`nd doctor` exits with a non-zero code when any check fails, making it suitable for CI pipelines and pre-commit hooks.

## Broken symlinks

**Symptoms:** [`nd status`](../reference/nd_status.md) shows unhealthy assets with an `✗` indicator. `nd doctor` reports deployment issues.

**Common causes:**

- Source directory was moved or deleted
- Git source was not synced after upstream changes
- Builtin cache was cleared (deleted `~/.cache/nd/`)
- Source was removed from nd but symlinks remain

**Fix:**

Use [`nd sync`](../reference/nd_sync.md) to repair broken symlinks:

```shell
# Repair all broken symlinks
nd sync

# Pull and repair a specific git source
nd sync --source <source-id>

# Re-create a specific link
nd remove skills/greeting && nd deploy skills/greeting

# Full diagnostic
nd doctor
```

## Missing assets

**Symptoms:** [`nd list`](../reference/nd_list.md) does not show expected assets. [`nd source list`](../reference/nd_source_list.md) shows the source but it reports zero assets.

**Common causes:**

- Source directory structure does not match the convention (wrong directory names like `skill/` instead of `skills/`)
- An `nd-source.yaml` manifest overrides convention scanning with incorrect paths
- Source path in config points to the wrong location

**Fix:**

```shell
# Verify the registered path
nd source list

# Check that subdirectories match expected names
ls <source-path>/

# Correct the path if needed
nd settings edit
```

Use [`nd settings edit`](../reference/nd_settings_edit.md) to fix source paths. The expected directory names are: `skills/`, `agents/`, `commands/`, `output-styles/`, `rules/`, `context/`, `plugins/`, `hooks/`. See [Create asset sources](creating-sources.md) for the full convention.

## Configuration problems

**Symptoms:** nd commands fail with YAML parse errors. Settings do not take effect.

**Common causes:**

- Invalid YAML syntax in `~/.config/nd/config.yaml`
- Project config (`.nd/config.yaml`) overrides global settings unexpectedly
- Unknown config keys from a different nd version

**Fix:**

```shell
# Validate config (doctor checks this first)
nd doctor

# Open config for manual inspection
nd settings edit

# Nuclear option: delete and reinitialize
# Warning: this removes all source registrations; profiles and snapshots
# remain on disk but the config referencing them is gone.
rm ~/.config/nd/config.yaml && nd init
```

If a project config exists, check `.nd/config.yaml` in your project root. Project config merges on top of global config, and CLI flags override both. See [Configuration](configuration.md) for the full merging order.

## Profile switch problems

**Symptoms:** Assets remain after switching profiles. Unexpected assets appear after a switch.

**Common causes:**

- Assets were deployed manually (outside a profile) and are not managed by profile switching
- Pinned assets persist across switches by design
- The profile references assets that no longer exist in any source

**Fix:**

```shell
# Check origin of each deployed asset
nd status

# Unpin an asset to let profile switching manage it
nd unpin skills/greeting

# Update a profile with a new asset
nd profile add-asset my-setup skills/greeting
```

The `nd status` output shows the origin of each asset: `manual`, `pinned`, or the profile name. Use [`nd unpin`](../reference/nd_unpin.md) to release pinned assets back to profile management, or [`nd profile add-asset`](../reference/nd_profile_add-asset.md) to add missing assets to a profile. Only assets with a profile origin are managed during switches. See [Profiles and snapshots](profiles-and-snapshots.md) for the full workflow.

## Context file conflicts

**Symptoms:** Deploying a context file reports that a backup was created.

**Explanation:** Only one context file can occupy each target location (e.g., `~/.claude/CLAUDE.md` for Claude Code, or `~/.copilot/copilot-instructions.md` for Copilot CLI). When you deploy a second context file to the same path, nd backs up the existing one to `~/.config/nd/backups/` before replacing it.

**Fix:**

```shell
# Check for backed-up context files
ls ~/.config/nd/backups/

# Deploy the desired context file explicitly
nd deploy context/go-standards --scope project
```

See [Context files](asset-types/context.md) for the special scoping rules that determine where context assets are deployed.

## Agent not detected

**Symptoms:** `nd doctor` reports `? Agent: <name> (not detected)` or the global directory is not found.

**Common causes:**

- The agent binary is not installed or not in `$PATH`
- The agent's global config directory does not exist yet (first-time setup)
- Custom agent config in `config.yaml` has incorrect directory paths

**Fix:**

```shell
# Check if the agent binary exists
which claude        # Claude Code
which copilot-cli   # Copilot CLI

# Create the global config directory if needed (Claude Code shown; Copilot CLI uses ~/.copilot/)
mkdir -p ~/.claude

# Verify agent configuration
nd settings edit
```

## Ambiguous asset name

**Symptoms:** [`nd deploy`](../reference/nd_deploy.md) `greeting` fails with `ambiguous asset "greeting" — matches: ...`

**Cause:** Multiple assets share the same name across different types or sources (e.g., `skills/greeting` and `commands/greeting`).

**Fix:** Use the `type/name` format to disambiguate:

```shell
nd deploy skills/greeting
```

Or use the `--type` flag:

```shell
nd deploy --type skills greeting
```

## Non-TTY confirmation error

**Symptoms:** A command fails with `confirmation required but stdin is not a terminal (use --yes to skip)`.

**Cause:** nd requires interactive confirmation for destructive operations ([`nd remove`](../reference/nd_remove.md), [`nd uninstall`](../reference/nd_uninstall.md), profile switch). In non-TTY environments (pipes, scripts, CI), there is no terminal to prompt.

**Fix:** Pass `--yes` to skip the confirmation:

```shell
nd source remove my-assets --yes
```

## Config already exists

**Symptoms:** [`nd init`](../reference/nd_init.md) fails with `config already exists at ~/.config/nd/config.yaml; edit with 'nd settings edit'`.

**Cause:** nd has already been initialized. `nd init` refuses to overwrite an existing config to prevent accidental data loss.

**Fix:** Use `nd settings edit` to modify your existing config. If you need to start fresh, delete the config first:

```shell
rm ~/.config/nd/config.yaml && nd init
```

## No active profile

**Symptoms:** [`nd profile deploy`](../reference/nd_profile_deploy.md) (with no name argument) fails with `no active profile; use 'nd profile deploy <name>' instead`.

**Cause:** You ran `nd profile deploy` without specifying a profile name, and no profile is currently active. nd only knows which profile to redeploy if one was previously activated.

**Fix:** Specify the profile name explicitly:

```shell
nd profile deploy my-setup
```

Or switch to a profile first with [`nd profile switch`](../reference/nd_profile_switch.md) `my-setup`.

## Deploy conflict

**Symptoms:** `nd deploy` fails with `conflict at <path>: existing <kind> blocks deployment of <asset>`.

**Cause:** A file or symlink that nd does not manage already exists at the target location. This happens when you have manually created a file where nd wants to place a symlink (e.g., a hand-written `CLAUDE.md` at `~/.claude/CLAUDE.md`, or `copilot-instructions.md` at `~/.copilot/copilot-instructions.md` for Copilot CLI).

**Fix:** Move or remove the conflicting file, then retry the deploy:

```shell
# Claude Code example (Copilot CLI equivalent: ~/.copilot/copilot-instructions.md)
mv ~/.claude/CLAUDE.md ~/.claude/CLAUDE.md.bak
nd deploy context/my-rules
```

For context files specifically, nd automatically backs up the existing file to `~/.config/nd/backups/` and replaces it. Conflicts only block deployment for non-context asset types.

## Copilot CLI path differences

nd supports multiple agents, each with its own directory layout. If you switched from Claude Code to Copilot CLI (or use both), note these differences:

| | Claude Code | Copilot CLI |
|---|---|---|
| Global config directory | `~/.claude/` | `~/.copilot/` |
| Project config directory | `.claude/` | `.github/` |
| Context file name | `CLAUDE.md` | `copilot-instructions.md` |

Most troubleshooting steps in this guide show Claude Code paths. Substitute the Copilot CLI equivalents when debugging Copilot deployments. Use `nd doctor` to confirm which agents nd detects and where it expects their directories.

## Git not found

**Symptoms:** `nd doctor` reports git is not available. `nd sync` cannot pull git sources.

**Fix:** Install git and ensure it is in your `$PATH`. On macOS, `xcode-select --install` installs git. On Linux, use your package manager (`apt install git`, `dnf install git`).

## Related pages

- **[`nd doctor` reference](../reference/nd_doctor.md):** Full flag and option reference for the doctor command
- **[`nd sync` reference](../reference/nd_sync.md):** Full reference for repairing symlinks and pulling git sources
- **[Configuration](configuration.md):** Config file locations, merging order, and all available settings
- **[How nd works](how-nd-works.md):** Understand symlinks, scopes, and what happens on disk
- **[Profiles and snapshots](profiles-and-snapshots.md):** Profile switching, pinning, and snapshot workflows
