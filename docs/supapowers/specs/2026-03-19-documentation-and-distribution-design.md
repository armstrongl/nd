# Documentation & Distribution Pipeline Design

**Date:** 2026-03-19
**Status:** Draft
**Audience:** Open-source community (developers managing Claude Code assets)

## Audit Remediations Applied

This spec was audited by three parallel agents (technical accuracy, completeness, risks). The following issues were found and fixed:

| Issue | Severity | Fix |
|-------|----------|-----|
| `cmd.NewRootCmd()` called with no args; actual signature requires `*App` | Critical | Fixed gendocs to pass `&cmd.App{}` |
| `brews:` deprecated in goreleaser v2 | Critical | Kept `brews:` (still valid in v2; `homebrew_casks` is for casks not formulas); added `token` field |
| Default GITHUB_TOKEN cannot push to cross-repo Homebrew tap | Critical | Added `TAP_GITHUB_TOKEN` PAT requirement in goreleaser config and release workflow |
| `archives.format` (singular) deprecated in goreleaser v2 | Significant | Changed to `formats: [tar.gz]` |
| `settings edit`, `profile add-asset`, `profile delete` undocumented | Significant | Added to user guide and profiles guide |
| TUI not documented | Significant | Added as explicit non-goal (TUI still evolving) |
| Shell completions only mentioned for zsh | Significant | Expanded to cover all 3 shells (bash/zsh/fish) in guide |
| CONTRIBUTING missing "run specific test" and "add new command" | Significant | Added both sections |
| Directory example shows only 4 of 8 asset types | Minor | Expanded to all 8 |
| Makefile in non-goals but `make docs` target defined | Minor | Removed Makefile target; use `go run` directly |
| No goreleaser config validation in CI | Minor | Added `goreleaser check` step to ci.yml |
| Dependency order implies CONTRIBUTING/ARCHITECTURE must wait for release | Minor | Restructured into parallel tracks |
| Interactive pickers not explained as general concept | Minor | Added "Interactive mode" section to user guide |
| `--yes` flag undocumented in guides | Minor | Added "Global flags for scripting" section to user guide |
| README.md already exists (stub) | Minor | Noted in dependency order |

---

## Goal

Ship comprehensive user-facing documentation, contributor guides, and a working distribution pipeline (goreleaser + GitHub Actions + Homebrew tap) for the nd CLI tool.

## Deliverables

| Deliverable | Path | Purpose |
|-------------|------|---------|
| README.md | `README.md` | Project overview, install, quick-start, command table, links |
| CONTRIBUTING.md | `CONTRIBUTING.md` | Dev setup, testing, PR/commit conventions |
| ARCHITECTURE.md | `ARCHITECTURE.md` | Package diagram, layers, patterns, data flow |
| Getting Started guide | `docs/guide/getting-started.md` | Install to first deploy in 5 minutes |
| User Guide | `docs/guide/user-guide.md` | Core workflows: sources, deploy, remove, sync, doctor |
| Profiles & Snapshots guide | `docs/guide/profiles-and-snapshots.md` | Profile switching, pinning, snapshots, restore |
| Configuration guide | `docs/guide/configuration.md` | config.yaml format, scoping, merging, defaults |
| Creating Sources guide | `docs/guide/creating-sources.md` | Directory conventions, _meta.yaml, manifests |
| Command Reference | `docs/reference/*.md` | Auto-generated from Cobra (one file per command) |
| Doc generator | `cmd/gendocs/main.go` | Utility to regenerate command reference |
| goreleaser config | `.goreleaser.yaml` | Build, archive, Homebrew tap, changelog |
| CI workflow | `.github/workflows/ci.yml` | Lint, test, build on PR/push |
| Release workflow | `.github/workflows/release.yml` | goreleaser on tag push |
| Homebrew tap repo | `armstrongl/homebrew-tap` | Created on GitHub for formula publishing |

---

## 1. README.md

### Outline

1. **Project name + badges** — CI status, Go version, license, latest release
2. **One-liner** — "Manage coding agent assets (skills, agents, commands, rules, and more) across tools like Claude Code with symlink-based deployment."
3. **What it does** (3-4 bullets):
   - Register local directories or git repos as asset sources
   - Deploy/remove assets via symlinks to agent config directories
   - Switch between named profiles of curated asset collections
   - Save and restore deployment snapshots
4. **Installation** — Homebrew, `go install`, GitHub Releases, build from source
5. **Quick Start** — 5-step walkthrough:
   ```
   nd init
   nd source add ~/my-assets
   nd list
   nd deploy skills/greeting
   nd status
   ```
6. **Command overview table** — Name | Description (links to docs/reference/)
7. **Configuration** — Key settings, link to full guide
8. **Documentation** — Links to guide pages and reference
9. **Contributing** — Link to CONTRIBUTING.md
10. **License** — MIT

### Constraints

- Target length: 150-250 lines
- No deep dives — link to guide pages for detail
- Install instructions must work (depends on distribution pipeline being set up first)

---

## 2. Distribution Pipeline

### goreleaser (.goreleaser.yaml)

```yaml
# Key configuration decisions:
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

### ldflags integration

The `internal/version` package must expose `Version`, `Commit`, and `Date` as `var` (not `const`) so goreleaser can inject values at build time via `-ldflags`. The `nd version` command reads these.

### GitHub Actions

**ci.yml** (triggers: push to main, pull requests):
1. Checkout
2. Setup Go 1.25.x
3. golangci-lint v2 (uses `golangci/golangci-lint-action`)
4. `go test ./... -race -coverprofile=coverage.out`
5. `go build -o /dev/null .` (verify compilation)
6. `goreleaser check` (validate goreleaser config syntax)

**release.yml** (triggers: push tag `v*`):
1. Checkout with `fetch-depth: 0` (goreleaser needs full history for changelog)
2. Setup Go 1.25.x
3. `goreleaser release --clean` (uses `goreleaser/goreleaser-action`)
4. Requires `TAP_GITHUB_TOKEN` secret (PAT with `repo` scope) for cross-repo Homebrew tap push; set via `env: TAP_GITHUB_TOKEN: ${{ secrets.TAP_GITHUB_TOKEN }}`

### Homebrew tap

- Create `armstrongl/homebrew-tap` repo on GitHub (public, with README)
- goreleaser auto-pushes formula on release
- Users install with: `brew install armstrongl/tap/nd`

---

## 3. CONTRIBUTING.md

### Sections

1. **Prerequisites**
   - Go 1.25+
   - git
   - golangci-lint v2 (`go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`)
   - gofumpt (`go install mvdan.cc/gofumpt@latest`)

2. **Getting started**
   ```
   git clone https://github.com/armstrongl/nd.git
   cd nd
   go test ./...
   go build -o nd .
   ```

3. **Development workflow**
   - Branch from `main`
   - Write tests first (TDD — red/green/refactor)
   - Run linter before committing: `golangci-lint run`
   - Format with gofumpt: `gofumpt -w .`

4. **Testing**
   - Unit tests: `go test ./...`
   - Run a specific test: `go test ./internal/deploy/ -run TestDeploySymlink -v`
   - Race detection: `go test -race ./...`
   - Integration tests: `go test ./tests/integration/ -v`
   - Coverage: `go test -coverprofile=coverage.out ./...`
   - Coverage expectations: 80%+ for business logic packages

5. **Adding a new CLI command**
   - Create `cmd/foo.go` with `func newFooCmd(app *App) *cobra.Command`
   - Create `cmd/foo_test.go` with tests (write tests first — TDD)
   - Register in `cmd/root.go` via `rootCmd.AddCommand(newFooCmd(app))`
   - Follow existing command patterns (see `cmd/deploy.go` as reference)
   - Add shell completion if the command takes arguments

6. **Code style**
   - Formatter: gofumpt (superset of gofmt)
   - Linter: golangci-lint v2
   - Conventional Commits required
   - Scopes: `cli`, `deploy`, `profile`, `source`, `agent`, `config`, `tui`, `docs`, `ci`

7. **Commit messages**
   - Format: `type(scope): description`
   - Types: `feat`, `fix`, `refactor`, `test`, `docs`, `style`, `ci`, `chore`
   - Examples: `feat(cli): add interactive picker to deploy`, `fix(deploy): handle broken symlinks on sync`

8. **Pull requests**
   - One feature/fix per PR
   - Tests required for all changes
   - CI must pass (lint + test + build)
   - Reference issue number if applicable

9. **Project structure** — Pointer to ARCHITECTURE.md

---

## 4. ARCHITECTURE.md

### Sections

1. **Overview**
   - nd manages coding agent assets via symlink deployment
   - Built in Go with Cobra (CLI) and Bubble Tea (TUI)
   - Layered architecture: CLI/TUI → services → core types

2. **Package diagram**
   ```
   ┌─────────────────────────────────────────────┐
   │                 cmd/ (CLI)                   │
   │           internal/tui/ (TUI)               │
   ├─────────────────────────────────────────────┤
   │  sourcemanager │  deploy  │ profile │ agent │  ← Services
   ├─────────────────────────────────────────────┤
   │   nd  │ config │ asset │ source │ state     │  ← Core types
   └─────────────────────────────────────────────┘
   ```

3. **Layer descriptions** — Each package: purpose, key types, public API
   - `internal/nd` — Core enums: AssetType, Scope, SourceType, SymlinkStrategy, Origin
   - `internal/config` — Config struct, SourceEntry, validation, merging
   - `internal/asset` — Asset struct, Identity, Index, Cache, Search
   - `internal/source` — Source struct, Manifest
   - `internal/sourcemanager` — Source lifecycle: add, remove, scan, sync
   - `internal/deploy` — Engine: deploy, remove, health, repair, bulk ops
   - `internal/profile` — Profile/Snapshot CRUD, Manager (switch, deploy, restore)
   - `internal/agent` — AgentRegistry: detect, lookup, default
   - `internal/state` — Deployment state persistence (lives at core layer as a storage mechanism, used by service layer)
   - `cmd/` — Cobra commands, App struct (lazy service init), helpers

4. **Data flow example** — `nd deploy skills/greeting`:
   ```
   cmd/deploy.go → app.SourceManager().Scan() → asset.Index.Resolve()
                 → app.DeployEngine().Deploy() → symlink created
                 → state.Save() → confirmation printed
   ```

5. **Key patterns**
   - Atomic writes for config and state files
   - Config merging: defaults → global → project → CLI flags
   - Convention-based source scanning (directory names = asset types)
   - Test doubles via function injection (SetLookPath, SetStat)
   - Lazy service initialization in App struct

6. **Asset lifecycle**
   ```
   Source registered → Scan discovers assets → Index built
   → Deploy creates symlink → State tracks deployment
   → Health check validates → Repair fixes issues
   ```

7. **Testing strategy**
   - TDD workflow (red/green/refactor)
   - Function injection for OS-level stubs
   - Integration tests in `tests/integration/`
   - Coverage targets: 80%+ business logic, lower acceptable for CLI/TUI

---

## 5. User Guide Pages

### docs/guide/getting-started.md

**Goal:** Zero to first deployed asset in 5 minutes.

**Sections:**
1. Install nd (link to README)
2. Initialize: `nd init`
3. Add your first source: `nd source add ~/my-assets` or `nd source add owner/repo`
4. Browse available assets: `nd list`
5. Deploy an asset: `nd deploy skills/greeting`
6. Verify: `nd status`
7. Optional setup:
   - Shell completions: `nd completion bash|zsh|fish --install` (covers all three shells with manual and auto-install)
   - Open config: `nd settings edit`
8. Next steps: profiles, snapshots, TUI (`nd` with no args)

### docs/guide/user-guide.md

**Goal:** Cover all core workflows.

**Sections:**
1. **Interactive mode** (brief intro)
   - Many commands support running without arguments to get an interactive picker
   - Works in deploy, remove, profile delete/switch/deploy, snapshot delete/restore
   - Disabled in non-TTY and `--json` mode (returns helpful error instead)
2. **Global flags for scripting**
   - `--json` — structured output for piping/parsing
   - `--yes` / `-y` — skip confirmation prompts (essential for scripts)
   - `--dry-run` — preview without making changes
   - `--quiet` / `--verbose` — control output verbosity
3. **Managing sources**
   - Adding local directories
   - Adding git repositories (HTTPS, SSH, GitHub shorthand)
   - Listing sources: `nd source list`
   - Syncing git sources: `nd sync --source <id>`
   - Removing sources: `nd source remove <id>`
4. **Deploying assets**
   - Single asset: `nd deploy skills/greeting`
   - Multiple assets: `nd deploy skills/greeting commands/hello`
   - By type: `nd deploy --type skills greeting`
   - Scopes: `--scope global` vs `--scope project`
   - Symlink strategy: `--relative` vs `--absolute`
   - Interactive picker: run `nd deploy` with no args
5. **Removing assets**
   - Single/multiple removal
   - Pinned asset warnings
   - Interactive picker
6. **Listing and status**
   - `nd list` with filters (--type, --source, --pattern)
   - `nd status` — health indicators, origins, scopes
   - JSON output for scripting: `--json`
7. **Settings**
   - `nd settings edit` — open config in `$EDITOR` / `$VISUAL` / vi
8. **Syncing and repair**
   - `nd sync` — fix broken symlinks
   - `nd sync --source <id>` — git pull + repair
   - Dry run: `nd sync --dry-run`
9. **Health checks**
   - `nd doctor` — config, sources, deployments, agents, git
   - Interpreting output
10. **Shell completions**
    - `nd completion bash|zsh|fish` — print completion script
    - `nd completion bash|zsh|fish --install` — auto-install to standard location
    - Per-shell setup instructions (fpath for zsh, etc.)
11. **Uninstalling**
    - `nd uninstall` — removes all managed symlinks
    - Does not delete config directory

### docs/guide/profiles-and-snapshots.md

**Goal:** Explain advanced workflow management.

**Sections:**
1. **What are profiles?** — Named collections of assets, like browser profiles
2. **Creating profiles**
   - From asset list: `nd profile create work --assets skills/a,skills/b,agents/c`
   - From current state: `nd profile create work --from-current`
3. **Building profiles incrementally**
   - `nd profile add-asset work skills/new-skill` — add assets to existing profile
4. **Deploying a profile:** `nd profile deploy work`
5. **Switching profiles**
   - `nd profile switch personal` — shows diff preview, confirms, switches
   - Auto-snapshot before switch (safety net)
   - What gets removed vs kept (origin tracking)
6. **Deleting profiles**
   - `nd profile delete work` — removes the profile definition (not deployed assets)
   - Interactive picker if no name provided
   - Requires confirmation
7. **Pinning assets**
   - `nd pin skills/greeting` — survives profile switches
   - `nd unpin skills/greeting` — returns to manual origin
8. **Snapshots**
   - Save current state: `nd snapshot save before-experiment`
   - List snapshots: `nd snapshot list`
   - Restore: `nd snapshot restore before-experiment`
   - Auto-snapshots: created automatically before destructive ops
9. **Workflow example**
   - Create "work" and "personal" profiles
   - Pin shared assets
   - Switch between them
   - Restore from snapshot after an experiment

### docs/guide/configuration.md

**Goal:** Full configuration reference.

**Sections:**
1. **Config file location**
   - Global: `~/.config/nd/config.yaml`
   - Project: `.nd/config.yaml`
   - Override: `--config <path>`
2. **Full annotated example**
   ```yaml
   version: 1
   default_scope: global
   default_agent: claude-code
   symlink_strategy: absolute
   sources:
     - id: my-assets
       type: local
       path: ~/coding-assets
     - id: community
       type: git
       url: https://github.com/org/assets.git
       alias: community-assets
   agents:
     - name: claude-code
       global_dir: ~/.claude
       project_dir: .claude
   ```
3. **Config key reference** — Table of all keys with type, default, description
4. **Config merging** — Defaults → global → project → CLI flags
5. **Project-level config** — When and why to use `.nd/config.yaml`
6. **Environment variables** — `$EDITOR`/`$VISUAL` for `nd settings edit`

### docs/guide/creating-sources.md

**Goal:** For people creating asset libraries.

**Sections:**
1. **Directory convention** — Asset type names as directories:
   ```
   my-assets/
   ├── skills/
   │   └── greeting/           (directory asset)
   ├── agents/
   │   └── researcher.md       (file asset)
   ├── commands/
   │   └── deploy-all.md       (file asset)
   ├── output-styles/
   │   └── concise.md          (file asset)
   ├── rules/
   │   └── no-emojis.md        (file asset)
   ├── context/
   │   └── CLAUDE.md           (file asset, special deploy rules)
   ├── plugins/
   │   └── my-plugin/          (directory asset, not symlink-deployed)
   └── hooks/
       └── pre-commit/         (directory asset)
   ```
2. **Asset types** — Which are file vs directory, what each type does
3. **Context files** — Special deployment rules, `_meta.yaml` format
4. **Manifest file** — `nd-source.yaml` for custom directory structures
5. **Publishing** — Push to git, share URL, others `nd source add` it

---

## 6. Auto-Generated Command Reference

### Implementation

Add `cmd/gendocs/main.go`:
```go
package main

import (
    "log"

    "github.com/armstrongl/nd/cmd"
    "github.com/spf13/cobra/doc"
)

func main() {
    rootCmd := cmd.NewRootCmd(&cmd.App{})
    if err := doc.GenMarkdownTree(rootCmd, "docs/reference/"); err != nil {
        log.Fatal(err)
    }
}
```

### Running

```sh
go run cmd/gendocs/main.go
```

### Output

One markdown file per command in `docs/reference/`, committed to the repo.

---

## 7. Constraints and Non-Goals

### Constraints

- All install instructions must be verified working before documenting
- Command reference must be generated from actual Cobra definitions (not hand-written)
- CI must pass before first release tag
- Homebrew tap requires the `armstrongl/homebrew-tap` repo to exist

### Non-Goals

- mdBook or docs site (can upgrade later)
- GoDoc comments (separate effort)
- Issue templates / PR templates (separate effort)
- TUI user guide (the TUI is functional but still evolving; document after stabilization)
- golangci-lint config file (use defaults for now; add custom rules as needed)
- Windows support documentation (darwin + linux only)

---

## 8. Dependency Order

```
Track A — Distribution pipeline (sequential):
  1. goreleaser config + version ldflags    (no external deps)
  2. GitHub Actions CI workflow             (no external deps, but validates 1)
  3. Create homebrew-tap repo               (requires GitHub)
  4. GitHub Actions release workflow        (depends on 1, 3)
  5. Tag v0.1.0 + release                  (depends on 1-4, validates pipeline)

Track B — Contributor docs (parallel with Track A):
  6. CONTRIBUTING.md                       (no deps on release)
  7. ARCHITECTURE.md                       (no deps on release)

Track C — User docs (after Track A completes):
  8. README.md                             (depends on 5 — real install instructions)
  9. docs/guide/ pages                     (depends on 8 — links to README install)

Track D — Reference docs (parallel with everything):
  10. docs/reference/ auto-generation      (no external deps — only needs existing cmd/ code)

Note: README.md already exists as a stub and will be replaced with the full version.
```
