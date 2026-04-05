---
title: "nd export"
description: "Export assets as a Claude Code plugin"
weight: 80
---

Export assets as a Claude Code plugin

## Synopsis

Export one or more nd-managed assets into the Claude Code plugin format.

Assets are specified with --assets in type/name format (e.g., skills/greeting).
Multiple assets can be comma-separated or the flag repeated.

```
nd export [flags]
```

## Examples

```
  # Export assets as a Claude Code plugin
  nd export --assets skills/greeting,commands/hello --output ./my-plugin

  # Generate a marketplace from plugins
  nd export marketplace --plugins ./plugin-a,./plugin-b --output ./marketplace
```

## Options

```
      --assets strings       assets to export (type/name format, comma-separated)
      --author string        author name
      --description string   plugin description
      --email string         author email
  -h, --help                 help for export
      --license string       SPDX license identifier
      --name string          plugin name (kebab-case)
      --output string        output directory (default ./<name>)
      --overwrite            overwrite existing output directory
      --source string        export only from this source
      --version string       plugin version (default "1.0.0")
```

## Options inherited from parent commands

```
      --config string   path to config file (default "~/.config/nd/config.yaml")
      --dry-run         show what would happen without making changes
      --json            output in JSON format
      --no-color        disable colored output
  -q, --quiet           suppress non-error output
  -s, --scope string    deployment scope (global|project) (default "global")
  -v, --verbose         verbose output to stderr
  -y, --yes             skip confirmation prompts
```

## Related

- [nd](nd.md) - Napoleon Dynamite - coding agent asset manager
- [nd export marketplace](nd_export_marketplace.md) - Generate a Claude Code marketplace from exported plugins

## Guides

- [Getting started](../guide/getting-started.md)
- [Create sources](../guide/creating-sources.md)
