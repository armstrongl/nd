---
title: "Additional test coverage gaps beyond TUI audit"
id: "wughc0"
status: pending
priority: low
type: chore
tags: ["testing"]
created_at: "2026-05-17"
dependencies: ["agtaqm"]
verify:
  - type: bash
    run: "go build -o nd ."
  - type: bash
    run: "go test ./internal/deploy/... ./internal/state/... ./internal/sourcemanager/... ./internal/config/... ./cmd/..."
  - type: bash
    run: "go test -race ./internal/deploy/... ./internal/state/... ./internal/sourcemanager/... ./internal/config/... ./cmd/..."
  - type: assert
    check: "Every non-struck checklist item below has at least one corresponding new Test* function, named in the item or its sibling test file, that exercises the cited code path (error return, nil/empty guard, dry-run no-side-effect, oplog entry, or symlink cleanup). Items struck through with ~~...~~ retain a one-line rationale and add no production code."
  - type: assert
    check: "internal/deploy.Engine.SetOrigin and cmd.removeSourceDeployments go from 0% to non-zero statement coverage (compare `go test -cover ./internal/deploy/... ./cmd/...` before vs after; overall statement coverage for each of internal/deploy, internal/state, internal/sourcemanager, internal/config, cmd does not decrease)."
context:
  - internal/deploy/deploy.go
  - internal/deploy/deploy_test.go
  - internal/state/store.go
  - internal/sourcemanager/git.go
  - internal/sourcemanager/register.go
  - internal/config/validation.go
  - cmd/source.go
  - cmd/source_test.go
  - cmd/profile.go
  - cmd/snapshot.go
  - cmd/sync.go
  - cmd/uninstall.go
  - cmd/deploy_test.go
  - cmd/oplog_integration_test.go
  - tasks/cli/agtaqm-test-coverage-gaps.md
---

## Additional test coverage gaps beyond TUI audit

### Objective

Sibling task `agtaqm` (`tasks/cli/agtaqm-test-coverage-gaps.md`) closes the same
classes of test-coverage gap (error paths, nil/empty-input guards, dry-run
no-side-effect, OpLog side effects, symlink target/cleanup) but is scoped
**exclusively** to `internal/tui/` screens. This task closes the identical gap
classes in the **non-TUI** packages (`internal/deploy`, `internal/state`,
`internal/sourcemanager`, `internal/config`, and the `cmd/` layer) so coverage
is consistent across the codebase. No item below overlaps `agtaqm`'s TUI-only
checklist (verified: `agtaqm` touches only `internal/tui/*` and
`internal/oplog`; nothing here touches `internal/tui`).

Why: these paths handle persistence failure, input validation, filesystem-
mutation safety (dry-run), and audit logging. Regressions are silent (wrong
behavior, lost data, no audit trail), so they need explicit tests.

All file:line references below were re-verified against the working tree on
2026-05-17 and corrected where they had drifted.

### Patterns to follow

Two test harnesses already exist — mirror the one matching the layer:

1. **Engine-level (`internal/deploy`)**: `mockStore` in
   `internal/deploy/deploy_test.go:19-60` implements `deploy.StateStore`. It
   has unused-but-wired fields `loadErr`, `saveErr`, `lockErr`
   (`internal/deploy/deploy_test.go:24-26`); setting `saveErr`/`loadErr`/
   `lockErr` makes `Save`/`Load`/`WithLock` return that error
   (`internal/deploy/deploy_test.go:36-59`). `testAgent()`
   (`:62-75`) builds a usable `*agent.Agent`. Build an `Engine` with
   `deploy.New(store, testAgent(), backupDir)` and inject filesystem fakes via
   `e.SetSymlink/SetRemove/SetLstat/...` (`internal/deploy/deploy.go:64-89`).
   Look at existing tests in `internal/deploy/deploy_test.go` for the exact
   `DeployRequest`/`RemoveRequest` construction shape.

2. **Cmd-level (`cmd/`)**: drive the real Cobra command. Helper
   `setupDeployEnv(t)` at `cmd/deploy_test.go:14-38` creates a temp config with
   a local source `my-source` pre-registered and two seed assets: a skill named
   **`greeting`** (`<src>/skills/greeting/SKILL.md`) and a command
   **`commands/hello.md`** (`<src>/commands/hello.md`). Run via
   `app := &App{}; rootCmd := NewRootCmd(app); rootCmd.SetArgs([]string{"--config", configPath, ...}); rootCmd.Execute()`
   (see `cmd/deploy_test.go:40-60`). For oplog assertions use the existing
   helpers in `cmd/oplog_integration_test.go`: `logDir(configPath)`
   (`:15-17`, returns `<configDir>/logs`) and `readLogEntries(t, dir)`
   (`:20-42`, JSON-decodes every line of `operations.log`). Existing models to
   copy: `TestOplog_DeploySingleWritesEntry`
   (`cmd/oplog_integration_test.go:44`) and `TestOplog_DryRunDoesNotLog`
   (`:160`). `App.LogOp` writes to `<configDir>/logs/operations.log`
   (`cmd/app.go:170-183`).

### Tasks

#### Error-path gaps (engine / store / git / source-remove)

- [ ] `internal/deploy/deploy.go:186` (`Deploy` → `e.store.Save`),
  `:501` (`DeployBulk` → `e.store.Save`), `:521` (`Remove` →
  `e.store.Save`), `:548` (`RemoveBulk` → `e.store.Save`): set
  `store.saveErr = errors.New("disk full")` and assert each public method
  returns a non-nil error wrapping it and persists nothing
  (`store.saved == nil`). One test per call site.
- [ ] `internal/state/store.go:88-90` (`Save` MkdirAll failure → "create state
  directory"), `:92-95` (`yaml.Marshal` failure → "marshal state"),
  `:102-104` (`WithLock` lock-acquire failure). Drive `Store` directly:
  point `NewStore` at a path whose parent dir cannot be created (e.g. a
  path under an existing regular file) for the MkdirAll branch; for
  WithLock, create the `.lock` file held by an un-releasable lock or use a
  0-timeout path (inspect `NewFileLock`/`FileLock.Acquire` in
  `internal/state/` to pick the cleanest trigger). Marshal failure may be
  hard to force with the real `DeploymentState`; if no input makes
  `yaml.Marshal` fail, strike the marshal sub-item with that one-line note.
- [ ] `internal/sourcemanager/git.go:62-64` (`gitClone` non-zero exit →
  `git clone ...: %w`) and `:72-74` (`gitPull` non-zero exit). Exercise via
  a bogus URL/dir so `git` exits non-zero; assert the wrapped error.
  `gitClone` is reached through `SourceManager.AddGit`
  (`internal/sourcemanager/register.go:154`).
- [ ] `internal/sourcemanager/register.go:154-157` -- `AddGit` clone-failure
  path: `gitClone` error → `os.RemoveAll(cloneTarget)` cleanup → returns
  the error; assert `cloneTarget` does not exist and config is unchanged.
  `:170-174` -- `WriteConfig` failure after a successful clone: rolls
  `sm.cfg.Sources` back to `oldSources`, removes `cloneTarget`, returns
  "save config: %w". Force `WriteConfig` failure by making `sm.configPath`
  unwritable (e.g. a directory at that path); assert in-memory sources
  reverted and clone dir removed.
- [ ] `cmd/source.go:341-374` -- `removeSourceDeployments` is currently 0%
  covered. Cover: (a) `eng.Status()` error return (`:343-345`);
  (b) `eng.RemoveBulk` error (`:366-368`); (c) partial failure where
  `result.Failed` is non-empty → `"failed to remove %d assets"`
  (`:370-372`); (d) the no-matching-deployments early return
  (`:362-364`). Reach it through `nd source remove <id> --yes` with prior
  deployments from that source (see the symlink-cleanup item below for the
  flow), or unit-test `removeSourceDeployments` directly with a fake
  engine.

#### Nil/empty-input guard gaps

- [ ] `internal/deploy/deploy.go:556-580` -- `Engine.SetOrigin` is 0% covered.
  Add tests for all four branches: (a) no matching deployment →
  `"deployment not found: ..."` (`:578`); (b) project-scope mismatch
  (`d.ProjectPath != projectRoot`) continues past a same-identity entry
  (`:570-572`); (c) a match updates `Origin` and calls `e.store.Save`
  (`:573-574`) — assert the persisted `Deployment.Origin` changed;
  (d) `e.store.Load` error returns `"load state: %w"` (`:561-563`) via
  `store.loadErr`.
- [ ] `internal/deploy/deploy.go:596-604` -- `removeOne` agent-filtering
  branch: when `req.Agent != ""`, a deployment whose `Agent` differs is
  skipped, and an empty `d.Agent` is treated as `"claude-code"`. Add a test
  with two deployments differing only by `Agent`, call `Remove`/`RemoveBulk`
  with `RemoveRequest.Agent` set, and assert only the matching one is
  removed; include the empty-`d.Agent` ⇒ `"claude-code"` compat case.
- [ ] `internal/config/validation.go:71-74` -- a `SourceEntry` with empty `ID`
  produces a `sources[i].id: must not be empty` `ValidationError`. `:17-22`
  -- `ValidationError.Error()`: assert the **no-file** branch (`File == ""`
  → `field %s: %s`) and the with-file branch (`File != "" → %s:%d: ...`).
  Build `config.Config` / `config.ValidationError` values directly and
  assert `.Validate()` / `.Error()` output.

#### Dry-run no-side-effect gaps (cmd layer)

Existing dry-run tests only assert the output text contains `"dry-run"`. Add
filesystem/state assertions (no symlink created, `deployments.yaml` unchanged,
config unchanged) to:

- [ ] `cmd/deploy_test.go:96` `TestDeployCmd_DryRun` -- after `--dry-run deploy
  greeting`, assert no symlink exists under the agent dir and the state file
  is absent/empty.
- [ ] `cmd/remove_test.go:60` `TestRemoveCmd_DryRun` -- assert the deployed
  symlink and its state entry still exist after `--dry-run remove`.
- [ ] `cmd/snapshot_test.go:250` `TestSnapshotRestoreCmd_DryRun` -- assert
  live deployment state is unchanged by the dry-run restore.
- [ ] `cmd/sync_test.go:33` `TestSyncCmd_DryRun` -- assert state file unchanged.
- [ ] `cmd/uninstall_test.go:12` `TestUninstallCmd_DryRun` -- assert symlinks
  and state still present.
- [ ] `cmd/profile_test.go:273` `TestProfileDeployCmd_DryRun` -- assert no
  symlink/state change.
- [ ] **New test**: profile-**switch** dry-run has no existing test
  (`cmd/profile_test.go` has `TestProfileSwitchCmd_*` at `:419/:535/:552/
  :604` but none for `--dry-run`). The guard is`cmd/profile.go:474-486`(prints`[dry-run] would ...`, returns before`app.DeployEngine()`at`:526`). Add`TestProfileSwitchCmd_DryRun`: with two profiles, run`--dry-run profile switch <name>` and assert no symlink/state change and
  output contains `[dry-run] would switch`.

#### OpLog side-effect gaps (cmd layer)

Use `readLogEntries(t, logDir(configPath))` (`cmd/oplog_integration_test.go:
15-42`). Existing coverage already in `cmd/oplog_integration_test.go`:
`OpDeploy` single/bulk (`:44`, `:74`), `OpRemove` (`:101`), `OpSync` (`:136`,
the full-reconcile `cmd/sync.go:92`), dry-run no-log for plain deploy (`:160`),
`OpSnapshotSave` (`:181`). Add the **missing** ones:

- [ ] `OpProfileSwitch` -- emitted at `cmd/profile.go:546-553` (op constant
  `cmd/profile.go:548`). Assert one entry with `Operation ==
  oplog.OpProfileSwitch` and `Detail == "<from> -> <to>"`.
- [ ] profile-deploy `OpDeploy` -- emitted at `cmd/profile.go:371-373` for
  `nd profile deploy <name>` (distinct from plain `nd deploy`); assert an
  `OpDeploy` entry results.
- [ ] `OpSnapshotRestore` -- emitted at `cmd/snapshot.go:194-196`; assert one
  entry after `nd snapshot restore`.
- [ ] `OpSourceAdd` -- emitted at `cmd/source.go:80-85` (git) and `:103-108`
  (local); assert an `OpSourceAdd` entry with the expected `Detail`
  (`"local <id>"` / `"git <id>"`) after `nd source add`.
- [ ] `OpSourceRemove` -- emitted at `cmd/source.go:237-242`; assert one entry
  after `nd source remove <id> --yes`.
- [ ] `OpSourceSync` -- emitted at `cmd/sync.go:48-50` (per-git-source path of
  `nd sync`; **not** the `OpSync` at `cmd/sync.go:92` which is already
  covered by `TestOplog_SyncWritesEntry`). Add a git source so this branch
  runs, then assert an `OpSourceSync` entry.
- [ ] `OpUninstall` -- emitted at `cmd/uninstall.go:90-92`; assert one entry
  after a non-dry-run `nd uninstall`.
- [ ] Dry-run no-log: mirror `TestOplog_DryRunDoesNotLog`
  (`cmd/oplog_integration_test.go:160`) for `remove`, `snapshot restore`,
  `profile deploy`, `profile switch`, `sync`, and `uninstall` — each with
  `--dry-run` must leave `operations.log` with zero entries (verify the
  dry-run guard precedes the `LogOp` call: e.g. `cmd/uninstall.go:48` guard
  before `:90` log; `cmd/profile.go:340` before `:371`;
  `cmd/profile.go:474` before `:546`; `cmd/snapshot.go:146` before `:194`;
  `cmd/sync.go:42`/`:66` before `:48`/`:90`).

#### Symlink target/cleanup gaps

- [ ] `internal/deploy/deploy.go:254-258` -- relative-symlink branch: when
  `req.Strategy == nd.SymlinkRelative` and `filepath.Rel` returns an error,
  `deployOne` returns `"compute relative path from %s to %s: %w"`.
  `filepath.Rel` errors when one path is absolute and the other relative;
  construct a `DeployRequest` with `Strategy: nd.SymlinkRelative` and a
  relative `Asset.SourcePath` (link path is absolute) so `Rel` fails, and
  assert the wrapped error. If a deterministic trigger is not reachable,
  strike with a one-line note.
- [ ] `internal/deploy/deploy.go:371-374` (foreign **symlink** + `ForceReplace`,
  `e.remove` fails → "remove conflicting symlink at %s: %w") and
  `:389-392` (plain **file** + `ForceReplace`, `e.remove` fails → "remove
  conflicting file at %s: %w"). Inject `e.SetLstat` returning a
  symlink/regular `fakeFileInfo` (`internal/deploy/deploy_test.go:82-90`),
  `e.SetReadlink`, and `e.SetRemove` returning an error; set
  `DeployRequest.ForceReplace = true`; assert the wrapped error.
- [ ] **New test** (no existing coverage): deploy assets from a source, then
  remove the source so its deployments are purged, and assert the symlinks
  AND state entries are gone. NOTE: there is **no `--purge` flag** anywhere
  in the codebase (verified by repo-wide grep). The actual mechanism is
  `cmd/source.go`: `nd source remove <id> --yes` (or the interactive
  "Remove source and all deployed assets" prompt choice at
  `cmd/source.go:185-211`) → `removeSourceDeployments(eng, sourceID)` at
  `cmd/source.go:201` / `:213` → `eng.RemoveBulk` (`cmd/source.go:366`),
  then `sm.Remove(sourceID)` (`cmd/source.go:233`). Existing
  `TestSourceRemove_WithYes` (`cmd/source_test.go:369`) only checks output
  text and deploys nothing first — extend it or add a new test: deploy
  `greeting` from a source via `nd deploy`, run `nd source remove <id>
  --yes`, assert (a) the symlink under the agent dir is gone and (b) no
  state entry for that source remains in `deployments.yaml`.

### Acceptance criteria

- Every non-struck checklist item has at least one new `Test*` function (in the
  appropriate `_test.go` for that package) exercising the cited code path; each
  struck item keeps its one-line N/A rationale and adds no production code.
- `mockStore.saveErr` (and `loadErr` where called for) is set by at least one
  test per `Save` call site (`internal/deploy/deploy.go:186,501,521,548`).
- `internal/deploy.Engine.SetOrigin` and `cmd.removeSourceDeployments` rise
  from 0% to non-zero statement coverage; overall package statement coverage
  for `internal/deploy`, `internal/state`, `internal/sourcemanager`,
  `internal/config`, and `cmd` does not decrease (record
  `go test -cover ./internal/deploy/... ./internal/state/...
  ./internal/sourcemanager/... ./internal/config/... ./cmd/...` before/after).
- `go build -o nd .` succeeds.
- `go test ./internal/deploy/... ./internal/state/...
  ./internal/sourcemanager/... ./internal/config/... ./cmd/...` passes,
  including the new tests, and the same with `-race`.
- No existing test is deleted or weakened to make new tests pass.
- No `internal/tui/*` test is added or modified (that scope belongs to
  `agtaqm`); no production source file is changed.

### References

- GitHub issue: https://GitHub.com/armstrongl/nd/issues/130
- Close this issue when the task is completed.
- Sibling/seed task: `agtaqm` -- `tasks/cli/agtaqm-test-coverage-gaps.md`
  (TUI-only; this task extends, does not duplicate it).
- Engine + `mockStore` harness: `internal/deploy/deploy.go`,
  `internal/deploy/deploy_test.go:19-90`.
- State store: `internal/state/store.go:86-107`.
- Source manager: `internal/sourcemanager/git.go`,
  `internal/sourcemanager/register.go`.
- Config validation: `internal/config/validation.go`.
- Source remove flow: `cmd/source.go:126-374`.
- OpLog cmd-test harness: `cmd/oplog_integration_test.go:15-42`;
  `cmd/app.go:170-183` (`OpLog`/`LogOp`, log dir `<configDir>/logs`).
- Cmd test env + seed assets (`greeting`, `commands/hello.md`):
  `cmd/deploy_test.go:14-38`.
- Dry-run guards: `cmd/profile.go:340,474`; `cmd/snapshot.go:146`;
  `cmd/sync.go:42,66`; `cmd/uninstall.go:48`.
- OpLog API: `internal/oplog/oplog.go`, `internal/oplog/writer.go`.
