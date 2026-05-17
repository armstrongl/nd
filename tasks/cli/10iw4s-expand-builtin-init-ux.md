---
title: "Bring builtin source nd init UX into spec"
id: "10iw4s"
status: pending
priority: high
type: feature
tags: ["core", "onboarding"]
created_at: "2026-05-17"
dependencies: ["unaa3u"]
---

## Bring builtin source nd init UX into spec

### Objective

Pattern expansion of seed unaa3u. The builtin-source feature is implemented and green (Units 1-5 match the plan; embedded content is real, not placeholder). The gap is concentrated in Unit 6: the `nd init` interactive step diverges from the seed/plan spec -- the `[Y/n/list]` prompt is a plain yes/no with no `list` action, the default is No instead of Yes, and `--json` emits only a deployed count instead of the deployed asset list.

### Tasks

- [ ] `cmd/init_cmd.go:195` -- add the `list` action: show built-in asset names, then re-prompt
- [ ] `cmd/helpers.go:66` / `cmd/init_cmd.go:195` -- make the built-in deploy prompt default to Yes (`[Y/n]`, Enter = deploy) per spec
- [ ] `cmd/init_cmd.go:196-201` -- reconcile the non-terminal fallback (currently flips to deploy) with the seed's explicit `n: skip` branch (coordinate with task wdisqq)
- [ ] `cmd/init_cmd.go:64-66` -- include the deployed asset list (names/types) in `--json` output, not just a count
- [ ] Add/extend `cmd/init_cmd_test.go` for: list-then-deploy, default-yes-on-Enter, `--json` asset list

### Acceptance criteria

- `nd init` prompt is `[Y/n/list]` with Enter = deploy all
- `list` prints asset names then re-prompts
- `nd init --json` output includes the deployed asset list
- Existing builtin-source tests still pass

### References

- Seed task: unaa3u -- `tasks/cli/unaa3u-builtin-source.md`
- Plan: `docs/plans/2026-04-02-002-feat-builtin-source-plan.md` (Unit 6, lines 303-340)
- `cmd/init_cmd.go`, `cmd/helpers.go:58`
- Related: wdisqq (flag/output-mode consistency)
