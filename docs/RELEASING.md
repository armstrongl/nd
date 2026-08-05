# Releasing nd

This document describes where the product version of `nd` comes from, how it
reaches a released binary, and how an automated check keeps the version embedded
in the binary from drifting away from the release metadata.

## Product version source of truth

The single source of truth for the **product version** of `nd` is the `Version`
variable in `internal/version/version.go` (line 9), which defaults to `"dev"`
for local builds:

```go
var (
    Version = "dev"
    Commit  = "none"
    Date    = "unknown"
)
```

At release time goreleaser overrides `Version` through linker flags (see
[Build-time version injection](#build-time-version-injection)); nothing else
sets it. It is read in exactly two places:

- `cmd/version.go` — the `nd version` command, via `version.String()`, which
  formats `nd version <ver> (commit: <commit>, built: <date>)`.
- `internal/builtin/cache.go` — `sanitizeVersion(version.Version)` names the
  built-in source cache directory (see
  [Dev-build cache aliasing](#dev-build-cache-aliasing)).

Do not introduce a second copy of the product version. The release-metadata
files below are driven by release-please, not edited by hand.

## Release flow

Releases are driven by
[release-please](https://github.com/googleapis/release-please) and cut by
[goreleaser](https://goreleaser.com). The version flows in one direction, from
the release tag into the binary:

1. release-please opens a release pull request that bumps
   `.release-please-manifest.json` and prepends an entry to `CHANGELOG.md`.
2. Merging that pull request to `main` makes release-please create the GitHub
   release and the `vX.Y.Z` git tag. The tag is the authoritative release
   identifier.
3. The tag / release triggers goreleaser, wired up in
   `.github/workflows/release-please.yml` (on `release_created`) and
   `.github/workflows/release.yml` (on a `v*` tag push).
4. goreleaser builds the binaries and injects the version through linker flags
   into `internal/version.Version`.

```text
release-please PR  ->  bump .release-please-manifest.json + CHANGELOG.md
                   ->  merge to main
                   ->  git tag vX.Y.Z
                   ->  release-please.yml / release.yml
                   ->  goreleaser (.goreleaser.yaml)
                   ->  -X .../internal/version.Version={{.Version}}
                   ->  internal/version.Version  (reported by `nd version`)
```

## Build-time version injection

`.goreleaser.yaml` (lines 15-17) sets the three build variables with `-ldflags`:

```yaml
ldflags:
  - -s -w
  - -X github.com/armstrongl/nd/internal/version.Version={{.Version}}
  - -X github.com/armstrongl/nd/internal/version.Commit={{.ShortCommit}}
  - -X github.com/armstrongl/nd/internal/version.Date={{.Date}}
```

goreleaser derives `{{.Version}}` from the git tag (with the leading `v`
stripped). If `internal/version` is renamed or moved, both these lines and this
document must be updated: an unresolved `-X` target is silently ignored by the
linker, so the binary would ship reporting `dev`.

## Version consistency check

`scripts/check-version-drift.sh` fails the release when the product version is
inconsistent across the three places that carry it:

- the release-please manifest (`jq -r '.["."]' .release-please-manifest.json`),
- the git tag being released (`$GITHUB_REF_NAME` on a tag push, or `tag_name`
  reported by release-please), and
- the binary built with the goreleaser linker flags, read back from
  `nd version`.

All three are normalized (leading `v` stripped, whitespace trimmed) and
compared; any mismatch exits non-zero with a message naming the three values.
The script runs after `actions/setup-go` and before goreleaser in both
`.github/workflows/release.yml` and `.github/workflows/release-please.yml`, and
is runnable locally:

```shell
./scripts/check-version-drift.sh
```

Because it rebuilds `nd` with the same `-X` injection and reads the version
back, it also catches a renamed or relocated `Version` var, which would
otherwise ship a binary reporting `dev`.

## Go toolchain version

The Go toolchain version is pinned once, in `go.mod` (`go 1.25.8`). CI and the
release workflows read it with `go-version-file: go.mod` in `actions/setup-go`,
so there is no separate hardcoded Go version to keep in sync. The `Go 1.25+`
note in `.github/copilot-instructions.md` records the supported minimum, not a
competing pin.

## Dev-build cache aliasing

`internal/builtin/cache.go` extracts embedded built-in source into
`$XDG_CACHE_HOME/nd/builtin/<version>/` (default
`~/.cache/nd/builtin/<version>/`), where `<version>` is
`sanitizeVersion(version.Version)`. Every locally built binary reports
`version.Version == "dev"`, so all local builds share one cache directory:
`~/.cache/nd/builtin/dev/`. `EnsureExtracted` skips extraction when that
directory already exists, so a stale `dev/` cache is reused across rebuilds.

After changing embedded source, clear the dev cache so the next build
re-extracts it:

```shell
rm -rf ~/.cache/nd/builtin/dev
```

Released binaries are unaffected: each carries a distinct injected version, so
each release gets its own cache directory.
