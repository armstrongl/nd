---
title: "Team and project config"
description: "Load when configuring shared team sources, git URLs, symlink strategy, or project-scoped versus user-scoped config."
lastValidated: "2026-07-05"
maxAgeDays: 90
weight: 45
paths:
  - "internal/config/**"
  - "internal/sourcemanager/**"
  - "cmd/deploy.go"
tags:
  - config
  - teams
  - sources
---

nd works for a single user out of the box, but the same building blocks share agent assets across a team: a git source everyone registers, a symlink strategy that survives different home directories, and a project-scoped config committed to the repository. This guide connects those pieces. For the full config schema, see [Configuration](configuration.md); for how sources are structured, see [Create asset sources](creating-sources.md).

## Share assets through a git source

A git source is the shared library. One person publishes a repository of assets, and every teammate registers it with [`nd source add`](../reference/nd_source_add.md). nd clones it to `~/.config/nd/sources/` and keeps it up to date with [`nd sync`](../reference/nd_sync.md).

```shell {filename="Terminal"}
# GitHub shorthand (expands to https://github.com/acme/agent-assets.git)
nd source add acme/agent-assets

# Pull the latest assets later
nd sync --source agent-assets
```

### SSH and git URLs

nd accepts several URL forms and leaves full URLs untouched. Shorthand like `owner/repo` expands to a GitHub HTTPS URL; anything with a scheme or a `git@` prefix is used as-is.

```shell {filename="Terminal"}
# HTTPS
nd source add https://github.com/acme/agent-assets.git

# SSH (scp-like syntax)
nd source add git@github.com:acme/agent-assets.git

# SSH (ssh:// scheme)
nd source add ssh://git@github.com/acme/agent-assets.git
```

Use an SSH form (`git@...` or `ssh://...`) when the repository is private and your team authenticates with SSH keys. nd runs `git clone` under the hood, so whatever authentication your `git` already uses applies here too.

## Portable symlinks

nd deploys assets as symlinks. The symlink strategy decides whether those links store an absolute path or a relative one, which matters when teammates have different home directories.

### Relative vs absolute symlinks

- **Absolute** (default): the link stores a full path such as `/Users/ada/.config/nd/sources/...`. Readable when debugging, but tied to one machine's layout.
- **Relative:** the link stores a path relative to its own location. Portable across machines where the home directory differs (different usernames or operating systems), which suits dotfiles synced between machines.

Override the strategy per deploy:

```shell {filename="Terminal"}
# Force relative symlinks for this deploy
nd deploy skills/greeting --relative

# Force absolute symlinks, ignoring a relative config default
nd deploy skills/greeting --absolute
```

The `--relative` and `--absolute` flags are mutually exclusive. To make the choice permanent, set `symlink_strategy` in your config:

```yaml {filename="config.yaml"}
symlink_strategy: relative
```

## Project config vs user config

nd reads two config layers. The user config lives at `~/.config/nd/config.yaml` and applies everywhere. A project config lives at `.nd/config.yaml` in a repository root and applies only inside that project. The project layer merges on top of the user layer, and CLI flags override both.

Commit a project config so everyone who clones the repository shares the same scope, sources, and strategy:

```yaml {filename=".nd/config.yaml"}
version: 1
default_scope: project
symlink_strategy: relative
sources:
  - id: project-assets
    type: local
    path: ./assets
```

With this file committed:

- `default_scope: project` deploys into the repository's `.claude/` directory instead of the user-wide `~/.claude/`, so assets stay scoped to the checkout.
- `symlink_strategy: relative` keeps links portable for every teammate.
- The `project-assets` source points at an in-repo `assets/` directory, so the assets travel with the code.

See [Configuration](configuration.md#config-merging) for the full merge order and precedence rules.

## Share a team setup

A typical rollout combines all three pieces:

```shell {filename="Terminal"}
# 1. Each teammate registers the shared git source over SSH
nd source add git@github.com:acme/agent-assets.git

# 2. Inside the project, deploy at project scope with portable links
nd deploy skills/house-style rules/no-emojis --scope project --relative

# 3. Pull updates when the shared repo changes
nd sync --source agent-assets
```

Commit a `.nd/config.yaml` with `default_scope: project` and `symlink_strategy: relative` so steps two and three need no extra flags: teammates run `nd deploy` and `nd sync` and get the shared behavior automatically.

## Next steps

- **[Configuration](configuration.md):** The full config schema, merge order, and every key
- **[Create asset sources](creating-sources.md):** Structure a source repository for your team
- **[How nd works](how-nd-works.md):** How absolute and relative symlinks resolve on disk
- **[User guide](user-guide.md):** Core workflows for deploying and syncing assets
- **[Troubleshooting](troubleshooting.md):** Fix broken symlinks and source scanning issues
