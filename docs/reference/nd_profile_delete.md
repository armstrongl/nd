---
title: "nd profile delete"
description: "Delete a profile"
weight: 160
---

Delete a profile

```
nd profile delete <name> [flags]
```

## Examples

```
  # Delete a profile
  nd profile delete my-setup

  # Delete without confirmation
  nd profile delete my-setup --yes
```

## Options

```
  -h, --help   help for delete
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
