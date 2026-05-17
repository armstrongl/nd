---
title: "Document undocumented CLI flags and correct stale doc tasks"
id: "oema5w"
status: pending
priority: medium
type: chore
tags: ["docs"]
created_at: "2026-05-17"
dependencies: ["rrts5a", "ahfhih"]
---

## Document undocumented CLI flags and correct stale doc tasks

### Objective

Pattern expansion / correction of seeds rrts5a and ahfhih. Several seed-task claims are factually wrong or already satisfied (T2-4, T2-5, T3-18, parts of T2-7/T3-19), and the genuinely undocumented surface (the full `nd export`/`export marketplace` flag set) is not enumerated by either seed. Correct the stale premises and document the real gaps.

### Tasks

- [ ] Document `nd export` flags in a guide: `--name`, `--version`, `--author`, `--email`, `--license`, `--overwrite` (`cmd/export.go:159-168`) and marketplace `--owner`, `--plugins` (`cmd/export.go:301,303`) -- none appear in any `docs/guide` file
- [ ] Document `--absolute` (`cmd/deploy.go:292`) in a guide (0 guide files; confirms ahfhih T3-11)
- [ ] `cmd/gendocs/main.go:183` -- remove or wire the single unused `glossary` guideTitle (NOT 8 unused as rrts5a T2-7 claims; the 8 asset-type slugs are used)
- [ ] Single-source the "Napoleon Dynamite" decision at `cmd/root.go:23` (affects 18 ref pages via `docs.related`, not 32)
- [ ] Correct rrts5a T2-4: only one non-heading "Filter by type:" at `docs/guide/getting-started.md:101` -- close as non-issue or restructure
- [ ] Correct rrts5a T2-5: `source remove --yes` warning already exists at `docs/guide/creating-sources.md:141` -- close as satisfied
- [ ] Correct ahfhih T3-18: `nd profile create` `Example` already populated (`cmd/profile.go:49`); optionally add a `--assets` example
- [ ] Document the TUI (`cmd/root.go:47` `tui.Run`; `internal/tui/`) -- ahfhih T3-8 valid, zero guide coverage

### Acceptance criteria

- Every `nd export` / `export marketplace` flag has a guide-level explanation/example
- `--absolute` documented in a guide
- `gendocs` has no unused guideTitle entries (or `glossary` is wired)
- Stale seed items reconciled (closed or rewritten with correct line refs)
- `scripts/lint-docs.sh` passes on modified guide files

### References

- Seed tasks: rrts5a -- `tasks/cli/rrts5a-docs-tier2-content-gaps.md`; ahfhih -- `tasks/cli/ahfhih-docs-tier3-new-content.md`
- `cmd/export.go:159-168,301-305`; `cmd/deploy.go:292`; `cmd/profile.go:49,130-131`; `cmd/gendocs/main.go:177-193`; `cmd/root.go:23,47`
