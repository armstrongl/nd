---
title: "Agents"
description: "Load when modifying agent file scanning, agent deployment, or the agents asset type."
lastValidated: "2026-04-04"
maxAgeDays: 90
weight: 20
paths:
  - "internal/sourcemanager/scanner.go"
  - "internal/deploy/**"
tags:
  - agents
  - assets
  - deployment
---

Use agents when you want to define a distinct persona or set of instructions that shapes how a coding agent behaves across an entire session. Unlike rules, which enforce individual constraints, an agent file provides holistic behavioral identity.

Both Claude Code and Copilot CLI support agent assets.

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

nd symlinks the individual file into the target location (see [How nd works](../how-nd-works.md) for details on the symlink strategy). Running [`nd deploy`](../../reference/nd_deploy.md) `agents/researcher` produces a symlink in the active agent's config directory:

```text
# Claude Code
~/.claude/agents/researcher.md → <source>/agents/researcher.md

# Copilot CLI
~/.copilot/agents/researcher.md → <source>/agents/researcher.md
```

## Scope rules

| Scope | Claude Code | Copilot CLI |
|-------|-------------|-------------|
| Global | `~/.claude/agents/<name>.md` | `~/.copilot/agents/<name>.md` |
| Project | `.claude/agents/<name>.md` | `.github/agents/<name>.md` |

To undeploy an agent, run [`nd remove`](../../reference/nd_remove.md) `agents/researcher`.

## Related

- [Asset type comparison](../creating-sources.md#asset-types) for a side-by-side overview of all types
- [Rules](rules.md) — enforce individual constraints rather than defining a full persona
- [Context](context.md) — provide broad project knowledge the agent reads automatically
- [Glossary: Agent](../glossary.md#agent-asset-type) — terminology definition

## Create an agent

```shell
cat > ~/my-assets/agents/researcher.md << 'EOF'
# Researcher agent

You are a research assistant. When given a topic, search thoroughly and provide well-sourced summaries.
EOF
nd deploy agents/researcher
```
