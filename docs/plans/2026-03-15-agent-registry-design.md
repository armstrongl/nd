# Agent Registry design

| Field | Value |
| --- | --- |
| **Date** | 2026-03-15 |
| **Author** | Larah |
| **Status** | Approved |
| **Package** | `internal/agent/` |
| **Spec refs** | FR-016, FR-033, A6 |

## Overview

The Agent Registry is a service layer that detects installed coding agents, applies config overrides, and provides agent lookup for the rest of nd. In v1, it knows about Claude Code only. The registry lives in the existing `internal/agent/` package alongside the `Agent` data type.

## Architecture

A `Registry` struct in `internal/agent/registry.go` that:

1. Holds a list of known agent definitions (hardcoded Claude Code in v1)
2. Applies `AgentOverride` from config to customize paths (FR-033)
3. Detects which agents are actually installed (PATH + config dir) (FR-016)
4. Provides lookup by name and a default agent accessor

The registry is created once at startup and reused throughout the session.

### Dependencies

- **Inputs**: `config.Config` (for `DefaultAgent`, `Agents` overrides)
- **External checks**: filesystem (`os.Stat`) and PATH (`exec.LookPath`)
- **Testability**: Filesystem and PATH checks are injected as function fields on the Registry struct, so tests can stub them without touching the real filesystem

## API surface

```go
// Registry manages agent detection, lookup, and config override application.
type Registry struct {
    agents     []Agent
    defaultIdx int
    detected   bool
    lookPath   func(string) (string, error)  // injected, defaults to exec.LookPath
    stat       func(string) (os.FileInfo, error) // injected, defaults to os.Stat
}

// New creates a Registry with known agent definitions and applies config overrides.
func New(cfg config.Config) *Registry

// Detect probes the system for installed agents (PATH + config dir).
// Populates Agent.Detected and Agent.InPath fields. Safe to call multiple times
// (subsequent calls are no-ops). Returns DetectionResult with agents and warnings.
func (r *Registry) Detect() DetectionResult

// Get returns the agent with the given name, or an error if not found.
func (r *Registry) Get(name string) (*Agent, error)

// Default returns the default agent: the one named in config.DefaultAgent if
// it's detected, otherwise the first detected agent, otherwise an error.
func (r *Registry) Default() (*Agent, error)

// All returns all known agents (detected or not).
func (r *Registry) All() []Agent
```

## Detection logic

For each known agent (Claude Code in v1):

1. **PATH check**: Call `lookPath("claude")` — sets `InPath = true` if found
2. **Config dir check**: Call `stat(globalDir)` — sets `Detected = true` if the directory exists OR if `InPath` is true
3. **Warning generation**: If neither PATH nor config dir is found, generate warning: "No coding agents detected. Install Claude Code or configure a custom agent path in config.yaml (see: nd settings edit)."

An agent is considered detected if EITHER the binary is in PATH OR the config directory exists. When Claude Code is in PATH but `~/.claude/` doesn't exist yet (fresh install, never run), nd treats it as detected. Directory creation is the deploy engine's responsibility, not the registry's.

## Config override application

During `New()`:

- Start with hardcoded Claude Code defaults: `GlobalDir: ~/.claude`, `ProjectDir: .claude`, binary name `claude`
- For each `AgentOverride` in config that matches by name, overwrite GlobalDir and ProjectDir
- Expand `~` in paths to the user's home directory

## Error handling

- `Get()` with unknown name returns `fmt.Errorf("agent %q not found", name)`
- `Default()` with no detected agents returns `fmt.Errorf("no coding agents detected...")`
- Detection failures (e.g., permission errors on `stat`) are logged as warnings in `DetectionResult`, not fatal errors

## Testing strategy

Unit tests with injected `lookPath` and `stat` stubs — no real filesystem needed.

### Test scenarios

| Scenario | InPath | Dir exists | Detected | Warning |
| --- | --- | --- | --- | --- |
| PATH + dir | true | true | true | none |
| PATH + no dir | true | false | true | none |
| No PATH + dir | false | true | true | none |
| No PATH + no dir | false | false | false | yes |

Additional scenarios:

- Config override changes GlobalDir — detection uses overridden path
- Default agent selection: config-named > first-detected > error
- Get by name: found vs not found
- Multiple `Detect()` calls are idempotent
- `~` expansion in override paths

Target: >90% test coverage.

## Files

| File | Purpose |
| --- | --- |
| `internal/agent/registry.go` | Registry struct, New, Detect, Get, Default, All |
| `internal/agent/registry_test.go` | Unit tests with injected stubs |

No changes needed to `agent.go` or `agent_test.go` — existing data types and DeployPath tests are unaffected.
