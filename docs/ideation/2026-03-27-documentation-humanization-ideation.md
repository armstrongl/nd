---
date: 2026-03-27
topic: documentation-humanization
focus: Drastically improve documentation to look human-written, not AI-generated
---

# Ideation: Documentation Humanization

## Codebase Context

nd is a Go CLI tool (v0.1.0) for managing coding agent assets (skills, agents, commands, output-styles, rules, context, plugins, hooks) via symlink-based deployment. Built with Cobra + Bubble Tea v2, released via goreleaser + Homebrew cask.

**Current documentation suite:** README.md, ARCHITECTURE.md, CONTRIBUTING.md, 5 guide pages (getting-started, user-guide, profiles-and-snapshots, configuration, creating-sources), 30+ auto-generated command reference pages.

**Core problem:** All docs were written from a spec using a doc generation pipeline. They are technically accurate and well-organized, but uniformly formatted — same heading cadence, bullet-heavy prose, zero personal voice, no "why" narrative, no gotchas, no real-world experience. Every guide follows the same template: heading, 1-2 sentence intro, code block, heading, repeat. No variation, no asides, no admissions of limitations.

**Specific issues found:**
- ARCHITECTURE.md has factual errors (TUI `app/` and `components/` subpackages don't exist; it's a flat 22-file package)
- getting-started.md line 139 says "TUI Dashboard -- Coming soon (under redesign)" but TUI is complete (7 phases, 290 tests)
- 0 of 17 commands have Cobra `Example` fields, so all auto-generated reference pages are skeletons
- No troubleshooting content anywhere
- No origin story, no mental model diagram, no "why symlinks" explanation
- The `settings.json` manual registration requirement for hooks/output-styles is a single unexplained parenthetical
- Zero annotated terminal output showing what `nd list` or `nd status` actually prints

**Past learnings (from .claude/ reports):**
- CLI UX audit found 0/17 commands had examples, ~70% of error messages lack actionable next steps
- Documentation design spec at `docs/supapowers/specs/2026-03-19-documentation-and-distribution-design.md` is the canonical content plan
- TUI is explicitly a documentation non-goal in the design spec (still evolving at time of authoring) — but it has since shipped

**Approach for personal voice:** The author will write all personal/opinionated content themselves. Ideas should provide structure and scaffolding; the author fills in voice, origin, and experience-based callouts.

## Ranked Ideas

### 1. Show what actually happens on disk after a deploy
**Description:** Add annotated before/after filesystem views (e.g., `ls -la ~/.claude/`) showing what symlinks look like, where they point, and how scope changes the target directory. Include the symlink arrow notation: `~/.claude/skills/greeting -> ~/my-assets/skills/greeting`. Can live in getting-started (after first deploy step) or as a standalone "How nd Works" concepts page.
**Rationale:** The entire product is symlinks, and not one doc shows what a symlink looks like on disk. This is the kind of thing only a real author writes — because they got the "wait, is this copying files?" question too many times. No AI pipeline generates this unprompted.
**Downsides:** Requires maintaining the example if directory conventions change. Adds length to getting-started.
**Confidence:** 95%
**Complexity:** Low
**Status:** Explored (brainstorm 2026-03-27)

### 2. Create a troubleshooting & gotchas guide
**Description:** A new `docs/guide/troubleshooting.md` covering the most common failure states with specific symptoms, causes, and recovery steps: broken symlinks after source moves, `nd doctor` output interpretation, context file collisions with existing files, profile switch partial failures and auto-snapshot recovery, `--ff-only` git failures, orphan assets after source removal. Include the "escape hatches" (manual `settings.json` registration for hooks/output-styles).
**Rationale:** Zero troubleshooting content exists today. Troubleshooting pages are the most authentically human documentation — they're literally a record of things that went wrong. Specific failure modes with specific resolutions can only come from someone who encountered the failure.
**Downsides:** Requires actually knowing or testing the failure modes to document accurately. Stale troubleshooting is worse than no troubleshooting.
**Confidence:** 92%
**Complexity:** Medium
**Status:** Unexplored

### 3. Dog-fooding notes sprinkled through all guides
**Description:** Add brief inline callouts throughout existing guides — one per major section — written as personal notes: "I use this daily," "I added --dry-run because I kept deploying to the wrong scope," "this is the command I actually run most." Not a new page, but a texture change across all pages. The author writes every callout.
**Rationale:** Impossible to generate from a spec. Even one sentence per section transforms a page from "template output" to "someone lives here." The strongest signal of tool authorship is evidence of tool usage embedded in the documentation itself.
**Downsides:** Requires the author to write every callout. Can feel forced if overdone.
**Confidence:** 90%
**Complexity:** Low (per callout, distributed across all docs)
**Status:** Unexplored

### 4. Document what the agent sees after deploy
**Description:** Add a section (in getting-started or user-guide) explaining what happens from Claude Code's perspective after `nd deploy`. Does it pick up new skills immediately? Need a session restart? For hooks and output-styles, explain the manual `settings.json` registration step that is currently a single unexplained parenthetical in creating-sources.md. For project-scope context files, explain that the agent must be launched from that project directory.
**Rationale:** nd's entire value proposition is that deployed assets become active agent capabilities. The link between "nd deploy" and "my agent now behaves differently" is completely undocumented. This is the kind of gap that only a real user notices.
**Downsides:** Requires documenting Claude Code behavior that could change. May need a caveat about agent-specific behavior.
**Confidence:** 88%
**Complexity:** Low
**Status:** Unexplored

### 5. Known limitations and honest scope statement
**Description:** A section (in README or a dedicated page) that states what nd deliberately does NOT do: doesn't install agents, doesn't manage `settings.json`, doesn't sync bidirectionally, doesn't version-pin assets, doesn't watch for changes. Include specific known edge cases. Frame as "honest and confident," not apologetic.
**Rationale:** AI-generated docs never volunteer limitations. A real developer knows exactly where the tool is fragile and says so. This is the single most reliable differentiator between generated and authored documentation.
**Downsides:** Could discourage adoption if too negative. Tone matters.
**Confidence:** 88%
**Complexity:** Low
**Status:** Unexplored

### 6. Annotated terminal output for `nd list` and `nd status`
**Description:** Add real terminal output blocks showing what `nd list` and `nd status` actually print, with inline comment annotations explaining every column, symbol, and indicator. Currently the docs describe output in prose ("Assets marked with `*` are deployed") but never show it.
**Rationale:** A reader running `nd list` for the first time has no reference frame. Annotated terminal output is unfakeable evidence that someone actually ran the tool and documented what they saw.
**Downsides:** Needs updating if output format changes. Must be generated from actual `nd` runs.
**Confidence:** 92%
**Complexity:** Low
**Status:** Unexplored

### 7. Rewrite the README with motivation, mental model, and personal voice
**Description:** Replace the feature-bullet opening with a problem statement (what was annoying before nd), a one-paragraph mental model ("nd connects source directories to your agent's config via symlinks"), and a concrete Quick Start that shows what happened, not just what to type. Keep the command table but frame it as reference, not the selling point. Optionally include a one-line deadpan name acknowledgment.
**Rationale:** The README is the front door. Currently it opens with four mechanism-describing bullets that mirror internal FR labels. A human-written project page starts with "I built this because..." and shows you the payoff.
**Downsides:** HIGH BACKFIRE RISK. An AI-generated origin story has a distinct smell. The personal motivation section must be written by the author, not generated.
**Confidence:** 80%
**Complexity:** Medium
**Status:** Unexplored

### 8. Design Decisions page (conversational, not ADR)
**Description:** A single page covering 3-4 non-obvious choices in a blog-post voice: symlinks over copies (tradeoff: source deletion breaks deploys silently), YAML over JSON for config, no central registry by design, XDG deviation (`~/.config/nd/` for everything instead of splitting config/data per XDG). Each entry: what was chosen, what was considered, what the known downside is.
**Rationale:** The spec has rich reasoning for every decision (FR-009 on symlinks, A8 on XDG) that never surfaces in user docs. Trade-off admissions are the strongest possible human authorship signal.
**Downsides:** Requires the author to state trade-offs honestly, including the ones they're not proud of.
**Confidence:** 85%
**Complexity:** Medium
**Status:** Unexplored

### 9. Populate Cobra Example fields for top commands
**Description:** A code change: add `Example` strings to the Cobra command definitions for `deploy`, `remove`, `source add`, `profile create`, `profile switch`, `snapshot save`, `snapshot restore`, and `status`. These auto-render into the 30+ reference pages under an "Examples" heading. Currently 0 of 17 commands have examples.
**Rationale:** A 30-minute code change that fixes every auto-generated reference page simultaneously. The reference pages currently look like skeleton man pages. Adding examples makes them feel curated rather than generated.
**Downsides:** Code change, not pure docs work. Examples need to be realistic.
**Confidence:** 95%
**Complexity:** Low (code change)
**Status:** Unexplored

### 10. The `settings.json` gap — honest about what nd can't automate
**Description:** Expand the terse "(requires manual settings.json registration)" footnote in creating-sources.md into a full explanation: why nd deliberately doesn't write to `settings.json` (risk of conflicts with agent's own config management), what the user needs to do manually (exact JSON snippet for hooks and output-styles), and where to find `settings.json`. The spec has a full non-goal paragraph about this that never made it to user docs.
**Rationale:** The single most likely place a first-time user will get stuck and think nd is broken. A user deploys a hook, nothing happens, and the only doc is a parenthetical. Converting frustration into a trust moment — "we thought about this; here's why and here's the workaround" — is peak authored documentation.
**Downsides:** Exposes a gap in the tool's automation. But hiding it is worse.
**Confidence:** 90%
**Complexity:** Low
**Status:** Unexplored

### 11. Rewrite profiles & snapshots as a workflow narrative
**Description:** Restructure profiles-and-snapshots.md to open with a problem scenario ("you do client work and personal projects — you don't want the same skills for both"), explain the origin tracking model (manual/pinned/profile:X) as a first-class concept before diving into commands, and document what happens when a switch fails partway through (auto-snapshot recovery path). Move the workflow example from the bottom to near the top.
**Rationale:** The most "command dump" feeling page in the docs. Origin tracking — the mechanism that makes switching non-destructive — appears once in a parenthetical and is never explained. Profile switch failure recovery has zero error paths documented.
**Downsides:** Longer page. Risk of over-explaining for users who just want command syntax.
**Confidence:** 85%
**Complexity:** Medium
**Status:** Unexplored

### 12. Create a scripting, automation & CI guide
**Description:** A new `docs/guide/scripting.md` covering: JSON output schemas for `nd list --json` and `nd status --json`, exit codes (especially on partial failure), a dotfiles bootstrap script example, a CI workflow snippet, and the operation log reframed as a debugging/audit tool. Include a filesystem map of all 5 locations nd touches.
**Rationale:** nd was clearly designed for automation (`--json`, `--yes`, `--dry-run`, `--quiet`, JSONL operation log) but the docs expose none of this to the power-user audience. One example line exists across all guides.
**Downsides:** Requires testing actual JSON output shapes and exit codes.
**Confidence:** 82%
**Complexity:** Medium-High
**Status:** Unexplored

### 13. CHANGELOG.md with editorial voice
**Description:** Create a CHANGELOG.md that reads like release notes written by a person: "v0.1.0 — First public release. Ships with source management, deploy engine, profiles, snapshots, and a TUI. The TUI took longer than everything else combined." Brief editorial asides per release.
**Rationale:** A changelog is the strongest "this project is alive" signal for any open-source tool. The project has no public changelog. One sentence of commentary per release transforms it from a git log dump to evidence of authorship.
**Downsides:** Premature for v0.1.0 (only one release). Sets an expectation of continued editorial releases.
**Confidence:** 78%
**Complexity:** Low
**Status:** Unexplored

### 14. Fix all factual errors and stale content
**Description:** Batch fix across all docs: correct ARCHITECTURE.md TUI section (remove `app/` and `components/` subpackage claims, fix "dashboard-centric with tabbed asset views" to "menu-driven wizard-style"), remove stale "TUI Dashboard — Coming soon (under redesign)" from getting-started.md line 139, verify the `backup` service in the architecture diagram matches an actual package.
**Rationale:** Factual errors erode trust in all other documentation. A "coming soon" for a shipped feature signals abandoned docs. These are table-stakes fixes that must happen regardless.
**Downsides:** None. Pure correctness fixes.
**Confidence:** 98%
**Complexity:** Low
**Status:** Unexplored

## Rejection Summary

| # | Idea | Reason Rejected |
|---|------|-----------------|
| 1 | Explain the name "nd" / Napoleon Dynamite | Too small for standalone; fold into README rewrite as one deadpan line |
| 2 | Collapse 30+ reference pages into a cheatsheet | Destroys searchability/linkability; auto-generated refs are low-maintenance |
| 3 | Multi-agent futures / agent registry as user-facing | Speculative roadmap in docs is a strong AI-generation signal; ships one agent today |
| 4 | De-AI the prose (cross-cutting voice pass) | Too vague independently; describes the outcome of other ideas, not a standalone task |
| 5 | Real-world example names instead of placeholders | The prose style is the problem, not the noun choices |
| 6 | Source aliasing and naming collision rules | Too niche for v0.1.0; partially documented already |
| 7 | Elevate nd-source.yaml manifest to own page | Better as enhanced coverage within creating-sources; own page is overshoot |
| 8 | "nd for teams" as standalone section | Better as subsection within configuration and profiles guides |
| 9 | Explain scope as standalone idea | Too small; fold into getting-started mental model and profiles rewrite |
| 10 | Surface operation log separately | Covered inside scripting/CI guide |
| 11 | Filesystem map separately | Covered inside scripting/CI guide |
| 12 | Origin tracking as standalone explainer | Folded into profiles & snapshots rewrite |
| 13 | Command reference grouped by task | Low leverage vs maintaining two reference structures |
| 14 | Canonical example source repo on GitHub | Outside doc scope; separate project |
| 15 | "How I structure my source repos" guide | Overlaps with dog-fooding notes; too narrow for standalone |
| 16 | "Last verified" timestamps on guides | Maintenance trap; will go stale and look worse than no timestamp |
| 17 | Badges that mean something | Too small; low leverage on "looks human" goal |
| 18 | Source manifest cookbook | Better as enhanced coverage within creating-sources |
| 19 | Source priority sidebar | Too small; fold into config guide |
| 20 | Absolute vs relative decision guide | Too small; fold into config guide as aside |
| 21 | XDG deviation footnote | Folded into Design Decisions page |
| 22 | Plugin export gap explanation | Niche; plugins already noted as special |
| 23 | "Symlink is the contract" conceptual explainer | Overlaps with idea #1 (on-disk mental model); merge |
| 24 | "Before you move your skills folder" callout | Folded into troubleshooting guide |
| 25 | "What nd does NOT do" as separate from known limitations | Merged into idea #5 (known limitations & honest scope) |
| 26 | Operation log query cookbook | Folded into scripting/CI guide |
| 27 | Deploy vs profile vs snapshot decision tree | Folded into profiles rewrite |

## Session Log
- 2026-03-27: Initial ideation — 47 raw candidates generated across 6 frames, filtered to 7 survivors
- 2026-03-27: Second round — 23 additional candidates from 3 new frames (doc structure, trust signals, author-only content), merged and filtered to 14 total survivors
- 2026-03-27: Idea #1 (on-disk mental model) selected for brainstorming
