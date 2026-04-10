---
title: "nd status"
description: "Show deployment status and health"
weight: 320
---

Show deployment status and health

```
nd status [flags]
```

## Examples

```
  # Show all deployed assets and their health
  nd status

  # Output as JSON for scripting
  nd status --json

  # Show project-scope deployments
  nd status --scope project
```

## Options

```
  -h, --help   help for status
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
- [nd doctor](nd_doctor.md) - Check nd configuration and deployment health
- [nd deploy](nd_deploy.md) - Deploy assets by creating symlinks

## Guides

- [Getting started](../guide/getting-started.md)
- [Troubleshoot](../guide/troubleshooting.md)
