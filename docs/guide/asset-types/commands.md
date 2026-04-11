---
title: "Commands"
description: "Load when modifying command file scanning, command deployment, or the commands asset type."
lastValidated: "2026-04-04"
maxAgeDays: 90
weight: 30
paths:
  - "internal/sourcemanager/scanner.go"
  - "internal/deploy/**"
tags:
  - commands
  - assets
  - deployment
---

Use commands when you want to give your agent a repeatable action you can trigger on demand with a slash command. Unlike context or rules, which apply passively, a command runs only when you explicitly invoke it.

Commands are a Claude Code-only asset type. Copilot CLI does not support commands.

Commands are single-file assets that define custom slash commands available to coding agents during a session.

## Directory layout

```text {filename="Source layout"}
commands/
├── deploy-all.md
└── review-pr.md
```

## File format

Each command is a markdown file whose base filename becomes the slash command name. For example, `deploy-all.md` registers as `/deploy-all`. The file contains the instructions the agent follows when the command is invoked. Frontmatter is optional.

## Deploy behavior

nd symlinks the individual file into the target location (see [How nd works](../how-nd-works.md) for details on the symlink strategy). Running [`nd deploy`](../../reference/nd_deploy.md) `commands/deploy-all` produces:

```text {filename="Deployment paths"}
~/.claude/commands/deploy-all.md → <source>/commands/deploy-all.md
```

## Scope rules

| Scope | Target path |
|-------|-------------|
| Global | `~/.claude/commands/<name>.md` |
| Project | `.claude/commands/<name>.md` |

To undeploy a command, run [`nd remove`](../../reference/nd_remove.md) `commands/deploy-all`.

## Related

- [Asset type comparison](../creating-sources.md#asset-types) for a side-by-side overview of all types
- [Skills](skills.md) — multi-file directory assets for complex, multi-step behaviors
- [Agents](agents.md) — define a full persona or instruction set for the coding agent
- [Glossary: Command](../glossary.md#command) — terminology definition

## Create a command

```shell {filename="Terminal"}
cat > ~/my-assets/commands/deploy-all.md << 'EOF'
Deploy all available assets from all sources using nd deploy.
List assets first with nd list, then deploy each one.
EOF
nd deploy commands/deploy-all
```
