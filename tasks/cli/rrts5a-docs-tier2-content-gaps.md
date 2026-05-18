---
title: "Docs Tier 2: fill content gaps"
id: "rrts5a"
status: pending
priority: medium
type: chore
tags: ["docs"]
created_at: "2026-04-20"
context:
  - "docs/guide/getting-started.md"
  - "docs/guide/troubleshooting.md"
  - "docs/guide/asset-types/plugins.md"
  - "docs/reference/nd_source_remove.md"
  - "cmd/source.go"
  - "cmd/gendocs/main.go"
  - "scripts/lint-docs.sh"
verify:
  - type: bash
    run: "cd /Users/larah/Repos/Personal/nd && go build -o /tmp/nd-rrts5a ./cmd/gendocs/ && rm -f /tmp/nd-rrts5a"
  - type: bash
    run: "cd /Users/larah/Repos/Personal/nd && go run ./cmd/gendocs/ && git diff --quiet docs/reference/ || (echo 'reference docs out of sync; commit regenerated output' && exit 1)"
  - type: bash
    run: "cd /Users/larah/Repos/Personal/nd && scripts/lint-docs.sh docs/guide/getting-started.md docs/guide/troubleshooting.md docs/guide/asset-types/plugins.md"
  - type: assert
    check: "docs/reference/nd_source_remove.md documents that --yes silently removes all deployed assets from the source (the destructive default)."
  - type: assert
    check: "cmd/gendocs/main.go guideTitles map has no entry that is absent from every docs.guides annotation in cmd/*.go (the only candidate was 'glossary')."
---

## Docs tier 2: fill content gaps

### Objective

The "better-docs audit" recorded on 2026-04-05 (see PR #45, `better-docs` branch — context only; do not depend on the branch existing) flagged 7 Tier 2 documentation gaps: existing pages with missing, misleading, or duplicated content. The docs were substantially reworked after that audit, so **most gaps are already fixed**. This task is now a verify-and-close-the-remainder pass: confirm the 5 already-addressed items are correct, fix the 2 that remain (T2-5 and T2-7), and correct stale claims in this task itself.

Critical correction: the audit used a flat `docs/` layout (`docs/getting-started.md`, `docs/troubleshooting.md`, `docs/plugins.md`, `docs/commands/init.md`). That layout no longer exists. The current layout is:

- Narrative guides: `docs/guide/*.md` (e.g. `docs/guide/getting-started.md`, `docs/guide/troubleshooting.md`)
- Asset-type guides: `docs/guide/asset-types/*.md` (e.g. `docs/guide/asset-types/plugins.md`)
- Command reference: `docs/reference/nd_*.md` — **auto-generated**, do not hand-edit. Regenerate with `go run ./cmd/gendocs/` (per `CONTRIBUTING.md:70`). Frontmatter `title`/`description` and the `## Guides` block come from Cobra `Short`/`Annotations["docs.guides"]` in `cmd/*.go`; `guideTitles` map in `cmd/gendocs/main.go:177-193` maps guide slugs to link titles.

There is no `.claude/CLAUDE.md` style file. The enforced doc style rules live in `scripts/lint-docs.sh`: use ` ```shell ` not ` ```bash ` fences; forbidden words `simply`/`straightforward`/`obviously` (error) and `just`/`easy`/`easily`/`simple` (warning outside code); standard tree glyphs `├──/└──/│` not `+--`; sentence-case H2/H3 headings; `:` not `--` as list-item separators. Markdown linting is `rumdl` configured by `.rumdl.toml`.

### Tasks

- [ ] **T2-1 (verify — likely already done): `nd init` prompt behavior.** Confirm `docs/guide/getting-started.md:64-68` already describes: answer `y` to deploy built-in assets / `n` to skip (deploy later with `nd deploy --source builtin`), `--yes` to skip the prompt, behavior when no agent is detected (skips with a warning), and the "config already exists" error pointing to `nd settings edit`. If accurate, leave unchanged and check the box. Only edit if a claim drifted from `cmd/init_cmd.go` behavior.
- [ ] **T2-2 (verify — likely already done): troubleshooting entries.** Confirm `docs/guide/troubleshooting.md` already contains all 5: "Ambiguous asset name" (lines ~189-205), "Non-TTY confirmation error" (~207-217), "Config already exists" (~219-229), "No active profile" (~231-243), "Deploy conflict" (~245-259). If present and accurate, leave unchanged and check the box.
- [ ] **T2-3 (verify — already done): `nd uninstall`.** Confirm `nd uninstall` is documented in `docs/guide/getting-started.md:215-223` (Uninstall section), and that `cmd/uninstall.go:23` sets `"docs.guides": "getting-started,troubleshooting"`, both of which resolve to existing pages in the generated `docs/reference/nd_uninstall.md` (`## Guides` block, lines 47-51). If correct, check the box.
- [ ] **T2-4 (verify — already fixed): no duplicate "Filter by type" heading.** Confirm `docs/guide/getting-started.md` now has exactly one "Filter by type" heading (line ~101 under "Browse available assets"). The old duplicate (deploy section) no longer exists. Verify with: `grep -n '^#* Filter by type' docs/guide/getting-started.md` returns one match. If so, check the box. No edit needed.
- [ ] **T2-5 (fix — still required): document `nd source remove --yes` destructive behavior.** Root cause: in `cmd/source.go`, the `newSourceRemoveCmd` RunE handler at lines ~212-215 — when deployed assets exist for the source AND `app.Yes` is true (`--yes`/`-y`) — calls `removeSourceDeployments(eng, sourceID)` unconditionally, silently deleting all of that source's deployed symlinks. Without `--yes` the user gets a 3-way `promptChoice` (lines ~185-211: "Remove source and all deployed assets" / "Remove source only (orphan deployed assets)" / "Cancel"). So `--yes` is not just "skip confirmation" — it picks the destructive option. The generated reference `docs/reference/nd_source_remove.md` only shows `--yes` as "Skip confirmation prompt" with no warning. Fix by adding a warning to the Cobra command so it propagates into the generated doc on regeneration:
  - In `cmd/source.go` `newSourceRemoveCmd`, extend the `Example` string (currently `cmd/source.go:132-136`) and/or `Short` (`cmd/source.go:131`) so the generated `## Examples` / description in `docs/reference/nd_source_remove.md` makes the destructive `--yes` behavior explicit (e.g. add an example comment: `# Remove source AND all its deployed assets, no prompt`). Do NOT hand-edit `docs/reference/nd_source_remove.md` — it is regenerated.
  - Add a sentence to `docs/guide/creating-sources.md` (the guide referenced by `cmd/source.go:138` `"docs.guides": "creating-sources"`) wherever source removal is described, calling out that `nd source remove <id> --yes` removes the source and deletes all of its deployed assets without prompting; to keep deployed assets, omit `--yes` and choose "Remove source only" at the prompt. (If a separate `docs/guide/troubleshooting.md` "Non-TTY confirmation error" example uses `nd source remove ... --yes`, add a one-line caution there too.)
  - Run `go run ./cmd/gendocs/` and commit the regenerated `docs/reference/nd_source_remove.md`.
- [ ] **T2-6 (verify — likely already done): `plugins.md` example separation.** `docs/guide/asset-types/plugins.md` already has a "Directory layout" source-structure block (lines ~22-33), and a "Create a plugin" section split into "Author a plugin in your source" (source layout, ~74-95) and "Package assets for distribution" (`nd export` output, ~97-99). Confirm these two are clearly distinct (source structure vs. export output) and not conflated. If clear, check the box; only restructure if the source-vs-export distinction is genuinely muddled.
- [ ] **T2-7 (fix — claim was stale; scope reduced): remove the single unused `guideTitles` entry.** The original "8 unused entries" claim is wrong. Verified: `cmd/gendocs/main.go:177-193` defines a 15-entry `guideTitles` map. Cross-referencing every `"docs.guides"` annotation in `cmd/*.go` (deploy, init_cmd, sync, list, pin, export, profile, settings, remove, uninstall, snapshot, status, doctor, source) shows **exactly one** map key is never referenced: `"glossary"` (`cmd/gendocs/main.go:183`). All other 14 keys are used; no referenced slug is missing from the map. Resolve by either:
  - (a) Remove the `"glossary": "Glossary"` line from the `guideTitles` map in `cmd/gendocs/main.go`; or
  - (b) Add `glossary` to a relevant command's `docs.guides` annotation (e.g. a command whose generated page should link the glossary) so the entry becomes used.
  - Prefer (a) (delete the dead entry) unless a clear command-to-glossary link is warranted. Then run `go run ./cmd/gendocs/` and commit any regenerated reference changes.
- [ ] Run `scripts/lint-docs.sh <each modified guide/*.md>` and fix all `error`-level findings (warnings reviewed) before committing.
- [ ] Run `go run ./cmd/gendocs/` after any `cmd/*.go` edit and commit the regenerated `docs/reference/` files so the generated docs stay in sync (enforced by `.github/workflows/deploy-docs.yml:51` and `docs-sync.yml`).

### Acceptance criteria

- T2-1..T2-4 and T2-6 are confirmed accurate against current code/docs (boxes checked) or corrected if drift is found; no edit is made to docs that are already correct.
- `nd source remove --yes` destructive behavior is explicitly called out: the regenerated `docs/reference/nd_source_remove.md` and `docs/guide/creating-sources.md` both state that `--yes` removes the source and deletes all its deployed assets without prompting.
- The `guideTitles` map in `cmd/gendocs/main.go` has no entry absent from every `docs.guides` annotation (the lone offender `glossary` is either removed or wired up); `go run ./cmd/gendocs/` produces no `git diff` after the change is committed.
- `scripts/lint-docs.sh` exits 0 (no errors) on every modified `docs/guide/**` file.
- `go build -o /tmp/x ./cmd/gendocs/` succeeds (no compile breakage from `cmd/source.go` / `cmd/gendocs/main.go` edits).
- No new duplicate headings introduced in `docs/guide/getting-started.md`.

### References

- Audit context only: PR #45 / `better-docs` branch, Tier 2 findings (2026-04-05). Do not require the branch to exist; ground everything in the live repo below.
- `docs/guide/getting-started.md:64-68` (init prompt), `:101` (single "Filter by type"), `:215-223` (uninstall) — T2-1, T2-4, T2-3.
- `docs/guide/troubleshooting.md:189-259` — the 5 T2-2 entries.
- `docs/guide/asset-types/plugins.md:22-33`, `:74-99` — T2-6 source-vs-export split.
- `cmd/source.go:130-261` `newSourceRemoveCmd`; destructive `--yes` path at `:212-215`; interactive 3-way prompt at `:185-211`; `"docs.guides": "creating-sources"` at `:138` — T2-5.
- `docs/reference/nd_source_remove.md` (auto-generated; regenerate, do not hand-edit) — T2-5 target.
- `cmd/gendocs/main.go:149-161` (guide-link generation), `:177-193` (`guideTitles` map; `"glossary"` at `:183` is the only unused key) — T2-7.
- Doc style + regen: `scripts/lint-docs.sh` (real style rules; there is no `.claude/CLAUDE.md`), `CONTRIBUTING.md:70` (`go run ./cmd/gendocs/`), `.rumdl.toml` (markdown lint), `.github/workflows/deploy-docs.yml:51` and `docs-sync.yml` (CI regenerates/checks reference docs).
- Build: `go build -o nd .` (root) and `go build ./cmd/gendocs/`; tests: `go test ./...`.
