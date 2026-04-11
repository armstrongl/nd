---
title: "nd sync"
description: "Repair symlinks and optionally pull git sources"
weight: 330
---

Repair symlinks and optionally pull git sources

```
nd sync [flags]
```

## Examples

```
  # Repair all broken symlinks
  nd sync

  # Pull and repair a specific git source
  nd sync --source my-git-source

  # Preview what would be repaired
  nd sync --dry-run
```

## Options

```
  -h, --help            help for sync
      --source string   sync a specific git source
```

## Options inherited from parent commands

```
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
- [nd status](nd_status.md) - Show deployment status and health

## Guides

- [Getting started](../guide/getting-started.md)
- [Create sources](../guide/creating-sources.md)
- [Troubleshoot](../guide/troubleshooting.md)
