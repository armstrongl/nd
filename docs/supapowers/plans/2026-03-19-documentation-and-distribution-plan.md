# Documentation & Distribution Pipeline Implementation Plan

> **For agentic workers:** REQUIRED: Use supapowers:subagent-driven-development (if subagents available) or supapowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship comprehensive user-facing documentation, contributor guides, and a working distribution pipeline (goreleaser + GitHub Actions + Homebrew tap) for the nd CLI tool.

**Architecture:** Four parallel tracks — distribution infrastructure (goreleaser, CI, release), contributor docs (CONTRIBUTING, ARCHITECTURE), command reference generator (gendocs utility), and user docs (README, 5 guide pages). Distribution must complete before the first release tag. User docs reference real install URLs.

**Tech Stack:** Go 1.25+, goreleaser v2, GitHub Actions, Cobra doc generation, Homebrew tap, Markdown.

**Spec:** `docs/supapowers/specs/2026-03-19-documentation-and-distribution-design.md`

---

## File Structure

| File | Responsibility | Action |
|------|---------------|--------|
| `.goreleaser.yaml` | Build config: binaries, archives, changelog, Homebrew formula | Create |
| `.github/workflows/ci.yml` | CI: lint, test, build, goreleaser check on push/PR | Create |
| `.github/workflows/release.yml` | Release: goreleaser on tag push | Create |
| `cmd/gendocs/main.go` | Utility to auto-generate command reference markdown | Create |
| `docs/reference/*.md` | Auto-generated command reference (one per command) | Generate |
| `README.md` | Project overview, install, quick-start, command table | Replace (stub exists) |
| `CONTRIBUTING.md` | Dev setup, testing, PR conventions, adding commands | Create |
| `ARCHITECTURE.md` | Package diagram, layers, patterns, data flow | Create |
| `docs/guide/getting-started.md` | Zero to first deploy in 5 minutes | Create |
| `docs/guide/user-guide.md` | Core workflows: sources, deploy, remove, sync, doctor | Create |
| `docs/guide/profiles-and-snapshots.md` | Profile switching, pinning, snapshots, restore | Create |
| `docs/guide/configuration.md` | config.yaml format, scoping, merging, defaults | Create |
| `docs/guide/creating-sources.md` | Directory conventions, _meta.yaml, manifests | Create |

**External:**
| Action | Where |
|--------|-------|
| Create `armstrongl/homebrew-tap` repo | GitHub (via `gh`) |
| Add `TAP_GITHUB_TOKEN` secret | GitHub repo settings (manual) |

---

## Chunk 1: Distribution Pipeline + Command Reference

### Task 1: Create goreleaser config

The goreleaser config builds nd for darwin/linux (amd64/arm64), creates archives, generates changelog from Conventional Commits, and pushes a Homebrew formula to the tap repo.

**Files:**
- Create: `.goreleaser.yaml`

**Key facts:**
- `internal/version/version.go` already has `var Version, Commit, Date` (strings, not const) — no changes needed
- Module path: `github.com/armstrongl/nd`
- goreleaser v2 uses `formats:` (plural, array) not `format:` (singular)
- Cross-repo Homebrew tap push requires `TAP_GITHUB_TOKEN` PAT

- [ ] **Step 1: Create `.goreleaser.yaml`**

```yaml
# yaml-language-server: $schema=https://goreleaser.com/static/schema.json
version: 2

builds:
  - main: .
    binary: nd
    goos: [darwin, linux]
    goarch: [amd64, arm64]
    ldflags:
      - -s -w
      - -X github.com/armstrongl/nd/internal/version.Version={{.Version}}
      - -X github.com/armstrongl/nd/internal/version.Commit={{.ShortCommit}}
      - -X github.com/armstrongl/nd/internal/version.Date={{.Date}}

archives:
  - formats: [tar.gz]
    files: [README.md, LICENSE]

checksum:
  algorithm: sha256

changelog:
  use: github
  groups:
    - title: Features
      regexp: '^feat'
    - title: Bug Fixes
      regexp: '^fix'
    - title: Other
      order: 999

# Note: If goreleaser check warns that 'brews' is deprecated, migrate to the
# replacement syntax per goreleaser's migration guide at that time.
brews:
  - repository:
      owner: armstrongl
      name: homebrew-tap
      token: "{{ .Env.TAP_GITHUB_TOKEN }}"
    homepage: https://github.com/armstrongl/nd
    description: Coding agent asset management CLI tool
    license: MIT
    install: bin.install "nd"
    test: system "#{bin}/nd", "version"
```

- [ ] **Step 2: Validate goreleaser config**

Run: `goreleaser check`
Expected: config is valid

If goreleaser is not installed: `go install github.com/goreleaser/goreleaser/v2@latest`

- [ ] **Step 3: Test local snapshot build**

Run: `goreleaser release --snapshot --clean`
Expected: Builds complete, `dist/` contains binaries for all 4 platform/arch combos. Verify binary runs: `./dist/nd_darwin_arm64/nd version`

- [ ] **Step 4: Clean up and commit**

Run:
```bash
rm -rf dist/
git add .goreleaser.yaml
git commit -m "ci: add goreleaser config for cross-platform builds and Homebrew tap"
```

---

### Task 2: Create GitHub Actions CI workflow

CI runs on every push to main and every PR. Lints, tests with race detection, verifies build, and validates goreleaser config.

**Files:**
- Create: `.github/workflows/ci.yml`

- [ ] **Step 1: Create directory structure**

Run: `mkdir -p .github/workflows`

- [ ] **Step 2: Create `.github/workflows/ci.yml`**

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

permissions:
  contents: read

jobs:
  lint:
    name: Lint
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.25"
      - uses: golangci/golangci-lint-action@v9
        with:
          version: latest

  test:
    name: Test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.25"
      - name: Run tests
        run: go test ./... -race -coverprofile=coverage.out
      - name: Upload coverage
        uses: actions/upload-artifact@v4
        with:
          name: coverage
          path: coverage.out

  build:
    name: Build
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.25"
      - name: Build
        run: go build -o /dev/null .
      - name: Validate goreleaser config
        uses: goreleaser/goreleaser-action@v6
        with:
          args: check
```

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: add GitHub Actions CI workflow (lint, test, build)"
```

---

### Task 3: Create Homebrew tap repository

goreleaser needs a target repo to push the Homebrew formula. This must exist before the first release.

- [ ] **Step 1: Create the tap repo on GitHub**

Run: `gh repo create armstrongl/homebrew-tap --public --description "Homebrew tap for nd CLI" --clone=false`
Expected: Repository created at `github.com/armstrongl/homebrew-tap`

- [ ] **Step 2: Initialize the tap repo with a README**

Run:
```bash
gh api repos/armstrongl/homebrew-tap/contents/README.md \
  --method PUT \
  -f message="Initial commit" \
  -f content="$(printf '# homebrew-tap\n\nHomebrew formulae for [nd](https://github.com/armstrongl/nd).\n\n## Install\n\n```bash\nbrew install armstrongl/tap/nd\n```\n' | base64)"
```

---

### Task 4: Create GitHub Actions release workflow

Triggers on version tags (`v*`). Runs goreleaser to build, create GitHub Release, and push Homebrew formula.

**Files:**
- Create: `.github/workflows/release.yml`

**Prerequisites:**
- `TAP_GITHUB_TOKEN` must be added as a repository secret (PAT with `repo` scope). This is a manual step — the plan will output instructions.

- [ ] **Step 1: Create `.github/workflows/release.yml`**

```yaml
name: Release

on:
  push:
    tags:
      - "v*"

permissions:
  contents: write

jobs:
  release:
    name: Release
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version: "1.25"
      - uses: goreleaser/goreleaser-action@v6
        with:
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          TAP_GITHUB_TOKEN: ${{ secrets.TAP_GITHUB_TOKEN }}
```

- [ ] **Step 2: Commit**

```bash
git add .github/workflows/release.yml
git commit -m "ci: add GitHub Actions release workflow (goreleaser on tag push)"
```

- [ ] **Step 3: Print manual step reminder**

Print to user:
```
MANUAL STEP REQUIRED:
1. Create a GitHub PAT with 'repo' scope at https://github.com/settings/tokens
2. Add it as a secret named TAP_GITHUB_TOKEN at:
   https://github.com/armstrongl/nd/settings/secrets/actions
```

---

### Task 5: Create command reference generator

Uses Cobra's built-in `doc.GenMarkdownTree()` to generate one markdown file per command. The `cobra/doc` package is a subpackage of the existing `github.com/spf13/cobra` dependency — no new module needed.

**Files:**
- Create: `cmd/gendocs/main.go`
- Generate: `docs/reference/*.md`

**Key facts:**
- `cmd.NewRootCmd(app *App)` requires an `*App` argument
- `cmd.App` is exported with public flag fields
- `github.com/spf13/cobra/doc` is a subpackage of cobra (already in go.mod as v1.10.2)

- [ ] **Step 1: Create gendocs directory**

Run: `mkdir -p cmd/gendocs`

- [ ] **Step 2: Create `cmd/gendocs/main.go`**

```go
// Command gendocs generates Markdown command reference from Cobra definitions.
package main

import (
	"log"
	"os"

	"github.com/armstrongl/nd/cmd"
	"github.com/spf13/cobra/doc"
)

func main() {
	outDir := "docs/reference"
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatal(err)
	}

	rootCmd := cmd.NewRootCmd(&cmd.App{})
	rootCmd.DisableAutoGenTag = true

	if err := doc.GenMarkdownTree(rootCmd, outDir); err != nil {
		log.Fatal(err)
	}
}
```

- [ ] **Step 3: Pull cobra/doc transitive dependency**

The `cobra/doc` subpackage imports `go-md2man/v2` which may not be in `go.sum` yet.

Run: `go get github.com/spf13/cobra/doc && go mod tidy`
Expected: Dependencies updated, go.sum updated.

- [ ] **Step 4: Verify it compiles**

Run: `go build ./cmd/gendocs/`
Expected: Compiles without errors.

- [ ] **Step 5: Generate the command reference**

Run: `go run ./cmd/gendocs/`
Expected: `docs/reference/` is populated with markdown files (one per command).

Verify: `ls docs/reference/ | head -10` — should show files like `nd.md`, `nd_deploy.md`, `nd_source_add.md`, etc.

- [ ] **Step 6: Commit**

```bash
git add cmd/gendocs/ docs/reference/ go.mod go.sum
git commit -m "docs: add command reference generator and auto-generated reference docs"
```

---

## Chunk 2: Contributor Docs + README

### Task 6: Write CONTRIBUTING.md

**Files:**
- Create: `CONTRIBUTING.md`

- [ ] **Step 1: Create `CONTRIBUTING.md`**

```markdown
# Contributing to nd

Thank you for considering contributing to nd! This guide will help you get started.

## Prerequisites

- [Go](https://go.dev/dl/) 1.25+
- [git](https://git-scm.com/)
- [golangci-lint](https://golangci-lint.run/) v2: `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`
- [gofumpt](https://github.com/mvdan/gofumpt): `go install mvdan.cc/gofumpt@latest`

## Getting Started

```bash
git clone https://github.com/armstrongl/nd.git
cd nd
go test ./...
go build -o nd .
./nd version
```

## Development Workflow

1. Create a branch from `main`
2. Write tests first (TDD -- red/green/refactor)
3. Implement the feature or fix
4. Run the linter: `golangci-lint run`
5. Format with gofumpt: `gofumpt -w .`
6. Commit with a Conventional Commit message
7. Open a pull request

## Testing

```bash
# Run all unit tests
go test ./...

# Run a specific test
go test ./internal/deploy/ -run TestDeploySymlink -v

# Run with race detection
go test -race ./...

# Run integration tests
go test ./tests/integration/ -v

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

**Coverage expectations:** 80%+ for business logic packages (`internal/`). Lower coverage is acceptable for CLI (`cmd/`) and TUI (`internal/tui/`) due to interactive code paths.

## Adding a New CLI Command

1. Create `cmd/foo.go` with a `func newFooCmd(app *App) *cobra.Command` function
2. Create `cmd/foo_test.go` with tests (write tests first)
3. Register in `cmd/root.go` via `rootCmd.AddCommand(newFooCmd(app))`
4. Follow existing command patterns (see `cmd/deploy.go` as a reference)
5. Add shell completion if the command takes arguments
6. Regenerate command reference: `go run ./cmd/gendocs/`

## Code Style

- **Formatter:** gofumpt (strict superset of gofmt)
- **Linter:** golangci-lint v2 with default configuration
- **Commits:** [Conventional Commits](https://www.conventionalcommits.org/) required

## Commit Messages

Format: `type(scope): description`

**Types:** `feat`, `fix`, `refactor`, `test`, `docs`, `style`, `ci`, `chore`

**Scopes:** `cli`, `deploy`, `profile`, `source`, `agent`, `config`, `tui`, `docs`, `ci`

Examples:
- `feat(cli): add interactive picker to deploy`
- `fix(deploy): handle broken symlinks on sync`
- `docs: update getting started guide`
- `ci: add coverage upload to CI workflow`

## Pull Requests

- One feature or fix per PR
- Tests required for all code changes
- CI must pass (lint + test + build)
- Reference the issue number if applicable
- Keep PRs focused and reviewable

## Project Structure

See [ARCHITECTURE.md](ARCHITECTURE.md) for a detailed overview of the codebase structure, package responsibilities, and key patterns.
```

- [ ] **Step 2: Commit**

```bash
git add CONTRIBUTING.md
git commit -m "docs: add CONTRIBUTING.md with dev setup, testing, and PR conventions"
```

---

### Task 7: Write ARCHITECTURE.md

**Files:**
- Create: `ARCHITECTURE.md`

- [ ] **Step 1: Create `ARCHITECTURE.md`**

```markdown
# Architecture

This document describes the internal architecture of nd for contributors and maintainers.

## Overview

nd is a CLI/TUI tool that manages coding agent assets (skills, agents, commands, output-styles, rules, context files, plugins, hooks) via symlink-based deployment. It is built in Go with [Cobra](https://github.com/spf13/cobra) for the CLI and [Bubble Tea](https://charm.land/bubbletea/) for the TUI.

## Layered Architecture

```
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
- `SourceType` -- `local` (directory) or `git` (repository)
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
- `Manifest` -- Convention-based directory structure discovered from a source

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
- `ScanSource()` -- Convention-based + manifest discovery of assets
- Git operations: clone, pull (--ff-only)

### internal/deploy

Symlink deployment engine.

- `Engine` -- Deploy, remove, health check, repair, bulk operations
- `Deploy()` -- Create symlink from agent config dir to source asset
- `Remove()` -- Delete managed symlinks
- `Check()` -- Health check: broken symlinks, drift detection
- `Sync()` -- Repair broken/drifted symlinks
- `Uninstall()` -- Remove all managed symlinks

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

```
cmd/deploy.go
  -> app.SourceManager().Scan()     -- discover assets from all sources
  -> asset.Index.Resolve()          -- find "skills/greeting" in index
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
```

- [ ] **Step 2: Commit**

```bash
git add ARCHITECTURE.md
git commit -m "docs: add ARCHITECTURE.md with package diagram, layers, and key patterns"
```

---

### Task 8: Write README.md

The README is the main entry point. It replaces the existing 2-line stub. Install instructions reference real URLs (Homebrew tap, GitHub Releases, go install).

**Files:**
- Replace: `README.md`

- [ ] **Step 1: Replace `README.md`**

Note: The existing README.md is a 2-line stub. Replace it entirely.

```markdown
# nd

[![CI](https://github.com/armstrongl/nd/actions/workflows/ci.yml/badge.svg)](https://github.com/armstrongl/nd/actions/workflows/ci.yml)
[![Go](https://img.shields.io/github/go-mod/go-version/armstrongl/nd)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/armstrongl/nd)](https://github.com/armstrongl/nd/releases)

Manage coding agent assets (skills, agents, commands, rules, and more) across tools like Claude Code with symlink-based deployment.

## What It Does

- **Register sources** -- Point nd at local directories or git repos containing agent assets
- **Deploy assets** -- Create symlinks from agent config directories to source assets, keeping everything in sync
- **Switch profiles** -- Group assets into named profiles and switch between them instantly
- **Save snapshots** -- Capture and restore deployment states as point-in-time snapshots

## Installation

### Homebrew (macOS/Linux)

```bash
brew install armstrongl/tap/nd
```

### Go Install

```bash
go install github.com/armstrongl/nd@latest
```

### GitHub Releases

Download pre-built binaries from [Releases](https://github.com/armstrongl/nd/releases).

### Build from Source

```bash
git clone https://github.com/armstrongl/nd.git
cd nd
go build -o nd .
```

## Quick Start

```bash
# Initialize nd configuration
nd init

# Register an asset source (local directory or git repo)
nd source add ~/my-assets
# or: nd source add github-user/asset-repo

# See available assets
nd list

# Deploy an asset
nd deploy skills/greeting

# Check deployment status
nd status
```

## Commands

| Command | Description |
|---------|-------------|
| `nd init` | Initialize nd configuration |
| `nd source add` | Register a local directory or git repo as an asset source |
| `nd source remove` | Remove a registered source |
| `nd source list` | List all registered sources |
| `nd list` | List all available assets from all sources |
| `nd deploy` | Deploy assets by creating symlinks |
| `nd remove` | Remove deployed assets |
| `nd status` | Show deployment status and health |
| `nd pin` / `nd unpin` | Pin/unpin assets to persist across profile switches |
| `nd sync` | Repair symlinks and pull git sources |
| `nd doctor` | Health check: validate config, sources, deployments |
| `nd profile create` | Create a named profile (asset collection) |
| `nd profile switch` | Switch between profiles |
| `nd profile deploy` | Deploy all assets from a profile |
| `nd profile delete` | Delete a profile |
| `nd profile add-asset` | Add an asset to an existing profile |
| `nd profile list` | List all profiles |
| `nd snapshot save` | Save current deployments as a snapshot |
| `nd snapshot restore` | Restore deployments from a snapshot |
| `nd snapshot list` | List all snapshots |
| `nd snapshot delete` | Delete a snapshot |
| `nd settings edit` | Open config file in your editor |
| `nd uninstall` | Remove all nd-managed symlinks |
| `nd version` | Print version information |
| `nd completion` | Generate shell completions (bash, zsh, fish) |

Run any command with `--help` for detailed usage, or see the full [Command Reference](docs/reference/nd.md).

Many commands support **interactive mode** -- run without arguments to get a picker. Use `--json` for scripted output and `--yes` to skip confirmations.

## Configuration

nd uses a YAML config file at `~/.config/nd/config.yaml`. Key settings:

```yaml
version: 1
default_scope: global       # or "project"
default_agent: claude-code
symlink_strategy: absolute  # or "relative"
```

See the full [Configuration Guide](docs/guide/configuration.md).

## Documentation

- [Getting Started](docs/guide/getting-started.md) -- Install to first deploy in 5 minutes
- [User Guide](docs/guide/user-guide.md) -- Core workflows: sources, deploying, syncing
- [Profiles & Snapshots](docs/guide/profiles-and-snapshots.md) -- Advanced workflow management
- [Configuration](docs/guide/configuration.md) -- Full config reference
- [Creating Sources](docs/guide/creating-sources.md) -- Build your own asset library
- [Command Reference](docs/reference/nd.md) -- Auto-generated from source

## Contributing

Contributions are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, testing, and PR guidelines.

For architecture details, see [ARCHITECTURE.md](ARCHITECTURE.md).

## License

[MIT](LICENSE)
```

- [ ] **Step 2: Commit**

```bash
git add README.md
git commit -m "docs: replace README stub with full project documentation"
```

---

## Chunk 3: User Guide Pages

### Task 9: Write Getting Started guide

**Files:**
- Create: `docs/guide/getting-started.md`

- [ ] **Step 1: Create directory**

Run: `mkdir -p docs/guide`

- [ ] **Step 2: Create `docs/guide/getting-started.md`**

```markdown
# Getting Started

This guide takes you from zero to your first deployed asset in about 5 minutes.

## 1. Install nd

Choose your preferred method:

```bash
# Homebrew (macOS/Linux)
brew install armstrongl/tap/nd

# Go install
go install github.com/armstrongl/nd@latest

# Or build from source
git clone https://github.com/armstrongl/nd.git && cd nd && go build -o nd .
```

Verify the installation:

```bash
nd version
```

## 2. Initialize

Create the nd configuration directory and default config:

```bash
nd init
```

This creates `~/.config/nd/config.yaml` with sensible defaults and sets up directories for profiles, snapshots, and state.

## 3. Add Your First Source

A **source** is a local directory or git repository containing agent assets organized by type.

```bash
# Local directory
nd source add ~/my-coding-assets

# Git repository (GitHub shorthand)
nd source add owner/repo

# Git repository (full URL)
nd source add https://github.com/owner/repo.git
```

nd scans the source for assets organized in convention-based directories (`skills/`, `agents/`, `commands/`, etc.). See [Creating Sources](creating-sources.md) for how to structure your own.

## 4. Browse Available Assets

List all assets discovered from your sources:

```bash
nd list
```

Filter by type:

```bash
nd list --type skills
```

Assets marked with `*` are already deployed.

## 5. Deploy an Asset

Deploy an asset by creating a symlink in your agent's config directory:

```bash
nd deploy skills/greeting
```

Deploy multiple assets at once:

```bash
nd deploy skills/greeting commands/hello agents/researcher
```

Or run `nd deploy` with no arguments to get an interactive picker.

## 6. Verify

Check that everything is healthy:

```bash
nd status
```

You should see your deployed assets with health indicators (checkmarks for healthy symlinks).

For a deeper health check of your entire setup:

```bash
nd doctor
```

## 7. Optional Setup

### Shell Completions

Enable tab-completion for your shell:

```bash
# Bash
nd completion bash --install

# Zsh
nd completion zsh --install

# Fish
nd completion fish --install
```

For zsh, you may need to add this to your `~/.zshrc` if not already present:

```bash
fpath+=~/.zfunc
autoload -Uz compinit && compinit
```

### Edit Configuration

Open your config file in your default editor:

```bash
nd settings edit
```

## Next Steps

- **[User Guide](user-guide.md)** -- Learn about managing sources, scopes, syncing, and more
- **[Profiles & Snapshots](profiles-and-snapshots.md)** -- Group assets into profiles and switch between them
- **[Configuration](configuration.md)** -- Customize nd behavior
- **[Creating Sources](creating-sources.md)** -- Build and share your own asset libraries
- **TUI Dashboard** -- Run `nd` with no arguments to launch the interactive dashboard
```

- [ ] **Step 3: Commit**

```bash
git add docs/guide/getting-started.md
git commit -m "docs: add Getting Started guide"
```

---

### Task 10: Write User Guide

**Files:**
- Create: `docs/guide/user-guide.md`

- [ ] **Step 1: Create `docs/guide/user-guide.md`**

```markdown
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

Asset references use the format `type/name`. If the name is unique across types, you can omit the type: `nd deploy greeting`.

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
```

- [ ] **Step 2: Commit**

```bash
git add docs/guide/user-guide.md
git commit -m "docs: add User Guide with core workflows"
```

---

### Task 11: Write Profiles & Snapshots guide

**Files:**
- Create: `docs/guide/profiles-and-snapshots.md`

- [ ] **Step 1: Create `docs/guide/profiles-and-snapshots.md`**

```markdown
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
```

- [ ] **Step 2: Commit**

```bash
git add docs/guide/profiles-and-snapshots.md
git commit -m "docs: add Profiles & Snapshots guide"
```

---

### Task 12: Write Configuration guide

**Files:**
- Create: `docs/guide/configuration.md`

- [ ] **Step 1: Create `docs/guide/configuration.md`**

```markdown
# Configuration

nd uses YAML configuration files with a layered merging system.

## Config File Locations

| Location | Path | Purpose |
|----------|------|---------|
| Global | `~/.config/nd/config.yaml` | User-wide settings and sources |
| Project | `.nd/config.yaml` | Project-specific overrides |
| CLI flag | `--config <path>` | One-time override |

The global config is created by `nd init`. Project-level config is optional.

## Full Annotated Example

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

# Registered asset sources
sources:
  - id: my-assets
    type: local
    path: ~/coding-assets

  - id: community
    type: git
    url: https://github.com/org/shared-assets.git
    alias: community-assets

# Agent configuration overrides (optional)
# Only needed if your agent uses non-standard directories
agents:
  - name: claude-code
    global_dir: ~/.claude
    project_dir: .claude
```

## Config Key Reference

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `version` | integer | `1` | Config schema version |
| `default_scope` | string | `global` | Default deployment scope |
| `default_agent` | string | `claude-code` | Default agent to target |
| `symlink_strategy` | string | `absolute` | Symlink type: `absolute` or `relative` |
| `sources` | array | `[]` | Registered asset sources |
| `sources[].id` | string | (generated) | Unique source identifier |
| `sources[].type` | string | -- | Source type: `local` or `git` |
| `sources[].path` | string | -- | Filesystem path to source |
| `sources[].url` | string | -- | Git URL (git sources only) |
| `sources[].alias` | string | -- | Human-readable alias (optional) |
| `agents` | array | (built-in) | Agent configuration overrides |
| `agents[].name` | string | -- | Agent name |
| `agents[].global_dir` | string | -- | Agent's global config directory |
| `agents[].project_dir` | string | -- | Agent's project config directory |

## Config Merging

nd merges configuration from multiple sources in this order (later overrides earlier):

1. **Built-in defaults** -- Sensible defaults for all settings
2. **Global config** -- `~/.config/nd/config.yaml`
3. **Project config** -- `.nd/config.yaml` (if present)
4. **CLI flags** -- `--scope`, `--config`, etc.

For sources, global sources appear first (higher priority), followed by project sources. This means if the same asset exists in both a global and project source, the global source wins.

## Project-Level Config

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

## Environment Variables

| Variable | Used By | Description |
|----------|---------|-------------|
| `$EDITOR` | `nd settings edit` | Preferred text editor |
| `$VISUAL` | `nd settings edit` | Visual editor (fallback if `$EDITOR` not set) |

If neither is set, `nd settings edit` falls back to `vi`.

## Editing Config

Open your config in your default editor:

```bash
nd settings edit
```

After editing, validate your config:

```bash
nd doctor
```

The doctor command checks config validity as its first step.
```

- [ ] **Step 2: Commit**

```bash
git add docs/guide/configuration.md
git commit -m "docs: add Configuration guide"
```

---

### Task 13: Write Creating Sources guide

**Files:**
- Create: `docs/guide/creating-sources.md`

- [ ] **Step 1: Create `docs/guide/creating-sources.md`**

```markdown
# Creating Asset Sources

An asset source is a directory (local or git) containing coding agent assets organized by type. This guide explains how to structure your own.

## Directory Convention

nd discovers assets by looking for directories named after asset types:

```
my-assets/
+-- skills/
|   +-- greeting/           # Directory asset
|   +-- code-review/        # Directory asset
+-- agents/
|   +-- researcher.md       # File asset
+-- commands/
|   +-- deploy-all.md       # File asset
+-- output-styles/
|   +-- concise.md          # File asset
+-- rules/
|   +-- no-emojis.md        # File asset
+-- context/
|   +-- CLAUDE.md           # File asset (special deploy rules)
+-- plugins/
|   +-- my-plugin/          # Directory asset (not symlink-deployed)
+-- hooks/
    +-- pre-commit/         # Directory asset
```

Not every directory needs to be present. nd only discovers assets in directories that exist.

## Asset Types

| Type | Format | Deployable | Description |
|------|--------|------------|-------------|
| `skills` | Directory | Yes | Multi-file skill definitions |
| `agents` | File | Yes | Agent configuration files |
| `commands` | File | Yes | Custom command definitions |
| `output-styles` | File | Yes | Output formatting styles (requires manual settings.json registration) |
| `rules` | File | Yes | Rule files for agent behavior |
| `context` | File | Yes | Context files (special deployment rules -- see below) |
| `plugins` | Directory | No | Plugin packages (uses export workflow, not symlinks) |
| `hooks` | Directory | Yes | Hook definitions (requires manual settings.json registration) |

## Context Files

Context files have special deployment rules:

- **Global scope:** Deployed to the agent's global directory (e.g., `~/.claude/CLAUDE.md`)
- **Project scope:** Deployed to the project root directly (e.g., `./CLAUDE.md`), not inside `.claude/`
- **Local files** (`*.local.md`): Can only be deployed at project scope

### _meta.yaml

Context files can include a `_meta.yaml` sidecar for metadata:

```yaml
description: "Project coding standards and conventions"
tags: ["standards", "conventions"]
```

## Manifest File

For sources that don't follow the convention-based directory structure, create an `nd-source.yaml` manifest at the source root:

```yaml
# nd-source.yaml
skills:
  - path: custom/path/to/skills
agents:
  - path: my-agents
```

This tells nd where to find each asset type. Convention-based directories are still discovered alongside manifest paths.

## Publishing Your Source

Sharing your asset source is as simple as pushing to git:

```bash
cd my-assets
git init
git add .
git commit -m "Initial asset collection"
git remote add origin https://github.com/you/my-assets.git
git push -u origin main
```

Others can add it with:

```bash
nd source add you/my-assets
# or
nd source add https://github.com/you/my-assets.git
```

Git sources are cloned to `~/.config/nd/sources/` and can be synced with `nd sync --source <id>`.
```

- [ ] **Step 2: Commit**

```bash
git add docs/guide/creating-sources.md
git commit -m "docs: add Creating Sources guide"
```

---

## Chunk 4: Release and Verification

### Task 14: Push all documentation and create first release

All documentation and infrastructure is committed. Push to remote, verify CI passes, and tag the first release.

- [ ] **Step 1: Push to remote**

Run: `git push origin main`

- [ ] **Step 2: Verify CI passes**

Run: `gh run watch` (or check GitHub Actions UI)
Expected: All CI jobs pass (lint, test, build, goreleaser check)

If CI fails, fix the issue, commit, and push again.

- [ ] **Step 3: Verify TAP_GITHUB_TOKEN secret is set**

Check: `gh secret list` should show `TAP_GITHUB_TOKEN`.
If not set, the user must add it manually (see Task 4, Step 3).

- [ ] **Step 4: Tag and push first release**

Run:
```bash
git tag -a v0.1.0 -m "v0.1.0: initial release with full documentation and distribution"
git push origin v0.1.0
```

- [ ] **Step 5: Verify release workflow**

Run: `gh run watch` (for the release workflow triggered by the tag)
Expected: goreleaser builds, creates GitHub Release, pushes Homebrew formula.

Verify:
```bash
# Check GitHub Release exists
gh release view v0.1.0

# Check Homebrew formula was pushed
gh api repos/armstrongl/homebrew-tap/contents/Formula/nd.rb --jq .name
```

- [ ] **Step 6: Verify install methods**

```bash
# Test Homebrew (if on macOS)
brew install armstrongl/tap/nd
nd version

# Test go install
go install github.com/armstrongl/nd@v0.1.0
nd version
```
