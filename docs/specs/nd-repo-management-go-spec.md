# nd repo management spec

| Field | Value |
| --- | --- |
| **Date** | 2026-03-14 |
| **Author** | Larah |
| **Status** | Draft |
| **Version** | 0.1 |
| **Last reviewed** | 2026-03-14 |
| **Last reviewed by** | Larah |

## Section index

- **Problem statement:** Defines why nd needs structured repo management before implementation begins.
- **Goals:** Lists the measurable outcomes the repo setup aims to achieve.
- **Non-goals:** Names what the repo setup will not cover and the scope creep each exclusion prevents.
- **Assumptions:** States conditions believed true but not verified that this spec depends on.
- **Functional requirements:** Specifies repo setup behaviors tagged by MoSCoW priority, ordered by implementation dependency.
- **Non-functional requirements:** Defines quality attributes for the repo infrastructure.
- **User stories:** Describes key workflows from the developer's perspective with acceptance criteria.
- **Technical design:** Captures project structure, tooling choices, CI/CD architecture, Git workflow, and documentation layout.
- **Boundaries:** Defines agent behavior tiers (always, ask-first, never) for AI agents implementing this spec.
- **Success criteria:** Defines how to determine whether the repo setup succeeded.
- **Open questions:** Lists unresolved decisions categorized by implementation impact.
- **Changelog:** Tracks document revisions.

## Problem statement

nd is a Go CLI/TUI tool for managing coding agent assets. Before any application code can be written, the repository needs foundational infrastructure: a Go project layout that scales to nd's five-component architecture, CI/CD pipelines that validate every change and automate releases, linting and formatting that enforce consistency from the first commit, and governance artifacts that prepare the project for eventual open-source release.

Without this infrastructure, the project risks: inconsistent code style that requires expensive retroactive cleanup, manual release processes that delay distribution, missing CI checks that allow regressions, and a chaotic repo structure that confuses contributors and AI agents alike.

The nd application spec (`docs/specs/nd-go-spec.md`) defines what nd does. This spec defines how the nd repository is structured, tooled, and managed.

## Goals

1. A developer can clone the repo, run `make build`, and produce a working nd binary on the first attempt.
2. A developer can run `make test`, `make lint`, and `make fmt` to validate code quality locally before pushing.
3. Every push to `main` and every PR triggers automated CI that validates lint, tests, build, and security before merge.
4. A tagged release (`git tag v0.1.0 && git push --tags`) automatically produces binaries, a Homebrew formula, and a GitHub Release without manual intervention.
5. The project layout maps directly to nd's component architecture (source, deploy, agent, profile, TUI, CLI), enabling AI agents and contributors to find and modify code by domain.
6. The repository is prepared for open-source release with governance artifacts (CONTRIBUTING.md, CODE_OF_CONDUCT.md, SECURITY.md, issue templates) in place before the repo goes public.

## Non-goals

- **Implementing nd application features.** This spec covers repo infrastructure only. Application features (source scanning, deploy engine, TUI) are governed by the nd application spec. Mixing concerns would make this spec unbounded.
- **Multi-platform CI from day one.** nd targets macOS initially. Adding Linux/Windows CI matrix builds before the core is stable wastes CI minutes and complicates debugging. Multi-platform support is added when nd approaches v1.0.
- **Automated changelog generation.** Tooling like `git-cliff` or `conventional-changelog` adds complexity for minimal benefit during early development. The changelog is hand-maintained until release cadence justifies automation.
- **Complex pre-commit hook infrastructure.** A lightweight pre-commit setup (gofumpt + golangci-lint) is sufficient. Heavy frameworks like `pre-commit` (Python dependency) or `lefthook` add tooling overhead disproportionate to a solo/small-team project.
- **Docker-based development environment.** nd is a local CLI tool. Containerizing the development environment adds friction without benefit for a project that targets macOS developers building a single Go binary.
- **Monorepo tooling.** nd is a single binary from a single module. Tools like Bazel, Nx, or Turborepo solve problems nd does not have.

## Assumptions

| # | Assumption | Status |
| --- | --- | --- |
| A1 | goreleaser v2 supports Homebrew tap formula generation and cross-repo push via PAT. | Unconfirmed |
| A2 | golangci-lint v2 is the current major version and supports the `formatters` section (separate from `linters`). | Unconfirmed |
| A3 | `actions/setup-go@v6` with `cache: true` handles Go module and build cache automatically. | Unconfirmed |
| A4 | GitHub Actions Dependabot supports both Go modules (`gomod`) and GitHub Actions version updates (`github-actions`). | Confirmed |
| A5 | `gofumpt` is a strict superset of `gofmt` and is widely adopted as the Go community formatting standard. | Confirmed |
| A6 | Go 1.23+ is available on GitHub Actions runners (`ubuntu-latest`, `macos-latest`). | Confirmed |
| A7 | Conventional Commits format is compatible with goreleaser's changelog generation. | Unconfirmed |
| A8 | A separate `homebrew-tap` repository is needed for goreleaser to push Homebrew formulas (cannot push to the same repo). | Unconfirmed |

## Functional requirements

This section specifies the repo setup behaviors tagged by MoSCoW priority. Requirements are ordered by implementation dependency within each tier. The prioritization constraint is quality.

### Must have

- **[FR-001]** The repository contains a `go.mod` file at the root declaring the module path and Go version (1.23+), with `go.sum` tracking dependency checksums.
- **[FR-002]** The repository contains a `main.go` file at the root that declares version variables (`version`, `commit`, `date`) with default values and calls `cmd.Execute()`.
- **[FR-003]** The repository contains a `cmd/` package with a `root.go` file defining the root Cobra command, global flags (`--verbose`, `--quiet`), and version information.
- **[FR-004]** The repository contains an `internal/` directory with empty packages for each domain: `config`, `source`, `asset`, `deploy`, `agent`, `profile`, `backup`, `state`, and `tui` (with `tui/views/` and `tui/components/` sub-packages).
- **[FR-005]** Each empty package in `internal/` contains a `doc.go` file with a package-level comment describing the package's responsibility.
- **[FR-006]** The repository contains a `Makefile` at the root with targets: `build`, `test`, `lint`, `fmt`, `vet`, `tidy`, `vuln`, `cover`, `snapshot`, `install`, `clean`. Each target has a `.PHONY` declaration.
- **[FR-007]** The repository contains a `.golangci.yml` file configuring golangci-lint v2 with linters organized by category: correctness (`govet`, `staticcheck`, `errcheck`, `ineffassign`), style (`gofumpt`, `goimports`, `misspell`), error handling (`errcheck`, `wrapcheck`, `nilerr`), security (`gosec`), performance (`prealloc`, `bodyclose`), and complexity (`cyclop`, `gocognit`). The linters `gochecknoglobals` and `gochecknoinits` are explicitly disabled with comments explaining why (Cobra patterns).
- **[FR-008]** The repository contains a `.goreleaser.yaml` file that builds `CGO_ENABLED=0` static binaries for `darwin/arm64` and `darwin/amd64`, embeds version/commit/date via ldflags, and configures Homebrew tap formula generation with `skip_upload: auto` for pre-release tags.
- **[FR-009]** The repository contains `.github/workflows/ci.yml` with four parallel jobs: lint (golangci-lint), test (`go test -race -coverprofile`), build (`go build`), and security (govulncheck). The build job depends on lint and test passing.
- **[FR-010]** The repository contains `.github/workflows/release.yml` triggered on `v*` tag push that runs goreleaser with `fetch-depth: 0` checkout.
- **[FR-011]** The repository contains a `.gitignore` file with Go-specific entries: the `nd` binary, `coverage.out`, `dist/` (goreleaser output), `*.test`, `go.work*`, and IDE directories (`.idea/`, `.vscode/` override files).
- **[FR-012]** The repository contains a `README.md` with sections: project description (1-2 sentences), installation (Homebrew, `go install`, binary download), quick start (3 commands), usage examples, contributing link, and license.
- **[FR-013]** Running `make build` compiles the binary and `./nd version` prints the version, commit hash, and build date.
- **[FR-014]** Running `make test` executes all tests with the race detector enabled and produces a coverage report.
- **[FR-015]** Running `make lint` runs golangci-lint and reports zero issues on the initial codebase.

### Should have

- **[FR-016]** The repository contains a `.editorconfig` file specifying: tabs for `*.go` files, 2-space indent for YAML/Markdown/JSON, UTF-8 encoding, LF line endings, and trailing whitespace trimming.
- **[FR-017]** The repository contains a `CHANGELOG.md` following the Keep a Changelog format with categories: Added, Changed, Deprecated, Removed, Fixed, Security. The initial entry documents the repo setup.
- **[FR-018]** The repository contains a `CONTRIBUTING.md` documenting: how to build and test locally, commit message format (Conventional Commits), PR submission process, and code style expectations.
- **[FR-019]** The repository contains a `CODE_OF_CONDUCT.md` using the Contributor Covenant v2.1 template.
- **[FR-020]** The repository contains a `SECURITY.md` documenting how to report security vulnerabilities via private disclosure.
- **[FR-021]** The repository contains `.github/ISSUE_TEMPLATE/bug_report.yml` and `.github/ISSUE_TEMPLATE/feature_request.yml` using YAML-based issue forms with nd-specific fields (`nd --version` output, OS selection).
- **[FR-022]** The repository contains `.github/PULL_REQUEST_TEMPLATE.md` with a checklist: tests pass, lint clean, changelog updated, description of changes.
- **[FR-023]** The repository contains `.github/dependabot.yml` configured for weekly updates of Go modules and GitHub Actions versions.
- **[FR-024]** The CI workflow uploads test coverage to Codecov and the README displays a coverage badge.
- **[FR-025]** The CI security job uploads govulncheck SARIF output to the GitHub Security tab.
- **[FR-026]** The repository contains a `testdata/` directory at the root with a README explaining its purpose (shared test fixtures).
- **[FR-027]** The repository contains a `docs/` directory with subdirectories: `specs/` (exists), `plans/` (exists), `user-guide/`, and `dev/`.
- **[FR-028]** The `cmd/version.go` file implements an `nd version` command that prints the version string, commit hash, build date, and Go version in a structured format.

### Could have

- **[FR-029]** The repository contains pre-commit hooks (via git hooks or `pre-commit` framework) that run `gofumpt` and `golangci-lint` before each commit.
- **[FR-030]** The README contains CI status, Go Report Card, Go Reference, and Codecov badges at the top.
- **[FR-031]** The repository contains `docs/dev/architecture.md` documenting the component overview and package responsibilities for contributors.
- **[FR-032]** The repository contains `docs/dev/testing.md` documenting how to run and write tests, including conventions for test fixtures and integration tests.
- **[FR-033]** The `.goreleaser.yaml` includes a checksum configuration and signs release artifacts.

### Won't have (this time)

- **[FR-034]** Multi-platform CI matrix (Linux, Windows). Deferred because: nd targets macOS in v1 and adding CI platforms before the core is stable wastes resources. Reconsider when: nd approaches v1.0 and cross-platform support is prioritized.
- **[FR-035]** Automated changelog generation from conventional commits. Deferred because: early development has low commit volume and hand-maintained changelogs are higher quality. Reconsider when: release cadence exceeds monthly.
- **[FR-036]** Docker-based development environment. Deferred because: nd is a local CLI tool and containerization adds friction without benefit. Reconsider when: the contributor base grows beyond the core team.
- **[FR-037]** Man page generation from Cobra. Deferred because: man pages are a polish feature and nd's `--help` output is sufficient during development. Reconsider when: nd is published via Homebrew and users expect `man nd`.

**MoSCoW distribution:** Must: 15, Should: 13, Could: 5, Won't: 4. Must-Have ratio: 15/37 = 41%. Within the 60% ceiling.

## Non-functional requirements

This section defines quality attributes for the repo infrastructure. These are measurable constraints on how the infrastructure operates, not what it does.

- **[NFR-001]** Build time: `make build` completes in under 30 seconds on a machine with Go 1.23+ installed and module cache warm.
- **[NFR-002]** CI duration: the full CI workflow (lint + test + build + security, running in parallel) completes in under 5 minutes.
- **[NFR-003]** Release automation: the release workflow from tag push to published GitHub Release (with binaries and Homebrew formula) completes in under 10 minutes.
- **[NFR-004]** Zero lint issues: the initial codebase (empty packages with `doc.go` files) produces zero golangci-lint warnings when checked.
- **[NFR-005]** Reproducible builds: building the same tagged commit produces identical binary checksums across runs on the same OS/arch.
- **[NFR-006]** CI reliability: the CI workflow does not produce false failures from flaky tests or network-dependent steps. All external tool installations use pinned versions.

## User stories

This section describes key workflows from the developer's perspective. Each story maps to functional requirements.

**US-001: First build after clone.**
As a developer cloning the nd repo for the first time, I want to build and run the binary immediately so that I can verify the development environment works.

- Acceptance criteria: after `git clone` + `make build`, running `./nd version` prints version information. No manual setup steps required beyond having Go 1.23+ installed.
- Related requirements: FR-001, FR-002, FR-003, FR-006, FR-013.

**US-002: Local quality validation.**
As a developer about to push changes, I want to run lint, format, and test locally so that I catch issues before CI does.

- Acceptance criteria: `make lint` runs golangci-lint and reports issues. `make fmt` formats code with gofumpt. `make test` runs all tests with race detection. All three commands complete without errors on clean code.
- Related requirements: FR-006, FR-007, FR-014, FR-015.

**US-003: Automated release.**
As a developer ready to release a version, I want to tag and push to trigger an automated release so that I do not manually build binaries or update Homebrew.

- Acceptance criteria: running `git tag v0.1.0 && git push --tags` triggers the release workflow. goreleaser builds binaries, creates a GitHub Release with checksums, and pushes a Homebrew formula to the tap repo. Users can `brew install nd` from the tap after the workflow completes.
- Related requirements: FR-008, FR-010, FR-028.

**US-004: Navigate codebase by domain.**
As a developer (human or AI agent) working on the deploy engine, I want to find all deploy-related code in one package so that I do not have to search across unrelated directories.

- Acceptance criteria: all deploy engine logic lives in `internal/deploy/`. The `cmd/deploy.go` file contains only Cobra command wiring that calls into `internal/deploy/`. No business logic exists in `cmd/`.
- Related requirements: FR-003, FR-004, FR-005.

**US-005: Contribute to the project.**
As a new contributor, I want clear guidance on how to build, test, and submit changes so that I can contribute without asking questions the docs should answer.

- Acceptance criteria: CONTRIBUTING.md explains the build/test workflow, commit message format, and PR process. Issue templates guide bug reports and feature requests. The PR template provides a checklist.
- Related requirements: FR-018, FR-021, FR-022.

## Technical design

This section captures the tooling architecture, project structure, and workflow conventions for the nd repository. It describes infrastructure decisions, not application architecture.

### Component overview

The repo infrastructure has four components:

1. **Project skeleton.** The Go module, entry point (`main.go`), Cobra command package (`cmd/`), and domain packages (`internal/`). This is the foundation that all nd application code builds on. The skeleton compiles and runs (`nd version`) before any application features are implemented.

2. **Quality tooling.** golangci-lint (linting), gofumpt (formatting), govulncheck (security scanning), and the race detector (concurrency safety). These tools run locally via Makefile targets and in CI via GitHub Actions. Configuration lives in `.golangci.yml`.

3. **CI/CD pipelines.** Two GitHub Actions workflows: `ci.yml` (validates every push and PR) and `release.yml` (automates releases on tag push). CI runs lint, test, build, and security in parallel. Release runs goreleaser to produce binaries and Homebrew formulas.

4. **Governance artifacts.** Documentation and templates that prepare the repo for open-source: README, CONTRIBUTING, CODE_OF_CONDUCT, SECURITY, issue templates, PR template, and CHANGELOG. These exist from day one so that the repo is public-ready at any point.

### Key technology choices

| Choice | Rationale |
| --- | --- |
| Go 1.23+ | Strong CLI ecosystem, single binary distribution, fast startup, `internal/` compiler-enforced privacy. |
| Cobra | Standard Go CLI framework. Subcommand structure, flag parsing, help generation, shell completions. |
| goreleaser | Automates the entire release pipeline: build, archive, Homebrew formula, GitHub Release. ~20 lines of YAML. |
| golangci-lint v2 | Meta-linter running ~30 linters in parallel. The `formatters` section (new in v2) separates formatting from linting. |
| gofumpt | Strict superset of gofmt. Widely adopted as the Go community formatting standard. |
| govulncheck | Official Go vulnerability scanner with reachability analysis (flags only reachable vulnerabilities). |
| Makefile | Standard Go ecosystem task runner. Pre-installed on macOS/Linux. Used by Go itself, Cobra, goreleaser, lazygit. |
| GitHub Actions | CI/CD platform. Native GitHub integration, generous free tier, `setup-go` action with built-in caching. |
| Dependabot | Built-in GitHub dependency update bot. Zero setup for Go modules and GitHub Actions versions. |
| Conventional Commits | Structured commit messages (`type(scope): description`) enabling future changelog automation. |

### Project structure

```text
nd/
├── main.go                      # Entry point, version vars, calls cmd.Execute()
├── go.mod / go.sum
├── cmd/                         # Cobra commands (thin wiring layer)
│   ├── root.go                  # Root command, global flags
│   └── version.go               # nd version command
├── internal/                    # Domain packages (compiler-enforced privacy)
│   ├── config/                  # Config loading, validation, merging
│   │   └── doc.go
│   ├── source/                  # Source registration, scanning, Git
│   │   └── doc.go
│   ├── asset/                   # Asset types, identity, discovery
│   │   └── doc.go
│   ├── deploy/                  # Symlink engine, health checks
│   │   └── doc.go
│   ├── agent/                   # Agent registry, detection
│   │   └── doc.go
│   ├── profile/                 # Profiles, snapshots
│   │   └── doc.go
│   ├── backup/                  # Backup management
│   │   └── doc.go
│   ├── state/                   # Deployment state, file locking
│   │   └── doc.go
│   └── tui/                     # Bubble Tea application
│       ├── doc.go
│       ├── views/
│       │   └── doc.go
│       └── components/
│           └── doc.go
├── testdata/                    # Shared test fixtures
│   └── README.md
├── docs/
│   ├── specs/                   # Application and repo specs
│   ├── plans/                   # Design and implementation plans
│   ├── user-guide/              # End-user documentation
│   └── dev/                     # Contributor documentation
├── .github/
│   ├── workflows/
│   │   ├── ci.yml
│   │   └── release.yml
│   ├── ISSUE_TEMPLATE/
│   │   ├── bug_report.yml
│   │   └── feature_request.yml
│   ├── PULL_REQUEST_TEMPLATE.md
│   └── dependabot.yml
├── .goreleaser.yaml
├── .golangci.yml
├── .editorconfig
├── .gitignore
├── Makefile
├── CHANGELOG.md
├── CONTRIBUTING.md
├── CODE_OF_CONDUCT.md
├── SECURITY.md
├── LICENSE
└── README.md
```

### CI/CD architecture

```text
Developer pushes code
        │
        ▼
  GitHub Actions (ci.yml)
        │
        ├──► lint (golangci-lint)
        ├──► test (go test -race + coverage)
        ├──► build (go build, depends on lint+test)
        └──► security (govulncheck + SARIF)
                │
                ▼
        All pass → PR mergeable

Developer pushes tag (v*)
        │
        ▼
  GitHub Actions (release.yml)
        │
        ▼
  goreleaser
        │
        ├──► Build static binaries (darwin/arm64, darwin/amd64)
        ├──► Create GitHub Release (binaries + checksums)
        └──► Push Homebrew formula to tap repo
```

### Git workflow conventions

**Branching:** GitHub Flow.

| Branch type | Naming pattern | Example |
| --- | --- | --- |
| Feature | `feat/<description>` | `feat/source-scanning` |
| Bug fix | `fix/<description>` | `fix/broken-symlink-detection` |
| Documentation | `docs/<description>` | `docs/contributing-guide` |
| Chore | `chore/<description>` | `chore/update-deps` |
| CI | `ci/<description>` | `ci/add-coverage-upload` |

**Commits:** Conventional Commits format.

| Element | Values |
| --- | --- |
| Types | `feat`, `fix`, `docs`, `chore`, `test`, `refactor`, `ci`, `style` |
| Scopes | `deploy`, `source`, `profile`, `snapshot`, `tui`, `cli`, `config`, `agent`, `state`, `repo` |

**Versioning:** SemVer starting at v0.1.0. Breaking changes allowed during v0.x. v1.0.0 criteria: stable CLI surface, core features implemented, real users, 80%+ test coverage on core packages.

**Branch protection on `main`:**

- Require PR before merging
- Require CI status checks: `test`, `lint`, `build`
- Require linear history
- Require conversations resolved

## Boundaries

This section defines behavior tiers for AI agents implementing this spec.

### Always

- Always run `make lint` and `make test` before committing changes to verify zero warnings and all tests pass.
- Always use `gofumpt` for formatting (not `gofmt`). The `.golangci.yml` enforces this.
- Always place business logic in `internal/` packages. The `cmd/` package contains only Cobra command wiring.
- Always use Conventional Commits format for commit messages.
- Always create files matching the project structure defined in this spec. Do not add directories not documented here without asking.
- Always use the Makefile targets for build, test, and lint operations rather than invoking `go` commands directly.

### Ask-first

- Ask before adding a new top-level directory to the repository.
- Ask before adding a new `internal/` sub-package not listed in the project structure.
- Ask before adding a new dependency to `go.mod`.
- Ask before modifying `.golangci.yml` to disable a linter or change severity.
- Ask before modifying CI workflows (`.github/workflows/`).
- Ask before modifying `.goreleaser.yaml`.

### Never

- Never place business logic in the `cmd/` package. Commands call into `internal/` packages.
- Never create a `pkg/` directory. nd is not a library.
- Never commit IDE-specific files (`.idea/`, `.vscode/settings.json` overrides) to the repository.
- Never skip CI checks or use `--no-verify` on git commits.
- Never vendor dependencies (no `vendor/` directory).
- Never modify the `LICENSE` file.

## Success criteria

**Core success criteria** (verifiable with Must-Have requirements):

1. `git clone` + `make build` + `./nd version` succeeds on a clean macOS machine with Go 1.23+ installed. Verified by: manual test on a fresh clone.
2. `make lint` reports zero issues on the initial codebase. Verified by: CI lint job passes.
3. `make test` passes with race detection enabled. Verified by: CI test job passes.
4. Pushing a `v*` tag triggers goreleaser and produces a GitHub Release with binaries. Verified by: tag `v0.1.0`, check GitHub Releases page.
5. Every `internal/` package has a `doc.go` describing its responsibility. Verified by: `go doc` on each package produces meaningful output.

**Extended success criteria** (require Should-Have or Could-Have requirements):

6. CONTRIBUTING.md, CODE_OF_CONDUCT.md, and SECURITY.md exist and contain complete, actionable content. Verified by: manual review.
7. Issue templates and PR template guide contributors through structured submissions. Verified by: create a test issue and PR using the templates.
8. Dependabot opens PRs for dependency updates within one week of setup. Verified by: check Dependabot PR activity.
9. Test coverage data is uploaded to Codecov and visible on PRs. Verified by: open a PR, check for Codecov report.

## Open questions

Question IDs are stable and not reused across revisions.

### Open questions

| # | Question | Category | Impact |
| --- | --- | --- | --- |
| Q1 | What is the GitHub organization or username for the `go.mod` module path? | Blocking | Determines `go.mod` module path and `go install` command. |
| Q2 | Does goreleaser v2 require a separate `homebrew-tap` repo, or can it push formulas to a branch in the same repo? | Non-blocking | Affects release workflow setup. Default: separate repo. |
| Q3 | Should the Homebrew tap repo be created now or deferred until the first tagged release? | Non-blocking | Affects whether FR-008 can be fully validated during setup. |
| Q4 | What Codecov organization should coverage be uploaded to? | Non-blocking | Affects CI workflow configuration. Can be deferred. |
| Q5 | Should branch protection rules be enforced immediately, or deferred until the project has multiple contributors? | Non-blocking | Solo development may benefit from direct-push-to-main during early development. |

### Resolved questions

| # | Question | Resolution |
| --- | --- | --- |
| ~~Q0~~ | ~~What language should nd be built in?~~ | Go 1.23+. Decided 2026-03-14 based on distribution advantages (single binary, goreleaser, Homebrew). |

## Changelog

| Version | Date | Author | Changes |
| --- | --- | --- | --- |
| 0.1 | 2026-03-14 | Larah | Initial draft. Derived from approved repo management design (`docs/plans/2026-03-14-repo-management-design.md`). |
