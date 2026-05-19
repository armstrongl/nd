---
title: "Ship built-in source with nd"
id: "unaa3u"
status: pending
priority: high
type: feature
tags: ["core", "onboarding"]
created_at: "2026-04-20"
context:
  - "docs/plans/2026-04-02-002-feat-builtin-source-plan.md"
  - "internal/nd/source_type.go"
  - "internal/config/validation.go"
  - "internal/builtin/embed.go"
  - "internal/builtin/cache.go"
  - "internal/builtin/cache_test.go"
  - "internal/sourcemanager/sourcemanager.go"
  - "internal/sourcemanager/config.go"
  - "internal/sourcemanager/register.go"
  - "internal/sourcemanager/register_test.go"
  - "cmd/init_cmd.go"
  - "cmd/init_cmd_test.go"
  - "cmd/source.go"
  - "tests/integration/source_test.go"
  - "tests/integration/helpers_test.go"
verify:
  - type: bash
    run: "go build -o nd ."
  - type: bash
    run: "go test ./internal/builtin/... ./internal/sourcemanager/... ./internal/config/... ./internal/nd/... ./cmd/..."
  - type: bash
    run: "go test -race ./internal/builtin/... ./internal/sourcemanager/..."
  - type: bash
    run: "go test ./tests/integration/ -run Builtin -v"
  - type: assert
    check: "An end-to-end integration test in tests/integration/ exercises `nd init --yes` then asserts builtin assets are deployed and `nd source list` includes a source with type builtin that `nd source remove builtin` refuses to delete"
---

## Ship built-in source with nd

### Objective

Ship a first-party "builtin" source embedded in the nd binary so a fresh
`nd init` deploys nd-specific skills, commands, and an expert agent out of the
box. This removes the cold-start problem where new users see an empty asset
list and must register an external source before nd is useful.

**IMPORTANT — current state (verified 2026-05-17):** Units 1–6 below are
**already implemented and all unit tests pass** (`go build -o nd .` and
`go test ./internal/builtin/... ./internal/sourcemanager/... ./internal/config/... ./internal/nd/... ./cmd/...`
are green). The plan in `docs/plans/2026-04-02-002-feat-builtin-source-plan.md`
has been executed. The remaining work for this task is to (a) verify each unit
against the acceptance criteria, (b) reconcile two intentional divergences from
the original plan, and (c) close the one genuine coverage gap: there is **no
end-to-end integration test** exercising the builtin source via the compiled
binary (the plan's "Integration coverage" item, plan line 348, was deferred).

Do not re-implement completed units. Treat the unchecked boxes below as a
verification checklist; only the items in **Unit 7** require new code.

### Tasks

- [x] **Unit 1: `SourceBuiltin` type and reserved ID** — DONE
  - `internal/nd/source_type.go:9` defines `SourceBuiltin SourceType = "builtin"`;
    line 14 defines `const BuiltinSourceID = "builtin"`.
  - `internal/config/validation.go:90` accepts `nd.SourceBuiltin` in the
    `switch s.Type` block.
  - Verify: `go test ./internal/nd/... ./internal/config/...` passes.

- [x] **Unit 2: Embedded source tree via `go:embed`** — DONE
  - `internal/builtin/embed.go:9-10` declares `//go:embed source` /
    `var FS embed.FS`. Note the embedded layout is **nested**, not flat:
    `internal/builtin/source/skills/<name>/SKILL.md`,
    `internal/builtin/source/commands/<name>.md`,
    `internal/builtin/source/agents/<name>.md`. Six assets exist:
    skills `nd-create-source`, `nd-scaffold-asset`, `nd-deploy-workflow`;
    commands `nd-quickstart`, `nd-audit`; agent `nd-expert`. Content is
    production-quality (the plan only required placeholders).
  - Verify: `go build ./internal/builtin/...` succeeds.

- [x] **Unit 3: Cache extraction** — DONE
  - `internal/builtin/cache.go`: `CacheDir()` (line 18) returns
    `$XDG_CACHE_HOME/nd/builtin/<sanitized-version>/` (fallback `~/.cache`,
    see `xdgCacheHome()` line 108); `EnsureExtracted()` (line 39) does
    temp-dir-then-`os.Rename` atomic extraction; `Path()` (line 29) is the
    entry point. `cacheBaseDir` (line 14) is the test override hook. Version
    key comes from `internal/version/version.go:9` (`Version = "dev"` by
    default, set via ldflags).
  - Tests live in `internal/builtin/cache_test.go`.
  - Verify: `go test -race ./internal/builtin/...` passes.

- [x] **Unit 4: Inject builtin source into the config pipeline** — DONE
  - `internal/sourcemanager/sourcemanager.go:97-110` defines
    `appendBuiltinSource(cfg)`, called from `New()` at line 42 (after load +
    merge). It appends a `SourceEntry{ID: builtin, Type: builtin, Path:
    cachePath, Alias: "nd"}` as the **last** (lowest-priority) entry;
    extraction failure is a stderr warning, not fatal (lines 99-102).
  - `internal/sourcemanager/config.go:127-148`: `WriteConfig()` calls
    `stripBuiltinSource()` so the builtin entry is never persisted to YAML.
  - Verify: `go test ./internal/sourcemanager/...` passes; assert a config
    round-tripped through `WriteConfig` contains no `id: builtin` line.

- [x] **Unit 5: Guard against removal and ID conflicts** — DONE
  - `internal/sourcemanager/register.go:101-103`: `Remove("builtin")` returns
    `"the builtin source cannot be removed"`.
  - ID-conflict prevention: `register.go:67` and `register.go:141` seed
    `existingIDs[nd.BuiltinSourceID] = true` before `GenerateSourceID()`
    (line 31), so a user dir named `builtin` gets `builtin-2`.
    `insertBeforeBuiltin()` (line 16) keeps the builtin entry last when adding
    sources.
  - `cmd/source.go` has **no** builtin-specific code — it relies on the
    `Remove()` guard; `nd source list` shows builtin via `sm.Sources()`
    naturally. This is correct per plan line 288.
  - Verify: `go test ./internal/sourcemanager/... ./cmd/...` passes;
    `register_test.go:169 TestRemove` and surrounding tests assert the
    "1 user + 1 builtin" source count.

- [x] **Unit 6: `nd init` deploy-with-opt-out** — DONE, with a divergence
  - `cmd/init_cmd.go`: `deployBuiltinAssets()` (line 160) extracts the cache
    (`builtin.Path()`), scans it (`sourcemanager.ScanSource`, line 173),
    builds a count summary, and prompts. `--yes`/`--json` deploy silently
    (line 193); `n` prints `"Skipped. Deploy later with 'nd deploy --source
    builtin'"` (line 206). Deploy uses `deploy.New(...).DeployBulk(...)`
    (lines 226-239) to global scope. JSON output adds `builtin_deployed`
    (init_cmd.go:64-66).
  - **DIVERGENCE 1 (decide & record):** the prompt is binary `[y/N]` via
    `confirm()` (`cmd/helpers.go:58`, called at `init_cmd.go:195`). The
    original task/plan text described a three-way `[Y/n/list]` prompt with a
    `list` option that re-prompts. The plan explicitly deferred this UX
    decision (plan line 80). **Action:** confirm with project conventions
    whether the binary prompt is acceptable as-is. If the `list` option is
    required, replace `confirm()` with `promptChoice()`
    (`cmd/helpers.go:77`) offering `Deploy all` / `Skip` / `List assets`,
    where `List assets` prints `scanResult.Assets` names then re-prompts.
    Either way, update this task and the plan so docs match the code; do not
    leave the discrepancy undocumented.
  - **DIVERGENCE 2 (cosmetic):** the original text claimed embedded files are
    flat (`nd-create-source/SKILL.md`). Reality is the nested
    `source/skills/...` layout. No code change needed — the description above
    is now correct.
  - Verify: `go test ./cmd/...` passes; `init_cmd_test.go` already covers
    `--yes` deploy (`TestInitCmd_WithYes_DeploysBuiltinAssets`, line 62),
    JSON count (`TestInitCmd_JSON_IncludesBuiltinDeployCount`, line 223), and
    registry use (`TestInitCmd_DeployBuiltinAssets_UsesRegistry`, line 496).

- [ ] **Unit 7: Add the missing end-to-end integration test (NEW WORK)**
  - Gap: no test in `tests/integration/` exercises the builtin source through
    the compiled binary. The plan (line 348) required: `nd init` → assets
    deployed → `nd list` shows them → `nd source list` shows builtin →
    `nd source remove builtin` errors.
  - Mirror the existing pattern in `tests/integration/source_test.go`. Tests
    build the binary once in `TestMain` (`helpers_test.go:15-33`) and invoke
    it with `runND(t, args...)` (`helpers_test.go:41`), which returns
    `runResult{Stdout, Stderr, ExitCode}`. Use an isolated `HOME`/config dir
    and an isolated `XDG_CACHE_HOME` (the binary subprocess does not see the
    Go-level `cacheBaseDir` override, so set the env var on the `exec.Command`
    or pass `--config` to a temp dir; follow how `source_test.go` isolates
    state).
  - Add a test (e.g. `TestBuiltinSource_InitDeploysAndCannotBeRemoved` in a
    new `tests/integration/builtin_test.go`, package `integration`) that:
    1. Runs `nd init --yes` (and `--json` variant) in a clean env; asserts
       exit 0 and that builtin assets were deployed (check `builtin_deployed`
       in JSON, or `nd status` / deployed symlinks for the six asset names).
    2. Runs `nd source list` (plain and `--json`); asserts an entry with
       `id`/type `builtin` is present.
    3. Runs `nd source remove builtin`; asserts non-zero exit and stderr
       contains `"builtin source cannot be removed"`.
    4. Runs `nd list`; asserts the six builtin asset names appear.
  - Verify: `go test ./tests/integration/ -run Builtin -v` passes; full
    `go test ./...` stays green.

### Acceptance criteria

- `go build -o nd .` succeeds and `go test ./internal/builtin/...
  ./internal/sourcemanager/... ./internal/config/... ./internal/nd/...
  ./cmd/...` passes with no regressions.
- `nd init --yes` in a fresh environment deploys the six builtin assets to
  global scope; `nd init --json` includes `builtin_deployed` in its output.
- `nd source list` includes an entry with type `builtin`; it never appears in
  the persisted `config.yaml` (verified via `WriteConfig` round-trip / by
  inspecting the written file).
- `nd source remove builtin` exits non-zero with a message matching
  `"builtin source cannot be removed"`.
- A user source named/aliased `builtin` is auto-suffixed (`builtin-2`); user
  sources always precede the builtin entry in `Sources` (lowest priority).
- The cache path is keyed by `version.Version`; deleting the cache dir causes
  transparent re-extraction on the next scan.
- **Unit 7:** a new integration test in `tests/integration/` covers the
  init→deploy→list→remove-guard flow via the compiled binary and passes under
  `go test ./tests/integration/ -run Builtin -v`.
- Divergence 1 is resolved: either the binary `[y/N]` prompt is confirmed
  acceptable (task + plan updated to say so) or the `[Y/n/list]` prompt is
  implemented with a passing `cmd/init_cmd_test.go` case for the `list` path.

### References

- GitHub issue: https://GitHub.com/armstrongl/nd/issues/126
- Close this issue when the task is completed.
- Plan (executed): `docs/plans/2026-04-02-002-feat-builtin-source-plan.md`
  (requirements trace R1–R8 at lines 20-27; integration-coverage gap at
  line 348; deferred prompt-UX decision at line 80).
- Embedded assets: `internal/builtin/source/{skills,commands,agents}/`.
- Integration test harness: `tests/integration/helpers_test.go`
  (`TestMain` line 15, `runND` line 41); pattern to mirror:
  `tests/integration/source_test.go`.
- Prompt helpers: `cmd/helpers.go` — `confirm()` line 58,
  `promptChoice()` line 77.
- Go embed docs: <https://pkg.go.dev/embed>
- XDG Base Directory Spec:
  <https://specifications.freedesktop.org/basedir-spec/latest/>
