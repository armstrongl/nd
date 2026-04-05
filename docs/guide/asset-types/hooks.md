---
title: "Hooks"
description: "Load when modifying hook scanning, hook deployment, or settings.json hook registration."
lastValidated: "2026-04-04"
maxAgeDays: 90
weight: 80
paths:
  - "internal/asset/scanner.go"
  - "internal/deploy/**"
tags:
  - hooks
  - assets
  - deployment
---

Hooks are directory assets that define event-driven automation triggered by agent lifecycle events, deployed as symlinked directories and activated via manual `settings.json` registration.

## Directory layout

```text
hooks/
└── pre-tool-lint/
    ├── hook.json
    └── run.sh
```

## File format

Each hook is a directory containing a `hook.json` configuration file and one or more executable scripts.

```json
{
  "event": "PreToolUse",
  "description": "Run linter before tool use"
}
```

The `event` field names the agent lifecycle event that triggers the hook. Script files in the directory are executed when the event fires.

## Deploy behavior

nd symlinks the hook directory into the target scope directory. After deployment, you must manually register the hook in your agent's `settings.json` to activate it. nd prints a reminder after deploying this asset type.

## Scope rules

| Scope | Target path |
|-------|-------------|
| Global | `~/.claude/hooks/<name>/` |
| Project | `.claude/hooks/<name>/` |

## Register after deploy

After running `nd deploy`, open your agent's `settings.json` and add the hook to the hooks configuration. For Claude Code, hooks are registered under the `hooks` key, keyed by event name.

## Related

- [Asset type comparison](../creating-sources.md#asset-types) for a side-by-side overview of all types

## Create a hook

```shell
mkdir -p ~/my-assets/hooks/lint-check

cat > ~/my-assets/hooks/lint-check/hook.json << 'EOF'
{
  "event": "PreToolUse",
  "description": "Run linter before tool use"
}
EOF

cat > ~/my-assets/hooks/lint-check/run.sh << 'EOF'
#!/usr/bin/env sh
set -e
npx eslint . --quiet
EOF

chmod +x ~/my-assets/hooks/lint-check/run.sh

nd deploy hooks/lint-check
# After deploying, register the hook in settings.json to activate it
```
