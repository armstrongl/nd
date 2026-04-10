---
title: "Create asset sources"
description: "Load when modifying source scanning, asset type discovery, manifest parsing, or the directory convention."
lastValidated: "2026-04-05"
maxAgeDays: 90
weight: 50
paths:
  - "internal/sourcemanager/**"
  - "internal/asset/**"
tags:
  - sources
  - assets
  - scanning
---

An asset source is a directory that organizes coding agent assets by type. nd supports three source types: **local** (a directory on disk), **git** (a cloned repository), and **builtin** (embedded in the nd binary). This guide explains how to structure your own local or git source.

## Directory convention

nd discovers assets by looking for directories named after asset types:

```text
my-assets/
├── skills/
│   ├── greeting/           # Directory asset
│   └── code-review/        # Directory asset
├── agents/
│   └── researcher.md       # File asset
├── commands/
│   └── deploy-all.md       # File asset
├── output-styles/
│   └── concise.md          # File asset
├── rules/
│   └── no-emojis.md        # File asset
├── context/
│   ├── go-project-rules/   # Folder-per-asset layout
│   │   ├── CLAUDE.md       # Context file
│   │   └── _meta.yaml      # Optional metadata
│   └── coding-standards/
│       └── CLAUDE.md
├── plugins/
│   └── my-plugin/          # Directory asset (not symlink-deployed)
└── hooks/
    └── pre-commit/         # Directory asset
```

Not every directory needs to be present. nd only discovers assets in directories that exist.

## Asset types

| Type | Format | Deployable | Description |
|------|--------|------------|-------------|
| [`skills`](asset-types/skills.md) | Directory | Yes | Multi-file skill definitions |
| [`agents`](asset-types/agents.md) | File | Yes | Agent configuration files |
| [`commands`](asset-types/commands.md) | File | Yes | Custom command definitions |
| [`output-styles`](asset-types/output-styles.md) | File | Yes | Output formatting styles (requires manual settings.json registration) |
| [`rules`](asset-types/rules.md) | File | Yes | Rule files for agent behavior |
| [`context`](asset-types/context.md) | Folder-per-asset | Yes | Context files (special deployment rules: see below) |
| [`plugins`](asset-types/plugins.md) | Directory | No | Plugin packages (uses export workflow, not symlinks) |
| [`hooks`](asset-types/hooks.md) | Directory | Yes | Hook definitions (requires manual settings.json registration) |

## Context files

Context files have special deployment rules:

- **Global scope:** Deployed to the agent's global directory (e.g., `~/.claude/CLAUDE.md`)
- **Project scope:** Deployed to the project root directly (e.g., `./CLAUDE.md`), not inside `.claude/`
- **Local files** (`*.local.md`): Deploy only at project scope

### _meta.yaml

Context files can include a `_meta.yaml` sidecar for metadata:

```yaml
description: "Project coding standards and conventions"
tags: ["standards", "conventions"]
```

## Manifest file

For sources that don't follow the convention-based directory structure, create an `nd-source.yaml` manifest at the source root:

```yaml
# nd-source.yaml
version: 1
paths:
  skills:
    - custom/path/to/skills
  agents:
    - my-agents
exclude:
  - vendor/
```

When an `nd-source.yaml` manifest is present, it **overrides** convention-based scanning entirely. nd falls back to convention-based discovery only when no manifest exists.

## Publish your source

To share your asset source, push it to git:

```shell
cd my-assets
git init
git add .
git commit -m "Initial asset collection"
git remote add origin https://github.com/you/my-assets.git
git push -u origin main
```

Others can add it with [`nd source add`](../reference/nd_source_add.md):

```shell
nd source add you/my-assets
# or
nd source add https://github.com/you/my-assets.git
```

nd clones git sources to `~/.config/nd/sources/`. Sync them with [`nd sync`](../reference/nd_sync.md) `--source <id>`.

## Remove a source

Remove a registered source with [`nd source remove`](../reference/nd_source_remove.md):

```shell
nd source remove <source-id>
```

If assets from the source are currently deployed, nd asks whether to remove them, keep them as orphans, or cancel. nd prevents removal of the `builtin` source.

> **Warning:** `nd source remove <id> --yes` skips the interactive prompt and **removes all deployed assets** from that source without confirmation. This is a destructive operation — use it only in scripts or when you are certain you want a clean removal.

## Next steps

- **[How nd works](how-nd-works.md):** Understand what happens on disk when assets are deployed as symlinks
- **[User guide](user-guide.md):** Core workflows for deploying, removing, and managing assets from your sources
- **[Profiles and snapshots](profiles-and-snapshots.md):** Group assets into named profiles and switch between them
- **[`nd source` reference](../reference/nd_source.md):** Full flag and option reference for all source subcommands
- **[Troubleshooting](troubleshooting.md):** Fix missing assets, broken symlinks, and source scanning issues
