---
title: "Show deployment indicators in asset lists"
id: "cc0u6u"
status: pending
priority: medium
type: feature
tags: ["tui", "deploy"]
created_at: "2026-04-20"
---

## Show deployment indicators in asset lists

### Objective

Improve asset discoverability in the TUI browse and deploy screens by adding visual indicators for deployment status and recency. Already-deployed assets should show a checkmark badge so users can see at a glance what is live. Recently added or updated assets (based on source scan timestamps or git modification time) should show a "new" badge and sort to the top of lists, making it easier to find freshly available content.

### Tasks

- [ ] Define "deployed" and "new/updated" indicator glyphs in `internal/tui/glyphs.go` (e.g., `GlyphDeployed = "✓"`, `GlyphNew = "●"`) with plain-text fallbacks for non-Unicode terminals
- [ ] In `browseScreen.View()`, replace the current `*` deployed marker with the new styled `GlyphDeployed` badge rendered with `styles.Success`
- [ ] Add a "new" badge (`GlyphNew` rendered with `styles.Primary`) for assets whose source file was modified within a configurable recency window (default: 7 days); read modification time from `asset.Asset.Meta` or fall back to `os.Stat` on the source path
- [ ] In `deployScreen.buildAssetForm()`, add the same deployed/new indicators to multi-select option labels so the deploy picker also shows status
- [ ] Sort assets so that new/updated items appear at the top of the list, followed by undeployed, then already-deployed; preserve alphabetical order within each group
- [ ] Add a `recency_days` config field (optional, default 7) to control the "new" badge threshold
- [ ] Ensure indicators render correctly in both dark and light terminal themes (test with `isDark` toggle)
- [ ] Add unit tests: badge rendering for deployed assets, badge rendering for new assets, sort order verification, edge cases (asset with no mtime, asset exactly at threshold boundary)
- [ ] Add TUI tests: browse screen shows checkmark for deployed assets, browse screen shows new badge for recent assets, deploy screen labels include indicators

### Acceptance criteria

- The browse screen shows a styled checkmark (e.g., green `✓`) next to deployed assets instead of the plain `*` marker
- Assets modified within the last 7 days (default) show a colored `●` "new" badge
- An asset can show both badges simultaneously (deployed + recently updated)
- New/updated assets sort to the top of browse and deploy lists
- The sort is stable: within each group (new, undeployed, deployed), assets remain alphabetically ordered
- `recency_days: 14` in config changes the threshold to 14 days
- Indicators degrade gracefully on terminals without Unicode support
- Existing browse and deploy screen tests continue to pass
- All new indicator and sort tests pass

### References

- https://GitHub.com/armstrongl/nd/issues/81
