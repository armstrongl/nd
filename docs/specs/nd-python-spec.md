# Napoleon Dynamite (nd)

| Field                | Value      |
| -------------------- | ---------- |
| **Date**             | 2026-03-13 |
| **Author**           | Larah      |
| **Status**           | Draft      |
| **Version**          | 0.8        |
| **Last reviewed**    | 2026-03-14 |
| **Last reviewed by** | Larah      |

## Section index

- **Problem statement:** Defines the asset management problem that nd solves for coding agent users.
- **Goals:** Lists the measurable outcomes nd aims to achieve.
- **Non-goals:** Names what nd will not do and the scope creep each exclusion prevents.
- **Assumptions:** States conditions believed true but not verified that the spec depends on.
- **Functional requirements:** Specifies nd's behaviors tagged by MoSCoW priority, ordered by implementation dependency.
- **Non-functional requirements:** Defines quality attributes for performance, reliability, and maintainability.
- **User stories:** Describes key workflows from the user's perspective with acceptance criteria.
- **Technical design:** Captures high-level architecture, component structure, technology decisions, asset deployment mapping, and asset identity model.
- **Boundaries:** Defines agent behavior tiers (always, ask-first, never) for AI agents implementing this spec.
- **Success criteria:** Defines how to determine whether the project succeeded.
- **Open questions:** Lists unresolved decisions categorized by implementation impact.
- **Changelog:** Tracks document revisions.

## Problem statement

Developers who use coding agents like Claude Code accumulate large libraries of assets: skills, agents, commands, output-styles, rules, context files, plugins, and hooks. Managing these assets is manual and fragile. Developers create symlinks by hand, maintain ad-hoc shell scripts, and lose track of what's deployed where. This workflow doesn't scale beyond a handful of assets, breaks silently when symlinks drift or directories change, and provides no mechanism for sharing curated asset collections with other developers.

Power users with hundreds of assets hit this pain multiple times per day. New users who want to adopt shared asset collections have no structured way to install and manage them.

## Goals

1. A developer can deploy any combination of assets to any supported coding agent's configuration directory (global or project-level) through a single command or interactive selection.
2. A developer can detect and repair broken, drifted, or stale asset deployments without manual investigation.
3. A developer can save, name, and switch between curated collections of assets (profiles) and point-in-time deployment snapshots.
4. A developer can connect local directories and Git repositories as asset sources, with nd auto-discovering assets by convention.
5. A developer can export selected assets as a Claude Code plugin or marketplace for distribution.
6. The tool provides both a direct CLI interface (single command to accomplish a task) and an interactive TUI experience (launch and navigate).

## Non-goals

- **Editing asset content.** nd deploys and organizes assets but never modifies their content. Content authoring belongs in the user's editor or AI agent. Adding editing would blur nd's role and create conflict with version-controlled source files.
- **Being a Git client.** nd clones repos to use as asset sources and can trigger sync operations, but it does not expose Git commands (commit, push, pull, branch, merge). Git operations are the user's responsibility. Wrapping Git would duplicate existing tooling and create maintenance burden.
- **Managing MCP server configurations.** MCP config (`.mcp.json`) has its own structure, lifecycle, and tooling. Mixing it with asset deployment would complicate nd's config model and conflict with agent-native MCP management.
- **Cross-machine synchronization.** nd manages assets on a single machine. Syncing across machines is the responsibility of Git, cloud storage, or dotfile managers. Adding sync would require conflict resolution, authentication, and networking concerns that are orthogonal to asset management.
- **Version control of assets.** nd does not track asset history, diffs, or revisions. The asset source (a Git repo or local directory) owns version control. Duplicating this would conflict with Git and create confusion about the source of truth.
- **Supporting coding agents beyond Claude Code in v1.** The architecture supports multiple agents, but v1 targets Claude Code only. Premature multi-agent support would slow the launch and dilute testing focus.
- **Modifying agent settings files.** nd does not write to Claude Code's `settings.json`, `settings.local.json`, or `.claude.json`. Some asset types (hooks, output-styles, enabled plugins) are registered in these files, which means nd cannot fully automate their activation. This is an intentional gap: modifying agent-managed settings files risks conflicts with the agent's own config management. Users must manually enable hooks or output-styles in their settings after deploying assets that require it. This gap may be revisited if Claude Code exposes a CLI or API for settings management.
- **Managing memory files.** Claude Code memory files (`MEMORY.md`, agent-specific memory) are created and maintained by Claude Code or specific agents at runtime. While they could theoretically be backed up and redeployed, managing them introduces complexity around ownership and staleness. Memory support may be added in a future version.

## Assumptions

| #    | Assumption                                                   | Status          |
| ---- | ------------------------------------------------------------ | --------------- |
| A1   | Claude Code's configuration directory structure (`~/.claude/` for global, `.claude/` for project-level) is stable and will not change in breaking ways. | Unconfirmed     |
| A2   | The Claude Code plugin format (`.claude-plugin/plugin.json`, `commands/`, `agents/`, `skills/`, `hooks/`) is stable enough to target for export. | Unconfirmed     |
| A3   | Symlinks are reliable on macOS for this use case (no permission issues, no filesystem limitations for typical developer setups). | Confirmed       |
| A4   | Users manage their asset source repositories outside of nd. Git clone, pull, and push operations are the user's responsibility (nd may trigger fetches for sync, but does not manage Git workflows). | Confirmed       |
| A5   | Asset sources follow a conventional directory layout (`skills/`, `agents/`, `commands/`, `output-styles/`, `rules/`, `context/`, `plugins/`, `hooks/`), making auto-discovery possible without a manifest. | Confirmed       |
| A6   | Each coding agent has a discoverable configuration directory at a known, fixed path or is detectable via PATH presence. | Unconfirmed     |
| A7   | Users have Python 3.12+ available, or will install nd via `uv`, `pipx`, or a pre-built PyInstaller binary. | Confirmed       |
| A8   | All nd data (configuration, sources, profiles, snapshots, state, logs) is stored under `~/.config/nd/`. This departs from the XDG convention of splitting config and data across `~/.config/` and `~/.local/share/`, favoring simplicity and a single directory to manage. The choice of `~/.config/` over macOS-native `~/Library/` is a design decision for cross-platform consistency and developer familiarity. | Design decision |
| A9   | Context files (`CLAUDE.md`, `AGENTS.md`) deploy to different locations than other asset types: at global scope they Go directly into `~/.claude/`, and at project scope they Go at the project root (not inside `.claude/`). Context files are stored in named folders within the source's `context/` directory, and the symlink targets the file inside the folder. This matches Claude Code's documented behavior as of March 2026. | Confirmed       |
| A10  | Claude Code recognizes symlinks and follows them when loading skills, agents, commands, and rules from its config directories. Claude Code documentation confirms symlinks are resolved and circular symlinks are handled gracefully. | Confirmed       |

## Functional requirements

This section specifies nd's behaviors tagged by MoSCoW priority. Requirements are ordered by implementation dependency within each tier. The prioritization constraint is quality (the tool must be polished for open-source release), not a deadline or budget.

### Must have

- **[FR-001]** The tool provides a root command `nd` that, when invoked with no arguments, launches an interactive TUI session.
- **[FR-002]** The tool provides subcommands and flags for all core operations so that users can accomplish any task in a single command without entering the TUI.
- **[FR-003]** The tool reads configuration from a global YAML file at `~/.config/nd/config.yaml`.
- **[FR-004]** The tool reads an optional project-level YAML configuration file (`.nd/config.yaml` in the current directory) that overrides global settings.
- **[FR-005]** The user can register one or more local directories as asset sources via CLI command or config file.
- **[FR-006]** The user can register one or more Git repositories as asset sources via CLI command or config file. nd accepts GitHub shorthand (`owner/repo`), full Git URLs (HTTPS or SSH for any host including GitLab, Bitbucket, and self-hosted), and performs a full clone of the repository.
- **[FR-007]** When a source is registered, nd auto-discovers assets by scanning for conventional directories at the source root only: `skills/`, `agents/`, `commands/`, `output-styles/`, `rules/`, `context/`, `plugins/`, `hooks/`. Nested structures (e.g., `go-skills/skills/`) are not discovered by convention and require either an `nd-source.yaml` manifest to specify custom paths, or a configuration option in `config.yaml` to define additional scan roots within a source. This is an intentional limitation: many real asset libraries use nested layouts, so users with non-flat source structures should expect to configure custom paths on day one.
- **[FR-008]** When a source contains an `nd-source.yaml` manifest at its root, nd uses the manifest to override convention-based discovery (custom paths, asset metadata, exclusions). When `nd-source.yaml` is present, it completely replaces convention-based scanning for that source. Only paths listed in the manifest are scanned; convention directories not listed are ignored.
- **[FR-009]** The user can deploy a single asset to a coding agent's configuration directory by creating a symlink. The symlink (link) is created at the target location in the agent's config directory, pointing to the source asset file or directory. Equivalent to `os.symlink(source_asset_path, agent_config_path)`.
- **[FR-009a]** The tool creates absolute symlinks by default. The user can configure the default symlink strategy to `relative` in `config.yaml`. A `--relative` / `--absolute` CLI flag overrides the configured default for any deploy operation.
- **[FR-010]** The user can deploy multiple assets in bulk by specifying asset types, names, or source directories.
- **[FR-011]** The user can select between global scope (`~/.claude/` for Claude Code) and project scope (`.claude/` in current directory) before deploying.
- **[FR-012]** The user can remove one or more deployed assets, which deletes the symlinks without affecting the source files.
- **[FR-013]** The tool detects deployment issues: broken symlinks, symlinks pointing to moved or deleted source files, and symlinks that have been renamed outside of nd.
- **[FR-014]** The tool repairs detected deployment issues by re-creating symlinks to the correct source locations (sync operation). When a source asset no longer exists (renamed or deleted at the source), the tool removes the orphaned symlink and removes the asset from the deployment state.
- **[FR-015]** The user can view all currently deployed assets, grouped by asset type, showing source path, deploy scope, and health status.
- **[FR-016]** The tool detects installed coding agents by checking for known configuration directory structures and PATH presence. In v1, this detects Claude Code only, but the detection mechanism is extensible.
- **[FR-016a]** When two registered sources contain an asset with the same type and name, the first registered source wins (source priority ordering). The tool prints a warning identifying the duplicate and which source takes precedence. The user can override priority via config alias.
- **[FR-016b]** Context files are stored in named folders within the `context/` directory (e.g., `context/go-project-rules/CLAUDE.md`). The named folder is the asset identity; the file inside determines the deploy target. Only one context file can be deployed per target location (e.g., one `CLAUDE.md` at global scope). If a context file already exists at the target location, the tool warns the user and offers to back up the existing file before replacing it. Backups are stored in `~/.config/nd/backups/` with the naming format `{original-filename}.{ISO-timestamp}.bak`.
- **[FR-016c]** Context folders may contain an optional `_meta.yaml` file with metadata (description, tags, target language, target project, target agent). The CLI and TUI display this metadata when listing context assets to help users decide which context to deploy.
- **[FR-015a]** The user can list all discovered assets from all registered sources, filtered by asset type, source name, or deployment status. Both CLI and TUI provide this capability.
- **[FR-005a]** The user can unregister a source. Unregistering warns about any currently deployed assets from that source and requires confirmation before proceeding. Deployed assets from the removed source are not automatically removed.

### Should have

- **[FR-017]** The TUI displays a dashboard with a tabbed interface. Tabs include an overview tab and one tab per asset type.
- **[FR-018]** The dashboard always displays the current profile name, current scope (global or project), current coding agent, and deployment status (count of deployed assets and count of issues).
- **[FR-019]** The dashboard provides inline actions: deploy asset, check and fix issues, sync assets, remove assets, save current state, and switch profiles.
- **[FR-020]** The user can save the currently deployed asset set as a named snapshot. A snapshot records which assets are deployed, their source paths, and the scope.
- **[FR-021]** The user can restore a previously saved snapshot, deploying all assets recorded in the snapshot.
- **[FR-022]** The user can create a named profile, which is a curated collection of assets that can be deployed as a unit.
- **[FR-023]** The user can switch between profiles. Switching removes assets that belong to the current profile (but not pinned or manually deployed assets) and deploys the target profile's assets. The tool warns if any target profile assets conflict with existing pinned or manually deployed assets.
- **[FR-024]** The user can deploy all assets defined in a profile with a single command.
- **[FR-024a]** The user can pin individual assets so they persist across profile switches. Pinned assets are tracked separately in the deployment state and are never removed by a profile switch operation. When a user runs `nd remove` on a pinned asset, the tool warns that the asset is pinned and requires explicit confirmation before removing it.
- **[FR-025]** The tool provides a settings initialization workflow (interactive walkthrough with opinionated defaults) on first run or via `nd init`.
- **[FR-026]** The tool opens the configuration file in the user's default editor via `nd settings edit`.
- **[FR-027]** The tool syncs a Git-sourced repository by performing a `git pull` on the cloned repo when the user requests a source sync.
- **[FR-028]** The TUI main menu presents options: dashboard, each asset type, settings (init and edit), and quit. A "back" action (triggered by the Escape or Backspace key) is available on all screens except the main menu.
- **[FR-029]** When the user launches the TUI, the tool prompts for scope selection (global or project) and then lists detected coding agents for selection.
- **[FR-029a]** Before any bulk deploy, bulk remove, profile switch, or snapshot restore operation, the tool automatically saves the current deployment state as an auto-snapshot. If the operation fails midway, the tool reports the partial state and offers to restore the auto-snapshot. Auto-snapshots are retained for the last 5 auto-snapshots on disk and are distinct from user-created snapshots. Individual single-asset operations (deploy one, remove one) do not trigger auto-snapshots.

### Could have

- **[FR-030]** The user can export selected assets as a Claude Code plugin by generating the standard plugin directory structure: `.claude-plugin/plugin.json`, plus `commands/`, `agents/`, `skills/`, `hooks/`, and `README.md` as applicable.
- **[FR-031]** The user can generate a `marketplace.json` catalog file from one or more exported plugins, suitable for distribution as a Claude Code plugin marketplace.
- **[FR-032]** The plugin export workflow is interactive, guiding the user through asset selection, metadata entry (plugin name, description, version, author), and output location.
- **[FR-033]** The user can configure custom deployment locations for coding agents (overriding the default config directory paths) via the config file.
- **[FR-034]** The user can register additional context file types beyond the built-in set (`CLAUDE.md`, `AGENTS.md`, `AGENTS.local.md`, `CLAUDE.local.md`) via the config file. The named folder structure and `_meta.yaml` support apply to custom context file types as well.
- **[FR-035]** The tool provides shell completions for Bash, Zsh, and Fish.
- **[FR-036]** The tool provides a `--dry-run` flag for deploy, remove, and sync operations that shows what would happen without making changes.
- **[FR-036a]** The tool provides an `nd uninstall` command that lists all nd-managed symlinks across all known scopes, removes them, and optionally deletes nd's own directories (`~/.config/nd/`, `~/.cache/nd/`). The command requires explicit confirmation and supports `--dry-run`.
- **[FR-036b]** The tool maintains an operation log at `~/.config/nd/logs/operations.log` recording timestamp, operation type, assets affected, and result for each operation.

### Won't have (this time)

- **[FR-037]** Support for coding agents other than Claude Code (Codex, Gemini, OpenCode). Deferred because: v1 focuses on Claude Code to deliver a polished experience for one agent before generalizing. Reconsider when: v1 is stable and user demand for other agents is validated.
- **[FR-038]** File copy as a deployment strategy (alternative to symlinks). Deferred because: symlinks cover macOS use cases; copy introduces sync complexity. Reconsider when: Windows or Linux support is prioritized or users report symlink limitations.
- **[FR-039]** Template rendering for deployed assets (variable substitution in asset files at deploy time). Deferred because: adds complexity to the deployment model and blurs the line with content editing. Reconsider when: users demonstrate a concrete need for per-project asset customization.
- **[FR-040]** Profile inheritance (profiles that extend other profiles). Deferred because: adds complexity to profile resolution with minimal immediate benefit. Reconsider when: users report managing many similar profiles that differ by a few assets.
- **[FR-041]** Asset content editing or creation within nd. Deferred because: content authoring belongs in editors and AI agents, not in a deployment tool. This is a permanent design principle rather than a deferral; it will not be reconsidered.
- **[FR-042]** Memory file management (`MEMORY.md`, agent-specific memory files). Deferred because: memory files are created and maintained by Claude Code or agents at runtime, and managing them introduces ownership and staleness concerns. Reconsider when: v1 is stable and users demonstrate a need for memory backup and redeployment workflows.

**MoSCoW distribution:** Must: 22 (added FR-009a, FR-016a, FR-016b, FR-016c, FR-015a, FR-005a), Should: 15, Could: 9 (added FR-036b), Won't: 6 (added FR-042). Must Have ratio: 22/52 = 42%. Within the 60% ceiling.

## Non-functional requirements

This section defines quality attributes for nd. These are measurable constraints on how the system operates, not what it does.

- **[NFR-001]** Startup performance: The TUI must render the initial menu within 800ms of invocation on a machine with a local asset source of 500+ assets. Python's interpreter startup is slower than compiled languages, but Textual's lazy widget loading and deferred imports keep this target achievable.
- **[NFR-002]** Deploy performance: A single asset deploy (symlink creation) must complete within 100ms, excluding filesystem latency for the source check.
- **[NFR-003]** Bulk deploy performance: Deploying 50 assets must complete within 5 seconds.
- **[NFR-004]** Error reporting: Every operation that fails must produce a human-readable error message that names the specific asset, path, and failure reason. No silent failures.
- **[NFR-005]** Configuration validation: The tool must validate config files on load and report all validation errors with line numbers before proceeding.
- **[NFR-006]** Graceful degradation: If a registered source is unavailable (directory missing, repo not cloned), the tool must warn and continue operating with available sources rather than crashing.
- **[NFR-007]** Maintainability: The codebase must follow a `src` layout with `pyproject.toml`, use Protocol classes for agent-specific logic (to support future agents), and include Google-style docstrings with complete type annotations on all public functions, classes, and methods.
- **[NFR-008]** Distribution: The tool must be installable via `uv tool install nd`, `pipx install nd`, or `pip install nd` from PyPI with no system-level dependencies beyond Python 3.12+ and Git. `uv` is the recommended installation method. An optional PyInstaller build target produces a standalone binary for users who prefer not to manage a Python installation. PyInstaller builds for Textual apps require explicit data collection hooks for `.tcss` files and may produce binaries of 50-100 MB+. Code signing may be needed for macOS distribution.
- **[NFR-009]** Test coverage: Core packages (source discovery, symlink management, profile/snapshot operations) must have unit test coverage above 80%. Tests use pytest with `coverage.py`.

### Security requirements

- **[NFR-010]** Asset name validation: Asset names derived from source directory/file names must be validated before use in path construction. Names must match the pattern `[a-zA-Z0-9][a-zA-Z0-9._-]*` (alphanumeric start, then alphanumeric plus dots, hyphens, underscores). Names containing path separators (`/`, `\`) or parent directory references (`..`) must be rejected. The deploy engine must verify that all constructed symlink paths resolve within the expected target directory.
- **[NFR-011]** Source manifest path confinement: All paths specified in `nd-source.yaml` manifests must resolve within the source root directory. Absolute paths are rejected. Relative paths are resolved against the source root and validated to ensure they do not escape it (e.g., via `..` traversal). Paths that fail confinement checks are skipped with a warning.
- **[NFR-012]** Safe deserialization: All YAML and JSON files loaded from external sources (`nd-source.yaml`, `_meta.yaml`, `hooks.json`) must use safe loading modes that prevent arbitrary code execution. For ruamel.yaml, this means using the default safe loader. For JSON, this means using `json.loads()` (which is inherently safe). This requirement applies to any file parsed from a registered source directory, whether local or cloned from Git.
- **[NFR-013]** Source scanning safety: Source directory scanning must detect and gracefully handle circular symlinks, excessively deep directory nesting (max depth: 10 levels), and excessively large directories (warn if a single asset type directory contains more than 1000 entries). Circular symlinks are skipped with a warning. Scanning must not follow symlinks that point outside the source root.
- **[NFR-014]** Git clone protections: Git clone operations use `--depth 1` (shallow clone) by default. The user can configure full clones via `config.yaml` or the `--full-clone` flag on `nd source add`. Clone operations have a configurable timeout (default: 120 seconds). If a clone exceeds 500MB on disk, nd warns the user and requires confirmation to continue.
- **[NFR-015]** Symlink creation safety: The deploy engine must handle `FileExistsError` from `os.symlink()` as the authoritative conflict check rather than relying solely on pre-creation path checks (TOCTOU mitigation). When `FileExistsError` occurs, nd reports the conflict and follows the appropriate conflict resolution flow (FR-016b for context files, confirmation prompt for other types).

## User stories

This section describes key workflows from the user's perspective. Each story maps to functional requirements and includes acceptance criteria.

**US-001: Deploy skills to a new project.**
As a developer starting a new project, I want to deploy my preferred set of skills to the project's `.claude/` directory so that Claude Code has my custom capabilities available immediately.

- Acceptance criteria: I can run a single command (or select from the TUI) to deploy multiple skills from my registered source to the project's `.claude/` directory as symlinks. The tool confirms which assets were deployed and reports any errors.
- Related requirements: FR-009, FR-010, FR-011.

**US-002: Diagnose and fix broken deployments.**
As a developer who moved or renamed source directories, I want nd to tell me which deployments are broken and fix them so that I don't have to manually check each symlink.

- Acceptance criteria: Running a sync or check command lists all broken/drifted symlinks with specific paths and reasons. Running fix/sync re-creates the correct symlinks. The tool reports what it fixed.
- Related requirements: FR-013, FR-014, FR-015.

**US-003: Switch between project profiles.**
As a developer who works on different types of projects (Go CLI, web frontend, documentation), I want to switch between pre-configured asset profiles so that each project type gets the right set of skills and agents without losing my always-on assets.

- Acceptance criteria: I can create a named profile from a curated selection of assets. I can switch profiles with one command. Switching removes the previous profile's assets and deploys the new profile's assets, but pinned assets and manually deployed assets are left in place. The dashboard reflects the active profile and distinguishes pinned from profile-managed assets.
- Related requirements: FR-022, FR-023, FR-024, FR-024a, FR-018.

**US-004: Share assets with other developers (could have).**
As a developer with a curated asset library, I want to export selected assets as a Claude Code plugin so that other developers can install them with a single command.

- Acceptance criteria: An interactive workflow guides me through selecting assets, entering metadata, and choosing an output directory. The output is a valid Claude Code plugin directory structure that other users can install via `/plugin install`.
- Related requirements: FR-030, FR-031, FR-032.
- Priority note: This story maps to Could Have requirements and is not in scope for the initial implementation.

**US-005: Add a community asset source.**
As a developer who found a Git repository of useful skills, I want to add it as an asset source so that I can browse and deploy assets from it.

- Acceptance criteria: I can register the repo by providing a GitHub shorthand or full Git URL (any host). nd clones the repo and auto-discovers assets by directory convention. The assets appear in the TUI and CLI for deployment. I can sync the source to pull updates.
- Related requirements: FR-006, FR-007, FR-027.

**US-006: First-time setup.**
As a developer installing nd for the first time, I want a guided setup that configures defaults so that I can start deploying assets without reading documentation.

- Acceptance criteria: Running `nd init` walks me through setting up my first asset source, selecting a coding agent, and choosing global vs. project scope. Opinionated defaults are pre-selected. The resulting config file is valid and the tool is operational after init completes.
- Related requirements: FR-025, FR-003, FR-016.

## Technical design

This section captures high-level architecture decisions and component structure. It describes what communicates with what, not implementation details.

### Component overview

nd has five major components:

1. **Source manager.** Handles registration, discovery, and syncing of asset sources. Scans local directories and cloned repos for assets using convention-based directory layout. Reads optional `nd-source.yaml` manifests. Maintains an index of all known assets across all registered sources. When duplicate asset identities are found across sources, resolves by source registration order and emits warnings.

2. **Deploy engine.** Creates and manages symlinks (absolute by default, configurable to relative) between source assets and agent configuration directories. Handles single, bulk, snapshot, and profile deploys. Performs health checks (broken symlinks, drift detection) and repairs.

3. **Agent registry.** Detects installed coding agents by checking known config directory paths and PATH presence. In v1, contains only a Claude Code agent definition. Defines a Protocol class that future agents implement, specifying their config directory structure and asset location conventions.

4. **Profile and snapshot store.** Persists named profiles and point-in-time snapshots as YAML files within nd's data directory (`~/.config/nd/profiles/` and `~/.config/nd/snapshots/`). Profiles define which assets to deploy. Snapshots record the exact deployed state at a moment in time. Auto-snapshots are created before destructive bulk operations (bulk deploy, bulk remove, profile switch, snapshot restore) and the last 5 auto-snapshots are retained on disk.

5. **UI layer.** Two interfaces sharing the same underlying commands:
   - CLI layer (Typer): Subcommands and flags for every operation. Designed for scripting and single-command workflows.
   - TUI layer (Textual): Interactive dashboard with tabbed navigation, scope/agent selection, and inline actions.

### Data flow

```text
Asset sources (local dirs, cloned repos)
        │
        ▼
  Source manager (discover + index)
        │
        ▼
  Deploy engine (symlink create/remove/sync)
        │                │
        │                ▼
        │    Profile/Snapshot store
        │    (~/.config/nd/profiles/,
        │     ~/.config/nd/snapshots/)
        │                │
        │                ▼
        ├──► Deployment state
        │    (~/.config/nd/state/deployments.yaml)
        │
        ▼
  Agent config directories (~/.claude/, .claude/)
```

The deploy engine reads from and writes to both the Profile/Snapshot store and the deployment state file on every operation. Profile switches and snapshot restores flow through the deploy engine, which updates `deployments.yaml` after each change.

### Configuration hierarchy

Configuration resolves in this order (later overrides earlier):

1. Built-in defaults (opinionated, ships with nd).
2. Global config (`~/.config/nd/config.yaml`).
3. Project config (`.nd/config.yaml` in current working directory).
4. CLI flags (highest priority, overrides everything).

### Key technology choices

| Choice             | Rationale                                                    |
| ------------------ | ------------------------------------------------------------ |
| Python 3.12+       | Modern type syntax (`type` statement, `TypeAlias`), `tomllib` in stdlib, broad developer adoption, strong CLI and TUI ecosystem. |
| Typer (via Click)  | Type-hint-driven CLI framework. Subcommand structure, flag parsing, help generation, shell completions. Reduces boilerplate compared to raw argparse. |
| Textual            | Python TUI framework with a component model, CSS-like styling, and async support. Active community, good documentation, maintained by the Textualize team. |
| Rich               | Terminal styling companion to Textual. Tables, panels, progress bars, and syntax highlighting for CLI output. |
| ruamel.yaml        | YAML parsing with round-trip comment preservation. Users can edit config files without losing comments or formatting. Use for user-edited files (`config.yaml`, `nd-source.yaml`, profiles) where comment preservation matters. Machine-managed files (`deployments.yaml`, auto-snapshots) may use Pydantic's JSON serialization for better performance. |
| os.symlink         | Zero-copy deployment. Source changes are immediately reflected. No sync lag. Absolute by default, configurable to relative. |
| Pydantic v2+       | Config and state model validation. Typed dataclasses with automatic YAML/JSON deserialization, validation errors with field paths, and schema generation. v2 is required for startup performance (NFR-001). The `pydantic.mypy` plugin must be configured for strict mode compatibility. |
| GitPython          | Git clone and pull operations for registered Git sources. Thin wrapper over Git CLI. Falls back gracefully if Git is not installed. For v1, direct `subprocess.run(['git', ...])` calls are an acceptable lighter alternative, since nd only uses clone and pull operations. |

### Project structure

```text
nd/
├── pyproject.toml           # Project metadata, dependencies, build config
├── README.md
├── src/
│   └── nd/
│       ├── __init__.py
│       ├── __main__.py      # Entry point (python -m nd)
│       ├── cli/             # Typer CLI layer
│       │   ├── __init__.py
│       │   ├── app.py       # Root Typer app, subcommand registration
│       │   ├── deploy.py    # Deploy/remove subcommands
│       │   ├── source.py    # Source add/remove/sync subcommands
│       │   ├── profile.py   # Profile/snapshot subcommands
│       │   └── settings.py  # Init, edit subcommands
│       ├── tui/             # Textual TUI layer
│       │   ├── __init__.py
│       │   ├── app.py       # Textual App, screen routing
│       │   ├── screens/     # Individual TUI screens
│       │   └── widgets/     # Reusable TUI widgets
│       ├── core/            # Business logic (no UI dependencies)
│       │   ├── __init__.py
│       │   ├── source.py    # Source manager
│       │   ├── deploy.py    # Deploy engine
│       │   ├── agent.py     # Agent registry and Protocol
│       │   ├── profile.py   # Profile and snapshot store
│       │   └── config.py    # Configuration loading and hierarchy
│       └── models/          # Pydantic models for config, state, assets
│           ├── __init__.py
│           ├── config.py
│           ├── asset.py
│           ├── profile.py
│           └── state.py
└── tests/
    ├── conftest.py          # Shared fixtures (tmp_path sources, mock agents)
    ├── test_source.py
    ├── test_deploy.py
    ├── test_profile.py
    └── test_config.py
```

The `core/` package contains all business logic and has no dependency on the CLI or TUI packages. Both UI layers import from `core/` and `models/`, never from each other. This separation allows testing core logic without UI dependencies and makes it straightforward to add new UI layers in the future.

### Asset deployment mapping (Claude Code v1)

Each asset type maps to a specific target location within Claude Code's configuration. The deploy engine must use this mapping table to determine where symlinks are created. The "link path" column shows where `os.symlink(target, link)` places the link; the "target" is always the asset in the source directory.

**Global scope deployment (`~/.claude/`):**

| Source type   | Source structure                         | Link created at                         | Notes                                                        |
| ------------- | ---------------------------------------- | --------------------------------------- | ------------------------------------------------------------ |
| skills        | `skills/skill-name/` (directory)         | `~/.claude/skills/skill-name`           | Directory symlink.                                           |
| agents        | `agents/agent-name.md` (file)            | `~/.claude/agents/agent-name.md`        | File symlink.                                                |
| commands      | `commands/cmd-name.md` (file)            | `~/.claude/commands/cmd-name.md`        | File symlink.                                                |
| output-styles | `output-styles/style-name.md` (file)     | `~/.claude/output-styles/style-name.md` | File symlink. nd creates the `output-styles/` directory if it does not exist. The user must register the output-style in `settings.json` or `settings.local.json` manually. |
| rules         | `rules/rule-name.md` (file)              | `~/.claude/rules/rule-name.md`          | File symlink. Rules may also be directories with nested `.md` files. |
| context       | `context/ctx-name/CLAUDE.md` | `~/.claude/CLAUDE.md`                   | Symlink targets the context file inside the named folder (e.g., `context/go-project-rules/CLAUDE.md`), not the folder itself. The deploy target is determined by the filename inside the folder. Only one context file can be deployed per target location; deploying a second offers to back up and replace the existing one. Each context folder may contain an optional `_meta.yaml` with metadata (description, tags, target language, target project, target agent) displayed by the CLI to help users choose which context to deploy. |
| plugins       | `plugins/plugin-name/` (directory)       | Managed by Claude Code's plugin system  | nd does not symlink plugins directly. Plugin deploy uses `nd export` to produce plugin format, then the user installs via Claude Code's `/plugin install`. |
| hooks         | `hooks/hook-name/` (directory)           | `~/.claude/hooks/hook-name`             | Directory symlink. Each hook folder contains a `hooks.json` config, an executable script (any supported language), and an optional `README.md`. nd creates the `hooks/` directory if it does not exist. The user must also register the hook in `settings.json` or `settings.local.json` manually. Hooks can alternatively be deployed via plugin export (FR-030). |

**Project scope deployment (`.claude/` in project root):**

| Source type               | Source structure                     | Link created at                        | Notes                                                        |
| ------------------------- | ------------------------------------ | -------------------------------------- | ------------------------------------------------------------ |
| skills                    | `skills/skill-name/` (directory)     | `.claude/skills/skill-name`            | Directory symlink.                                           |
| agents                    | `agents/agent-name.md` (file)        | `.claude/agents/agent-name.md`         | File symlink.                                                |
| commands                  | `commands/cmd-name.md` (file)        | `.claude/commands/cmd-name.md`         | File symlink.                                                |
| output-styles             | `output-styles/style-name.md` (file) | `.claude/output-styles/style-name.md`  | File symlink. The user must register the output-style in `settings.json` or `settings.local.json` manually. |
| rules                     | `rules/rule-name.md` (file)          | `.claude/rules/rule-name.md`           | File symlink.                                                |
| context (CLAUDE.md)       | `context/ctx-name/CLAUDE.md`         | `./CLAUDE.md` (project root)           | Symlink targets the file inside the named folder. Placed at the project root, outside `.claude/`. Only one per target location. |
| context (CLAUDE.local.md) | `context/ctx-name/CLAUDE.local.md`   | `./CLAUDE.local.md` (project root)     | Placed at the project root, outside `.claude/`. Only one per target location. |
| context (AGENTS.md)       | `context/ctx-name/AGENTS.md`         | `./AGENTS.md` (project root)           | Placed at the project root, outside `.claude/`. Only one per target location. |
| context (AGENTS.local.md) | `context/ctx-name/AGENTS.local.md`   | `./AGENTS.local.md` (project root)     | Placed at the project root, outside `.claude/`. Only one per target location. |
| plugins                   | `plugins/plugin-name/` (directory)   | Managed by Claude Code's plugin system | Same as global: use `nd export` then `/plugin install`.      |
| hooks                     | `hooks/hook-name/` (directory)       | `.claude/hooks/hook-name`              | Directory symlink. Same structure as global (script, `hooks.json`, optional `README.md`). The user must register the hook in `settings.json`, `settings.local.json`, or `.claude/settings.local.json` manually. Can also deploy via plugin export. |

Context files are special cases in two ways. First, unlike other asset types that deploy into subdirectories of the agent's config directory, context files deploy to specific fixed paths determined by the filename inside the named folder (not the folder name). `CLAUDE.md` and `AGENTS.md` Go directly into `~/.claude/` at global scope, or at the project root (not inside `.claude/`) at project scope. `CLAUDE.local.md` and `AGENTS.local.md` always deploy at the project root. Second, context files are exclusive per deployment location: only one context file can occupy a given target path at a time. If a context file already exists at the target, nd warns the user and offers to back up the existing file before replacing it. The named folder structure (e.g., `context/go-project-rules/CLAUDE.md`) allows users to maintain multiple context files for different purposes and switch between them. Each folder may contain an optional `_meta.yaml` file with metadata such as description, tags, target language, target project, and target agent. The CLI displays this metadata to help users choose which context to deploy.

Assets are either files or directories. Skills and plugins are always directories (a skill is `skill-name/SKILL.md` plus optional subdirectories; a plugin is `plugin-name/` with a `.claude-plugin/` subdirectory). Commands, agents, output-styles, rules are typically single files. Context files are stored as directories (named folder containing the context file and optional `_meta.yaml`), but the symlink targets the file inside, not the directory. Hooks are directories containing a `hooks.json` configuration file, an executable script in any supported language, and an optional `README.md`. The deploy engine must handle all variants: `os.symlink(source_path, link_path)` works for both files and directories, but health checks, display logic, and context file exclusivity checks must account for the differences.

### Asset identity

Each asset is uniquely identified by the tuple: `(source_id, asset_type, asset_name)`. The `source_id` is a user-assigned name or an auto-generated identifier for each registered source. The `asset_type` is one of the conventional directory names. The `asset_name` is the file or directory name within that type directory (without extension for files). For context files, the `asset_name` is the named folder (e.g., `go-project-rules`), not the filename inside it. This identity is used in profiles, snapshots, deployment state, and conflict detection.

When two sources contain an asset with the same `(asset_type, asset_name)`, the source registered first takes priority. The tool emits a warning identifying both sources and which one wins.

```text
source-root/
├── nd-source.yaml      # Optional manifest (overrides conventions)
├── skills/
│   ├── skill-one/
│   │   └── SKILL.md
│   └── skill-two/
│       └── SKILL.md
├── agents/
│   └── agent-one.md
├── commands/
│   ├── review.md
│   └── deploy.md
├── output-styles/
│   └── style-one.md
├── rules/
│   └── rule-one.md
├── context/
│   ├── go-project-rules/
│   │   ├── CLAUDE.md
│   │   └── _meta.yaml
│   └── agent-instructions/
│       ├── AGENTS.md
│       └── _meta.yaml
├── plugins/
│   └── plugin-one/
│       └── .claude-plugin/
│           └── plugin.json
└── hooks/
    └── hook-one/
        ├── hook-one.py       # Hook script (any supported language)
        ├── README.md         # About the hook
        └── hooks.json        # Hook configuration
```

### nd-source.yaml manifest

The `nd-source.yaml` manifest allows sources with non-standard directory layouts to define custom asset paths and exclusions. This is required for sources that use nested structures (e.g., `go-skills/skills/`).

Minimal skeleton:

```yaml
# nd-source.yaml
# Override convention-based asset discovery for this source.

paths:
  skills:
    - skills/               # Standard location
    - go-skills/skills/     # Nested layout
  agents:
    - agents/
  commands:
    - commands/

exclude:
  - experimental/           # Exclude a directory from all discovery
  - skills/deprecated/      # Exclude a specific asset path
```

The full schema for `nd-source.yaml` (including metadata, tags, and categories) will be designed during implementation. This skeleton provides a starting point for the `paths` and `exclude` fields.

### nd directory structure

nd consolidates its data under `~/.config/nd/`. Configuration, persistent data (cloned sources, profiles, snapshots, deployment state), and logs all live under this directory. Ephemeral data goes in `~/.cache/nd/`.

```text
~/.config/nd/
├── config.yaml          # Global configuration
├── logs/
│   └── operations.log   # Operation history (Could Have, FR-036b)
├── sources/             # Cloned Git repos
│   └── owner-repo/
├── profiles/
│   └── profile-name.yaml
├── snapshots/
│   ├── user/            # User-created snapshots
│   │   └── snapshot-name.yaml
│   └── auto/            # Auto-snapshots (last 5 retained)
│       └── auto-2026-03-13T10-30-00.yaml
└── state/
    └── deployments.yaml # Tracks current deployment state

~/.cache/nd/
└── index/               # Asset discovery cache (rebuildable)
```

## Boundaries

This section defines behavior tiers for AI agents implementing this spec. These are constraints on how an implementing agent should operate.

### Always

- Always validate that a source directory exists and is readable before attempting asset discovery.
- Always verify that the target agent config directory exists before creating symlinks.
- Always check for existing files or symlinks at the target path before deploying, and report conflicts rather than silently overwriting.
- Always write deployment state changes to `~/.config/nd/state/deployments.yaml` after any deploy, remove, or sync operation.
- Always use the configuration hierarchy (defaults, global config, project config, CLI flags) when resolving settings.
- Always run tests for core packages (source discovery, symlink management, profile/snapshot operations) before committing changes.
- Always validate asset names against the allowed pattern before constructing symlink paths.
- Always verify that constructed symlink paths resolve within the expected target directory.
- Always use safe YAML/JSON loading for files from registered sources.

### Ask-first

- Ask before removing deployed assets that are not managed by nd (symlinks or files that nd did not create).
- Ask before overwriting an existing profile or snapshot with the same name.
- Ask before performing a source sync (`git pull`) that is triggered autonomously (e.g., as part of a bulk deploy that detects a stale source). User-initiated sync commands (e.g., `nd sync`) do not require confirmation.
- Ask before creating the `.nd/` directory in a project (project-level config initialization).
- Ask before modifying any file outside of nd's own directories (`~/.config/nd/`).

### Never

- Never modify the content of source asset files. nd reads and symlinks assets but never writes to them.
- Never execute Git commands beyond clone and pull (no commit, push, branch, merge, rebase).
- Never delete source asset files. Removal operations only delete symlinks in the target agent config directory.
- Never store secrets, API keys, or credentials in nd's configuration files.
- Never make network requests other than Git clone and pull operations for registered Git sources.
- Never follow symlinks that point outside a source root directory during scanning.
- Never construct symlink paths using unsanitized asset names from external sources.

## Success criteria

1. A user with 500+ assets in a local source directory can deploy, remove, and sync assets without errors and without manual symlink management. Verified by: end-to-end test with a 500-asset source directory.
2. A user can complete first-time setup (`nd init`) and deploy their first asset within 5 minutes of installing the tool. Verified by: timed walkthrough with a new user unfamiliar with the tool.
3. The TUI dashboard accurately reflects the current deployment state (deployed assets, issues, active profile) at all times. Verified by: deploy, remove, and sync operations from both CLI and TUI, then checking dashboard consistency.
4. A user can export a set of assets as a valid Claude Code plugin that installs successfully via Claude Code's `/plugin install` command. Verified by: export a plugin from nd and install it in a fresh Claude Code environment.
5. A user can switch profiles and the resulting deployment state matches the target profile exactly (no leftover assets from the previous profile, no missing assets from the new profile), while pinned and manually deployed assets remain untouched. Verified by: profile switch test comparing expected vs. actual symlink state, including verification that pinned assets survive the switch.
6. The tool handles edge cases without crashing: missing source directories, broken symlinks, empty sources, duplicate asset names across sources, and read-only target directories. Verified by: automated tests for each edge case.
7. The tool passes Ruff (linting and formatting) and mypy in strict mode with no errors, and has >80% test coverage on core packages. Verified by: CI pipeline checks.
8. The tool ships with a `README.md` covering installation and first-time setup, and complete `nd help` output for all commands, before the first public release. Verified by: review of documentation against the command tree.
9. The tool has no open blocking bugs at the time of first public release. Verified by: issue tracker review.

## Open questions

| #    | Question                                                     | Category             | Impact                                                       |
| ---- | ------------------------------------------------------------ | -------------------- | ------------------------------------------------------------ |
| Q1   | What is the full format and schema for `nd-source.yaml` manifests beyond the `paths` and `exclude` fields? Should it support metadata like asset descriptions, tags, and categories? | Non-blocking         | A minimal skeleton is defined in the Technical design. The full schema can be iterated on after v1. |
| Q3   | What specific fields should `plugin.json` contain when exporting assets as Claude Code plugins? | Non-blocking         | Plugin export is a "could have" feature. Schema can be defined when that feature is implemented. |
| Q4   | ~~Should profiles store asset references (names and source identifiers) or absolute paths?~~ **Resolved:** Profiles store asset references `(source_id, asset_type, asset_name)` as primary, with the absolute path cached as a hint. Resolution uses the reference first; falls back to the cached path if the source is not registered. | Resolved             | Closed. |
| Q5   | ~~How should nd handle a source sync (`git pull`) that deletes or renames assets that are currently deployed?~~ **Resolved:** nd removes the orphaned symlink and removes the asset from the deployment state (FR-014). | Resolved             | Closed. |
| Q6   | What is the correct detection method for each coding agent? Claude Code uses `~/.claude/`, but what about Codex, Gemini, and OpenCode? | Non-blocking         | v1 only supports Claude Code. Detection for other agents can be researched when multi-agent support is added. |
| Q7   | Should the deployment state file (`deployments.yaml`) be human-editable, or is it purely machine-managed? | Non-blocking         | Affects schema complexity and validation strictness.         |
| Q8   | How should nd handle asset sources on case-insensitive filesystems (macOS default) where directory names like `Skills/` and `skills/` would collide? | Assumption-dependent | Assumed that convention uses lowercase directory names and users follow this. If assumption is wrong, collision detection logic is needed. |
| Q9   | ~~Should nd use a single PyInstaller binary as the default distribution, or prioritize PyPI/pipx with PyInstaller as an optional build?~~ **Resolved:** `uv tool install` is the recommended method, with `pipx` and `pip` also supported. PyInstaller is an optional build target. | Resolved         | Closed. |
| Q13  | Should nd post-deploy print a reminder when deployed assets require manual `settings.json` changes (e.g., registering an output-style or a hook)? | Non-blocking         | Affects UX polish but not core functionality. Relevant for hooks and output-styles, which require manual settings registration. |

## Changelog

| Version | Date       | Author | Changes                                                      |
| ------- | ---------- | ------ | ------------------------------------------------------------ |
| 0.8     | 2026-03-14 | Larah  | Audit remediation: added CLI command reference section, security requirements (NFR-010 through NFR-015), deployment state schema with atomic writes and journaling, schema versioning and migration, graceful degradation for corrupted state; promoted FR-027 and FR-029a to Must Have; qualified Goal 5 as stretch goal; resolved plugin v1 exclusions; added profile population mechanism, snapshot user story (US-007) and success criterion (SC-10); restored and resolved Q2, Q10, Q11, Q12, Q13; completed global deployment table; clarified context folder semantics, directory creation policy, conflict resolution, git pull failure handling; added implementation guidance to NFR-001, NFR-008, NFR-009. |
| 0.7     | 2026-03-14 | Larah  | Profile format: closed Q4 — profiles store asset references `(source_id, asset_type, asset_name)` as primary with absolute path cached as fallback hint. |
| 0.6     | 2026-03-14 | Larah  | Distribution: added `uv tool install` as recommended installation method in A7 and NFR-008; closed Q9 as resolved (`uv` primary, `pipx`/`pip` also supported, PyInstaller optional). |
| 0.5     | 2026-03-14 | Larah  | Orphan removal: updated FR-014 to specify that orphaned symlinks (source asset renamed or deleted) are automatically removed along with their deployment state entry; closed Q5 as resolved. |
| 0.4     | 2026-03-14 | Larah  | Python migration: replaced Go 1.23+ with Python 3.12+; replaced Cobra with Typer, Bubble Tea with Textual, Lip Gloss with Rich, gopkg.in/yaml.v3 with ruamel.yaml; added Pydantic for model validation and GitPython for Git operations; replaced Go interface with Python Protocol class in agent registry; replaced golangci-lint with Ruff + mypy (strict) in success criteria; replaced GoDoc with Google-style docstrings and type annotations in NFR-007; replaced single static binary (NFR-008) with PyPI/pipx as primary distribution and optional PyInstaller binary; relaxed NFR-001 startup target from 500ms to 800ms to account for Python interpreter startup; added src layout project structure to technical design; added Q9 (distribution strategy); updated A7 for Python/pipx/PyInstaller; updated os.Symlink references to os.symlink throughout. |
| 0.3     | 2026-03-14 | Larah  | Remediation revision: consolidated all nd data under `~/.config/nd/` (removed `~/.local/share/nd/` split); restructured context files into named folders with optional `_meta.yaml` metadata, added deploy exclusivity with backup offer (FR-016b, FR-016c); resolved hooks deployment (symlink to `.claude/hooks/` plus manual settings.json registration, or via plugin export); confirmed output-styles directory support and added settings.json registration note; removed SOUL.md references (not supported by Claude Code); deferred memory file management to Won't Have (FR-042) with non-goal entry; added both/configurable symlink strategy with absolute default (FR-009a); added source priority ordering with warnings for duplicate assets (FR-016a); narrowed ask-first sync rule to autonomous syncs only; removed "back" from main menu, added Escape/Backspace navigation (FR-028); moved release-readiness from Goal 7 to Success criteria 8-9; added pinned asset removal warning behavior (FR-024a); clarified nested source layout limitation in FR-007 with explicit callout; reclassified A8 as design decision; updated data flow diagram to include Profile/Snapshot Store and deployments.yaml; refined auto-snapshot triggers to bulk deploy, bulk remove, profile switch, snapshot restore only (FR-029a); added operation logging as Could Have (FR-036b); added README and nd-help success criteria; added nd-source.yaml manifest skeleton to Technical design; labeled US-004 as Could Have; closed Q2 (duplicate assets), Q9 (merged into Q1), Q10 (deployment state file), Q11 (hooks), Q12 (output-styles). |
| 0.2     | 2026-03-14 | Larah  | Audit revision: added commands as asset type; added asset deployment mapping table with context file special cases; added asset identity definition; clarified symlink direction in FR-009; added pinned assets (FR-024a) for profile switching safety; added auto-snapshots (FR-029a) for rollback on bulk operations; fixed XDG compliance (sources/state in ~/.config/nd/); broadened Git support beyond GitHub; added uninstall command (FR-036a); added settings.json non-goal with gap explanation; clarified root-only source scanning in FR-007; fixed .mcp.json filename; added assumptions A9 (context file locations) and A10 (symlink resolution); added open questions Q11 (hooks format), Q12 (output-styles directory), Q13 (settings.json reminders). |
| 0.1     | 2026-03-13 | Larah  | Initial draft from elicitation.                              |
