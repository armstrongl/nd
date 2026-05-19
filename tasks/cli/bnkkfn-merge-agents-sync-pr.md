---
title: "Review and merge agents sync PR"
id: "bnkkfn"
status: pending
priority: medium
type: chore
tags: ["agents"]
created_at: "2026-04-20"
context:
  - ".GitHub/workflows/docs-sync.yml"
  - "scripts/agents/build-index.py"
  - "AGENTS.md"
  - ".gitignore"
verify:
  - type: bash
    run: "gh pr view 101 --json state,baseRefName,files -q '\"state=\" + .state + \" base=\" + .baseRefName + \" files=\" + ([.files[].path] | join(\",\"))'"
  - type: bash
    run: "git ls-files | grep -c '\\.pyc$' | grep -qx 0"
  - type: assert
    check: "PR #101 is closed (not merged), because its only diff is the stale scripts/agents/__pycache__/frontmatter.cpython-311.pyc artifact that PR #99 already deleted and gitignored, and the real frontmatter/index sync was already merged via PR #102"
---

## Review and merge agents sync PR

### Objective

Resolve the open automated PR #101 ("[agents] sync frontmatter and index",
branch `agents/sync-fc302ec5c1c17be885adb6c2d3d010d24836a5bc`). The task title
says "merge", but investigation shows this PR must be **closed, not merged** —
see Current state. The goal is to leave no stale automated sync PR open and
confirm `main` already has the intended agent frontmatter/index sync.

Background: `.github/workflows/docs-sync.yml` runs on pushes touching
`docs/guide/**`. It regenerates `AGENTS.md` via
`python scripts/agents/build-index.py`, then opens an auto PR titled
"[agents] sync frontmatter and index" on a branch `agents/sync-<sha>` with base
`${{ github.ref_name }}` (the branch that was pushed — NOT necessarily `main`).
Many such PRs exist; most are superseded and closed.

### Current state (verified 2026-05-17 via `gh`/`git`)

- **PR #101 is OPEN** but should NOT be merged:
  - Base branch is `improve-docs`, **not** `main`. `improve-docs` was merged
    via PR #100 (commit `dd6935b`) and the branch is gone.
  - Its only file change is `scripts/agents/__pycache__/frontmatter.cpython-311.pyc`
    with **0 additions / 0 deletions** — a stale Python bytecode artifact, no
    meaningful content.
  - That `.pyc` was **already deleted** and `__pycache__/` added to `.gitignore`
    by PR #99 ("fix(scripts): recursive doc discovery and artifact hygiene",
    merged 2026-04-11, commit `f877b15`). It is no longer tracked on `main`
    (`git ls-files | grep -c '\.pyc$'` → 0).
  - Created by the `github-actions` bot.
- **The real sync already landed**: PR #102 (identical title, base `main`,
  branch `agents/sync-dd6935b...`) added 8 lines to `AGENTS.md` and was
  **MERGED** 2026-04-11 as commit `cfa89e5`
  ("chore: sync frontmatter and regenerate AGENTS.md index (#102)").
- Therefore PR #101 is obsolete and superseded; merging it would only try to
  reintroduce a gitignored bytecode artifact onto a deleted base branch.

### Tasks

- [ ] Re-confirm state has not changed:
      `gh pr view 101 --json state,baseRefName,mergedAt,files,additions,deletions`
      — expect `state=OPEN`, `baseRefName=improve-docs`, single `.pyc` file,
      0/0 lines. If reality differs (e.g. content is now substantive), STOP and
      re-evaluate against the diff (`gh pr diff 101`).
- [ ] Confirm the intended sync is already on `main`:
      `git log --oneline | grep cfa89e5` shows PR #102's merge commit, and
      `git ls-files | grep -c '\.pyc$'` returns `0`.
- [ ] Close PR #101 with an explanatory comment, do NOT merge:
      `gh pr close 101 --comment "Superseded by #102 (merged to main as cfa89e5). Sole diff is the scripts/agents/__pycache__/frontmatter.cpython-311.pyc artifact that #99 already deleted and gitignored; base branch improve-docs is also gone."`
- [ ] Delete the stale remote branch if it still exists:
      `git push origin --delete agents/sync-fc302ec5c1c17be885adb6c2d3d010d24836a5bc`
      (skip if already gone).
- [ ] Sanity-check the build is unaffected (no source changes expected from
      this task): `go build -o nd .` succeeds.
- [ ] (Optional, prevents recurrence) Verify
      `.github/workflows/docs-sync.yml:84` still sets `base: ${{ github.ref_name }}`;
      note that running the sync workflow off non-`main` branches is what
      produced this orphaned PR. Filing a follow-up to pin `base: main` is out
      of scope here — only note it if not already tracked.

### Acceptance criteria

- PR #101 is in state `CLOSED` (verify: `gh pr view 101 --json state` →
  `state=CLOSED`), and explicitly **not** `MERGED`.
- A comment on PR #101 records why it was closed (superseded by #102, stale
  `.pyc`-only diff, dead base branch).
- The remote branch `agents/sync-fc302ec5c1c17be885adb6c2d3d010d24836a5bc` no
  longer exists (`git ls-remote --heads origin agents/sync-fc302ec5c1c17be885adb6c2d3d010d24836a5bc`
  returns nothing).
- No `.pyc` files are tracked in git (`git ls-files | grep -c '\.pyc$'` → `0`)
  and `main` already contains commit `cfa89e5` (PR #102's sync).
- `go build -o nd .` still succeeds; no source files modified by this task.

### References

- GitHub issue: https://GitHub.com/armstrongl/nd/issues/129
- Close this issue when the task is completed.
- PR #101 (close, do not merge): https://GitHub.com/armstrongl/nd/pull/101
  — branch `agents/sync-fc302ec5c1c17be885adb6c2d3d010d24836a5bc`, base
  `improve-docs`.
- PR #102 (the real sync, already merged): https://GitHub.com/armstrongl/nd/pull/102
  — merge commit `cfa89e5`.
- PR #99 (deleted the `.pyc`, added `__pycache__/` to `.gitignore`):
  https://GitHub.com/armstrongl/nd/pull/99 — merge commit `f877b15`.
- PR #100 (merged the now-dead `improve-docs` base branch): commit `dd6935b`.
- Sync mechanism: `.github/workflows/docs-sync.yml`,
  `scripts/agents/build-index.py`.
- Multi-agent support original implementation: PR #92 (commit `2d9e66c`).
