---
title: "nd list"
description: "List available assets from all sources"
weight: 110
---

List available assets from all sources

```
nd list [flags]
```

## Examples

```
  # List all available assets
  nd list

  # Filter by type
  nd list --type skills

  # Filter by source
  nd list --source my-assets

  # Filter by name pattern
  nd list --pattern greeting

  # Output as JSON for scripting
  nd list --json
```

## Options

```
  -h, --help             help for list
      --pattern string   filter by name pattern
      --source string    filter by source ID
      --type string      filter by asset type
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
- [Create sources](../guide/creating-sources.md)
- [Skills](../guide/asset-types/skills.md)
- [Agents](../guide/asset-types/agents.md)
- [Commands](../guide/asset-types/commands.md)
- [Rules](../guide/asset-types/rules.md)
