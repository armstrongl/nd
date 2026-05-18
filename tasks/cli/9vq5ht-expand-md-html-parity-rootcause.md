---
title: "Fix markdown HTML parity and llms.txt placement at the root"
id: "9vq5ht"
status: pending
priority: medium
type: bug
tags: ["ci", "docs"]
created_at: "2026-05-17"
dependencies: []
context:
  - "site/layouts/section.markdown.md"
  - "site/layouts/home.markdown.md"
  - "site/layouts/llms.txt"
  - "site/layouts/partials/custom/footer.html"
  - "site/layouts/partials/custom/head-end.html"
  - "site/hugo.yaml"
  - "site/content/docs/_index.md"
  - "site/content/docs/guide/_index.md"
  - "site/content/docs/reference/_index.md"
  - "site/content/docs/about.md"
  - "docs/guide/asset-types/_index.md"
  - ".GitHub/workflows/afdocs-check.yml"
  - ".GitHub/workflows/deploy-docs.yml"
  - "tasks/cli/s7mpza-fix-afdocs-compliance.md"
verify:
  - type: bash
    run: "cd site && rm -rf /tmp/nd-md-parity && hugo --quiet --destination /tmp/nd-md-parity --baseURL 'http://localhost/' && echo BUILD_OK"
  - type: assert
    check: "/tmp/nd-md-parity/docs/index.md, docs/guide/index.md, docs/guide/asset-types/index.md, and docs/reference/index.md each contain the navigation that follows their first {{< cards >}} shortcode (e.g. docs/index.md links to guide/ and reference/; reference/index.md still contains the Profile/Snapshot/Source/Other command sections)"
  - type: assert
    check: "The visible llms.txt directive text appears in the first 50% of the converted (visible) text of every built HTML page, not only as a <head> meta tag and an end-of-body display:none div"
  - type: bash
    run: "npx -y afdocs@0.10.1 check http://localhost:1313/ --format scorecard --sampling deterministic || true   # serve /tmp/nd-md-parity first: hugo server -s site, or npx serve /tmp/nd-md-parity"
  - type: assert
    check: "afdocs markdown-content-parity reports 0 substantive-diff pages; llms-txt-directive passes (top 50%); content-start-position < 10% on all sampled pages; overall score >= 98"
---

## Fix markdown/HTML parity and llms.txt placement at the root

### Objective

The "AFDocs check" GitHub Action (`.github/workflows/afdocs-check.yml`) fails and keeps GitHub issue #104 open. This task fixes the two structural root causes found while investigating cancelled seed `s7mpza` (symptoms only):

1. **Markdown/HTML parity loss.** Hugo emits a `markdown` output format for every section page (`site/hugo.yaml:66-69`, `outputs.section: [html, markdown, rss]`). The section markdown layout `site/layouts/section.markdown.md` line 2 runs `.RawContent | replaceRE \`(?s)\{\{[<%].*\` "" | strings.TrimSpace`. The `(?s)` flag makes `.` match newlines and `.*` is greedy, so this strips **from the first Hugo shortcode delimiter (`{{<` or `{{%`) all the way to end of content** — not just the shortcode. Result: every section index page's `.md` output loses the card navigation (and any prose) that follows its first `{{< cards >}}` block, while the `.html` version keeps it. afdocs' `markdown-content-parity` check compares the two and flags substantive divergence.
2. **llms.txt directive placement.** The agent directive is emitted only as `<meta>` tags in `<head>` (`site/layouts/partials/custom/head-end.html:1-2`) and as a `<div style="display:none">…</div>` appended at the very end of `<body>` by `site/layouts/partials/custom/footer.html:1` (after Hextra's `<article>`/`<main>`). When afdocs converts a page to visible text, the `display:none` div is dropped (or, if counted, lands at the bottom), so `llms-txt-directive` warns the directive is buried past 50% / not in visible content.

Why it matters: agents that consume the `.md` output of `https://armstrongl.github.io/nd/docs/` (and `/docs/guide/`, `/docs/guide/asset-types/`, `/docs/reference/`) get pages stripped of all navigation, and the agent directive pointing them at `llms.txt` is not in the readable body.

### Affected pages (verified)

The bug hits exactly the four section index pages that use `section.markdown.md` and contain a `{{< cards >}}` shortcode. Verified by building locally (`hugo` 0.160.1 extended) and reading the generated `.md`:

| URL | Source file | `.md` output loses |
|-----|-------------|--------------------|
| `/docs/` | `site/content/docs/_index.md` (cards at line 9-12) | the entire Guides + Command Reference card block (all nav) |
| `/docs/guide/` | `site/content/docs/guide/_index.md` (cards at line 10-19) | all 8 guide cards (all nav) |
| `/docs/guide/asset-types/` | `docs/guide/asset-types/_index.md` (mounted via `site/hugo.yaml:17-20`; cards at line 9-18) | all 8 asset-type cards (all nav) |
| `/docs/reference/` | `site/content/docs/reference/_index.md` (first cards at line 12-23) | everything after `## Core commands` heading — the Profile, Snapshot, Source, and Other command sections (~45 of ~50 reference cards) |

Note: `site/content/docs/about.md` (output `/docs/about.md`) is a regular `page`, has **no shortcodes**, and its `.md` output is already complete — it is **not** part of this bug. The original task line referencing `about.md` was a stale assumption; do not change `about.md`.

### Reproduction

```shell
cd site
rm -rf /tmp/nd-md-parity
hugo --quiet --destination /tmp/nd-md-parity --baseURL 'http://localhost/'
cat /tmp/nd-md-parity/docs/index.md
# Observed (broken): only "# Documentation\nSet up, configure, and use nd to manage coding agent assets."
# Expected: also includes the markdown rendering of the two cards (links to guide/ and reference/)
cat /tmp/nd-md-parity/docs/reference/index.md
# Observed (broken): ends right after "## Core commands" — Profile/Snapshot/Source/Other sections gone
```

CI repro (from cancelled seed `s7mpza`, `tasks/cli/s7mpza-fix-afdocs-compliance.md:21-36`): push a `docs/**` or `site/**` change to `main` (triggers "Deploy docs" → "AFDocs check"), or run the "AFDocs check" workflow via `workflow_dispatch`. The workflow runs `afdocs@0.10.1 check https://armstrongl.github.io/nd/ --score --sampling deterministic`; on non-zero exit it creates/updates the `afdocs-check`-labelled issue (#104). Reported symptoms: `markdown-content-parity` FAIL (16/50 pages, avg 19% missing), `llms-txt-directive` WARN (buried past 50%), `content-start-position` WARN (15/50 pages start 10-50% in), overall score 98/100.

### Tasks

- [ ] Fix `site/layouts/section.markdown.md:2`. The current single line is:
  `{{ .RawContent | replaceRE \`(?s)\{\{[<%].*\` "" | strings.TrimSpace }}`
  Replace the greedy whole-tail strip with a per-shortcode-block removal so prose/markdown that follows a shortcode survives. Preferred approach: render `.Content` instead of `.RawContent` (Hugo expands `{{< cards >}}` to HTML, which is acceptable in an agent-facing `.md`), OR strip only individual shortcode invocations non-greedily: match an opening `{{<`/`{{%` and its nearest closing `>}}`/`%}}` (lazy quantifier, not a greedy run to end of content) plus self-closing tags, rather than `.*` to EOF. Mirror the simpler sibling layout `site/layouts/home.markdown.md` (no greedy strip) and `site/layouts/llms.txt` for how this repo accesses page content via `.OutputFormats`/`site.GetPage`. Whatever you choose, the four pages in the table above must keep their post-shortcode navigation.
- [ ] Rebuild and confirm parity for the four affected pages by inspecting the generated `.md` files under `/tmp/nd-md-parity/` (`docs/index.md`, `docs/guide/index.md`, `docs/guide/asset-types/index.md`, `docs/reference/index.md`) — each must contain the content/links that appear after its first `{{< cards >}}` in the source.
- [ ] Move/duplicate the visible llms.txt directive to the top of page content. Add an early-body partial (or extend an existing top-of-content Hextra partial) that emits a **visible** line such as `For AI agents: see <llms.txt URL> for a structured index of this documentation.` so it lands in the first 50% of converted text. Keep the `<meta>` tags in `site/layouts/partials/custom/head-end.html:1-2` unchanged (they are harmless). The current sink is `site/layouts/partials/custom/footer.html:1` (`<div style="display:none">…</div>` at end of `<body>`); either remove that div or keep it in addition to the new early visible directive — but the early one must not be `display:none`.
- [ ] Reduce/segment Hextra pre-article boilerplate so documentation content starts in the first <10% of converted text on the sampled pages, OR ensure the new early visible directive line itself is the first readable content so `content-start-position` passes. (Hextra renders a heavy navbar/sidebar before `<main id="content">`; placing a short machine-readable summary line high in the body is the lightest fix.)
- [ ] `site/hugo.yaml:75` — `params.description` currently says "…across tools like Claude Code." This string feeds the home `.md`/`llms.txt` output (`site/layouts/home.markdown.md:2`, `site/layouts/llms.txt:3`). Make it agent-agnostic (drop the "Claude Code" specificity). This is the same agent-agnostic-phrasing concern tracked by `tasks/cli/r2uj81-expand-docs-agent-agnostic-scope.md` (which targets `cmd/` source strings, not `hugo.yaml`); fix it here and note the cross-reference in your PR so the two tasks don't double-edit.
- [ ] Run afdocs locally against the built site before pushing. Serve the output (`cd site && hugo server` on `:1313`, or `npx -y serve /tmp/nd-md-parity`) then `npx -y afdocs@0.10.1 check <local-url> --format scorecard --fixes --sampling deterministic`. Confirm the three checks pass and score >= 98.

### Acceptance criteria

- `afdocs markdown-content-parity` reports **0** substantive-diff pages; specifically the four pages in the table keep their post-shortcode navigation in `.md` output.
- The visible llms.txt directive appears in the **top 50%** of every page's converted (visible) text — not only as `<head>` meta and an end-of-body `display:none` div.
- `afdocs content-start-position` is **< 10%** on all sampled pages.
- `hugo --destination /tmp/nd-md-parity` builds with exit code 0 (no template errors from the regex/partial change).
- The "AFDocs check" workflow exits 0, overall score **>= 98**, and GitHub issue #104 auto-closes (handled by `.github/workflows/afdocs-check.yml:121-129`).
- This satisfies cancelled seed `s7mpza`'s bar; no separate action needed for it.

### References

- Root cause 1 (parity): `site/layouts/section.markdown.md:2` (greedy `(?s)\{\{[<%].*` regex). Sibling layouts to mirror: `site/layouts/home.markdown.md`, `site/layouts/llms.txt`.
- Root cause 2 (directive): `site/layouts/partials/custom/footer.html:1` (`display:none` end-of-body div) and `site/layouts/partials/custom/head-end.html:1-2` (`<head>` meta tags).
- Hugo output config: `site/hugo.yaml:60-72` (`outputs.section: [html, markdown, rss]`); mounts: `site/hugo.yaml:14-20`; description: `site/hugo.yaml:75`.
- Affected source pages: `site/content/docs/_index.md`, `site/content/docs/guide/_index.md`, `site/content/docs/reference/_index.md`, `docs/guide/asset-types/_index.md`.
- CI: `.github/workflows/afdocs-check.yml` (uses `afdocs@0.10.1`, `--score`, `--sampling deterministic`; issue lifecycle at lines 82-129). Trigger chain: `.github/workflows/deploy-docs.yml` → "AFDocs check".
- GitHub issue: https://GitHub.com/armstrongl/nd/issues/104
- Cancelled seed (symptoms only, do not action): `tasks/cli/s7mpza-fix-afdocs-compliance.md`.
- Related (agent-agnostic phrasing, `cmd/` scope): `tasks/cli/r2uj81-expand-docs-agent-agnostic-scope.md`.

### Merged scope (supersedes s7mpza)

This task supersedes cancelled seed `s7mpza` (`tasks/cli/s7mpza-fix-afdocs-compliance.md`, marked `status: cancelled`, `cancelled_at: 2026-05-17`), which described only symptoms of the same afdocs failure / GitHub issue #104. Its reproduction and acceptance bar are folded into the sections above. Do not action `s7mpza` separately.
