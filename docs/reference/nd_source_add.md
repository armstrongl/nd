---
title: "nd source add"
description: "Register a new asset source"
weight: 290
---

Register a new asset source

```
nd source add <path|url> [flags]
```

## Examples

```
  # Add a local directory
  nd source add ~/my-assets

  # Add with a custom alias
  nd source add ~/my-assets --alias my-stuff

  # Add a GitHub repository (shorthand)
  nd source add owner/repo

  # Add via HTTPS
  nd source add https://github.com/owner/repo.git

  # Add via SSH
  nd source add git@github.com:owner/repo.git
```

## Options

```
      --alias string   human-readable alias for the source
  -h, --help           help for add
```

## Options inherited from parent commands

```
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

- [nd source](nd_source.md) - Manage asset sources

## Guides

- [Create sources](../guide/creating-sources.md)
- [Getting started](../guide/getting-started.md)
