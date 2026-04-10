---
title: "nd snapshot save"
description: "Save current deployments as a named snapshot"
weight: 270
---

Save current deployments as a named snapshot

```
nd snapshot save <name> [flags]
```

## Examples

```
  # Save current deployments as a snapshot
  nd snapshot save before-update
```

## Options

```
  -h, --help   help for save
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

- [nd snapshot](nd_snapshot.md) - Manage deployment snapshots

## Guides

- [Profiles and snapshots](../guide/profiles-and-snapshots.md)
- [Getting started](../guide/getting-started.md)
