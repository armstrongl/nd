---
title: "Show deployment indicators in asset lists"
id: "cc0u6u"
status: pending
priority: medium
type: feature
tags: ["tui", "deploy"]
created_at: "2026-04-20"
verify:
  - type: bash
    run: "go build -o nd ."
  - type: bash
    run: "go test ./internal/tui/... ./internal/config/..."
  - type: bash
    run: "go test -race ./internal/tui/..."
  - type: assert
    check: "Browse screen renders a styled checkmark for deployed assets and a styled 'new' badge for assets whose source file mtime is within recency_days; new/updated assets sort to the top, alphabetical within each group; new tests cover deployed/new/sort/boundary cases and all existing tui+config tests pass."
context:
  - internal/tui/theme.go
  - internal/tui/browse.go
  - internal/tui/deploy.go
  - internal/tui/scroll.go
  - internal/tui/profile.go
  - internal/tui/pin.go
  - internal/tui/doctor.go
  - internal/tui/header.go
  - internal/tui/services.go
  - internal/asset/asset.go
  - internal/asset/context.go
  - internal/asset/identity.go
  - internal/config/config.go
  - cmd/app.go
---

## Show deployment indicators in asset lists

### Objective

Improve asset discoverability in the TUI by adding visual indicators for deployment
status and recency, and (merged from cancelled task `7l7r5d`) centralize every
ad-hoc TUI marker into one styled glyph module.

Concretely:

- Already-deployed assets in the **browse** screen and **deploy** picker show a
  styled checkmark instead of the current plain `*` marker.
- Assets whose source file was modified within a configurable window (default
  7 days) show a styled "new" badge and sort to the top of the list.
- All scattered hardcoded markers (`*`, `!`, `↑`/`↓`, `[DRY RUN]`, `[pinned]`,
  `(active)`) are replaced by shared constants + render helpers so the same
  concept renders identically on every screen.

Why: GitHub issue armstrongl/nd#81 — "Show deployment indicators and list recent
assets first" — users cannot tell at a glance what is live or what is new.

### Ground truth (verified against the repo — read before coding)

- **There is no `internal/tui/glyphs.go`.** Glyph constants currently live,
  unstyled, in `internal/tui/theme.go:51-60`:
  `GlyphOK="ok"`, `GlyphBroken="!!"`, `GlyphDrifted="??"`, `GlyphOrphan="--"`,
  `GlyphMissing="xx"`, `GlyphDot="·"`, `GlyphArrow="->"`. These are plain ASCII
  (intentionally "readable without color"). The new module must follow the same
  ASCII-safe convention; do not break the no-color readability contract.
- The browse deployed marker is `browse.go:249-252`: `marker := " "` then
  `if b.deployed[a.String()] { marker = "*" }`, printed in the
  `fmt.Fprintf(&buf, "%s%s  %-12s  %-24s  %s%s\n", ...)` at `browse.go:262-263`.
- The deployed set is built in `browse.go:92-100`: keyed by
  `dep.Identity().String()` from `svc.StateStore().Load()`. `a.String()` for an
  asset resolves to `asset.Identity.String()` (`internal/asset/identity.go:17-23`),
  format `source:subdir/name`. The keys match — use `b.deployed[a.String()]`.
- **`asset.Asset` has NO modification-time field.** `asset.Asset`
  (`internal/asset/asset.go:4-11`) embeds `Identity` and has
  `SourcePath string`, `IsDir bool`, `GroupDir`, `ContextFile`, `Meta *ContextMeta`.
  `ContextMeta` (`internal/asset/context.go:14-20`) is `Description`, `Tags`,
  `TargetLanguage`, `TargetProject`, `TargetAgent` — **no timestamp**. The ONLY
  source of recency is `os.Stat(a.SourcePath).ModTime()`. `SourcePath` is
  populated by the scanner (`internal/sourcemanager/scanner.go:124` and `:255`)
  and may be empty for some synthetic entries — treat empty path / stat error as
  "not new" (no badge), do not fail rendering.
- The deploy picker builds option labels in `deploy.go:378-397`
  (`buildAssetForm`): `label := fmt.Sprintf("%s  %s", a.Name, a.SourceID)`
  (with description appended when `a.Meta != nil && a.Meta.Description != ""`),
  option value `assetKey(a)` (`deploy.go:714-722`, `"sourceID:type/name"`).
  `ds.assets` here is already the **undeployed** set (`filterUndeployed`,
  `deploy.go:725`), so a deployed badge there is only meaningful if the picker is
  ever shown the full set; for now apply the **new** badge here and keep the
  deployed badge logic available via the shared helper.
- `recency_days` is **not** in config and is **not** reachable from the TUI.
  `config.Config` (`internal/config/config.go:8-16`) has no such field. The
  `Services` interface (`internal/tui/services.go:16-45`) exposes no `Config()`.
  Config is reachable from the App via `SourceManager().Config()`
  (`*config.Config`; see `cmd/app.go:47-58`, `cmd/app.go:69`). You must:
  add the field to `config.Config`, surface it through the TUI (either add a
  small accessor to the `Services` interface + `cmd/app.go` + the
  `mockServices` test double in `internal/tui/testutil_test.go:16-43`, or read
  it once via `svc.SourceManager().Config()` inside `browseScreen.Init`).
  Document the chosen approach in the worklog.
- Existing TUI test pattern to mirror: `internal/tui/browse_test.go:59-90`
  (`TestBrowseScreen_ViewWithAssets` / `_ViewNoAssets`) — construct assets,
  `newMockServices()` with `scanIndexFn`, drive `s.Update(browseLoadedMsg{...})`,
  assert on `s.View().Content`. Test double lives in
  `internal/tui/testutil_test.go`.

### Tasks

**Glyph module (the file every item below references):**

- [ ] Create `internal/tui/glyphs.go` (package `tui`). Move/reuse the constant
      block from `theme.go:51-60` OR add new constants there — pick one home and
      keep it consistent; do not duplicate symbols. Export at minimum:
      `GlyphDeployed`, `GlyphNew`, `GlyphActive`, `GlyphPinned`,
      `GlyphScrollUp`, `GlyphScrollDown`, `GlyphDryRun`, plus a warning glyph
      constant (replacing the bare `"!"`). Keep values ASCII-safe (match the
      existing "readable without color" contract in `theme.go:51`). Add styled
      render helpers, e.g. `func (s Styles) Deployed() string`,
      `func (s Styles) NewBadge() string`, mirroring how `theme.go`/screens pair
      a glyph with a `lipgloss.Style` (see `Styles` in `theme.go:28-49`:
      `Success`, `Primary`, `Warning`, `Subtle`).

**Deployment + recency indicators (issue #81 core):**

- [ ] `internal/config/config.go:8-16` — add field
      `RecencyDays int \`yaml:"recency_days,omitempty" json:"recency_days,omitempty"\``
      to `config.Config`. Default behavior: `0` (unset) means use 7 days. Mirror
      the `omitempty` style of the adjacent `ContextTypes` field. Confirm no
      strict-unknown-key validation rejects it (`internal/config/validation.go`
      has no field allowlist today — verify it stays that way).
- [ ] Surface `RecencyDays` to the TUI. Either (a) add `Config() *config.Config`
      to the `Services` interface (`internal/tui/services.go:16-45`), implement
      it on `*App` in `cmd/app.go` via `SourceManager().Config()`, and add the
      function field + method to `mockServices`
      (`internal/tui/testutil_test.go:16-43`); or (b) read it inside
      `browseScreen.Init` (`browse.go:73-104`) via
      `svc.SourceManager().Config()` and store the threshold on `browseScreen`.
      Resolve effective window: `days := cfg.RecencyDays; if days <= 0 { days = 7 }`.
- [ ] `browse.go:249-252` — replace the `marker = "*"` literal with the styled
      deployed glyph from the new module (use `styles.Success`). Keep the column
      alignment in the `fmt.Fprintf` at `browse.go:262-263` correct (the marker
      occupies a fixed slot; a styled string changes width — render into a
      fixed-width cell or measure with `lipgloss.Width`).
- [ ] `browse.go` View loop (`browse.go:242-264`) — compute per-asset "new":
      `isNew(a, window) bool` that does `os.Stat(a.SourcePath)`, returns false on
      empty path / error, else `time.Since(info.ModTime()) <= window`. Render
      `GlyphNew` styled with `styles.Primary` when new. An asset may show BOTH
      the deployed and new badges simultaneously.
- [ ] Add stable sort in browse (apply to `b.assets` after load in the
      `browseLoadedMsg` case `browse.go:119-128`, or inside `visibleAssets`
      `browse.go:317-328`): order = new/updated first, then undeployed, then
      deployed; alphabetical by `a.Name` within each group. Use
      `sort.SliceStable`. Cursor/scroll math (`clampCursor`, `scroll`) must stay
      correct after reordering.
- [ ] `deploy.go:378-397` (`buildAssetForm`) — append the styled new badge to the
      option `label` using the shared helper (deployed badge logic available but
      `ds.assets` is undeployed-only, so it is a no-op there today). Keep the
      existing description-append branch (`deploy.go:382-384`) intact.

**Merged scope from cancelled task `7l7r5d` (verified file:line):**
`7l7r5d` ("Centralize and style TUI status indicators",
`tasks/cli/7l7r5d-expand-tui-indicators.md`, status `cancelled`,
`cancelled_at: 2026-05-17`) was folded here because both tasks own
`internal/tui/glyphs.go` — splitting guarantees a merge conflict. Do not action
`7l7r5d` separately; do its work here:

- [ ] `internal/tui/profile.go:406` — replace `marker = "*"` (inside the
      `for _, p := range s.profiles` loop, active-profile marker) with the styled
      `GlyphActive`.
- [ ] `internal/tui/profile.go:259` — replace the `(active)` string literal
      (`label += " (active)"`) with the same active badge used at `:406` so the
      picker and the list agree.
- [ ] `internal/tui/pin.go:158` — replace `[pinned]` literal
      (`label += " [pinned]"`) with the styled `GlyphPinned` badge.
- [ ] `internal/tui/doctor.go:346` — replace `d.styles.Warning.Render("!")` with
      the shared warning glyph constant rendered via `styles.Warning`.
- [ ] Replace the three repeated `"[DRY RUN]"` literals with one
      `GlyphDryRun`/constant: `deploy.go:548`
      (`ds.styles.Warning.Render("[DRY RUN]")`), `remove.go:423`
      (`m.styles.Warning.Render("[DRY RUN]")`), and `header.go:28`
      (`left = "  [DRY RUN] " + left[2:]`).
- [ ] Replace the scroll-arrow literals with `GlyphScrollUp`/`GlyphScrollDown`.
      The single render chokepoint is `scrollIndicatorLine` (`scroll.go:93-95`);
      the literals are passed in by callers `browse.go:239`, `browse.go:267`,
      `listview.go:32`, `listview.go:36` (the task's previous `scroll.go:32,36`
      reference was stale — those lines are inside `ScrollDown`). Prefer changing
      the callers to pass the new constants (or have `scrollIndicatorLine` take a
      direction enum); keep the rendered output ("↑/↓ N more") visually stable.

**Tests:**

- [ ] Unit tests for the glyph module: each helper returns the expected
      glyph+style; `isNew` true/false for recent vs old mtime; edge cases:
      empty `SourcePath`, `os.Stat` error, mtime exactly at the window boundary;
      `RecencyDays=0` ⇒ 7-day default; `RecencyDays=14` ⇒ 14-day window.
- [ ] Sort test: assets in new / undeployed / deployed groups end up in that
      order, alphabetical within each group; sort is stable.
- [ ] TUI tests mirroring `browse_test.go:59-90`: browse `View().Content`
      contains the deployed glyph for a deployed asset; contains the new glyph
      for an asset with a recent `SourcePath` mtime (write a temp file, set
      `SourcePath`); deploy picker option labels include the new badge.
- [ ] Run the full `internal/tui` and `internal/config` suites — all existing
      tests must still pass (none assert on the literal `*`/`[DRY RUN]` strings;
      grep first and update any that do).

### Acceptance criteria

- Browse shows a styled checkmark (e.g. green via `styles.Success`) next to
  deployed assets instead of the plain `*` from `browse.go:251`.
- Assets whose `SourcePath` file mtime is within the effective window (default 7)
  show a styled `styles.Primary` "new" badge; an asset can show deployed + new
  simultaneously.
- New/updated assets sort to the top of the browse list; sort is stable and
  alphabetical (by `a.Name`) within each of new / undeployed / deployed groups.
- `recency_days: 14` in config widens the window to 14 days;
  `recency_days` unset behaves as 7 days; `taskmd verify cc0u6u` config build
  passes (`go build -o nd .`).
- The deploy picker option labels (`deploy.go:378-397`) include the new badge.
- Indicators are ASCII-safe / readable without color (same contract as
  `theme.go:51`).
- `internal/tui/glyphs.go` exists and is the single home for
  `GlyphDeployed/New/Active/Pinned/ScrollUp/ScrollDown/DryRun` + a warning glyph;
  the ad-hoc markers at `profile.go:259`, `profile.go:406`, `pin.go:158`,
  `doctor.go:346`, `deploy.go:548`, `remove.go:423`, `header.go:28`,
  `browse.go:239/267`, `listview.go:32/36` all use the shared constants.
- `go build -o nd .`, `go test ./internal/tui/... ./internal/config/...`, and
  `go test -race ./internal/tui/...` all pass; new indicator/sort tests pass;
  no existing test regresses.

### References

- GitHub issue: https://GitHub.com/armstrongl/nd/issues/81
- Merged-from (cancelled): `tasks/cli/7l7r5d-expand-tui-indicators.md`
- Browse marker + deployed-set: `internal/tui/browse.go:92-100,242-264`
- Glyph constants today: `internal/tui/theme.go:51-60`; styles: `theme.go:28-49`
- Asset model (no mtime field): `internal/asset/asset.go:4-11`,
  `internal/asset/context.go:14-20`; identity key: `internal/asset/identity.go:17-23`
- `SourcePath` population: `internal/sourcemanager/scanner.go:124,255`
- Config struct: `internal/config/config.go:8-16`; App config access:
  `cmd/app.go:47-58,69`; Services interface: `internal/tui/services.go:16-45`
- Test pattern + mock: `internal/tui/browse_test.go:59-90`,
  `internal/tui/testutil_test.go:16-43`
