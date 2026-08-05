---
title: "rumdl MD044 mangles bare GitHub URLs in task files"
id: "l7atoq"
status: pending
priority: low
type: bug
tags: ["lint", "rumdl", "tooling"]
created_at: "2026-08-05"
---

## Rumdl MD044 mangles bare GitHub URLs in task files

### Steps to reproduce

1. Create a task with a linked issue: `scripts/taskmd-issue-sync.py add "Title" --group ci --template bug`
2. The script appends a bare URL bullet under `### References` (see `ISSUE_BULLET_RE`,
   `scripts/taskmd-issue-sync.py:16`)
3. `git commit` — the rumdl pre-commit hook runs with `--fix`

### Expected behavior

The URL host is left alone. MD044 enforces proper-name casing in prose, not inside link
targets.

### Actual behavior

MD044 rewrites the host segment: `https://github.com/...` becomes `https://GitHub.com/...`.
46 task files under `tasks/` already carry the mangled host. It is cosmetic — DNS is
case-insensitive, so the links still resolve — but it is noise in every diff and it fights the
sync script on every task creation.

### Root cause

MD044 only rewrites **bare** URLs. Testing against rumdl 0.1.69 shows angle-bracket autolinks
(`<https://github.com/a/b>`), inline links (`[text](https://github.com/a/b)`), and code spans
are all left untouched; only the bare form is matched.

The repo disables MD034 (`extend-disable` in `.rumdl.toml`), which is what lets bare URLs
through in the first place. `code-blocks = false` under `[MD044]` does not help — it does not
cover link targets.

### Fix options

1. **Change the emitted form** (preferred): have `scripts/taskmd-issue-sync.py` write
   `- GitHub issue: <https://github.com/…>` or an inline link, and widen `ISSUE_BULLET_RE`
   (`scripts/taskmd-issue-sync.py:16`) and `find_issue_url` to accept both the bare and
   wrapped forms so the 46 existing files keep parsing.
2. **Exempt the path**: add `"tasks/**" = ["MD044"]` to `[per-file-ignores]` in `.rumdl.toml`.
   Cheapest, but it also drops proper-name checking on all task prose.
3. **Upstream**: file against [rumdl](https://github.com/rvben/rumdl) — MD044 arguably should
   never touch URL/link targets. Slowest path; do 1 regardless.

### Tasks

- [ ] Pick a fix option (default: 1)
- [ ] Update the reference bullet emitted by `update_task_references` in
      `scripts/taskmd-issue-sync.py`
- [ ] Widen `ISSUE_BULLET_RE` and `find_issue_url` to accept bare, angle-bracket, and inline
      link forms so existing files still parse
- [ ] One-time repair pass over the 46 affected files under `tasks/` (`grep -rl "GitHub.com" tasks/`)
- [ ] Verify `scripts/taskmd-issue-sync.py set <id> --done` still finds and closes the issue on
      a repaired file
- [ ] Confirm a fresh `add` survives the pre-commit hook with the URL intact
- [ ] Related: MD063 sentence-case capitalizes the lowercase tool name `rumdl` in headings —
      add it to `ignore-words` under `[MD063]` in `.rumdl.toml` alongside `nd`

### Environment

- OS: macOS (Darwin 25.5.0)
- Version: rumdl 0.1.69; nd on `main` as of 2026-08-05

### References

- GitHub issue: https://GitHub.com/armstrongl/nd/issues/152
- Close this issue when the task is completed.
