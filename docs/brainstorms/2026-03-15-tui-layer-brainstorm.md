# TUI Layer brainstorm

| Field | Value |
| --- | --- |
| **Date** | 2026-03-15 |
| **Topic** | tui-layer |

## What We're Building

An interactive TUI for nd that serves as the primary interface when `nd` is invoked with no arguments. The TUI is a pure presentation layer on top of the existing service layer (deploy engine, profile manager, source manager, agent registry).

## Design Decisions

| Decision | Choice | Alternatives considered |
| --- | --- | --- |
| UX model | Dashboard-centric (persistent view, tabs in-place) | Menu-driven flow; hybrid menu+dashboard |
| Information density | Clean and focused (one-line-per-asset, detail on expand) | Dense/power-user (htop-style); minimal/elegant (sparse) |
| Keyboard navigation | Arrow keys + Enter | Vim-style (j/k/h/l); both with vim default |
| Launch flow | Inline picker for scope/agent, becomes header bar | Auto-detect and skip prompt; modal overlay |
| Tab content | Filtered table (same structure, pre-filtered by type) | Split pane (available vs. deployed) |
| Asset detail | Inline expand (row expands in-place) | Side panel; modal overlay |
| Deploy action | Fuzzy finder modal | Multi-select checklist; context-aware from tab |
| Status bar | Context-sensitive help (changes per view) | Static keybinding reference; minimal + help modal |
| Confirmations | Inline in status bar (no modals for single actions) | Confirmation modal; severity-dependent |
| Issue display | Badge in header + tab labels, highlight rows, issues sort to top | Dedicated issues tab; toast notifications |
| Mouse support | None (keyboard only) | Basic mouse (click rows/tabs); full mouse |
| Bubble Tea architecture | Nested models (each component is a Model) | Single model + view funcs; state machine + nested models |

## Why This Approach

The dashboard-centric model was chosen because nd's primary users are developers who already use terminal tools like lazygit and k9s. A persistent dashboard with keyboard shortcuts is faster than navigating menus for repeated operations (deploy, check status, fix issues).

Clean density with inline expand balances information visibility with readability. Power users can see issues at a glance; details are one Enter press away without leaving the view.

Arrow keys + Enter was chosen over vim bindings for accessibility. The target audience uses coding agents — they may be proficient developers but not necessarily vim users.

Nested Bubble Tea models provide clean separation of concerns. Each component is testable in isolation. The root model handles coordination and message routing, keeping component logic focused.

## Open Questions

- Should the picker be re-accessible after launch via a hotkey?
- Should the table support multi-select for bulk operations?
- Should there be a Sources management view in the TUI?
- Bubble Tea v2 vs. v1 API targeting?

## Next Steps

Design doc: `docs/plans/2026-03-15-tui-layer-design.md`
Next: Audit design, then create implementation plan
