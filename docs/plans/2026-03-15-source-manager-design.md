# Source Manager Design

| Field | Value |
| --- | --- |
| **Date** | 2026-03-15 |
| **Author** | Larah |
| **Status** | Approved |
| **Depends on** | Data types layer (complete) |

## Overview

The Source Manager is the first service layer component for nd. It sits between the config file (YAML on disk) and the asset index (in-memory), owning the full source lifecycle: config loading, source registration, asset scanning, Git sync, and source removal.

## Decisions

| Decision | Choice | Rationale |
| --- | --- | --- |
| Git operations | Shell out to `git` CLI | Simpler, smaller binary, uses existing user credentials/SSH. Aligns with non-goal of not being a Git client. |
| Source persistence | Config file only (`config.yaml`) | Single source of truth. User can hand-edit. Already modeled by `SourceEntry`. |
| Architecture | Single `SourceManager` struct | Clean API surface, straightforward. Split later if it grows. |

## Package Location

`internal/sourcemanager/`

## API Surface

```go
type SourceManager struct { ... }

func New(configPath string, projectDir string) (*SourceManager, error)
func (sm *SourceManager) Config() *config.Config
func (sm *SourceManager) AddLocal(path, alias string) (*source.Source, error)
func (sm *SourceManager) AddGit(url, alias string) (*source.Source, error)
func (sm *SourceManager) Remove(sourceID string) error
func (sm *SourceManager) Scan() (*asset.Index, error)
func (sm *SourceManager) SyncSource(sourceID string) error
func (sm *SourceManager) Sources() []source.Source
```

## Config Loading and Merging

### Load flow

1. Read `~/.config/nd/config.yaml` into `config.Config` (create defaults if missing).
2. If `projectDir` is provided, read `.nd/config.yaml` into `config.ProjectConfig`.
3. Merge: project overrides global using pointer-nil-check pattern already in `ProjectConfig`.
4. Validate merged config (fill in the existing `Config.Validate()` stub).
5. If validation fails, return all `ValidationError`s. No partial operation.

### Config writing

- Atomic write: write to temp file in same directory, fsync, rename (NFR-010).
- Round-trip YAML: use `yaml.v3` node-based API to preserve comments and formatting when possible. Fall back to marshal if the file does not exist yet.

### Default config

```yaml
version: 1
default_scope: global
default_agent: claude-code
symlink_strategy: absolute
sources: []
```

### Merge rules

- `ProjectConfig` fields override global when non-nil.
- `Sources` lists are appended (project sources after global, so global sources have higher priority per FR-016a).
- `Agents` overrides from project replace global entries by agent name.

## Source Registration

### Adding a local directory (`AddLocal`)

1. Resolve path to absolute.
2. Validate path exists and is a directory.
3. Validate path traversal (NFR-012).
4. Generate source ID from directory base name. Deduplicate with numeric suffix if needed.
5. Apply alias if provided.
6. Check for duplicate (same resolved path already registered).
7. Append `SourceEntry` to config, write config.
8. Return the `source.Source`.

### Adding a Git repo (`AddGit`)

1. Parse URL. Accept GitHub shorthand (`owner/repo`), HTTPS, or SSH URLs.
2. Determine clone target: `~/.config/nd/sources/<source-id>/`.
3. Shell out: `git clone <url> <target>`.
4. On failure, clean up partial clone directory.
5. Generate source ID from repo name.
6. Create `SourceEntry` with `Type: Git`, `Path` = clone target, `URL` = original URL.
7. Append to config, write config.

### GitHub shorthand expansion

- Input `owner/repo` becomes `https://github.com/owner/repo.git`.
- Input already contains `://` or starts with `git@`: use as-is.

### Removing a source (`Remove`)

1. Find source by ID in config.
2. Query deployment state for deployed assets from this source. Return them as a warning (do not modify deployments).
3. Remove `SourceEntry` from config, write config.
4. For Git sources, optionally delete the cloned directory (caller decides).

## Asset Scanning and Indexing

### Scanning flow (`Scan`)

For each registered source:

1. Check availability (directory exists, is readable).
2. If source has `nd-source.yaml`: load manifest, validate (existing `Manifest.Validate`), use manifest paths.
3. If no manifest: convention-based scan for `skills/`, `agents/`, `commands/`, `output-styles/`, `rules/`, `context/`, `plugins/`, `hooks/` at source root only (FR-007).
4. Skip excluded directories: `.git`, `node_modules` (NFR-017).
5. For each discovered directory entry:
   - Determine asset type from parent directory name.
   - Create `asset.Asset` with `Identity`, `SourcePath`, `IsDir`.
   - For context assets: parse folder structure, look for `_meta.yaml`, populate `ContextInfo` and `ContextMeta`.
6. Collect into `source.ScanResult` (assets, warnings, errors).
7. Unavailable sources produce a warning, not a fatal error (NFR-006).

### Building the index

- Concatenate all `ScanResult.Assets` in source registration order.
- Pass to existing `asset.NewIndex()` which handles conflict detection by source priority.

### Context asset scanning (FR-016b)

```text
context/
  go-project-rules/
    CLAUDE.md         <- the asset file
    _meta.yaml        <- optional metadata
  web-frontend/
    CLAUDE.md
    _meta.yaml
```

- Asset name = folder name (e.g., `go-project-rules`).
- `ContextInfo.FileName` = the `.md` file inside (e.g., `CLAUDE.md`).
- `ContextMeta` = parsed `_meta.yaml` if present.

### Asset type detection

| Directory | AssetType |
| --- | --- |
| `skills/` | Skill |
| `agents/` | Agent |
| `commands/` | Command |
| `output-styles/` | OutputStyle |
| `rules/` | Rule |
| `context/` | Context |
| `plugins/` | Plugin |
| `hooks/` | Hook |

## Git Sync

### `SyncSource` flow

1. Find source by ID, verify it is a Git source.
2. Shell out: `git -C <source.Path> pull --ff-only`.
3. `--ff-only` avoids creating merge commits. On failure, report the error and suggest manual resolution.
4. Re-scan is not automatic. Caller can call `Scan()` again.

## Error Handling

| Scenario | Behavior |
| --- | --- |
| Config parse errors | Return all `ValidationError`s, do not proceed |
| Missing config file | Create defaults, proceed (first-run experience) |
| Unavailable sources | Warn, continue scanning other sources (NFR-006) |
| Git clone/pull failure | Return error with stderr output |
| Path traversal attempts | Reject with `PathTraversalError` (existing type) |
| Manifest too large | Reject per NFR-013 (1 MB limit check before parsing) |

## Testing Strategy

- Use `t.TempDir()` for all filesystem operations.
- Create mock source directories with conventional layouts.
- Test manifest validation with path traversal edge cases.
- Test config merge with various nil/non-nil field combinations.
- Git operations: test URL parsing and shorthand expansion as unit tests. Integration tests with actual git repos are optional and separate.
- Fill in existing `Config.Validate()` and `ContextMeta.Validate()` stubs.

## Package Structure

```text
internal/sourcemanager/
  sourcemanager.go      - SourceManager struct, New(), Sources()
  config.go             - load, merge, write, defaults
  register.go           - AddLocal, AddGit, Remove
  scanner.go            - Scan, convention scanning, manifest loading
  git.go                - clone, pull, URL parsing
  sourcemanager_test.go - tests for all of the above
```
