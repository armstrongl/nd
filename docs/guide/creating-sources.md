---
title: "Creating sources"
weight: 50
---

An asset source is a directory (local or git) containing coding agent assets organized by type. This guide explains how to structure your own.

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
| `skills` | Directory | Yes | Multi-file skill definitions |
| `agents` | File | Yes | Agent configuration files |
| `commands` | File | Yes | Custom command definitions |
| `output-styles` | File | Yes | Output formatting styles (requires manual settings.json registration) |
| `rules` | File | Yes | Rule files for agent behavior |
| `context` | Folder-per-asset | Yes | Context files (special deployment rules: see below) |
| `plugins` | Directory | No | Plugin packages (uses export workflow, not symlinks) |
| `hooks` | Directory | Yes | Hook definitions (requires manual settings.json registration) |

## Context files

Context files have special deployment rules:

- **Global scope:** Deployed to the agent's global directory (e.g., `~/.claude/CLAUDE.md`)
- **Project scope:** Deployed to the project root directly (e.g., `./CLAUDE.md`), not inside `.claude/`
- **Local files** (`*.local.md`): Can only be deployed at project scope

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

When an `nd-source.yaml` manifest is present, it **overrides** convention-based scanning entirely. Convention-based discovery only happens if no manifest is found.

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

Others can add it with:

```shell
nd source add you/my-assets
# or
nd source add https://github.com/you/my-assets.git
```

Git sources are cloned to `~/.config/nd/sources/` and can be synced with `nd sync --source <id>`.
