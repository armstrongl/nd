#!/usr/bin/env python3
"""Bumps lastValidated for auto-resolvable docs in a staleness report.

Reads a staleness-report.json produced by check-staleness.py, filters
entries where ``autoResolvable`` is true, and rewrites ``lastValidated``
in each file's YAML frontmatter to today's date (or a date given via
``--date``). Prints a JSON array of bumped file paths to stdout.
"""

import argparse
import json
import re
import sys
from datetime import date


def bump_last_validated(filepath: str, new_date: str) -> bool:
    """Rewrite ``lastValidated`` in *filepath*'s frontmatter to *new_date*.

    Returns ``True`` if the file was modified, ``False`` otherwise.
    """
    with open(filepath, "r", encoding="utf-8") as f:
        content = f.read()

    new_content = re.sub(
        r'(lastValidated:\s*["\']?)\d{4}-\d{2}-\d{2}(["\']?)',
        rf"\g<1>{new_date}\2",
        content,
        count=1,
    )

    if new_content == content:
        return False

    with open(filepath, "w", encoding="utf-8") as f:
        f.write(new_content)
    return True


def main():
    parser = argparse.ArgumentParser(
        description="Auto-bump lastValidated for trivially stale docs.",
    )
    parser.add_argument(
        "--report",
        default="staleness-report.json",
        help="Path to the staleness report JSON (default: staleness-report.json)",
    )
    parser.add_argument(
        "--date",
        default=str(date.today()),
        help="Date to set lastValidated to (default: today)",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Print what would be bumped without writing files.",
    )
    args = parser.parse_args()

    try:
        with open(args.report, "r", encoding="utf-8") as f:
            report = json.load(f)
    except FileNotFoundError:
        print(json.dumps([]))
        return
    except json.JSONDecodeError as exc:
        print(f"Error: invalid JSON in {args.report}: {exc}", file=sys.stderr)
        sys.exit(1)

    bumped = []
    for entry in report:
        if not entry.get("autoResolvable"):
            continue
        filepath = entry["file"]
        if args.dry_run:
            print(f"Would bump {filepath} → {args.date}", file=sys.stderr)
            bumped.append(filepath)
        else:
            if bump_last_validated(filepath, args.date):
                bumped.append(filepath)
                print(f"Bumped {filepath} → {args.date}", file=sys.stderr)
            else:
                print(f"Warning: no lastValidated found in {filepath}", file=sys.stderr)

    # Machine-readable output for workflow consumption
    print(json.dumps(bumped))


if __name__ == "__main__":
    main()
