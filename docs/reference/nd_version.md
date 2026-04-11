---
title: "nd version"
description: "Print nd version information"
weight: 360
---

Print nd version information

```shell {filename="Terminal"}
nd version [flags]
```

## Examples

```shell {filename="Terminal"}
  nd version
```

## Options

```text {filename="Flags"}
  -h, --help   help for version
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
