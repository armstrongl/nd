---
title: "nd uninstall"
description: "Remove all nd-managed symlinks and optionally config"
weight: 340
---

Remove all nd-managed symlinks and optionally config

```shell {filename="Terminal"}
nd uninstall [flags]
```

## Examples

```shell {filename="Terminal"}
  # Remove all nd-managed symlinks
  nd uninstall

  # Skip confirmation prompt
  nd uninstall --yes
```

## Options

```text {filename="Flags"}
  -h, --help   help for uninstall
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

## Guides

- [Getting started](../guide/getting-started.md)
- [Troubleshoot](../guide/troubleshooting.md)
