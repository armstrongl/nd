---
title: "nd source"
description: "Manage asset sources"
weight: 280
---

Manage asset sources

## Synopsis

Add, remove, and list asset source directories.

## Examples

```shell {filename="Terminal"}
  nd source add ~/my-assets
  nd source list
  nd source remove my-assets
```

## Options

```text {filename="Flags"}
  -h, --help   help for source
```

## Options inherited from parent commands

```text {filename="Flags"}
      --agent string    target agent (e.g., claude-code, copilot)
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
- [nd source add](nd_source_add.md) - Register a new asset source
- [nd source list](nd_source_list.md) - List registered sources
- [nd source remove](nd_source_remove.md) - Remove a registered source

## Guides

- [Create sources](../guide/creating-sources.md)
