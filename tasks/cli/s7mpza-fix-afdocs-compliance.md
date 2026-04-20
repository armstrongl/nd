---
title: "Fix AFDocs compliance check"
id: "s7mpza"
status: pending
priority: medium
type: bug
tags: ["ci", "docs"]
created_at: "2026-04-20"
---

## Fix AFDocs compliance check

### Objective

The AFDocs compliance check GitHub Action (`.github/workflows/afdocs-check.yml`) is failing with an overall score of 98/100. Two checks need to be addressed so the workflow passes and the auto-managed GitHub issue closes itself.

### Steps to reproduce

1. Push a docs change to `main` (triggers the "Deploy docs" workflow, which triggers the AFDocs check)
2. Alternatively, run the "AFDocs check" workflow manually via `workflow_dispatch`
3. Observe the workflow fails and creates/updates GitHub issue #104

### Expected behavior

The AFDocs check workflow passes (exit code 0) and auto-closes the tracking issue.

### Actual behavior

The workflow exits non-zero due to two findings:

1. **FAIL `markdown-content-parity`**: 16 of 50 pages have substantive content differences between their markdown and HTML versions (average 19% missing). Agents receiving the markdown version get outdated or incomplete content.
2. **WARN `llms-txt-directive`**: The `llms.txt` directive is found on all 50 sampled pages but is buried past the 50% mark. It should appear near the top of every documentation page.
3. **WARN `content-start-position`**: 15 of 50 pages have documentation content starting 10-50% into the converted output due to inline CSS or boilerplate.

### Tasks

- [ ] Run `afdocs check` locally to reproduce the failures and identify the affected pages
- [ ] Investigate Hugo build pipeline to determine why markdown and HTML content diverge on 16 pages
- [ ] Fix the markdown generation or source content so markdown versions match HTML (resolve `markdown-content-parity` FAIL)
- [ ] Move the `llms.txt` directive closer to the top of each documentation page template (resolve `llms-txt-directive` WARN)
- [ ] Reduce boilerplate/inline CSS that pushes content start past 10% (resolve `content-start-position` WARN)
- [ ] Re-run `afdocs check` locally to confirm all checks pass
- [ ] Push fix and verify the GitHub Actions workflow passes

### Acceptance criteria

- The AFDocs compliance check workflow passes with exit code 0
- `markdown-content-parity` check reports 0 pages with substantive differences
- `llms-txt-directive` shows the directive appears in the top 50% of every page
- GitHub issue #104 is auto-closed by the passing workflow run
- No regression in the overall score (must remain >= 98)

### References

- GitHub issue: https://GitHub.com/armstrongl/nd/issues/104
- Workflow file: `.github/workflows/afdocs-check.yml`
- AFDocs spec: https://agentdocsspec.com/spec/
