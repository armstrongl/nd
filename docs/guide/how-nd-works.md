---
title: "How nd works"
description: "Load when modifying symlink creation, deploy logic, scope handling, or debugging broken deployments."
lastValidated: "2026-04-05"
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

When you run [`nd deploy`](../reference/nd_deploy.md) `skills/greeting`, nd creates a symlink from your agent's config directory back to the original source. The source stays where it is. Edit the source, and the change shows up instantly in the deployed location: no redeploy needed.

nd supports multiple agents. When no `--agent` flag is provided, nd targets the configured default agent if detected, falling back to the first detected agent. You can target a specific agent with the `--agent` flag on any command (e.g., `--agent copilot`). Each agent has its own config directories, supported asset types, and context file conventions.

## The mental model

nd wires each deployed asset from your source into the target agent's config directory:

```text {filename="Source layout"}
  your source               nd               agent config dir
┌────────────────┐                        ┌──────────────────┐
│ ~/my-assets/   │  ─── nd deploy ───▶    │ Claude Code:     │
│   skills/      │                        │   ~/.claude/     │
│   rules/       │                        ├──────────────────┤
│   agents/      │  ─── nd deploy ───▶    │ Copilot CLI:     │
│   context/     │      --agent copilot   │   ~/.copilot/    │
└────────────────┘                        └──────────────────┘
```

The same source can serve multiple agents. nd creates links into whichever agent you target. You manage the source; nd manages the wiring.

## What deploys look like

Here is a source directory with two assets: a skill (directory) and a rule (file):

```text {filename="Source layout"}
~/my-assets/
├── skills/
│   └── greeting/
│       └── SKILL.md
└── rules/
    └── no-emojis.md
```

After running `nd deploy skills/greeting rules/no-emojis`, your agent's config directory looks like this:

```text {filename="Source layout"}
~/.claude/                                              # Claude Code (default)
├── skills/
│   └── greeting -> ~/my-assets/skills/greeting         # directory symlink
└── rules/
    └── no-emojis.md -> ~/my-assets/rules/no-emojis.md  # file symlink
```

The target directory depends on the agent. For Copilot CLI, assets deploy into `~/.copilot/` instead. Pass `--agent copilot` to target Copilot.

nd creates the parent directories (for example, `~/.claude/skills/`) if they don't already exist.

The `->` arrow shows where the symlink points. `greeting` is a directory symlink (the whole skill folder), while `no-emojis.md` is a file symlink. Both point back to the original source.

Verify this:

```shell {filename="Terminal"}
ls -la ~/.claude/skills/
# greeting -> /Users/you/my-assets/skills/greeting
```

## Global vs project scope

nd deploys to one of two places depending on the scope. The exact directories depend on which agent you target:

| Agent | Global directory | Project directory |
|---|---|---|
| Claude Code (default) | `~/.claude/` | `<project>/.claude/` |
| Copilot CLI | `~/.copilot/` | `<project>/.github/` |

**Global scope** (default) deploys to the agent's user-wide config directory:

```text {filename="Deployment paths"}
# Claude Code
~/.claude/skills/greeting -> ~/my-assets/skills/greeting

# Copilot CLI
~/.copilot/skills/greeting -> ~/my-assets/skills/greeting
```

**Project scope** deploys to the current project's config directory:

```text {filename="Deployment paths"}
# Claude Code
~/myproject/.claude/skills/greeting -> ~/my-assets/skills/greeting

# Copilot CLI
~/myproject/.github/skills/greeting -> ~/my-assets/skills/greeting
```

Use global scope for assets you want everywhere. Use project scope for assets that only make sense in a specific repo.

```shell {filename="Terminal"}
# Global (default)
nd deploy skills/greeting

# Project
nd deploy skills/greeting --scope project

# Target a specific agent
nd deploy skills/greeting --agent copilot
```

See the [configuration guide](configuration.md) for how to change the default scope.

## Context files (the exception)

Context files break the pattern. Every other asset type deploys into a subdirectory (`skills/`, `rules/`, `agents/`, and others). Context files deploy directly into the config directory or project root, and each agent expects a different filename.

### Claude Code

Claude Code reads context from a file named `CLAUDE.md`.

**Global scope:** deploys into the agent's config directory:

```text {filename="Deployment paths"}
~/.claude/CLAUDE.md -> ~/my-assets/context/go-project-rules/CLAUDE.md
```

**Project scope:** deploys into the project root, NOT inside `.claude/`:

```text {filename="Deployment paths"}
~/myproject/CLAUDE.md -> ~/my-assets/context/go-project-rules/CLAUDE.md
```

This is intentional. Claude Code reads project-level context from the project root (`./CLAUDE.md`), not from `.claude/CLAUDE.md`.

### Copilot CLI

Copilot CLI reads context from a file named `copilot-instructions.md`.

**Global scope:** deploys into the agent's config directory:

```text {filename="Deployment paths"}
~/.copilot/copilot-instructions.md -> ~/my-assets/context/go-project-rules/CLAUDE.md
```

**Project scope:** deploys into the project directory, NOT the project root:

```text {filename="Deployment paths"}
~/myproject/.github/copilot-instructions.md -> ~/my-assets/context/go-project-rules/CLAUDE.md
```

Note the difference from Claude Code: Copilot context goes inside `.github/`, not at the project root.

### Automatic context file renaming

When a context source has a filename that doesn't match the target agent's expected name, nd renames the deployed symlink automatically. For example, deploying a `CLAUDE.md` source to Copilot CLI creates a link named `copilot-instructions.md`. This means a single context source can serve both agents without manual renaming.

### Context rules

Two things to keep in mind:

- **Local-only context files** (`*.local.md`) can only be deployed at project scope. Attempting to deploy them globally fails with an error.
- **One context file per target.** If you deploy a second context file to the same location, nd backs up the existing file to `~/.config/nd/backups/` before replacing it.

## Absolute vs relative symlinks

nd supports two symlink strategies. The default is absolute:

**Absolute** (default): the symlink target is a full path:

```text {filename="Deployment paths"}
~/.claude/skills/greeting -> /Users/you/my-assets/skills/greeting
```

**Relative:** the symlink target is a relative path from the link's location:

```text {filename="Deployment paths"}
~/.claude/skills/greeting -> ../../my-assets/skills/greeting
```

Absolute symlinks show the full path, making them more readable when debugging. Use relative symlinks if you sync your dotfiles across machines where your home directory path differs (different usernames or OS layouts).

```shell {filename="Terminal"}
# Deploy with relative symlinks
nd deploy skills/greeting --relative

# Or set it as the default in config
# symlink_strategy: relative
```

## What the agent sees

Once you deploy an asset, the target agent typically picks it up. Each agent loads assets from its own config directory at session start, and project-scope assets when you run it from that project's directory.

Not all agents support all asset types:

| Asset type | Claude Code | Copilot CLI |
|---|---|---|
| skills | ✓ | ✓ |
| agents | ✓ | ✓ |
| context | ✓ | ✓ |
| commands | ✓ | — |
| output-styles | ✓ | — |
| rules | ✓ | — |
| hooks | ✓ | — |

nd prevents you from deploying unsupported types to an agent.

For Claude Code, two asset types need an extra step after deploying:

- **Hooks** and **output-styles** require manual registration in Claude Code's `settings.json`. nd creates the symlink, but Claude Code needs to be told about them in its settings file. Check Claude Code's documentation for the specific settings entries.

For everything else, deploy and go: no additional nd configuration required.

## Next steps

- **[User guide](user-guide.md):** Core workflows for deploying, removing, and managing assets
- **[Profiles and snapshots](profiles-and-snapshots.md):** Group assets into named profiles and switch between them without touching individual symlinks
- **[Creating sources](creating-sources.md):** Structure your own asset directories for nd to discover
- **[`nd deploy` reference](../reference/nd_deploy.md):** Full flag and option reference for the deploy command
- **[Troubleshooting](troubleshooting.md):** Fix broken symlinks, missing assets, and other common issues
