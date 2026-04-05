---
title: "Configuration"
description: "Load when modifying config loading, merging, validation, or debugging config file issues."
lastValidated: "2026-04-05"
maxAgeDays: 90
weight: 40
paths:
  - "internal/config/**"
  - "cmd/settings.go"
  - "cmd/init.go"
tags:
  - config
  - settings
  - yaml
---

nd uses YAML configuration files with a layered merging system.

## Config file locations

| Location | Path | Purpose |
|----------|------|---------|
| Global | `~/.config/nd/config.yaml` | User-wide settings and sources |
| Project | `.nd/config.yaml` | Project-specific overrides |
| CLI flag | `--config <path>` | One-time override |

`nd init` creates the global config. Project-level config is optional.

## Data directories

nd stores all data under `~/.config/nd/`:

| Directory | Purpose |
|-----------|---------|
| `config.yaml` | Main configuration file |
| `sources/` | Cloned git sources |
| `profiles/` | Named profile definitions |
| `snapshots/` | User and auto snapshots |
| `state/` | Deployment state (`deployments.yaml`) |
| `backups/` | Context file conflict backups |
| `logs/` | Operation log (`operations.log`) |

## Full annotated example

```yaml
# Schema version (always 1)
version: 1

# Default deployment scope: "global" or "project"
# Global deploys to ~/.claude/, project deploys to .claude/
default_scope: global

# Default coding agent to target
default_agent: claude-code

# Symlink strategy: "absolute" or "relative"
# Relative symlinks are more portable across machines
symlink_strategy: absolute

# Registered asset sources (user-defined only; the builtin source
# is injected at runtime and does not appear in this file)
sources:
  - id: my-assets
    type: local
    path: ~/coding-assets

  - id: community
    type: git
    url: https://github.com/org/shared-assets.git
    alias: community-assets

# Recognized context file names (optional)
# Defaults to ["CLAUDE.md"]
# context_types: ["CLAUDE.md", "AGENTS.md"]

# Agent configuration overrides (optional)
# Only needed if your agent uses non-standard directories
agents:
  - name: claude-code
    global_dir: ~/.claude
    project_dir: .claude
```

## Config key reference

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `version` | integer | `1` | Config schema version |
| `default_scope` | string | `global` | Default deployment scope |
| `default_agent` | string | `claude-code` | Default agent to target |
| `symlink_strategy` | string | `absolute` | Symlink type: `absolute` or `relative` |
| `sources` | array | `[]` | Registered asset sources |
| `sources[].id` | string | (generated) | Unique source identifier |
| `sources[].type` | string | -- | Source type: `local`, `git`, or `builtin` |
| `sources[].path` | string | -- | Filesystem path to source |
| `sources[].url` | string | -- | Git URL (git sources only) |
| `sources[].alias` | string | -- | Human-readable alias (optional) |
| `context_types` | array | `["CLAUDE.md"]` | Recognized context file names |
| `agents` | array | (built-in) | Agent configuration overrides |
| `agents[].name` | string | -- | Agent name |
| `agents[].global_dir` | string | -- | Agent's global config directory |
| `agents[].project_dir` | string | -- | Agent's project config directory |

## Config merging

nd merges configuration from multiple sources in this order (later overrides earlier):

1. **Built-in defaults:** Default values for all settings
2. **Global config:** `~/.config/nd/config.yaml`
3. **Project config:** `.nd/config.yaml` (if present)
4. **CLI flags:** `--scope`, `--config`, and others

For sources, global sources appear first (higher priority), followed by project sources. The built-in source always has the lowest priority. If the same asset exists in both a user source and the builtin source, the user source takes priority.

## Project-level config

Create `.nd/config.yaml` in your project root to override settings per-project:

```yaml
version: 1
default_scope: project
sources:
  - id: project-assets
    type: local
    path: ./assets
```

Use cases:

- Force project scope for a repository
- Add project-specific asset sources
- Override symlink strategy for a team

## Environment variables

| Variable | Used By | Description |
|----------|---------|-------------|
| `$EDITOR` | `nd settings edit` | Preferred text editor |
| `$VISUAL` | `nd settings edit` | Visual editor (fallback if `$EDITOR` not set) |
| `$NO_COLOR` | All commands | Disable colored output (equivalent to `--no-color` flag) |

If you have not set `$EDITOR` or `$VISUAL`, `nd settings edit` uses `vi`.

## Edit config

Open your config in your default editor:

```shell
nd settings edit
```

After editing, validate your config:

```shell
nd doctor
```

`nd doctor` checks config validity as its first step.

If your config file contains invalid YAML, nd commands report a parse error with the line number. Fix the syntax and retry.

## Global flags

These flags work with every nd command and override config file settings for a single invocation:

| Flag | Description |
|------|-------------|
| `--json` | Output structured JSON for piping and parsing |
| `--yes` / `-y` | Skip confirmation prompts (essential for scripts) |
| `--dry-run` | Preview changes without applying them |
| `--verbose` / `-v` | Show detailed output on stderr |
| `--quiet` / `-q` | Suppress non-error output |
| `--scope` / `-s` | Set deployment scope: `global` or `project` |
| `--config` | Override config file path |
| `--no-color` | Disable colored output |

Example scripted workflow:

```shell
nd deploy skills/greeting --yes --json | jq '.status'
```

## Operation log

nd records every mutating operation to a JSONL log file at `~/.config/nd/logs/operations.log`. Each line is a JSON object with the timestamp, operation type, affected assets, scope, and success/failure counts.

### View the log

```shell
# Last 10 operations
tail -10 ~/.config/nd/logs/operations.log

# Pretty-print with jq
tail -5 ~/.config/nd/logs/operations.log | jq .

# Filter by operation type
cat ~/.config/nd/logs/operations.log | jq 'select(.operation == "deploy")'

# Count operations by type
cat ~/.config/nd/logs/operations.log | jq -r '.operation' | sort | uniq -c | sort -rn
```

### Log entry fields

| Field | Description |
|-------|-------------|
| `timestamp` | ISO 8601 timestamp |
| `operation` | Operation type: `deploy`, `remove`, `sync`, `profile-switch`, `snapshot-save`, `snapshot-restore`, `source-add`, `source-remove`, `source-sync`, `uninstall` |
| `assets` | Array of affected asset identities (source, type, name) |
| `scope` | Deployment scope (`global` or `project`) |
| `succeeded` | Number of successful operations |
| `failed` | Number of failed operations |
| `detail` | Additional context (profile name, source ID, snapshot name) |

### Log rotation

nd rotates the log file automatically when it exceeds 1 MB. nd preserves the previous log as `operations.log.1` and keeps only one rotated backup.

Dry-run operations (`--dry-run`) do not write log entries.
