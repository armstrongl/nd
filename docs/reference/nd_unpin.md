---
title: "nd unpin"
description: "Unpin an asset, allowing profile switches to manage it"
weight: 350
---

Unpin an asset, allowing profile switches to manage it

```
nd unpin <asset> [flags]
```

## Examples

```
  # Unpin an asset
  nd unpin skills/greeting
```

## Options

```
  -h, --help   help for unpin
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

## Guides

- [Profiles and snapshots](../guide/profiles-and-snapshots.md)
