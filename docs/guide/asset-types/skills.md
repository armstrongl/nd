---
title: "Skills"
description: "Load when modifying skill directory scanning, skill deployment, or the skills asset type."
lastValidated: "2026-04-04"
maxAgeDays: 90
weight: 10
paths:
  - "internal/asset/scanner.go"
  - "internal/deploy/**"
tags:
  - skills
  - assets
  - deployment
---

Skills are multi-file directory assets that package reusable coding-agent behaviors into a named, self-contained directory.

## Directory layout

```text
skills/
├── greeting/
│   └── greeting.md
└── code-review/
    ├── code-review.md
    └── examples/
        └── sample-review.md
```

## File format

The entry point is a markdown file that typically shares the directory name (e.g., `greeting/greeting.md`). It may include YAML frontmatter. Supporting files inside the directory can use any format and are deployed alongside the entry point.

## Deploy behavior

nd symlinks the entire skill directory into the target location. Running `nd deploy skills/greeting` produces:

```text
~/.claude/skills/greeting → <source>/skills/greeting
```

The agent sees the full directory contents through the symlink.

## Scope rules

| Scope | Target path |
|-------|-------------|
| Global | `~/.claude/skills/<name>` |
| Project | `.claude/skills/<name>` |

## Related

- [Asset type comparison](../creating-sources.md#asset-types) for a side-by-side overview of all types

## Create a skill

```shell
mkdir -p ~/my-assets/skills/greeting
cat > ~/my-assets/skills/greeting/greeting.md << 'EOF'
# Greeting skill

When the user says hello, respond with a friendly greeting.
EOF
nd deploy skills/greeting
```
