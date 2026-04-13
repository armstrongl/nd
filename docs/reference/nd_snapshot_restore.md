---
title: "nd snapshot restore"
description: "Restore deployments from a snapshot"
weight: 260
---

Restore deployments from a snapshot

```shell {filename="Terminal"}
nd snapshot restore <name> [flags]
```

## Examples

```shell {filename="Terminal"}
  # Restore deployments from a snapshot
  nd snapshot restore before-update

  # Preview what would change
  nd snapshot restore before-update --dry-run
```

## Options

```text {filename="Flags"}
  -h, --help   help for restore
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

- [nd snapshot](nd_snapshot.md) - Manage deployment snapshots

## Guides

- [Profiles and snapshots](../guide/profiles-and-snapshots.md)
