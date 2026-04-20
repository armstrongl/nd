---
title: "Docs Tier 2: fill content gaps"
id: "rrts5a"
status: pending
priority: medium
type: chore
tags: ["docs"]
created_at: "2026-04-20"
---

## Docs tier 2: fill content gaps

### Objective

Address 7 documentation content gaps identified during the better-docs audit (2026-04-05). These are Tier 2 items: existing pages that have missing, misleading, or duplicated content. Fixing these improves accuracy and usability of the current documentation without requiring new guides.

### Tasks

- [ ] **T2-1: Document `nd init` prompt behavior.** Describe what yes/no means during interactive init, and what happens when the target agent is not installed. Update the relevant section in `docs/getting-started.md` or `docs/commands/init.md`.
- [ ] **T2-2: Add 5 missing troubleshooting entries.** Add entries for: ambiguous asset resolution, non-TTY confirmation failures, config-already-exists errors, no-active-profile errors, and deploy conflict resolution. Add these to `docs/troubleshooting.md` or the appropriate troubleshooting section.
- [ ] **T2-3: Document `nd uninstall`.** The command has a `docs.guides` annotation but is absent from getting-started and all user-facing guides. Add uninstall instructions to `docs/getting-started.md` and verify the guide reference resolves correctly.
- [ ] **T2-4: Fix duplicate "Filter by type" headings.** In `docs/getting-started.md`, there are two "Filter by type" headings (approx. line 96 for list, line 120 for deploy). Disambiguate them (e.g., "Filter list by type" and "Filter deploy by type") or restructure the sections.
- [ ] **T2-5: Call out `nd source remove --yes` destructive behavior.** The `--yes` flag silently removes all deployed assets from the source. This destructive default is not documented. Add a warning/admonition to the source remove command reference and any guides that mention source removal.
- [ ] **T2-6: Fix `plugins.md` "Create a plugin" example.** The current example conflates source directory layout with export output layout. Separate them into two distinct examples showing: (a) the source structure before export, and (b) the exported plugin structure.
- [ ] **T2-7: Clean up unused `guideTitles` in gendocs.** There are 8 unused `guideTitles` entries for asset-type slugs. Either add the corresponding `docs.guides` annotations to commands so the entries are used, or remove the dead entries from gendocs.
- [ ] Run `scripts/lint-docs.sh` on all modified documentation files before committing.

### Acceptance criteria

- All 7 content gaps are addressed with accurate, style-compliant documentation
- `scripts/lint-docs.sh` passes on all modified files
- No new duplicate headings exist in getting-started.md
- `nd uninstall` appears in at least one user-facing guide
- The `nd source remove --yes` destructive behavior is explicitly called out with a warning
- The plugins.md example clearly separates source layout from export output
- Unused gendocs entries are either wired up or removed

### References

- Better-docs audit (2026-04-05), Tier 2 findings
- PR #45 (`better-docs` branch) for audit context
- Documentation style rules: `.claude/CLAUDE.md` (sentence case, base verbs, shell fences, etc.)
