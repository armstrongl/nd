---
title: "Docs Tier 3: new content guides"
id: "ahfhih"
status: pending
priority: low
type: chore
tags: ["docs"]
created_at: "2026-04-20"
---

## Docs tier 3: new content guides

### Objective

Create 12 new documentation guides and sections identified during the better-docs audit (2026-04-05). These are Tier 3 items: entirely new content that does not exist anywhere in the current documentation. Each item addresses a feature, workflow, or behavior that users have no way to discover from the docs today.

### Tasks

- [ ] **T3-8: Write a TUI guide.** The `nd` bare command launches a full interactive TUI, but there is zero guide coverage. Create `docs/guides/tui.md` covering navigation, key bindings, screen descriptions, and scope switching.
- [ ] **T3-9: Write a dedicated export/plugin workflow guide.** Document the two-phase workflow: exporting a source as a plugin, then publishing to the marketplace. Create `docs/guides/export-plugin-workflow.md`.
- [ ] **T3-10: Write a team/project-scoped config guide.** Cover relative paths, git sources for team sharing, and project-scoped vs. user-scoped configuration. Create `docs/guides/team-config.md`.
- [ ] **T3-11: Document missing CLI flags in guides.** Add usage examples and explanations for `--absolute`, `--source`, `--pattern`, `--alias`, and SSH URL support. Update the relevant command reference pages and getting-started guide.
- [ ] **T3-12: Document auto-snapshot on bulk deploy/remove failures.** Explain that nd automatically creates a snapshot before bulk operations and how to restore from it if something goes wrong.
- [ ] **T3-13: Document state file locking and concurrent process errors.** Explain the locking mechanism, what error users see when two nd processes collide, and how to resolve stale locks.
- [ ] **T3-14: Add `agents[].source_alias` to config key reference table.** The key is missing from the configuration reference. Add it with a description and example.
- [ ] **T3-15: Document ghost deployment pruning.** Explain the silent self-healing behavior during deploy/status where stale deployments are automatically cleaned up.
- [ ] **T3-16: Explain `nd sync` "Repaired" vs "Removed" output.** Users see these labels but have no documentation explaining what each means or when each occurs.
- [ ] **T3-17: Document backup pruning policy.** nd keeps the 5 most recent backups per filename. This retention policy is not documented anywhere.
- [ ] **T3-18: Add `nd profile create --assets` example.** The Cobra `Example` field for this command is empty. Add a concrete usage example showing asset selection during profile creation.
- [ ] **T3-19: Resolve "Napoleon Dynamite" branding in Related sections.** Every reference page includes a "Napoleon Dynamite" entry in its Related section. Decide whether to keep it as an Easter egg, replace it with something useful, or remove it entirely.
- [ ] Run `scripts/lint-docs.sh` on all new and modified documentation files before committing.

### Acceptance criteria

- All 12 new content items are created or addressed
- Each new guide follows the project documentation style rules (sentence case headings, base verb forms, `shell` code fences, no forbidden words)
- `scripts/lint-docs.sh` passes on all new and modified files
- New guides are linked from the appropriate index or navigation pages
- The config key reference table includes `agents[].source_alias`
- The `nd sync` output labels are explained with concrete examples
- A decision is documented for the "Napoleon Dynamite" branding question (keep, replace, or remove)

### References

- Better-docs audit (2026-04-05), Tier 3 findings
- PR #45 (`better-docs` branch) for audit context
- Documentation style rules: `.claude/CLAUDE.md`
