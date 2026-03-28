---
title: Command Reference
weight: 2
sidebar:
  open: true
---

Complete reference for every nd command, including flags, options, and usage examples.

## Core commands

{{< cards >}}
  {{< card link="nd" title="nd" subtitle="Root command and global flags" >}}
  {{< card link="nd_init" title="nd init" subtitle="Initialize nd configuration" >}}
  {{< card link="nd_deploy" title="nd deploy" subtitle="Deploy assets by creating symlinks" >}}
  {{< card link="nd_remove" title="nd remove" subtitle="Remove deployed assets" >}}
  {{< card link="nd_list" title="nd list" subtitle="List available assets" >}}
  {{< card link="nd_status" title="nd status" subtitle="Show deployment status" >}}
  {{< card link="nd_sync" title="nd sync" subtitle="Sync sources and repair symlinks" >}}
  {{< card link="nd_doctor" title="nd doctor" subtitle="Run health checks" >}}
  {{< card link="nd_pin" title="nd pin" subtitle="Pin assets across profile switches" >}}
  {{< card link="nd_unpin" title="nd unpin" subtitle="Unpin a pinned asset" >}}
{{< /cards >}}

## Profile commands

{{< cards >}}
  {{< card link="nd_profile" title="nd profile" subtitle="Manage named asset profiles" >}}
  {{< card link="nd_profile_create" title="nd profile create" subtitle="Create a new profile" >}}
  {{< card link="nd_profile_list" title="nd profile list" subtitle="List all profiles" >}}
  {{< card link="nd_profile_deploy" title="nd profile deploy" subtitle="Deploy a profile's assets" >}}
  {{< card link="nd_profile_switch" title="nd profile switch" subtitle="Switch active profile" >}}
  {{< card link="nd_profile_add-asset" title="nd profile add-asset" subtitle="Add asset to a profile" >}}
  {{< card link="nd_profile_delete" title="nd profile delete" subtitle="Delete a profile" >}}
{{< /cards >}}

## Snapshot commands

{{< cards >}}
  {{< card link="nd_snapshot" title="nd snapshot" subtitle="Manage deployment snapshots" >}}
  {{< card link="nd_snapshot_save" title="nd snapshot save" subtitle="Save current state" >}}
  {{< card link="nd_snapshot_list" title="nd snapshot list" subtitle="List saved snapshots" >}}
  {{< card link="nd_snapshot_restore" title="nd snapshot restore" subtitle="Restore a snapshot" >}}
  {{< card link="nd_snapshot_delete" title="nd snapshot delete" subtitle="Delete a snapshot" >}}
{{< /cards >}}

## Source commands

{{< cards >}}
  {{< card link="nd_source" title="nd source" subtitle="Manage asset sources" >}}
  {{< card link="nd_source_add" title="nd source add" subtitle="Add a local or git source" >}}
  {{< card link="nd_source_list" title="nd source list" subtitle="List registered sources" >}}
  {{< card link="nd_source_remove" title="nd source remove" subtitle="Remove a source" >}}
{{< /cards >}}

## Other commands

{{< cards >}}
  {{< card link="nd_export" title="nd export" subtitle="Export assets as a plugin" >}}
  {{< card link="nd_export_marketplace" title="nd export marketplace" subtitle="Generate marketplace listing" >}}
  {{< card link="nd_settings" title="nd settings" subtitle="Manage nd settings" >}}
  {{< card link="nd_settings_edit" title="nd settings edit" subtitle="Open config in editor" >}}
  {{< card link="nd_version" title="nd version" subtitle="Print nd version" >}}
  {{< card link="nd_uninstall" title="nd uninstall" subtitle="Remove all nd symlinks" >}}
  {{< card link="nd_completion" title="nd completion" subtitle="Generate shell completion scripts" >}}
  {{< card link="nd_completion_bash" title="nd completion bash" subtitle="Bash completion script" >}}
  {{< card link="nd_completion_fish" title="nd completion fish" subtitle="Fish completion script" >}}
  {{< card link="nd_completion_zsh" title="nd completion zsh" subtitle="Zsh completion script" >}}
{{< /cards >}}
