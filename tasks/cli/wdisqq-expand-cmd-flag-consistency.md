---
title: "Normalize yes/json/quiet/non-TTY handling across commands"
id: "wdisqq"
status: pending
priority: medium
type: bug
tags: ["cli", "ux"]
created_at: "2026-05-17"
dependencies: ["ce0bw6", "nxvr4w", "ba5xah"]
---

## Normalize --yes/--json/--quiet/non-TTY handling across commands

### Objective

Pattern expansion of seeds ce0bw6, nxvr4w, ba5xah, which all assume consistent flag/output-mode handling. Prompt-gating and JSON-envelope handling are inconsistent across `cmd/`: some commands skip prompts on `--json`/`--yes`/`--quiet`/non-TTY and emit actionable errors; others drop into prompts, auto-act, or corrupt `--json` stdout. Normalize to the deploy/remove pattern.

### Tasks

- [ ] `cmd/source.go:181-211` -- pre-check `!isTerminal()` before the 3-way `promptChoice`; return the "use --yes" guidance like deploy/remove
- [ ] `cmd/export.go:57` -- skip `runExportInteractive` and error when `app.JSON` (mirror `deploy.go:59`)
- [ ] `cmd/export.go:475-552` -- honor `--yes` (require `--name`/`--assets` or error instead of launching huh forms)
- [ ] `cmd/init_cmd.go:193-202` -- do NOT auto-deploy builtins on `confirm` error in non-TTY; treat non-TTY without `--yes` as decline (ce0bw6 opt-in)
- [ ] `cmd/init_cmd.go:58-71` -- `--json` path swallows `deployErr` and reports `status:"ok"`; surface the failure in the JSON envelope
- [ ] `cmd/root.go:219-225` -- emit a JSON error in `offerInit` when `app.JSON` and init is needed
- [ ] `cmd/snapshot.go:163-165` & `:332-334` -- gate cancellation messages with `if !app.Quiet` and a JSON guard (match `profile.go`/`source.go`)
- [ ] `cmd/source.go:206-230` -- "Cancel" branch returns nil with no JSON envelope under `--json`; emit proper JSON
- [ ] `cmd/profile.go:189,516` -- cancellation printed without a JSON guard; fix
- [ ] `cmd/uninstall.go:60` -- use `cmd.InOrStdin()` instead of `os.Stdin`; `cmd/uninstall.go:66-69` -- gate "Aborted." on `--json`/`--quiet`
- [ ] `cmd/settings.go:57-61` -- under `--json`/`--quiet`/non-TTY do not exec `$EDITOR` unconditionally (would hang scripted runs)
- [ ] `cmd/profile.go:76-115` -- `--from-current` silently ignores `--assets`; mark mutually exclusive or warn
- [ ] `cmd/remove.go:80-94` -- gate the pinned re-confirm with `!app.DryRun` so `--dry-run` never prompts
- [ ] Tests: per-command non-TTY / `--json` / `--yes` / `--quiet` matrix asserting no prompt + correct exit code + valid JSON

### Acceptance criteria

- Piped `nd source remove <id>` with deployed assets returns the actionable `--yes` error, not a raw picker error
- `nd export --json` with missing flags errors instead of launching forms
- `nd init` non-interactive without `--yes` does NOT deploy builtins; `--json` reports a failed builtin deploy as an error
- `nd remove <pinned> --dry-run` prints the plan without prompting
- No command writes non-JSON text to stdout under `--json`; cancellation messages suppressed under `--quiet` uniformly
- All `cmd/` tests pass

### References

- Seed tasks: ce0bw6 -- `tasks/cli/ce0bw6-init-shell-completions.md`; nxvr4w -- `tasks/cli/nxvr4w-deploy-scope-prompt.md`; ba5xah -- `tasks/cli/ba5xah-select-deploy-agents.md`
- `cmd/source.go`, `cmd/export.go`, `cmd/init_cmd.go`, `cmd/root.go`, `cmd/snapshot.go`, `cmd/remove.go`, `cmd/uninstall.go`, `cmd/settings.go`, `cmd/profile.go`, `cmd/helpers.go`
