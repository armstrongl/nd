---
title: "How nd works"
description: "Load when modifying symlink creation, deploy logic, scope handling, or debugging broken deployments."
lastValidated: "2026-03-28"
maxAgeDays: 90
weight: 20
paths:
  - "internal/deploy/**"
  - "cmd/deploy.go"
  - "cmd/remove.go"
tags:
  - deploy
  - symlinks
  - scope
---

nd doesn't copy files. It creates symlinks.

When you run `nd deploy skills/greeting`, nd creates a symlink from your agent's config directory back to the original source. The source stays where it is. Edit the source, and the change shows up instantly in the deployed location: no redeploy needed.

## The mental model

nd wires each deployed asset from your source into the agent's config directory:

```text
  your source               nd               agent config dir
┌────────────────┐     (creates link)     ┌──────────────────┐
│ ~/my-assets/   │  ─── nd deploy ───▶    │ ~/.claude/        │
│   skills/      │                        │   skills/         │
│   rules/       │                        │   rules/          │
│   agents/      │                        │   agents/         │
└────────────────┘                        └──────────────────┘
```

Your files stay in the source. nd creates links so the agent can find them. You manage the source; nd manages the wiring.

## What deploys look like

Here is a source directory with two assets: a skill (directory) and a rule (file):

```text
~/my-assets/
├── skills/
│   └── greeting/
│       └── SKILL.md
└── rules/
    └── no-emojis.md
```

After running `nd deploy skills/greeting rules/no-emojis`, your agent's config directory looks like this:

```text
~/.claude/
├── skills/
│   └── greeting -> ~/my-assets/skills/greeting   # directory symlink
└── rules/
    └── no-emojis.md -> ~/my-assets/rules/no-emojis.md   # file symlink
```

nd creates the parent directories (`~/.claude/skills/`, `~/.claude/rules/`) if they don't already exist.

The `->` arrow shows where the symlink points. `greeting` is a directory symlink (the whole skill folder), while `no-emojis.md` is a file symlink. Both point back to the original source.

Verify this:

```shell
ls -la ~/.claude/skills/
# greeting -> /Users/you/my-assets/skills/greeting
```

## Global vs project scope

nd deploys to one of two places depending on the scope:

**Global scope** (default) deploys to your agent's user-wide config directory:

```text
~/.claude/skills/greeting -> ~/my-assets/skills/greeting
```

**Project scope** deploys to the current project's config directory:

```text
~/myproject/.claude/skills/greeting -> ~/my-assets/skills/greeting
```

Use global scope for assets you want everywhere. Use project scope for assets that only make sense in a specific repo.

```shell
# Global (default)
nd deploy skills/greeting

# Project
nd deploy skills/greeting --scope project
```

See the [configuration guide](configuration.md) for how to change the default scope.

## Context files (the exception)

Context files break the pattern. Every other asset type deploys into a subdirectory (`skills/`, `rules/`, `agents/`, and others). Context files deploy directly into the config directory or project root.

**Global scope:** deploys into the agent's config directory:

```text
~/.claude/CLAUDE.md -> ~/my-assets/context/go-project-rules/CLAUDE.md
```

**Project scope:** deploys into the project root, NOT inside `.claude/`:

```text
~/myproject/CLAUDE.md -> ~/my-assets/context/go-project-rules/CLAUDE.md
```

This is intentional. Claude Code reads project-level context from the project root (`./CLAUDE.md`), not from `.claude/CLAUDE.md`.

Two things to keep in mind:

- **Local-only context files** (`*.local.md`) can only be deployed at project scope. Attempting to deploy them globally fails with an error.
- **One context file per target.** If you deploy a second context file to the same location, nd backs up the existing file to `~/.config/nd/backups/` before replacing it.

## Absolute vs relative symlinks

nd supports two symlink strategies. The default is absolute:

**Absolute** (default): the symlink target is a full path:

```text
~/.claude/skills/greeting -> /Users/you/my-assets/skills/greeting
```

**Relative:** the symlink target is a relative path from the link's location:

```text
~/.claude/skills/greeting -> ../../my-assets/skills/greeting
```

Absolute symlinks show the full path, making them more readable when debugging. Use relative symlinks if you sync your dotfiles across machines where your home directory path differs (different usernames or OS layouts).

```shell
# Deploy with relative symlinks
nd deploy skills/greeting --relative

# Or set it as the default in config
# symlink_strategy: relative
```

## What the agent sees

Once you deploy an asset, Claude Code uses it. Claude Code loads skills, agents, commands, and rules from `~/.claude/` in your next session. It loads project-scope assets when you run it from that project's directory.

Two asset types need an extra step after deploying:

- **Hooks** and **output-styles** require manual registration in Claude Code's `settings.json`. nd creates the symlink, but Claude Code needs to be told about them in its settings file. Check Claude Code's documentation for the specific settings entries.

For everything else, deploy and go: no additional nd configuration required.
