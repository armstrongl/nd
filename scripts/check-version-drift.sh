#!/usr/bin/env bash
# Release-drift guard for nd.
#
# Fails when the product version disagrees across the three places that carry it:
#   1. the release-please manifest (.release-please-manifest.json)
#   2. the git tag being released (vX.Y.Z)
#   3. the binary built with goreleaser-equivalent ldflags (`nd version`)
#
# The single product-version source of truth is the `Version` var in
# internal/version/version.go (default "dev"), which goreleaser overrides at
# release time via `-X github.com/armstrongl/nd/internal/version.Version=...`
# (see .goreleaser.yaml). This script rebuilds nd with the same injection so a
# renamed or relocated `Version` var (which would silently make the -X a no-op
# and ship a binary reporting "dev") is caught here.
#
# Usage:
#   scripts/check-version-drift.sh
#
# Tag resolution order:
#   $RELEASE_TAG      explicit override (release-please.yml passes tag_name)
#   $GITHUB_REF_NAME  set to the tag on a `v*` tag push (release.yml)
#   latest git tag    local fallback so the check is runnable off-CI
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

manifest_file=".release-please-manifest.json"
version_pkg="github.com/armstrongl/nd/internal/version"

# strip_v removes a single leading "v" so v0.6.0 and 0.6.0 compare equal.
strip_v() { printf '%s' "${1#v}"; }

resolve_tag() {
  if [ -n "${RELEASE_TAG:-}" ]; then
    printf '%s' "$RELEASE_TAG"
    return 0
  fi
  if [ -n "${GITHUB_REF_NAME:-}" ]; then
    printf '%s' "$GITHUB_REF_NAME"
    return 0
  fi
  local t
  t="$(git describe --tags --abbrev=0 2>/dev/null || true)"
  if [ -z "$t" ]; then
    t="$(git tag --sort=-creatordate 2>/dev/null | head -n 1 || true)"
  fi
  printf '%s' "$t"
}

# 1. Manifest version (release-please's record of the last released version).
manifest_version="$(jq -r '.["."]' "$manifest_file")"
if [ -z "$manifest_version" ] || [ "$manifest_version" = "null" ]; then
  echo "check-version-drift: could not read version from $manifest_file" >&2
  exit 1
fi
manifest_version="$(strip_v "$manifest_version")"

# 2. Tag version.
tag="$(resolve_tag)"
if [ -z "$tag" ]; then
  echo "check-version-drift: no tag found (set RELEASE_TAG or GITHUB_REF_NAME, or create a git tag)" >&2
  exit 1
fi
tag_version="$(strip_v "$tag")"

# 3. Binary version. Build with the same ldflags goreleaser injects, passing the
#    tag as {{.Version}}, then read it back via `nd version`.
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT
tmp_bin="$tmp_dir/nd"
go build -ldflags "-X ${version_pkg}.Version=${tag_version}" -o "$tmp_bin" .
# version.String() prints: "nd version <ver> (commit: ..., built: ...)".
version_line="$("$tmp_bin" version)"
binary_version="$(printf '%s\n' "$version_line" | awk '{print $3}')"
binary_version="$(strip_v "$binary_version")"

# Compare all three (they are already stripped of the leading "v").
if [ "$manifest_version" != "$tag_version" ] || [ "$binary_version" != "$tag_version" ]; then
  {
    echo "check-version-drift: version mismatch"
    echo "  manifest ($manifest_file): $manifest_version"
    echo "  git tag:                    $tag_version"
    echo "  binary (nd version):        $binary_version"
  } >&2
  exit 1
fi

echo "check-version-drift: OK — manifest, tag, and binary all report $tag_version"
