---
title: "Shell doc-lint script: broken pipelines, false positives in code blocks, and grep prefix artifacts"
date: 2026-03-27
category: logic-errors
module: Documentation Tooling
problem_type: logic_error
component: tooling
symptoms:
  - "'integer expected' errors in bash comparisons from pipeline producing 0\\n0"
  - "False positive warnings for patterns inside fenced code blocks"
  - "Forbidden-word heuristics never matching due to grep -n line-number prefix"
  - "Misleading script header comment overstating default file coverage"
root_cause: logic_error
resolution_type: code_fix
severity: medium
tags: [shell-scripting, pipefail, grep, awk, fenced-code-blocks, linting, pr-review]
---

# Shell doc-lint script: broken pipelines, false positives in code blocks, and grep prefix artifacts

## Problem

A custom documentation style linter (`scripts/lint-docs.sh`) contained five bugs caused by interactions between `set -eo pipefail`, missing fenced-code-block awareness, and `grep -n` line-number prefixes corrupting downstream filters. Bugs 1-2 were caught during local testing; bugs 3-5 were caught by Copilot PR review on PR #10.

## Symptoms

- `[: 0\n0: integer expected` arithmetic errors when scanning headings with no capitalized words
- False positive `+--` (tree notation) violations reported for lines inside fenced code blocks
- Forbidden-word warnings on lines inside code blocks and comments, because `grep -n` prefixes prevented content filters from matching
- False positive `--` (em-dash separator) violations inside fenced code blocks
- Header comment claiming "all docs + README" when only `docs/guide/` and root markdown files were checked

## What Didn't Work

- **`|| echo 0` pipeline guard**: `caps=$(... | grep -oE ... | wc -l || echo 0)` was meant to handle grep's exit code 1 (no matches), but under `set -eo pipefail` the pipeline was already dead before the `||` fired. The variable received `0\n0` (wc output + echo fallback) instead of `0`.
- **`grep -n` with downstream filters**: `grep -inw "$word" "$file" | grep -v '^\s*#'` compared against decorated strings like `42:some text` rather than the original line content, so patterns anchored to line-start (`^\s*#`, `^\s{4,}`) never matched.
- **Simple `grep -v '```'`**: Filtering out lines that literally contain triple-backticks does not skip content *between* fence markers. Lines inside a code block rarely contain `` ``` `` themselves.

## Solution

**Bug 1: Isolate grep's exit code so pipefail cannot abort the pipeline**

```shell
# Before (produces 0\n0):
caps=$(echo "$rest" | grep -oE '\b[A-Z][a-z]+' | wc -l | tr -d ' ' || echo 0)

# After:
caps=$(echo "$rest" | { grep -oE '\b[A-Z][a-z]+' || true; } | wc -l | tr -d ' ')
```

**Bug 2: Replace grep with awk that tracks fence state**

```shell
# Before (false positives in code blocks):
grep -n '+--' "$file"

# After:
awk 'BEGIN{fence=0} /^```/{fence=!fence; next} !fence && /\+--/{print NR": "$0}' "$file"
```

**Bug 3: Replace grep chain with single awk pass for forbidden words**

```shell
# Before (grep -n prefix breaks all downstream filters):
grep -inw "$word" "$file" | grep -v '^\s*#' | grep -v '```'

# After:
awk -v w="$word" '
  BEGIN { fence=0; IGNORECASE=1 }
  /^```/ { fence=!fence; next }
  fence { next }
  { for(i=1;i<=NF;i++) { gsub(/[^a-zA-Z]/, "", $i); if(tolower($i)==w) { print NR": "$0; next } } }
' "$file"
```

**Bug 4: Same awk fence-tracking pattern for em-dash check**

```shell
# Before (not fence-aware):
grep -nE '^\s*-.*\s--\s' "$file" | grep -v '```'

# After:
awk '
  /^```/ { fence=!fence; next }
  fence { next }
  /^[[:space:]]*-.*[[:space:]]--[[:space:]]/ { print NR": "$0 }
' "$file"
```

**Bug 5: Corrected header comment**

```shell
# Before:
# With no arguments, checks all docs + README.

# After:
# With no arguments, checks docs/guide/ plus root markdown files (README, CONTRIBUTING, ARCHITECTURE).
```

**Additional fixes**: Changed `bash` fences to `shell` in the solution doc (the very rule the linter enforces). Added explanatory comment in `.markdownlint-cli2.yaml` for the `docs/solutions/**` exclusion.

## Why This Works

**pipefail + grep**: `set -eo pipefail` causes any non-zero exit in a pipeline to abort the entire pipeline immediately. When `grep` finds no matches it exits 1, killing the pipeline before `wc -l` or `|| echo 0` can compensate. Wrapping grep in `{ grep ... || true; }` absorbs the non-zero exit *inside* the pipeline element, so downstream commands still receive correct input.

**Fence-state tracking**: `grep` processes lines individually with no state between them, making it structurally incapable of distinguishing content inside a fenced block from content outside one. `awk` maintains a `fence` toggle variable across lines, enabling checks to be skipped while inside a block.

**grep -n prefix corruption**: `grep -n` embeds the line number into the string it outputs (`42:text`). Any subsequent `grep -v '^...'` pattern compares against the decorated string, not the original line content. Moving all logic into a single `awk` program avoids the decorated-output problem entirely because awk accesses line content and line number (`NR`) independently.

## Prevention

- Treat `set -eo pipefail` as a first-class constraint: any command that legitimately exits non-zero (especially `grep` with no matches, `diff`, `test`) must be wrapped with `{ cmd || true; }` before it enters a pipeline.
- Prefer `awk` over chained `grep` pipelines when you need stateful processing (fence tracking, line-number emission paired with filtering, multi-condition logic).
- Never use `grep -n` output as input to further pattern filters unless downstream patterns explicitly account for the `linenum:` prefix.
- Include test cases that place forbidden patterns inside fenced code blocks; fence-awareness bugs are invisible without such fixtures.
- When adding exclusions to linter config files, always include an inline comment explaining the reason.

## Related Issues

- PR #10: https://github.com/armstrongl/nd/pull/10
- Related solution (same PR series, different problem): `docs/solutions/build-errors/golangci-lint-staticcheck-qf1012-sa5011-ci-failure-2026-03-27.md`
- Related solution (same PR series, different problem): `docs/solutions/documentation-gaps/pr-review-terminology-and-incomplete-normalization-2026-03-27.md`
