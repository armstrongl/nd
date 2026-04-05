---
title: "nd deploy"
description: "Deploy assets by creating symlinks"
weight: 60
---

Deploy assets by creating symlinks

## Synopsis

Deploy one or more assets by creating symlinks from source to agent config.

Asset references can be:
  name Search all types for matching name
  type/name Search specific type (e.g., skills/greeting)

```
nd deploy <asset> [assets...] [flags]
```

## Examples

```
  # Deploy a single asset
  nd deploy skills/greeting

  # Deploy by name (if unique across types)
  nd deploy greeting

  # Deploy multiple assets at once
  nd deploy skills/greeting commands/hello agents/researcher

  # Filter by type
  nd deploy --type skills greeting

  # Deploy to project scope
  nd deploy skills/greeting --scope project

  # Use relative symlinks
  nd deploy skills/greeting --relative

  # Script-friendly: skip prompts, output JSON
  nd deploy skills/greeting --yes --json
```

## Options

```
      --absolute      use absolute symlinks (overrides config)
  -h, --help          help for deploy
      --relative      use relative symlinks (overrides config)
      --type string   asset type filter (skills, commands, rules, etc.)
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

- [nd](nd.md) - Napoleon Dynamite - coding agent asset manager

## Guides

- [Getting started](../guide/getting-started.md)
- [How nd works](../guide/how-nd-works.md)
