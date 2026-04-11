#!/usr/bin/env python3
"""Checks docs for time-based and path-based staleness. Outputs a JSON report.

Commit-type triage
------------------
When staleness is path-based, each commit touching the tracked paths is
classified by its conventional-commit prefix. If every non-merge commit
uses an *internal* prefix (fix, refactor, test, ci, chore, build, style,
perf), the doc is marked ``autoResolvable: true`` — the workflow can bump
``lastValidated`` without human review. Commits with ``feat``, ``docs``,
or an unrecognised prefix require manual review.
"""

import argparse
import glob
import json
import os
import re
import subprocess
import sys
from datetime import date, timedelta

import yaml

from frontmatter import parse_frontmatter

# Conventional-commit prefixes that indicate internal-only changes.
# Docs tracking these paths almost never need content updates.
INTERNAL_PREFIXES = frozenset({
    "fix", "refactor", "test", "ci", "chore", "build", "style", "perf",
})


def classify_commit(message: str) -> str:
    """Return the conventional-commit prefix of *message*, or a sentinel.

    Returns ``"merge"`` for merge commits, the lowercased prefix for
    conventional commits (e.g. ``"feat"``, ``"fix"``), or ``"unknown"``
    when no prefix is recognised.
    """
    if message.startswith("Merge "):
        return "merge"
    match = re.match(r"^(\w+)(?:\([^)]*\))?!?:", message)
    if match:
        return match.group(1).lower()
    return "unknown"


def classify_log_output(log_output: str) -> dict:
    """Classify every commit in *git log --oneline* output.

    Returns a dict with:
      ``prefixes``
        Mapping of prefix → count (merge commits excluded).
      ``autoResolvable``
        ``True`` when every non-merge commit has an internal prefix.
    """
    prefixes: dict[str, int] = {}
    for line in log_output.strip().splitlines():
        # Format: "<hash> <message>"
        parts = line.split(" ", 1)
        if len(parts) < 2:
            continue
        prefix = classify_commit(parts[1])
        if prefix == "merge":
            continue
        prefixes[prefix] = prefixes.get(prefix, 0) + 1

    auto = bool(prefixes) and all(p in INTERNAL_PREFIXES for p in prefixes)
    return {"prefixes": prefixes, "autoResolvable": auto}


def run_git_log(since: str, paths: list[str], repo_root: str) -> str:
    """Run git log --since for the given paths. Returns stdout."""
    cmd = ["git", "log", "--oneline", f"--since={since}", "--"] + paths
    try:
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            cwd=repo_root,
        )
        if result.returncode != 0:
            print(
                f"Warning: git log failed with exit code {result.returncode} "
                f"for paths {paths!r} in {repo_root!r}: {result.stderr.strip()}",
                file=sys.stderr,
            )
            return ""
        return result.stdout.strip()
    except (subprocess.SubprocessError, FileNotFoundError) as exc:
        print(
            f"Warning: failed to run git log for paths {paths!r} in {repo_root!r}: {exc}",
            file=sys.stderr,
        )
        return ""


def check_time_staleness(last_validated: str, max_age_days: int) -> dict | None:
    """Check if a doc is time-stale. Returns a dict with reason if stale, None otherwise.

    Raises ``ValueError`` if *max_age_days* cannot be converted to ``int``.
    """
    # Validate max_age_days is numeric
    max_age_days = int(max_age_days)

    try:
        validated_date = date.fromisoformat(last_validated)
    except (ValueError, TypeError):
        return {"reason": "time", "detail": f"Invalid lastValidated: {last_validated}"}

    threshold = validated_date + timedelta(days=max_age_days)
    if date.today() >= threshold:
        days_over = (date.today() - threshold).days
        return {
            "reason": "time",
            "detail": f"Last validated {last_validated}, {days_over} days past {max_age_days}-day threshold",
        }
    return None


def check_path_staleness(paths: list[str], last_validated: str, repo_root: str) -> dict | None:
    """Check if code paths have changed since lastValidated.

    Returns a dict with ``reason``, ``detail``, commit ``prefixes``, and
    ``autoResolvable`` flag when stale, or ``None`` otherwise.
    """
    if not paths:
        return None

    output = run_git_log(last_validated, paths, repo_root)
    if output:
        classification = classify_log_output(output)
        # Use the classified prefix counts (merge commits excluded)
        # so the reported count matches the prefixes breakdown.
        commit_count = sum(classification["prefixes"].values())
        if commit_count == 0:
            return None
        return {
            "reason": "paths",
            "detail": f"{commit_count} commit(s) touching tracked paths since {last_validated}",
            "prefixes": classification["prefixes"],
            "autoResolvable": classification["autoResolvable"],
        }
    return None


def _check_single_doc(
    filepath: str, default_max_age: int, repo_root: str,
) -> tuple[dict, dict | None, dict | None, int] | None:
    """Validate one doc and run staleness checks.

    Returns ``(fm, time_result, path_result, max_age_int)`` when the doc
    has valid frontmatter and a ``lastValidated`` field, or ``None`` to skip.
    """
    fm = parse_frontmatter(filepath)
    if not fm or not fm.get("lastValidated"):
        return None

    max_age = fm.get("maxAgeDays", default_max_age)
    try:
        max_age_int = int(max_age)
    except (ValueError, TypeError):
        print(
            f"Warning: skipping {filepath} — maxAgeDays is not numeric: {max_age!r}",
            file=sys.stderr,
        )
        return None

    last_validated = str(fm["lastValidated"])
    time_result = check_time_staleness(last_validated, max_age_int)
    path_result = check_path_staleness(fm.get("paths", []), last_validated, repo_root)
    return fm, time_result, path_result, max_age_int


def _build_flagged_entry(
    filepath: str,
    fm: dict,
    time_result: dict | None,
    path_result: dict | None,
    repo_root: str,
    max_age_int: int,
) -> dict | None:
    """Build a flagged-entry dict if at least one staleness check fired."""
    if not time_result and not path_result:
        return None

    reasons = []
    details = []
    for tag, result in [("time", time_result), ("paths", path_result)]:
        if result:
            reasons.append(tag)
            details.append(result["detail"])

    # Auto-resolvable only when staleness is purely path-based
    # and every commit uses an internal prefix.
    auto = (
        path_result is not None
        and time_result is None
        and path_result.get("autoResolvable", False)
    )

    entry = {
        "file": os.path.relpath(filepath, repo_root),
        "title": fm.get("title", os.path.basename(filepath)),
        "reason": " + ".join(reasons),
        "details": details,
        "lastValidated": str(fm["lastValidated"]),
        "maxAgeDays": max_age_int,
        "autoResolvable": auto,
    }
    if path_result and path_result.get("prefixes"):
        entry["prefixes"] = path_result["prefixes"]

    return entry


def check_all_docs(docs_dir: str, default_max_age: int, repo_root: str) -> list[dict]:
    """Check all docs for staleness. Returns list of flagged doc reports."""
    pattern = os.path.join(docs_dir, "**/*.md")
    flagged = []
    for filepath in sorted(glob.glob(pattern, recursive=True)):
        result = _check_single_doc(filepath, default_max_age, repo_root)
        if result is None:
            continue
        fm, time_result, path_result, max_age_int = result
        entry = _build_flagged_entry(filepath, fm, time_result, path_result, repo_root, max_age_int)
        if entry:
            flagged.append(entry)
    return flagged


def load_default_max_age(repo_root: str) -> int:
    """Load default maxAgeDays from .agentsrc.yaml, falling back to 90."""
    config_path = os.path.join(repo_root, ".agentsrc.yaml")
    if not os.path.exists(config_path):
        return 90

    try:
        with open(config_path, "r", encoding="utf-8") as f:
            config = yaml.safe_load(f)
        return config.get("defaults", {}).get("maxAgeDays", 90)
    except yaml.YAMLError:
        return 90


def main():
    parser = argparse.ArgumentParser(description="Check docs for staleness.")
    parser.add_argument(
        "--docs-dir",
        default="docs/guide",
        help="Path to docs directory (default: docs/guide)",
    )
    parser.add_argument(
        "--repo-root",
        default=".",
        help="Path to repo root (default: current directory)",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Print the report without taking any action.",
    )
    args = parser.parse_args()

    default_max_age = load_default_max_age(args.repo_root)
    results = check_all_docs(args.docs_dir, default_max_age, args.repo_root)

    report = json.dumps(results, indent=2, ensure_ascii=False)
    print(report)

    if not args.dry_run and results:
        report_path = os.path.join(args.repo_root, "staleness-report.json")
        with open(report_path, "w", encoding="utf-8") as f:
            f.write(report)
        print(f"\nReport written to {report_path}")


if __name__ == "__main__":
    main()
