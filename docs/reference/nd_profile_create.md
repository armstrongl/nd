---
title: "nd profile create"
weight: 150
---


Create a new profile

```
nd profile create <name> [flags]
```

## Options

```
      --assets string        comma-separated list of assets (type/name)
      --description string   profile description
      --from-current         create profile from current deployments
  -h, --help                 help for create
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

## SEE ALSO

* [nd profile](nd_profile.md)	 - Manage deployment profiles

