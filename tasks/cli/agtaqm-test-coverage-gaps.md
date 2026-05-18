---
title: "Address test coverage gaps from TUI audit"
id: "agtaqm"
status: pending
priority: low
type: chore
tags: ["testing"]
created_at: "2026-04-20"
verify:
  - type: bash
    run: "go build -o nd ."
  - type: bash
    run: "go test ./internal/tui/..."
  - type: bash
    run: "go test -race ./internal/tui/..."
  - type: assert
    check: "Every checklist item below has at least one corresponding new test function in internal/tui/*_test.go, or is explicitly struck out with a one-line rationale because the behavior does not exist in the code."
context:
  - internal/tui/deploy.go
  - internal/tui/deploy_test.go
  - internal/tui/remove.go
  - internal/tui/remove_test.go
  - internal/tui/doctor.go
  - internal/tui/profile.go
  - internal/tui/snapshot.go
  - internal/tui/source.go
  - internal/tui/pin.go
  - internal/tui/settings.go
  - internal/tui/status.go
  - internal/tui/browse.go
  - internal/tui/listview.go
  - internal/tui/scroll.go
  - internal/tui/firstrun_test.go
  - internal/tui/testutil_test.go
  - internal/tui/integration_test.go
  - internal/oplog/oplog.go
  - internal/oplog/writer.go
---

## Address test coverage gaps from TUI audit

### Objective

Close the TUI test-coverage gaps for the `internal/tui` package by adding unit
tests around four behaviors that are currently implemented but largely
untested: form-abort handling (`huh.StateAborted`), OpLog recording on
deploy/remove, dry-run preview, and nil-`DeployEngine` error paths — plus a
handful of list/scroll/empty-state edge cases.

Why: these code paths handle user cancellation, filesystem-mutation safety
(dry-run), and audit logging. Regressions here are silent (no panic, wrong
behavior) so they need explicit test coverage.

Note on provenance: this task originally referenced an audit report at
`.claude/reports/2026-03-23-tui-phases4-6-audit.md` and "32 recommendations".
That report was never committed to this repo and does not exist (verified:
`.claude/reports/` contains only `scope-switching-flow-analysis.md`). The
authoritative spec is therefore the checklist below, which has been corrected
against the actual code. Several originally-listed items asserted behavior the
code does not have; those are struck through with a rationale and are out of
scope (no production code changes are part of this task).

`StateAborted` here means `huh.StateAborted` from `charm.land/huh/v2` — the
state a `*huh.Form` enters when the user presses Esc/Ctrl+C inside the form.
Screens check `form.State == huh.StateAborted` after delegating `Update` to the
form and then navigate back / cancel.

### Patterns to follow

- Existing abort test: `internal/tui/firstrun_test.go:95`
  (`TestFirstRun_StateAborted_Quits`) and `:112`
  (`TestFirstRun_StateAborted_SetsNavigated`) — set
  `screen.<form>.State = huh.StateAborted`, call the relevant update method,
  assert the resulting `tea.Cmd`/state.
- Existing back-nav test: `internal/tui/deploy_test.go:649`
  (`TestDeploy_EscOnPickType_SendsBackMsg`) — drive the screen's update method
  with `tea.KeyPressMsg{Code: tea.KeyEscape}` and assert the returned cmd
  yields a `BackMsg{}` (defined `internal/tui/screens.go:18`).
- Test screen constructor pattern: `newTestDeployScreen` at
  `internal/tui/deploy_test.go:52` — build the screen struct directly with
  `testStyles()` (unstyled, deterministic) and `newMockServices()`.
- Mock services double: `internal/tui/testutil_test.go` (`mockServices`). All
  methods return zero values by default; override behavior via the `*Fn`
  fields, e.g. `svc.isDryRunFn = func() bool { return true }`
  (`internal/tui/integration_test.go:583`), `svc.deployEngineFn`,
  `svc.opLogFn`, `svc.getScopeFn`.
- Dry-run view test: `internal/tui/deploy_test.go:574`
  (`TestDeploy_DryRunView`) sets `ds.dryRun = true` and asserts the rendered
  `View().Content`. Dry-run remove view test:
  `internal/tui/integration_test.go` around line 570 (asserts "Would remove").

### Tasks

#### StateAborted (huh form abort) transition tests

For each screen, set the relevant `*huh.Form` field to `huh.StateAborted`,
call the update method that owns that form, and assert it navigates back /
cancels (per the verified handler line). Mirror
`TestFirstRun_StateAborted_*`.

- [ ] deploy: `updatePickType` (abort handler `internal/tui/deploy.go:321`,
      emits `BackMsg{}`), `updateSelectAssets` (handler
      `internal/tui/deploy.go:422`), and conflict-confirm abort
      (`internal/tui/deploy.go:661` → `cancelConflictResolution`,
      `internal/tui/deploy.go:669`)
- [ ] remove: asset form abort (`internal/tui/remove.go:264`) and confirm
      form abort (`internal/tui/remove.go:309`)
- [ ] doctor: confirm form abort (`internal/tui/doctor.go:242`)
- [ ] profile: menu (`internal/tui/profile.go:238`), switch
      (`internal/tui/profile.go:287`), create
      (`internal/tui/profile.go:352`) form aborts
- [ ] snapshot: menu (`internal/tui/snapshot.go:237`), save
      (`internal/tui/snapshot.go:271`), restore
      (`internal/tui/snapshot.go:338`), confirm
      (`internal/tui/snapshot.go:356`) form aborts
- [ ] source: menu (`internal/tui/source.go:253`), add
      (`internal/tui/source.go:299`), remove
      (`internal/tui/source.go:369`), confirm
      (`internal/tui/source.go:390`) form aborts
- [ ] pin: asset form (`internal/tui/pin.go:187`) and confirm form
      (`internal/tui/pin.go:231`) aborts
- [ ] settings: form (`internal/tui/settings.go:159`) and scope form
      (`internal/tui/settings.go:220`) aborts
- [ ] ~~status screen StateAborted~~ — N/A: `internal/tui/status.go` uses a
      `filterInput` (`internal/tui/listview.go:83`), not a `huh.Form`; it has
      no `StateAborted`. Covered instead by the filter edge-case item below.
- [ ] ~~browse screen StateAborted~~ — N/A: `internal/tui/browse.go` is the
      same `filterInput` pattern, no `huh.Form`, no `StateAborted`. Covered by
      the filter edge-case item below.

#### OpLog recording tests

OpLog is written by exactly two screens (verified — only call sites):
`deployScreen.logOplog` at `internal/tui/deploy.go:679` (called from
`Update` on `deployDoneMsg` at `internal/tui/deploy.go:194` and `:224`, and
from `cancelConflictResolution` at `internal/tui/deploy.go:674`), and the
inline block in `internal/tui/remove.go:161`. `LogEntry`/`OperationType` are
defined in `internal/oplog/oplog.go`; `Writer.Log` JSON-appends to
`<logDir>/operations.log` (`internal/oplog/writer.go:45`).

Test by injecting a real `oplog.NewWriter(t.TempDir())` via
`svc.opLogFn = func() *oplog.Writer { return w }`, driving the deploy/remove
flow to its done state, then reading back `operations.log` and JSON-decoding
the line.

- [ ] deploy: assert `logOplog` writes one `oplog.OpDeploy` entry with
      `Succeeded`/`Failed` counts matching `ds.succeeded`/`ds.failed` and
      `Scope` from `svc.GetScope()` (drive `Update(deployDoneMsg{...})` with
      no conflicts so the `internal/tui/deploy.go:224` path runs)
- [ ] remove: assert the `internal/tui/remove.go:161` block writes one
      `oplog.OpRemove` entry; drive the remove flow to the equivalent
      done/result transition
- [ ] ~~OpLog test for profile deploy/switch~~ — N/A: `internal/tui/profile.go`
      makes no `svc.OpLog()` call (verified: only OpLog sites are deploy.go
      and remove.go). No behavior to test without a production change, which is
      out of scope for this chore.
- [ ] ~~OpLog test for snapshot restore~~ — N/A: `internal/tui/snapshot.go`
      makes no `svc.OpLog()` call. Same rationale as above; out of scope.
- [ ] Verify the written `LogEntry` has a non-zero `Timestamp` (set via
      `time.Now()` at `internal/tui/deploy.go:686`) and the correct
      `Operation`, `Assets` (`asset.Identity` list), `Scope`, `Succeeded`,
      `Failed` fields

#### Dry-run behavior tests

When `svc.IsDryRun()` is true the screen must not invoke the deploy engine.
Deploy guard: `internal/tui/deploy.go:471` (sets `ds.dryRun=true`,
`ds.step=deployResult`, returns before `DeployEngine()` at `:482`). Remove
guard: `internal/tui/remove.go:321`.

- [ ] deploy dry-run: with `svc.isDryRunFn` true and `svc.deployEngineFn` set
      to a fn that fails the test if called, drive the asset-confirm path and
      assert no engine call, `ds.dryRun == true`, `ds.step == deployResult`
- [ ] remove dry-run: same shape against the `internal/tui/remove.go:321`
      guard
- [ ] dry-run view format: assert deploy `View().Content` contains
      `"[DRY RUN]"` and lists asset names (rendered by `buildResultContent` at
      `internal/tui/deploy.go:546`; mirror `TestDeploy_DryRunView`,
      `internal/tui/deploy_test.go:574`); assert remove dry-run view contains
      `"Would remove"` (mirror the existing integration test near
      `internal/tui/integration_test.go:570`)
- [ ] ~~dry-run tests for bulk profile deploy / snapshot restore~~ — verify
      before writing: `profile.go`/`snapshot.go` do not branch on
      `svc.IsDryRun()` (only deploy.go and remove.go do). If no dry-run branch
      exists, strike this item with that note rather than asserting absent
      behavior.

#### Nil DeployEngine safety tests

When `svc.DeployEngine()` returns `(nil, nil)` (the `mockServices` default,
`internal/tui/testutil_test.go:73-78`) the screen must set a graceful
`err` and not panic. Verified guards: `internal/tui/deploy.go:487-490`
("deploy engine not available"), `internal/tui/deploy.go:653-657` (conflict
re-run), `internal/tui/profile.go:309-311` (`profileSwitchedMsg{err:...}`),
`internal/tui/snapshot.go:392-393` (`snapshotRestoredMsg{err:...}`),
`internal/tui/remove.go` (analogous guard — locate the `DeployEngine()` call
and its nil check).

- [ ] deploy: drive the start-deploy path (non-dry-run) with the default
      nil-engine mock; assert `ds.err` is non-nil and the message is
      "deploy engine not available"; assert no panic
- [ ] remove: same for the remove screen's engine guard
- [ ] profile switch: assert `profileSwitchedMsg.err` is non-nil
      ("deploy engine not available") from the cmd built at
      `internal/tui/profile.go:289`/`runSwitch`
- [ ] snapshot restore: assert `snapshotRestoredMsg.err` non-nil from
      `runRestore` (`internal/tui/snapshot.go:377`)
- [ ] assert each error is user-facing (rendered by the screen's error view,
      not a panic/empty string)

#### Symlink strategy tests

Note: the TUI does not create symlinks itself — it builds `deploy.DeployRequest`
values with a `Strategy` field and delegates to `deploy.Engine`. Stale/broken
detection is surfaced via `state.HealthCheck`/`state.HealthStatus`
(`state.HealthBroken`, `state.HealthMissing` — see
`internal/tui/doctor.go:361`). Scope these tests to the TUI's observable
behavior; do not duplicate `internal/deploy`/`internal/state` package tests.

- [ ] deploy request strategy: assert `startDeploy` populates each
      `deploy.DeployRequest.Strategy` from config — default `nd.SymlinkAbsolute`
      (`internal/tui/deploy.go:443`, `nd.SymlinkStrategy` defined
      `internal/nd/symlink.go:4-8`), overridden by
      `SourceManager().Config().SymlinkStrategy` when set
      (`internal/tui/deploy.go:446-448`). Capture requests by injecting a
      `deployEngineFn` whose `DeployBulk` records its argument.
- [ ] deploy request source path: assert each request's `Asset.SourcePath`
      matches the selected asset (built at `internal/tui/deploy.go:459-465`)
- [ ] doctor stale/broken rendering: with mock health-check data containing a
      `state.HealthBroken`/`state.HealthMissing` entry, assert the doctor view
      renders it via `styleGlyphWith` (`internal/tui/doctor.go:357`) /
      `GlyphBroken`
- [ ] ~~symlink conflict handling (existing file at target)~~ — covered at the
      TUI layer by the deploy conflict-resolution flow (`ConflictError` →
      `deployConflictConfirm` step, `internal/tui/deploy.go:198-217`). Add a
      test that a `deployDoneMsg` whose `failed` contains an
      `errors.As`-matchable `*nd.ConflictError` transitions to
      `deployConflictConfirm` (mirror existing conflict tests in
      `deploy_test.go` if present; otherwise this is the new coverage).

#### Miscellaneous coverage gaps

- [ ] filter edge cases for status (`internal/tui/status.go`, filter at
      `:124`/`:162`) and browse (`internal/tui/browse.go:145`/`:323`): empty
      query (all rows shown), no-match query (0 rows, status footer "0/N
      matching" rendered at `internal/tui/status.go:202-203`), and Esc clears
      the filter (`InputActive()` false afterward,
      `internal/tui/status.go:50`, `internal/tui/browse.go:52`)
- [ ] scroll boundary tests for screens using `RenderScrolledLines`
      (`internal/tui/listview.go:13`): verify behavior at offset 0 (scroll-up
      no-op via `listScroll.ScrollUp`, `internal/tui/scroll.go:38`) and at the
      bottom (`listScroll.ScrollDown` clamps, `internal/tui/scroll.go:27`) for
      a representative consumer (e.g. deploy result lines via
      `internal/tui/deploy.go:595`, or doctor `internal/tui/doctor.go:286`)
- [ ] empty-state rendering: assert the relevant screens render the helpers in
      `internal/tui/empty.go` when their data slice is empty —
      `NothingDeployed()` (status/remove), `NoAssets()` (browse),
      `NoProfiles()` (profile), `NoSnapshots()` (snapshot), `NoSources()`
      (source)

### Acceptance criteria

- Every non-struck checklist item above has at least one corresponding `Test*`
  function in the appropriate `internal/tui/*_test.go` file; every struck
  (`~~...~~`) item retains its one-line N/A rationale and adds no production
  code.
- New tests use `testStyles()` + `newMockServices()` and the `mockServices`
  `*Fn` override fields; they do not start a real Bubble Tea program.
- `go build -o nd .` succeeds.
- `go test ./internal/tui/...` passes, including all new tests.
- `go test -race ./internal/tui/...` passes (OpLog tests touch the
  filesystem; use `t.TempDir()`).
- No existing test is deleted or weakened to make new tests pass.
- Measured `internal/tui` coverage does not decrease: record
  `go test -cover ./internal/tui/...` before and after; the "after" statement
  coverage percentage is >= the "before" value.

### References

- Missing audit report: `.claude/reports/2026-03-23-tui-phases4-6-audit.md`
  does **not** exist in the repo (only `scope-switching-flow-analysis.md` is
  present in `.claude/reports/`); the checklist above is authoritative.
- Test utilities / mock: `internal/tui/testutil_test.go`
- Existing abort-test pattern: `internal/tui/firstrun_test.go:95`
- Existing back-nav pattern: `internal/tui/deploy_test.go:649`
- Dry-run view pattern: `internal/tui/deploy_test.go:574`
- OpLog API: `internal/oplog/oplog.go`, `internal/oplog/writer.go`
- Shared list/filter rendering: `internal/tui/listview.go` (+
  `listview_test.go`); scroll helpers: `internal/tui/scroll.go`
- Empty-state helpers: `internal/tui/empty.go`
