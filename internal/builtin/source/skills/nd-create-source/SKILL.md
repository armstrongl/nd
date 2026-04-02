---
name: nd-create-source
description: Use when creating a new nd source directory from scratch. Handles directory layout, asset type selection, manifest generation, and source registration.
---

# Create an nd source directory

Create a well-structured nd source directory from scratch. An nd source is a directory containing coding agent assets (skills, agents, commands, rules, context files, output-styles, plugins, hooks) that nd discovers and deploys via symlinks.

## When to use

- Starting a new collection of reusable agent assets
- Setting up a shared team source repository
- Converting an existing directory of assets into an nd-compatible source

## Step 1: gather requirements

Ask the user which asset types they need in their source. Present these options:

| Asset type | Directory name | Structure |
|------------|---------------|-----------|
| Skills | `skills/` | Each skill is a subdirectory containing a `SKILL.md` file |
| Agents | `agents/` | Each agent is a standalone `.md` file |
| Commands | `commands/` | Each command is a standalone `.md` file |
| Output styles | `output-styles/` | Each output style is a standalone `.md` file |
| Rules | `rules/` | Each rule is a `.md` file or a directory |
| Context | `context/` | Each context asset is a subdirectory containing a `.md` file (typically `CLAUDE.md`) |
| Plugins | `plugins/` | Each plugin is a subdirectory containing a `.claude-plugin/` directory with `plugin.json` inside |
| Hooks | `hooks/` | Each hook is a subdirectory containing a `hooks.json` file |

Also ask:

- **Source path**: where to create the directory (absolute path or relative to cwd)
- **Source name**: a human-readable name for the source (used in the manifest metadata)
- **Include manifest**: whether to generate an `nd-source.yaml` manifest file (recommended for sources with non-standard layouts or metadata)

## Step 2: create the directory structure

Create the source root directory and one subdirectory for each selected asset type, using the exact directory names from the table above.

Example for a source with skills, agents, and rules:

```
my-source/
├── skills/
├── agents/
└── rules/
```

## Step 3: generate starter assets (optional)

If the user wants example assets to start from, create one placeholder per selected type:

- **Skills**: `skills/example-skill/SKILL.md` with a heading, description placeholder, and a steps section
- **Agents**: `agents/example-agent.md` with a heading and role description placeholder
- **Commands**: `commands/example-command.md` with a heading and usage section
- **Output styles**: `output-styles/example-style.md` with a heading and format description
- **Rules**: `rules/example-rule.md` with a heading and rule content placeholder
- **Context**: `context/example-context/CLAUDE.md` with a heading and instructions placeholder
- **Plugins**: `plugins/example-plugin/.claude-plugin/plugin.json` with `{"name": "example-plugin", "version": "0.1.0", "description": ""}`
- **Hooks**: `hooks/example-hook/hooks.json` with a minimal valid hooks structure

## Step 4: generate the manifest (if requested)

If the user opted for a manifest, create `nd-source.yaml` at the source root. The manifest overrides convention-based discovery with explicit paths.

Format:

```yaml
version: 1
metadata:
  name: "<source-name>"
  description: "<one-line description>"
  author: "<optional author>"
  tags: []
paths:
  skills:
    - skills
  agents:
    - agents
  # include only the asset types the user selected
exclude: []
```

The `paths` map uses nd asset type names as keys (`skills`, `agents`, `commands`, `output-styles`, `rules`, `context`, `plugins`, `hooks`) and lists of directory paths as values. All paths are relative to the source root. The `exclude` list contains asset names to skip during scanning.

If the user is using standard directory names and has no exclusions, inform them that the manifest is optional since nd discovers assets by convention (matching directory names to asset types automatically).

## Step 5: register the source with nd

After creating the directory, run:

```shell
nd source add <path-to-source> --alias <alias>
```

The `--alias` flag provides a human-readable name for the source. The path can be absolute or relative. nd also accepts git URLs and GitHub shorthand (`owner/repo`) for remote sources.

Verify registration succeeded:

```shell
nd source list
```

## Asset structure reference

### Skills (directory-based)

Each skill must be a directory containing `SKILL.md`. Additional files (scripts, templates, data) can live alongside it.

```
skills/
└── my-skill/
    ├── SKILL.md        # required
    ├── helper.sh       # optional supporting files
    └── template.md     # optional supporting files
```

### Agents, commands, output styles (file-based)

Each asset is a single `.md` file placed directly in the corresponding directory.

```
agents/
├── code-reviewer.md
└── test-writer.md
```

### Rules (file or directory)

Rules can be either standalone `.md` files or directories (for rules that need supporting files).

```
rules/
├── code-style.md
└── testing-standards/
    ├── unit-tests.md
    └── integration-tests.md
```

### Context (folder-per-asset)

Each context asset is a directory containing at least one `.md` file. Files prefixed with `_` are ignored.

```
context/
└── project-setup/
    └── CLAUDE.md
```

### Plugins (directory with .claude-plugin)

Each plugin is a directory that must contain a `.claude-plugin/` subdirectory with a `plugin.json` file inside.

```
plugins/
└── my-plugin/
    ├── .claude-plugin/
    │   └── plugin.json
    └── skills/
        └── plugin-skill/
            └── SKILL.md
```

### Hooks (directory with hooks.json)

Each hook is a directory that must contain a `hooks.json` file defining the hook configuration.

```
hooks/
└── pre-commit-lint/
    ├── hooks.json
    └── lint.sh
```

## Rules

- Always use the exact directory names listed in the table: `skills`, `agents`, `commands`, `output-styles`, `rules`, `context`, `plugins`, `hooks`. nd relies on these names for convention-based discovery.
- Do not nest asset directories inside each other. All asset type directories must be direct children of the source root.
- Never create directories named `.git` or `node_modules` inside the source; nd skips these during scanning.
- Do not create symlinks inside the source directory; nd ignores symlinked entries during scanning.
- The `nd-source.yaml` manifest must not exceed 1 MB.
- If the source root path does not exist, create it. If it exists and is not empty, warn the user before adding structure to avoid overwriting existing content.
