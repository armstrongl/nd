---
title: "Centralize and style TUI status indicators"
id: "7l7r5d"
status: cancelled
priority: medium
type: feature
tags: ["tui", "deploy"]
created_at: "2026-05-17"
dependencies: []
cancelled_at: 2026-05-17
---

## Centralize and style TUI status indicators

> **⚠ Superseded by cc0u6u** — folded into `tasks/cli/cc0u6u-deployment-indicators.md` (both own `internal/tui/glyphs.go`). This task is cancelled; do not action it.

### Objective

Pattern expansion of seed cc0u6u. Deployment and status markers are rendered with scattered hardcoded strings (`"*"`, `"!"`, `"↑"`/`"↓"`, `"[DRY RUN]"`, `" [pinned]"`, `" (active)"`) instead of centralized styled glyphs, and the same concept is represented differently across screens. There is no `internal/tui/glyphs.go`; constants live unstyled in `internal/tui/theme.go:51-60`. Create a glyphs module and replace the ad-hoc markers.

### Tasks

- [ ] Create `internal/tui/glyphs.go` with `GlyphDeployed`, `GlyphNew`, `GlyphActive`, `GlyphPinned`, `GlyphScrollUp/Down`, `GlyphDryRun` and styled render helpers with plain-text fallback for non-Unicode terminals
- [ ] `internal/tui/browse.go:251` -- replace `marker = "*"` with styled `GlyphDeployed` (`styles.Success`) (the seed)
- [ ] `internal/tui/profile.go:406` -- replace `marker = "*"` with styled `GlyphActive`
- [ ] `internal/tui/profile.go:258-259` -- replace `" (active)"` literal with the same active badge as :406
- [ ] `internal/tui/pin.go:158-159` -- replace `" [pinned]"` literal with `GlyphPinned` badge
- [ ] `internal/tui/doctor.go:346` -- replace `Render("!")` with a warning glyph constant
- [ ] `internal/tui/deploy.go:548`, `remove.go:423`, `header.go:28` -- replace repeated `"[DRY RUN]"` literals with one constant
- [ ] `internal/tui/scroll.go:32,36` and `browse.go:239,267` -- replace `"↑"`/`"↓"` literals with scroll glyph constants
- [ ] Unit tests: badge rendering deployed/new/active/pinned in dark and light themes; non-Unicode fallback
- [ ] `go test ./internal/tui/...`

### Acceptance criteria

- Browse shows a styled deployed badge instead of plain `*`
- Active profile / pinned / dry-run / scroll indicators use shared constants, consistent across screens
- Indicators degrade gracefully without Unicode
- Existing browse/profile/deploy tests pass

### References

- Seed task: cc0u6u -- `tasks/cli/cc0u6u-deployment-indicators.md`
- `internal/tui/theme.go:51-60`, `browse.go`, `profile.go`, `pin.go`, `scroll.go`
