# Profiles & Snapshots

Profiles and snapshots help you manage multiple sets of agent assets and switch between them.

## What Are Profiles?

A **profile** is a named collection of assets -- like browser profiles for your coding agent. You might have a "work" profile with enterprise-focused skills and a "personal" profile with hobby project tools.

## Creating Profiles

### From an Asset List

Specify exactly which assets belong in the profile:

```bash
nd profile create work --assets skills/enterprise-auth,skills/jira-integration,agents/code-reviewer
```

### From Current Deployments

Capture whatever is currently deployed:

```bash
nd profile create work --from-current
```

Add a description:

```bash
nd profile create work --from-current --description "Enterprise development setup"
```

## Building Profiles Incrementally

Add assets to an existing profile one at a time:

```bash
nd profile add-asset work skills/new-skill
nd profile add-asset work commands/deploy-staging
```

## Listing Profiles

```bash
nd profile list
```

The active profile is marked with `*`.

## Deploying a Profile

Deploy all assets from a profile:

```bash
nd profile deploy work
```

This resolves each asset reference from your registered sources and creates symlinks. Missing assets are reported as warnings.

Preview first:

```bash
nd profile deploy work --dry-run
```

## Switching Profiles

Switch from the current active profile to another:

```bash
nd profile switch personal
```

This shows a diff preview of what will change:
- **Remove:** Assets from the current profile (origin: `profile:<current>`)
- **Deploy:** Assets from the new profile
- **Keep:** Pinned and manually deployed assets

Before switching, nd automatically saves a snapshot (safety net). After confirming, it removes old profile assets and deploys new ones.

## Deleting Profiles

```bash
nd profile delete work
```

This removes the profile definition but does **not** remove any currently deployed assets. Run `nd profile delete` with no arguments to get an interactive picker.

## Pinning Assets

**Pinned assets persist across profile switches.** Use this for assets you always want available regardless of which profile is active.

```bash
# Pin an asset
nd pin skills/greeting

# Unpin (returns to "manual" origin)
nd unpin skills/greeting
```

When switching profiles, nd skips pinned assets entirely -- they are neither removed nor redeployed.

## Snapshots

A **snapshot** is a point-in-time record of all current deployments. Think of it as a bookmark you can return to.

### Save a Snapshot

```bash
nd snapshot save before-experiment
```

### List Snapshots

```bash
nd snapshot list
```

Both user-created and auto-created snapshots are shown. Auto-snapshots (created before destructive operations) are tagged with `(auto)`.

### Restore a Snapshot

```bash
nd snapshot restore before-experiment
```

This removes all current deployments and redeploys the snapshot's assets. nd saves an auto-snapshot before restoring (so you can undo the restore).

Run `nd snapshot restore` with no arguments to get an interactive picker.

### Delete a Snapshot

```bash
nd snapshot delete old-snapshot
```

### Auto-Snapshots

nd automatically saves snapshots before destructive operations like profile switching and snapshot restoring. The 5 most recent auto-snapshots are retained; older ones are cleaned up.

## Workflow Example

Here is a complete workflow using profiles, pinning, and snapshots:

```bash
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
