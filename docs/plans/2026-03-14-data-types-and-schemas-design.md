# nd data types and schemas design

| Field | Value |
| --- | --- |
| **Date** | 2026-03-14 |
| **Author** | Larah |
| **Status** | Approved |
| **Spec version** | nd-Go-spec.md v0.6 |
| **Depends on** | repo-management-design.md |

## Overview

This document defines every data type, enum, schema, and on-disk YAML format for the nd project. It covers the complete type system across all domain packages, with Go struct definitions, validation strategy, and dependency graph. No implementation code should be written without this design in place.

### Design decisions

| Decision | Choice | Rationale |
| --- | --- | --- |
| Package organization | Domain packages + shared enums | Matches repo-management-design.md layout. Shared `internal/nd/` package avoids import cycles. |
| Validation | Struct tags + custom `Validate()` methods | No extra dependencies. gopkg.in/yaml.v3 tags handle serialization; `Validate()` returns structured errors. |
| Schema versioning | v1 only for now | Design v1 schemas. Version field is a constant. Migration infrastructure added when v2 is needed. |
| Asset type system | Method-bearing string enum | `AssetType` carries behavior (IsDirectory, DeploySubdir) directly on the type. |
| Serialization tags | Both `yaml:` and `json:` on all persisted and API-facing types | Enables `--json` CLI flag without extra mapping. JSON output contract is defined by struct tags. |

### Dependency graph

```text
Level 0:  internal/nd/          (shared enums, constants, error types)

Level 1:  internal/config/      (Config, ProjectConfig, SourceEntry, ...)
          internal/asset/       (Identity, Asset, Index, CachedIndex, ...)
          internal/agent/       (Agent, DetectionResult)
          internal/backup/      (Backup, OriginalFileKind)

Level 2:  internal/source/      (Source, Manifest, ScanResult)
              imports: nd, asset
          internal/state/       (DeploymentState, Deployment, FileLock, ...)
              imports: nd, asset
          internal/profile/     (Profile, Snapshot, SwitchDiff)
              imports: nd, asset
          internal/oplog/       (LogEntry)
              imports: nd, asset

Level 3:  internal/deploy/      (Request, Result, BulkResult, UninstallPlan, ...)
              imports: nd, asset, state, agent
          internal/doctor/      (DoctorReport)
              imports: nd, config, source, state, agent
```

No cycles. All arrows point upward from higher levels to lower levels. The `nd` package is the universal root. `deploy` and `doctor` are the deepest nodes.

## 1. shared enums, constants, and error types (`internal/nd/`)

Thin package providing shared vocabulary. Every domain package imports this.

```go
package nd

import "fmt"

// AssetType represents the category of a deployable asset.
type AssetType string

const (
    AssetSkill       AssetType = "skills"
    AssetAgent       AssetType = "agents"
    AssetCommand     AssetType = "commands"
    AssetOutputStyle AssetType = "output-styles"
    AssetRule        AssetType = "rules"
    AssetContext     AssetType = "context"
    AssetPlugin      AssetType = "plugins"
    AssetHook        AssetType = "hooks"
)

// AllAssetTypes returns all recognized asset types in discovery order.
func AllAssetTypes() []AssetType {
    return []AssetType{
        AssetSkill, AssetAgent, AssetCommand, AssetOutputStyle,
        AssetRule, AssetContext, AssetPlugin, AssetHook,
    }
}

// DeployableAssetTypes returns asset types that can be deployed via symlink.
// Plugins are excluded (they use the export workflow, not symlink deployment).
func DeployableAssetTypes() []AssetType {
    return []AssetType{
        AssetSkill, AssetAgent, AssetCommand, AssetOutputStyle,
        AssetRule, AssetContext, AssetHook,
    }
}

// IsDeployable returns true if this asset type can be deployed via symlink.
// Plugins are not deployable (they use nd export + /plugin install).
func (t AssetType) IsDeployable() bool {
    return t != AssetPlugin
}

// IsDirectory returns true if this asset type deploys as a directory symlink.
// skills, plugins, hooks -> true
// agents, commands, output-styles, rules -> false
// context -> false (symlinks the file inside the folder, not the folder)
func (t AssetType) IsDirectory() bool {
    switch t {
    case AssetSkill, AssetPlugin, AssetHook:
        return true
    default:
        return false
    }
}

// DeploySubdir returns the subdirectory name within an agent's config dir.
// Returns "" for context (which deploys to fixed paths determined by filename).
func (t AssetType) DeploySubdir() string {
    if t == AssetContext {
        return ""
    }
    return string(t)
}

// RequiresSettingsRegistration returns true if deploying this asset type
// requires the user to manually edit settings.json afterward.
func (t AssetType) RequiresSettingsRegistration() bool {
    switch t {
    case AssetHook, AssetOutputStyle:
        return true
    default:
        return false
    }
}

// Scope represents a deployment scope.
type Scope string

const (
    ScopeGlobal  Scope = "global"
    ScopeProject Scope = "project"
)

// DeployOrigin tracks how an asset was deployed.
type DeployOrigin string

const (
    OriginManual DeployOrigin = "manual"
    OriginPinned DeployOrigin = "pinned"
    // Profile origins are formatted as "profile:<name>"
)

// OriginProfile returns a profile-scoped deploy origin.
func OriginProfile(name string) DeployOrigin {
    return DeployOrigin("profile:" + name)
}

// IsProfile returns true if this origin is a profile deployment.
func (o DeployOrigin) IsProfile() bool {
    return len(o) > 8 && o[:8] == "profile:"
}

// ProfileName extracts the profile name, or "" if not a profile origin.
func (o DeployOrigin) ProfileName() string {
    if o.IsProfile() {
        return string(o[8:])
    }
    return ""
}

// SymlinkStrategy controls how symlinks are created.
type SymlinkStrategy string

const (
    SymlinkAbsolute SymlinkStrategy = "absolute"
    SymlinkRelative SymlinkStrategy = "relative"
)

// SourceType distinguishes local directories from Git repos.
type SourceType string

const (
    SourceLocal SourceType = "local"
    SourceGit   SourceType = "git"
)

// Exit code constants per NFR-016.
const (
    ExitSuccess        = 0
    ExitError          = 1
    ExitPartialFailure = 2
    ExitInvalidUsage   = 3
)

// SchemaVersion is the current version for all nd-managed YAML files.
const SchemaVersion = 1

// ContextFileName constants for the built-in context file names.
// ContextInfo.FileName is typed as string (not this type) to support
// custom context file types registered via FR-034.
const (
    ContextCLAUDE      = "CLAUDE.md"
    ContextAGENTS      = "AGENTS.md"
    ContextCLAUDELocal = "CLAUDE.local.md"
    ContextAGENTSLocal = "AGENTS.local.md"
)

// BuiltinContextFileNames returns all built-in context file name constants.
func BuiltinContextFileNames() []string {
    return []string{
        ContextCLAUDE, ContextAGENTS, ContextCLAUDELocal, ContextAGENTSLocal,
    }
}

// IsLocalOnlyContext returns true if a context filename deploys only at project scope.
// Works for both built-in and custom context file types (checks the .local.md suffix).
func IsLocalOnlyContext(filename string) bool {
    return len(filename) > 9 && filename[len(filename)-9:] == ".local.md"
}

// --- Domain error types ---

// PathTraversalError is returned when a path escapes its allowed root (NFR-012).
type PathTraversalError struct {
    Path       string // The offending path
    Root       string // The root it should be confined to
    SourceID   string // Which source contained the path
}

func (e *PathTraversalError) Error() string {
    return fmt.Sprintf("path %q escapes source root %q in source %s", e.Path, e.Root, e.SourceID)
}

// LockError is returned when the state file lock cannot be acquired (NFR-011).
type LockError struct {
    Path    string // The file being locked
    Timeout string // How long we waited
    Stale   bool   // True if an existing lock was detected as stale
}

func (e *LockError) Error() string {
    if e.Stale {
        return fmt.Sprintf("stale lock on %s (held >60s), breaking and retrying", e.Path)
    }
    return fmt.Sprintf("could not acquire lock on %s within %s: another nd process may be running", e.Path, e.Timeout)
}

// ConflictError is returned when a deploy target already has a file/symlink (FR-016b).
type ConflictError struct {
    TargetPath  string         // Where we want to deploy
    ExistingKind OriginalFileKind // What's already there
    AssetName   string         // What we're trying to deploy
}

func (e *ConflictError) Error() string {
    return fmt.Sprintf("conflict at %s: existing %s blocks deployment of %s", e.TargetPath, e.ExistingKind, e.AssetName)
}

// OriginalFileKind describes what kind of file already exists at a target path.
// Used by backup and conflict detection to determine warning severity.
type OriginalFileKind string

const (
    FileKindManagedSymlink OriginalFileKind = "nd-managed-symlink"
    FileKindForeignSymlink OriginalFileKind = "foreign-symlink"
    FileKindPlainFile      OriginalFileKind = "plain-file"
)
```

## 2. configuration types (`internal/config/`)

Maps to `~/.config/nd/config.yaml` (global) and `.nd/config.yaml` (project).

```go
package config

import "nd/internal/nd"

// Config represents the merged, resolved configuration.
// This is what the rest of the application uses after loading + merging.
// Merge order: built-in defaults -> global config -> project config -> CLI flags.
type Config struct {
    Version         int                `yaml:"version"          json:"version"`
    DefaultScope    nd.Scope           `yaml:"default_scope"    json:"default_scope"`
    DefaultAgent    string             `yaml:"default_agent"    json:"default_agent"`
    SymlinkStrategy nd.SymlinkStrategy `yaml:"symlink_strategy" json:"symlink_strategy"`
    Sources         []SourceEntry      `yaml:"sources"          json:"sources"`
    Agents          []AgentOverride    `yaml:"agents,omitempty" json:"agents,omitempty"`
    ContextTypes    []string           `yaml:"context_types,omitempty" json:"context_types,omitempty"` // FR-034
}

// SourceEntry represents a source registration in the config file.
// Sources are listed in registration order (first registered = highest priority per FR-016a).
type SourceEntry struct {
    ID    string        `yaml:"id"              json:"id"`
    Type  nd.SourceType `yaml:"type"            json:"type"`
    Path  string        `yaml:"path"            json:"path"`
    Alias string        `yaml:"alias,omitempty" json:"alias,omitempty"`
}

// AgentOverride lets users customize agent config directory paths (FR-033).
type AgentOverride struct {
    Name       string `yaml:"name"        json:"name"`
    GlobalDir  string `yaml:"global_dir"  json:"global_dir"`
    ProjectDir string `yaml:"project_dir" json:"project_dir"`
}

// ProjectConfig represents .nd/config.yaml (project-level overrides).
// Fields are pointers so we can distinguish "not set" from "set to zero value"
// during the merge with global config.
type ProjectConfig struct {
    Version         int                 `yaml:"version"`
    DefaultScope    *nd.Scope           `yaml:"default_scope,omitempty"`
    DefaultAgent    *string             `yaml:"default_agent,omitempty"`
    SymlinkStrategy *nd.SymlinkStrategy `yaml:"symlink_strategy,omitempty"`
    Sources         []SourceEntry       `yaml:"sources,omitempty"`
    Agents          []AgentOverride     `yaml:"agents,omitempty"`
}

// Validate checks all fields for correctness.
// Returns a slice of ValidationError with file and line info (NFR-005).
func (c *Config) Validate() []ValidationError { ... }

// ValidationError represents a single config validation failure.
type ValidationError struct {
    File    string `json:"file"`
    Line    int    `json:"line"`
    Field   string `json:"field"`
    Message string `json:"message"`
}

func (e ValidationError) Error() string { ... }
```

### On-disk format: `config.yaml`

```yaml
version: 1
default_scope: global
default_agent: claude-code
symlink_strategy: absolute

sources:
  - id: my-assets
    type: local
    path: /Users/dev/my-assets
  - id: team-skills
    type: git
    path: https://github.com/team/skills.git
    alias: Team Skills

# Optional: override agent config directory paths (FR-033)
agents:
  - name: claude-code
    global_dir: /Users/dev/.claude
    project_dir: .claude

# Optional: additional context file types beyond built-ins (FR-034)
context_types:
  - CUSTOM.md
```

## 3. asset types (`internal/asset/`)

Core domain objects representing discovered assets from sources.

```go
package asset

import (
    "time"

    "nd/internal/nd"
)

// Identity uniquely identifies an asset across all sources.
// Used as the primary key in profiles, snapshots, and deployment state.
// The tuple (SourceID, Type, Name) is globally unique.
type Identity struct {
    SourceID string       `yaml:"source_id"  json:"source_id"`
    Type     nd.AssetType `yaml:"asset_type" json:"asset_type"`
    Name     string       `yaml:"asset_name" json:"asset_name"`
}

// String returns "source:type/name" for display and logging.
func (id Identity) String() string { ... }

// Asset represents a discovered asset from a registered source.
type Asset struct {
    Identity
    SourcePath  string       `yaml:"-" json:"source_path"` // Absolute path to the asset in the source
    IsDir       bool         `yaml:"-" json:"is_dir"`
    ContextFile *ContextInfo `yaml:"-" json:"context_file,omitempty"`
    Meta        *ContextMeta `yaml:"-" json:"meta,omitempty"`
}

// ContextInfo holds context-specific details for context assets.
// FileName is a plain string (not an enum) to support custom context file
// types registered via config (FR-034).
type ContextInfo struct {
    FolderName string // The named folder (e.g., "go-project-rules")
    FileName   string // The file inside (e.g., "CLAUDE.md", or a custom type)
}

// ContextMeta represents the _meta.yaml file inside a context folder (FR-016c).
type ContextMeta struct {
    Description    string   `yaml:"description"                json:"description"`
    Tags           []string `yaml:"tags,omitempty"              json:"tags,omitempty"`
    TargetLanguage string   `yaml:"target_language,omitempty"   json:"target_language,omitempty"`
    TargetProject  string   `yaml:"target_project,omitempty"    json:"target_project,omitempty"`
    TargetAgent    string   `yaml:"target_agent,omitempty"      json:"target_agent,omitempty"`
}

// Validate checks ContextMeta fields for correctness.
func (m *ContextMeta) Validate() error { ... }

// Index is an in-memory collection of all discovered assets across all sources.
// Built once after source scanning, queried by the deploy engine, TUI, and CLI.
type Index struct {
    assets    []Asset
    byID      map[Identity]*Asset
    byType    map[nd.AssetType][]*Asset
    bySource  map[string][]*Asset
    conflicts []Conflict
}

// Conflict records when two sources have the same (type, name) pair (FR-016a).
type Conflict struct {
    Type   nd.AssetType `json:"asset_type"`
    Name   string       `json:"asset_name"`
    Winner string       `json:"winner"` // Source ID that takes priority (first registered)
    Loser  string       `json:"loser"`
}

// NewIndex builds an asset index from a slice of assets,
// detecting conflicts per FR-016a (first source wins by registration order).
func NewIndex(assets []Asset) *Index { ... }

// Lookup finds an asset by identity. Returns nil if not found.
func (idx *Index) Lookup(id Identity) *Asset { ... }

// ByType returns all assets of a given type.
func (idx *Index) ByType(t nd.AssetType) []*Asset { ... }

// BySource returns all assets from a given source.
func (idx *Index) BySource(sourceID string) []*Asset { ... }

// All returns all assets in discovery order.
func (idx *Index) All() []*Asset { ... }

// Conflicts returns all detected duplicate-name conflicts.
func (idx *Index) Conflicts() []Conflict { ... }

// CachedIndex is the on-disk representation of the asset discovery cache.
// Stored at ~/.cache/nd/index/<source_id>.yaml.
// Rebuilt when the source is modified or when the cache is missing.
// Enables NFR-001 (500ms TUI startup with 500+ assets) by avoiding
// full source scans on every invocation.
type CachedIndex struct {
    Version   int       `yaml:"version"`
    SourceID  string    `yaml:"source_id"`
    BuiltAt   time.Time `yaml:"built_at"`
    SourceMod time.Time `yaml:"source_mod"` // Source directory mtime at cache time
    Assets    []Asset   `yaml:"assets"`
}

// IsStale returns true if the cache is older than the source's last modification.
func (c *CachedIndex) IsStale(currentSourceMod time.Time) bool {
    return currentSourceMod.After(c.SourceMod)
}
```

### On-disk format: `_meta.yaml`

```yaml
description: Go project rules and conventions for Claude Code
tags:
  - go
  - backend
  - cli
target_language: go
target_project: cli-tools
target_agent: claude-code
```

## 4. source types (`internal/source/`)

Registered asset sources and the `nd-source.yaml` manifest.

```go
package source

import (
    "nd/internal/nd"
    "nd/internal/asset"
)

// Source represents a registered asset source (local directory or Git repo).
type Source struct {
    ID        string        `yaml:"id"              json:"id"`
    Type      nd.SourceType `yaml:"type"            json:"type"`
    Path      string        `yaml:"path"            json:"path"`
    URL       string        `yaml:"url,omitempty"   json:"url,omitempty"`
    Alias     string        `yaml:"alias,omitempty" json:"alias,omitempty"`
    Order     int           `yaml:"-"               json:"order"`
    Manifest  *Manifest     `yaml:"-"               json:"-"`
    Available bool          `yaml:"-"               json:"available"`
}

// Manifest represents an nd-source.yaml file (FR-008).
// Overrides convention-based discovery with custom paths and exclusions.
type Manifest struct {
    Version  int                       `yaml:"version"`
    Paths    map[nd.AssetType][]string `yaml:"paths"`
    Exclude  []string                  `yaml:"exclude,omitempty"`
    Metadata *ManifestMetadata         `yaml:"metadata,omitempty"`
}

// ManifestMetadata is optional metadata about the source itself.
type ManifestMetadata struct {
    Name        string   `yaml:"name"`
    Description string   `yaml:"description,omitempty"`
    Author      string   `yaml:"author,omitempty"`
    URL         string   `yaml:"url,omitempty"`
    Tags        []string `yaml:"tags,omitempty"`
}

// Validate checks the manifest for correctness.
// Enforces:
//   - NFR-012: all paths resolve within sourceRoot (no path traversal).
//     Returns *nd.PathTraversalError for violations.
//   - NFR-013: path lists limited to 1,000 entries, file size bounded at 1 MB.
//   - Unrecognized keys in Paths (not a valid AssetType) are rejected.
func (m *Manifest) Validate(sourceRoot string) []error { ... }

// ScanResult holds the output of scanning a single source.
type ScanResult struct {
    SourceID string
    Assets   []asset.Asset
    Warnings []string // Non-fatal issues (e.g., unreadable directories)
    Errors   []error  // Fatal issues for specific paths
}
```

### On-disk format: `nd-source.yaml`

```yaml
version: 1

paths:
  skills:
    - skills/
    - go-skills/skills/
  agents:
    - agents/
  commands:
    - commands/

exclude:
  - experimental/
  - skills/deprecated/

metadata:
  name: My Asset Library
  description: Personal collection of Claude Code assets
  author: dev
  tags:
    - go
    - productivity
```

## 5. deployment state types (`internal/state/`)

Maps to `~/.config/nd/state/deployments.yaml`. Also provides file locking.

```go
package state

import (
    "time"

    "nd/internal/nd"
    "nd/internal/asset"
)

// DeploymentState is the root structure of deployments.yaml.
// Written atomically (write-to-temp-then-rename) per NFR-010.
// Guarded by advisory file lock per NFR-011.
type DeploymentState struct {
    Version       int          `yaml:"version"                  json:"version"`
    ActiveProfile string       `yaml:"active_profile,omitempty" json:"active_profile,omitempty"`
    Deployments   []Deployment `yaml:"deployments"              json:"deployments"`
}

// Deployment represents a single managed symlink.
type Deployment struct {
    SourceID    string          `yaml:"source_id"                json:"source_id"`
    AssetType   nd.AssetType    `yaml:"asset_type"               json:"asset_type"`
    AssetName   string          `yaml:"asset_name"               json:"asset_name"`
    SourcePath  string          `yaml:"source_path"              json:"source_path"`
    LinkPath    string          `yaml:"link_path"                json:"link_path"`
    Scope       nd.Scope        `yaml:"scope"                    json:"scope"`
    ProjectPath string          `yaml:"project_path,omitempty"   json:"project_path,omitempty"`
    Origin      nd.DeployOrigin `yaml:"origin"                   json:"origin"`
    DeployedAt  time.Time       `yaml:"deployed_at"              json:"deployed_at"`
}

// Identity returns the asset identity for this deployment.
func (d *Deployment) Identity() asset.Identity {
    return asset.Identity{
        SourceID: d.SourceID,
        Type:     d.AssetType,
        Name:     d.AssetName,
    }
}

// HealthStatus represents the result of checking a single deployment.
type HealthStatus int

const (
    HealthOK       HealthStatus = iota // Symlink exists and points to correct target
    HealthBroken                       // Symlink exists but target is missing
    HealthDrifted                      // Symlink points to wrong target
    HealthOrphaned                     // Source no longer exists in any registered source
    HealthMissing                      // Symlink was deleted externally
)

func (h HealthStatus) String() string {
    switch h {
    case HealthOK:
        return "ok"
    case HealthBroken:
        return "broken"
    case HealthDrifted:
        return "drifted"
    case HealthOrphaned:
        return "orphaned"
    case HealthMissing:
        return "missing"
    default:
        return "unknown"
    }
}

// HealthCheck is the result of checking one deployment's health.
type HealthCheck struct {
    Deployment Deployment   `json:"deployment"`
    Status     HealthStatus `json:"status"`
    Detail     string       `json:"detail"`
}

// Validate checks the deployment state for internal consistency.
func (s *DeploymentState) Validate() []error { ... }

// FindByIdentity returns deployments matching an asset identity.
func (s *DeploymentState) FindByIdentity(id asset.Identity) []Deployment { ... }

// FindByScope returns all deployments for a given scope.
func (s *DeploymentState) FindByScope(scope nd.Scope) []Deployment { ... }

// FindByOrigin returns all deployments with a specific origin.
func (s *DeploymentState) FindByOrigin(origin nd.DeployOrigin) []Deployment { ... }

// FindByProject returns all project-scoped deployments for a given project path.
func (s *DeploymentState) FindByProject(projectPath string) []Deployment { ... }

// --- File locking (NFR-011) ---

// FileLock provides advisory file locking on deployments.yaml.
// Acquired before read-modify-write cycles, released after the atomic rename.
type FileLock struct {
    Path       string        // Path to the lock file (deployments.yaml.lock)
    AcquiredAt time.Time     // When the lock was acquired
    fd         int           // File descriptor (internal)
}

// Acquire attempts to acquire the lock within the given timeout.
// Returns *nd.LockError if the lock cannot be acquired.
// If a stale lock (>60s) is detected, it is broken automatically.
func (l *FileLock) Acquire(timeout time.Duration) error { ... }

// Release releases the file lock.
func (l *FileLock) Release() error { ... }
```

### On-disk format: `deployments.yaml`

```yaml
version: 1
active_profile: go-backend
deployments:
  - source_id: my-assets
    asset_type: skills
    asset_name: code-review
    source_path: /Users/dev/assets/skills/code-review
    link_path: /Users/dev/.claude/skills/code-review
    scope: global
    origin: manual
    deployed_at: "2026-03-10T14:30:00Z"

  - source_id: team-assets
    asset_type: agents
    asset_name: go-specialist
    source_path: /Users/dev/assets/agents/go-specialist.md
    link_path: /Users/dev/projects/myapp/.claude/agents/go-specialist.md
    scope: project
    project_path: /Users/dev/projects/myapp
    origin: "profile:go-backend"
    deployed_at: "2026-03-11T09:15:00Z"

  - source_id: my-assets
    asset_type: commands
    asset_name: deploy
    source_path: /Users/dev/assets/commands/deploy.md
    link_path: /Users/dev/.claude/commands/deploy.md
    scope: global
    origin: pinned
    deployed_at: "2026-03-09T08:00:00Z"
```

## 6. profile and snapshot types (`internal/profile/`)

Maps to `~/.config/nd/profiles/*.yaml` and `~/.config/nd/snapshots/**/*.yaml`.

```go
package profile

import (
    "time"

    "nd/internal/nd"
    "nd/internal/asset"
)

// Profile represents a named, curated collection of assets (FR-022).
// Stored as ~/.config/nd/profiles/<name>.yaml
type Profile struct {
    Version     int            `yaml:"version"               json:"version"`
    Name        string         `yaml:"name"                  json:"name"`
    Description string         `yaml:"description,omitempty" json:"description,omitempty"`
    CreatedAt   time.Time      `yaml:"created_at"            json:"created_at"`
    UpdatedAt   time.Time      `yaml:"updated_at"            json:"updated_at"`
    Assets      []ProfileAsset `yaml:"assets"                json:"assets"`
}

// ProfileAsset is a reference to an asset within a profile.
// Uses Identity as primary key with a cached path as fallback hint (resolved Q4).
// Each entry is scope-aware: one profile can mix global and project-scoped assets.
type ProfileAsset struct {
    SourceID  string       `yaml:"source_id"            json:"source_id"`
    AssetType nd.AssetType `yaml:"asset_type"           json:"asset_type"`
    AssetName string       `yaml:"asset_name"           json:"asset_name"`
    Scope     nd.Scope     `yaml:"scope"                json:"scope"`
    PathHint  string       `yaml:"path_hint,omitempty"  json:"path_hint,omitempty"`
}

// Identity returns the asset identity for this profile entry.
func (pa *ProfileAsset) Identity() asset.Identity {
    return asset.Identity{
        SourceID: pa.SourceID,
        Type:     pa.AssetType,
        Name:     pa.AssetName,
    }
}

// Validate checks the profile for internal consistency.
// Enforces: profiles must not reference plugin assets (spec line 106).
func (p *Profile) Validate() []error { ... }

// Snapshot represents a point-in-time record of all deployments (FR-020).
// User-created: ~/.config/nd/snapshots/user/<name>.yaml
// Auto-created: ~/.config/nd/snapshots/auto/auto-<timestamp>.yaml (last 5 retained)
type Snapshot struct {
    Version     int             `yaml:"version"    json:"version"`
    Name        string          `yaml:"name"       json:"name"`
    CreatedAt   time.Time       `yaml:"created_at" json:"created_at"`
    Auto        bool            `yaml:"auto"       json:"auto"`
    Deployments []SnapshotEntry `yaml:"deployments" json:"deployments"`
}

// SnapshotEntry captures the exact state of one deployment at snapshot time.
// Intentionally a full copy (not a reference) -- snapshots are immutable
// records that remain valid even if sources or profiles change later.
type SnapshotEntry struct {
    SourceID    string          `yaml:"source_id"              json:"source_id"`
    AssetType   nd.AssetType    `yaml:"asset_type"             json:"asset_type"`
    AssetName   string          `yaml:"asset_name"             json:"asset_name"`
    SourcePath  string          `yaml:"source_path"            json:"source_path"`
    LinkPath    string          `yaml:"link_path"              json:"link_path"`
    Scope       nd.Scope        `yaml:"scope"                  json:"scope"`
    ProjectPath string          `yaml:"project_path,omitempty" json:"project_path,omitempty"`
    Origin      nd.DeployOrigin `yaml:"origin"                 json:"origin"`
    DeployedAt  time.Time       `yaml:"deployed_at"            json:"deployed_at"`
}

// Validate checks the snapshot for internal consistency.
// Enforces: snapshots must not reference plugin assets (spec line 106).
func (s *Snapshot) Validate() []error { ... }

// SwitchDiff represents the computed difference between two profiles
// for the profile switch algorithm (spec: profile scope semantics).
type SwitchDiff struct {
    // Keep contains assets in both profiles with matching (source_id, asset_type,
    // asset_name, scope). Per spec line 563, scope IS part of the equality key.
    // An asset in both profiles but with different scopes is treated as a
    // remove + deploy, not a keep.
    // For kept assets, the origin is updated to profile:<target> and
    // deployed_at is refreshed (spec line 566).
    Keep   []ProfileAsset

    Remove []ProfileAsset // Only in current profile (to be removed)
    Deploy []ProfileAsset // Only in target profile (to be deployed)
}

// ComputeSwitchDiff computes the diff between current and target profiles.
// Equality is determined by (source_id, asset_type, asset_name, scope).
// Used by the deploy engine to execute profile switches.
func ComputeSwitchDiff(current, target *Profile) SwitchDiff { ... }
```

### On-disk format: profile YAML

```yaml
version: 1
name: go-backend
description: Go backend development assets
created_at: "2026-03-10T10:00:00Z"
updated_at: "2026-03-12T14:30:00Z"
assets:
  - source_id: my-assets
    asset_type: skills
    asset_name: code-review
    scope: global
    path_hint: /Users/dev/assets/skills/code-review
  - source_id: team-assets
    asset_type: agents
    asset_name: go-specialist
    scope: project
    path_hint: /Users/dev/team/agents/go-specialist.md
```

### On-disk format: snapshot YAML

```yaml
version: 1
name: before-profile-switch
created_at: "2026-03-12T14:29:00Z"
auto: true
deployments:
  - source_id: my-assets
    asset_type: skills
    asset_name: code-review
    source_path: /Users/dev/assets/skills/code-review
    link_path: /Users/dev/.claude/skills/code-review
    scope: global
    origin: manual
    deployed_at: "2026-03-10T14:30:00Z"
  - source_id: team-assets
    asset_type: agents
    asset_name: go-specialist
    source_path: /Users/dev/team/agents/go-specialist.md
    link_path: /Users/dev/projects/myapp/.claude/agents/go-specialist.md
    scope: project
    project_path: /Users/dev/projects/myapp
    origin: "profile:go-backend"
    deployed_at: "2026-03-11T09:15:00Z"
```

## 7. agent registry types (`internal/agent/`)

Coding agent detection and config directory resolution (FR-016, FR-033).

```go
package agent

import "nd/internal/nd"

// Agent represents a detected coding agent installation.
type Agent struct {
    Name       string `json:"name"`
    GlobalDir  string `json:"global_dir"`
    ProjectDir string `json:"project_dir"`
    Detected   bool   `json:"detected"`
    InPath     bool   `json:"in_path"`
}

// DeployPath computes the full path where an asset's symlink should be created.
// Handles the special cases:
//   - Context files deploy to project root (not inside .claude/) at project scope
//   - Context files deploy to ~/.claude/ (not a subdirectory) at global scope
//   - .local.md context files deploy only at project scope; returns an error
//     if scope=global and IsLocalOnlyContext(contextFile) is true
//   - All other types deploy to <configDir>/<assetType>/<assetName>
//
// contextFile is the filename inside the context folder (e.g., "CLAUDE.md").
// Pass "" for non-context asset types.
// Returns ("", error) for invalid combinations (e.g., global + .local.md).
func (a *Agent) DeployPath(
    assetType nd.AssetType,
    assetName string,
    scope nd.Scope,
    projectRoot string,
    contextFile string,
) (string, error) { ... }

// DetectionResult holds the output of scanning for installed agents.
type DetectionResult struct {
    Agents   []Agent  `json:"agents"`
    Warnings []string `json:"warnings,omitempty"`
}
```

## 8. deploy engine types (`internal/deploy/`)

Operational types used during deploy, remove, sync, and uninstall operations.

```go
package deploy

import (
    "nd/internal/nd"
    "nd/internal/asset"
    "nd/internal/state"
    "nd/internal/agent"
)

// Request represents a single deploy or remove operation.
type Request struct {
    Asset       asset.Asset
    Agent       agent.Agent
    Scope       nd.Scope
    ProjectRoot string
    Strategy    nd.SymlinkStrategy
    Origin      nd.DeployOrigin
    DryRun      bool
}

// Result represents the outcome of a single deploy/remove operation.
type Result struct {
    Request  Request        `json:"-"`
    AssetID  asset.Identity `json:"asset"`
    Success  bool           `json:"success"`
    Action   Action         `json:"action"`
    Error    error          `json:"-"`
    ErrorMsg string         `json:"error,omitempty"` // Serialized form of Error
    LinkPath string         `json:"link_path"`
}

// Action describes what a deploy/remove operation did.
type Action int

const (
    ActionCreated   Action = iota // New symlink created
    ActionRemoved                 // Symlink removed
    ActionReplaced                // Existing symlink replaced
    ActionSkipped                 // No action needed (already correct)
    ActionBackedUp                // Existing file backed up before replace (context files)
    ActionFailed                  // Operation failed
    ActionDryRun                  // Would have done this (dry-run mode)
)

func (a Action) String() string {
    switch a {
    case ActionCreated:
        return "created"
    case ActionRemoved:
        return "removed"
    case ActionReplaced:
        return "replaced"
    case ActionSkipped:
        return "skipped"
    case ActionBackedUp:
        return "backed-up"
    case ActionFailed:
        return "failed"
    case ActionDryRun:
        return "dry-run"
    default:
        return "unknown"
    }
}

// MarshalJSON implements json.Marshaler for Action.
func (a Action) MarshalJSON() ([]byte, error) { ... }

// BulkResult summarizes a batch of operations.
type BulkResult struct {
    Results   []Result `json:"results"`
    Succeeded int      `json:"succeeded"`
    Failed    int      `json:"failed"`
}

// HasFailures returns true if any operation failed.
func (br *BulkResult) HasFailures() bool { return br.Failed > 0 }

// FailedResults returns only the failed results.
func (br *BulkResult) FailedResults() []Result { ... }

// SyncPlan represents what the sync command will do.
type SyncPlan struct {
    Repairs []SyncAction `json:"repairs"`
    Removes []SyncAction `json:"removes"`
    Healthy int          `json:"healthy"`
}

// SyncAction describes a single repair or removal during sync.
type SyncAction struct {
    Deployment state.Deployment `json:"deployment"`
    Health     state.HealthCheck `json:"health"`
    Action     Action            `json:"action"`
}

// UninstallPlan represents what nd uninstall --dry-run would do (FR-036a).
type UninstallPlan struct {
    Symlinks     []state.Deployment `json:"symlinks"`      // All nd-managed symlinks to remove
    Directories  []string           `json:"directories"`   // nd directories to delete (optional)
    SymlinkCount int                `json:"symlink_count"`
}
```

## 9. backup types (`internal/backup/`)

Context file backup management (FR-016b).

```go
package backup

import (
    "time"

    "nd/internal/nd"
)

// Backup represents a backed-up file.
// Naming convention: <filename>.<ISO-8601-timestamp>.bak
// Stored in: ~/.config/nd/backups/
// Retention: last 5 per target location (grouped by OriginalPath).
type Backup struct {
    OriginalPath string              `json:"original_path"`
    BackupPath   string              `json:"backup_path"`
    CreatedAt    time.Time           `json:"created_at"`
    OriginalKind nd.OriginalFileKind `json:"original_kind"`
}
```

`OriginalKind` replaces the former `WasManual bool`. It distinguishes three cases:

- `FileKindManagedSymlink` — an existing nd-managed symlink (standard replace)
- `FileKindForeignSymlink` — a symlink created by something else (moderate warning)
- `FileKindPlainFile` — a manually created file (strongest warning per FR-016b)

## 10. operation log types (`internal/oplog/`)

Operation logging (FR-036b, Could Have). Maps to `~/.config/nd/logs/operations.log`.

```go
package oplog

import (
    "time"

    "nd/internal/nd"
    "nd/internal/asset"
)

// LogEntry records a single nd operation for the operation log.
type LogEntry struct {
    Timestamp time.Time        `json:"timestamp"`
    Operation OperationType    `json:"operation"`
    Assets    []asset.Identity `json:"assets,omitempty"`
    Scope     nd.Scope         `json:"scope,omitempty"`
    Succeeded int              `json:"succeeded"`
    Failed    int              `json:"failed"`
    Detail    string           `json:"detail,omitempty"`
}

// OperationType categorizes log entries.
type OperationType string

const (
    OpDeploy        OperationType = "deploy"
    OpRemove        OperationType = "remove"
    OpSync          OperationType = "sync"
    OpProfileSwitch OperationType = "profile-switch"
    OpSnapshotSave  OperationType = "snapshot-save"
    OpSnapshotRestore OperationType = "snapshot-restore"
    OpSourceAdd     OperationType = "source-add"
    OpSourceRemove  OperationType = "source-remove"
    OpSourceSync    OperationType = "source-sync"
    OpUninstall     OperationType = "uninstall"
)
```

## 11. doctor report types (`internal/doctor/`)

Aggregate health check (FR-045). Prevents business logic from leaking into the CLI layer.

```go
package doctor

import (
    "nd/internal/config"
    "nd/internal/state"
)

// Report is the aggregate output of nd doctor.
// Each section corresponds to a check category.
type Report struct {
    Config      ConfigCheck      `json:"config"`
    Sources     []SourceCheck    `json:"sources"`
    Deployments []state.HealthCheck `json:"deployments"`
    Agents      []AgentCheck     `json:"agents"`
    Git         GitCheck         `json:"git"`
    Summary     Summary          `json:"summary"`
}

// ConfigCheck reports config file validation results.
type ConfigCheck struct {
    GlobalValid  bool                    `json:"global_valid"`
    ProjectValid bool                    `json:"project_valid"`
    Errors       []config.ValidationError `json:"errors,omitempty"`
}

// SourceCheck reports the accessibility and health of one source.
type SourceCheck struct {
    SourceID   string `json:"source_id"`
    Available  bool   `json:"available"`
    AssetCount int    `json:"asset_count"`
    Detail     string `json:"detail,omitempty"`
}

// AgentCheck reports whether an agent's directories exist and are writable.
type AgentCheck struct {
    AgentName  string `json:"agent_name"`
    Detected   bool   `json:"detected"`
    GlobalDir  string `json:"global_dir"`
    GlobalOK   bool   `json:"global_ok"`
    ProjectDir string `json:"project_dir,omitempty"`
    ProjectOK  bool   `json:"project_ok,omitempty"`
    Detail     string `json:"detail,omitempty"`
}

// GitCheck reports whether Git is available (needed for Git-sourced repos).
type GitCheck struct {
    Available bool   `json:"available"`
    Version   string `json:"version,omitempty"`
    Detail    string `json:"detail,omitempty"`
}

// Summary provides pass/warn/fail counts across all checks.
type Summary struct {
    Pass int `json:"pass"`
    Warn int `json:"warn"`
    Fail int `json:"fail"`
}
```

## 12. JSON output envelope

For the `--json` global flag. Used by all commands that support JSON output.

```go
// This lives in a thin package like internal/output/ or in each command handler.
// Shown here as a reference for the JSON output contract.

// JSONResponse is the standard envelope for all --json output.
type JSONResponse struct {
    Status string      `json:"status"` // "ok", "error", "partial"
    Data   interface{} `json:"data,omitempty"`
    Errors []JSONError `json:"errors,omitempty"`
}

// JSONError represents a single error in JSON output.
type JSONError struct {
    Code    string `json:"code"`              // Machine-readable error code
    Message string `json:"message"`           // Human-readable message
    Field   string `json:"field,omitempty"`   // Relevant field or path
}
```

## Type inventory

| Package | Types | On-disk format |
| --- | --- | --- |
| `internal/nd` | AssetType, Scope, DeployOrigin, SymlinkStrategy, SourceType, OriginalFileKind, PathTraversalError, LockError, ConflictError | (constants and errors only) |
| `internal/config` | Config, ProjectConfig, SourceEntry, AgentOverride, ValidationError | `config.yaml` |
| `internal/asset` | Identity, Asset, ContextInfo, ContextMeta, Index, Conflict, CachedIndex | `_meta.yaml`, `~/.cache/nd/index/*.yaml` |
| `internal/source` | Source, Manifest, ManifestMetadata, ScanResult | `nd-source.yaml` |
| `internal/state` | DeploymentState, Deployment, HealthStatus, HealthCheck, FileLock | `deployments.yaml` |
| `internal/profile` | Profile, ProfileAsset, Snapshot, SnapshotEntry, SwitchDiff | `profiles/*.yaml`, `snapshots/**/*.yaml` |
| `internal/agent` | Agent, DetectionResult | (none) |
| `internal/deploy` | Request, Result, Action, BulkResult, SyncPlan, SyncAction, UninstallPlan | (none) |
| `internal/backup` | Backup | (none) |
| `internal/oplog` | LogEntry, OperationType | `operations.log` |
| `internal/doctor` | Report, ConfigCheck, SourceCheck, AgentCheck, GitCheck, Summary | (none) |
| `internal/output` | JSONResponse, JSONError | (none) |

**Total:** 12 packages, 48 types, 7 on-disk formats.

## Audit remediation log

All 16 findings from the parallel audit were addressed:

| # | Finding | Fix |
| --- | --- | --- |
| 1 | `SnapshotEntry` missing `deployed_at` | Added `DeployedAt time.Time` field; updated snapshot YAML example |
| 2 | `ContextInfo.FileName` can't hold FR-034 custom types | Changed to `string`; `ContextFileName` constants are now plain `string` consts |
| 3 | Active profile not tracked in `DeploymentState` | Added `ActiveProfile string` field; updated deployments.yaml example |
| 4 | Cache type for `~/.cache/nd/index/` missing | Added `CachedIndex` type in `internal/asset/` |
| 5 | Import graph description wrong | Rewrote as leveled graph with explicit imports per package |
| 6 | Plugin exclusion not enforced in profiles/snapshots | Added validation notes to `Profile.Validate()` and `Snapshot.Validate()` |
| 7 | `SwitchDiff.Keep` equality missing scope | Added explicit comment: equality is (source_id, asset_type, asset_name, scope) |
| 8 | `DoctorReport` aggregate type missing | Added `internal/doctor/` package (section 11) |
| 9 | Domain error types missing | Added `PathTraversalError`, `LockError`, `ConflictError` to `internal/nd/` |
| 10 | `deploy` needs `agent` import | Added `agent` import and `Agent agent.Agent` field to `deploy.Request` |
| 11 | `ProjectConfig` missing `DefaultAgent` | Added `DefaultAgent *string` and `Agents []AgentOverride` fields |
| 12 | `Backup.WasManual` insufficient granularity | Replaced with `OriginalKind nd.OriginalFileKind` (3-value enum) |
| 13 | JSON output tags missing | Added `json:` tags to all persisted and API-facing types; added JSON envelope |
| 14 | Operation log type missing | Added `internal/oplog/` package (section 10) |
| 15 | `FileLock` type missing | Added `FileLock` type to `internal/state/` |
| 16 | `UninstallPlan` type missing | Added `UninstallPlan` type to `internal/deploy/` |
