---
title: "Merge v0.7.0 release"
id: "ps3zxi"
status: pending
priority: high
type: chore
tags: ["release"]
created_at: "2026-04-20"
---

## Merge v0.7.0 release

### Objective

Review and merge the release-please PR #93 for v0.7.0. Ensure the changelog is accurate, version bumps are correct across all relevant files, and CI passes before merging.

### Tasks

- [ ] Open PR #93 and read the generated changelog: verify all merged PRs since v0.6.x are accounted for and correctly categorized (features, fixes, chores)
- [ ] Check version bump consistency: confirm the version string is updated in `version.go` (or equivalent), `go.mod` (if applicable), and any other files that embed the version
- [ ] Verify CI status: all checks on PR #93 must be green (lint, test, build)
- [ ] Review for any release-please misconfigurations: wrong base branch, missing release type, stale entries
- [ ] If changelog needs edits, push a fixup commit to the release branch before merging
- [ ] Merge PR #93 into `main`
- [ ] Confirm the GitHub Release is created automatically by release-please after merge (tag `v0.7.0`, release notes populated)
- [ ] Verify the release tag exists and points to the correct merge commit

### Acceptance criteria

- PR #93 is merged into `main`
- The `v0.7.0` tag exists on the repository and points to the merge commit
- A GitHub Release for v0.7.0 is published with accurate release notes
- All version strings in the codebase reflect `0.7.0`
- CI passed on the final state of the PR before merge

### References

- PR: https://GitHub.com/armstrongl/nd/pull/93
- release-please documentation: https://GitHub.com/googleapis/release-please
