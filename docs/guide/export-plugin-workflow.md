---
title: "Export a plugin or marketplace"
description: "Load when modifying the export workflow: plugin packaging, marketplace generation, or export flags."
lastValidated: "2026-07-05"
maxAgeDays: 90
weight: 55
paths:
  - "cmd/export.go"
  - "internal/export/**"
tags:
  - export
  - plugins
  - marketplace
---

Most assets deploy as symlinks, but plugins are different: you package them into a self-contained directory and hand them off. The export workflow has two phases. First, [`nd export`](../reference/nd_export.md) bundles selected assets into a Claude Code plugin. Second, [`nd export marketplace`](../reference/nd_export_marketplace.md) collects one or more exported plugins into a marketplace others can install from.

For how plugins fit alongside the other asset types, see [Plugins](asset-types/plugins.md).

## Phase 1: export a plugin

`nd export` copies the assets you name into a new plugin directory with a `.claude-plugin/plugin.json` manifest. The source assets stay where they are; export produces a distributable copy.

```shell {filename="Terminal"}
# Export two assets as a Claude Code plugin
nd export --assets skills/greeting,commands/hello --output ./my-plugin
```

Name the assets in `type/name` format. Repeat `--assets` or comma-separate the values to include several. Plugins themselves cannot be exported; export a plugin's individual assets instead.

Add metadata for the manifest with the remaining flags:

```shell {filename="Terminal"}
nd export \
  --name my-toolbox \
  --version 1.2.0 \
  --author "Ada Lovelace" \
  --email ada@example.com \
  --license MIT \
  --assets skills/greeting,commands/hello \
  --output ./my-toolbox
```

### Export flags

| Flag | Description |
|------|-------------|
| `--name` | Plugin name in kebab-case (required in non-interactive mode) |
| `--assets` | Assets to include, `type/name` format, comma-separated (required in non-interactive mode) |
| `--description` | Plugin description written to the manifest |
| `--version` | Plugin version (default `1.0.0`) |
| `--author` | Author name |
| `--email` | Author email |
| `--license` | SPDX license identifier (for example, `MIT`) |
| `--source` | Resolve assets only from this source ID |
| `--output` | Output directory (default `./<name>`) |
| `--overwrite` | Overwrite an existing output directory |

When you omit `--output`, nd writes to `./<name>`. Preview the plan without writing anything by adding `--dry-run`.

Exporting into a directory that already exists fails unless you add `--overwrite`, which replaces the previous contents:

```shell {filename="Terminal"}
# Replace a previous export in place
nd export \
  --name my-toolbox \
  --assets skills/greeting,commands/hello \
  --output ./my-toolbox \
  --overwrite
```

### Interactive export

Run `nd export` in a terminal without `--name` or `--assets` and nd walks you through it: a multi-select list of exportable assets, then a form for the name, description, version, author, email, and license, then a confirmation. Provide any flags you already know as defaults, and nd only prompts for the rest.

```shell {filename="Terminal"}
nd export
```

The name must be kebab-case; the form rejects other formats. Outside a terminal (a pipe or CI job), nd cannot prompt, so `--name` and `--assets` are required.

## Phase 2: generate a marketplace

Once you have one or more exported plugin directories, bundle them into a marketplace. `nd export marketplace` copies each plugin into the output directory and writes a `.claude-plugin/marketplace.json` manifest that lists them.

```shell {filename="Terminal"}
# Generate a marketplace from exported plugins
nd export marketplace \
  --name my-marketplace \
  --owner "Ada Lovelace" \
  --description "Handy plugins for Claude Code" \
  --plugins ./plugin-a,./plugin-b \
  --output ./marketplace
```

Each `--plugins` path must point to a directory that already contains a `.claude-plugin/plugin.json` file: an output directory from phase 1. nd reads each plugin's name, version, description, and author from that manifest.

### Marketplace flags

| Flag | Description |
|------|-------------|
| `--name` | Marketplace name in kebab-case (required) |
| `--owner` | Marketplace owner name (required) |
| `--plugins` | Paths to exported plugin directories (required) |
| `--description` | Marketplace description |
| `--email` | Owner email |
| `--output` | Output directory (default `./<name>`) |
| `--overwrite` | Overwrite an existing output directory |

A missing `--name`, `--owner`, or `--plugins` stops the command with an error naming the missing flag.

## Output layout

A single exported plugin looks like this:

```text {filename="Plugin layout"}
my-plugin/
├── .claude-plugin/
│   └── plugin.json          # manifest: name, version, description, author
├── skills/
│   └── greeting/
│       └── SKILL.md
└── commands/
    └── hello.md
```

A generated marketplace copies each plugin under `plugins/` and adds a marketplace manifest:

```text {filename="Marketplace layout"}
marketplace/
├── .claude-plugin/
│   └── marketplace.json     # lists each plugin with a ./plugins/<name> path
└── plugins/
    ├── plugin-a/
    │   └── .claude-plugin/
    │       └── plugin.json
    └── plugin-b/
        └── .claude-plugin/
            └── plugin.json
```

The marketplace manifest references each plugin by a relative `./plugins/<name>` path, so the whole directory is self-contained and can be version-controlled or published as one unit.

## Distribute and install

Both an exported plugin and a generated marketplace are plain directories. Commit them to git, publish them, or share them directly. Install a plugin through your agent's plugin mechanism rather than [`nd deploy`](../reference/nd_deploy.md): plugins bypass the symlink deployment system entirely.

## Next steps

- **[Plugins](asset-types/plugins.md):** How plugin sources are authored and discovered
- **[Create asset sources](creating-sources.md):** Structure the sources that supply exportable assets
- **[`nd export` reference](../reference/nd_export.md):** Full flag reference for the export command
- **[`nd export marketplace` reference](../reference/nd_export_marketplace.md):** Full flag reference for marketplace generation
- **[User guide](user-guide.md):** Core workflows for deploying and managing assets
