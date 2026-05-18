---
title: "Normalize yes/json/quiet/non-TTY handling across commands"
id: "wdisqq"
status: pending
priority: medium
type: bug
tags: ["cli", "ux"]
created_at: "2026-05-17"
dependencies: ["ce0bw6", "nxvr4w", "ba5xah"]
context:
  - "cmd/source.go"
  - "cmd/export.go"
  - "cmd/init_cmd.go"
  - "cmd/root.go"
  - "cmd/snapshot.go"
  - "cmd/remove.go"
  - "cmd/uninstall.go"
  - "cmd/settings.go"
  - "cmd/profile.go"
  - "cmd/deploy.go"
  - "cmd/helpers.go"
  - "internal/output/json.go"
  - "cmd/source_test.go"
  - "cmd/deploy_test.go"
  - "tests/integration/flags_test.go"
verify:
  - type: bash
    run: "go build -o nd ."
  - type: bash
    run: "go test ./cmd/..."
  - type: bash
    run: "go test -race ./cmd/..."
  - type: bash
    run: "golangci-lint run ./cmd/..."
  - type: bash
    run: "go test ./tests/integration/ -v -run 'Flag|JSON'"
  - type: assert
    check: "Piped (non-TTY) `nd source remove <id>` for a source with deployed assets returns the actionable 'use --yes to remove' error, not a raw 'interactive choice required but stdin is not a terminal' error."
  - type: assert
    check: "`nd export --json` (or non-TTY) with missing --name/--assets returns the '--name is required' / '--assets is required' error and never launches huh forms; with --json it does not write non-JSON text to stdout."
  - type: assert
    check: "`nd init` run non-interactively (piped stdin) WITHOUT --yes does NOT deploy built-in assets; with --json a failed builtin deploy is surfaced (not reported as status:\"ok\")."
  - type: assert
    check: "`nd remove <pinned-asset> --dry-run` prints the dry-run plan and never prompts."
  - type: assert
    check: "Under --json, no command writes human (non-JSON) text to stdout (cancellation/aborted lines go to stderr or are suppressed); under --quiet, cancellation messages are uniformly suppressed."
  - type: assert
    check: "`nd settings edit --json` / `--quiet` / non-TTY does not exec $EDITOR (does not hang a scripted run); it returns an actionable error instead."
---

## Normalize --yes/--json/--quiet/non-TTY handling across commands

### Objective

Make every `cmd/` command behave consistently for the four non-interactive
signals: `--yes` (`app.Yes`), `--json` (`app.JSON`), `--quiet` (`app.Quiet`),
and non-TTY stdin (`!isTerminal()`). Today this handling is ad-hoc: some
commands cleanly skip prompts and return an actionable error, while others drop
into an interactive prompt that errors with an opaque message, auto-act on a
prompt error, or write human text to stdout under `--json` (which corrupts the
JSON stream for scripted callers).

This is a **pattern-expansion** task. The three dependency seed tasks each fix
one command and assume the rest of the CLI already follows the same
non-interactive contract; this task brings every remaining command up to that
contract so the seeds compose cleanly:

- **ce0bw6** (`tasks/cli/ce0bw6-init-shell-completions.md`, status: pending) —
  adds an opt-in completion prompt to `nd init` that must be skipped under
  `--yes`/`--json`/`--quiet`/non-TTY. Assumes `nd init`'s non-interactive
  contract is otherwise correct.
- **nxvr4w** (`tasks/cli/nxvr4w-deploy-scope-prompt.md`, status: pending) —
  adds a deploy scope prompt that must not block `--json`/non-TTY callers.
  Assumes the deploy/remove "JSON-then-non-TTY guard" pattern is the norm.
- **ba5xah** (`tasks/cli/ba5xah-select-deploy-agents.md`, status: pending) —
  adds a multi-agent picker that follows the same skip rules.

These dependencies are listed so the work is sequenced after the canonical
patterns land; this task does not modify those commands' new prompts, only the
surrounding consistency issues in other commands.

#### The canonical "deploy/remove pattern" to mirror

The reference implementation is the no-args interactive picker guard used by
`nd deploy` and `nd remove`. Verified at **`cmd/deploy.go:58-64`** and
**`cmd/remove.go:39-45`**:

```go
if len(args) == 0 {
    if app.JSON {
        return fmt.Errorf("requires at least one asset argument; run 'nd list --json' to see available assets")
    }
    if !isTerminal() {
        return fmt.Errorf("requires at least one asset argument; run 'nd list' to see available assets")
    }
    // ... only now call promptChoice / huh form ...
}
```

Rule: **check `app.JSON` first, then `!isTerminal()`, and return an actionable
error BEFORE entering any prompt/form.** Errors returned from `RunE` are printed
to stderr and mapped to an exit code by `Execute()` (`cmd/root.go:103-110`), so
a returned error does not corrupt `--json` stdout. The corrupting bug is
**human text written to stdout (`cmd.OutOrStdout()`) under `--json`** — those
sites must be guarded with `!app.JSON` (and routed to stderr or suppressed) or,
where an envelope is expected, emitted via `printJSONError`
(`cmd/helpers.go:36`, builds an `output.JSONError`, see `internal/output/json.go`).

#### Relevant helpers (verified)

- `confirm(r io.Reader, w io.Writer, prompt string, yesFlag bool) (bool, error)`
  — `cmd/helpers.go:58-73`. Returns `(true, nil)` immediately if `yesFlag`;
  returns an **error** (`"confirmation required but stdin is not a terminal
  (use --yes to skip)"`) when `!isTerminal()` and `!yesFlag`.
- `promptChoice(r, w, prompt, choices)` — `cmd/helpers.go:77-97`. Returns the
  error `"interactive choice required but stdin is not a terminal"` when
  `!isTerminal()`. This is the opaque error users hit today; a pre-check must
  return a better message first.
- `isTerminal()` — `cmd/helpers.go:115-117`. Global, reads `os.Stdin`. Not
  injectable; in unit tests it returns `false` (no PTY). Tests can force
  deterministic non-TTY by swapping `os.Stdin` with a pipe — see the existing
  pattern in `cmd/root_test.go:168-170`.
- `printJSONError(w, []output.JSONError)` — `cmd/helpers.go:36-48`.
  `output.JSONError{Code, Message, Field}` is defined in
  `internal/output/json.go:12-16`.
- App flag fields: `App.Yes`, `App.JSON`, `App.Quiet`, `App.DryRun`,
  `App.Verbose` — `cmd/app.go:25-30`. Registered as persistent flags in
  `cmd/root.go:58-63`.

### Tasks

Line numbers below were re-verified against the working tree. Each item names
the exact site and the change.

- [ ] **`cmd/source.go:181`** — The 3-way `promptChoice` block is
  `if deployedCount > 0 && !app.Yes { ... }` at line 181; the JSON guard at
  `cmd/source.go:182-184` already returns
  `"source %q has %d deployed assets; use --yes to remove"`. There is **no
  non-TTY guard**, so a piped call falls into `promptChoice`
  (`cmd/source.go:190-194`) and dies with the opaque
  `"interactive choice required..."` error. Add a `!isTerminal()` check
  immediately after the JSON guard (before line 185's `choices := ...`) that
  returns the same actionable `"...use --yes to remove"` error (mirror
  `cmd/deploy.go:62`).
- [ ] **`cmd/source.go:206-210`** — In the `promptChoice` "Cancel" branch
  (`case choices[2]:`), `printHuman(w, "Cancelled.\n")` (line 208) is guarded
  by `!app.Quiet` but not by `!app.JSON`. Reached only on a TTY (promptChoice
  errors otherwise), so JSON corruption is unlikely here, but for consistency
  also gate it with `&& !app.JSON` and route nothing to stdout under `--json`.
- [ ] **`cmd/export.go:57-59`** — `if isTerminal() && (!hasName || !hasAssets)
  { return runExportInteractive(...) }`. Under `--json` on a TTY this still
  enters the huh form (`isTerminal()` is true), then the form may
  `printHuman` to stdout, corrupting JSON. Change the condition to also require
  `!app.JSON` so a `--json` invocation falls through to the explicit
  `"--name is required"` / `"--assets is required"` errors at
  `cmd/export.go:62-67` (mirror the JSON-first rule from `cmd/deploy.go:59`).
- [ ] **`cmd/export.go:474-552`** — `runExportInteractive` launches huh forms
  unconditionally (`assetForm` at `:475`, `metaForm` at `:493`, `confirmForm`
  at `:537`, "Export cancelled." printed to stdout at `:550`). Add an early
  guard at the top of `runExportInteractive` (after `w := cmd.OutOrStdout()`,
  `cmd/export.go:435`): if `app.Yes || app.JSON || !isTerminal()`, do not run
  forms — require `flagName != ""` and `len`/non-empty `flagAssets`, returning
  the same `"--name is required"` / `"--assets is required"` errors used by the
  non-interactive path (`cmd/export.go:62-67`); only run the forms when
  genuinely interactive. (Note: `runExportInteractive`'s assets arg is
  `flagSource`/asset selection via the form; the caller passes the original
  flag values, so the guard can re-use `flagName`/the parsed `--assets`.)
- [ ] **`cmd/init_cmd.go:193-202`** — `shouldDeploy := app.Yes || app.JSON`
  (line 193); when neither is set, `confirm(...)` is called (line 195) and on
  error (non-TTY) `shouldDeploy = true` (line 198) — i.e. a non-interactive
  `nd init` **auto-deploys built-ins**. Per ce0bw6 the builtin deploy is
  opt-in. Change the `if err != nil` branch (`cmd/init_cmd.go:196-201`) so a
  non-TTY confirm error is treated as **decline** (`shouldDeploy = false`),
  not auto-deploy. `--yes` and `--json` still deploy (they short-circuit at
  line 193). Update the "Skipped." message guard at `cmd/init_cmd.go:204-208`
  if needed (it already checks `!app.Quiet && !app.JSON`).
- [ ] **`cmd/init_cmd.go:58-70`** — The `--json` branch builds the result map
  and `return printJSON(w, result, false)` (line 67) **without ever consulting
  `deployErr`**; `deployErr` is only returned on the non-JSON path (line 70).
  A failed builtin deploy under `--json` is silently reported as
  `status:"ok"`. Fix: when `deployErr != nil` and `app.JSON`, surface it —
  either return `deployErr` (after building/printing nothing to stdout) or, to
  keep an envelope, call `printJSONError(w, []output.JSONError{{Code:
  "builtin_deploy_failed", Message: deployErr.Error()}})` and return a non-nil
  error so the exit code is non-zero. Do not emit `status:"ok"` when
  `deployErr != nil`.
- [ ] **`cmd/root.go:209-242`** — `offerInit` has no `--json` handling. When
  `app.JSON` and config is missing for a command that needs init, it prints
  human text to `cmd.ErrOrStderr()` (stderr, so JSON stdout is not corrupted)
  and may run `runInitSetup`/`deployBuiltinAssets`. Add, near the top of
  `offerInit` (after the `app.DryRun` block at `cmd/root.go:210-214`): if
  `app.JSON`, do not auto-init or print human prose to stdout — return an
  actionable error such as `fmt.Errorf("nd is not initialized; run 'nd init'
  first")` (errors go to stderr via `Execute()`, leaving `--json` stdout
  clean) instead of silently continuing. Keep the existing interactive/`--yes`
  behavior (`cmd/root.go:219-238`) unchanged.
- [ ] **`cmd/snapshot.go:163-166`** — `if !ok { printHuman(w, "Restore
  cancelled.\n"); return nil }`. Not guarded by `!app.Quiet` or `!app.JSON`.
  Gate with `if !app.Quiet && !app.JSON` (match `cmd/profile.go:187-191` style;
  note this path is only reachable on a TTY since `confirm` errors otherwise,
  but `--yes` returns `ok=true` so the message is dead code under `--yes` —
  still add the guard for consistency).
- [ ] **`cmd/snapshot.go:332-335`** — `if !ok { printHuman(w, "Cancelled.\n");
  return nil }` in `newSnapshotDeleteCmd`. Same fix: gate with
  `if !app.Quiet && !app.JSON`.
- [ ] **`cmd/profile.go:187-191`** (`Delete cancelled.`) and
  **`cmd/profile.go:514-519`** (`Switch cancelled.`) — both already guard with
  `!app.Quiet` but not `!app.JSON`. Add `&& !app.JSON` to each guard for
  consistency (low risk: only reachable on a TTY). Verify the verified line
  for the `printHuman` call is `cmd/profile.go:189` (Delete) and
  `cmd/profile.go:516` (Switch).
- [ ] **`cmd/uninstall.go:60`** — `confirm(os.Stdin, w, ...)` passes the raw
  `os.Stdin` instead of `cmd.InOrStdin()`. Change to `cmd.InOrStdin()` so
  tests/scripted callers can inject input consistently with every other
  command (all other `confirm` call sites use `cmd.InOrStdin()`, e.g.
  `cmd/remove.go:81`, `cmd/profile.go:180`). Add `"os"` import cleanup if it
  becomes unused (it is still used elsewhere — verify with `go build`).
- [ ] **`cmd/uninstall.go:66-69`** — `if !ok { printHuman(w, "Aborted.\n");
  return nil }`. Not guarded. Gate with `if !app.Quiet && !app.JSON` (under
  `--json` the no-deployments / dry-run paths at `cmd/uninstall.go:40-57`
  already use `printJSON`; the abort path must not write plain text to
  stdout).
- [ ] **`cmd/settings.go:49-61`** — `newSettingsEditCmd` builds and runs
  `exec.Command(editor, configPath)` wired to `os.Stdin/Stdout/Stderr`
  (`cmd/settings.go:57-61`) unconditionally (only `--dry-run` is handled, at
  `:44-47`). A scripted/`--json`/`--quiet`/non-TTY caller would hang on an
  interactive editor. Add a guard before line 49 (after the dry-run block):
  if `app.JSON || app.Quiet || !isTerminal()`, return an actionable error like
  `fmt.Errorf("settings edit requires an interactive terminal; edit %s
  directly or use --dry-run", configPath)` instead of exec-ing the editor.
- [ ] **`cmd/profile.go:76-115`** — `newProfileCreateCmd`: `if fromCurrent
  { ... } else if assets != "" { ... }`. When both `--from-current` and
  `--assets` are passed, `--assets` is silently ignored. Flags registered at
  `cmd/profile.go:130-132`. Either mark them mutually exclusive via
  `cmd.MarkFlagsMutuallyExclusive("from-current", "assets")` (added after the
  three `cmd.Flags()...` registrations, `cmd/profile.go:132`) — preferred,
  matches the existing `rootCmd.MarkFlagsMutuallyExclusive("verbose",
  "quiet")` pattern at `cmd/root.go:65` — or print a `warning:` to
  `cmd.ErrOrStderr()` when both are set. Prefer mutual exclusion.
- [ ] **`cmd/remove.go:80`** — `if d.Origin == nd.OriginPinned && !app.Yes {`
  prompts to re-confirm a pinned asset. Unlike the non-pinned confirm at
  `cmd/remove.go:97` (`if d.Origin != nd.OriginPinned && !app.DryRun {`), the
  pinned branch has **no `!app.DryRun` guard**, so `nd remove <pinned>
  --dry-run` prompts (and on non-TTY errors out) instead of just printing the
  plan. Add `&& !app.DryRun` to the condition at `cmd/remove.go:80` so
  `--dry-run` never prompts; the request still gets appended and the existing
  dry-run branch at `cmd/remove.go:128-136` prints the plan.
- [ ] **Tests** — Add a per-command non-interactive matrix. For each command
  touched above, assert: (a) non-TTY without `--yes` returns the actionable
  error (not the opaque `promptChoice`/`confirm` message), (b) `--json`
  produces only valid JSON on stdout (parseable with `json.Unmarshal`) or a
  clean returned error with empty stdout, (c) `--yes` skips prompts and
  succeeds, (d) `--quiet` suppresses cancellation/abort lines. Mirror existing
  patterns: `cmd/source_test.go:403-431`
  (`TestSourceRemove_NonTTY_NoYes_Errors` — build `App{}` + `NewRootCmd`,
  `SetArgs`, `Execute`, assert error), the `*_JSON` tests in
  `cmd/source_test.go` / `cmd/deploy_test.go:118`, and force deterministic
  non-TTY via the `os.Stdin` pipe swap in `cmd/root_test.go:168-170` where a
  real EOF is needed. Add at least one integration case to
  `tests/integration/flags_test.go` (use `setupIntegrationEnv` + `runND`, see
  `tests/integration/flags_test.go:8-20`, `34-50`) covering `nd settings edit
  --json` not hanging and `nd export --json` (missing flags) erroring without
  stdout noise.

### Acceptance criteria

- Piped `nd source remove <id>` for a source with deployed assets and no
  `--yes` returns `source "<id>" has N deployed assets; use --yes to remove`
  (exit non-zero), NOT `interactive choice required but stdin is not a
  terminal`.
- `nd export --json` (or piped, no TTY) with missing `--name`/`--assets`
  returns the `--name is required` / `--assets is required` error, never
  launches a huh form, and writes no non-JSON text to stdout.
- `nd init` with piped stdin and no `--yes` does NOT deploy built-in assets
  (prints the "Skipped." hint on the human path); `nd init --json` with a
  failing builtin deploy returns a non-zero exit and does NOT emit
  `status:"ok"`.
- `nd remove <pinned-asset> --dry-run` prints `[dry-run] would remove ...`
  and never prompts (works on non-TTY without error).
- `nd settings edit` under `--json`/`--quiet`/non-TTY returns an actionable
  error and does not exec `$EDITOR` (no hang).
- `nd profile create <name> --from-current --assets x` errors (mutually
  exclusive) instead of silently ignoring `--assets`.
- Under `--json`, no touched command writes human text to stdout; all
  cancellation/abort lines are routed to stderr or suppressed. Under `--quiet`,
  every cancellation/abort message is suppressed uniformly.
- `go build -o nd .`, `go test ./cmd/...`, `go test -race ./cmd/...`,
  `golangci-lint run ./cmd/...`, and `go test ./tests/integration/ -v -run
  'Flag|JSON'` all pass; all pre-existing `cmd/` and integration tests remain
  green.

### References

- Canonical pattern to mirror: `cmd/deploy.go:54-87` (JSON-then-non-TTY guard
  at `:58-64`), `cmd/remove.go:35-57` (`:39-45`).
- Helpers: `confirm` `cmd/helpers.go:58-73`; `promptChoice`
  `cmd/helpers.go:77-97`; `isTerminal` `cmd/helpers.go:115-117`; `printJSON`
  `cmd/helpers.go:22-33`; `printJSONError` `cmd/helpers.go:36-48`.
- JSON envelope: `internal/output/json.go:4-16` (`JSONResponse`, `JSONError`).
- Error→exit-code path: `cmd/root.go:99-111` (`Execute`),
  `cmd/root.go:262-294` (`withExitCode`/`exitCodeFromError`); usage-error
  code `nd.ExitInvalidUsage` used at e.g. `cmd/remove.go:74`.
- App flags: `cmd/app.go:20-32`; registration `cmd/root.go:54-65`
  (`MarkFlagsMutuallyExclusive` example at `:65`).
- Sites to change (verified): `cmd/source.go:181`,`:206-210`;
  `cmd/export.go:57-59`,`:474-552`; `cmd/init_cmd.go:58-70`,`:193-202`;
  `cmd/root.go:209-242`; `cmd/snapshot.go:163-166`,`:332-335`;
  `cmd/profile.go:76-115`,`:187-191`,`:514-519`; `cmd/uninstall.go:60`,
  `:66-69`; `cmd/settings.go:49-61`; `cmd/remove.go:80`.
- Test patterns: `cmd/source_test.go:403-431`
  (`TestSourceRemove_NonTTY_NoYes_Errors`); `*_JSON` tests in
  `cmd/source_test.go` / `cmd/deploy_test.go:118`; deterministic non-TTY swap
  `cmd/root_test.go:168-170`; integration scaffold
  `tests/integration/flags_test.go:8-50`.
- Seed tasks: `tasks/cli/ce0bw6-init-shell-completions.md` (pending),
  `tasks/cli/nxvr4w-deploy-scope-prompt.md` (pending),
  `tasks/cli/ba5xah-select-deploy-agents.md` (pending).
