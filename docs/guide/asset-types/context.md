---
title: "Context"
description: "Load when modifying context asset scanning, context deployment paths, or context conflict handling."
lastValidated: "2026-04-04"
maxAgeDays: 90
weight: 60
paths:
  - "internal/sourcemanager/scanner.go"
  - "internal/deploy/**"
  - "internal/deploy/context.go"
tags:
  - context
  - assets
  - deployment
  - conflicts
---

Use context when you want project-wide conventions or persistent instructions that the agent reads automatically at the start of every session. Unlike rules, which state individual constraints, context provides broad project knowledge such as architecture decisions, coding standards, or team workflows.

Context assets provide persistent instructions or project conventions to coding agents, deployed to fixed paths derived from the target agent's configuration rather than into a type subdirectory.

## Directory layout

Each context asset is a directory containing a context file and an optional `_meta.yaml` sidecar.

```text
context/
├── go-project-rules/
│   ├── CLAUDE.md
│   └── _meta.yaml
└── coding-standards/
    └── CLAUDE.md
```

## File format

The context file (e.g., `CLAUDE.md` or `copilot-instructions.md`) contains the instructions or conventions in plain Markdown. The optional `_meta.yaml` sidecar carries metadata used by nd for listing and filtering.

```yaml
# _meta.yaml
description: "Project coding standards and conventions"
tags: ["standards", "conventions"]
target_language: "go"
target_project: "my-project"
target_agent: "claude-code"
```

The `target_agent` field controls which coding agent this context asset targets. Set it to `claude-code` (default) or `copilot` to indicate the intended agent. nd uses this when filtering or listing assets.

## Deploy behavior

Context assets deploy to **fixed paths determined by the target agent**, not into a `context/` subdirectory. This differs from all other asset types — see [How nd works](../how-nd-works.md#context-files-the-exception) for details on context deployment.

The deploy path depends on which agent nd is targeting:

- **Claude Code**: at project scope, nd symlinks to `./CLAUDE.md` at the project root — not inside `.claude/`. At global scope, nd symlinks to `~/.claude/CLAUDE.md`.
- **Copilot**: at project scope, nd symlinks to `.github/copilot-instructions.md` — inside the agent's project directory, not the project root. At global scope, nd symlinks to `~/.copilot/copilot-instructions.md`.

Files named `*.local.md` are treated as local-only and deploy at project scope regardless of the `--scope` flag.

### Context file renaming

When deploying a context asset to an agent that defines a default context filename, nd automatically renames the deployed file if all of these conditions are met:

1. The agent has a `DefaultContextFile` set (Copilot uses `copilot-instructions.md`; Claude Code does not rename)
2. The source file's name differs from the agent's default
3. The asset does not originate from the agent's own source alias
4. The file is not a `*.local.md` local-only context

For example, deploying a `CLAUDE.md` context asset to Copilot renames the symlink target to `copilot-instructions.md`, ensuring the agent can find it. This means multiple context assets targeting the same agent may collide — nd detects these collisions during bulk deploy and reports an error.

## Scope rules

| Scope | Claude Code target path | Copilot target path |
|-------|-------------------------|---------------------|
| Global | `~/.claude/CLAUDE.md` | `~/.copilot/copilot-instructions.md` |
| Project | `./CLAUDE.md` (project root) | `.github/copilot-instructions.md` (inside project dir) |
| Local (`*.local.md`) | `./<filename>` (project scope only) | `.github/<filename>` (project scope only) |

To undeploy a context asset, run [`nd remove`](../../reference/nd_remove.md) `context/go-standards`.

## Related

- [Asset type comparison](../creating-sources.md#asset-types) for a side-by-side overview of all types
- [Rules](rules.md) — enforce individual constraints rather than broad project knowledge
- [Agents](agents.md) — define a full persona or instruction set for the coding agent
- [Glossary: Context file](../glossary.md#context-file) — terminology definition

## Create a context asset

```shell
mkdir -p ~/my-assets/context/go-standards

cat > ~/my-assets/context/go-standards/CLAUDE.md << 'EOF'
Always use table-driven tests. Prefer stdlib over third-party libraries.
Return errors explicitly; do not panic in library code.
EOF

cat > ~/my-assets/context/go-standards/_meta.yaml << 'EOF'
description: "Go project conventions"
tags: ["go", "standards"]
EOF

nd deploy context/go-standards --scope project
```
