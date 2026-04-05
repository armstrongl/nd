---
title: "nd profile create"
description: "Create a new profile"
weight: 150
---

Create a new profile

```
nd profile create <name> [flags]
```

## Examples

```
  # Create a profile from current deployments
  nd profile create my-setup

  # Create a profile for project scope
  nd profile create my-setup --scope project
```

## Options

```
      --assets string        comma-separated list of assets (type/name)
      --description string   profile description
      --from-current         create profile from current deployments
  -h, --help                 help for create
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

- [nd profile](nd_profile.md) - Manage deployment profiles

## Guides

- [Profiles and snapshots](../guide/profiles-and-snapshots.md)
