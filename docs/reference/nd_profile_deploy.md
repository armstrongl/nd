---
title: "nd profile deploy"
description: "Deploy all assets in a profile"
weight: 170
---

Deploy all assets in a profile

```shell {filename="Terminal"}
nd profile deploy <name> [flags]
```

## Examples

```shell {filename="Terminal"}
  # Deploy all assets in a profile
  nd profile deploy my-setup

  # Preview what would be deployed
  nd profile deploy my-setup --dry-run
```

## Options

```text {filename="Flags"}
  -h, --help   help for deploy
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

- [nd profile](nd_profile.md) - Manage deployment profiles

## Guides

- [Profiles and snapshots](../guide/profiles-and-snapshots.md)
