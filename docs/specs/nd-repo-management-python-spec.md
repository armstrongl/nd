# nd repo management spec (Python)

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

nd is a Python CLI/TUI tool for managing coding agent assets. Before any application code can be written, the repository needs foundational infrastructure: a Python project layout that scales to nd's multi-component architecture, a modern packaging setup with `pyproject.toml` and uv, CI/CD pipelines that validate every change and automate PyPI releases, linting and type checking that enforce consistency from the first commit, and governance artifacts that prepare the project for eventual open-source release.

Without this infrastructure, the project risks: inconsistent code style that requires expensive retroactive cleanup, manual release processes that delay distribution, missing CI checks that allow regressions, type errors that surface only at runtime, and a chaotic repo structure that confuses contributors and AI agents alike.

The nd application spec (`docs/specs/nd-python-spec.md`) defines what nd does. This spec defines how the nd repository is structured, tooled, and managed when using the Python approach.

## Goals

1. A developer can clone the repo, run `just build` (or `uv sync`), and have a working nd command available on the first attempt.
2. A developer can run `just lint`, `just typecheck`, and `just test` to validate code quality locally before pushing.
3. Every push to `main` and every PR triggers automated CI that validates lint, type checking, tests, and formatting before merge.
4. A tagged release (`git tag v0.1.0 && git push --tags`) automatically builds distributions, publishes to PyPI, and creates a GitHub Release without manual intervention.
5. The project layout maps directly to nd's component architecture (cli, tui, core, models, config), enabling AI agents and contributors to find and modify code by domain.
6. The repository is prepared for open-source release with governance artifacts (CONTRIBUTING.md, CODE_OF_CONDUCT.md, SECURITY.md, issue templates) in place before the repo goes public.

## Non-goals

- **Implementing nd application features.** This spec covers repo infrastructure only. Application features (source scanning, deploy engine, TUI) are governed by the nd application spec. Mixing concerns would make this spec unbounded.
- **Multi-platform CI from day one.** nd targets macOS initially. Adding a full Linux/Windows CI matrix before the core is stable wastes CI minutes and complicates debugging. The CI test job uses macOS + Ubuntu as a basic matrix; full platform coverage is added when nd approaches v1.0.
- **Automated changelog generation.** Tooling like Python Semantic Release or Towncrier adds complexity for minimal benefit during early development. The changelog is hand-maintained until release cadence justifies automation.
- **PyInstaller/Nuitka binary builds.** Standalone binary distribution adds CI complexity (no cross-compilation, ~94 MB binaries, OS-specific matrix builds) disproportionate to early-stage needs. Users install via `uv tool install` or `pipx`. Reconsider when nd is published and users request binary distribution.
- **Docker-based development environment.** nd is a local CLI tool. Containerizing the development environment adds friction without benefit for a project that targets macOS developers.
- **Monorepo tooling.** nd is a single Python package. Tools like Nx, Turborepo, or hatch workspaces solve problems nd does not have.
- **mypy as the type checker.** pyright provides better Pydantic v2 support, faster execution, and superior real-time editor integration via Pylance. mypy remains a valid alternative but is not the default choice for this project.

## Assumptions

| # | Assumption | Status |
| --- | --- | --- |
| A1 | uv is stable and widely adopted as the Python package manager for CLI projects in 2026. uv handles virtual environment creation, dependency resolution, lock file generation, Python version management, and tool installation. | Confirmed |
| A2 | hatchling with hatch-vcs can derive the package version from git tags at build time, producing PEP 440-compliant version strings (e.g., `0.1.0` from `v0.1.0` tag, `0.1.1.dev4+gabc1234` between tags). | Unconfirmed |
| A3 | `astral-sh/setup-uv@v7` is the current GitHub Action for setting up uv in CI, and it supports `enable-cache: true` for automatic dependency caching. | Unconfirmed |
| A4 | PyPI Trusted Publishing (OIDC) allows publishing from GitHub Actions without API tokens or secrets, using the `pypa/gh-action-pypi-publish` action with `permissions: id-token: write`. | Confirmed |
| A5 | ruff handles both linting and formatting for Python, replacing Black, isort, Flake8, and dozens of plugins with a single Rust-based tool. | Confirmed |
| A6 | pyright provides accurate type checking for Pydantic v2 models via `@dataclass_transform` (PEP 681) support. | Confirmed |
| A7 | Textual snapshot testing (`pytest-textual-snapshot`) produces SVG snapshots for visual regression testing of TUI components. | Unconfirmed |
| A8 | The `src/` layout is the Python community standard for application packages, preventing import shadowing during testing and catching packaging bugs early. | Confirmed |
| A9 | PEP 735 dependency groups (`[dependency-groups]`) are supported by uv for separating dev/test/docs dependencies in `pyproject.toml`. | Unconfirmed |
| A10 | A Homebrew tap formula for a Python CLI requires explicit `resource` stanzas for each Python dependency, generated via `brew update-python-resources`. | Confirmed |

## Functional requirements

This section specifies the repo setup behaviors tagged by MoSCoW priority. Requirements are ordered by implementation dependency within each tier. The prioritization constraint is quality.

### Must have

- **[FR-001]** The repository contains a `pyproject.toml` at the root declaring project metadata (name, description, license, authors, Python version requirement >=3.12), build system (hatchling), dependencies (typer, textual, rich, pydantic, ruamel.yaml), and tool configurations (ruff, pytest, pyright).
- **[FR-002]** The `pyproject.toml` uses hatch-vcs as the version source, deriving the version from git tags. The `[tool.hatch.version]` section configures the tag pattern and fallback version.
- **[FR-003]** The repository contains a `uv.lock` file committed to version control, ensuring reproducible dependency resolution across environments.
- **[FR-004]** The repository contains a `.python-version` file pinning the development Python version (e.g., `3.12`).
- **[FR-005]** The repository uses the `src/` layout with the package at `src/nd/`. The package contains `__init__.py` (version access via `importlib.metadata`), `__main__.py` (Typer app entry point), and `py.typed` (PEP 561 marker).
- **[FR-006]** The `src/nd/` package contains sub-packages for each domain: `cli/` (Typer commands), `tui/` (Textual app, screens, widgets), `core/` (business logic), `models/` (Pydantic v2 models), `config/` (settings and hierarchy), and `util/` (shared helpers). Each sub-package contains an `__init__.py`.
- **[FR-007]** The `tui/` sub-package contains `screens/` and `widgets/` sub-packages for Textual Screen classes and reusable widgets respectively.
- **[FR-008]** The repository contains a `tests/` directory at the root with subdirectories: `unit/`, `integration/`, and `snapshot/`. Each subdirectory contains its own `conftest.py` for fixtures. The root `tests/` directory also contains a `conftest.py` for shared fixtures.
- **[FR-009]** The repository contains a `Justfile` at the root with recipes: `build` (uv sync), `test` (pytest with coverage), `lint` (ruff check + ruff format --check), `typecheck` (pyright), `fmt` (ruff format + ruff check --fix), `dev` (uv sync --dev), `clean` (remove build artifacts, `.pyc` files, `__pycache__/`), `install` (uv tool install from local), and `run` (uv run nd).
- **[FR-010]** The repository contains a `ruff` configuration in `pyproject.toml` under `[tool.ruff]` specifying: target Python version, line length (88), selected rule sets (E, F, W, I, N, UP, B, A, C4, SIM, TCH, RUF, PT for pytest), and per-file ignores for test files.
- **[FR-011]** The repository contains a `pyrightconfig.json` at the root (or `[tool.pyright]` section in `pyproject.toml`) configuring: `pythonVersion` = "3.12", `typeCheckingMode` = "standard", `venvPath` and `venv` pointing to the uv-managed virtual environment.
- **[FR-012]** The repository contains `.github/workflows/ci.yml` with three parallel jobs: lint (`ruff check` + `ruff format --check`), typecheck (`pyright`), and test (`pytest` with coverage across a Python version matrix of 3.12 and 3.13). All jobs use `astral-sh/setup-uv@v7` with caching enabled.
- **[FR-013]** The repository contains `.github/workflows/release.yml` triggered on `v*` tag push. The workflow builds distributions with `uv build`, publishes to PyPI via Trusted Publishing (OIDC), and creates a GitHub Release with auto-generated notes.
- **[FR-014]** The repository contains a `.gitignore` file with Python-specific entries: `__pycache__/`, `*.pyc`, `*.pyo`, `dist/`, `build/`, `*.egg-info/`, `.ruff_cache/`, `.mypy_cache/`, `.pyright/`, `coverage.xml`, `htmlcov/`, `.coverage`, `.pytest_cache/`, and IDE directories.
- **[FR-015]** The repository contains a `README.md` with sections: project description (1-2 sentences), installation (`uv tool install`, `pipx install`, `brew install`), quick start (3 commands), usage examples, contributing link, and license.
- **[FR-016]** Running `just build && uv run nd --version` prints the version string derived from the git tag or development version.
- **[FR-017]** Running `just test` executes all tests with coverage and produces a coverage report.
- **[FR-018]** Running `just lint` runs ruff linter and formatter checks and reports zero issues on the initial codebase.

### Should have

- **[FR-019]** The repository contains a `.editorconfig` file specifying: 4-space indent for `*.py` files (PEP 8), 2-space indent for YAML/Markdown/JSON, UTF-8 encoding, LF line endings, and trailing whitespace trimming.
- **[FR-020]** The repository contains a `CHANGELOG.md` following the Keep a Changelog format with categories: Added, Changed, Deprecated, Removed, Fixed, Security. The initial entry documents the repo setup.
- **[FR-021]** The repository contains a `CONTRIBUTING.md` documenting: how to set up the development environment with uv, commit message format (Conventional Commits), PR submission process, code style expectations (ruff), and type checking expectations (pyright).
- **[FR-022]** The repository contains a `CODE_OF_CONDUCT.md` using the Contributor Covenant v2.1 template.
- **[FR-023]** The repository contains a `SECURITY.md` documenting how to report security vulnerabilities via private disclosure.
- **[FR-024]** The repository contains `.github/ISSUE_TEMPLATE/bug_report.yml` and `.github/ISSUE_TEMPLATE/feature_request.yml` using YAML-based issue forms with nd-specific fields (`nd --version` output, Python version, OS selection).
- **[FR-025]** The repository contains `.github/PULL_REQUEST_TEMPLATE.md` with a checklist: tests pass, lint clean, type checks pass, changelog updated, description of changes.
- **[FR-026]** The repository contains `.github/dependabot.yml` configured for weekly updates of pip dependencies and GitHub Actions versions.
- **[FR-027]** The CI workflow uploads test coverage to Codecov and the README displays a coverage badge.
- **[FR-028]** The repository contains a `.pre-commit-config.yaml` with hooks for `ruff-pre-commit` (lint + format) and `mirrors-pyright` (type checking). Hooks are enforced in CI via `uvx pre-commit run --all-files`.
- **[FR-029]** The repository contains a `tests/snapshot/` directory with a `__snapshots__/` subdirectory for Textual SVG snapshot files and a placeholder test file.
- **[FR-030]** The repository contains a `docs/` directory with subdirectories: `specs/` (exists), `plans/` (exists), `user-guide/`, and `dev/`.
- **[FR-031]** The `pyproject.toml` uses PEP 735 dependency groups (`[dependency-groups]`) to separate dev, test, and docs dependencies from production dependencies.
- **[FR-032]** The `pyproject.toml` declares a `[project.scripts]` entry point mapping `nd` to the Typer app in `src/nd/__main__.py`.

### Could have

- **[FR-033]** The README contains CI status, PyPI version, Python version, and Codecov badges at the top.
- **[FR-034]** The repository contains `docs/dev/architecture.md` documenting the package structure and module responsibilities for contributors.
- **[FR-035]** The repository contains `docs/dev/testing.md` documenting how to run and write tests, including pytest conventions, fixture patterns, and Textual snapshot testing.
- **[FR-036]** The repository contains a Homebrew tap formula (in a separate repo) for `brew install nd`, with `resource` stanzas generated via `brew update-python-resources`.
- **[FR-037]** The CI workflow includes a `pre-commit` job that runs `uvx pre-commit run --all-files` to enforce hook compliance.

### Won't have (this time)

- **[FR-038]** PyInstaller/Nuitka standalone binary builds. Deferred because: no cross-compilation, ~94 MB binaries, complex CI matrix required. Reconsider when: users request binary distribution or `uv tool install` adoption is insufficient.
- **[FR-039]** Automated changelog generation from conventional commits (Python Semantic Release, Towncrier). Deferred because: early development has low commit volume and hand-maintained changelogs are higher quality. Reconsider when: release cadence exceeds monthly.
- **[FR-040]** Docker-based development environment. Deferred because: nd is a local CLI tool and containerization adds friction without benefit. Reconsider when: the contributor base grows beyond the core team.
- **[FR-041]** Full multi-platform CI matrix (Windows, multiple Linux distros). Deferred because: nd targets macOS in v1; Ubuntu is included in the test matrix for basic cross-platform validation. Reconsider when: nd approaches v1.0 and cross-platform support is prioritized.

**MoSCoW distribution:** Must: 18, Should: 14, Could: 5, Won't: 4. Must-Have ratio: 18/41 = 44%. Within the 60% ceiling.

## Non-functional requirements

This section defines quality attributes for the repo infrastructure. These are measurable constraints on how the infrastructure operates, not what it does.

- **[NFR-001]** Environment setup time: `uv sync --dev` completes in under 30 seconds on a machine with uv installed and cache warm.
- **[NFR-002]** CI duration: the full CI workflow (lint + typecheck + test, running in parallel) completes in under 5 minutes.
- **[NFR-003]** Release automation: the release workflow from tag push to published PyPI package and GitHub Release completes in under 10 minutes.
- **[NFR-004]** Zero lint issues: the initial codebase (empty modules with `__init__.py` files) produces zero ruff warnings and zero pyright errors.
- **[NFR-005]** Reproducible builds: `uv build` from the same tagged commit with `uv.lock` produces identical wheel contents across runs.
- **[NFR-006]** CI reliability: the CI workflow does not produce false failures from flaky tests or network-dependent steps. All tool versions are pinned via `uv.lock` and GitHub Actions version tags.
- **[NFR-007]** Type coverage: pyright reports zero errors in `standard` mode on the initial codebase.

## User stories

This section describes key workflows from the developer's perspective. Each story maps to functional requirements.

**US-001: First run after clone.**
As a developer cloning the nd repo for the first time, I want to install dependencies and run the CLI immediately so that I can verify the development environment works.

- Acceptance criteria: after `git clone` + `just build` (or `uv sync --dev`), running `uv run nd --version` prints version information. No manual setup steps required beyond having uv and Python 3.12+ installed.
- Related requirements: FR-001, FR-003, FR-004, FR-005, FR-009, FR-016.

**US-002: Local quality validation.**
As a developer about to push changes, I want to run lint, type checking, and tests locally so that I catch issues before CI does.

- Acceptance criteria: `just lint` runs ruff and reports issues. `just typecheck` runs pyright and reports type errors. `just test` runs all tests with coverage. All three commands complete without errors on clean code.
- Related requirements: FR-009, FR-010, FR-011, FR-017, FR-018.

**US-003: Automated release.**
As a developer ready to release a version, I want to tag and push to trigger an automated release so that I do not manually build and upload to PyPI.

- Acceptance criteria: running `git tag v0.1.0 && git push --tags` triggers the release workflow. The workflow builds distributions, publishes to PyPI via Trusted Publishing, and creates a GitHub Release. Users can `uv tool install nd` after the workflow completes.
- Related requirements: FR-002, FR-013, FR-032.

**US-004: Navigate codebase by domain.**
As a developer (human or AI agent) working on the deploy engine, I want to find all deploy-related code in one module so that I do not have to search across unrelated directories.

- Acceptance criteria: all deploy engine logic lives in `src/nd/core/deployer.py`. The `src/nd/cli/deploy.py` file contains only Typer command wiring that calls into `core/`. No business logic exists in `cli/`.
- Related requirements: FR-005, FR-006.

**US-005: Contribute to the project.**
As a new contributor, I want clear guidance on how to set up the development environment, run tests, and submit changes so that I can contribute without asking questions the docs should answer.

- Acceptance criteria: CONTRIBUTING.md explains the uv-based setup, lint/typecheck workflow, commit message format, and PR process. Issue templates guide bug reports and feature requests. The PR template provides a checklist.
- Related requirements: FR-021, FR-024, FR-025.

## Technical design

This section captures the tooling architecture, project structure, and workflow conventions for the nd repository under the Python approach. It describes infrastructure decisions, not application architecture.

### Component overview

The repo infrastructure has four components:

1. **Project skeleton.** The `pyproject.toml`, `src/nd/` package with sub-packages for each domain (cli, tui, core, models, config, util), entry point (`__main__.py`), and type stub marker (`py.typed`). This is the foundation that all nd application code builds on. The skeleton installs and runs (`nd --version`) before any application features are implemented.

2. **Quality tooling.** ruff (linting + formatting), pyright (type checking), and pytest (testing with coverage). These tools run locally via Justfile recipes and in CI via GitHub Actions. Configuration lives in `pyproject.toml` (ruff, pytest) and `pyrightconfig.json` (pyright).

3. **CI/CD pipelines.** Two GitHub Actions workflows: `ci.yml` (validates every push and PR) and `release.yml` (automates releases on tag push). CI runs lint, typecheck, and test in parallel. Release builds distributions with `uv build` and publishes to PyPI via Trusted Publishing.

4. **Governance artifacts.** Documentation and templates that prepare the repo for open-source: README, CONTRIBUTING, CODE_OF_CONDUCT, SECURITY, issue templates, PR template, and CHANGELOG. These exist from day one so that the repo is public-ready at any point.

### Key technology choices

| Choice | Rationale |
| --- | --- |
| Python 3.12+ | Type parameter syntax (PEP 695), `tomllib` in stdlib, performance improvements, broad ecosystem support. |
| Typer | Type-hint-based CLI framework. Commands are Python functions with annotated parameters. Less boilerplate than argparse or click. |
| Textual | Modern Python TUI framework with CSS-based styling, component model, and built-in snapshot testing. Most successful Python TUI framework on GitHub. |
| Rich | Terminal rendering library (tables, trees, progress bars, syntax highlighting). Textual's companion library. |
| Pydantic v2 | Data validation and settings management via Python type annotations. Handles config validation, YAML models, and state schemas with minimal code. |
| ruamel.yaml | YAML parser that preserves comments and formatting. Required for round-trip editing of user config files. |
| uv | Python package manager (10-100x faster than pip/poetry). Handles virtual environments, dependency resolution, lock files, Python version management, and tool installation in a single binary. |
| hatchling + hatch-vcs | Build backend with git tag-based versioning. No manual version strings to maintain. |
| ruff | Rust-based linter and formatter replacing Black, isort, Flake8, and dozens of plugins. Processes ~1M LOC/min. |
| pyright | Type checker with strong Pydantic v2 support via `@dataclass_transform` (PEP 681). 10x faster than mypy. |
| pytest | Standard Python test framework. Fixture system, parametrize, markers, plugin ecosystem. |
| Justfile | Rust-powered command runner with Make-like syntax. No tab sensitivity, cross-platform, widely adopted with uv. |
| GitHub Actions | CI/CD platform. Native GitHub integration, `astral-sh/setup-uv` action with built-in caching. |
| Dependabot | Built-in GitHub dependency update bot. Zero setup for pip and GitHub Actions versions. |
| Conventional Commits | Structured commit messages (`type(scope): description`) enabling future changelog automation. |

### Project structure

```text
nd/
├── pyproject.toml                # Metadata, deps, tool config (ruff, pytest)
├── uv.lock                      # Locked dependencies (committed)
├── .python-version               # Pins dev Python version
│
├── src/
│   └── nd/
│       ├── __init__.py           # Version via importlib.metadata
│       ├── __main__.py           # Typer app entry point
│       ├── py.typed              # PEP 561 marker
│       ├── cli/                  # Typer commands (thin wiring layer)
│       │   ├── __init__.py
│       │   ├── source.py
│       │   ├── deploy.py
│       │   ├── remove.py
│       │   ├── status.py
│       │   ├── sync.py
│       │   ├── profile.py
│       │   ├── snapshot.py
│       │   ├── doctor.py
│       │   ├── init.py
│       │   ├── settings.py
│       │   └── version.py
│       ├── tui/                  # Textual application
│       │   ├── __init__.py
│       │   ├── app.py            # Main Textual App subclass
│       │   ├── app.tcss          # Textual CSS stylesheet
│       │   ├── screens/
│       │   │   └── __init__.py
│       │   └── widgets/
│       │       └── __init__.py
│       ├── core/                 # Business logic (no UI deps)
│       │   ├── __init__.py
│       │   ├── scanner.py
│       │   ├── deployer.py
│       │   ├── registry.py
│       │   ├── profiles.py
│       │   ├── snapshots.py
│       │   ├── resolver.py
│       │   ├── backup.py
│       │   └── git.py
│       ├── models/               # Pydantic v2 data models
│       │   ├── __init__.py
│       │   ├── asset.py
│       │   ├── source.py
│       │   ├── profile.py
│       │   ├── agent_info.py
│       │   └── deploy.py
│       ├── config/               # Configuration management
│       │   ├── __init__.py
│       │   ├── settings.py
│       │   ├── hierarchy.py
│       │   └── defaults.py
│       └── util/                 # Shared helpers
│           ├── __init__.py
│           ├── fs.py
│           ├── yaml.py
│           ├── display.py
│           └── logging.py
│
├── tests/
│   ├── conftest.py               # Shared fixtures
│   ├── unit/
│   │   └── conftest.py
│   ├── integration/
│   │   └── conftest.py
│   └── snapshot/
│       ├── __snapshots__/
│       └── conftest.py
│
├── docs/
│   ├── specs/
│   ├── plans/
│   ├── user-guide/
│   └── dev/
│
├── .github/
│   ├── workflows/
│   │   ├── ci.yml
│   │   └── release.yml
│   ├── ISSUE_TEMPLATE/
│   │   ├── bug_report.yml
│   │   └── feature_request.yml
│   ├── PULL_REQUEST_TEMPLATE.md
│   └── dependabot.yml
│
├── .pre-commit-config.yaml
├── pyrightconfig.json
├── .editorconfig
├── .gitignore
├── Justfile
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
        ├──► lint (ruff check + ruff format --check)
        ├──► typecheck (pyright)
        └──► test (pytest --cov, Python 3.12 + 3.13 matrix)
                │
                ▼
        All pass → PR mergeable

Developer pushes tag (v*)
        │
        ▼
  GitHub Actions (release.yml)
        │
        ▼
  Build + publish
        │
        ├──► uv build (sdist + wheel)
        ├──► Publish to TestPyPI (validation)
        ├──► Publish to PyPI (Trusted Publishing / OIDC)
        └──► Create GitHub Release (auto-generated notes)
```

### Distribution channels

| Channel | Command | Audience |
| --- | --- | --- |
| PyPI via uv | `uv tool install nd` | Developers with uv installed (primary) |
| PyPI via pipx | `pipx install nd` | Developers with pipx installed |
| Homebrew tap | `brew install user/tap/nd` | macOS users preferring Homebrew (secondary) |
| Source | `uv pip install git+https://github.com/user/nd` | Developers wanting latest from git |

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
| Scopes | `deploy`, `source`, `profile`, `snapshot`, `tui`, `cli`, `config`, `agent`, `models`, `repo` |

**Versioning:** SemVer starting at v0.1.0. Breaking changes allowed during v0.x. v1.0.0 criteria: stable CLI surface, core features implemented, real users, 80%+ test coverage on core packages. Version derived from git tags via hatch-vcs (no manual version string).

**Branch protection on `main`:**

- Require PR before merging
- Require CI status checks: `lint`, `typecheck`, `test`
- Require linear history
- Require conversations resolved

## Boundaries

This section defines behavior tiers for AI agents implementing this spec.

### Always

- Always run `just lint` and `just test` before committing changes to verify zero warnings and all tests pass.
- Always run `just typecheck` to verify pyright reports zero errors before committing.
- Always use ruff for both linting and formatting. Do not introduce Black, isort, Flake8, or other tools that ruff replaces.
- Always place business logic in `src/nd/core/`. The `cli/` package contains only Typer command wiring.
- Always use Conventional Commits format for commit messages.
- Always create files matching the project structure defined in this spec. Do not add packages not documented here without asking.
- Always use Justfile recipes for build, test, and lint operations rather than invoking `uv run` or `pytest` directly.
- Always use Pydantic v2 `BaseModel` for data structures in `models/` and `pydantic-settings` for configuration in `config/`.

### Ask-first

- Ask before adding a new top-level directory to the repository.
- Ask before adding a new sub-package under `src/nd/` not listed in the project structure.
- Ask before adding a new dependency to `pyproject.toml`.
- Ask before modifying ruff configuration in `pyproject.toml`.
- Ask before modifying CI workflows (`.github/workflows/`).
- Ask before modifying `pyrightconfig.json` to relax type checking strictness.
- Ask before adding a new dependency group in `pyproject.toml`.

### Never

- Never place business logic in the `cli/` package. Commands call into `core/` modules.
- Never use the flat layout (package at root instead of `src/`). The `src/` layout is required.
- Never commit IDE-specific files (`.idea/`, `.vscode/settings.json` overrides) to the repository.
- Never skip CI checks or use `--no-verify` on git commits.
- Never use `pip install` directly. All dependency management goes through uv.
- Never modify the `LICENSE` file.
- Never use `# type: ignore` without a specific error code and explanatory comment.
- Never introduce mypy, Black, isort, or Flake8 as alternatives to the chosen tools (pyright, ruff).

## Success criteria

**Core success criteria** (verifiable with Must-Have requirements):

1. `git clone` + `just build` + `uv run nd --version` succeeds on a clean macOS machine with uv and Python 3.12+ installed. Verified by: manual test on a fresh clone.
2. `just lint` reports zero ruff issues on the initial codebase. Verified by: CI lint job passes.
3. `just typecheck` reports zero pyright errors on the initial codebase. Verified by: CI typecheck job passes.
4. `just test` passes and produces a coverage report. Verified by: CI test job passes.
5. Pushing a `v*` tag triggers the release workflow and publishes to PyPI. Verified by: tag `v0.1.0`, check PyPI project page.
6. Every sub-package under `src/nd/` has an `__init__.py` and is importable. Verified by: `python -c "from nd.core import scanner"` succeeds in the virtual environment.

**Extended success criteria** (require Should-Have or Could-Have requirements):

7. CONTRIBUTING.md, CODE_OF_CONDUCT.md, and SECURITY.md exist and contain complete, actionable content. Verified by: manual review.
8. Issue templates and PR template guide contributors through structured submissions. Verified by: create a test issue and PR using the templates.
9. Dependabot opens PRs for dependency updates within one week of setup. Verified by: check Dependabot PR activity.
10. Test coverage data is uploaded to Codecov and visible on PRs. Verified by: open a PR, check for Codecov report.
11. Pre-commit hooks run ruff and pyright before each commit. Verified by: commit with a lint violation, observe hook failure.

## Open questions

Question IDs are stable and not reused across revisions.

### Open questions

| # | Question | Category | Impact |
| --- | --- | --- | --- |
| Q1 | What is the GitHub organization or username for the PyPI package name and `pyproject.toml` metadata? | Blocking | Determines package name on PyPI and repository URL. |
| Q2 | Should the PyPI package name be `nd` or `nd-cli` to avoid name conflicts with existing packages on PyPI? | Blocking | `nd` may already be taken on PyPI. Must check availability before publishing. |
| Q3 | Should the Homebrew tap repo be created now or deferred until the first tagged release? | Non-blocking | Affects whether FR-036 can be validated during setup. |
| Q4 | What Codecov organization should coverage be uploaded to? | Non-blocking | Affects CI workflow configuration. Can be deferred. |
| Q5 | Should branch protection rules be enforced immediately, or deferred until the project has multiple contributors? | Non-blocking | Solo development may benefit from direct-push-to-main during early development. |
| Q6 | Should TestPyPI be used as a validation step before publishing to PyPI, or is direct PyPI publishing acceptable? | Non-blocking | TestPyPI adds safety but doubles the publish step. |

### Resolved questions

| # | Question | Resolution |
| --- | --- | --- |
| ~~Q0~~ | ~~Justfile or Makefile for task running?~~ | Justfile. Decided 2026-03-14 based on community momentum with uv and cross-platform compatibility. |

## Changelog

| Version | Date | Author | Changes |
| --- | --- | --- | --- |
| 0.1 | 2026-03-14 | Larah | Initial draft. Derived from approved Python approach (`/.claude/docs/approach-2-python.md`) and Python research reports. |
