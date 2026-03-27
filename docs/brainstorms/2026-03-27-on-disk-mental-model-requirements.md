---
date: 2026-03-27
topic: on-disk-mental-model
---

# On-Disk Mental Model: How nd Works Page

## Problem Frame

nd's entire mechanism is symlinks, but no documentation shows what a symlink deployment actually looks like on the filesystem. A user who runs `nd deploy skills/greeting` sees a success message but has no reference for what changed on disk, what the resulting directory tree looks like, or why editing the source file updates the deployed asset instantly. This gap makes it impossible for users to reason about broken symlinks, scope differences, or why moving a source directory causes problems.

The current docs describe symlinks in prose ("creating a symlink in your agent's config directory") without ever showing one. ARCHITECTURE.md has a code-flow diagram but no filesystem diagram. The getting-started guide goes from `nd deploy` to `nd status` with no explanation of what happened in between.

## Requirements

- R1. Create a new `docs/guide/how-nd-works.md` standalone concepts page
- R2. Include a "Mental Model" section showing the three-layer relationship: source directory, nd (the connector), and agent config directory. Use an annotated ASCII diagram or tree view
- R3. Include a "What deploys look like" section with annotated before/after filesystem views (tree or `ls -la` style output) showing symlink arrows pointing from agent config back to source. Cover at least: a directory asset (skills/) and a file asset (rules/)
- R4. Include a "Global vs Project Scope" section showing how the same deploy command targets different directories depending on scope. Show the concrete paths for both (`~/.claude/` vs `.claude/` in project root)
- R5. Include a "Context Files" section explaining that context files are the exception — project-scope context deploys to `./CLAUDE.md` (project root), not `.claude/CLAUDE.md`. Mention `*.local.md` scope restriction
- R6. Include a brief "Absolute vs Relative Symlinks" section showing the difference with a concrete example of each
- R7. Include a brief "What the agent sees" section explaining whether Claude Code picks up deployed assets immediately, whether sessions need restarting, and that hooks/output-styles need manual `settings.json` registration (with a pointer to where/how)
- R8. Add a brief "What just happened" callout in `docs/guide/getting-started.md` after step 5 (deploy), linking to the new page. 2-3 sentences max
- R9. Link the new page from the README's Documentation section and from the getting-started Next Steps

## Success Criteria

- A reader who has never used symlinks can look at the page and understand what `nd deploy` does to their filesystem
- The before/after views use concrete paths (not abstract descriptions) that match nd's actual behavior
- The page answers "is this copying files?" within the first 10 seconds of reading
- Getting-started readers hit the callout at exactly the right moment (after their first deploy)

## Scope Boundaries

- This page explains what nd puts on disk. It does not document full nd workflows (deploy, remove, sync, profiles)
- The "What the agent sees" section is brief (3-5 sentences + one code snippet for settings.json). Full agent behavior documentation is a separate effort
- No Mermaid or image files — use ASCII/text diagrams that render in any markdown viewer
- The author will add personal voice, asides, and "heads up" callouts during implementation. The requirements define structure, not tone

## Key Decisions

- **Standalone page over inline**: The full mental model needs more space than a callout can provide. A standalone page becomes a durable reference that other docs can link to
- **Include brief agent-side section**: Users shouldn't have to look in two places to understand the full deploy → agent pickup flow
- **Getting-started gets a cross-link callout**: A brief "what just happened" note after step 5 bridges the gap without slowing the tutorial flow
- **ASCII diagrams only**: No build tooling, no images, no Mermaid. Works everywhere markdown renders

## Dependencies / Assumptions

- The filesystem paths used in examples must match nd's actual deploy behavior (verify against `internal/deploy/` engine)
- Context file deployment rules must match the implementation in the deploy engine (project scope deploys to project root, not `.claude/`)
- The "What the agent sees" section assumes Claude Code reads from `~/.claude/` at session start — verify this is accurate before documenting

## Outstanding Questions

### Deferred to Planning
- [Affects R3][Needs research] What does `nd status` output actually look like for the annotated terminal example? Run the command to capture real output
- [Affects R7][Needs research] Does Claude Code pick up new skills mid-session or only at session start? Verify before documenting the agent-side behavior
- [Affects R7][Needs research] What is the exact `settings.json` snippet needed for hooks and output-styles manual registration?

## Next Steps

-> /ce:plan for structured implementation planning
