# Napoleon dynamite (nd) spec (Go)

| Field | Value |
| -------------------- | ---------- |
| **Date** | 2026-03-13 |
| **Author** | Larah |
| **Status** | Implemented (TUI deferred) |
| **Version** | 0.7 |
| **Last reviewed** | 2026-03-22 |
| **Last reviewed by** | Larah |

## Section index

- **Problem statement:** Defines the asset management problem that nd solves for coding agent users.
- **Goals:** Lists the measurable outcomes nd aims to achieve.
- **Non-goals:** Names what nd will not do and the scope creep each exclusion prevents.
- **Assumptions:** States conditions believed true but not verified that the spec depends on.
- **Functional requirements:** Specifies nd's behaviors tagged by MoSCoW priority, ordered by implementation dependency.
- **Non-functional requirements:** Defines quality attributes for performance, reliability, and maintainability.
- **User stories:** Describes key workflows from the user's perspective with acceptance criteria.
- **Technical design:** Captures high-level architecture, component structure, technology decisions, asset deployment mapping, asset identity model, CLI command reference, deployment state schema, profile scope semantics, and error behavior.
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
5. A developer can export selected assets as a Claude Code plugin or marketplace for distribution. (Note: Plugin export is a Could-Have feature for v1. The architecture supports it, but delivery is not guaranteed in the initial release.)
6. The tool provides both a direct CLI interface (single command to accomplish a task) and an interactive TUI experience (launch and navigate).

**Progressive complexity:** nd is designed for progressive disclosure. Users encounter concepts in stages:

- **Level 0 (first 5 minutes):** Sources, assets, deploy. A user can run `nd source add ~/my-skills && nd deploy skill-name` and be productive.
- **Level 1 (first week):** Scopes (global vs. project), status, sync.
- **Level 2 (power user):** Profiles, snapshots, pinned assets.
- **Level 3 (advanced):** Manifests (`nd-source.yaml`), plugin export, custom context file types.

Documentation, help text, and the TUI should reflect these levels, presenting Level 0 concepts prominently and deferring advanced concepts until the user needs them.

## Non-goals

- **Editing asset content.** nd deploys and organizes assets but never modifies their content. Content authoring belongs in the user's editor or AI agent. Adding editing would blur nd's role and create conflict with version-controlled source files.
- **Being a Git client.** nd clones repos to use as asset sources and can trigger sync operations, but it does not expose Git commands (commit, push, pull, branch, merge). Git operations are the user's responsibility. Wrapping Git would duplicate existing tooling and create maintenance burden. Note: nd performs `git clone` and `git pull` as internal implementation details of source management (FR-006, FR-027). These are not exposed as user-facing Git commands.
- **Managing MCP server configurations.** MCP config (`.mcp.json`) has its own structure, lifecycle, and tooling. Mixing it with asset deployment would complicate nd's config model and conflict with agent-native MCP management.
- **Cross-machine synchronization.** nd manages assets on a single machine. Syncing across machines is the responsibility of Git, cloud storage, or dotfile managers. Adding sync would require conflict resolution, authentication, and networking concerns that are orthogonal to asset management.
- **Version control of assets.** nd does not track asset history, diffs, or revisions. The asset source (a Git repo or local directory) owns version control. Duplicating this would conflict with Git and create confusion about the source of truth.
- **Supporting coding agents beyond Claude Code in v1.** The architecture supports multiple agents, but v1 targets Claude Code only. Premature multi-agent support would slow the launch and dilute testing focus. (see FR-037)
- **Modifying agent settings files.** nd does not write to Claude Code's `settings.json`, `settings.local.json`, or `.claude.json`. Some asset types (hooks, output-styles, enabled plugins) are registered in these files, which means nd cannot fully automate their activation. This is an intentional gap: modifying agent-managed settings files risks conflicts with the agent's own config management. Users must manually enable hooks or output-styles in their settings after deploying assets that require it. This gap may be revisited if Claude Code exposes a CLI or API for settings management.
- **Managing memory files.** Claude Code memory files (`MEMORY.md`, agent-specific memory) are created and maintained by Claude Code or specific agents at runtime. While they could theoretically be backed up and redeployed, managing them introduces complexity around ownership and staleness. Memory support may be added in a future version. (see FR-042)

## Assumptions

| # | Assumption | Status |
| ---- | ------------------------------------------------------------ | --------------- |
| A1 | Claude Code's configuration directory structure (`~/.claude/` for global, `.claude/` for project-level) is stable and will not change in breaking ways. | Unconfirmed |
| A2 | The Claude Code plugin format (`.claude-plugin/plugin.json`, `commands/`, `agents/`, `skills/`, `hooks/`) is stable enough to target for export. | Unconfirmed |
| A3 | Symlinks are reliable on macOS for this use case (no permission issues, no filesystem limitations for typical developer setups). | Confirmed |
| A4 | Users manage their asset source repositories outside of nd. Git clone, pull, and push operations are the user's responsibility (nd may trigger fetches for sync, but does not manage Git workflows). | Confirmed |
| A5 | Asset sources follow a conventional directory layout (`skills/`, `agents/`, `commands/`, `output-styles/`, `rules/`, `context/`, `plugins/`, `hooks/`), making auto-discovery possible without a manifest. | Confirmed |
| A6 | Each coding agent has a discoverable configuration directory at a known, fixed path or is detectable via PATH presence. | Unconfirmed |
| A7 | Users have Go 1.23+ available for building from source, or will install nd via a pre-built binary. | Confirmed |
| A8 | All nd data (configuration, sources, profiles, snapshots, state, logs) is stored under `~/.config/nd/`. This departs from the XDG convention of splitting config and data across `~/.config/` and `~/.local/share/`, favoring simplicity and a single directory to manage. The choice of `~/.config/` over macOS-native `~/Library/` is a design decision for cross-platform consistency and developer familiarity. | Design decision |
| A9 | Context files (`CLAUDE.md`, `AGENTS.md`, `CLAUDE.local.md`, `AGENTS.local.md`) deploy to different locations than other asset types: at global scope they deploy directly into `~/.claude/`, and at project scope they deploy at the project root (not inside `.claude/`). Context files are stored in named folders within the source's `context/` directory, and the symlink targets the file inside the folder. `.local.md` variants (`CLAUDE.local.md`, `AGENTS.local.md`) deploy only at project scope (never at global scope). This matches Claude Code's documented behavior as of March 2026. | Confirmed |
| A10 | Claude Code recognizes symlinks and follows them when loading skills, agents, commands, and rules from its config directories. Claude Code documentation confirms symlinks are resolved and circular symlinks are handled gracefully. | Confirmed |

## Functional requirements

This section specifies nd's behaviors tagged by MoSCoW priority. Requirements are ordered by implementation dependency within each tier. The prioritization constraint is quality (the tool must be polished for open-source release), not a deadline or budget.

### Must have

- **[FR-001]** The tool provides a root command `nd` that, when invoked with no arguments, launches an interactive TUI session.
- **[FR-002]** The tool provides subcommands and flags for all core operations so that users can accomplish any task in a single command without entering the TUI.
- **[FR-003]** The tool reads configuration from a global YAML file at `~/.config/nd/config.yaml`. When the config file fails validation, the tool prints all validation errors with line numbers and exits with a non-zero exit code. No operations proceed until the config is valid.
- **[FR-004]** The tool reads an optional project-level YAML configuration file (`.nd/config.yaml` in the current directory) that overrides global settings.
- **[FR-005]** The user can register one or more local directories as asset sources via CLI command or config file.
- **[FR-006]** The user can register one or more Git repositories as asset sources via CLI command or config file. nd accepts GitHub shorthand (`owner/repo`), full Git URLs (HTTPS or SSH for any host including GitLab, Bitbucket, and self-hosted), and performs a full clone of the repository.
- **[FR-007]** When a source is registered, nd auto-discovers assets by scanning for conventional directories at the source root only: `skills/`, `agents/`, `commands/`, `output-styles/`, `rules/`, `context/`, `plugins/`, `hooks/`. Nested structures (e.g., `go-skills/skills/`) are not discovered by convention and require either an `nd-source.yaml` manifest to specify custom paths, or a configuration option in `config.yaml` to define additional scan roots within a source. This is an intentional limitation: many real asset libraries use nested layouts, so users with non-flat source structures should expect to configure custom paths on day one.
- **[FR-008]** When a source contains an `nd-source.yaml` manifest at its root, nd uses the manifest to override convention-based discovery (custom paths, asset metadata, exclusions).
- **[FR-009]** The user can deploy a single asset to a coding agent's configuration directory by creating a symlink. The symlink (link) is created at the target location in the agent's config directory, pointing to the source asset file or directory. Equivalent to `os.Symlink(sourceAssetPath, agentConfigPath)`.
- **[FR-010]** The user can deploy multiple assets in bulk by specifying asset types, names, or source directories. If individual asset deploys fail during a bulk operation, the tool continues with remaining assets (fail-open). After completion, it outputs a summary showing succeeded and failed counts, with per-asset error details for failures.
- **[FR-011]** The user can select between global scope (`~/.claude/` for Claude Code) and project scope (`.claude/` in current directory) before deploying. When deploying to project scope and the `.claude/` directory does not exist, the tool creates it automatically. Subdirectories (e.g., `.claude/skills/`) are also created as needed.
- **[FR-012]** The user can remove one or more deployed assets, which deletes the symlinks without affecting the source files.
- **[FR-013]** The tool detects deployment issues: broken symlinks, symlinks pointing to moved or deleted source files, and symlinks that have been renamed outside of nd.
- **[FR-014]** The tool repairs detected deployment issues by re-creating symlinks to the correct source locations (sync operation). When a source asset no longer exists (renamed or deleted at the source), the tool removes the orphaned symlink and removes the asset from the deployment state.
- **[FR-015]** The user can view all currently deployed assets, grouped by asset type, showing source path, deploy scope, and health status.
- **[FR-016]** The tool detects installed coding agents by checking for known configuration directory structures and PATH presence. In v1, this detects Claude Code only, but the detection mechanism is extensible.
- **[FR-016a]** When two registered sources contain an asset with the same type and name, the first registered source wins (source priority ordering). The tool prints a warning identifying the duplicate and which source takes precedence. The user can override priority via config alias.
- **[FR-016b]** Context files are stored in named folders within the `context/` directory (e.g., `context/go-project-rules/CLAUDE.md`). The named folder is the asset identity; the file inside determines the deploy target. Only one context file can be deployed per target location (e.g., one `CLAUDE.md` at global scope). If a context file already exists at the target location, the tool warns the user and offers to back up the existing file before replacing it. Backups are stored in `~/.config/nd/backups/` with the naming convention `<filename>.<ISO-8601-timestamp>.bak`. The last 5 backups per target location are retained. When the existing file is not an nd-managed symlink, the tool uses stronger warning language indicating it is a manually created file.
- **[FR-016d]** The user can list all discovered assets across all registered sources, filtered by asset type, source, or name pattern. The output shows asset name, type, source, and deployment status (deployed/available/broken).
- **[FR-043]** The user can unregister (remove) a source. The tool warns about any currently deployed assets from that source and any profiles referencing those assets. The user can choose to remove deployed assets, leave them as orphaned symlinks, or cancel the operation.
- **[FR-022]** The user can create a named profile, which is a curated collection of assets that can be deployed as a unit.
- **[FR-024]** The user can deploy all assets defined in a profile with a single command.

**Note:** Plugins are discoverable and indexable but are excluded from the deploy, remove, sync, profile, and snapshot workflows. Plugin assets appear in asset listings for informational purposes and are deployable only via the `nd export` workflow (FR-030). Profiles and snapshots must not reference plugin assets.

### Should have

- **[FR-017]** The TUI displays a dashboard with a tabbed interface. Tabs include an overview tab and one tab per asset type.
- **[FR-018]** The dashboard always displays the current profile name, current scope (global or project), current coding agent, and deployment status (count of deployed assets and count of issues).
- **[FR-019]** The dashboard provides inline actions: deploy asset, check and fix issues, sync assets, remove assets, save current state, and switch profiles.
- **[FR-020]** The user can save the currently deployed asset set as a named snapshot. A snapshot records which assets are deployed, their source paths, and the scope.
- **[FR-021]** The user can restore a previously saved snapshot, deploying all assets recorded in the snapshot.
- **[FR-023]** The user can switch between profiles. Switching removes assets that belong to the current profile (but not pinned or manually deployed assets) and deploys the target profile's assets. The tool warns if any target profile assets conflict with existing pinned or manually deployed assets. Before executing, the tool displays a summary of assets to be removed and assets to be deployed, and requires confirmation. The `--yes` flag bypasses the confirmation for scripting. Switching removes only assets with deployment origin `profile:<current_profile>`. Assets with origin `pinned` or `manual` are never removed by a profile switch.
- **[FR-024a]** The user can pin individual assets so they persist across profile switches. Pinned assets are tracked separately in the deployment state and are never removed by a profile switch operation. When a user runs `nd remove` on a pinned asset, the tool warns that the asset is pinned and requires explicit confirmation before removing it.
- **[FR-025]** The tool provides a settings initialization workflow (interactive walkthrough with opinionated defaults) on first run or via `nd init`. The init workflow uses these defaults: scope = global, symlink strategy = absolute, agent = Claude Code (auto-detected). If Claude Code is not detected, the tool displays installation guidance and allows manual path configuration. If no asset sources are provided, the tool suggests example community sources and allows skipping source registration.
- **[FR-026]** The tool opens the configuration file in the user's default editor via `nd settings edit`.
- **[FR-027]** The tool syncs a Git-sourced repository by performing a `git pull` on the cloned repo when the user requests a source sync.
- **[FR-028]** The TUI main menu presents options: dashboard, each asset type, settings (init and edit), and quit. A "back" action (triggered by the Escape or Backspace key) is available on all screens except the main menu.
- **[FR-029]** When the user launches the TUI, the tool prompts for scope selection (global or project) and then lists detected coding agents for selection.
- **[FR-029a]** Before any bulk deploy, bulk remove, profile switch, or snapshot restore operation, the tool automatically saves the current deployment state as an auto-snapshot. If the operation fails midway, the tool reports the partial state and offers to restore the auto-snapshot. Auto-snapshots are retained for the last 5 auto-snapshots on disk and are distinct from user-created snapshots. Individual single-asset operations (deploy one, remove one) do not trigger auto-snapshots.
- **[FR-016c]** Context folders may contain an optional `_meta.yaml` file with metadata (description, tags, target language, target project, target agent). The CLI and TUI display this metadata when listing context assets to help users decide which context to deploy.
- **[FR-009a]** The tool supports configurable symlink strategies. The user can configure the default symlink strategy to `relative` in `config.yaml`. A `--relative` / `--absolute` CLI flag overrides the configured default for any deploy operation. Absolute is the only strategy in Must-Have scope; relative symlink support is added when this FR is delivered.
- **[FR-036]** The tool provides a `--dry-run` flag for deploy, remove, and sync operations that shows what would happen without making changes.
- **[FR-033]** The user can configure custom deployment locations for coding agents (overriding the default config directory paths) via the config file. This mitigates the risk of assumption A1 (Claude Code's configuration directory structure remaining stable) being wrong.
- **[FR-044]** The tool provides `nd version` and `nd --version` that display the version number, build date, and Go version.
- **[FR-045]** The tool provides `nd doctor` that performs a comprehensive health check: validates config files, verifies all sources are accessible, checks all deployments for broken symlinks, verifies target agent directories exist and are writable, and checks Git availability for Git-sourced repos. Reports all issues with suggested fixes.
- **[FR-046]** The user can list all registered sources via `nd source list`, showing source ID, type (local/Git), path or URL, asset count, and accessibility status.

### Could have

- **[FR-030]** The user can export selected assets as a Claude Code plugin by generating the standard plugin directory structure: `.claude-plugin/plugin.json`, plus `commands/`, `agents/`, `skills/`, `hooks/`, and `README.md` as applicable.
- **[FR-031]** The user can generate a `marketplace.json` catalog file from one or more exported plugins, suitable for distribution as a Claude Code plugin marketplace.
- **[FR-032]** The plugin export workflow is interactive, guiding the user through asset selection, metadata entry (plugin name, description, version, author), and output location.
- **[FR-034]** The user can register additional context file types beyond the built-in set (`CLAUDE.md`, `AGENTS.md`, `AGENTS.local.md`, `CLAUDE.local.md`) via the config file. The named folder structure and `_meta.yaml` support apply to custom context file types as well.
- **[FR-035]** The tool provides shell completions for Bash, Zsh, and Fish.
- **[FR-036a]** The tool provides an `nd uninstall` command that lists all nd-managed symlinks across all known scopes, removes them, and optionally deletes nd's own directories (`~/.config/nd/`, `~/.cache/nd/`). The command requires explicit confirmation and supports `--dry-run`.
- **[FR-036b]** The tool maintains an operation log at `~/.config/nd/logs/operations.log` recording timestamp, operation type, assets affected, and result for each operation.

### Won't have (this time)

- **[FR-037]** Support for coding agents other than Claude Code (Codex, Gemini, OpenCode). Deferred because: v1 focuses on Claude Code to deliver a polished experience for one agent before generalizing. Reconsider when: v1 is stable and user demand for other agents is validated.
- **[FR-038]** File copy as a deployment strategy (alternative to symlinks). Deferred because: symlinks cover macOS use cases; copy introduces sync complexity. Reconsider when: Windows or Linux support is prioritized or users report symlink limitations.
- **[FR-039]** Template rendering for deployed assets (variable substitution in asset files at deploy time). Deferred because: adds complexity to the deployment model and blurs the line with content editing. Reconsider when: users demonstrate a concrete need for per-project asset customization.
- **[FR-040]** Profile inheritance (profiles that extend other profiles). Deferred because: adds complexity to profile resolution with minimal immediate benefit. Reconsider when: users report managing many similar profiles that differ by a few assets.
- **[FR-041]** Asset content editing or creation within nd. Deferred because: content authoring belongs in editors and AI agents, not in a deployment tool. Reconsider when: never (this is a design principle, not a deferral).
- **[FR-042]** Memory file management (`MEMORY.md`, agent-specific memory files). Deferred because: memory files are created and maintained by Claude Code or agents at runtime, and managing them introduces ownership and staleness concerns. Reconsider when: v1 is stable and users demonstrate a need for memory backup and redeployment workflows.

**MoSCoW distribution:** Must: 22 (added FR-016d, FR-043, FR-022, FR-024; removed FR-009a, FR-016c), Should: 20 (added FR-016c, FR-009a, FR-036, FR-033, FR-044, FR-045, FR-046; removed FR-022, FR-024), Could: 7 (removed FR-033, FR-036), Won't: 6. Must-Have ratio: 22/55 = 40%. Within the 60% ceiling.

## Non-functional requirements

This section defines quality attributes for nd. These are measurable constraints on how the system operates, not what it does.

- **[NFR-001]** Startup performance: The TUI must render the initial menu within 500ms of invocation on a machine with a local asset source of 500+ assets.
- **[NFR-002]** Deploy performance: A single asset deploy (symlink creation) must complete within 100ms, excluding filesystem latency for the source check.
- **[NFR-003]** Bulk deploy performance: Deploying 50 assets must complete within 5 seconds.
- **[NFR-004]** Error reporting: Every operation that fails must produce a human-readable error message that names the specific asset, path, and failure reason. No silent failures.
- **[NFR-005]** Configuration validation: The tool must validate config files on load and report all validation errors with line numbers before proceeding.
- **[NFR-006]** Graceful degradation: If a registered source is unavailable (directory missing, repo not cloned), the tool must warn and continue operating with available sources rather than crashing.
- **[NFR-007]** Maintainability: The codebase must follow standard Go project layout conventions, use interfaces for agent-specific logic (to support future agents), and include GoDoc comments on all exported types and functions.
- **[NFR-008]** Binary distribution: The tool must compile to a single static binary with no runtime dependencies beyond the OS.
- **[NFR-009]** Test coverage: Core packages (source discovery, symlink management, profile/snapshot operations) must have unit test coverage above 80%.
- **[NFR-010]** Atomic writes: All state file writes (`deployments.yaml`, profile files, snapshot files) must use atomic file operations: write to a temporary file in the same directory, call `fsync`, then rename to the target path. This prevents data loss from crashes or power failures mid-write.
- **[NFR-011]** Concurrent access: The tool must use advisory file locking (e.g., `flock` or lockfile with `O_CREATE|O_EXCL`) on `deployments.yaml` during read-modify-write cycles. If a lock cannot be acquired within 5 seconds, the tool reports that another nd process is running and exits. Stale locks older than 60 seconds are automatically broken.
- **[NFR-012]** Path validation: All asset paths resolved from sources (including paths specified in `nd-source.yaml` manifests) must be validated to reside within the source's root directory. Paths containing `..` components that resolve outside the source root, or symlinks within sources that point outside the source root, must be rejected with a warning identifying the offending path.
- **[NFR-013]** Safe YAML loading: All YAML parsing must use safe loading (no arbitrary type deserialization). Path lists in `nd-source.yaml` are limited to 1,000 entries. Manifest file sizes are bounded at 1 MB. These limits protect against malicious community source manifests.
- **[NFR-014]** Schema versioning: All YAML files managed by nd (`config.yaml`, `deployments.yaml`, profile files, snapshot files, `nd-source.yaml`) must include a `version` integer field. When nd detects a file with an older schema version, it migrates the file automatically (with a backup of the original). When nd detects a file with a newer schema version (downgrade scenario), it refuses to load and explains the version mismatch.
- **[NFR-015]** Debug logging: The tool supports a `--verbose` global flag that enables detailed logging to stderr, including config resolution steps, source scanning results, symlink operations attempted, and full error details. A `--quiet` flag suppresses all output except errors.
- **[NFR-016]** Exit codes: The tool uses consistent exit codes: 0 for success, 1 for general error, 2 for partial failure (some operations succeeded, some failed), 3 for invalid usage or arguments.
- **[NFR-017]** Source scanning exclusions: Source scanning must exclude `.git/`, `.git` (submodule files), `node_modules/`, and other well-known non-asset directories. This exclusion list is hardcoded and not user-configurable.

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

- Acceptance criteria: I can create a named profile from a curated selection of assets. I can switch profiles with one command. Switching removes the previous profile's assets and deploys the new profile's assets, but pinned assets and manually deployed assets are left in place. The dashboard reflects the active profile and distinguishes pinned from profile-managed assets. Running `nd status` shows the active profile and distinguishes pinned, profile-managed, and manually deployed assets.
- Related requirements: FR-022, FR-023, FR-024, FR-024a, FR-018.

**US-004: Share assets with other developers (Could Have).**
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

**US-007: Switch project context files.**
As a developer who maintains different CLAUDE.md files for different project types (Go CLI, web frontend, data pipeline), I want to browse my context file library with descriptions and switch the active CLAUDE.md for a project, so that each project gets appropriate agent instructions.

- Acceptance criteria: I can list available context files with their descriptions (from `_meta.yaml`). I can deploy a context file to the project root, with nd backing up any existing file. I can see which context file is currently active. Deploying a second context file to the same target offers to replace the current one.
- Related requirements: FR-016b, FR-016c, FR-009.

**US-008: Manage multiple asset sources.**
As a developer with both a personal asset library and team-shared libraries, I want to register, list, and remove asset sources, and control which source takes priority when both contain an asset with the same name.

- Acceptance criteria: I can register local directories and Git repositories as sources. I can list all registered sources with their asset counts and status. I can remove a source and choose what happens to its deployed assets. When two sources have the same asset name, the first registered source wins and I see a warning.
- Related requirements: FR-005, FR-006, FR-016a, and the new source removal FR.

## Technical design

This section captures high-level architecture decisions and component structure. It describes what communicates with what, not implementation details.

### Component overview

nd has five major components:

1. **Source manager.** Handles registration, discovery, and syncing of asset sources. Scans local directories and cloned repos for assets using convention-based directory layout. Reads optional `nd-source.yaml` manifests. Maintains an index of all known assets across all registered sources. When duplicate asset identities are found across sources, resolves by source registration order and emits warnings.

2. **Deploy engine.** Creates and manages symlinks (absolute by default, configurable to relative) between source assets and agent configuration directories. Handles single, bulk, snapshot, and profile deploys. Performs health checks (broken symlinks, drift detection) and repairs.

3. **Agent registry.** Detects installed coding agents by checking known config directory paths and PATH presence. In v1, contains only a Claude Code agent definition. Defines an interface that future agents implement, specifying their config directory structure and asset location conventions.

4. **Profile and snapshot store.** Persists named profiles and point-in-time snapshots as YAML files within nd's data directory (`~/.config/nd/profiles/` and `~/.config/nd/snapshots/`). Profiles define which assets to deploy. Snapshots record the exact deployed state at a moment in time. Auto-snapshots are created before destructive bulk operations (bulk deploy, bulk remove, profile switch, snapshot restore) and the last 5 auto-snapshots are retained on disk.

5. **UI layer.** Two interfaces sharing the same underlying commands:
   - CLI layer (Cobra): Subcommands and flags for every operation. Designed for scripting and single-command workflows.
   - TUI layer (Bubble Tea): Interactive dashboard with tabbed navigation, scope/agent selection, and inline actions. Modeled after the GitHub CLI's interactive experience.

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

| Choice | Rationale |
| ----------------------- | ------------------------------------------------------------ |
| Go 1.23+ | Strong CLI ecosystem, single binary distribution, fast startup, good cross-platform support for future expansion. |
| Cobra | Standard Go CLI framework. Subcommand structure, flag parsing, help generation, shell completions. |
| Bubble Tea | Go TUI framework with component model. Active community, good documentation, GitHub CLI uses it. |
| Lip Gloss | Styling companion to Bubble Tea. Consistent terminal styling without ANSI escape code management. |
| YAML (gopkg.in/yaml.v3) | Human-readable config format. Single format reduces cognitive load. |
| Symlinks (os.Symlink) | Zero-copy deployment. Source changes are immediately reflected. No sync lag. Absolute by default, configurable to relative. |

### Asset deployment mapping (Claude Code v1)

Each asset type maps to a specific target location within Claude Code's configuration. The deploy engine must use this mapping table to determine where symlinks are created. The "link path" column shows where `os.Symlink(target, link)` places the link; the "target" is always the asset in the source directory.

**Global scope deployment (`~/.claude/`):**

| Source type | Source structure | Link created at | Notes |
| ------------- | ---------------------------------------- | --------------------------------------- | ------------------------------------------------------------ |
| skills | `skills/skill-name/` (directory) | `~/.claude/skills/skill-name` | Directory symlink. |
| agents | `agents/agent-name.md` (file) | `~/.claude/agents/agent-name.md` | File symlink. |
| commands | `commands/cmd-name.md` (file) | `~/.claude/commands/cmd-name.md` | File symlink. |
| output-styles | `output-styles/style-name.md` (file) | `~/.claude/output-styles/style-name.md` | File symlink. nd creates the `output-styles/` directory if it does not exist. The user must register the output-style in `settings.json` or `settings.local.json` manually. |
| rules | `rules/rule-name.md` (file) | `~/.claude/rules/rule-name.md` | File symlink. Rules may also be directories with nested `.md` files. |
| context | `context/ctx-name/CLAUDE.md` | `~/.claude/CLAUDE.md` | Symlink targets the context file inside the named folder (e.g., `context/go-project-rules/CLAUDE.md`), not the folder itself. The deploy target is determined by the filename inside the folder. Only one context file can be deployed per target location; deploying a second offers to back up and replace the existing one. Each context folder may contain an optional `_meta.yaml` with metadata (description, tags, target language, target project, target agent) displayed by the CLI to help users choose which context to deploy. |
| context | `context/ctx-name/AGENTS.md` | `~/.claude/AGENTS.md` | Same rules as CLAUDE.md context row. Symlink targets the file inside the named folder. Only one per target location. |
| plugins | `plugins/plugin-name/` (directory) | Managed by Claude Code's plugin system | nd does not symlink plugins directly. Plugin deploy uses `nd export` to produce plugin format, then the user installs via Claude Code's `/plugin install`. |
| hooks | `hooks/hook-name/` (directory) | `~/.claude/hooks/hook-name` | Directory symlink. Each hook folder contains a `hooks.json` config, an executable script (any supported language), and an optional `README.md`. nd creates the `hooks/` directory if it does not exist. The user must also register the hook in `settings.json` or `settings.local.json` manually. Hooks can alternatively be deployed via plugin export (FR-030). |

**Project scope deployment (`.claude/` in project root):**

| Source type | Source structure | Link created at | Notes |
| ------------------------- | ------------------------------------ | -------------------------------------- | ------------------------------------------------------------ |
| skills | `skills/skill-name/` (directory) | `.claude/skills/skill-name` | Directory symlink. |
| agents | `agents/agent-name.md` (file) | `.claude/agents/agent-name.md` | File symlink. |
| commands | `commands/cmd-name.md` (file) | `.claude/commands/cmd-name.md` | File symlink. |
| output-styles | `output-styles/style-name.md` (file) | `.claude/output-styles/style-name.md` | File symlink. The user must register the output-style in `settings.json` or `settings.local.json` manually. |
| rules | `rules/rule-name.md` (file) | `.claude/rules/rule-name.md` | File symlink. |
| context (CLAUDE.md) | `context/ctx-name/CLAUDE.md` | `./CLAUDE.md` (project root) | Symlink targets the file inside the named folder. Placed at the project root, outside `.claude/`. Only one per target location. |
| context (CLAUDE.local.md) | `context/ctx-name/CLAUDE.local.md` | `./CLAUDE.local.md` (project root) | Placed at the project root, outside `.claude/`. Only one per target location. |
| context (AGENTS.md) | `context/ctx-name/AGENTS.md` | `./AGENTS.md` (project root) | Placed at the project root, outside `.claude/`. Only one per target location. |
| context (AGENTS.local.md) | `context/ctx-name/AGENTS.local.md` | `./AGENTS.local.md` (project root) | Placed at the project root, outside `.claude/`. Only one per target location. |
| plugins | `plugins/plugin-name/` (directory) | Managed by Claude Code's plugin system | Same as global: use `nd export` then `/plugin install`. |
| hooks | `hooks/hook-name/` (directory) | `.claude/hooks/hook-name` | Directory symlink. Same structure as global (script, `hooks.json`, optional `README.md`). The user must register the hook in `settings.json`, `settings.local.json`, or `.claude/settings.local.json` manually. Can also deploy via plugin export. |

`.local.md` context file variants (`CLAUDE.local.md`, `AGENTS.local.md`) deploy only at project scope. They are not supported at global scope.

Context files are special cases in two ways. First, unlike other asset types that deploy into subdirectories of the agent's config directory, context files deploy to specific fixed paths determined by the filename inside the named folder (not the folder name). `CLAUDE.md` and `AGENTS.md` deploy directly into `~/.claude/` at global scope, or at the project root (not inside `.claude/`) at project scope. `CLAUDE.local.md` and `AGENTS.local.md` always deploy at the project root. Second, context files are exclusive per deployment location: only one context file can occupy a given target path at a time. If a context file already exists at the target, nd warns the user and offers to back up the existing file before replacing it. The named folder structure (e.g., `context/go-project-rules/CLAUDE.md`) allows users to maintain multiple context files for different purposes and switch between them. Each folder may contain an optional `_meta.yaml` file with metadata such as description, tags, target language, target project, and target agent. The CLI displays this metadata to help users choose which context to deploy.

Assets are either files or directories. Skills and plugins are always directories (a skill is `skill-name/SKILL.md` plus optional subdirectories; a plugin is `plugin-name/` with a `.claude-plugin/` subdirectory). Commands, agents, output-styles, rules are typically single files. Context files are stored as directories (named folder containing the context file and optional `_meta.yaml`), but the symlink targets the file inside, not the directory. Hooks are directories containing a `hooks.json` configuration file, an executable script in any supported language, and an optional `README.md`. The deploy engine must handle all variants: `os.Symlink(sourcePath, linkPath)` works for both files and directories, but health checks, display logic, and context file exclusivity checks must account for the differences.

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

### CLI command reference

The complete command tree, derived from the functional requirements. Every operation available in the TUI is also available as a CLI command (FR-002).

**Command tree:**

| Command | Description | FR |
| --- | --- | --- |
| `nd` | No args: show help (TUI deferred) | FR-001 |
| `nd init` | First-run setup wizard | FR-025 |
| `nd deploy <asset>` | Deploy one or more assets | FR-009, FR-010 |
| `nd remove <asset>` | Remove deployed asset(s) | FR-012 |
| `nd status` | View deployed assets, health, and active profile | FR-015 |
| `nd sync` | Sync sources and fix broken symlinks | FR-013, FR-014, FR-027 |
| `nd source add <path\|url>` | Register a local or Git source | FR-005, FR-006 |
| `nd source remove <id>` | Unregister a source | -- |
| `nd source list` | List registered sources | -- |
| `nd profile create <name>` | Create a named profile | FR-022 |
| `nd profile switch <name>` | Switch to a different profile | FR-023 |
| `nd profile list` | List all profiles | -- |
| `nd profile deploy <name>` | Deploy all assets in a profile | FR-024 |
| `nd snapshot save <name>` | Save current state as a named snapshot | FR-020 |
| `nd snapshot restore <name>` | Restore a previously saved snapshot | FR-021 |
| `nd snapshot list` | List all snapshots | -- |
| `nd settings edit` | Open config in user's default editor | FR-026 |
| `nd export` | Export assets as a Claude Code plugin | FR-030 |
| `nd uninstall` | Remove all nd-managed symlinks and data | FR-036a |
| `nd version` | Show version, commit, and build info | -- |
| `nd doctor` | Holistic health check (sources, symlinks, config) | -- |

**Global flags:**

| Flag | Description | FR |
| --- | --- | --- |
| `--scope global\|project` | Target deployment scope | FR-011 |
| `--dry-run` | Show what would happen without making changes | FR-036 |
| `--verbose` | Increase output detail | -- |
| `--quiet` | Suppress non-error output | -- |
| `--json` | Machine-readable JSON output | -- |
| `--no-color` | Disable colored output | -- |
| `--config <path>` | Override config file location | -- |
| `--relative` / `--absolute` | Override symlink strategy for this invocation | FR-009a |

**Exit codes:**

| Code | Meaning |
| --- | --- |
| 0 | Success |
| 1 | General error |
| 2 | Partial failure (some operations succeeded, some failed) |
| 3 | Invalid usage or arguments |

### Deployment state schema

The file `~/.config/nd/state/deployments.yaml` tracks every symlink nd has created. The deploy engine reads it on startup and writes it after every deploy, remove, or sync operation.

**Schema fields:**

| Field | Type | Description |
| --- | --- | --- |
| `version` | integer | Schema version (currently `1`) |
| `deployments` | list | List of deployment entries |

Each entry in `deployments` contains:

| Field | Type | Description |
| --- | --- | --- |
| `source_id` | string | Identifier of the registered source |
| `asset_type` | string | One of: `skills`, `agents`, `commands`, `output-styles`, `rules`, `context`, `plugins`, `hooks` |
| `asset_name` | string | Name of the asset (file/directory name, without extension for files) |
| `source_path` | string | Absolute path to the source asset |
| `link_path` | string | Absolute path to the created symlink |
| `scope` | string | `global` or `project` |
| `project_path` | string | Absolute path to the project root (present only when `scope` is `project`) |
| `origin` | string | One of: `profile:<name>`, `pinned`, `manual` |
| `deployed_at` | string | ISO 8601 timestamp of when the asset was deployed |

**Annotated example:**

```yaml
version: 1
deployments:
  # Global, manually deployed skill
  - source_id: my-assets
    asset_type: skills
    asset_name: code-review
    source_path: /Users/dev/assets/skills/code-review
    link_path: /Users/dev/.claude/skills/code-review
    scope: global
    origin: manual
    deployed_at: "2026-03-10T14:30:00Z"

  # Project-scoped agent deployed via profile
  - source_id: team-assets
    asset_type: agents
    asset_name: go-specialist
    source_path: /Users/dev/assets/agents/go-specialist.md
    link_path: /Users/dev/projects/myapp/.claude/agents/go-specialist.md
    scope: project
    project_path: /Users/dev/projects/myapp
    origin: "profile:go-backend"
    deployed_at: "2026-03-11T09:15:00Z"

  # Global, pinned command that survives profile switches
  - source_id: my-assets
    asset_type: commands
    asset_name: deploy
    source_path: /Users/dev/assets/commands/deploy.md
    link_path: /Users/dev/.claude/commands/deploy.md
    scope: global
    origin: pinned
    deployed_at: "2026-03-09T08:00:00Z"
```

**Write safety:** All writes to `deployments.yaml` use atomic file operations: the new content is written to a temporary file in the same directory, then renamed over the target. This prevents corruption from interrupted writes or crashes.

**Concurrency:** Concurrent access (e.g., two terminal sessions running nd simultaneously) is guarded by advisory file locking on `deployments.yaml`. A process acquires the lock before reading, holds it through the write, and releases it after the rename completes. If the lock cannot be acquired within 5 seconds, the operation fails with a clear error message.

### Profile scope semantics

Profiles are **scope-aware**: each asset entry in a profile specifies its target scope (`global` or `project`). This means a single profile can contain both global and project-scoped assets.

**Scope behavior during profile operations:**

- Profile switches operate within the **current scope context**. Switching profiles while inside a project directory affects both: (1) project-scoped assets from that profile deploy to the current project's `.claude/` directory, and (2) global-scoped assets in the profile are always deployed to `~/.claude/` regardless of the current directory.
- When a profile containing project-scoped assets is activated from a different project directory than the one where the previous profile was active, the project-scoped assets deploy to the **current** project's `.claude/` directory. Assets deployed to a previous project's `.claude/` are not automatically removed; use `nd sync` to detect and clean up orphaned project-scoped deployments.

**Deployment origin categories:**

| Origin | Meaning | Affected by profile switch |
| --- | --- | --- |
| `profile:<name>` | Deployed as part of the named profile | Yes -- removed when switching away from `<name>` |
| `pinned` | Explicitly pinned by the user (FR-024a) | No -- persists across all profile switches |
| `manual` | Deployed via direct `nd deploy` outside any profile | No -- persists across all profile switches |

**Profile switch algorithm:**

1. **Compute diff** between the current profile's asset list and the target profile's asset list.
2. **Assets in both profiles** (same `source_id`, `asset_type`, `asset_name`, and `scope`) are left in place. Their `origin` is updated to `profile:<target>`.
3. **Remove assets unique to the current profile.** Only assets with origin `profile:<current_profile_name>` are removed. Pinned and manual assets are never touched.
4. **Deploy assets unique to the target profile.** New symlinks are created. Conflicts with pinned or manual assets trigger a warning (FR-023).
5. **Update deployment state.** All affected entries in `deployments.yaml` are updated with the new origin and timestamp.

An auto-snapshot is saved before step 3 begins (FR-029a).

### Error behavior

This subsection defines how nd handles error scenarios. The guiding principle is: never crash silently, always provide actionable guidance, and prefer continuing over aborting when partial progress is useful.

**Config validation failure:**

- Invalid global config (`~/.config/nd/config.yaml`): nd exits with code 1 and prints all validation errors with line numbers. No operations proceed.
- Invalid project config (`.nd/config.yaml`): nd prints a warning with the validation errors and line numbers, then falls back to global config only. Operations proceed.

**Partial failure in bulk operations:**

- nd uses **fail-open behavior**: when one asset in a bulk deploy/remove fails, nd continues processing the remaining assets.
- After completion, nd outputs a summary: `Deployed 47/50 assets. 3 failed:` followed by per-asset error details (asset identity, path, and failure reason).
- In interactive mode (TUI), nd offers a "Retry failed" option. In CLI mode, nd exits with code 2 (partial failure).

**Permission errors:**

- nd catches `EACCES`/`EPERM` errors and wraps them with an actionable message: `Permission denied: cannot write to <path>. Run ls -la <parent> to check ownership and permissions.`

**Git clone/sync failures:**

- Authentication failures produce nd-specific guidance: `Authentication failed for <url>. Check your SSH keys (ssh-add -l) or access token configuration.`
- Network timeouts produce: `Network timeout while connecting to <host>. Check your internet connection and try again.`
- Merge conflicts during `git pull` in a source sync: `Merge conflict in source <source_id>. Resolve manually in <source_path>, then run nd sync again.`

**Missing Claude Code:**

- If no coding agents are detected (no `~/.claude/` directory, `claude` not in PATH), nd displays: `No coding agents detected. Install Claude Code or configure a custom agent path in config.yaml (see: nd settings edit).`
- nd allows proceeding with manual path configuration (FR-033) rather than blocking all operations.

**Missing `.claude/` directory:**

- When deploying to project scope and the project's `.claude/` directory does not exist, nd creates it automatically. The user has already explicitly chosen project scope, so no confirmation is needed.
- Subdirectories (`skills/`, `agents/`, etc.) are created as needed during deployment.

**`deployments.yaml` corruption:**

- If the state file fails YAML parsing, nd renames the corrupted file to `deployments.yaml.corrupt.<timestamp>` (e.g., `deployments.yaml.corrupt.2026-03-14T10-30-00`), starts with an empty deployment state, and prints: `Warning: deployments.yaml was corrupted and has been renamed to deployments.yaml.corrupt.<timestamp>. Run nd sync to rebuild deployment state from the filesystem.`

## Boundaries

This section defines behavior tiers for AI agents implementing this spec. These are constraints on how an implementing agent should operate.

### Always

- Always validate that a source directory exists and is readable before attempting asset discovery.
- Always verify that the target agent config directory exists **and is writable** before creating symlinks.
- Always check for existing files or symlinks at the target path before deploying, and report conflicts rather than silently overwriting.
- Always write deployment state changes to `~/.config/nd/state/deployments.yaml` after any deploy, remove, or sync operation.
- Always use the configuration hierarchy (defaults, global config, project config, CLI flags) when resolving settings.
- Always run tests for core packages (source discovery, symlink management, profile/snapshot operations) before committing changes.
- Always use atomic file writes (write-to-temp-then-rename) for all state and data files.
- Always validate that asset paths from source manifests resolve within the source root directory (no path traversal).
- Always exclude `.git/` and `node_modules/` directories during source scanning.

### Ask-first

- Ask before removing deployed assets that are not managed by nd (symlinks or files that nd did not create).
- Ask before overwriting an existing profile or snapshot with the same name.
- Ask before performing a source sync (`git pull`) that is triggered autonomously (e.g., as part of a bulk deploy that detects a stale source). User-initiated sync commands (e.g., `nd sync`) do not require confirmation.
- Ask before creating the `.nd/` directory in a project (project-level config initialization).
- Ask before modifying any file outside of nd's own directories (`~/.config/nd/`) and outside of recognized agent configuration directories (`~/.claude/`, `.claude/`, project root for context files). Symlink creation in agent config directories and recognized context file locations does not require confirmation, as the user has already selected a deploy target.

### Never

- Never modify the content of source asset files. nd reads and symlinks assets but never writes to them.
- Never execute Git commands beyond clone and pull (no commit, push, branch, merge, rebase).
- Never delete source asset files. Removal operations only delete symlinks in the target agent config directory.
- Never store secrets, API keys, or credentials in nd's configuration files.
- Never make network requests other than Git clone and pull operations for registered Git sources.
- Never deserialize arbitrary YAML types from untrusted sources. Use safe YAML loading for all community-sourced manifests.
- Never create symlinks pointing outside of recognized agent configuration directories or context file target locations.

## Success criteria

**Core success criteria** (verifiable with Must-Have requirements):

1. A user with 500+ assets in a local source directory can deploy, remove, and sync assets without errors and without manual symlink management. Verified by: end-to-end test with a 500-asset source directory.
2. A user can complete first-time setup (`nd init`) and deploy their first asset within 5 minutes of installing the tool. Verified by: timed walkthrough with a new user unfamiliar with the tool. (Depends on FR-025; if FR-025 is deferred, verified via manual config creation.)
3. The tool accurately reflects the current deployment state (deployed assets, issues, active profile) in both CLI output and TUI dashboard. Verified by: deploy, remove, and sync operations from both CLI and TUI, then checking consistency. (TUI verification depends on FR-017/FR-018.)
4. A user can create a profile and deploy it with a single command. Verified by: create a profile, deploy it, verify symlinks match expected state. (Profile switching and snapshot verification depend on Should-Have FRs.)
5. The tool handles edge cases without crashing: missing source directories, broken symlinks, empty sources, duplicate asset names across sources, and read-only target directories. Verified by: automated tests for each edge case.
6. The tool passes `golangci-lint` with a strict configuration and has >80% test coverage on core packages. Verified by: CI pipeline checks.
7. The tool ships with a `README.md` covering installation and first-time setup, and complete `nd help` output for all commands, before the first public release. Verified by: review of documentation against the command tree.
8. The tool has no open blocking bugs at the time of first public release. Verified by: issue tracker review.

**Extended success criteria** (require Should-Have or Could-Have requirements):

9. A user can switch profiles and the resulting deployment state matches the target profile exactly (no leftover assets from the previous profile, no missing assets from the new profile), while pinned and manually deployed assets remain untouched. Verified by: profile switch test comparing expected vs. actual symlink state. (Requires FR-023.)
10. A user can export a set of assets as a valid Claude Code plugin that installs successfully via Claude Code's `/plugin install` command. Verified by: export a plugin from nd and install it in a fresh Claude Code environment. (Requires FR-030, FR-031, FR-032.)
11. The TUI dashboard accurately reflects the current deployment state at all times, with tabbed navigation and inline actions. Verified by: TUI testing after deploy, remove, and sync operations. (Requires FR-017, FR-018, FR-019.)

## Open questions

Question IDs are stable and not reused across revisions.

### Open questions

| # | Question | Category | Impact |
| ---- | ------------------------------------------------------------ | -------------------- | ------------------------------------------------------------ |
| Q1 | What is the full format and schema for `nd-source.yaml` manifests beyond the `paths` and `exclude` fields? Should it support metadata like asset descriptions, tags, and categories? | Non-blocking | A minimal skeleton is defined in the Technical design. The full schema can be iterated on after v1. |
| Q3 | What specific fields should `plugin.json` contain when exporting assets as Claude Code plugins? | Non-blocking | Plugin export is a "could have" feature. Schema can be defined when that feature is implemented. |
| Q6 | What is the correct detection method for each coding agent? Claude Code uses `~/.claude/`, but what about Codex, Gemini, and OpenCode? | Non-blocking | v1 only supports Claude Code. Detection for other agents can be researched when multi-agent support is added. |
| Q7 | Should the deployment state file (`deployments.yaml`) be human-editable, or is it purely machine-managed? | Non-blocking | Affects schema complexity and validation strictness. |

### Resolved questions

| # | Question | Resolution |
| ---- | ------------------------------------------------------------ | ------------------------------------------------------------ |
| Q4 | ~~Should profiles store asset references (names and source identifiers) or absolute paths?~~ | Profiles store asset references `(source_id, asset_type, asset_name)` as primary, with the absolute path cached as a hint. Resolution uses the reference first; falls back to the cached path if the source is not registered. |
| Q5 | ~~How should nd handle a source sync (`git pull`) that deletes or renames assets that are currently deployed?~~ | nd removes the orphaned symlink and removes the asset from the deployment state (FR-014). |
| Q8 | ~~How should nd handle asset sources on case-insensitive filesystems (macOS default) where directory names like `Skills/` and `skills/` would collide?~~ | nd normalizes asset names to lowercase for identity comparison purposes. During discovery, case-variant duplicates (e.g., `My-Skill` and `my-skill`) are detected and warned. This is required for correctness on macOS's default case-insensitive filesystem. |
| Q13 | ~~Should nd post-deploy print a reminder when deployed assets require manual `settings.json` changes (e.g., registering an output-style or a hook)?~~ | Yes. nd must always print a post-deploy reminder when deployed assets require manual `settings.json` changes. The reminder includes the settings file path and the specific configuration snippet needed. This applies to hooks and output-styles. |

## Changelog

| Version | Date | Author | Changes |
| ------- | ---------- | ------ | ------------------------------------------------------------ |
| 0.7 | 2026-03-22 | Larah | Status update: all non-TUI FRs implemented (43/55). Marked TUI FRs as deferred pending redesign. Updated command reference (`nd` without args shows help). Bumped status from Draft to Implemented (TUI deferred). |
| 0.6 | 2026-03-14 | Larah | Audit remediation: promoted FR-022 and FR-024 to Must-Have for Goal 3 coverage; promoted FR-036 (--dry-run) and FR-033 (custom deploy locations) to Should-Have; demoted FR-016c and FR-009a to Should-Have; added new FRs for source removal, asset listing, nd version, nd doctor, source list; added CLI command reference, deployment state schema, profile scope semantics, and error behavior subsections to Technical Design; added NFRs for atomic writes, file locking, path validation, safe YAML loading, schema versioning, debug logging, exit codes, and source scanning exclusions; added security items to Boundaries; resolved Q8 (case normalization) and Q13 (settings.json reminders); added AGENTS.md to global deployment table; fixed heading capitalization; added user stories for context file management and source management; restructured success criteria into core and extended tiers; added concept ladder to Goals; updated non-goal to carve out git clone/pull as source management internals. |
| 0.5 | 2026-03-14 | Larah | Profile format: closed Q4 — profiles store asset references `(source_id, asset_type, asset_name)` as primary with absolute path cached as fallback hint. |
| 0.4 | 2026-03-14 | Larah | Orphan removal: updated FR-014 to specify that orphaned symlinks (source asset renamed or deleted) are automatically removed along with their deployment state entry; closed Q5 as resolved. |
| 0.3 | 2026-03-14 | Larah | Remediation revision: consolidated all nd data under `~/.config/nd/` (removed `~/.local/share/nd/` split); restructured context files into named folders with optional `_meta.yaml` metadata, added deploy exclusivity with backup offer (FR-016b, FR-016c); resolved hooks deployment (symlink to `.claude/hooks/` plus manual settings.json registration, or via plugin export); confirmed output-styles directory support and added settings.json registration note; removed SOUL.md references (not supported by Claude Code); deferred memory file management to Won't Have (FR-042) with non-goal entry; added both/configurable symlink strategy with absolute default (FR-009a); added source priority ordering with warnings for duplicate assets (FR-016a); narrowed ask-first sync rule to autonomous syncs only; removed "back" from main menu, added Escape/Backspace navigation (FR-028); moved release-readiness from Goal 7 to Success criteria 8-9; added pinned asset removal warning behavior (FR-024a); clarified nested source layout limitation in FR-007 with explicit callout; reclassified A8 as design decision; updated data flow diagram to include Profile/Snapshot Store and deployments.yaml; refined auto-snapshot triggers to bulk deploy, bulk remove, profile switch, snapshot restore only (FR-029a); added operation logging as Could Have (FR-036b); added README and nd-help success criteria; added nd-source.yaml manifest skeleton to Technical design; labeled US-004 as Could Have; closed Q2 (duplicate assets), Q9 (merged into Q1), Q10 (deployment state file), Q11 (hooks), Q12 (output-styles). |
| 0.2 | 2026-03-14 | Larah | Audit revision: added commands as asset type; added asset deployment mapping table with context file special cases; added asset identity definition; clarified symlink direction in FR-009; added pinned assets (FR-024a) for profile switching safety; added auto-snapshots (FR-029a) for rollback on bulk operations; fixed XDG compliance (sources/state in ~/.config/nd/); broadened Git support beyond GitHub; added uninstall command (FR-036a); added settings.json non-goal with gap explanation; clarified root-only source scanning in FR-007; fixed .mcp.json filename; added assumptions A9 (context file locations) and A10 (symlink resolution); added open questions Q11 (hooks format), Q12 (output-styles directory), Q13 (settings.json reminders). |
| 0.1 | 2026-03-13 | Larah | Initial draft from elicitation. |
