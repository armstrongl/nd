---
title: "nd doctor"
description: "Check nd configuration and deployment health"
weight: 70
---

Check nd configuration and deployment health

```shell {filename="Terminal"}
nd doctor [flags]
```

## Examples

```shell {filename="Terminal"}
  # Run a full health check
  nd doctor

  # Output as JSON for CI
  nd doctor --json
```

## Options

```text {filename="Flags"}
  -h, --help   help for doctor
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

- [nd](nd.md) - Napoleon Dynamite - coding agent asset manager
- [nd status](nd_status.md) - Show deployment status and health
- [nd sync](nd_sync.md) - Repair symlinks and optionally pull git sources

## Guides

- [Getting started](../guide/getting-started.md)
- [Troubleshoot](../guide/troubleshooting.md)
