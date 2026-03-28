nd is a Go CLI tool for managing coding agent assets (skills, agents, commands, output styles, rules, context files, plugins, hooks) via symlink deployment. It uses Cobra for the CLI framework, gopkg.in/yaml.v3 for configuration, and Bubble Tea for the TUI.

The index below lists all available guide documentation. Read the description column for each row and load only the docs that are relevant to your current task.

## How to use this index

Load a doc if its description matches the concepts, components, or tasks involved in your current work. The description field is written as a trigger condition: "Load when [conditions]." If the conditions match your task, load the doc. If they do not, skip it.

If no doc in the index is relevant to your task, proceed without loading any. The absence of a relevant doc is useful signal — it may mean the area you are working in is undocumented. Note this if it affects your ability to complete the task accurately.

Reference docs for individual CLI commands are in `docs/reference/` and are auto-generated.

<!-- AGENTS-INDEX-START -->

| Doc | When to load | Last validated | Status | Paths |
|---|---|---|---|---|
| [Configuration](docs/guide/configuration.md) | Load when modifying config loading, merging, validation, or debugging config file issues. | 2026-03-28 | current | `internal/config/**`<br>`cmd/settings.go`<br>`cmd/init.go` |
| [Create asset sources](docs/guide/creating-sources.md) | Load when modifying source scanning, asset type discovery, manifest parsing, or the directory convention. | 2026-03-28 | current | `internal/sourcemanager/**`<br>`internal/asset/**` |
| [Get started](docs/guide/getting-started.md) | Load when setting up nd for the first time, troubleshooting installation, or onboarding a new user. | 2026-03-28 | current | `cmd/init.go`<br>`cmd/source.go`<br>`cmd/deploy.go` |
| [How nd works](docs/guide/how-nd-works.md) | Load when modifying symlink creation, deploy logic, scope handling, or debugging broken deployments. | 2026-03-28 | current | `internal/deploy/**`<br>`cmd/deploy.go`<br>`cmd/remove.go` |
| [Profiles and snapshots](docs/guide/profiles-and-snapshots.md) | Load when modifying profile CRUD, snapshot save/restore, profile switching, or pinning logic. | 2026-03-28 | current | `internal/profile/**`<br>`cmd/profile.go`<br>`cmd/snapshot.go`<br>`cmd/pin.go`<br>`cmd/unpin.go` |
| [User guide](docs/guide/user-guide.md) | Load when modifying CLI commands, interactive pickers, JSON output, scripting flags, or sync/doctor workflows. | 2026-03-28 | current | `cmd/**` |

<!-- AGENTS-INDEX-END -->
