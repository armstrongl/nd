---
title: "nd export marketplace"
description: "Generate a Claude Code marketplace from exported plugins"
weight: 90
---

Generate a Claude Code marketplace from exported plugins

## Synopsis

Generate a marketplace directory structure from one or more previously exported plugins.

Each --plugins path must point to a directory containing a .claude-plugin/plugin.json file.

```
nd export marketplace [flags]
```

## Examples

```
  # Generate marketplace from exported plugins
  nd export marketplace --plugins ./plugin-a,./plugin-b --output ./marketplace
```

## Options

```
      --description string   marketplace description
      --email string         owner email
  -h, --help                 help for marketplace
      --name string          marketplace name (kebab-case)
      --output string        output directory (default ./<name>)
      --overwrite            overwrite existing output directory
      --owner string         marketplace owner name
      --plugins strings      paths to exported plugin directories
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

- [nd export](nd_export.md) - Export assets as a Claude Code plugin
