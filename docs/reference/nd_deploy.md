---
title: "nd deploy"
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
