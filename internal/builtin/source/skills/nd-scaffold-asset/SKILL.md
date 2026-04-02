---
name: nd-scaffold-asset
description: Use when creating a new asset (skill, agent, command, rule, context, output-style, or hook) inside an existing nd source. Handles directory layout, file scaffolding, scanner validation, and deployment instructions.
---

# Scaffold a new asset in an nd source

Create a new coding agent asset inside a registered nd source with the correct structure so nd's scanner can discover and deploy it.

## When to use

- Adding a new skill, agent, command, rule, context file, output-style, or hook to an existing nd source directory.
- Unsure what directory layout or files an asset type requires.
- Want to confirm the asset will pass nd's scanner validation before deploying.

## Prerequisites

- nd is installed and at least one source is registered. Run `nd source list` to confirm.
- You know which source directory you want to add the asset to (its path on disk).

## Workflow

### Step 1: identify the target source

If the user has not specified a source, list registered sources and ask which one to use.

```shell
nd source list
```

Record the source path from the output. All new files will be created under this path.

### Step 2: determine the asset type

Ask the user what type of asset to create. The supported types and their directory names are:

| Type | Directory | Format | Scanner requirement |
|------|-----------|--------|---------------------|
| Skill | `skills/` | Directory containing `SKILL.md` | Directory must contain a file named `SKILL.md` |
| Agent | `agents/` | Single `.md` file | File must end in `.md`, must not be a directory |
| Command | `commands/` | Single `.md` file | File must end in `.md`, must not be a directory |
| Output style | `output-styles/` | Single `.md` file | File must end in `.md`, must not be a directory |
| Rule | `rules/` | Single `.md` file or directory | File must end in `.md`; directories are also valid |
| Context | `context/` | Folder containing a `.md` file | Folder must contain at least one `.md` file (typically `CLAUDE.md`); optional `_meta.yaml` |
| Hook | `hooks/` | Directory containing `hooks.json` | Directory must contain a file named `hooks.json` |

### Step 3: collect name and description

Ask the user for:

1. **Name**: a kebab-case identifier (e.g., `code-review`, `no-emojis`, `go-project-rules`). This becomes the filename or directory name.
2. **Description**: a one-line summary of what the asset does.

### Step 4: create the asset

Use the templates below. Replace `<SOURCE>` with the source path, `<NAME>` with the asset name, and `<DESCRIPTION>` with the description.

#### Skill

Create a directory with a `SKILL.md` file inside.

```text
<SOURCE>/skills/<NAME>/
└── SKILL.md
```

`SKILL.md` content:

```markdown
---
name: <NAME>
description: <DESCRIPTION>
---

# <Title derived from NAME>

<DESCRIPTION>

## When to use

- (describe the situations where this skill applies)

## Workflow

1. (step-by-step instructions for the agent)
```

#### Agent

Create a single `.md` file.

```text
<SOURCE>/agents/<NAME>.md
```

Content:

```markdown
---
name: <NAME>
description: <DESCRIPTION>
---

# <Title derived from NAME>

<DESCRIPTION>

## Role

(describe the agent's role and responsibilities)

## Instructions

(provide the agent's behavioral instructions)
```

#### Command

Create a single `.md` file.

```text
<SOURCE>/commands/<NAME>.md
```

Content:

```markdown
---
name: <NAME>
description: <DESCRIPTION>
---

# /<NAME>

<DESCRIPTION>

## Usage

`/<NAME> [arguments]`

## Behavior

(describe what happens when the command is invoked)
```

#### Output style

Create a single `.md` file.

```text
<SOURCE>/output-styles/<NAME>.md
```

Content:

```markdown
---
name: <NAME>
description: <DESCRIPTION>
---

# <Title derived from NAME>

<DESCRIPTION>

## Formatting rules

(describe the output formatting rules)
```

Note: output styles require manual registration in the agent's `settings.json` after deployment.

#### Rule

Create a single `.md` file (or a directory if the rule spans multiple files).

```text
<SOURCE>/rules/<NAME>.md
```

Content:

```markdown
# <Title derived from NAME>

<DESCRIPTION>

(rule content: instructions the agent must follow)
```

#### Context

Create a folder with a `CLAUDE.md` file and an optional `_meta.yaml`.

```text
<SOURCE>/context/<NAME>/
├── CLAUDE.md
└── _meta.yaml   (optional)
```

`CLAUDE.md` content:

```markdown
(context content: project rules, conventions, or instructions for the agent)
```

`_meta.yaml` content (if the user provides metadata):

```yaml
description: "<DESCRIPTION>"
tags: []
```

The `_meta.yaml` file supports these optional fields: `description`, `tags`, `target_language`, `target_project`, `target_agent`.

#### Hook

Create a directory with a `hooks.json` file.

```text
<SOURCE>/hooks/<NAME>/
└── hooks.json
```

`hooks.json` content:

```json
{
  "hooks": []
}
```

Note: hooks require manual registration in the agent's `settings.json` after deployment.

### Step 5: create parent directories

If the asset type directory does not exist yet in the source (e.g., there is no `skills/` folder), create it. The scanner only looks for directories that exist.

```shell
mkdir -p "<SOURCE>/<TYPE_DIR>/<NAME>"
```

### Step 6: validate the asset

After creating the files, verify the scanner will recognize the asset by running:

```shell
nd list --source <SOURCE_ID>
```

The new asset should appear in the output. If it does not, check:

- **Skill**: the directory contains a file named exactly `SKILL.md` (case-sensitive).
- **Agent/command/output-style**: the file ends in `.md` and is not inside a subdirectory deeper than one grouping level.
- **Context**: the folder contains at least one `.md` file that does not start with `_`.
- **Hook**: the directory contains a file named exactly `hooks.json`.
- **General**: the entry name does not start with `.` (dot-prefixed entries are skipped). The entry is not a symlink (symlinks are skipped during scanning). The entry is not inside `.git` or `node_modules`.

### Step 7: show deployment instructions

Tell the user how to deploy the new asset:

```shell
# Deploy globally (default)
nd deploy <NAME>

# Deploy to a specific project
nd deploy <NAME> --scope project

# Deploy with explicit type qualifier
nd deploy <TYPE>/<NAME>
```

Where `<TYPE>` is the asset type directory name (`skills`, `agents`, `commands`, `output-styles`, `rules`, `context`, or `hooks`).

If the asset type is `hooks` or `output-styles`, remind the user that they must also register the asset in the agent's `settings.json` after deploying.

## Grouping directories

nd supports one level of grouping inside asset type directories. For example:

```text
skills/
├── coding/              # grouping directory
│   ├── code-review/     # skill (has SKILL.md)
│   └── refactoring/     # skill (has SKILL.md)
└── writing/             # grouping directory
    └── tone-check/      # skill (has SKILL.md)
```

If the user wants to organize their asset inside a group, create it one level deeper. The scanner recurses one level into non-matching directories to find assets inside grouping folders.

## Manifest-based sources

If the source contains an `nd-source.yaml` manifest, convention-based scanning is disabled. The manifest must explicitly list the directory paths for each asset type. After creating the asset, verify the manifest includes a path entry that covers the new asset's location. If not, update `nd-source.yaml`:

```yaml
paths:
  skills:
    - skills
    - custom/path/to/more-skills
```
