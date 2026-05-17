---
title: "Agent-agnostic docs: cmd source strings and full file scope"
id: "r2uj81"
status: pending
priority: high
type: chore
tags: ["docs"]
created_at: "2026-05-17"
dependencies: ["vkerqg"]
---

## Agent-agnostic docs: cmd/ source strings and full file scope

### Objective

Pattern expansion / correction of seed vkerqg. The seed's "~150 occurrences across 52 files + 3 in README" is a significant overcount: real totals are ~76 "Claude Code" lines across ~22 files. More importantly, generated `docs/reference/` text comes from Cobra command definitions in `cmd/` -- editing the `.md` files alone is futile because `gendocs` regenerates them. This task adds the cmd/ source root cause and the files the seed never enumerated.

### Tasks

- [ ] `cmd/export.go:36,37,41,216` -- replace "Claude Code plugin/marketplace" phrasing in `Short`/`Long`/`Example` (root cause of `docs/reference/nd_export*.md`)
- [ ] `cmd/root.go:68` -- generalize `--scope` completion hints away from `~/.claude/` and `.claude/`
- [ ] Re-run `go run ./cmd/gendocs/` and confirm `docs/reference/nd_export.md`, `nd_export_marketplace.md`, `nd.md` regenerate agent-agnostic
- [ ] `docs/guide/how-nd-works.md` (12 lines), `glossary.md` (10), `troubleshooting.md` (7), `configuration.md` (3)
- [ ] `docs/guide/getting-started.md` (4 lines: 62,125,135,136) -- NOT in seed
- [ ] `docs/guide/user-guide.md` (2: 138,139) -- NOT in seed
- [ ] `docs/guide/creating-sources.md` (1: 66) -- NOT in seed
- [ ] `docs/guide/asset-types/{skills,context,agents,output-styles,hooks,rules,plugins,commands}.md`
- [ ] `README.md` (lines 14,21)
- [ ] `site/hugo.yaml:75` -- site description param mentions Claude Code (also feeds llms.txt summary) -- NOT in seed
- [ ] Leave legitimate agent IDs intact (`claude-code` as `--agent` value, `default_agent` enum at `cmd/root.go:57`, `cmd/init_cmd.go:98,123`)
- [ ] Decide scope of `docs/brainstorms`, `docs/plans`, `docs/solutions` (historical artifacts; recommend out of scope)

### Acceptance criteria

- `grep -rIin "claude code" docs/guide docs/reference README.md` returns 0 (except an explicit "supported agents" list)
- `gendocs` output is agent-agnostic after regeneration
- `scripts/lint-docs.sh` passes on modified guide files
- No broken internal links

### References

- Seed task: vkerqg -- `tasks/cli/vkerqg-docs-agent-agnostic.md`
- Root cause: `cmd/export.go:36-41,216`; `cmd/root.go:68`; `cmd/gendocs/main.go`
- `site/hugo.yaml:75`
