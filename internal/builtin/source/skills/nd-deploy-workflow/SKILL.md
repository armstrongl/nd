---
name: nd-deploy-workflow
description: Use when deploying assets in batch, auditing deployment health, switching profiles, or syncing git sources. Orchestrates multi-step nd workflows.
---

# Deployment workflows

Orchestrate common nd deployment tasks: batch-deploy assets, audit deployment health, switch profiles, and sync sources. Determine which workflow the user needs from their message, then execute the corresponding steps.

## Workflows

Parse the user's request to select one of the four workflows below. If the intent is ambiguous, present the options and ask the user to choose.

---

### 1. deploy my usual setup

Batch-deploy multiple assets from available sources.

**Step 1 — list available assets (script):**
Run `nd list --json` to retrieve all assets across registered sources. Parse the JSON output and group results by type (skills, commands, rules, agents, context, output-styles, plugins, hooks).

**Step 2 — show current status:**
Run `nd status --json` to identify which assets are already deployed. Mark them in the list presented to the user.

**Step 3 — let the user choose:**
Present the grouped asset list. Already-deployed assets should be clearly indicated. Ask the user which assets they want to deploy. Accept asset references in any format nd supports: bare name, `type/name`, or comma-separated lists.

**Step 4 — optional scope selection:**
Ask whether to deploy globally (`--scope global`) or to the current project (`--scope project`). Default to global if the user does not specify.

**Step 5 — deploy (script):**
Run `nd deploy <asset1> <asset2> ... --scope <scope> --yes` with all selected assets in a single command. If any asset fails to deploy, capture the error and continue reporting on the rest.

**Step 6 — verify:**
Run `nd status` and present a summary showing which assets are now deployed and any that failed.

---

### 2. audit deployments

Check deployment health and surface problems.

**Step 1 — run doctor (script):**
Run `nd doctor --json`. Capture the full output.

**Step 2 — run status (script):**
Run `nd status --json`. Capture the full output.

**Step 3 — interpret results:**
Analyze the doctor and status output for:

- Broken symlinks: symlinks pointing to targets that no longer exist
- Orphaned deployments: deployed assets whose source has been removed or unregistered
- Stale assets: assets that have been modified at the source but the symlink target is outdated
- Configuration warnings: missing config, unregistered sources, or permission issues

**Step 4 — present findings:**
Report each category of issue with the affected asset names and paths. For each issue, include the recommended fix command. Example:

```
Broken symlinks (2):
  skills/old-skill  ->  /path/to/missing/source
    Fix: nd remove skills/old-skill

Orphaned deployments (1):
  rules/stale-rule  ->  source "my-source" no longer registered
    Fix: nd remove rules/stale-rule
```

**Step 5 — offer remediation:**
Ask the user whether to execute the suggested fixes. If they agree, run the repair commands. For broken symlinks, use `nd sync --yes` first (it repairs repairable links), then `nd remove` for anything that cannot be repaired.

---

### 3. switch profiles

Save current state and switch to a different profile.

**Step 1 — show current state (script):**
Run `nd status --json` and `nd profile list --json` in parallel. Present the current deployments and the list of available profiles.

**Step 2 — offer snapshot:**
Ask the user whether to save the current deployment state as a snapshot before switching. If yes, ask for a snapshot name (suggest a default based on the current date and time, e.g., `pre-switch-2026-04-02`). Run `nd snapshot save <name>`.

**Step 3 — select target profile:**
If the user has not already specified a profile, ask them to choose from the available profiles listed in Step 1. If no profiles exist, offer to create one:

- Ask for a profile name and description
- Run `nd profile create <name> --from-current --description "<desc>"` to create a profile from the current deployments, or collect asset references for a new custom profile

**Step 4 — switch (script):**
Run `nd profile switch <name> --yes`. This removes assets not in the target profile and deploys the ones that are.

**Step 5 — verify and report:**
Run `nd status` to confirm the switch completed. Present a summary showing which assets were added, removed, and retained. If any pinned assets were preserved across the switch, note that.

---

### 4. sync and refresh

Pull updates from git sources and verify deployment health.

**Step 1 — check sources (script):**
Run `nd source list --json` to enumerate registered sources. Identify which sources are git-backed.

**Step 2 — sync (script):**
Run `nd sync --yes --verbose`. This pulls updates from all git sources and repairs any broken symlinks. Capture the output.

**Step 3 — report sync results:**
Present a summary of what happened:

- Which sources were pulled and whether updates were found
- How many symlinks were repaired
- Any errors encountered during sync

**Step 4 — post-sync health check (script):**
Run `nd doctor --json` and `nd status --json` in parallel. Check for issues introduced or resolved by the sync.

**Step 5 — present health report:**
If all checks pass, confirm that deployments are healthy. If issues remain, follow the same interpretation and remediation steps from the "Audit deployments" workflow (Steps 3-5).

---

## Rules

- Always use `--json` when running nd commands for parsing. Use the non-JSON form only for final output shown to the user.
- Always use `--yes` to skip confirmation prompts when executing commands on the user's behalf, since the user has already confirmed intent through the conversation.
- Never run `nd remove` or `nd profile switch` without first explaining what will happen and getting the user's confirmation.
- When multiple nd commands are independent of each other, run them in parallel.
- If nd is not installed or not on PATH, inform the user and stop. Do not attempt to install it.
- If `nd doctor` or `nd status` returns errors about missing configuration, suggest running `nd init` before proceeding.
- Use `--scope project` only when the user explicitly requests project-scoped operations. Default to `--scope global`.
- When reporting results, always use absolute paths for any file or symlink references.
