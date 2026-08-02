---
title: "Establish version source of truth and release drift check"
id: "8t7acq"
status: completed
priority: high
type: chore
tags: ["release"]
created_at: "2026-05-17"
dependencies: ["ps3zxi"]
context:
  - "internal/version/version.go"
  - "cmd/version.go"
  - ".release-please-manifest.json"
  - "release-please-config.json"
  - ".goreleaser.yaml"
  - "CHANGELOG.md"
  - "internal/builtin/cache.go"
  - "go.mod"
  - ".GitHub/workflows/ci.yml"
  - ".GitHub/workflows/release.yml"
  - ".GitHub/workflows/release-please.yml"
  - ".GitHub/copilot-instructions.md"
  - "tasks/cli/ps3zxi-merge-v070-release.md"
verify:
  - type: bash
    run: "go build -o /tmp/nd-8t7acq . && /tmp/nd-8t7acq version"
  - type: bash
    run: "go test ./... && go build -o /dev/null ."
  - type: assert
    check: "A docs/ or top-level doc names internal/version/version.go:9 as the single product-version source of truth and explains the release-please -> git tag -> goreleaser ldflags -> internal/version flow"
  - type: assert
    check: ".release-please-manifest.json version, the latest git tag (stripped of leading 'v'), and `nd version` output (stripped of leading 'v') all agree, enforced by an automated CI/release check that fails on mismatch"
  - type: assert
    check: "go.mod, .GitHub/workflows/ci.yml, and .GitHub/copilot-instructions.md reference a consistent Go toolchain version (no contradictory pins)"
completed_at: 2026-08-01
---

## Establish version source of truth and release drift check

### Objective

Define and document a single source of truth for the **product version** of `nd`,
and add an automated release-time check that fails when the version embedded in the
binary, the release-please manifest, and the git tag disagree.

Why: the runtime version is injected at build time by goreleaser ldflags into
`internal/version/version.go` (`Version` var, default `"dev"`). But three other
artifacts carry version/release metadata that are hand- or tool-edited
independently and can silently desync from the release tag:

1. `.release-please-manifest.json` — release-please's record of the last released
   version. Bumped by the release-please bot in its release PR.
2. `CHANGELOG.md` — top entry, also bumped by release-please.
3. The Go-toolchain pins in `go.mod`, `.github/workflows/ci.yml`, and
   `.github/copilot-instructions.md` (a *different* "version" — the Go language
   version, not the product version — but currently inconsistent: `go.mod` pins
   `go 1.25.8`, CI pins `"1.25"`, copilot-instructions says `Go 1.25+`).

There is **no automated check** today that the manifest, the git tag, and the
built binary's `nd version` output agree, so a release can ship with a binary
reporting the wrong version.

#### Current state (verified 2026-05-17)

- `internal/version/version.go:9` → `Version = "dev"` (the runtime SOT; overridden
  by ldflags at release build time).
- `.release-please-manifest.json:2` → `".": "0.6.0"`.
- `CHANGELOG.md:3` → `## [0.6.0](...) (2026-04-10)`.
- Latest git tag: `v0.6.0` (run `git tag --sort=-creatordate | head -1`). There is
  **no `v0.7.0` tag yet** — the manifest (`0.6.0`) and latest tag (`v0.6.0`)
  currently *agree* (modulo the `v` prefix).
- The "0.7.0 release target" referenced by the seed task is the unmerged
  release-please PR #93 tracked by dependency **ps3zxi**
  (`tasks/cli/ps3zxi-merge-v070-release.md`). This task (8t7acq) is the
  pattern-expansion of ps3zxi's checklist item *"version bumps correct across all
  relevant files"* — generalize that one-off review into a permanent guardrail.
  Do not hand-bump the manifest to `0.7.0` here; release-please owns that bump
  when ps3zxi's PR merges. This task adds the *drift check*, not the bump.

### Tasks

- [ ] **Document the source of truth.** Add a short section to
  `.github/copilot-instructions.md` (the existing "Architecture" / "Coding
  conventions" doc — see line 26 `version/` entry and line 41 Go-version line) OR
  a new `docs/RELEASING.md`, stating: the product-version SOT is the `Version` var
  in `internal/version/version.go:9` (default `"dev"`); at release time goreleaser
  injects it via `-X github.com/armstrongl/nd/internal/version.Version={{.Version}}`
  (`.goreleaser.yaml:15`); release-please drives the version through
  `.release-please-manifest.json` + `CHANGELOG.md`; the git tag (`vX.Y.Z`) is the
  authoritative release identifier. Document the flow:
  `release-please PR → manifest/CHANGELOG bump → merge → tag vX.Y.Z → release.yml
  / release-please.yml → goreleaser → ldflags → internal/version.Version`.
- [ ] **Verify the runtime SOT is unchanged.** Confirm `internal/version/version.go:9`
  still reads `Version = "dev"` and is consumed only via ldflags. It is read by
  `cmd/version.go:19,24` (`nd version` command, via `version.String()` at
  `internal/version/version.go:15`) and `internal/builtin/cache.go:23`
  (`sanitizeVersion(version.Version)`). No source change expected — assert, don't edit.
- [ ] **Verify goreleaser ldflags path.** In `.goreleaser.yaml`, confirm lines
  **15-17** (NOT 17-19 — original task was stale) still import
  `github.com/armstrongl/nd/internal/version` and set `.Version`, `.Commit`,
  `.Date`. If `internal/version` moves, both these lines and the doc must update.
- [ ] **Verify release-please config.** `release-please-config.json` —
  `release-type: "go"` (line 3), `bump-minor-pre-major: true` (line 5),
  `exclude-paths: ["docs", "site", ".github"]` (line 9). There is no explicit base
  branch in the config; both `.github/workflows/release-please.yml` (line 5) and
  `.github/workflows/release.yml` (lines 5-6, tag-push `v*`) target `main`.
  Confirm these are consistent with the intended release flow; no change expected
  unless a misconfiguration is found.
- [ ] **Confirm CHANGELOG/manifest agreement.** `.release-please-manifest.json:2`
  (`0.6.0`) must equal `CHANGELOG.md:3`'s top entry version (`0.6.0`) and the
  latest git tag with `v` stripped (`v0.6.0` → `0.6.0`). They currently agree. Do
  **not** bump to `0.7.0` (owned by release-please via ps3zxi/PR #93).
- [ ] **Note dev-build cache aliasing (no fix required).** In
  `internal/builtin/cache.go`, `CacheDir()` (line 18) joins
  `sanitizeVersion(version.Version)` (line 23; `sanitizeVersion` defined line 120).
  Every locally-built binary has `version.Version == "dev"`, so all dev builds
  share one builtin cache dir `~/.cache/nd/builtin/dev/`. `EnsureExtracted`
  (line 39) skips extraction when the dir already exists, so a stale `dev/` cache
  is reused across rebuilds. Document this caveat in the releasing doc (and the
  manual workaround: `rm -rf ~/.cache/nd/builtin/dev` after embedding changes).
  This is an observation to record, not a bug to fix in this task.
- [ ] **Reconcile the Go-toolchain version strings.** Three locations disagree:
  `go.mod:3` (`go 1.25.8`), `.github/workflows/ci.yml:30,42,64`
  (`go-version: "1.25"` in the lint, test, and build jobs), and
  `.github/copilot-instructions.md:41` (`Go 1.25+`). Also
  `.github/workflows/release.yml:21` and `.github/workflows/release-please.yml:30`
  pin `go-version: "1.25"`. Decide a convention (recommended: `setup-go` reads
  the version from `go.mod` via `go-version-file: go.mod`, eliminating the
  hardcoded `"1.25"` pins) and apply it consistently across all five workflow
  references; ensure the prose in copilot-instructions stays accurate.
- [ ] **Add the release-drift check.** Add a CI/release step that fails when
  the manifest version, the git tag, and the built binary's `nd version` disagree.
  Recommended placement: a step in `.github/workflows/release.yml` (triggers on
  `v*` tag push; the tag is `$GITHUB_REF_NAME`) and/or
  `.github/workflows/release-please.yml`, run *after* `actions/setup-go` and
  *before* the goreleaser step. The check must:
  - read `jq -r '.["."]' .release-please-manifest.json` → manifest version
    (e.g. `0.6.0`),
  - derive the tag from `${GITHUB_REF_NAME#v}` (strip leading `v`),
  - build with the same ldflags goreleaser uses and run
    `nd version`, stripping the leading `v` from the printed version
    (`version.String()` formats as `nd version v0.6.0 (commit: ..., built: ...)`),
  - normalize all three (strip `v`, trim whitespace) and `exit 1` with a clear
    message if any differ.
  Prefer a small committed script (e.g. `scripts/check-version-drift.sh`) invoked
  by the workflow so it is locally runnable, over inline YAML shell.

### Acceptance criteria

- A committed doc (e.g. `docs/RELEASING.md` or a section in
  `.github/copilot-instructions.md`) names `internal/version/version.go:9` as the
  single product-version source of truth and describes the
  release-please → tag → goreleaser-ldflags → `internal/version` flow, including
  the dev-build cache-aliasing caveat.
- An automated check exists (workflow step + ideally a runnable script) that
  exits non-zero when `.release-please-manifest.json` version, the git tag (sans
  `v`), and `nd version` output (sans `v`) do not all match. Verify by
  temporarily setting the manifest to a wrong value locally and confirming the
  script fails (then revert).
- `go.mod`, `.github/workflows/ci.yml` (lines 30/42/64),
  `.github/workflows/release.yml`, `.github/workflows/release-please.yml`, and
  `.github/copilot-instructions.md` no longer carry contradictory Go-toolchain
  pins.
- No `internal/` source files are modified for this task (version.go,
  cache.go etc. are asserted unchanged); only docs, workflows, config, and a new
  script.
- `go build -o /dev/null .` and `go test ./...` still pass.

### References

- GitHub issue: https://GitHub.com/armstrongl/nd/issues/116
- Close this issue when the task is completed.
- Seed / blocking dependency: **ps3zxi** —
  `tasks/cli/ps3zxi-merge-v070-release.md` (review/merge release-please PR #93 for
  v0.7.0; this task generalizes its "version bumps correct" checklist item into a
  permanent guardrail). 8t7acq is blocked on ps3zxi but the drift-check work does
  not require the v0.7.0 tag to exist.
- Runtime SOT: `internal/version/version.go:9` (`Version = "dev"`),
  `version.String()` at `internal/version/version.go:15`.
- Consumers: `cmd/version.go:19,24` (`nd version`),
  `internal/builtin/cache.go:23` (`sanitizeVersion(version.Version)`,
  `sanitizeVersion` at line 120, `CacheDir` at line 18, `EnsureExtracted` at
  line 39).
- ldflags injection: `.goreleaser.yaml:15-17`.
- Release plumbing: `.github/workflows/release-please.yml` (release-please bot,
  goreleaser at line 32), `.github/workflows/release.yml` (tag-push `v*`,
  goreleaser at line 22), `release-please-config.json`,
  `.release-please-manifest.json:2`, `CHANGELOG.md:3`.
- Go-toolchain pins: `go.mod:3` (`go 1.25.8`),
  `.github/workflows/ci.yml:30,42,64`, `.github/workflows/release.yml:21`,
  `.github/workflows/release-please.yml:30`,
  `.github/copilot-instructions.md:41` (`Go 1.25+`).
- release-please docs: https://GitHub.com/googleapis/release-please
</content>
</invoke>
