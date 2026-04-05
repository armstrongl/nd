---
title: "nd pin"
description: "Pin an asset to prevent profile switches from removing it"
weight: 120
---

Pin an asset to prevent profile switches from removing it

```
nd pin <asset> [flags]
```

## Examples

```
  # Pin an asset to survive profile switches
  nd pin skills/greeting

  # Pin using just the name
  nd pin greeting
```

## Options

```
  -h, --help   help for pin
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
