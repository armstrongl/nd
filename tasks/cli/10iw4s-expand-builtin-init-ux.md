---
title: "Bring builtin source nd init UX into spec"
id: "10iw4s"
status: pending
priority: high
type: feature
tags: ["core", "onboarding"]
created_at: "2026-05-17"
dependencies: ["unaa3u"]
verify:
  - type: bash
    run: "go build -o nd ."
  - type: bash
    run: "go test ./cmd/..."
  - type: bash
    run: "go test -race ./cmd/..."
  - type: assert
    check: "Interactive `nd init` prints a `[Y/n/list]` prompt; pressing Enter (empty input) deploys all built-in assets; typing `list` prints asset names then re-prompts"
  - type: assert
    check: "`nd init --json` Data includes a `builtin_assets` array of deployed asset objects (name + type), not only the `builtin_deployed` count"
context:
  - "cmd/init_cmd.go"
  - "cmd/helpers.go"
  - "cmd/init_cmd_test.go"
  - "cmd/source.go"
  - "internal/deploy/deploy.go"
  - "internal/state/state.go"
  - "tasks/cli/unaa3u-builtin-source.md"
  - "docs/plans/2026-04-02-002-feat-builtin-source-plan.md"
---

## Bring builtin source nd init UX into spec

### Objective

Make the `nd init` interactive deploy step match the built-in-source spec
(seed task `unaa3u`, plan Unit 6). The built-in source feature is implemented
and green: config type `builtin`, embedded `embed.FS`, version-keyed cache
extraction, source-manager injection, removal guard, and a working `nd init`
deploy path all exist. The remaining gap is **only** in the `nd init` interactive
UX, which diverges from spec in three ways:

1. The deploy prompt is a plain yes/no with **no `list` action** and a
   **default of No**. `cmd/init_cmd.go:195` calls
   `confirm(cmd.InOrStdin(), w, promptMsg, false)`, and `confirm()` in
   `cmd/helpers.go:58` renders `"%s [y/N] "` (line 66) and returns true only
   for `y`/`yes`. The spec (seed `unaa3u` Unit 6, plan
   `docs/plans/2026-04-02-002-feat-builtin-source-plan.md` lines 303-323)
   requires the prompt `Deploy all? [Y/n/list]` where Enter/empty = deploy
   (default Yes), `n` = skip, and `list` = print asset names then re-prompt.
2. `nd init --json` reports only a count. `cmd/init_cmd.go:64-66` sets
   `result["builtin_deployed"] = deployed` (an `int`). The spec requires the
   `--json` output to include the **deployed asset list** (names + types),
   not just a number.
3. The non-terminal fallback deploys instead of declining. When stdin is not
   a TTY, `confirm()` returns an error (`cmd/helpers.go:62-64`), and
   `cmd/init_cmd.go:196-201` treats that error as "default to deploying"
   (`shouldDeploy = true`). The seed's explicit branch is `n: skip`; a
   non-interactive run without `--yes` must **not** silently deploy.

This task is the UX-only slice. The broader cross-command flag/non-TTY
normalization lives in task `wdisqq`
(`tasks/cli/wdisqq-expand-cmd-flag-consistency.md`); item 3 here overlaps its
`cmd/init_cmd.go:193-202` and `:58-71` entries — implement item 3 consistently
with `wdisqq` (non-TTY without `--yes` declines; `--json` surfaces a failed
deploy in the envelope). Do not implement the rest of `wdisqq` here.

### Tasks

- [ ] **Add a `list`-aware 3-way prompt, default Yes.** In
  `cmd/init_cmd.go`, replace the single `confirm(cmd.InOrStdin(), w, promptMsg, false)`
  call at line 195 with a loop that prints `<promptMsg> [Y/n/list]` and reads
  one line:
  - empty input (Enter) **or** `y`/`yes` → deploy all (default Yes)
  - `n`/`no` → skip
  - `l`/`list` → print each asset as `<type>/<name>` (one per line, derived
    from `scanResult.Assets` already scanned at `cmd/init_cmd.go:173`), then
    loop and re-prompt
  - any other input → re-prompt
  `scanResult.Assets[i]` exposes `.Type` (`nd.AssetType`) and `.Name`
  (`string`). Do **not** reuse `confirm()` (it is yes/no only and defaults No);
  add a small helper (e.g. `promptDeployBuiltin`) near `confirm`/`promptChoice`
  in `cmd/helpers.go:55-97`, mirroring `promptChoice`'s `bufio.NewScanner` +
  non-TTY-guard structure. Model the multi-action shape on the 3-way picker in
  `cmd/source.go:181-211`.
- [ ] **Decline on non-TTY without `--yes`.** Update the
  `shouldDeploy := app.Yes || app.JSON` block at `cmd/init_cmd.go:193-202`
  so that when stdin is not a terminal and neither `--yes` nor `--json` is
  set, the result is **skip** (not deploy). The new helper must return the
  same non-TTY error contract as `confirm()`
  (`cmd/helpers.go:62-64`: `"confirmation required but stdin is not a terminal
  (use --yes to skip)"`); on that error set `shouldDeploy = false` (replace the
  current `shouldDeploy = true` at line 198) so the existing skip message at
  `cmd/init_cmd.go:204-209` ("Skipped. Deploy later with 'nd deploy --source
  builtin'") runs. Keep behavior aligned with task `wdisqq`.
- [ ] **Emit the deployed asset list in `--json`.** In the JSON block at
  `cmd/init_cmd.go:58-68`, in addition to (or instead of) `builtin_deployed`,
  add `result["builtin_assets"]` = a slice of `{name, type}` objects. The
  source of truth is `bulkResult.Succeeded` (`[]deploy.DeployResult`,
  `internal/deploy/deploy.go:140-144`); each element's `.Deployment` is a
  `state.Deployment` (`internal/state/state.go:20-32`) with `AssetName`,
  `AssetType`, and `Scope` fields. `deployBuiltinAssets`
  (`cmd/init_cmd.go:160-253`) currently returns only `int` (`deployed`); change
  it to also return the asset list (e.g. add a return value or a small struct)
  and thread it into the JSON map built at `cmd/init_cmd.go:59-67`. Keep the
  existing `builtin_deployed` key so `TestInitCmd_JSON` and
  `TestInitCmd_JSON_IncludesBuiltinDeployCount`
  (`cmd/init_cmd_test.go:189-262`) still pass; add the new key alongside it.
- [ ] **Surface deploy failure in `--json`.** At `cmd/init_cmd.go:58-71` the
  JSON branch returns `printJSON(...)` and never returns `deployErr` (it is
  only returned at line 70 in the non-JSON path). When `deployErr != nil` and
  `app.JSON`, emit a JSON error envelope via `printJSONError`
  (`cmd/helpers.go:36-48`) instead of a `status:"ok"` response. (Shared with
  `wdisqq` item `cmd/init_cmd.go:58-71`.)
- [ ] **Tests.** Extend `cmd/init_cmd_test.go` (existing pattern: build
  `NewRootCmd(app)`, `app.initAgent = testInitAgent(t, tmp)`, set
  `--config <tmp>/.config/nd/config.yaml`, capture out/err buffers). Add:
  - default-Yes-on-Enter: feed `"\n"` to `cmd.InOrStdin()` and assert assets
    deploy (a real TTY is not available in tests; if the non-TTY guard blocks
    interactive simulation, drive the new helper directly with an
    `io.Reader`/`io.Writer` like `confirm`/`promptChoice` are unit-testable,
    or inject a fake `isTerminal`). Prefer a direct helper unit test mirroring
    how `promptChoice` would be tested.
  - `list`-then-deploy: input `"list\n\n"` (or `"list\ny\n"`) prints each
    asset name then deploys.
  - `--json` asset list: after `--yes --json init`, assert
    `resp.Data["builtin_assets"]` is a non-empty array whose elements carry a
    name and type (JSON numbers decode as `float64`; strings as `string`).
  - non-TTY without `--yes`: assert the "Skipped." path (no deploy) and a
    non-zero/clear outcome consistent with `wdisqq`.

### Acceptance criteria

- Interactive `nd init` prints `Deploy <N> built-in asset(s) (...)?  [Y/n/list]`
  (exact count/type fragment from `buildAssetCountParts`/`joinParts`,
  `cmd/init_cmd.go:185-190,255-281`); Enter or `y` deploys all, `n` skips.
- `list` prints every built-in asset as `<type>/<name>`, one per line, then
  re-prompts (no deploy until a Y/n is given).
- `nd init --json` Data contains `builtin_assets`: a non-empty array of
  objects with the asset name and type for each deployed asset (the existing
  `builtin_deployed` count key is preserved).
- A non-interactive (piped stdin) `nd init` **without** `--yes`/`--json` does
  NOT deploy built-ins and prints the existing "Skipped..." hint.
- `nd init --json` reports a failed built-in deploy as a JSON error envelope
  (`status != "ok"`), not `status:"ok"`.
- All existing built-in-source/init tests still pass:
  `go test ./cmd/...` and `go test -race ./cmd/...` are green, including
  `TestInitCmd_WithYes`, `TestInitCmd_WithYes_DeploysBuiltinAssets`,
  `TestInitCmd_JSON`, `TestInitCmd_JSON_IncludesBuiltinDeployCount`,
  `TestInitCmd_AlreadyExists` (`cmd/init_cmd_test.go`).

### References

- GitHub issue: https://GitHub.com/armstrongl/nd/issues/106
- Close this issue when the task is completed.
- Seed task: `unaa3u` — `tasks/cli/unaa3u-builtin-source.md` (Unit 6 defines
  the `[Y/n/list]` prompt, default Yes, `n: skip`, `list`, `--json` asset list).
- Plan: `docs/plans/2026-04-02-002-feat-builtin-source-plan.md`, Unit 6,
  lines 303-340 (approach, patterns, test scenarios).
- Code to change: `cmd/init_cmd.go` — JSON block `:58-71`, deploy/skip
  decision `:193-202`, prompt call `:195`, JSON count `:64-66`,
  `deployBuiltinAssets` `:160-253`.
- Helpers to extend/mirror: `cmd/helpers.go` — `confirm` `:55-73`,
  `promptChoice` `:75-97`, `printJSON` `:21-33`, `printJSONError` `:35-48`.
- Pattern to follow for the multi-action prompt: `cmd/source.go:181-231`
  (3-way `promptChoice` + switch + Cancel branch).
- Data shapes: `internal/deploy/deploy.go:110-144` (`DeployResult`,
  `BulkDeployResult`); `internal/state/state.go:20-32` (`Deployment`:
  `AssetName`, `AssetType`, `Scope`).
- Tests: `cmd/init_cmd_test.go` (helpers `testInitAgent` `:19`,
  `testInitRegistry` `:266`; existing JSON/yes/exists cases `:35-262`).
- Related: `wdisqq` — `tasks/cli/wdisqq-expand-cmd-flag-consistency.md`
  (overlapping non-TTY/`--json` normalization for `cmd/init_cmd.go`; keep
  consistent, do not implement the rest here).
