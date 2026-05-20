---
title: "git clone fails with --stdin error when adding source from GitHub URL"
id: "dilcbc"
status: completed
priority: high
type: bug
tags: ["git", "source"]
created_at: "2026-05-20"
completed_at: 2026-05-20
---

## Git clone fails with --stdin error when adding source from GitHub URL

### Steps to reproduce

1. Run `nd source add` (or equivalent) with a GitHub repo URL
2. Use URL: `https://github.com/armstrongl/larah-plugins`

### Expected behavior

nd should clone the repository successfully and register it as a source.

### Actual behavior

The git clone fails with the following error:

```
!! Error: git clone https://github.com/armstrongl/larah-plugins: Cloning into
fatal: --stdin requires a git repository
fatal: fetch-pack: invalid index-pack output
: exit status 128
```

The `--stdin requires a git repository` message suggests that `git clone` is being invoked incorrectly — possibly piping input via `--stdin` or passing arguments in an unexpected way to `git fetch-pack`.

### Objective

Fix the git clone invocation in the source-add workflow so that cloning from a GitHub HTTPS URL works correctly.

### Tasks

- [x] Reproduce the error locally with `nd source add <github-url>`
- [x] Trace the git clone call in the source management code (`internal/source/` or `internal/sourcemanager/`)
- [x] Identify why `--stdin` is being passed or why fetch-pack receives invalid input
- [x] Fix the clone command construction
- [x] Test with HTTPS GitHub URLs, SSH URLs, and local paths
- [x] Add a regression test for cloning from a remote URL

### Acceptance criteria

- `nd source add https://github.com/armstrongl/larah-plugins` clones successfully and registers the source
- Other source URL formats (SSH, local path) continue to work
- No `--stdin` or `fetch-pack` errors during clone
