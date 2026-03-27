---
title: "Pre-existing staticcheck warnings (QF1012, SA5011) fail CI on unrelated PR"
date: 2026-03-27
category: build-errors
module: CI/CD
problem_type: build_error
component: tooling
symptoms:
  - "golangci-lint v2.11.4 fails in GitHub Actions on docs-only PR #9 due to pre-existing staticcheck warnings"
  - "QF1012: WriteString(fmt.Sprintf(...)) should use fmt.Fprintf — 35+ instances across 9 TUI files"
  - "SA5011: possible nil pointer dereference in 6 test files — staticcheck does not track t.Fatal as noreturn"
  - "CI lint job blocks PR merge despite no Go code changes in the PR"
root_cause: config_error
resolution_type: code_fix
severity: medium
tags: [golangci-lint, staticcheck, qf1012, sa5011, ci-failure, pre-commit-hook, tui, github-actions]
---

# Pre-existing staticcheck warnings (QF1012, SA5011) fail CI on unrelated PR

## Problem

A docs-only PR (#9) failed CI linting because golangci-lint v2 runs against the entire codebase (`./...`), surfacing ~40 pre-existing staticcheck warnings across TUI source files and test files in multiple packages. No Go code was changed in the PR.

## Symptoms

- GitHub Actions "Lint" job failed on PR #9; Build and Test jobs passed
- `gh pr checks 9` confirmed: Lint fail, Build pass, Test pass
- golangci-lint v2.11.4 reported QF1012 warnings (unnecessary `WriteString(Sprintf(...))` allocations) across 9 TUI files
- SA5011 warnings (possible nil pointer dereference after `t.Fatal`) across 6 test files in 4 packages
- Fixing each batch revealed more warnings — the linter caps output per run

## What Didn't Work

- **Iterative fixing was required**: The initial CI failure showed only 4 errors in `internal/tui/browse.go`. Fixing those and re-running revealed 3 more in `deploy.go`. Fixing those surfaced 5 more across `doctor.go`, `status.go`, and `main_menu_test.go`. Then 6 more in `asset/` and `sourcemanager/` test files. Then 3 more in `scanner_test.go`. golangci-lint limits reported issues per run, so the true scope (~40 warnings across 15 files) was not apparent from the first failure. Each round required a full re-run to discover the next batch.
- **The PR changed zero Go files** — the lint failures had nothing to do with the PR's content. The debt accumulated silently because the linter was not running locally before commits or pushes.

## Solution

**QF1012: Replace `WriteString(fmt.Sprintf(...))` with `fmt.Fprintf` (~35 instances across 9 TUI files)**

```go
// Before (triggers QF1012 — allocates intermediate string):
buf.WriteString(fmt.Sprintf("  %s %s%s\n\n", arg1, arg2, arg3))

// After (writes directly to the buffer):
fmt.Fprintf(&buf, "  %s %s%s\n\n", arg1, arg2, arg3)
```

Note the `&buf`: when `buf` is a `strings.Builder` value (not pointer), you must take its address since `Fprintf` requires an `io.Writer`.

Files fixed: `browse.go`, `deploy.go`, `doctor.go`, `status.go`, `source.go`, `snapshot.go`, `profile.go`, `pin.go`, `remove.go` (all in `internal/tui/`).

**SA5011: Restructure nil-check-then-use patterns in tests (6 files across 4 packages)**

```go
// Before (triggers SA5011 — staticcheck doesn't model t.Fatal as noreturn):
got := someFunc()
if got == nil {
    t.Fatal("expected non-nil")
}
if got.Field != expected {  // SA5011: possible nil dereference
    t.Errorf(...)
}

// After — single assertion (else-if):
got := someFunc()
if got == nil {
    t.Fatal("expected non-nil")
} else if got.Field != expected {
    t.Errorf(...)
}

// After — multiple assertions (else block):
if got == nil {
    t.Fatal("expected non-nil")
} else {
    if got.Meta != nil {
        t.Error("Meta should be nil")
    }
    if got.ContextFile == nil {
        t.Fatal("ContextFile should be set")
    }
}
```

Files fixed: `internal/tui/main_menu_test.go`, `internal/asset/index_test.go`, `internal/asset/search_test.go`, `internal/profile/store_test.go`, `internal/sourcemanager/config_test.go`, `internal/sourcemanager/scanner_test.go`.

## Why This Works

**QF1012**: `buf.WriteString(fmt.Sprintf(...))` first allocates a formatted string on the heap, then copies it into the buffer. `fmt.Fprintf(&buf, ...)` writes directly into the buffer with no intermediate allocation. The replacement is semantically identical but eliminates unnecessary work.

**SA5011**: staticcheck performs dataflow analysis but does not model `testing.T.Fatal` (or `t.Fatalf`, `t.FailNow`) as functions that terminate execution. After seeing `if got == nil { t.Fatal(...) }`, the analyzer still considers the `got == nil` branch survivable, so any subsequent `got.Field` access is flagged as a possible nil dereference. Wrapping the access in an `else` clause makes mutual exclusivity explicit in the control flow graph.

**Why a docs PR triggered this**: golangci-lint in CI runs `./...` (the full module), not just changed files. The lint debt existed before the PR but had never been caught because the linter was not running locally.

## Prevention

**1. Git pre-commit hook** — runs golangci-lint before every commit that includes Go files:

```shell
#!/bin/bash
# .git/hooks/pre-commit

STAGED_GO=$(git diff --cached --name-only --diff-filter=ACM | grep '\.go$' || true)

if [ -z "$STAGED_GO" ]; then
  exit 0
fi

echo "Running golangci-lint..."
if ! golangci-lint run ./...; then
  echo ""
  echo "Lint failed. Fix the issues above before committing."
  exit 1
fi
```

**2. Claude Code PreToolUse hook** — intercepts `git push` commands and gates on clean lint:

```shell
#!/bin/bash
# .claude/hooks/lint-before-push.sh

COMMAND=$(echo "$TOOL_INPUT" | jq -r '.command // empty' 2>/dev/null)

if ! echo "$COMMAND" | grep -qE '\bgit\s+push\b'; then
  exit 0
fi

if ! golangci-lint run ./... 2>&1; then
  echo ""
  echo "BLOCKED: golangci-lint failed. Fix lint issues before pushing."
  exit 2
fi
```

**3. General strategies**:

- Run `golangci-lint run ./...` locally before opening any PR — CI lints the whole module regardless of what files changed.
- When fixing linter output that appears capped, re-run after fixing to check for more. golangci-lint may limit reported issues per run.
- For SA5011 specifically, prefer `else` clauses over sequential nil-guard-then-use in tests, even though `t.Fatal` logically terminates.
- For QF1012, use `fmt.Fprintf` whenever writing formatted output to a `strings.Builder`, `bytes.Buffer`, or any `io.Writer`.

## Related Issues

- No existing `docs/solutions/` docs (this is the first)
- No GitHub issues related to lint/staticcheck in this repo
- `.claude/docs/reference/go-cicd-tooling-best-practices.md` covers golangci-lint setup generally but not this specific incident
- `docs/specs/nd-go-spec.md` acceptance criteria #6 mentions golangci-lint with strict config
