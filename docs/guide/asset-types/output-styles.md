---
title: "Output styles"
description: "Load when modifying output style scanning, deployment, or settings.json registration behavior."
lastValidated: "2026-04-04"
maxAgeDays: 90
weight: 40
paths:
  - "internal/asset/scanner.go"
  - "internal/deploy/**"
tags:
  - output-styles
  - assets
  - deployment
---

Output styles are single-file assets that define formatting instructions for agent output, deployed as symlinks and activated via manual `settings.json` registration.

## Directory layout

```text
output-styles/
├── concise.md
└── learning.md
```

## File format

Each output style is a plain Markdown file describing the formatting behavior the agent should apply when the style is active. There is no required frontmatter.

```markdown
Respond with minimal text. Use bullet points. No explanations unless asked.
Maximum 3 sentences per response.
```

## Deploy behavior

nd symlinks the `.md` file into the target scope directory. After deployment, you must manually add the style to your agent's `settings.json` to activate it. nd prints a reminder after deploying this asset type.

## Scope rules

| Scope | Target path |
|-------|-------------|
| Global | `~/.claude/output-styles/<name>.md` |
| Project | `.claude/output-styles/<name>.md` |

## Register after deploy

After running `nd deploy`, open your agent's `settings.json` and add the style to the output styles configuration. The exact key depends on your agent; for Claude Code it lives under the `outputStyles` array.

## Related

- [Asset type comparison](../creating-sources.md#asset-types) for a side-by-side overview of all types

## Create an output style

```shell
mkdir -p ~/my-assets/output-styles

cat > ~/my-assets/output-styles/concise.md << 'EOF'
Respond with minimal text. Use bullet points. No explanations unless asked.
Maximum 3 sentences per response.
EOF

nd deploy output-styles/concise
# After deploying, add the style to settings.json to activate it
```
