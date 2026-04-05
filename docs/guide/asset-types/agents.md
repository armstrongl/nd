---
title: "Agents"
description: "Load when modifying agent file scanning, agent deployment, or the agents asset type."
lastValidated: "2026-04-04"
maxAgeDays: 90
weight: 20
paths:
  - "internal/asset/scanner.go"
  - "internal/deploy/**"
tags:
  - agents
  - assets
  - deployment
---

Agents are single-file assets that define the behavior, persona, or instructions for a named coding agent.

## Directory layout

```text
agents/
├── researcher.md
└── code-reviewer.md
```

## File format

Each agent is a markdown file containing agent instructions, a system prompt, or behavioral rules. There are no required frontmatter fields, but frontmatter may be included for tooling purposes.

## Deploy behavior

nd symlinks the individual file into the target location. Running `nd deploy agents/researcher` produces:

```text
~/.claude/agents/researcher.md → <source>/agents/researcher.md
```

## Scope rules

| Scope | Target path |
|-------|-------------|
| Global | `~/.claude/agents/<name>.md` |
| Project | `.claude/agents/<name>.md` |

## Related

- [Asset type comparison](../creating-sources.md#asset-types) for a side-by-side overview of all types

## Create an agent

```shell
cat > ~/my-assets/agents/researcher.md << 'EOF'
# Researcher agent

You are a research assistant. When given a topic, search thoroughly and provide well-sourced summaries.
EOF
nd deploy agents/researcher
```
