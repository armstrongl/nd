---
title: "Ship built-in source with nd"
id: "unaa3u"
status: pending
priority: high
type: feature
tags: ["core", "onboarding"]
created_at: "2026-04-20"
---

## Ship built-in source with nd

### Objective

Add a first-party "builtin" source embedded in the nd binary that provides starter skills, commands, and agents out of the box. This eliminates the cold-start problem where new users run `nd init` and see an empty asset list with no way to get value without first finding or creating an external source.

### Tasks

- [ ] **Unit 1: Add `SourceBuiltin` type and reserved ID to source type enum**
  - Add `SourceBuiltin SourceType = "builtin"` constant to `internal/nd/source_type.go`
  - Add `BuiltinSourceID = "builtin"` constant in `internal/nd/`
  - Update `Validate()` switch in `internal/config/validation.go` to accept `nd.SourceBuiltin`
  - Write tests confirming config with `type: builtin` passes validation

- [ ] **Unit 2: Create embedded source directory with placeholder assets using Go embed**
  - Create `internal/builtin/embed.go` with `//go:embed source` directive exporting `var FS embed.FS`
  - Create placeholder skill directories: `nd-create-source/SKILL.md`, `nd-scaffold-asset/SKILL.md`, `nd-deploy-workflow/SKILL.md`
  - Create placeholder command files: `nd-quickstart.md`, `nd-audit.md`
  - Create placeholder agent file: `nd-expert.md`
  - Write tests that walk `embed.FS` and verify expected entries exist

- [ ] **Unit 3: Implement cache extraction logic to materialize embedded files to disk**
  - Create `internal/builtin/cache.go` with `CacheDir()`, `EnsureExtracted()`, and `Path()` functions
  - Extract to `$XDG_CACHE_HOME/nd/builtin/<version>/` (default `~/.cache/nd/builtin/<version>/`)
  - Use atomic extraction (temp dir + `os.Rename`) to prevent partial extraction
  - Use `version.Version` for cache invalidation; re-extract on version mismatch
  - Write tests for first-call extraction, no-op on second call, `XDG_CACHE_HOME` override, and fallback

- [ ] **Unit 4: Inject builtin source into config pipeline as lowest-priority source**
  - Add `appendBuiltinSource()` helper called from `New()` in `internal/sourcemanager/sourcemanager.go`
  - Append builtin entry last in `cfg.Sources` (lowest priority per FR-016a ordering)
  - Strip builtin entry in `WriteConfig()` to avoid persisting it to YAML
  - Write tests confirming builtin appears last, user sources retain priority, and YAML excludes builtin

- [ ] **Unit 5: Guard builtin source against removal and ID conflicts**
  - Make `Remove("builtin")` return a clear error in `internal/sourcemanager/register.go`
  - Make `AddLocal()`/`AddGit()` reject or suffix the ID `"builtin"` to avoid conflicts
  - Write tests for removal guard, ID conflict handling, and `nd source list` showing builtin type

- [ ] **Unit 6: Update `nd init` with deploy-all-with-opt-out interactive step**
  - After config creation in `cmd/init_cmd.go`, ensure builtin cache is extracted and scanned
  - Prompt: "nd includes N built-in assets. Deploy all? [Y/n/list]"
  - `Y` (default): deploy all builtin assets to global scope
  - `n`: skip with hint to use `nd deploy --source builtin` later
  - `list`: show asset names, then re-prompt
  - `--yes` flag deploys all silently; `--json` outputs deployed asset list
  - Write tests for `--yes` auto-deploy, JSON output, and skip behavior

### Acceptance criteria

- `nd init` in a fresh environment shows builtin asset count and offers to deploy them
- `nd source list` includes the builtin source with type `builtin`
- `nd source remove builtin` returns a clear error message
- User-defined sources always override builtin assets on name conflicts (lowest priority)
- Builtin source entry never appears in the persisted `config.yaml`
- Cache is version-keyed and automatically re-extracted on nd version change
- All existing tests pass with no regressions
- `go test ./internal/builtin/... ./internal/sourcemanager/... ./internal/config/... ./cmd/...` passes

### References

- Plan: `docs/plans/2026-04-02-002-feat-builtin-source-plan.md`
- Go embed docs: https://pkg.go.dev/embed
- XDG Base Directory Spec: https://specifications.freedesktop.org/basedir-spec/latest/
