# nd repo management design

| Field | Value |
| --- | --- |
| **Date** | 2026-03-14 |
| **Author** | Larah |
| **Status** | Approved |
| **Language** | Go 1.23+ |
| **Approach** | Approach 1 (Go) |

## Overview

This document defines how the nd repository is structured, tooled, and managed. It covers project layout, build/release tooling, CI/CD, Git workflow, versioning, documentation, and governance artifacts.

## 1. project layout

```text
nd/
├── main.go                      # Entry point, declares version vars, calls cmd.Execute()
├── go.mod / go.sum
│
├── cmd/                         # Cobra commands (thin layer, no business logic)
│   ├── root.go                  # Root command, global flags (--verbose, --quiet, --scope)
│   ├── source.go                # nd source add|list|remove|sync
│   ├── deploy.go                # nd deploy <asset>
│   ├── remove.go                # nd remove <asset>
│   ├── status.go                # nd status
│   ├── sync.go                  # nd sync
│   ├── profile.go               # nd profile create|switch|list|delete
│   ├── snapshot.go              # nd snapshot save|restore|list
│   ├── doctor.go                # nd doctor
│   ├── init_cmd.go              # nd init (init is a Go keyword, hence the suffix)
│   ├── settings.go              # nd settings edit
│   └── version.go               # nd version
│
├── internal/                    # All domain logic (compiler-enforced privacy)
│   ├── config/                  # Config loading, validation, hierarchy merging
│   ├── source/                  # Source registration, scanning, Git clone/pull
│   ├── asset/                   # Asset types, identity tuples, discovery
│   ├── deploy/                  # Symlink creation/removal, health checks, sync
│   ├── agent/                   # Agent registry, Claude Code detection
│   ├── profile/                 # Profile CRUD, snapshot save/restore
│   ├── backup/                  # Backup management (context file backups)
│   ├── state/                   # Deployment state (deployments.yaml), file locking
│   └── tui/                     # Bubble Tea application
│       ├── app.go               # Main Bubble Tea model
│       ├── views/               # Screen-level views (dashboard, source browser, etc.)
│       └── components/          # Reusable widgets (asset list, status bar, etc.)
│
├── testdata/                    # Shared test fixtures (mock source trees, configs)
│
├── docs/
│   ├── specs/                   # nd-go-spec.md (exists)
│   ├── plans/                   # Design and implementation plans (this file)
│   ├── user-guide/              # End-user documentation
│   └── dev/                     # Contributor documentation
│
├── .github/
│   ├── workflows/
│   │   ├── ci.yml               # Lint + test + build + security scan
│   │   └── release.yml          # goreleaser on v* tag push
│   ├── ISSUE_TEMPLATE/
│   │   ├── bug_report.yml
│   │   └── feature_request.yml
│   ├── PULL_REQUEST_TEMPLATE.md
│   └── dependabot.yml
│
├── .goreleaser.yaml
├── .golangci.yml
├── .editorconfig
├── .gitignore
├── Makefile
├── CHANGELOG.md
├── CONTRIBUTING.md
├── CODE_OF_CONDUCT.md
├── SECURITY.md
├── LICENSE                      # MIT (exists)
└── README.md                    # (exists, will be expanded)
```

### Key decisions

- `main.go` at root: single binary, shortest `go install` path
- `cmd/` holds Cobra commands only: they call into `internal/` packages, never contain business logic
- `internal/` packages organized by domain (source, deploy, agent, profile), not by layer (handlers, services, models)
- `state/` is separate from `deploy/` because file locking and atomic writes are distinct concerns
- TUI is nested under `internal/tui/` with `views/` and `components/` sub-packages (max 3 levels)
- No `pkg/` directory: nd is not a library, nothing to export

## 2. tooling and configuration

### Build and release

**goreleaser** handles the entire release pipeline:

- Builds `CGO_ENABLED=0` static binaries for `darwin/arm64` and `darwin/amd64` (Linux later)
- Embeds version/commit/date via ldflags into `main.go` variables
- Auto-generates Homebrew formula and pushes to a separate `homebrew-tap` repo
- Creates GitHub Releases with checksums and changelogs
- Triggered by `git tag v0.1.0 && git push --tags`

### Linting

**golangci-lint v2** with ~30 linters organized by category:

- Correctness: `govet`, `staticcheck`, `errcheck`, `ineffassign`
- Style: `gofumpt` (strict superset of gofmt), `goimports`, `misspell`
- Error handling: `errcheck`, `wrapcheck`, `nilerr`
- Security: `gosec`
- Performance: `prealloc`, `bodyclose`
- Complexity: `cyclop`, `gocognit`
- Disabled intentionally: `gochecknoglobals`, `gochecknoinits` (conflict with Cobra patterns)

### Task runner

**Makefile** with targets:

| Target | Command |
| --- | --- |
| `make build` | Build binary |
| `make test` | Run tests with race detector |
| `make lint` | Run golangci-lint |
| `make fmt` | Format with gofumpt |
| `make vet` | Run `go vet` |
| `make tidy` | Run `go mod tidy` |
| `make vuln` | Run govulncheck |
| `make cover` | Generate coverage report |
| `make snapshot` | goreleaser snapshot (local test build) |
| `make install` | Run `go install` |
| `make clean` | Remove build artifacts |

### Dependencies

- **`go.mod`/`go.sum`**: standard Go module management
- **Dependabot**: weekly updates for Go modules and GitHub Actions versions
- **No vendoring**: the module proxy provides sufficient reliability

### Pre-commit

- gofumpt formatting check
- golangci-lint (fast mode)
- Via `pre-commit` framework or standalone git hooks

## 3. CI/CD

### CI workflow (`ci.yml`)

Runs on every push and PR. Four parallel jobs:

1. **lint**: `golangci-lint-action@v9`, runs against changed files on PRs, all files on push to main
2. **test**: `go test -race -coverprofile=coverage.out ./...`, uploads coverage to Codecov
3. **build**: `go build ./...`, depends on lint+test passing (ensures the binary compiles)
4. **security**: `govulncheck` with SARIF output uploaded to GitHub Security tab

**Caching:** `actions/setup-go@v6` with `cache: true` handles module and build cache automatically.

### Release workflow (`release.yml`)

Triggered on `v*` tag push:

- Checkout with `fetch-depth: 0` (goreleaser needs full history for changelog)
- Run `goreleaser-action@v6`
- Publishes: GitHub Release (binaries + checksums) + Homebrew tap formula
- Requires a PAT with `repo` scope stored as a repository secret (for cross-repo tap publishing)

### Branch protection on `main`

- Require PR before merging
- Require 1 approval minimum (can be relaxed for solo development)
- Require CI status checks: `test`, `lint`, `build`
- Require linear history (no merge commits)
- Require conversations resolved

## 4. git workflow and versioning

### Branching: GitHub Flow

- `main` is always deployable
- Feature branches: `feat/source-scanning`, `feat/deploy-engine`, `feat/tui-dashboard`
- Bug fixes: `fix/broken-symlink-detection`
- Chores: `chore/update-deps`, `docs/contributing-guide`
- Short-lived branches, merged via PR, deleted after merge

### Commit conventions: Conventional Commits

Format: `type(scope): description`

**Types:** `feat`, `fix`, `docs`, `chore`, `test`, `refactor`, `ci`, `style`

**Scopes** (match nd domain concepts): `deploy`, `source`, `profile`, `snapshot`, `tui`, `cli`, `config`, `agent`, `state`

Examples:

- `feat(source): add Git repository scanning`
- `fix(deploy): handle broken symlinks during bulk deploy`
- `docs(readme): add installation instructions`

### Versioning: SemVer

- Start at **v0.1.0**: no backward compatibility guarantees during v0.x
- Breaking changes allowed freely during v0.x
- **v1.0.0 criteria:** stable CLI surface (command names, flags, config format), core spec features implemented, real users, adequate test coverage
- No Go `/v2` import path needed until v2.0.0

### Changelog

- **Keep a Changelog** format (`CHANGELOG.md`)
- Categories: Added, Changed, Deprecated, Removed, Fixed, Security
- Hand-maintained initially
- goreleaser auto-generates release notes for GitHub Releases from commit history

## 5. documentation and governance

### README.md (expand existing)

Section order (following glow/lazygit patterns):

1. Badges (CI status, Go Report Card, Go Reference, Codecov)
2. Short description (1-2 sentences)
3. Demo GIF (critical for TUI tools, add when TUI is built)
4. Quick start (3 commands to get productive)
5. Installation (Homebrew, `go install`, binary download)
6. Usage examples (core commands)
7. Contributing link
8. License

Keep under 300 lines; detailed docs belong in `docs/`.

### Documentation structure

```text
docs/
├── specs/
│   └── nd-go-spec.md            # Exists
├── plans/
│   └── (this file and future plans)
├── user-guide/
│   ├── getting-started.md       # First-time setup walkthrough
│   ├── sources.md               # Managing asset sources
│   ├── deploying.md             # Deploy, remove, sync workflows
│   ├── profiles.md              # Profile and snapshot management
│   └── troubleshooting.md       # Common issues and solutions
└── dev/
    ├── architecture.md          # Component overview for contributors
    └── testing.md               # How to run and write tests
```

### Governance artifacts (for eventual open-source)

- **CONTRIBUTING.md**: how to build, test, submit PRs, commit message format
- **CODE_OF_CONDUCT.md**: Contributor Covenant v2.1
- **SECURITY.md**: how to report vulnerabilities (private disclosure)
- **Issue templates**: YAML-based forms for bug reports and feature requests
- **PR template**: checklist (tests pass, lint clean, changelog updated)

### .gitignore (update existing)

Add Go-specific entries: `nd` (binary), `coverage.out`, `dist/` (goreleaser), `*.test`, `go.work*`, IDE dirs.

### .editorconfig

Tabs for `*.go` (matches gofmt), 2-space indent for YAML/Markdown/JSON, UTF-8, LF line endings.

## Research artifacts

Detailed research reports informing this design:

- `/.claude/docs/reference/go-cli-project-layout-research.md`
- `/.claude/docs/reference/go-cicd-tooling-best-practices.md`
- `/.claude/docs/reports/go-cli-repo-governance-best-practices.md`
- `/.claude/docs/approach-1-go.md`
- `/.claude/docs/approach-2-python.md`
- `/.claude/docs/approach-3-go-ai-assisted.md`
- `/.claude/docs/approach-comparison.md`
