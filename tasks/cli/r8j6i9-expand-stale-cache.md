---
title: "Fix stale builtin cache and long-lived SourceManager config"
id: "r8j6i9"
status: pending
priority: medium
type: bug
tags: ["cli", "deploy"]
created_at: "2026-05-17"
dependencies: ["unaa3u"]
---

## Fix stale builtin cache and long-lived SourceManager config

### Objective

Pattern expansion of seed 7s19kh. `SourceManager.Scan()` correctly re-walks the filesystem each call, so the seed's "cached ScanSummary" hypothesis is not the real root cause. The genuine staleness is: (1) the builtin source extraction cache never re-extracts after first run, and (2) the long-lived TUI session caches a `SourceManager` whose source list (`sm.cfg`) is frozen at process start, so a concurrent `nd source add/remove` or external config edit is invisible.

### Tasks

- [ ] `internal/builtin/cache.go:40-42` -- invalidate/re-extract when embedded content changes (version/checksum stamp file) instead of an unconditional early return; note `Version == "dev"` shares one cache dir
- [ ] `internal/sourcemanager/sourcemanager.go:121` -- have `Scan()` (or a new `ReloadConfigIfChanged`) re-stat the config path(s) and reload+remerge `sm.cfg` when mtime/size changed since `New()`; store observed mtime on the struct
- [ ] `cmd/app.go:48-58` / `cmd/app.go:216` -- make `ScanIndex`/`SourceManager` pick up external config edits in a long-lived process (TUI)
- [ ] `cmd/deploy.go:66` and `cmd/deploy.go:89` -- collapse the redundant double `ScanIndex` in the deploy `RunE` into one scan
- [ ] Confirm `internal/tui/source.go` `runAdd` / `sm.Remove` in-memory mutation is not clobbered by a stale reload
- [ ] Regression test: register temp source, scan, externally add a source to config, scan again -> new source's assets appear
- [ ] Regression test: register temp source, scan, add a file inside it, scan again -> file appears (guards against re-introducing an mtime short-circuit)
- [ ] Regression test: builtin cache re-extracts when the stamp differs

### Acceptance criteria

- After config changes on disk (concurrent `nd source add/remove` or manual edit), the next `Scan()`/`ScanIndex()` in the same process reflects the new source set
- Adding/removing asset files within a registered source is still reflected (no mtime regression)
- Builtin source re-extracts after an nd upgrade with changed embedded assets
- `nd deploy` performs one scan, not two
- `go test ./internal/sourcemanager/... ./internal/builtin/... ./cmd/...` passes

### References

- Seed task: 7s19kh -- `tasks/cli/7s19kh-fix-deploy-stale-cache.md`
- `cmd/app.go`, `cmd/deploy.go`, `internal/sourcemanager/sourcemanager.go`, `internal/builtin/cache.go`

### Merged scope (supersedes 7s19kh)

This task supersedes cancelled seed `7s19kh`, whose "source manager caches a ScanSummary" hypothesis the codebase sweep disproved (`Scan()` re-walks the FS every call). The user-facing behavior 7s19kh wanted is preserved here. Reproduction (from 7s19kh): `nd source add ./my-source`; `nd deploy` (note assets); add/remove an asset in `./my-source`; `nd deploy` again -- stale list. Added acceptance from 7s19kh: after add/remove in a registered source, `nd deploy` interactive and tab-completion reflect it; `nd list` reflects it (same scan path); no scan-performance regression for large sources.
