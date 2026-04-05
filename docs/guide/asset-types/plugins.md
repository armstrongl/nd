---
title: "Plugins"
description: "Load when modifying plugin scanning, export workflow, or plugin.json manifest handling."
lastValidated: "2026-04-04"
maxAgeDays: 90
weight: 70
paths:
  - "internal/sourcemanager/scanner.go"
  - "internal/export/**"
  - "cmd/export.go"
tags:
  - plugins
  - assets
  - export
---

Plugins are directory assets that bundle multiple nd assets into a Claude Code plugin package, distributed and installed via the `nd export` workflow rather than symlink deployment.

## Directory layout

```text
plugins/
└── my-toolbox/
    ├── .claude-plugin/
    │   └── plugin.json
    ├── skills/
    │   └── helper/
    │       └── SKILL.md
    └── commands/
        └── run-all.md
```

## File format

A plugin directory contains a `.claude-plugin/` subdirectory with a `plugin.json` manifest inside it, plus asset subdirectories that follow the standard nd structure.

```json
{
  "name": "my-toolbox",
  "version": "1.0.0",
  "description": "A collection of useful agent tools"
}
```

Asset subdirectories inside the plugin (e.g., `skills/`, `commands/`) follow the same authoring conventions as their standalone counterparts.

## Deploy behavior

Plugins are **not deployable via `nd deploy`**. Instead, use `nd export` to package the assets you want to include, then install the exported package through your agent's plugin installation mechanism.

```shell
nd export --assets skills/greeting,commands/hello --output ./my-plugin
```

The exported directory is a self-contained plugin that can be handed off, version-controlled, or published independently.

## Scope rules

Not applicable. Plugins bypass the symlink deployment system entirely and have no global or project scope path.

## Related

- [Asset type comparison](../creating-sources.md#asset-types) for a side-by-side overview of all types

## Create a plugin

```shell
mkdir -p ~/my-assets/plugins/my-toolbox/.claude-plugin
mkdir -p ~/my-assets/plugins/my-toolbox/skills/greeting

cat > ~/my-assets/plugins/my-toolbox/.claude-plugin/plugin.json << 'EOF'
{
  "name": "my-toolbox",
  "version": "1.0.0",
  "description": "A collection of useful agent tools"
}
EOF

cat > ~/my-assets/plugins/my-toolbox/skills/greeting/SKILL.md << 'EOF'
Greet the user by name with a short, friendly message.
EOF

# Package the plugin for distribution
nd export --assets skills/greeting --output ~/my-assets/plugins/my-toolbox
```
