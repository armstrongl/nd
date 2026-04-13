---
title: "nd source remove"
description: "Remove a registered source"
weight: 310
---

Remove a registered source

```shell {filename="Terminal"}
nd source remove <source-id> [flags]
```

## Examples

```shell {filename="Terminal"}
  # Remove a source by ID
  nd source remove my-assets

  # Skip confirmation prompt
  nd source remove my-assets --yes
```

## Options

```text {filename="Flags"}
  -h, --help   help for remove
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
