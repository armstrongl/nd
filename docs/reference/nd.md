---
title: "nd"
description: "Napoleon Dynamite - coding agent asset manager"
weight: 1
---

Napoleon Dynamite - coding agent asset manager

## Synopsis

nd manages coding agent assets (skills, commands, rules, etc.) via symlink deployment.

```
nd [flags]
```

## Examples

```
  # Deploy an asset
  nd deploy skills/greeting

  # List available assets
  nd list --type skills

  # Check deployment health
  nd doctor

  # Open the TUI
  nd
```

## Options

```
      --config string   path to config file (default "~/.config/nd/config.yaml")
      --dry-run         show what would happen without making changes
  -h, --help            help for nd
      --json            output in JSON format
      --no-color        disable colored output
  -q, --quiet           suppress non-error output
  -s, --scope string    deployment scope (global|project) (default "global")
  -v, --verbose         verbose output to stderr
  -y, --yes             skip confirmation prompts
```

## Related

- [nd deploy](nd_deploy.md) - Deploy assets by creating symlinks
- [nd doctor](nd_doctor.md) - Check nd configuration and deployment health
- [nd export](nd_export.md) - Export assets as a Claude Code plugin
- [nd init](nd_init.md) - Initialize nd configuration
- [nd list](nd_list.md) - List available assets from all sources
- [nd pin](nd_pin.md) - Pin an asset to prevent profile switches from removing it
- [nd profile](nd_profile.md) - Manage deployment profiles
- [nd remove](nd_remove.md) - Remove deployed assets
- [nd settings](nd_settings.md) - Manage nd settings
- [nd snapshot](nd_snapshot.md) - Manage deployment snapshots
- [nd source](nd_source.md) - Manage asset sources
- [nd status](nd_status.md) - Show deployment status and health
- [nd sync](nd_sync.md) - Repair symlinks and optionally pull git sources
- [nd uninstall](nd_uninstall.md) - Remove all nd-managed symlinks and optionally config
- [nd unpin](nd_unpin.md) - Unpin an asset, allowing profile switches to manage it
- [nd version](nd_version.md) - Print nd version information
