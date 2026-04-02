---
title: "Get started"
description: "Load when setting up nd for the first time, troubleshooting installation, or onboarding a new user."
lastValidated: "2026-03-28"
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

Verify the installation:

```shell
nd version
```

## 2. Initialize

Create the nd configuration directory and default config:

```shell
nd init
```

This creates `~/.config/nd/config.yaml` with sensible defaults and sets up directories for profiles, snapshots, and state.

## 3. Add your first source

A **source** is a local directory or git repository containing agent assets organized by type.

```shell
# Local directory
nd source add ~/my-coding-assets

# Git repository (GitHub shorthand)
nd source add owner/repo

# Git repository (full URL)
nd source add https://github.com/owner/repo.git
```

nd scans the source for assets organized in convention-based directories (`skills/`, `agents/`, `commands/`, etc.). See [Creating sources](creating-sources.md) for how to structure your own.

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

Or run `nd deploy` with no arguments to get an interactive picker.

nd created a symlink from your agent's config directory (`~/.claude/skills/greeting`) back to the source. The source stays where it is: edit it and the change shows up immediately. See [How nd works](how-nd-works.md) for the full picture of what happens on disk.

## 6. Verify

Check that everything is healthy:

```shell
nd status
```

You should see your deployed assets with health indicators (checkmarks for healthy symlinks).

For a deeper health check of your entire setup:

```shell
nd doctor
```

## 7. Optional setup

### Shell completions

Enable tab-completion for your shell:

```shell
# Bash
nd completion bash --install

# Zsh
nd completion zsh --install

# Fish
nd completion fish --install
```

For zsh, you may need to add this to your `~/.zshrc` if not already present:

```shell
fpath+=~/.zfunc
autoload -Uz compinit && compinit
```

### Edit configuration

Open your config file in your default editor:

```shell
nd settings edit
```

## Next steps

- **[How nd works](how-nd-works.md):** What happens on disk when you deploy
- **[User guide](user-guide.md):** Learn about managing sources, scopes, syncing, and more
- **[Profiles & snapshots](profiles-and-snapshots.md):** Group assets into profiles and switch between them
- **[Configuration](configuration.md):** Customize nd behavior
- **[Creating sources](creating-sources.md):** Build and share your own asset libraries
