---
title: "TUI minor cleanups for dead code and unsurfaced sync error"
id: "qrmcc2"
status: pending
priority: low
type: chore
tags: ["tui"]
created_at: "2026-05-17"
context:
  - "internal/tui/header.go"
  - "internal/tui/header_test.go"
  - "internal/tui/source.go"
  - "internal/tui/source_test.go"
  - "internal/tui/profile.go"
  - "internal/tui/snapshot.go"
verify:
  - type: bash
    run: "go build -o nd ."
  - type: bash
    run: "go test ./internal/tui/..."
  - type: bash
    run: "go test -race ./internal/tui/..."
  - type: assert
    check: "header.go View() no longer declares a misleadingly-named `leftStyled` variable that holds an unstyled string; the left segment is emitted via `left` directly (or genuinely styled)."
  - type: assert
    check: "A source sync failure renders through the same error-display code path/styling as a source load failure (the `s.err != nil && s.step == sourceDone` branch in View()), not a separate doneMsg-only path."
---

## TUI minor cleanups

### Objective

Two low-severity quality issues found during a codebase sweep (net-new, no
upstream seed/spec — "no seed" just means there is no pre-existing review
finding or design doc driving this; treat this task as the source of truth).
Both live in `internal/tui`:

1. **Dead/misleading code** in `internal/tui/header.go`: a variable named
   `leftStyled` is assigned the *raw, unstyled* string `left`. The `*Styled`
   suffix implies a styled value but none is applied, which misleads future
   readers into thinking the left header segment is styled when it is not.
2. **Inconsistent error surfacing** in `internal/tui/source.go`: source
   *sync* failures are routed into `s.doneMsg` (via `formatSyncResult`) and
   rendered by the generic `sourceDone` view, whereas source *load* failures
   are routed into `s.err` and rendered by the dedicated red error branch in
   `View()`. The two paths differ in styling ("Danger"-rendered vs. plain),
   trailing hint ("Press esc to go back." vs. "Press enter to return."), and
   layout. The goal is one consistent error presentation for source errors.

Why: reduce reader confusion (header) and make the Sources screen's failure
UX consistent with every other screen so users get predictable error display.

### Background: how source errors are displayed today (verified)

`internal/tui/source.go`:

- `sourceLoadedMsg` handler sets `s.err = msg.err` and `s.step = sourceDone`
  (`Update`, ~line 116-121).
- `View()` has a dedicated error branch:
  `if s.err != nil && s.step == sourceDone { ... s.styles.Danger.Render(s.err.Error()) ... "Press esc to go back." }`
  (~line 168-172). This branch is **reachable** for load errors (so the
  original "the branch is dead" framing is inaccurate — the real problem is
  that sync errors never reach it).
- `startSync()` (~line 427-450) wraps failures into
  `sourceSyncedMsg{errors: [...]}` at the `SourceManager()`-error guard
  (~line 434) and the nil-manager guard (~line 437), plus per-source
  `SyncSource` failures (~line 442-446).
- The `sourceSyncedMsg` handler (~line 143-147) sets `s.step = sourceDone`
  and `s.doneMsg = s.formatSyncResult(msg)` — it never touches `s.err`.
- `formatSyncResult` (~line 507-519) builds a string mixing the success
  count and `Danger`-rendered per-error lines into `doneMsg`.
- The `sourceDone` view branch (~line 195-196) renders `s.doneMsg` followed
  by "Press enter to return." — a different presentation than the `s.err`
  branch.

Existing tests that constrain the fix (`internal/tui/source_test.go`):

- `TestSourceScreen_LoadError` (~line 79): asserts a load error string is in
  `View().Content`.
- `TestSourceScreen_SyncDone_Success` (~line 130): asserts the synced count
  is shown.
- `TestSourceScreen_SyncDone_PartialError` (~line 140): asserts a partial
  sync error string ("git pull failed") is shown — i.e. on partial success
  the synced count *and* the error must both remain visible. Do not collapse
  partial-success into a bare `s.err`.
- `TestSourceScreen_RefreshHeaderAfterSync` (~line 150): asserts the sync
  handler still returns a `RefreshHeaderMsg` cmd. Preserve that cmd.

Analogous "correct" pattern to mirror (same struct shape: `s.err`,
`s.doneMsg`, `xxxDone` step, identical `View()` error branch):

- `internal/tui/profile.go`: `s.err = msg.err` then `s.step = profileDone`
  (~line 114-115); error branch `if s.err != nil && s.step == profileDone`
  (~line 164-166).
- `internal/tui/snapshot.go`: same shape (~line 111-112, 161-163).

### Tasks

- [ ] **header.go — remove misleading dead variable.** In
  `func (h Header) View(s Styles, width int) string` (`internal/tui/header.go`),
  delete the line `leftStyled := left` (currently ~line 34) and replace the
  use of `leftStyled` in the final return (currently ~line 48,
  `return leftStyled + strings.Repeat(...) + rightStyled`) with `left`
  directly. Rationale: the left segment is intentionally unstyled (the
  `[DRY RUN]` branch at ~line 28-30 also leaves it unstyled), so the minimal
  correct change is to drop the misleading alias rather than invent a style.
  Do not change `rightStyled` or the gap/width math.

- [ ] **source.go — surface sync errors via the same path as load errors.**
  Make a failed sync render through the existing
  `s.err != nil && s.step == sourceDone` branch in `View()` so it matches
  load-error and other-screen presentation. Recommended approach (mirrors
  `profile.go`/`snapshot.go`): in the `sourceSyncedMsg` case of `Update`
  (~line 143-147), when `len(msg.errors) > 0`, set
  `s.err = <combined sync error>` in addition to (or instead of) `s.doneMsg`,
  then `s.step = sourceDone`. On full success (`len(msg.errors) == 0`) leave
  `s.err` nil and keep the existing `doneMsg` success path. Constraints:
  - Partial success (`msg.synced > 0` with errors present) must still show
    the synced count AND the error text (keep `TestSourceScreen_SyncDone_PartialError`
    green). If you fold everything into `s.err`, include the synced count in
    the error string; or keep `formatSyncResult` for the message body and
    additionally set a non-nil `s.err` so the consistent branch is taken —
    pick one and keep the rendered text covering both pieces.
  - Still return the `RefreshHeaderMsg` cmd from the handler (keep
    `TestSourceScreen_RefreshHeaderAfterSync` green).
  - Reset `s.err` to nil when leaving `sourceDone` (e.g. in `buildMenu`,
    ~line 203, where the screen is re-entered via `updateDone` pressing
    enter) so a prior sync error does not leak into a later view. Verify
    whether `buildMenu` already clears it; if not, clear it there.

- [ ] **Tests.** Add/extend `internal/tui/source_test.go` so a sync failure
  is asserted to render through the same branch/styling as a load failure
  (e.g. a `TestSourceScreen_SyncDone_AllErrors` that feeds
  `sourceSyncedMsg{synced: 0, errors: []error{...}}` and asserts the error
  text appears and uses the consistent error view). Keep all existing
  `internal/tui/source_test.go` and `internal/tui/header_test.go` tests
  passing unchanged (do not edit `header_test.go` — the header change is
  presentation-neutral; `TestHeaderViewDryRun` etc. assert on substrings of
  the returned string, which are unaffected by removing the alias).

### Acceptance criteria

- `internal/tui/header.go` `View()` no longer declares `leftStyled`; the
  final return uses `left` (or a genuinely styled value). No behavior change
  in rendered header output.
- A source sync failure is displayed through the same error-display branch
  and styling as a source load failure (the
  `s.err != nil && s.step == sourceDone` branch). Partial-success sync still
  shows both the synced count and the error message.
- The `sourceSyncedMsg` handler still emits a `RefreshHeaderMsg` cmd.
- `go build -o nd .` succeeds.
- `go test ./internal/tui/...` and `go test -race ./internal/tui/...` pass,
  including all pre-existing tests in `source_test.go` and `header_test.go`,
  plus a new test covering the consistent sync-error display path.

### References

- GitHub issue: https://GitHub.com/armstrongl/nd/issues/114
- Close this issue when the task is completed.
- `internal/tui/header.go` — `View()` ~line 26-49; dead alias ~line 34, use ~line 48.
- `internal/tui/source.go` — `sourceSyncedMsg` handler ~line 143-147;
  `View()` error branch ~line 168-172; `sourceDone` view ~line 195-196;
  `startSync` ~line 427-450 (error wrap ~line 434, 437, 442-446);
  `formatSyncResult` ~line 507-519; `buildMenu` ~line 203.
- Pattern to mirror: `internal/tui/profile.go` ~line 114-115 & 164-166;
  `internal/tui/snapshot.go` ~line 111-112 & 161-163.
- Tests to keep green: `internal/tui/source_test.go`
  `TestSourceScreen_LoadError` (~79), `TestSourceScreen_SyncDone_Success`
  (~130), `TestSourceScreen_SyncDone_PartialError` (~140),
  `TestSourceScreen_RefreshHeaderAfterSync` (~150);
  `internal/tui/header_test.go` `TestHeaderViewDryRun` (~87) and siblings.
- Line numbers are approximate (~) — re-read the files; do not trust them blindly.
