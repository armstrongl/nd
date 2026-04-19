nd is a Go CLI tool for managing coding agent assets (skills, agents, commands, output styles, rules, context files, plugins, hooks) via symlink deployment. It uses Cobra for the CLI framework, gopkg.in/yaml.v3 for configuration, and Bubble Tea for the TUI.

The index below lists all available guide documentation. Read the description column for each row and load only the docs that are relevant to your current task.

## How to use this index

Load a doc if its description matches the concepts, components, or tasks involved in your current work. The description field is written as a trigger condition: "Load when [conditions]." If the conditions match your task, load the doc. If they do not, skip it.

If no doc in the index is relevant to your task, proceed without loading any. The absence of a relevant doc is useful signal: it may mean the area you are working in is undocumented. Note this if it affects your ability to complete the task accurately.

Reference docs for individual CLI commands are in `docs/reference/` and are auto-generated.

<!-- AGENTS-INDEX-START -->

| Doc | When to load | Last validated | Status | Paths |
|---|---|---|---|---|
| [Agents](docs/guide/asset-types/agents.md) | Load when modifying agent file scanning, agent deployment, or the agents asset type. | 2026-04-04 | current | `internal/sourcemanager/scanner.go`<br>`internal/deploy/**` |
| [Commands](docs/guide/asset-types/commands.md) | Load when modifying command file scanning, command deployment, or the commands asset type. | 2026-04-04 | current | `internal/sourcemanager/scanner.go`<br>`internal/deploy/**` |
| [Configuration](docs/guide/configuration.md) | Load when modifying config loading, merging, validation, or debugging config file issues. | 2026-04-05 | current | `internal/config/**`<br>`cmd/settings.go`<br>`cmd/init.go` |
| [Context](docs/guide/asset-types/context.md) | Load when modifying context asset scanning, context deployment paths, or context conflict handling. | 2026-04-04 | current | `internal/sourcemanager/scanner.go`<br>`internal/deploy/**`<br>`internal/deploy/context.go` |
| [Create asset sources](docs/guide/creating-sources.md) | Load when modifying source scanning, asset type discovery, manifest parsing, or the directory convention. | 2026-04-05 | current | `internal/sourcemanager/**`<br>`internal/asset/**` |
| [Get started](docs/guide/getting-started.md) | Load when setting up nd for the first time, troubleshooting installation, or onboarding a new user. | 2026-04-05 | current | `cmd/init.go`<br>`cmd/source.go`<br>`cmd/deploy.go` |
| [Glossary](docs/guide/glossary.md) | Load when encountering unfamiliar nd terminology or when disambiguating overloaded terms like agent, context, or command. | 2026-04-03 | current | `internal/nd/**`<br>`internal/asset/**` |
| [Hooks](docs/guide/asset-types/hooks.md) | Load when modifying hook scanning, hook deployment, or settings.JSON hook registration. | 2026-04-04 | current | `internal/sourcemanager/scanner.go`<br>`internal/deploy/**` |
| [How nd works](docs/guide/how-nd-works.md) | Load when modifying symlink creation, deploy logic, scope handling, or debugging broken deployments. | 2026-04-05 | current | `internal/deploy/**`<br>`cmd/deploy.go`<br>`cmd/remove.go` |
| [Output styles](docs/guide/asset-types/output-styles.md) | Load when modifying output style scanning, deployment, or settings.JSON registration behavior. | 2026-04-04 | current | `internal/sourcemanager/scanner.go`<br>`internal/deploy/**` |
| [Plugins](docs/guide/asset-types/plugins.md) | Load when modifying plugin scanning, export workflow, or plugin.JSON manifest handling. | 2026-04-04 | current | `internal/sourcemanager/scanner.go`<br>`internal/export/**`<br>`cmd/export.go` |
| [Profiles and snapshots](docs/guide/profiles-and-snapshots.md) | Load when modifying profile CRUD, snapshot save/restore, profile switching, or pinning logic. | 2026-04-05 | current | `internal/profile/**`<br>`cmd/profile.go`<br>`cmd/snapshot.go`<br>`cmd/pin.go`<br>`cmd/unpin.go` |
| [Rules](docs/guide/asset-types/rules.md) | Load when modifying rule file scanning, rule deployment, or the rules asset type. | 2026-04-04 | current | `internal/sourcemanager/scanner.go`<br>`internal/deploy/**` |
| [Skills](docs/guide/asset-types/skills.md) | Load when modifying skill directory scanning, skill deployment, or the skills asset type. | 2026-04-04 | current | `internal/sourcemanager/scanner.go`<br>`internal/deploy/**` |
| [Troubleshoot](docs/guide/troubleshooting.md) | Load when debugging nd issues: broken symlinks, missing assets, config errors, profile switching problems, or context file conflicts. | 2026-04-04 | current | `cmd/doctor.go`<br>`cmd/sync.go`<br>`internal/deploy/**` |
| [User guide](docs/guide/user-guide.md) | Load when modifying CLI commands, interactive pickers, JSON output, scripting flags, or sync/doctor workflows. | 2026-04-05 | current | `cmd/**` |

<!-- AGENTS-INDEX-END -->

## Documented solutions

`docs/solutions/` and `.claude/docs/solutions/` contain documented solutions to past problems (bugs, best practices, workflow patterns), organized by category with YAML frontmatter (`module`, `tags`, `problem_type`). Relevant when implementing or debugging in documented areas.

## Vexp <!-- vexp v1.2.30 -->

**MANDATORY: use `run_pipeline`: do NOT grep or glob the codebase.**
vexp returns pre-indexed, graph-ranked context in a single call.

### Workflow

1. `run_pipeline` with your task description: ALWAYS FIRST (replaces all other tools)
2. Make targeted changes based on the context returned
3. `run_pipeline` again only if you need more context

### Available MCP tools

- `run_pipeline`: **PRIMARY TOOL**. Runs capsule + impact + memory in 1 call.
  Auto-detects intent. Includes file content. Example: `run_pipeline({ "task": "fix auth bug" })`
- `get_context_capsule`: lightweight, for simple questions only
- `get_impact_graph`: impact analysis of a specific symbol
- `search_logic_flow`: execution paths between functions
- `get_skeleton`: compact file structure
- `index_status`: indexing status
- `get_session_context`: recall observations from sessions
- `search_memory`: cross-session search
- `save_observation`: persist insights (prefer run_pipeline's observation param)

### Agentic search

- Do NOT use built-in file search, grep, or codebase indexing: always call `run_pipeline` first
- If you spawn sub-agents or background tasks, pass them the context from `run_pipeline`
  rather than letting them search the codebase independently

### Smart features

Intent auto-detection, hybrid ranking, session memory, auto-expanding budget.

### Multi-Repo

`run_pipeline` auto-queries all indexed repos. Use `repos: ["alias"]` to scope. Run `index_status` to see aliases.
<!-- /vexp -->
