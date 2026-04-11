---
title: "nd profile switch"
description: "Switch from current profile to another"
weight: 190
---

Switch from current profile to another

```
nd profile switch <name> [flags]
```

## Examples

```
  # Switch to a different profile
  nd profile switch my-setup

  # Switch without confirmation
  nd profile switch my-setup --yes
```

## Options

```
  -h, --help   help for switch
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

- [nd profile](nd_profile.md) - Manage deployment profiles

## Guides

- [Profiles and snapshots](../guide/profiles-and-snapshots.md)
