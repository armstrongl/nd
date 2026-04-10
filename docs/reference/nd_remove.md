---
title: "nd remove"
description: "Remove deployed assets"
weight: 200
---

Remove deployed assets

```
nd remove <asset> [assets...] [flags]
```

## Examples

```
  # Remove a deployed asset
  nd remove skills/greeting

  # Remove multiple assets
  nd remove skills/greeting commands/hello

  # Skip confirmation prompt
  nd remove skills/greeting --yes

  # Preview what would be removed
  nd remove skills/greeting --dry-run
```

## Options

```
  -h, --help   help for remove
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

- [Getting started](../guide/getting-started.md)
- [How nd works](../guide/how-nd-works.md)
- [Profiles and snapshots](../guide/profiles-and-snapshots.md)
