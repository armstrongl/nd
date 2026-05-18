---
title: "Fix stale builtin cache and long-lived SourceManager config"
id: "r8j6i9"
status: pending
priority: medium
type: bug
tags: ["cli", "deploy"]
created_at: "2026-05-17"
dependencies: ["unaa3u"]
verify:
  - type: bash
    run: "go build -o nd ."
  - type: bash
    run: "go test ./internal/sourcemanager/... ./internal/builtin/... ./cmd/..."
  - type: bash
    run: "go test -race ./internal/sourcemanager/... ./internal/builtin/..."
  - type: bash
    run: "golangci-lint run"
  - type: assert
    check: "After an external edit to config.yaml (e.g. concurrent `nd source add`), the next Scan() in the same long-lived process reflects the new source set"
  - type: assert
    check: "Adding/removing an asset file inside an already-registered source is still reflected on the next scan (no mtime short-circuit regression)"
  - type: assert
    check: "Builtin source re-extracts when the embedded content stamp differs (simulated nd upgrade with changed embedded assets)"
  - type: assert
    check: "`nd deploy` with no args performs exactly one source scan, not two"
context:
  - "internal/builtin/cache.go"
  - "internal/sourcemanager/sourcemanager.go"
  - "internal/sourcemanager/config.go"
  - "internal/sourcemanager/register.go"
  - "internal/sourcemanager/scanner.go"
  - "cmd/app.go"
  - "cmd/deploy.go"
  - "internal/tui/source.go"
  - "internal/version/version.go"
  - "tasks/cli/7s19kh-fix-deploy-stale-cache.md"
---

## Fix stale builtin cache and long-lived SourceManager config

### Objective

Make `nd` reflect on-disk reality within a single long-lived process (the TUI).
Two concrete staleness bugs exist; the seed task `7s19kh` blamed a cached
`ScanSummary`, but that hypothesis is **disproven** — `SourceManager.Scan()`
(`internal/sourcemanager/sourcemanager.go:121-138`) re-walks the filesystem on
every call via `ScanSource` (`internal/sourcemanager/scanner.go:45-82`), with no
cached summary and no mtime short-circuit. The genuine root causes are:

1. **Stale builtin cache.** `EnsureExtracted` (`internal/builtin/cache.go:39-42`)
   early-returns whenever the cache dir already exists and never re-extracts.
   Because the default `version.Version` is `"dev"`
   (`internal/version/version.go:9`) and `CacheDir()` keys the cache only by
   version (`cache.go:18-24`), every `dev` build shares
   `~/.cache/nd/builtin/dev/`. After rebuilding `nd` with changed embedded
   assets, the old extracted copy is served forever.
2. **Frozen `SourceManager` config.** `sm.cfg` is loaded once in `New()`
   (`internal/sourcemanager/sourcemanager.go:27-50`) and never re-read. `cmd/App`
   memoizes the manager in `a.sm` (`cmd/app.go:48-58`). The TUI runs as one
   long-lived process built from a single `App` (`cmd/root.go:44-50` →
   `tui.Run(app)`), so an external `config.yaml` edit (a concurrent
   `nd source add/remove` in another shell, or a manual edit) is invisible until
   the process restarts.

In-process source mutations from the TUI itself already work: `runAdd` /
`Remove` mutate `sm.cfg.Sources` in memory and persist via `WriteConfig`
(`internal/sourcemanager/register.go:80-87`, `:116-123`). The fix must not
clobber those in-memory edits with a reload.

"NFR-006" cited in `sourcemanager.go:120` is only a code-comment label meaning
"unavailable sources produce warnings, not errors" — preserve that behavior.

### Reproduction

Builtin cache (bug 1):

1. `go build -o nd .` (Version defaults to `dev`).
2. `./nd deploy` — note builtin assets listed.
3. Edit/add a file under `internal/builtin/source/`, `go build -o nd .` again.
4. `./nd deploy` — old builtin asset list still shown (cache not re-extracted).

Frozen config (bug 2), via the long-lived TUI:

1. `./nd source add ./my-source`; open the TUI (`./nd`) and view Sources.
2. In another shell: `./nd source add ./other-source` (edits the same
   `config.yaml`).
3. Back in the still-running TUI, trigger another scan/source view — the
   externally added source and its assets do not appear.

### Tasks

- [ ] **Builtin cache re-extraction.** In `internal/builtin/cache.go`, replace
  the unconditional early return at `EnsureExtracted` (`cache.go:39-42`) with a
  content-stamp check. Write a stamp file (e.g. `.nd-builtin-stamp`) into the
  extracted dir containing a checksum/hash of the embedded `FS` tree (walk
  `FS` rooted at `"source"` like `cache.go:66`, hash paths+contents). On
  `EnsureExtracted`: if the dir exists AND a matching stamp file exists, return
  early; otherwise remove the stale dir and re-extract (reuse the existing
  temp-dir + atomic `os.Rename` flow at `cache.go:44-104`). Compute the stamp
  once and write it inside the temp dir before the rename. Tests can override
  the cache root via the package var `cacheBaseDir` (`cache.go:14`).
- [ ] **Config reload in `SourceManager`.** In
  `internal/sourcemanager/sourcemanager.go`: add fields to the `SourceManager`
  struct (`:17-22`) to record the observed config-file `mtime`+`size` of
  `configPath` (and the project config path
  `<projectDir>/.nd/config.yaml`, see `sourcemanager.go:33-40`) at `New()`
  time. Add an unexported `reloadConfigIfChanged()` that re-stats those paths;
  if mtime/size changed since last observation, re-run the
  `LoadConfig`+`LoadProjectConfig`+`MergeConfigs`+`appendBuiltinSource`
  pipeline (mirror `New()` `:27-49`) and replace `sm.cfg`, then update the
  recorded stat. Call it at the top of `Scan()` (`:121`). Use a `sync.Mutex`
  to guard `sm.cfg` if mutating concurrently is plausible (Scan is callable
  from TUI goroutine commands).
- [ ] **Do not clobber in-memory mutations.** Ensure `AddLocal` / `AddGit` /
  `Remove` (`internal/sourcemanager/register.go:45-126`) update the recorded
  config stat after their successful `WriteConfig`, so the reload check treats
  the self-written file as already-observed and does not re-read (which would
  be harmless here but avoids a redundant load and any race window). Confirm
  TUI `runAdd` / `runRemove` (`internal/tui/source.go:305-322`, `:412-425`)
  still see their own change after a subsequent `Scan()`.
- [ ] **Pick up external edits in the long-lived TUI.** No change to
  `cmd/app.go` memoization is needed if `Scan()` reloads internally —
  `App.ScanIndex()` (`cmd/app.go:216-222`) delegates to `sm.Scan()` and
  `App.SourceManager()` (`:48-58`) returns the same `sm`. Verify
  `App.SourceManager().Config()` callers (e.g. `cmd/deploy.go:152-156`,
  `cmd/app.go:69`) tolerate a config that may have been reloaded; if a caller
  needs the fresh source set, route it through `Scan()` or add a
  `Config()`-side reload. Document the chosen approach in a code comment.
- [ ] **Collapse the double scan in `nd deploy`.** In `cmd/deploy.go`, the
  no-args interactive path scans once at `cmd/deploy.go:66` to build the
  picker, then unconditionally scans again at `cmd/deploy.go:89`. Reuse the
  first `scanResult` for the args path (only the interactive branch double
  scans; when args are provided only line 89 runs). Restructure so the
  picker-built `args` flow into a single retained `summary`/`index` without a
  second `app.ScanIndex()` call. Keep the args-supplied path scanning exactly
  once.
- [ ] **Regression test — external config edit visible.** In
  `internal/sourcemanager/sourcemanager_test.go`, add a test: create a
  `SourceManager` with `sourcemanager.New(configPath, "")`, `Scan()`, then
  externally rewrite `configPath` to add a second local source (use
  `makeSourceTree` from `scanner_test.go:14` to build a real source dir, write
  YAML matching the `TestSourcesPopulated` format at `:84-97`), bump its mtime
  if needed, `Scan()` again, assert the new source's assets appear. Account
  for the always-present builtin source like `TestScan` (`:157-170`).
- [ ] **Regression test — in-source file add still visible (mtime guard).** Add
  a test mirroring `TestScan` (`sourcemanager_test.go:134-171`): register a
  temp source via `sm.AddLocal`, `Scan()`, add a new asset file inside the
  registered source dir, `Scan()` again, assert the new asset appears. This
  guards against accidentally introducing a directory-mtime short-circuit.
- [ ] **Regression test — builtin re-extracts on stamp change.** In
  `internal/builtin/cache_test.go` (follow `TestEnsureExtracted_Idempotent`
  at `:43-54` and the `cacheBaseDir` override pattern in
  `TestCacheDir_RespectsXDGOverride` `:84-95`): extract once, corrupt/delete
  the stamp file or write a mismatching stamp, call `EnsureExtracted` again,
  assert the dir was re-extracted (e.g. a sentinel file you injected into the
  stale dir is gone and `skills/` is freshly present).

### Acceptance criteria

- After an external `config.yaml` change (concurrent `nd source add/remove` or
  manual edit), the next `Scan()` / `ScanIndex()` in the **same process**
  reflects the new source set.
- Adding/removing an asset file *within* an already-registered source is still
  reflected on the next scan (no mtime-based short-circuit regression).
- The builtin source re-extracts when the embedded-content stamp differs
  (simulated `nd` upgrade with changed embedded assets); identical content is a
  no-op (idempotent, no needless re-extract).
- TUI-initiated `source add`/`remove` are not lost by the reload (in-memory
  mutation survives a subsequent `Scan()`).
- `nd deploy` with no args performs exactly one source scan, not two; the
  args-supplied path still scans exactly once.
- Unavailable sources still warn (not error) — `Scan()` returns `nil` error
  with `Warnings` populated (preserve `TestScanUnavailableSourceWarning`,
  `sourcemanager_test.go:196-218`).
- `go build -o nd .` succeeds.
- `go test ./internal/sourcemanager/... ./internal/builtin/... ./cmd/...`
  passes; `go test -race ./internal/sourcemanager/... ./internal/builtin/...`
  passes; `golangci-lint run` is clean.

### References

- Seed (cancelled, superseded by this task):
  `tasks/cli/7s19kh-fix-deploy-stale-cache.md` (GitHub issue #60). Its
  user-facing requirements (add/remove in a registered source reflected by
  `nd deploy` interactive + tab-completion and `nd list`; no scan-perf
  regression) are folded into the criteria above.
- Dependency `unaa3u` (`tasks/cli/unaa3u-builtin-source.md`): ships the builtin
  source and the `appendBuiltinSource` / `builtin.Path()` / cache machinery
  this task hardens. Its Unit 3 spec'd version-keyed cache invalidation; this
  task closes the gap that `dev` builds reuse one unversioned-by-content cache.
- Code: `internal/builtin/cache.go`, `internal/sourcemanager/sourcemanager.go`,
  `internal/sourcemanager/config.go` (`LoadConfig`/`MergeConfigs`/`WriteConfig`),
  `internal/sourcemanager/register.go`, `internal/sourcemanager/scanner.go`,
  `cmd/app.go` (`SourceManager`/`ScanIndex`), `cmd/deploy.go`,
  `internal/tui/source.go`, `internal/version/version.go`.
- Test helpers: `makeSourceTree` (`internal/sourcemanager/scanner_test.go:14`),
  `newTestManager` (`internal/sourcemanager/register_test.go:28`), `execGit`
  (`register_test.go:126`), `cacheBaseDir` override (`internal/builtin/cache.go:14`).
