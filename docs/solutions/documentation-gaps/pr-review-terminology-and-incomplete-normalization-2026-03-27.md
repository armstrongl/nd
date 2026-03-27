---
title: "PR review catches terminology inaccuracies and incomplete style normalization"
date: 2026-03-27
category: documentation-gaps
module: Documentation
problem_type: documentation_gap
component: documentation
symptoms:
  - "Docs say 'file' when describing skill deployment, but skills deploy as directory symlinks"
  - "Mental model intro says 'connects two directories' but nd creates per-asset symlinks"
  - "'What the agent sees' section contradicts itself: 'next session' then 'no restart needed'"
  - "README still has Title Case headings and bash fences after a PR claiming full style normalization"
root_cause: inadequate_documentation
resolution_type: documentation_update
severity: low
tags: [pr-review, terminology, style-normalization, copilot, documentation-accuracy]
---

# PR review catches terminology inaccuracies and incomplete style normalization

## Problem

Automated PR review (GitHub Copilot) on a docs PR (#9) flagged 5 issues: terminology that said "file" when skills are directories, a misleading mental model description, contradictory session/restart claims, and incomplete README style normalization that the PR description claimed was complete.

## Symptoms

- Copilot flagged "original file" in intro when the example deploys a skill (directory asset)
- "connects two directories with a symlink" implies a single dir-to-dir link, not per-asset symlinks
- "available in your next session" followed by "no restart needed" is contradictory
- README still had `## What It Does`, `### Go Install`, `### Build from Source`, `## Quick Start`, `## Configuration`, `## Contributing` in Title Case and 4 `bash` code fences after the normalization commit

## What Didn't Work

- The original style normalization commit only updated the Documentation section links and list separators in the README, leaving all other headings and code fences untouched. The PR description implied full normalization was done.

## Solution

1. Changed "original file" to "original source" in `how-nd-works.md` and "source file" to "source" in `getting-started.md`
2. Rephrased mental model intro from "nd connects two directories with a symlink" to "nd wires each deployed asset from your source into the agent's config directory"
3. Changed "available in your next Claude Code session" to "typically available in your next Claude Code session" and "no restart or configuration needed" to "no additional nd configuration is needed"
4. Normalized all remaining README headings to sentence case and all 4 `bash` fences to `shell`

## Why This Works

The terminology fixes ensure docs match what actually happens on disk (skills deploy as directory symlinks, not file symlinks). The hedging in "What the agent sees" avoids promising behavior that varies by agent version. The README normalization completes what the previous commit started.

## Prevention

- When claiming a cross-cutting style change in a PR description ("normalize all docs"), grep-verify every target file after the change, not just the ones you edited intentionally.
- Use terms like "source" or "asset" instead of "file" when describing nd deployments, since both files and directories are valid deploy targets.
- When describing agent behavior after deploy, hedge with "typically" since session pickup timing depends on the agent, not nd.

## Related Issues

- PR #9: https://github.com/armstrongl/nd/pull/9
- Previous solution: `docs/solutions/build-errors/golangci-lint-staticcheck-qf1012-sa5011-ci-failure-2026-03-27.md` (same PR, different problem)
