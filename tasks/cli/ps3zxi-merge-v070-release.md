---
title: "Merge v0.7.0 release"
id: "ps3zxi"
status: pending
priority: high
type: chore
tags: ["release"]
created_at: "2026-04-20"
context:
  - "release-please-config.json"
  - ".release-please-manifest.json"
  - "CHANGELOG.md"
  - ".GitHub/workflows/release-please.yml"
  - ".GitHub/workflows/release.yml"
  - ".GitHub/workflows/ci.yml"
  - ".goreleaser.yaml"
  - "internal/version/version.go"
verify:
  - type: bash
    run: "gh pr view 93 --json state -q .state"  # expect: MERGED
  - type: bash
    run: "git fetch --tags origin && git tag --list v0.7.0"  # expect: v0.7.0
  - type: bash
    run: "gh release view v0.7.0 --json tagName,isDraft -q '.tagName + \" draft=\" + (.isDraft|tostring)'"  # expect: v0.7.0 draft=false
  - type: bash
    run: "git show origin/main:.release-please-manifest.json"  # expect: {\".\": \"0.7.0\"}
  - type: bash
    run: "go build -o /dev/null ."  # main builds clean
  - type: assert
    check: "CHANGELOG.md on main has a top [0.7.0] section listing the features/fixes merged since v0.6.0"
---

## Merge v0.7.0 release

### Objective

Cut the `v0.7.0` release by reviewing and merging the open release-please PR
**#93** (`chore(main): release 0.7.0`) into `main`. Merging the PR triggers the
automated release pipeline (tag `v0.7.0`, GitHub Release, goreleaser binaries +
Homebrew tap update). The why: `v0.7.0` bundles user-facing work merged since
`v0.6.0` (multi-agent/Copilot CLI support, taskmd setup, AFDocs CI, doc-script
fixes) that is currently unreleased and unavailable to `brew`/binary users.

### Current state (verified 2026-05-17)

- PR **#93** is **OPEN**, base `main`, head branch
  `release-please--branches--main`, `mergeable: MERGEABLE`,
  `mergeStateStatus: CLEAN`. URL: <https://github.com/armstrongl/nd/pull/93>
- Latest published release/tag is **v0.6.0**. No `v0.7.0` tag or release exists
  yet. The task is **not** done.
- PR #93 changes exactly two files: `CHANGELOG.md` (prepends a `[0.7.0]`
  section) and `.release-please-manifest.json` (`0.6.0` -> `0.7.0`). On `main`
  the manifest still reads `{ ".": "0.6.0" }`.
- This repo uses release-please `release-type: "go"` with **no version file**.
  There is no `version.go` constant or `go.mod` version to bump. The runtime
  version lives in `internal/version/version.go` as `Version = "dev"` and is
  injected at build time via goreleaser ldflags
  (`-X github.com/armstrongl/nd/internal/version.Version={{.Version}}`, see
  `.goreleaser.yaml` line 15). Do **not** hand-edit any version string.

### How the release pipeline works (read before merging)

1. `release-please-config.json` — release-type `go`, single package `.`,
   `bump-minor-pre-major: true`, `exclude-paths: ["docs","site",".github"]`,
   changelog at `CHANGELOG.md`.
2. `.github/workflows/release-please.yml` — on push to `main`, runs
   `googleapis/release-please-action@v4`. Merging PR #93 sets
   `release_created == true`, which then checks out, sets up Go 1.25, and runs
   `goreleaser release --clean` (creates the `v0.7.0` tag + GitHub Release,
   builds darwin/Linux amd64/arm64 binaries, updates `armstrongl/homebrew-tap`).
3. `.github/workflows/release.yml` — also fires on the `v0.7.0` tag push as a
   secondary goreleaser release path.
4. CI nuance: `.github/workflows/ci.yml` (lint-markdown / lint / test / build)
   runs on `pull_request` to `main`, so it *should* run on PR #93. The only
   check currently in PR #93's status rollup is `submit-pypi` (GitHub
   Dependency Submission, passing). Confirm the `ci.yml` jobs via
   `gh pr checks 93` before merging; if absent, validate locally instead
   (commands below).

### Tasks

- [ ] Read the PR #93 body / changelog:
  `gh pr view 93 --json title,body -q .body`. Confirm the `[0.7.0]`
  section accounts for and correctly categorizes the conventional commits
  merged since `v0.6.0` (Features: #92 Copilot CLI, #105 taskmd, #103
  AFDocs CI; Bug Fixes: #99 doc-script hygiene). Cross-check against
  `git log --oneline v0.6.0..origin/main` for missed feat/fix commits.
- [ ] Verify the version bump: PR diff must set
  `.release-please-manifest.json` to `{ ".": "0.7.0" }`. No other version
  edits are expected (no `version.go`/`go.mod` change — see Current state).
  `gh pr diff 93`.
- [ ] Verify CI: `gh pr checks 93`. All present checks must be green. If the
  `ci.yml` jobs (Lint Markdown, Lint, Test, Build) are not listed for the
  release-please branch, validate locally on `main`:
  `go build -o /dev/null .`, `go test ./... -race`,
  `goreleaser check`, `rumdl check .`.
- [ ] Check for release-please misconfig: base must be `main`, head
  `release-please--branches--main`, single `[0.7.0]` section, no stale or
  duplicated entries, version `0.7.0` (minor bump from `0.6.0`, correct
  under `bump-minor-pre-major`).
- [ ] If the changelog needs edits, push a fixup commit to
  `release-please--branches--main` (do not rebase/squash the
  release-please commit) before merging.
- [ ] Merge PR #93 into `main`:
  `gh pr merge 93 --merge` (use a merge commit, not squash — release-please
  tracks the release commit).
- [ ] Wait for the `Release Please` workflow run on `main` to finish:
  `gh run list --workflow=release-please.yml --branch main --limit 1` then
  `gh run watch <run-id>`. It must create the tag and run goreleaser.
- [ ] Confirm the GitHub Release `v0.7.0` exists and is published (not draft)
  with release notes: `gh release view v0.7.0`.
- [ ] Confirm the `v0.7.0` tag exists and points to the PR #93 merge commit:
  `git fetch --tags origin && git rev-list -n1 v0.7.0` vs the merge SHA.
- [ ] Confirm `.release-please-manifest.json` on `main` now reads
  `{ ".": "0.7.0" }`: `git show origin/main:.release-please-manifest.json`.

### Acceptance criteria

- PR #93 state is `MERGED` into `main`
  (`gh pr view 93 --json state -q .state` -> `MERGED`).
- Tag `v0.7.0` exists locally after fetch and resolves to the PR #93 merge
  commit (`git tag --list v0.7.0` non-empty).
- A non-draft GitHub Release `v0.7.0` is published with release notes
  (`gh release view v0.7.0` succeeds, `isDraft=false`).
- `.release-please-manifest.json` on `main` equals `{ ".": "0.7.0" }`.
- `go build -o /dev/null .` succeeds on `main` post-merge.
- `CHANGELOG.md` on `main` has a top `[0.7.0]` section reflecting the merged
  feat/fix commits since `v0.6.0`.

### References

- GitHub issue: https://GitHub.com/armstrongl/nd/issues/120
- Close this issue when the task is completed.
- PR #93: <https://github.com/armstrongl/nd/pull/93>
- release-please config: `release-please-config.json`,
  `.release-please-manifest.json`
- Release workflows: `.github/workflows/release-please.yml`,
  `.github/workflows/release.yml`; build config `.goreleaser.yaml`
- release-please docs: <https://github.com/googleapis/release-please>
