---
title: "Fix markdown HTML parity and llms.txt placement at the root"
id: "9vq5ht"
status: pending
priority: medium
type: bug
tags: ["ci", "docs"]
created_at: "2026-05-17"
dependencies: []
---

## Fix markdown/HTML parity and llms.txt placement at the root

### Objective

Pattern expansion of seed s7mpza, which only describes symptoms. The investigation found two concrete structural root causes: a greedy `(?s).*` regex in the section markdown layout that truncates the entire markdown body at the first Hugo shortcode (losing card navigation on every section index page), and an llms.txt directive placed only in `<head>` meta and a `display:none` footer div emitted at the very end of the body (after Hextra's heavy navbar/sidebar).

### Tasks

- [ ] `site/layouts/section.markdown.md:2` -- replace the greedy `(?s)\{\{[<%].*` strip with a non-greedy per-shortcode-block strip (or render shortcodes) so prose after `{{< cards >}}` survives in `.md` output
- [ ] Verify `.md` output of `docs/guide/asset-types/_index.md`, `site/content/docs/guide/_index.md`, `reference/_index.md`, `docs/_index.md`, `about.md` matches HTML
- [ ] `site/layouts/partials/custom/footer.html:1` -- move/duplicate the visible llms.txt directive to a top-of-content partial so it lands in the top 50% of converted text
- [ ] `site/layouts/partials/custom/head-end.html:1-2` -- keep meta tags; add an early-body visible directive line
- [ ] Reduce/segment Hextra pre-article boilerplate or add an early machine-readable summary so afdocs content-start lands < 10%
- [ ] `site/hugo.yaml:75` -- fix the description (overlaps the agent-agnostic task r2uj81)
- [ ] Run `npx afdocs check <site>` locally to confirm before pushing

### Acceptance criteria

- `markdown-content-parity` reports 0 substantive-diff pages
- The llms.txt directive appears in the top 50% of every page's converted text
- `content-start-position` < 10% on all sampled pages
- The afdocs workflow exits 0, overall score >= 98, issue #104 auto-closes

### References

- Seed task: s7mpza -- `tasks/cli/s7mpza-fix-afdocs-compliance.md`
- Root cause: `site/layouts/section.markdown.md:2`; `site/layouts/partials/custom/footer.html:1`, `head-end.html:1-2`
- `site/hugo.yaml:60-75`; `.github/workflows/afdocs-check.yml`

### Merged scope (supersedes s7mpza)

This task supersedes cancelled seed `s7mpza`, which described only symptoms of the same afdocs failure / GitHub issue #104. Reproduction (from s7mpza): push a docs change to `main` (or run the "AFDocs check" workflow via `workflow_dispatch`); the workflow exits non-zero and creates/updates issue #104. Symptom detail: `markdown-content-parity` FAIL (16/50 pages, avg 19% missing), `llms-txt-directive` WARN (buried past 50%), `content-start-position` WARN (15/50 pages start 10-50% in). The acceptance criteria above already cover s7mpza's bar (workflow exits 0, score >= 98, issue #104 auto-closes).
