---
title: "Establish version source of truth and release drift check"
id: "8t7acq"
status: pending
priority: high
type: chore
tags: ["release"]
created_at: "2026-05-17"
dependencies: ["ps3zxi"]
---

## Establish version source of truth and release drift check

### Objective

Pattern expansion of seed ps3zxi ("version bumps correct across all relevant files"). The runtime version source of truth is `internal/version/version.go:9` injected via goreleaser ldflags, but the release-please manifest, CHANGELOG, and Go-toolchain pins are independently hand-editable and can desync from the release tag. The manifest is currently `0.6.0` while the active release target is v0.7.0. Document the single source of truth and add a release-time consistency check.

### Tasks

- [ ] `internal/version/version.go:9` -- document this as the runtime source of truth; verify default stays `"dev"`
- [ ] `.release-please-manifest.json:2` -- confirm/bump to the release target (currently `0.6.0`)
- [ ] `release-please-config.json` -- verify `release-type`, base branch, exclude-paths
- [ ] `.goreleaser.yaml:17-19` -- verify the `-X` ldflags import path still matches `internal/version`
- [ ] `CHANGELOG.md:3` -- confirm the top entry matches the manifest/tag
- [ ] `internal/builtin/cache.go:23` -- note dev-build cache aliasing (all `"dev"` builds share one builtin cache dir)
- [ ] `go.mod:3` vs `.github/workflows/ci.yml:30,42,64` vs `.github/copilot-instructions.md:41` -- reconcile the Go-toolchain version strings
- [ ] Add a CI/release check asserting `.release-please-manifest.json` version == git tag == `nd version` output

### Acceptance criteria

- A documented single source of truth for the product version exists
- A release-time check fails if manifest, git tag, and the built binary disagree
- All version strings reflect the release target (0.7.0)

### References

- Seed task: ps3zxi -- `tasks/cli/ps3zxi-merge-v070-release.md`
- `internal/version/version.go`, `.release-please-manifest.json`, `.goreleaser.yaml`
