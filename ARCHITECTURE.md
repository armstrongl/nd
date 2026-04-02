# Architecture

This document describes the internal architecture of nd for contributors and maintainers.

## Overview

nd is a CLI/TUI tool that manages coding agent assets (skills, agents, commands, output-styles, rules, context files, plugins, hooks) via symlink-based deployment. It is built in Go with [Cobra](https://github.com/spf13/cobra) for the CLI and [Bubble Tea](https://charm.land/bubbletea/) for the TUI.

## Layered Architecture

```text
+---------------------------------------------+
|              cmd/ (CLI layer)                |
|          internal/tui/ (TUI layer)           |
+---------------------------------------------+
| sourcemanager | deploy | profile |   agent   |  <- Service layer
|            doctor | backup | oplog           |  <- Supporting services
+---------------------------------------------+
|  nd  | config | asset | source |    state    |  <- Core types
|          version | output                    |  <- Utilities
+---------------------------------------------+
```

**Core types** define data structures and enums. **Services** implement business logic. **Supporting services** handle diagnostics, backup, and operation logging. **CLI/TUI** handle user interaction and wire services together.

## Core Types (Bottom Layer)

### internal/nd

Core enums and constants shared across the codebase.

- `AssetType` -- 8 asset types: skills, agents, commands, output-styles, rules, context, plugins, hooks
- `Scope` -- `global` (agent-wide) or `project` (repo-specific)
- `SourceType` -- `local` (directory), `git` (repository), or `builtin` (embedded in binary)
- `SymlinkStrategy` -- `absolute` or `relative` symlinks
- `Origin` -- Deployment origin: `manual`, `pinned`, or `profile:<name>`
- Utility functions: `FindProjectRoot()`, `AtomicWrite()`

### internal/config

Configuration types and validation.

- `Config` -- Top-level config: version, default_scope, default_agent, symlink_strategy, sources, agents
- `SourceEntry` -- Source registration: id, type, path, url, alias
- `Config.Validate()` -- Validates config against schema rules
- Config merging: defaults -> global -> project -> CLI flags

### internal/asset

Asset types and indexing.

- `Asset` -- An asset discovered from a source: type, name, path, source ID, metadata
- `Identity` -- Unique tuple: (source_id, asset_type, asset_name)
- `Index` -- In-memory asset index with lookup and search
- `CachedIndex` -- Caching layer over Index for repeated lookups

### internal/source

Source data types.

- `Source` -- A registered source: ID, type, path, URL, alias, manifest
- `Manifest` -- Explicit `nd-source.yaml` file defining custom asset paths and exclusions (overrides convention scanning)

### internal/state

Deployment state persistence.

- `Store` -- Load/Save deployment state to YAML files
- `DeploymentState` -- Tracks all active deployments with health status
- File locking for concurrent access safety

## Service Layer (Middle)

### internal/sourcemanager

Source lifecycle management.

- `SourceManager` -- Config loading, source registration, asset scanning, git syncing
- `AddLocal()` / `AddGit()` -- Register new sources
- `Remove()` -- Unregister sources (with deployed asset handling)
- `ScanSource()` -- Asset discovery: manifest (`nd-source.yaml`) overrides convention; convention used when no manifest present
- Git operations: clone, pull (--ff-only)

### internal/deploy

Symlink deployment engine.

- `Engine` -- Deploy, remove, health check, repair, bulk operations
- `Deploy()` / `DeployBulk()` -- Create symlinks from agent config dir to source assets
- `Remove()` / `RemoveBulk()` -- Delete managed symlinks
- `SetOrigin()` -- Update deployment origin (manual, pinned, profile)

### internal/profile

Profile and snapshot management.

- `Profile` -- Named collection of assets
- `Snapshot` -- Point-in-time deployment state record
- `Store` -- CRUD for profiles and snapshots (YAML files)
- `Manager` -- Orchestrates `Switch()`, `DeployProfile()`, `Restore()`
- Switch diff: calculates which assets to add/remove/keep

### internal/agent

Agent detection and registry.

- `Registry` -- Detect installed coding agents, lookup by name, select default
- `Agent` -- Agent metadata: name, global_dir, project_dir, detected, in_path
- Hardcoded default: `claude-code` (~/.claude)
- Testability: `SetLookPath()` / `SetStat()` for injecting stubs

## CLI Layer (Top)

### cmd/

Cobra commands and application wiring.

- `App` -- Central struct with lazily initialized services
- `NewRootCmd(app *App)` -- Builds the root command with 8 global flags
- One file per command group: `deploy.go`, `remove.go`, `source.go`, `profile.go`, `snapshot.go`, etc.
- `helpers.go` -- Shared utilities: `confirm()`, `promptChoice()`, `isTerminal()`, `extractChoiceNames()`

### internal/tui/

Bubble Tea v2 dashboard.

- `app/` -- Main TUI application
- `components/` -- Reusable UI components (tables, pickers, dialogs)
- Dashboard-centric design with tabbed asset views

## Data Flow Example

`nd deploy skills/greeting`:

```text
cmd/deploy.go
  -> app.SourceManager().Scan()     -- discover assets from all sources
  -> asset.Index.Lookup()           -- find "skills/greeting" in index
  -> app.DeployEngine().Deploy()    -- create symlink
  -> state.Save()                   -- persist deployment record
  -> print confirmation
```

## Key Patterns

- **Atomic writes** -- Config and state files are written atomically (write to temp, rename) to prevent corruption
- **Config merging** -- Defaults -> global config -> project config -> CLI flags, each layer can override
- **Convention-based scanning** -- Source directories named `skills/`, `agents/`, etc. are auto-discovered
- **Test doubles via function injection** -- `agent.SetLookPath()`, `agent.SetStat()` allow injecting stubs without interfaces
- **Lazy service initialization** -- `App` struct initializes services on first access, not at startup
- **Origin tracking** -- Each deployment records its origin (manual, pinned, profile:X) for smart profile switching

## Testing Strategy

- **TDD workflow** -- Red/green/refactor for all business logic
- **Function injection** -- OS-level operations stubbed via injected functions
- **Integration tests** -- `tests/integration/` for end-to-end scenarios
- **Coverage targets** -- 80%+ for business logic, lower acceptable for CLI/TUI interactive paths
