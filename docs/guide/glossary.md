---
title: "Glossary"
description: "Load when encountering unfamiliar nd terminology or when disambiguating overloaded terms like agent, context, or command."
lastValidated: "2026-04-03"
maxAgeDays: 90
weight: 5
paths:
  - "internal/nd/**"
  - "internal/asset/**"
tags:
  - terminology
  - reference
  - glossary
---

Terms used throughout nd documentation, organized alphabetically. Terms in `code font` are values you type in commands or config files.

## Agent (asset type)

A deployable asset that configures how a coding agent behaves. Agent assets are single Markdown files stored in the `agents/` directory of a source.

Not to be confused with [coding agent](#coding-agent), which is the AI tool (like Claude Code) that consumes agent assets.

```text
my-source/
└── agents/
    └── code-reviewer.md    # This is an agent asset
```

See [Create asset sources](creating-sources.md) for the full directory layout.

## Agent config directory

The directory where a coding agent reads its configuration. For Claude Code, this is `~/.claude/` (global) or `.claude/` (project). nd creates symlinks inside this directory when you deploy assets.

Configure custom agent directories in your [config file](configuration.md) using the `agents[]` array.

## Asset

Any deployable unit that nd manages. nd supports eight asset types: [skills](#skill), [agents](#agent-asset-type), [commands](#command), [output styles](#output-style), [rules](#rule), [context files](#context-file), [plugins](#plugin), and [hooks](#hook).

Each asset has a unique identity: the combination of its source, type, and name. Reference assets in commands using the format `type/name` (for example, `skills/greeting`).

## Builtin source

A source embedded inside the nd binary. The builtin source ships nd-specific assets (skills, commands, and an agent) and is always present. It has the reserved ID `builtin` and the lowest priority — your sources override it when names collide.

## Coding agent

An AI-powered development tool that reads assets from a config directory. Claude Code is the default coding agent. nd also supports other agents (Cursor, Windsurf, Copilot) through the `agents[]` config array.

Not to be confused with [agent (asset type)](#agent-asset-type), which is a file you deploy *to* a coding agent.

## Command

A deployable asset that defines a custom slash command for a coding agent. Command assets are single Markdown files stored in the `commands/` directory of a source.

In Claude Code, deployed commands become available as `/command-name` in the chat interface.

## Context file

A deployable asset that provides instructions or context to a coding agent. Context files have special deployment rules that differ from other asset types:

- **Global scope**: deploys to the agent config directory (for example, `~/.claude/CLAUDE.md`)
- **Project scope**: deploys to the project root (for example, `./CLAUDE.md`), not inside `.claude/`
- **Local files** (`*.local.md`): deploy only at project scope

Context files use a folder-per-asset layout with an optional `_meta.yaml` sidecar for metadata.

Not to be confused with *context window*, which is the amount of text a language model processes at once.

See [How nd works](how-nd-works.md#context-files-the-exception) for deployment details.

## Deploy

Create a symlink from a coding agent's config directory to an asset in a source. Deploying does not copy files — it creates a link so that edits to the source appear instantly in the deployed location.

```shell
nd deploy skills/greeting
```

The opposite of deploy is [remove](#remove).

See [How nd works](how-nd-works.md) for what happens on disk.

## Doctor

A comprehensive health check command that validates your entire nd setup. `nd doctor` checks five areas:

1. Config file validity
2. Source accessibility
3. Deployment health (broken, drifted, or orphaned symlinks)
4. Coding agent detection
5. Git availability

Run `nd doctor` after editing config files or when deployments behave unexpectedly.

## Export

Package assets from a source into a standalone plugin directory or marketplace listing. Exporting is the deployment path for [plugins](#plugin) — unlike other asset types, plugins use `nd export` instead of symlink deployment.

```shell
nd export                    # Export a plugin
nd export marketplace        # Generate a marketplace listing
```

## Health status

The condition of a deployed symlink. nd tracks five states:

| Status | Meaning |
|---|---|
| **OK** | Symlink exists and points to the correct target |
| **Broken** | Symlink exists but the target is missing |
| **Drifted** | Symlink points to the wrong target |
| **Orphaned** | The source no longer contains this asset |
| **Missing** | The symlink was deleted externally |

Check health with `nd status` or `nd doctor`. Repair issues with `nd sync`.

## Hook

A deployable asset that defines lifecycle hooks for a coding agent. Hook assets are directories stored in the `hooks/` directory of a source.

Hooks require manual registration in the coding agent's `settings.json` after deployment. nd creates the symlink but does not modify `settings.json`.

## Manifest

An optional `nd-source.yaml` file at the root of a source that overrides convention-based asset discovery. Use a manifest when your source does not follow nd's standard directory layout.

When a manifest is present, nd uses it exclusively and ignores convention-based scanning.

See [Create asset sources](creating-sources.md#manifest-file) for the schema.

## Operation log

A JSONL file at `~/.config/nd/logs/operations.log` that records every mutating operation nd performs (deploys, removals, syncs, profile switches, and others). Each entry includes a timestamp, operation type, affected assets, and success/failure counts.

nd rotates the log automatically when it exceeds 1 MB.

## Origin

How an asset was deployed. nd tracks three origins:

| Origin | Meaning |
|---|---|
| `manual` | Deployed directly via `nd deploy` |
| `pinned` | Locked in place via `nd pin` |
| `profile:<name>` | Deployed as part of a named profile |

Origin determines what happens during a [profile switch](#profile): nd keeps pinned and manually deployed assets, and swaps profile-origin assets.

## Output style

A deployable asset that controls how a coding agent formats its output. Output style assets are single Markdown files stored in the `output-styles/` directory of a source.

Like [hooks](#hook), output styles require manual registration in the coding agent's `settings.json` after deployment.

## Pin

Lock a deployed asset so it persists across profile switches. nd neither removes nor redeploys pinned assets when you switch profiles.

```shell
nd pin skills/greeting      # Lock the asset
nd unpin skills/greeting    # Release the lock
```

Pinning changes the asset's [origin](#origin) to `pinned`.

## Plugin

A deployable asset type that uses an export workflow instead of symlink deployment. Plugin assets are directories stored in the `plugins/` directory of a source.

Export plugins with `nd export` and generate marketplace listings with `nd export marketplace`. Plugins cannot be included in [profiles](#profile) or [snapshots](#snapshot).

## Profile

A named collection of assets that you deploy and switch between as a group. Profiles work like browser profiles for your coding agent — for example, a "work" profile with enterprise tools and a "personal" profile with hobby project assets.

```shell
nd profile create work --assets skills/jira,agents/reviewer
nd profile switch personal
```

nd stores profiles as YAML files in `~/.config/nd/profiles/`.

See [Profiles and snapshots](profiles-and-snapshots.md) for the full workflow.

## Remove

Delete a deployed symlink, disconnecting the asset from the coding agent's config directory. nd removes only the symlink, not the source file.

```shell
nd remove skills/greeting
```

The opposite of [deploy](#deploy).

## Rule

A deployable asset that defines behavioral rules for a coding agent. Rule assets are single Markdown files stored in the `rules/` directory of a source.

The coding agent auto-loads rules from its config directory. Deployment requires no manual registration.

## Scope

Where nd deploys an asset. nd supports two scopes:

| Scope | Target directory | Use case |
|---|---|---|
| `global` (default) | Agent config directory (`~/.claude/`) | Assets you want in every project |
| `project` | Project config directory (`.claude/`) | Assets specific to one project |

```shell
nd deploy skills/greeting              # Global (default)
nd deploy skills/greeting --scope project  # Project
```

See [How nd works](how-nd-works.md#global-vs-project-scope) for details.

## Skill

A deployable asset that teaches a coding agent a reusable workflow or capability. Skill assets are directories containing a required `SKILL.md` entry-point file and optional supporting files, stored in the `skills/` directory of a source.

Skills are specific to Claude Code. Other coding agents use different terminology for similar concepts (tools, instructions, prompts).

```text
my-source/
└── skills/
    └── greeting/
        └── SKILL.md
```

## Snapshot

A point-in-time record of all current deployments. Snapshots act as bookmarks you can restore to return to a known-good state.

nd creates auto-snapshots before destructive operations (profile switches, snapshot restores) as a safety net. nd retains the 5 most recent auto-snapshots.

```shell
nd snapshot save before-experiment
nd snapshot restore before-experiment
```

See [Profiles and snapshots](profiles-and-snapshots.md#snapshots) for the full workflow.

## Source

A directory containing assets organized by type. nd supports three source types:

| Type | Description |
|---|---|
| `local` | A directory on your filesystem |
| `git` | A cloned Git repository |
| `builtin` | Assets embedded in the nd binary |

Register sources with `nd source add`. nd scans sources for assets in convention-based directories (`skills/`, `agents/`, `commands/`, and others).

See [Create asset sources](creating-sources.md) for how to structure your own.

## Sync

Pull the latest changes from git sources and repair broken symlinks across all deployments. `nd sync` fixes [health status](#health-status) issues like broken, drifted, or orphaned symlinks.

```shell
nd sync                        # Repair all deployments
nd sync --source <source-id>   # Pull a git source and repair
nd sync --dry-run              # Preview repairs without applying
```

## Symlink strategy

How nd constructs the symlink path when deploying. nd supports two strategies:

| Strategy | Example | Use case |
|---|---|---|
| `absolute` (default) | `~/.claude/skills/greeting -> /Users/you/my-assets/skills/greeting` | Readable paths for debugging |
| `relative` | `~/.claude/skills/greeting -> ../../my-assets/skills/greeting` | Portable across machines with different home directory paths |

Set the default in your config file with `symlink_strategy: relative` or per-deploy with `--relative`.
