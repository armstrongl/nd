---
title: "nd source list"
description: "List registered sources"
weight: 300
---

List registered sources

```shell {filename="Terminal"}
nd source list [flags]
```

## Examples

```shell {filename="Terminal"}
  # List all registered sources
  nd source list

  # Output as JSON
  nd source list --json
```

## Options

```text {filename="Flags"}
  -h, --help   help for list
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

- [nd source](nd_source.md) - Manage asset sources

## Guides

- [Create sources](../guide/creating-sources.md)
