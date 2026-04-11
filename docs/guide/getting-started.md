---
title: "Get started"
description: "Load when setting up nd for the first time, troubleshooting installation, or onboarding a new user."
lastValidated: "2026-04-05"
maxAgeDays: 90
weight: 10
paths:
  - "cmd/init.go"
  - "cmd/source.go"
  - "cmd/deploy.go"
tags:
  - setup
  - installation
  - onboarding
---

nd is an asset manager for AI coding agents. It organizes reusable agent components — skills, agents, commands, rules, context files, output styles, hooks, and plugins — into source directories and deploys them as symlinks into your agent's config directory. Use nd when you want to version, share, or switch between sets of agent assets without copying files around.

This guide takes you from zero to your first deployed asset in about 5 minutes.

## 1. Install nd

Choose your preferred method:

```shell
# Homebrew (macOS/Linux)
brew install --cask armstrongl/tap/nd

# Go install
go install github.com/armstrongl/nd@latest

# Or build from source
git clone https://github.com/armstrongl/nd.git && cd nd && go build -o nd .
```

Verify the installation with [`nd version`](../reference/nd_version.md):

```shell
nd version
```

### Update nd

If you installed nd via Homebrew, update it with:

```shell
brew update && brew upgrade nd
```

If `brew upgrade nd` installs an older version, your local tap index may be stale. Run `brew update` first to refresh it, then upgrade again.

nd also notifies you when a newer version is available — the message appears after a command completes, once per day.

## 2. Initialize

Create the nd configuration directory and default config:

```shell
nd init
```

This creates `~/.config/nd/config.yaml` with sensible defaults and sets up directories for profiles, snapshots, and state. nd also detects all installed coding agents (Claude Code, Copilot CLI, and others) and selects a default agent to deploy to.

[`nd init`](../reference/nd_init.md) then prompts you to deploy nd's built-in assets (skills, commands, and an agent) to the detected default agent. Answer **y** to deploy them immediately so you have something to work with, or **n** to skip — you can deploy them later with [`nd deploy`](../reference/nd_deploy.md) `--source builtin`. Pass `--yes` to skip the prompt entirely and deploy automatically.

If nd cannot detect any coding agent (e.g., none are installed or not in `$PATH`), it skips the built-in deploy with a warning and continues. Install an agent and run `nd deploy --source builtin` afterward.

If a config file already exists, `nd init` exits with an error. Use [`nd settings edit`](../reference/nd_settings_edit.md) to modify an existing configuration.

Browse the built-in assets with [`nd list`](../reference/nd_list.md):

```shell
nd list
```

## 3. Add your first source

nd ships with a **builtin** source containing nd-specific assets. To add your own assets, register a **source** with [`nd source add`](../reference/nd_source_add.md): a local directory or git repository containing agent assets organized by type.

```shell
# Local directory
nd source add ~/my-coding-assets

# Git repository (GitHub shorthand)
nd source add owner/repo

# Git repository (full URL)
nd source add https://github.com/owner/repo.git
```

nd scans the source for assets organized in convention-based directories (`skills/`, `agents/`, `commands/`, and others). See [Creating sources](creating-sources.md) for how to structure your own.

## 4. Browse available assets

List all assets discovered from your sources:

```shell
nd list
```

Filter by type:

```shell
nd list --type skills
```

Assets marked with `*` are already deployed.

## 5. Deploy an asset

Deploy an asset by creating a symlink in your agent's config directory:

```shell
nd deploy skills/greeting
```

Deploy multiple assets at once:

```shell
nd deploy skills/greeting commands/hello agents/researcher
```

Or run `nd deploy` with no arguments to get an interactive picker. Many nd commands support this interactive mode — [`nd remove`](../reference/nd_remove.md), [`nd profile switch`](../reference/nd_profile_switch.md), [`nd snapshot restore`](../reference/nd_snapshot_restore.md), and others present a picker when run without arguments. nd disables interactive mode in non-TTY environments (pipes, scripts) and when `--json` is set.

nd creates a symlink from your agent's config directory (e.g., `~/.claude/skills/greeting` for Claude Code) back to the source. The source stays where it is: edit it and the change shows up immediately. See [How nd works](how-nd-works.md) for the full picture of what happens on disk.

**Deploy by type:**

```shell
nd deploy --type skills greeting
```

**Scopes:**

- **Global** (`--scope global`, default): Deploys to your agent's global config directory (e.g., `~/.claude/` for Claude Code, `~/.copilot/` for Copilot CLI)
- **Project** (`--scope project`): Deploys to the project-level config directory (e.g., `.claude/` for Claude Code, `.github/` for Copilot CLI)

```shell
nd deploy skills/greeting --scope project
```

**Target a specific agent:**

If you have multiple agents installed, nd deploys to the default agent. Use `--agent` to target a different one:

```shell
nd deploy skills/greeting --agent copilot
```

Note that not all agents support every asset type. Copilot CLI supports skills, agents, and context only.

**Symlink strategy:**

- **Absolute** (default): Symlinks use absolute paths
- **Relative** (`--relative`): Symlinks use relative paths (better for portable setups)

```shell
nd deploy skills/greeting --relative
```

Change the default strategy in your config file (`symlink_strategy: relative`).

## 6. Verify

Check that everything is healthy with [`nd status`](../reference/nd_status.md):

```shell
nd status
```

The output shows your deployed assets with health indicators (checkmarks for healthy symlinks).

For a full health check, run [`nd doctor`](../reference/nd_doctor.md):

```shell
nd doctor
```

## 7. Optional setup

### Shell completions

Enable tab-completion for your shell with [`nd completion`](../reference/nd_completion.md):

```shell
# Print completion script
nd completion bash
nd completion zsh
nd completion fish

# Auto-install to standard location
nd completion bash --install
nd completion zsh --install
nd completion fish --install

# Install to custom directory
nd completion zsh --install-dir ~/.my-completions
```

For zsh, add this to your `~/.zshrc` if not already present:

```shell
fpath+=~/.zfunc
autoload -Uz compinit && compinit
```

### Edit configuration

Open your config file in your default editor:

```shell
nd settings edit
```

## Uninstall

To remove all nd-managed symlinks from your agent's config directory, run [`nd uninstall`](../reference/nd_uninstall.md):

```shell
nd uninstall
```

This does not delete your sources, config, profiles, or snapshots — it only removes the deployed symlinks. Pass `--yes` to skip the confirmation prompt.

## Next steps

- **[How nd works](how-nd-works.md):** What happens on disk when you deploy
- **[Profiles & snapshots](profiles-and-snapshots.md):** Group assets into profiles and switch between them
- **[Configuration](configuration.md):** Customize nd behavior
- **[Creating sources](creating-sources.md):** Build and share your own asset libraries
- **[Glossary](glossary.md):** Definitions for nd terms like asset, source, scope, and profile
