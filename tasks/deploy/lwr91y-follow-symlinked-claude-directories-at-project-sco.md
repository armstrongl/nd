---
title: "Follow symlinked .claude directories at project scope"
id: "lwr91y"
status: pending
priority: high
type: feature
tags: ["symlink", "project-scope", "deploy"]
created_at: "2026-08-05"
---

## Follow symlinked .claude directories at project scope

### Objective

When a project's agent config directory (`.claude/`, `.agents/`, `.github/`, …) is itself a
symlink pointing at a shared location — a dotfiles checkout, a monorepo-wide config dir, or
another project — nd should resolve it and deploy against the real target instead of treating
the logical path as authoritative.

Today every deploy path is built by string-joining the project root with the agent's
`ProjectDir` (`internal/agent/agent.go:83`, `internal/agent/agent.go:72`), and nothing calls
`filepath.EvalSymlinks`. The OS transparently follows the symlinked parent when creating the
link, so deploys appear to work, but three things break:

1. **Relative symlinks dangle.** With `nd.SymlinkRelative`, the target is computed as
   `filepath.Rel(filepath.Dir(linkPath), sourcePath)` (`internal/deploy/deploy.go:260`) using
   the *logical* parent. The kernel resolves it from the *physical* parent, so the link points
   at the wrong place. Example: `/proj/.claude -> /shared/claude`, asset written to
   `/proj/.claude/skills/x` lands physically at `/shared/claude/skills/x`, and a target computed
   relative to `/proj/.claude/skills` resolves from `/shared/claude/skills` — dangling.
2. **State identity splits.** `state.Deployment.LinkPath` (`internal/state/state.go:26`) records
   the unresolved path. Two projects sharing one symlink target write to the same physical file
   under two different `LinkPath` values, so the same-asset re-deploy check in `handleConflict`
   (`internal/deploy/deploy.go:358`, exact `d.LinkPath == linkPath` comparison) misses and the
   file is misclassified as a foreign symlink — erroring, or silently clobbering the other
   project's deployment under `--force`.
3. **Removal crosses project boundaries.** `findDeployedAsset` (`cmd/remove.go:220`) and the
   uninstall path match on `LinkPath` plus `ProjectPath`, so removing from project A deletes a
   physical file that project B's state still claims as deployed.

`nd.FindProjectRoot` (`internal/nd/project.go:12`) already works, because `os.Stat` follows
symlinks — a symlinked `.claude` is detected as a project marker. The gap is entirely
downstream of root detection.

### Tasks

- [ ] Add a symlink-resolving helper for agent config dirs (resolve the deepest existing
      ancestor when the leaf does not exist yet, so first-time deploys work)
- [ ] Resolve the config dir in `agent.configDir` and `agent.contextDeployPath`
      (`internal/agent/agent.go:63-88`) so `DeployPath` returns a physical path
- [ ] Fix relative-target computation in `internal/deploy/deploy.go:258-266` to use the resolved
      parent directory
- [ ] Compare deployments by resolved path in `handleConflict`
      (`internal/deploy/deploy.go:340-390`) instead of raw `LinkPath` string equality
- [ ] Handle a broken symlinked config dir with a clear error instead of an opaque
      `mkdirAll` failure (`internal/deploy/deploy.go:255`)
- [ ] Apply the same resolution in health checks (`internal/deploy/health.go:72`,
      `internal/deploy/health.go:142`) and in `findDeployedAsset` (`cmd/remove.go:220`)
- [ ] Decide and document migration for existing state entries recorded under unresolved paths
      (resolve on read vs. one-time rewrite)
- [ ] Add unit tests in `internal/agent/agent_test.go` and `internal/deploy/deploy_test.go`
      covering a symlinked project config dir under both `SymlinkRelative` and absolute
      strategies
- [ ] Add an integration test in `tests/` for deploy → status → remove against a symlinked
      `.claude`
- [ ] Document the supported layout in the docs site

### Acceptance criteria

- Deploying at project scope into a symlinked `.claude` produces a symlink whose target
  resolves correctly under both `nd.SymlinkAbsolute` and `nd.SymlinkRelative`
- Re-deploying the same asset into a symlinked config dir is detected as a same-asset
  re-deploy, not a foreign-symlink conflict
- `nd status` reports assets deployed through a symlinked config dir as healthy
- `nd remove` deletes only the deployment it owns and leaves other projects' state intact
- A `.claude` symlink whose target does not exist fails with an actionable error naming the
  symlink and its target
- Existing state entries written before this change continue to resolve without manual repair
- `go test ./...` passes

### Context

#### Path construction (no symlink resolution today)

- `internal/agent/agent.go:83` — `configDir` joins `projectRoot` + `a.ProjectDir`
- `internal/agent/agent.go:63` — `contextDeployPath`, same pattern for context files
- `internal/nd/project.go:12` — `FindProjectRoot` walks up for `.git`/`.agents`/`.claude`
  using `os.Stat`, which already follows symlinks

#### Deploy engine

- `internal/deploy/deploy.go:255` — `mkdirAll` on the logical parent
- `internal/deploy/deploy.go:258-266` — relative target computation (the dangling-link bug)
- `internal/deploy/deploy.go:340-390` — `handleConflict`: `lstat`s only the leaf, matches
  state by exact `LinkPath` string
- Engine indirects `symlink`/`lstat`/`readlink`/`mkdirAll` through injectable fields
  (`internal/deploy/deploy.go:47-85`), so tests can fake symlink topology without touching disk

#### State and consumers

- `internal/state/state.go:26-28` — `LinkPath`, `ProjectPath`
- `internal/state/state.go:49-65` — dedup key is `{AssetName, Agent, Scope, ProjectPath}`
- `cmd/remove.go:220`, `cmd/uninstall.go:89`, `cmd/status.go:62` — all match on `LinkPath` /
  `ProjectPath`
- `internal/deploy/health.go:72`, `internal/deploy/health.go:142` — `lstat` the recorded path

#### Prior art in the repo

- `internal/sourcemanager/scanner.go` and `internal/sourcemanager/config.go` already handle
  symlinks on the source side; reuse their approach for consistency

### References

- GitHub issue: https://GitHub.com/armstrongl/nd/issues/150
- Close this issue when the task is completed.
