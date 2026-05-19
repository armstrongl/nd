---
title: "Update docs to be agent-agnostic"
id: "vkerqg"
status: pending
priority: high
type: chore
tags: ["docs"]
created_at: "2026-04-20"
verify:
  - type: bash
    run: "scripts/lint-docs.sh"
  - type: bash
    run: "go build -o /tmp/nd-gendocs ./cmd/gendocs && rm -f /tmp/nd-gendocs"
  - type: bash
    run: "go run ./cmd/gendocs /tmp/nd-refdocs && rm -rf /tmp/nd-refdocs"
  - type: assert
    check: "Every remaining 'Claude Code' / '~/.claude' / '.claude/' string in docs/guide/ and README.md is either (a) part of a per-agent table/example that also shows the Copilot CLI equivalent, or (b) inside an explicit agent-specific callout. No generic prose or generic example assumes Claude Code is the only agent."
  - type: assert
    check: "The canonical agent identifier `claude-code` is preserved wherever it documents the `--agent` flag value or the `default_agent` config value; it is NOT reworded to prose."
  - type: assert
    check: "All intra-doc Markdown links (`[text](path)` / `[text](#anchor)`) in docs/guide/ still resolve after edits."
context:
  - "docs/guide/glossary.md"
  - "docs/guide/how-nd-works.md"
  - "docs/guide/troubleshooting.md"
  - "docs/guide/configuration.md"
  - "docs/guide/getting-started.md"
  - "docs/guide/user-guide.md"
  - "docs/guide/creating-sources.md"
  - "docs/guide/asset-types/agents.md"
  - "docs/guide/asset-types/skills.md"
  - "docs/guide/asset-types/context.md"
  - "README.md"
  - "internal/agent/registry.go"
  - "cmd/gendocs/main.go"
  - "cmd/root.go"
  - "scripts/lint-docs.sh"
---

## Update docs to be agent-agnostic

### Objective

Implement GitHub issue #77 ("Update docs to be non-specific to Claude Code",
https://GitHub.com/armstrongl/nd/issues/77): make `docs/guide/` and `README.md`
read as a reusable, agent-agnostic template instead of assuming Claude Code is
the only supported agent.

Important grounding (verify before starting — the original premise was stale):

- nd supports exactly **two** built-in agents, hardcoded in
  `internal/agent/registry.go:35-58`:
  - `claude-code`: global dir `~/.claude`, project dir `.claude`,
    context file `CLAUDE.md`, supports all deployable asset types.
  - `copilot`: global dir `~/.copilot`, project dir `.github`,
    context file `copilot-instructions.md`, supports only
    `skill`, `agent`, `context`.
  There is **no** generic support for Cursor/Windsurf/etc. Do NOT add them to
  any "supported agents" list — the registry would not back the claim.
- `claude-code` (kebab-case) is the **canonical agent identifier**, not
  branding. It is a valid `--agent` flag value (`cmd/root.go:57`:
  `"target agent (e.g., claude-code, copilot)"`) and the `default_agent`
  config value (`cmd/init_cmd.go:123`). It MUST be preserved everywhere it
  documents the flag/config value.
- `docs/reference/*.md` is **auto-generated** by `cmd/gendocs/main.go` from the
  Cobra command definitions in `cmd/`. Its `claude-code` strings come from
  Cobra flag help (e.g. `cmd/root.go:57`) — legitimate, not branding. Do NOT
  hand-edit `docs/reference/`; it is excluded from `scripts/lint-docs.sh`
  (the linter skips `*/reference/*`). No source-string change is required for
  this task — the existing flag help (`claude-code, copilot`) is already
  agent-correct.
- The docs are **already largely agent-aware**. Most `Claude Code` / `.claude/`
  occurrences sit in per-agent tables/examples that also show the Copilot CLI
  equivalent (e.g. `docs/guide/glossary.md:38`, `:214-217`;
  `docs/guide/how-nd-works.md:82`, `:203`; `docs/guide/troubleshooting.md:265-271`).
  Those are **correct documentation** and must be kept. The work is to fix the
  **remaining generic prose and single-agent examples** that silently assume
  Claude Code, not to strip every mention.

Real counts (verified 2026-05-17, will drift — re-measure with the commands in
Acceptance criteria): `grep -rio "claude code" docs/guide README.md` ≈ 78 total
across `docs/` but only the `docs/guide/` + `README.md` set is in scope (the
linter's default set, excluding `docs/brainstorms/`, `docs/plans/`,
`docs/solutions/`, `docs/reference/`). README has 2 occurrences, both already
list Copilot CLI alongside Claude Code and need no change.

### Tasks

Scope = files checked by `scripts/lint-docs.sh` with no args: everything under
`docs/guide/` (recursively, including `docs/guide/asset-types/*.md`) plus
`README.md`, `CONTRIBUTING.md`, `ARCHITECTURE.md`. Do NOT edit `docs/reference/`,
`docs/brainstorms/`, `docs/plans/`, or `docs/solutions/`.

For each file below, the rule is: keep occurrences that appear in a per-agent
table or a labelled per-agent example; rewrite occurrences in generic prose or
single-agent-only examples so they read agent-neutrally (e.g. "your coding
agent", "the agent's config directory") OR show both agents.

- [ ] `docs/guide/how-nd-works.md` (14 "Claude Code", 13 ".claude/"): the
  single-agent ASCII diagram (`:30-31`) and the global/project deploy examples
  (`:56`, `:65`, `:72`, `:88-89`, `:98-99`, `:178`, `:184`) only show
  `~/.claude/`. Either add the Copilot CLI equivalent inline or replace the
  literal path with a generic placeholder plus a one-line note that the real
  path depends on the agent (see the existing per-agent table at `:82` and the
  dedicated `### Claude Code` / Copilot sections at `:124-158` as the pattern to
  mirror). Keep the per-agent table (`:82`, `:203`) and the labelled
  `### Claude Code` section unchanged.
- [ ] `docs/guide/glossary.md` (11 "Claude Code", 7 ".claude/"): the
  "coding agent" definition at `:55` and the commands note at `:63` are correct
  (already name both agents) — leave them. The path examples at `:69-70`,
  `:282-285` already show both — leave them. Check `:22` ("like Claude Code")
  and the per-agent tables at `:38`, `:214-217` are kept as-is (correct). No
  rewrite likely needed here beyond verifying; do NOT add Cursor/Windsurf to the
  "coding agent" definition.
- [ ] `docs/guide/troubleshooting.md` (7 "Claude Code", 6 ".claude/"): the
  diagnostic commands and the "Agent directory differences" table at
  `:263-271` are already agent-aware — keep. Verify the example at `:179`,
  `:182-183`, `:254-255` shows or notes the Copilot CLI path (they already do —
  confirm no single-agent-only example remains).
- [ ] `docs/guide/configuration.md` (3 "Claude Code", 4 "claude-code",
  2 ".claude/"): `:50`, `:54`, `:80-82`, `:96`, `:181` reference `claude-code`
  as the `default_agent`/`--agent` value and the built-in default dirs — these
  are correct config documentation; KEEP the literal `claude-code` token.
  Confirm `:77`, `:88` already mention both built-in agents (they do).
- [ ] `docs/guide/getting-started.md` (4 "Claude Code", 3 ".claude/"): find
  any "install Claude Code"/`~/.claude/` step that reads as the only path and
  add the Copilot CLI alternative or a generic note.
- [ ] `docs/guide/user-guide.md` (2 "Claude Code", 1 "claude-code",
  2 ".claude/"): same rule — rewrite generic prose, keep `--agent claude-code`
  command examples literal.
- [ ] `docs/guide/creating-sources.md` (1 "Claude Code", 2 ".claude/"):
  same rule.
- [ ] `docs/guide/asset-types/*.md` — `agents.md` (3/3), `skills.md` (3/3),
  `context.md` (3 + 2 kebab/3), `commands.md` (1/3), `hooks.md` (2/2),
  `output-styles.md` (2/2), `rules.md` (1/3), `plugins.md` (1/0): for each,
  ensure single-agent deploy-path examples either show both agents or use a
  generic placeholder with a per-agent note. Note `commands.md` and
  `output-styles.md`/`hooks.md` legitimately describe Claude-Code-only asset
  types (per registry: Copilot supports only skill/agent/context) — keep
  Claude-Code specificity there but state explicitly that the asset type is
  Claude-Code-only (mirror `glossary.md:63` and `:230`).
- [ ] `README.md` (2 "Claude Code"): lines 14 and 21 already say
  "Claude Code and Copilot CLI" / "Deploy to Claude Code or Copilot CLI" — these
  are correct multi-agent statements. Verify only; no edit expected.
- [ ] Run `scripts/lint-docs.sh` (no args) and fix any new style errors the
  rewrites introduce (notably: forbidden words "simply/just/easy", `\`\`\`bash`
  fences must be `\`\`\`shell`, sentence-case headings).
- [ ] Re-check all intra-doc links after edits: `grep -rnoE '\]\([^)#][^)]*\)' docs/guide`
  targets must resolve relative to their file, and `](#anchor)` targets must
  match a heading slug in the same file.

### Acceptance criteria

- `scripts/lint-docs.sh` exits 0 (errors == 0; pre-existing warnings allowed but
  do not introduce new ones via the rewrites).
- For every file in `docs/guide/**` and `README.md`, each surviving
  `Claude Code` / `~/.claude` / `.claude/` string is either in a per-agent
  table/labelled per-agent example/explicit agent-specific callout, or in a
  literal CLI/config token (`--agent claude-code`, `default_agent: claude-code`).
  No generic sentence or unlabelled example implies Claude Code is the only
  agent. Spot-check with:
  `grep -rn "\.claude/\|Claude Code" docs/guide README.md` and review each hit
  against this rule.
- The literal token `claude-code` is unchanged in `docs/guide/configuration.md`
  (`default_agent`, `agents[].name`, `--agent` rows) and any `--agent`
  command example.
- `go run ./cmd/gendocs /tmp/refcheck` regenerates without error and the
  regenerated `nd.md`/`nd_deploy.md`/`nd_export.md` still contain the
  `--agent string  target agent (e.g., claude-code, copilot)` line (proves no
  Cobra source string was wrongly altered); then `rm -rf /tmp/refcheck`.
- No file under `docs/reference/`, `docs/brainstorms/`, `docs/plans/`,
  `docs/solutions/` is modified by this task (`git status` shows changes only in
  the in-scope set).
- All intra-doc Markdown links in `docs/guide/` resolve (no broken
  `[text](path)` or `[text](#anchor)`).

### References

- GitHub issue: https://GitHub.com/armstrongl/nd/issues/77
- Close this issue when the task is completed.
- Agent registry (source of truth for supported agents + their dirs):
  `internal/agent/registry.go:35-58` (`claude-code` and `copilot` only).
- Canonical agent IDs in CLI: `cmd/root.go:57` (`--agent` flag help),
  `cmd/init_cmd.go:98,123` (`default_agent: claude-code`).
- Reference docs generator (do not hand-edit `docs/reference/`):
  `cmd/gendocs/main.go` (writes `docs/reference/`, line 20).
- Doc linter (defines the in-scope file set; excludes `*/reference/*`):
  `scripts/lint-docs.sh`.
- Pattern to mirror for correct per-agent documentation:
  `docs/guide/glossary.md:38,214-217`, `docs/guide/how-nd-works.md:82,124-158`,
  `docs/guide/troubleshooting.md:263-271` (Claude Code shown next to the Copilot
  CLI equivalent).
- Key files (full list in frontmatter `context:`).
