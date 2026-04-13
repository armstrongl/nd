---
title: "nd init"
description: "Initialize nd configuration"
weight: 100
---

Initialize nd configuration

## Synopsis

Interactive walkthrough to set up nd for the first time.

Creates the config directory structure, writes a default config file, and
deploys built-in assets (skills, commands, agents) to your coding agent's
config directory. Use --yes to skip the deploy confirmation prompt.

```shell {filename="Terminal"}
nd init [flags]
```

## Examples

```shell {filename="Terminal"}
  # Interactive setup
  nd init

  # Non-interactive setup (skip prompts)
  nd init --yes
```

## Options

```text {filename="Flags"}
  -h, --help   help for init
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
- [Configuration](../guide/configuration.md)
