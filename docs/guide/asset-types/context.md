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

Context assets provide persistent instructions or project conventions to coding agents, deployed to fixed paths derived from the context filename rather than into a type subdirectory.

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

The context file (e.g., `CLAUDE.md`) contains the instructions or conventions in plain Markdown. The optional `_meta.yaml` sidecar carries metadata used by nd for listing and filtering.

```yaml
# _meta.yaml
description: "Project coding standards and conventions"
tags: ["standards", "conventions"]
target_language: "go"
target_project: "my-project"
target_agent: "claude-code"
```

## Deploy behavior

Context assets deploy to **fixed paths determined by the context filename**, not into a `context/` subdirectory. This differs from all other asset types.

For a context asset whose context file is `CLAUDE.md`:

- At global scope, nd symlinks to `~/.claude/CLAUDE.md`
- At project scope, nd symlinks to `./CLAUDE.md` at the project root — not inside `.claude/`

Files named `*.local.md` are treated as local-only and deploy at project scope regardless of the `--scope` flag.

## Scope rules

| Scope | Target path |
|-------|-------------|
| Global | `~/.claude/<filename>` |
| Project | `./<filename>` (project root) |
| Local (`*.local.md`) | `./<filename>` (project scope only) |

## Related

- [Asset type comparison](../creating-sources.md#asset-types) for a side-by-side overview of all types

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
