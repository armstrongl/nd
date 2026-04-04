---
title: "Profiles and snapshots"
description: "Load when modifying profile CRUD, snapshot save/restore, profile switching, or pinning logic."
lastValidated: "2026-03-28"
maxAgeDays: 90
weight: 60
paths:
  - "internal/profile/**"
  - "cmd/profile.go"
  - "cmd/snapshot.go"
  - "cmd/pin.go"
  - "cmd/unpin.go"
tags:
  - profiles
  - snapshots
  - pinning
---

Use profiles and snapshots to manage multiple sets of agent assets and switch between them.

## What are profiles?

A **profile** is a named collection of assets: like browser profiles for your coding agent. For example, a "work" profile holds enterprise-focused skills and a "personal" profile holds hobby project tools.

## Create profiles

### From an asset list

Specify exactly which assets belong in the profile:

```shell
nd profile create work --assets skills/enterprise-auth,skills/jira-integration,agents/code-reviewer
```

### From current deployments

Capture whatever is currently deployed:

```shell
nd profile create work --from-current
```

Add a description:

```shell
nd profile create work --from-current --description "Enterprise development setup"
```

## Build profiles incrementally

Add assets to an existing profile one at a time:

```shell
nd profile add-asset work skills/new-skill
nd profile add-asset work commands/deploy-staging
```

## List profiles

```shell
nd profile list
```

nd marks the active profile with `*`.

## Deploy a profile

Deploy all assets from a profile:

```shell
nd profile deploy work
```

This resolves each asset reference from your registered sources and creates symlinks. nd reports missing assets as warnings.

Preview first:

```shell
nd profile deploy work --dry-run
```

## Switch profiles

Switch from the current active profile to another:

```shell
nd profile switch personal
```

This shows a diff preview of what changes:

- **Remove:** Assets from the current profile (origin: `profile:<current>`)
- **Deploy:** Assets from the new profile
- **Keep:** Pinned and manually deployed assets

Before switching, nd automatically saves a snapshot (safety net). After confirming, it removes old profile assets and deploys new ones.

## Delete profiles

```shell
nd profile delete work
```

This removes the profile definition but does **not** remove any currently deployed assets. Run `nd profile delete` with no arguments to get an interactive picker.

## Pin assets

**Pinned assets persist across profile switches.** Use this for assets you always want available regardless of which profile is active.

```shell
# Pin an asset
nd pin skills/greeting

# Unpin (returns to "manual" origin)
nd unpin skills/greeting
```

When switching profiles, nd skips pinned assets entirely: nd neither removes nor redeploys them.

## Snapshots

A **snapshot** is a point-in-time record of all current deployments. Think of it as a bookmark you can return to.

### Save a snapshot

```shell
nd snapshot save before-experiment
```

### List snapshots

```shell
nd snapshot list
```

The list shows both user-created and auto-created snapshots. nd tags auto-snapshots (created before destructive operations) with `(auto)`.

### Restore a snapshot

```shell
nd snapshot restore before-experiment
```

This removes all current deployments and redeploys the snapshot's assets. nd saves an auto-snapshot before restoring (so you can undo the restore).

Run `nd snapshot restore` with no arguments to get an interactive picker.

### Delete a snapshot

```shell
nd snapshot delete old-snapshot
```

### Auto-snapshots

nd automatically saves snapshots before destructive operations like profile switching and snapshot restoring. nd retains the 5 most recent auto-snapshots and cleans up older ones.

## Workflow example

Here is a complete workflow using profiles, pinning, and snapshots:

```shell
# Create two profiles
nd profile create work --assets skills/jira,skills/enterprise-auth,agents/reviewer
nd profile create personal --assets skills/blog-writer,skills/recipe-helper

# Pin assets you always want
nd pin skills/greeting
nd pin rules/no-emojis

# Deploy the work profile
nd profile deploy work

# Later, switch to personal
nd profile switch personal
# Shows diff, confirms, switches

# Try something experimental
nd snapshot save before-experiment
nd deploy skills/experimental-thing

# Didn't work out -- restore
nd snapshot restore before-experiment

# Back to work
nd profile switch work
```

Pinned assets (`skills/greeting`, `rules/no-emojis`) persist through every switch.
