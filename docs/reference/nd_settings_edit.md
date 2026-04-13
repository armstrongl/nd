---
title: "nd settings edit"
description: "Open settings in your editor"
weight: 220
---

Open settings in your editor

```shell {filename="Terminal"}
nd settings edit [flags]
```

## Examples

```shell {filename="Terminal"}
  # Open config in your default editor
  nd settings edit
```

## Options

```text {filename="Flags"}
  -h, --help   help for edit
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

- [nd settings](nd_settings.md) - Manage nd settings

## Guides

- [Configuration](../guide/configuration.md)
- [Getting started](../guide/getting-started.md)
- [Troubleshoot](../guide/troubleshooting.md)
- [Hooks](../guide/asset-types/hooks.md)
- [Output styles](../guide/asset-types/output-styles.md)
