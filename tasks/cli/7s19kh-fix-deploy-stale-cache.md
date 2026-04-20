---
title: "Fix nd deploy listing stale cached assets"
id: "7s19kh"
status: pending
priority: medium
type: bug
tags: ["cli", "deploy"]
created_at: "2026-04-20"
---

## Fix nd deploy listing stale cached assets

### Objective

When a source directory changes (assets added, renamed, or removed), `nd deploy` should reflect the current filesystem state. Currently the interactive picker and tab-completion show stale entries that no longer exist on disk.

### Steps to reproduce

1. Register a source: `nd source add ./my-source`
2. Run `nd deploy` (interactive picker) and note the listed assets
3. Add or remove an asset file/directory inside `./my-source`
4. Run `nd deploy` again
5. Observe that the picker still shows the old asset list

### Expected behavior

Each invocation of `nd deploy` (and its tab-completion function) rescans the source directories and returns a fresh asset index reflecting the current filesystem contents.

### Actual behavior

The deploy command's interactive picker and `ValidArgsFunction` call `app.ScanIndex()` which delegates to `sm.Scan()`. If the source manager or scanner caches results across calls within the same process (or if the index is built from stale metadata), removed assets still appear and new assets are missing.

### Tasks

- [ ] Trace the call path from `app.ScanIndex()` -> `SourceManager.Scan()` -> scanner to identify where caching occurs
- [ ] Determine whether the source manager caches a `ScanSummary` and returns it on subsequent calls without rescanning the filesystem
- [ ] If the scanner uses a file-modification-time check, verify it detects added/removed entries (not just modified files)
- [ ] Invalidate or bypass the scan cache when the source directory's contents have changed (compare directory mtime or entry count)
- [ ] Add a regression test: register a temp source, scan, add a file, scan again, assert the new asset appears
- [ ] Add a regression test: register a temp source, scan, remove a file, scan again, assert the removed asset is gone
- [ ] Verify the fix works for both the interactive picker (`RunE` path) and tab-completion (`ValidArgsFunction` path)

### Acceptance criteria

- After adding an asset to a registered source, `nd deploy` (interactive) lists the new asset without restarting
- After removing an asset from a registered source, `nd deploy` (interactive) no longer lists the removed asset
- Tab-completion for `nd deploy <TAB>` reflects the current source contents
- `nd list` also reflects the current source contents (same scan path)
- No regression in scan performance for sources with many assets (rescan should not be slower than initial scan)
- Tests pass: `go test ./internal/sourcemanager/... ./cmd/... -run TestStale`

### References

- GitHub issue: https://GitHub.com/armstrongl/nd/issues/60
- Source: `cmd/deploy.go`, `cmd/app.go` (`ScanIndex`), `internal/sourcemanager/`
