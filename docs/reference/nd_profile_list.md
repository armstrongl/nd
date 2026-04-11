---
title: "nd profile list"
description: "List all profiles"
weight: 180
---

List all profiles

```
nd profile list [flags]
```

## Examples

```
  # List all saved profiles
  nd profile list

  # Output as JSON
  nd profile list --json
```

## Options

```
  -h, --help   help for list
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
