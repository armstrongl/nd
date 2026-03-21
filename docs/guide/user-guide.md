# User Guide

This guide covers the core workflows for managing assets with nd.

## Interactive Mode

Many nd commands support running without arguments to get an interactive picker. This works for:

- `nd deploy` -- pick assets to deploy
- `nd remove` -- pick deployed assets to remove
- `nd profile delete` / `switch` / `deploy` -- pick a profile
- `nd snapshot delete` / `restore` -- pick a snapshot

Interactive mode is automatically disabled in non-TTY environments (pipes, scripts) and when `--json` is set. In those cases, nd returns an error with a helpful message.

## Global Flags for Scripting

These flags work with every command:

| Flag | Description |
|------|-------------|
| `--json` | Output structured JSON for piping and parsing |
| `--yes` / `-y` | Skip confirmation prompts (essential for scripts) |
| `--dry-run` | Preview what would happen without making changes |
| `--verbose` / `-v` | Show detailed output on stderr |
| `--quiet` / `-q` | Suppress non-error output |
| `--scope` / `-s` | Set deployment scope: `global` or `project` |
| `--config` | Override config file path |
| `--no-color` | Disable colored output |

Example scripted workflow:

```bash
nd deploy skills/greeting --yes --json | jq '.status'
```

## Managing Sources

### Adding a Local Directory

```bash
nd source add ~/my-assets
nd source add ~/my-assets --alias my-stuff
```

nd scans the directory for convention-based subdirectories (`skills/`, `agents/`, `commands/`, `output-styles/`, `rules/`, `context/`, `plugins/`, `hooks/`).

### Adding a Git Repository

```bash
# GitHub shorthand
nd source add owner/repo

# HTTPS
nd source add https://github.com/owner/repo.git

# SSH
nd source add git@github.com:owner/repo.git
```

Git sources are cloned to `~/.config/nd/sources/` and can be synced later.

### Listing Sources

```bash
nd source list
```

Output shows source ID, type (local/git), asset count, and path.

### Syncing Git Sources

Pull the latest changes from a git source:

```bash
nd sync --source <source-id>
```

This runs `git pull --ff-only` and then repairs any broken symlinks.

### Removing a Source

```bash
nd source remove <source-id>
```

If assets from this source are currently deployed, nd asks whether to remove them, keep them as orphans, or cancel.

## Deploying Assets

### Single Asset

```bash
nd deploy skills/greeting
```

Asset references use the format `type/name`. If the name is unique across types, you can omit the type: `nd deploy greeting`. If a name is ambiguous (exists in multiple types), nd reports the conflict and asks you to qualify with the type prefix.

### Filtering by Type

```bash
nd deploy --type skills greeting
```

### Multiple Assets

```bash
nd deploy skills/greeting commands/hello agents/researcher
```

Bulk operations continue on per-asset failure and report a summary.

### Scopes

- **Global** (`--scope global`, default): Deploys to your agent's global config directory (`~/.claude/`)
- **Project** (`--scope project`): Deploys to the project-level config directory (`.claude/` in project root)

```bash
nd deploy skills/greeting --scope project
```

### Symlink Strategy

- **Absolute** (default): Symlinks use absolute paths
- **Relative** (`--relative`): Symlinks use relative paths (better for portable setups)

```bash
nd deploy skills/greeting --relative
```

The default strategy can be changed in your config file (`symlink_strategy: relative`).

## Removing Assets

```bash
nd remove skills/greeting
```

If the asset is pinned, nd warns and asks for explicit confirmation.

Run `nd remove` with no arguments to get an interactive picker of deployed assets.

## Listing and Status

### List Available Assets

```bash
# All assets
nd list

# Filter by type
nd list --type skills

# Filter by source
nd list --source my-assets

# Filter by name pattern
nd list --pattern greeting
```

Assets marked with `*` are currently deployed.

### Check Deployment Status

```bash
nd status
```

Shows all deployed assets with:
- Health indicators (checkmark = healthy, X = issue)
- Scope (global or project)
- Origin (manual, pinned, or profile name)
- Source

### JSON Output

```bash
nd list --json
nd status --json
```

## Settings

Open your config file in your default editor (`$EDITOR`, `$VISUAL`, or `vi`):

```bash
nd settings edit
```

See [Configuration](configuration.md) for all available settings.

## Syncing and Repair

Fix broken symlinks across all deployments:

```bash
nd sync
```

Sync a specific git source (pull + repair):

```bash
nd sync --source <source-id>
```

Preview what would be repaired:

```bash
nd sync --dry-run
```

## Health Checks

Run a comprehensive health check:

```bash
nd doctor
```

This validates:
1. Config file validity
2. Source accessibility
3. Deployment health (broken symlinks, drift)
4. Agent detection
5. Git availability

## Operation Log

nd records every mutating operation to a JSONL log file at `~/.config/nd/logs/operations.log`. Each line is a JSON object with the timestamp, operation type, affected assets, scope, and success/failure counts.

### Viewing the Log

```bash
# Last 10 operations
tail -10 ~/.config/nd/logs/operations.log

# Pretty-print with jq
tail -5 ~/.config/nd/logs/operations.log | jq .

# Filter by operation type
cat ~/.config/nd/logs/operations.log | jq 'select(.operation == "deploy")'

# Count operations by type
cat ~/.config/nd/logs/operations.log | jq -r '.operation' | sort | uniq -c | sort -rn
```

### Log Entry Fields

| Field | Description |
|-------|-------------|
| `timestamp` | ISO 8601 timestamp |
| `operation` | Operation type: `deploy`, `remove`, `sync`, `profile-switch`, `snapshot-save`, `snapshot-restore`, `source-add`, `source-remove`, `source-sync`, `uninstall` |
| `assets` | Array of affected asset identities (source, type, name) |
| `scope` | Deployment scope (`global` or `project`) |
| `succeeded` | Number of successful operations |
| `failed` | Number of failed operations |
| `detail` | Additional context (profile name, source ID, etc.) |

### Log Rotation

The log file rotates automatically when it exceeds 1 MB. The previous log is preserved as `operations.log.1`. Only one rotated backup is kept.

Dry-run operations (`--dry-run`) do not write log entries.

## Shell Completions

Generate and install shell completions:

```bash
# Print completion script
nd completion bash
nd completion zsh
nd completion fish

# Auto-install to standard location
nd completion bash --install
nd completion zsh --install
nd completion fish --install

# Install to custom directory
nd completion zsh --install-dir ~/.my-completions
```

For zsh, ensure your `~/.zshrc` includes:

```bash
fpath+=~/.zfunc
autoload -Uz compinit && compinit
```

## Uninstalling

Remove all nd-managed symlinks from agent config directories:

```bash
nd uninstall
```

This removes symlinks but does **not** delete your config directory (`~/.config/nd/`). To fully uninstall, also remove that directory and the nd binary.
