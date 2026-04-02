# /nd-audit

Audit the user's nd setup for health issues, broken deployments, unavailable sources, and configuration problems. Produce a structured report with remediation steps.

## Accept arguments

Parse any arguments passed after `/nd-audit`:

- `--json`: Output the final report as a JSON object instead of markdown
- No arguments: Run the full audit with human-readable output

## Phase 1: run health check

Run:

```shell
nd doctor
```

Capture and parse the output. Record each check and its result (pass/fail/warning). The doctor command validates:

1. Config file validity
2. Source accessibility
3. Deployment health (broken symlinks, drift)
4. Agent detection
5. Git availability

If `nd doctor` itself fails to run, record that as a critical finding and continue with the remaining phases.

## Phase 2: collect deployment status

Run:

```shell
nd status --json
```

Parse the JSON output. For each deployed asset, record:

- Asset type and name
- Scope (global or project)
- Origin (manual, pinned, or profile name)
- Source
- Health status

Count totals: healthy deployments, broken deployments, pinned assets, profile-managed assets.

## Phase 3: check for broken symlinks

Using the deployment status data from Phase 2, identify any assets with unhealthy status. For each broken deployment, determine the cause:

- **Missing source**: The source directory or file no longer exists at the expected path
- **Deleted source**: The source was removed from nd but symlinks remain
- **Stale cache**: For builtin assets, the cache directory may need re-extraction

Run:

```shell
nd list --json
```

Cross-reference deployed assets against available assets to find orphaned deployments (deployed from sources that no longer provide them).

## Phase 4: check source availability

Run:

```shell
nd source list --json
```

For each source, verify:

- **Local sources**: The path exists and is accessible
- **Git sources**: The clone directory exists under `~/.config/nd/sources/`
- **Builtin source**: Present and functional

Record any sources that are unavailable or have issues.

## Phase 5: compile report

### Human-readable format (default)

Structure the report as follows:

```
nd audit report
===============

Summary
-------
  Sources:      N registered (M healthy, X unavailable)
  Deployments:  N total (M healthy, X broken)
  Pinned:       N assets
  Profile:      <active profile name or "none">

Health check results
--------------------
  Config validity:     pass/fail
  Source accessibility: pass/fail
  Deployment health:   pass/fail
  Agent detection:     pass/fail
  Git availability:    pass/fail

Issues found
------------
  [CRITICAL] Broken symlink: skills/greeting -> /path/that/no/longer/exists
    Fix: nd remove skills/greeting && nd deploy skills/greeting

  [WARNING] Source "my-old-assets" is unavailable (path does not exist)
    Fix: nd source remove my-old-assets
    Or:  Update the path with nd settings edit

  [WARNING] 2 deployments from removed source "old-source"
    Fix: nd remove skills/orphaned-skill commands/orphaned-cmd

Remediation commands
--------------------
  # Fix all broken symlinks at once
  nd sync

  # Remove unavailable source
  nd source remove my-old-assets

  # Re-deploy broken assets
  nd deploy skills/greeting
```

If no issues are found, print:

```
nd audit report
===============

Summary
-------
  Sources:      N registered (all healthy)
  Deployments:  N total (all healthy)

No issues found. Your nd setup is healthy.
```

### JSON format (--json)

Output a JSON object with this structure:

```json
{
  "summary": {
    "sources_total": 2,
    "sources_healthy": 2,
    "sources_unavailable": 0,
    "deployments_total": 5,
    "deployments_healthy": 5,
    "deployments_broken": 0,
    "pinned_count": 1,
    "active_profile": null
  },
  "doctor": {
    "config_valid": true,
    "sources_accessible": true,
    "deployments_healthy": true,
    "agent_detected": true,
    "git_available": true
  },
  "issues": [],
  "remediation": []
}
```

Each issue in the `issues` array:

```json
{
  "severity": "critical",
  "category": "broken_symlink",
  "asset": "skills/greeting",
  "message": "Symlink target does not exist: /path/that/no/longer/exists",
  "fix": "nd remove skills/greeting && nd deploy skills/greeting"
}
```

## Rules

- Run each nd command using the Bash tool. Do not fabricate output.
- Categorize issues by severity: `critical` (broken symlinks, missing config), `warning` (unavailable sources, orphaned deployments), `info` (suggestions for improvement).
- Always provide a concrete remediation command for every issue found.
- If `--json` is specified, output only the JSON object with no surrounding prose.
- Do not modify any nd state during the audit. This is a read-only operation.
- If a command fails, record the failure as a finding and continue with the remaining phases.
