---
title: "Review and merge agents sync PR"
id: "bnkkfn"
status: pending
priority: medium
type: chore
tags: ["agents"]
created_at: "2026-04-20"
---

## Review and merge agents sync PR

### Objective

Review and merge PR #101 ("[agents] sync frontmatter and index") on branch `agents/sync-fc302ec5c1c17be885adb6c2d3d010d24836a5bc`. This PR synchronizes agent frontmatter metadata and index files to ensure consistency across the multi-agent configuration.

### Tasks

- [ ] Open PR #101 and review the diff: verify that frontmatter changes are consistent across all affected agent files
- [ ] Check that the index file(s) accurately reflect the current set of agents and their metadata
- [ ] Look for unintended changes: files that should not have been modified, formatting regressions, or metadata fields set to incorrect values
- [ ] Verify CI status: all checks on PR #101 must be green
- [ ] If the branch is behind `main`, rebase or merge main into the branch and confirm CI still passes
- [ ] Merge PR #101 into `main`
- [ ] Verify the merge completed cleanly and no agent configuration is broken by running `nd` or relevant agent commands

### Acceptance criteria

- PR #101 is merged into `main`
- All agent frontmatter files have consistent, correct metadata
- The agent index file(s) are in sync with the actual agent definitions
- CI passed on the final state of the PR before merge
- No regressions in agent functionality after merge

### References

- PR: https://GitHub.com/armstrongl/nd/pull/101
- Branch: `agents/sync-fc302ec5c1c17be885adb6c2d3d010d24836a5bc`
- Multi-agent support context: PR #92 (original implementation)
