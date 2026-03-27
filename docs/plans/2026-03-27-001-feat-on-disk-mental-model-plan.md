---
title: "feat: Add on-disk mental model documentation page"
type: feat
status: completed
date: 2026-03-27
origin: docs/brainstorms/2026-03-27-on-disk-mental-model-requirements.md
---

# feat: Add on-disk mental model documentation page

## Overview

Create a new `docs/guide/how-nd-works.md` that shows what nd actually does to the filesystem when you deploy an asset. This is the single most important missing piece in the current documentation: the entire tool is built on symlinks, but no doc shows what a symlink deployment looks like on disk. Cross-link it from getting-started.md and README.md.

## Problem Frame

Users who run `nd deploy skills/greeting` see a success message but have no reference for what changed on their filesystem. The current docs describe symlinks in prose without ever showing one. This makes it impossible to reason about broken symlinks, scope differences, or why moving a source directory causes problems. (see origin: docs/brainstorms/2026-03-27-on-disk-mental-model-requirements.md)

## Requirements Trace

- R1. Create `docs/guide/how-nd-works.md`
- R2. Mental model section with ASCII diagram of source → nd → agent config
- R3. Before/after filesystem views for directory asset (skills/) and file asset (rules/)
- R4. Global vs project scope section with concrete paths
- R5. Context file exception section (project scope deploys to project root, not `.claude/`)
- R6. Absolute vs relative symlinks section
- R7. Brief "What the agent sees" section
- R8. Cross-link callout in getting-started.md after step 5
- R9. Links from README.md Documentation section and getting-started Next Steps

## Scope Boundaries

- This page explains what nd puts on disk. It does not document workflows (deploy, remove, sync, profiles)
- The "What the agent sees" section is 3-5 sentences + one code snippet. Full agent behavior is a separate effort
- ASCII diagrams only — no Mermaid, no images
- The author will add personal voice and callouts during or after implementation

## Context & Research

### Relevant Code and Patterns

**Deploy path computation** (`internal/agent/agent.go`):
- Non-context: `filepath.Join(configDir, assetType.DeploySubdir(), assetName)` — e.g., `~/.claude/skills/greeting`
- Context global: `filepath.Join(a.GlobalDir, contextFile)` — e.g., `~/.claude/CLAUDE.md`
- Context project: `filepath.Join(projectRoot, contextFile)` — e.g., `./CLAUDE.md` (NOT `.claude/CLAUDE.md`)
- `*.local.md` at global scope returns an error

**Scope resolution** (`internal/agent/agent.go`):
- Global: `a.GlobalDir` = `~/.claude`
- Project: `filepath.Join(projectRoot, a.ProjectDir)` = `<projectRoot>/.claude`

**Symlink strategy** (`internal/deploy/deploy.go`):
- Absolute (default): target = full source path
- Relative: target = `filepath.Rel(filepath.Dir(linkPath), sourcePath)`

**Asset type subdirectories** (`internal/nd/asset_type.go`):
- skills/, agents/, commands/, output-styles/, rules/, hooks/ — standard pattern
- context — no subdirectory, deploys directly into config dir or project root

**Directory vs file assets**:
- Directory: skills, hooks (directory symlinks)
- File: agents, commands, output-styles, rules (file symlinks)
- Context: file symlink to the `.md` file inside the source folder

### Institutional Learnings

- Context files are the most confusing asset type — they break the standard `<configDir>/<type>/<name>` pattern (documentation design spec)
- The deploy engine emits warnings for hooks and output-styles that need manual `settings.json` registration
- Conflict handling: context assets get backed up and replaced; non-context assets fail with `ConflictError`
- Existing docs should use the mental model: Source → Scan → Index → Deploy → State → Health → Repair

### Existing Guide Patterns

All existing guides (`docs/guide/*.md`) use this structure:
- `## H2` section headings, `### H3` for subsections
- Imperative voice, 1-2 sentence intros before code blocks
- `bash` code blocks for commands, plain ``` blocks for output/config
- Tables for structured references

The new page should follow these conventions while breaking the uniform cadence with the before/after filesystem views and annotated diagrams.

## Key Technical Decisions

- **Concrete paths over abstractions**: Use real filesystem paths (`~/.claude/skills/greeting`, `~/my-assets/skills/greeting`) verified against the deploy engine code, not abstract descriptions
- **Tree-style diagrams**: Use indented tree notation (like `tree` command output) for filesystem views, with `->` arrow notation for symlinks. This is the most universally readable format
- **Both directory and file assets**: Show a skills/ deploy (directory symlink) and a rules/ deploy (file symlink) to cover both cases — these look different in `ls -la`
- **Context files get their own section**: The exception is important enough to warrant dedicated treatment rather than a footnote
- **Agent pickup caveat**: State what we know (Claude Code reads from `~/.claude/`) and hedge on exact timing rather than making unverified claims

## Open Questions

### Resolved During Planning

- **Deploy paths verified**: Non-context goes to `<configDir>/<type>/<name>`, context goes to `<configDir>/<filename>` (global) or `<projectRoot>/<filename>` (project). Verified in `internal/agent/agent.go`
- **Scope paths verified**: Global = `~/.claude`, Project = `<projectRoot>/.claude`. Verified in `internal/agent/registry.go`
- **Relative symlink behavior**: Uses `filepath.Rel(filepath.Dir(linkPath), sourcePath)`. Verified in `internal/deploy/deploy.go`

### Deferred to Implementation

- [Affects R7][Needs research] Exact Claude Code skill pickup timing (mid-session vs session start). Verify empirically or state with a hedge
- [Affects R7][Needs research] Exact `settings.json` snippet for hooks/output-styles registration. Research from Claude Code documentation during implementation

## Implementation Units

- [x] **Unit 1: Create `docs/guide/how-nd-works.md`**

**Goal:** Write the standalone concepts page covering the full on-disk mental model

**Requirements:** R1, R2, R3, R4, R5, R6, R7

**Dependencies:** None

**Files:**
- Create: `docs/guide/how-nd-works.md`

**Approach:**

Page structure (7 sections):

1. **Opening** — One-paragraph hook: "nd doesn't copy files. It creates symlinks." Explain what that means in one sentence: edits to the source appear instantly in the deployed location, no redeploy needed. This section must answer "is this copying files?" within 10 seconds of reading.

2. **The Mental Model** (R2) — ASCII diagram showing the three-layer relationship:
   - Left: source directory (`~/my-assets/`)
   - Center: nd (the connector)
   - Right: agent config directory (`~/.claude/`)
   Annotate with: "Your files stay in the source. nd creates links so the agent can find them."

3. **What deploys look like** (R3) — Two before/after filesystem views:
   - **Directory asset** (skills): Show source tree with `skills/greeting/` containing `SKILL.md`, then show `~/.claude/` after deploy with `skills/greeting -> ~/my-assets/skills/greeting`
   - **File asset** (rules): Show `rules/no-emojis.md` in source, then `~/.claude/rules/no-emojis.md -> ~/my-assets/rules/no-emojis.md`
   Use tree notation with `->` arrows. Show that the parent directory (`~/.claude/skills/`) is created by nd if it doesn't exist.

4. **Global vs Project Scope** (R4) — Side-by-side or sequential comparison:
   - Global: `~/.claude/skills/greeting -> source`
   - Project: `<project>/.claude/skills/greeting -> source`
   One sentence on when to use each. Link to configuration guide for details.

5. **Context Files (The Exception)** (R5) — Explain the exception clearly:
   - Global: `~/.claude/CLAUDE.md -> source/context/my-rules/CLAUDE.md`
   - Project: `./CLAUDE.md -> source` (directly in project root, NOT `.claude/CLAUDE.md`)
   Mention `*.local.md` files are project-scope only (blocked at global scope).
   Mention that deploying a second context file to the same target backs up the existing one.

6. **Absolute vs Relative Symlinks** (R6) — Show same deploy with both strategies:
   - Absolute: `~/.claude/skills/greeting -> /Users/you/my-assets/skills/greeting`
   - Relative: `~/.claude/skills/greeting -> ../../my-assets/skills/greeting`
   One sentence on when to use relative (portable dotfiles across machines with different home dirs).

7. **What the agent sees** (R7) — Brief section (3-5 sentences):
   - Claude Code reads from `~/.claude/` — deployed skills, agents, commands, and rules are available
   - Project-scope assets require launching the agent from that project directory
   - Hooks and output-styles need an extra step: manual registration in Claude Code's `settings.json` (show the path and a minimal example snippet if researchable, otherwise note the requirement and link to Claude Code docs)

**Patterns to follow:**
- Existing guide page structure: `docs/guide/getting-started.md` (heading hierarchy, code block style)
- Tree notation examples from `docs/guide/creating-sources.md` (the directory convention section already uses tree format)

**Test scenarios:**
- Every filesystem path shown in the page matches the actual deploy engine behavior (verified against `internal/agent/agent.go` and `internal/deploy/deploy.go`)
- Context file exception paths are correct (project root, not `.claude/`)
- Both directory and file symlink cases are covered
- `*.local.md` restriction is mentioned

**Verification:**
- A reader unfamiliar with symlinks can understand what `nd deploy` does after reading the first two sections
- All paths match the code verified during planning
- The page renders correctly as plain markdown (no Mermaid or image dependencies)

---

- [x] **Unit 2: Cross-link from getting-started.md and README.md**

**Goal:** Add a brief callout in getting-started after step 5 and link the new page from README and getting-started Next Steps

**Requirements:** R8, R9

**Dependencies:** Unit 1

**Files:**
- Modify: `docs/guide/getting-started.md`
- Modify: `README.md`

**Approach:**

**getting-started.md — after step 5 (deploy):**
Add a 2-3 sentence "What just happened" paragraph after the deploy code blocks and before step 6 (Verify). Content: nd created a symlink pointing from the agent's config directory back to the source file. The source stays where it is — edit it, and the change shows up immediately. Link to `how-nd-works.md` for the full picture.

**getting-started.md — Next Steps section:**
Add a link to the new page in the Next Steps list at the bottom. Also fix the stale "TUI Dashboard -- Coming soon (under redesign)" line (line 139) since it's in the same section and the TUI is complete.

**README.md — Documentation section:**
Add "How nd Works" to the documentation links list, positioned first (before Getting Started) since it's the conceptual foundation.

**Patterns to follow:**
- getting-started.md's existing Next Steps format: `- **[Title](path)** -- Description`
- README.md's existing Documentation section format

**Test scenarios:**
- The callout appears at the right position (after step 5, before step 6)
- All links resolve to the correct file path
- The stale TUI "Coming soon" note is removed or updated
- README link list remains consistently formatted

**Verification:**
- All cross-links work when navigating from getting-started and README
- Getting-started tutorial flow reads naturally with the new callout (doesn't break the step numbering or pacing)

## Risks & Dependencies

- **Low risk**: The `settings.json` snippet for hooks/output-styles may not be easily researchable. If not found, state the requirement and link to Claude Code docs rather than guessing
- **Low risk**: Claude Code skill pickup timing is not documented in the nd codebase. Hedge the language ("skills are available in your next Claude Code session") rather than making a specific claim

## Sources & References

- **Origin document:** [docs/brainstorms/2026-03-27-on-disk-mental-model-requirements.md](docs/brainstorms/2026-03-27-on-disk-mental-model-requirements.md)
- Deploy path code: `internal/agent/agent.go` (DeployPath, contextDeployPath)
- Symlink creation: `internal/deploy/deploy.go` (deployOne)
- Asset type subdirs: `internal/nd/asset_type.go` (DeploySubdir)
- Agent config dirs: `internal/agent/registry.go` (hardcoded claude-code defaults)
- Existing tree notation example: `docs/guide/creating-sources.md` (directory convention section)
