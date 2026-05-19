---
title: "Fix deploy/agent edge cases in backup target default detection and init symlink strategy"
id: "j54b7l"
status: pending
priority: medium
type: bug
tags: ["deploy"]
created_at: "2026-05-17"
context:
  - "internal/deploy/deploy.go"
  - "internal/deploy/deploy_test.go"
  - "internal/agent/registry.go"
  - "internal/agent/registry_test.go"
  - "internal/agent/agent.go"
  - "cmd/init_cmd.go"
  - "cmd/init_cmd_test.go"
  - "cmd/deploy.go"
  - "cmd/app.go"
verify:
  - type: bash
    run: "cd /Users/larah/Repos/Personal/nd && go build -o nd ."
  - type: bash
    run: "cd /Users/larah/Repos/Personal/nd && go test ./internal/deploy/... ./internal/agent/... ./cmd/..."
  - type: bash
    run: "cd /Users/larah/Repos/Personal/nd && go test -race ./internal/deploy/... ./internal/agent/... ./cmd/..."
  - type: assert
    check: "Foreign-symlink conflict backup warning for a context asset includes the previous symlink target path"
  - type: assert
    check: "Registry.Default() does not select a binary-less stale GlobalDir over an agent whose binary is in PATH"
  - type: assert
    check: "nd init built-in deploy uses SymlinkRelative when the written config sets symlink_strategy: relative"
---

## Fix deploy/agent edge cases

### Objective

Fix three independent, low-blast-radius defects in the deploy/agent path. These were found during a manual codebase sweep (no seed task or generated fixture — "net-new, no seed" just means there is no upstream design doc; treat this task as the sole source of truth):

1. A foreign-symlink conflict on a context asset is silently backed up, but the backup warning never reports where the old symlink pointed, destroying the only recovery breadcrumb.
2. `Registry.Default()` can pick a stale, empty agent directory (e.g. an abandoned `~/.copilot` with no `copilot` binary) over a fully installed agent, because directory existence alone marks an agent "detected".
3. `nd init`'s built-in asset deploy hardcodes `SymlinkAbsolute`, ignoring the `symlink_strategy` the user just configured, so init produces absolute symlinks even when the config says `relative`.

All file:line references below were verified against the current tree on branch `main`.

### Tasks

- [ ] **Foreign-symlink backup warning loses the old target.** In `internal/deploy/deploy.go`, `backupAndWarn` (signature at line 403, `func (e *Engine) backupAndWarn(linkPath string, kind nd.OriginalFileKind, target string)`) discards its `target` parameter at line 415 (`_ = target // suppress unused warning`). The only caller that passes a non-empty `target` is the foreign-symlink-on-context branch at `internal/deploy/deploy.go:368` (`backed, w := e.backupAndWarn(linkPath, nd.FileKindForeignSymlink, target)`, where `target` is the result of `e.readlink(linkPath)` at line 345). The plain-file caller at line 386 passes `""`. Fix: when `kind == nd.FileKindForeignSymlink` and `target != ""`, append the old target to the warning message built at lines 411–414, e.g. `msg = fmt.Sprintf("Backed up existing %s at %s (was pointing to %s) to %s", kind, linkPath, target, backed)`. Remove the now-unnecessary `_ = target` discard at line 415. Do not change behavior for `FileKindPlainFile` (line 412–414 special case) or when `target == ""`.

- [ ] **`Registry.Default()` can pick a binary-less stale dir.** In `internal/agent/registry.go`, `Detect()` sets `r.agents[i].Detected = r.agents[i].InPath || dirExists` at line 142 — so an agent whose `GlobalDir` merely exists (line 138 `r.stat(r.agents[i].GlobalDir)`) is "detected" even with no binary in PATH. `Default()` (line 188) then, after the configured-default check (lines 193–199), falls back to "first detected" at lines 201–205, which can return that stale dir over a real agent. The `Agent` struct (`internal/agent/agent.go:21-22`) exposes both `Detected` and `InPath` (set true only when the binary is found in PATH *and* version-verified — see `registry.go:128-134`). Fix the fallback loop at `internal/agent/registry.go:201-205`: do a first pass that returns the first agent with `InPath == true`, and only if none has `InPath` fall back to the first `Detected` agent (current behavior). Keep the configured-default block (lines 193–199) and the final no-agents error (line 207) unchanged. Order of `r.agents` is `claude-code` then `copilot` (defined `registry.go:35-58`); preserve that ordering within each pass.

- [ ] **`nd init` built-in deploy ignores configured symlink strategy.** In `cmd/init_cmd.go`, `deployBuiltinAssets` (signature line 160) builds requests at lines 230–237 with a hardcoded `Strategy: nd.SymlinkAbsolute` (line 235). By the time `deployBuiltinAssets` runs, `runInitSetup` has already written the config file (`cmd/init_cmd.go:96-104`, default body includes `symlink_strategy: absolute`) to `app.ConfigPath`. Mirror the resolution logic already used in `cmd/deploy.go:150-157`:

  ```go
  strategy := nd.SymlinkAbsolute
  if sm, smErr := app.SourceManager(); smErr == nil {
      cfg := sm.Config()
      if cfg.SymlinkStrategy != "" {
          strategy = cfg.SymlinkStrategy
      }
  }
  ```

  Then set `Strategy: strategy` in the request loop (line 235). `app.SourceManager()` (`cmd/app.go:48`) lazily constructs and caches a `SourceManager` from `a.ConfigPath`; `sm.Config()` (`internal/sourcemanager/sourcemanager.go:53`) returns the loaded `*config.Config` whose `SymlinkStrategy` field is `nd.SymlinkStrategy` (`internal/config/config.go:12`; valid values `nd.SymlinkAbsolute`/`nd.SymlinkRelative` from `internal/nd/symlink.go:7-8`). `nd init` has no `--relative`/`--absolute` flags, so do NOT add the flag-override branch from `cmd/deploy.go:158-162` — config-or-default only.

- [ ] **Add regression tests for all three.**
  - In `internal/deploy/deploy_test.go`, add a test modeled on `TestDeployForeignSymlinkContextBacksUp` (line 431). Use the same harness (`newMockStore`, `engine.SetLstat` → symlink mode, `engine.SetReadlink` returning a known sentinel like `/some/other/target`, context asset). Assert `len(result.Warnings) > 0` and that one warning `strings.Contains(...)` the readlink target string. `DeployResult.Warnings` is the exported field (`internal/deploy/deploy.go:113`).
  - In `internal/agent/registry_test.go`, add a test using the `stubRegistry(cfg, lookPath, stat)` helper (line 15) plus `lookPathFound`/`lookPathNotFound` (lines 25,29). Configure: `copilot`'s `GlobalDir` stat succeeds but its binary is NOT in PATH; `claude-code`'s `claude` binary IS in PATH. Assert `Default()` returns `claude-code`, not `copilot`. Follow the existing pattern in `TestDefaultFallsBackToFirstDetected` (line 211) for structure.
  - In `cmd/init_cmd_test.go`, add a test modeled on `TestInitCmd_WithYes_DeploysBuiltinAssets` (line 62). Run `nd init` non-interactively with a config whose `symlink_strategy: relative`, then assert a deployed built-in symlink is relative (`os.Readlink` returns a non-absolute path, i.e. `!filepath.IsAbs(target)`). Note: `runInitSetup` writes a default config with `symlink_strategy: absolute`; the test must either override `app.ConfigPath` content after setup or use the test seam (`app.initRegistry`/`app.initAgent`, see `cmd/init_cmd.go:56,114-118`) consistent with existing init tests.

### Acceptance criteria

- A foreign-symlink conflict on a context asset produces a backup warning that contains the previous symlink target path; plain-file and empty-target backups are unchanged.
- `Registry.Default()` prefers an agent with `InPath == true` and never returns a binary-less stale `GlobalDir` agent when an in-PATH agent exists; the configured-default behavior (`registry.go:193-199`) and no-agent error (`registry.go:207`) are unchanged.
- `nd init` built-in deploy creates relative symlinks when the configured `symlink_strategy` is `relative`, and absolute otherwise (default).
- New regression tests exist for all three fixes and pass under `go test -race`.
- `go build -o nd .` succeeds and the full test suite (`go test ./...`) is green.

### References

- GitHub issue: https://GitHub.com/armstrongl/nd/issues/134
- Close this issue when the task is completed.
- Bug 1: `internal/deploy/deploy.go:403-418` (`backupAndWarn`, discard at line 415), caller `internal/deploy/deploy.go:368` (`target` from `readlink` at line 345). Result field `internal/deploy/deploy.go:113`. Test pattern: `internal/deploy/deploy_test.go:431` (`TestDeployForeignSymlinkContextBacksUp`).
- Bug 2: `internal/agent/registry.go:142` (Detected = InPath || dirExists), `registry.go:201-205` (first-detected fallback in `Default()`), `registry.go:128-134` (InPath set), `internal/agent/agent.go:21-22` (`Detected`/`InPath` fields). Test pattern: `internal/agent/registry_test.go:15` (`stubRegistry`), `registry_test.go:211` (`TestDefaultFallsBackToFirstDetected`).
- Bug 3: `cmd/init_cmd.go:230-237` (hardcoded `SymlinkAbsolute` at line 235), pattern to mirror `cmd/deploy.go:150-157`, `cmd/app.go:48` (`SourceManager()`), `internal/sourcemanager/sourcemanager.go:53` (`Config()`), `internal/config/config.go:12` + `internal/nd/symlink.go:7-8` (strategy type/values). Test pattern: `cmd/init_cmd_test.go:62` (`TestInitCmd_WithYes_DeploysBuiltinAssets`).
- No seed/design doc — this task file is the complete specification.
