---
title: "Add embedded help instructions to TUI"
id: "6bije3"
status: pending
priority: medium
type: feature
tags: ["tui", "ux"]
created_at: "2026-04-20"
---

## Add embedded help instructions to TUI

### Objective

Provide contextual, discoverable help within the TUI so users can learn available keybindings and actions without leaving the interface. The existing `HelpBar` (bottom bar with key hints) and `HelpProvider`/`FullHelpProvider` interfaces are a foundation but users need more detailed guidance: a `?`-triggered help overlay per screen, first-run tips, and consistent "? for help" affordance text.

### Tasks

- [ ] Design a `HelpOverlay` component that renders a full-screen (or modal) overlay listing all keybindings and descriptions for the current screen
- [ ] Wire the `?` key in the root TUI model to toggle the help overlay on/off
- [ ] Implement `HelpProvider` on all screens that lack it: deploy, snapshot, profile, pin, doctor, source (audit each screen in `internal/tui/`)
- [ ] Add a `HelpSection` type to group help items under headings (e.g. "Navigation", "Actions", "Filters") for the overlay
- [ ] Add first-run detection: on the first TUI launch (no state file exists), show a brief welcome tip ("Press ? for help at any time") that dismisses on any key
- [ ] Store a `help_seen` flag in the nd state directory so the first-run tip only appears once
- [ ] Add screen-specific contextual hints in the help overlay (e.g. deploy screen: "Use / to filter assets by name", status screen: "Press f to toggle status filters")
- [ ] Write unit tests for the `HelpOverlay` component: toggle on/off, render with basic items, render with sections
- [ ] Write unit tests verifying every screen implements `HelpProvider` or `FullHelpProvider`
- [ ] Verify the help overlay respects terminal width (truncation or wrapping for narrow terminals)

### Acceptance criteria

- Pressing `?` on any TUI screen opens a help overlay showing all available keybindings for that screen
- Pressing `?` or `esc` while the overlay is open closes it
- The help overlay groups keybindings under section headings when a screen provides `HelpSection` items
- On first TUI launch, a dismissible "Press ? for help" tip appears
- The tip does not reappear on subsequent launches
- All TUI screens implement either `HelpProvider` or `FullHelpProvider`
- The bottom help bar continues to function as before (no regression)
- Tests pass: `go test ./internal/tui/... -run TestHelp`

### References

- GitHub issue: https://GitHub.com/armstrongl/nd/issues/75
- Existing help bar: `internal/tui/helpbar.go`
- Help bar tests: `internal/tui/helpbar_test.go`
