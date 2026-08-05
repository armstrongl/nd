#!/usr/bin/env python3

from __future__ import annotations

import json
import os
import re
import subprocess
import sys
import tempfile
from pathlib import Path
from typing import Any


ISSUE_URL_RE = re.compile(r"https://github\.com/([^/\s]+)/([^/\s]+)/issues/(\d+)(?:[/?#][^\s]*)?", re.IGNORECASE)
ISSUE_BULLET_RE = re.compile(r"^- (?:GitHub issue|Issue): https://github\.com/[^/\s]+/[^/\s]+/issues/\d+(?:[/?#][^\s]*)?\s*$", re.IGNORECASE)
CLOSE_BULLET_RE = re.compile(r"^- Close this issue when the task is completed\.?\s*$", re.IGNORECASE)


def main() -> int:
    if len(sys.argv) < 2 or sys.argv[1] not in {"add", "set"}:
        print("usage: taskmd-issue-sync.py <add|set> [taskmd args...]", file=sys.stderr)
        return 2

    command = sys.argv[1]
    args = sys.argv[2:]

    try:
        if command == "add":
            return run_add(args)
        return run_set(args)
    except subprocess.CalledProcessError as exc:
        print(f"error: {command_error_message(exc)}", file=sys.stderr)
        return exc.returncode or 1
    except RuntimeError as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 1


def run_add(args: list[str]) -> int:
    output_format = find_option_value(args, "--format") or "plain"
    taskmd_args = replace_or_append_option(args, "--format", "json")
    repo_root = get_repo_root()

    result = run_json(["taskmd", "add", *taskmd_args], cwd=repo_root)
    task_path = resolve_task_path(repo_root, require_string(result, "file_path"))
    task_text = task_path.read_text()

    issue_url = find_issue_url(task_text)
    issue_created = False
    if not issue_url:
        ensure_gh_auth(repo_root)
        repo = get_repo_name(repo_root)
        objective = extract_objective(task_text)
        task_id = require_string(result, "id")
        title = require_string(result, "title")
        relative_path = os.path.relpath(task_path, repo_root)
        issue_url = create_issue(
            repo=repo,
            title=title,
            task_id=task_id,
            relative_path=relative_path,
            objective=objective,
            cwd=repo_root,
        )
        issue_created = True

    update_task_references(task_path, issue_url)
    issue_number = parse_issue_number(issue_url)

    payload = {
        "task": {
            "id": require_string(result, "id"),
            "title": require_string(result, "title"),
            "file_path": os.path.relpath(task_path, repo_root),
            "status": result.get("status"),
            "priority": result.get("priority"),
        },
        "issue": {
            "number": issue_number,
            "url": issue_url,
            "created": issue_created,
        },
    }

    if output_format == "json":
        print(json.dumps(payload, indent=2))
        return 0

    action = "Created" if issue_created else "Reused"
    print(f"Created task {payload['task']['id']}: {payload['task']['file_path']}")
    print(f"{action} issue #{issue_number}: {issue_url}")
    print(f"Updated references in {payload['task']['file_path']}")
    return 0


def run_set(args: list[str]) -> int:
    repo_root = get_repo_root()
    task_id = get_task_id(args)
    dry_run = has_flag(args, "--dry-run")
    should_close_issue = not dry_run and is_completion_status(args)

    completed = subprocess.run(["taskmd", "set", *args], cwd=repo_root)
    if completed.returncode != 0:
        return completed.returncode

    if not should_close_issue:
        return 0

    task_path = find_task_path(repo_root, task_id)
    task_text = task_path.read_text()
    issue_url = find_issue_url(task_text)
    if not issue_url:
        print(f"warning: task {task_id} has no linked GitHub issue to close", file=sys.stderr)
        return 0

    ensure_gh_auth(repo_root)
    issue_repo, issue_number = parse_issue_reference(issue_url)
    issue = run_json(["gh", "issue", "view", str(issue_number), "--repo", issue_repo, "--json", "state,url"], cwd=repo_root)
    if issue.get("state") == "CLOSED":
        print(f"Issue already closed: {issue.get('url', issue_url)}")
        return 0

    comment = f"Closed automatically after taskmd task `{task_id}` was marked completed."
    subprocess.run(
        ["gh", "issue", "close", str(issue_number), "--repo", issue_repo, "--comment", comment],
        cwd=repo_root,
        text=True,
        capture_output=True,
        check=True,
    )
    print(f"Closed issue #{issue_number}: {issue_url}")
    return 0


def update_task_references(task_path: Path, issue_url: str) -> None:
    text = task_path.read_text()
    issue_line = f"- GitHub issue: {issue_url}"
    close_line = "- Close this issue when the task is completed."

    def clean_section(body: str) -> str:
        kept: list[str] = []
        for line in body.splitlines():
            if ISSUE_BULLET_RE.match(line) or CLOSE_BULLET_RE.match(line):
                continue
            kept.append(line)
        while kept and kept[-1] == "":
            kept.pop()
        if kept:
            kept.extend(["", issue_line, close_line])
        else:
            kept = [issue_line, close_line]
        return "\n".join(kept) + "\n"

    section_re = re.compile(r"(?ms)^### References\s*\n(?P<body>.*?)(?=^### |\Z)")
    match = section_re.search(text)
    if match:
        cleaned = clean_section(match.group("body"))
        replacement = f"### References\n\n{cleaned}"
        updated = text[: match.start()] + replacement + text[match.end() :]
    else:
        stripped = text.rstrip()
        suffix = f"\n\n### References\n\n{issue_line}\n{close_line}\n"
        updated = stripped + suffix

    task_path.write_text(updated)


def create_issue(repo: str, title: str, task_id: str, relative_path: str, objective: str, cwd: Path) -> str:
    ensure_gh_auth(cwd)
    body = "\n".join(
        [
            f"Task ID: {task_id}",
            f"Task file: {relative_path}",
            "",
            "This GitHub issue tracks the corresponding taskmd task.",
            "",
            "## Objective",
            objective.strip() or "See the linked task file for details.",
            "",
            "*This issue is managed from the local `scripts/taskmd-issue-sync.py` helper.*",
            "",
        ]
    )

    with tempfile.NamedTemporaryFile("w", delete=False) as handle:
        handle.write(body)
        body_path = handle.name

    try:
        completed = run_checked(
            ["gh", "issue", "create", "--repo", repo, "--title", title, "--body-file", body_path],
            cwd=cwd,
        )
    finally:
        Path(body_path).unlink(missing_ok=True)

    issue_url = completed.stdout.strip().splitlines()[-1]
    if not ISSUE_URL_RE.search(issue_url):
        raise RuntimeError(f"unexpected gh issue create output: {issue_url}")
    return issue_url


def ensure_gh_auth(cwd: Path) -> None:
    completed = subprocess.run(
        ["gh", "auth", "status", "--hostname", "github.com"],
        cwd=cwd,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )
    if completed.returncode != 0:
        raise RuntimeError("GitHub CLI is not authenticated; run `gh auth login` first")


def extract_objective(task_text: str) -> str:
    match = re.search(r"(?ms)^### Objective\s*\n+(.*?)(?=^### |\Z)", task_text)
    if match:
        return match.group(1).strip()
    return ""


def find_issue_url(task_text: str) -> str | None:
    match = ISSUE_URL_RE.search(task_text)
    if match:
        return match.group(0)
    return None


def parse_issue_number(issue_url: str) -> int:
    _, issue_number = parse_issue_reference(issue_url)
    return issue_number


def parse_issue_reference(issue_url: str) -> tuple[str, int]:
    match = ISSUE_URL_RE.search(issue_url)
    if not match:
        raise RuntimeError(f"invalid GitHub issue URL: {issue_url}")
    owner = match.group(1)
    repo = match.group(2)
    return f"{owner}/{repo}", int(match.group(3))


def get_repo_root() -> Path:
    completed = run_checked(["git", "rev-parse", "--show-toplevel"], cwd=Path.cwd())
    return Path(completed.stdout.strip())


def get_repo_name(cwd: Path) -> str:
    env_repo = os.environ.get("GITHUB_REPOSITORY")
    if env_repo:
        return env_repo
    data = run_json(["gh", "repo", "view", "--json", "nameWithOwner"], cwd=cwd)
    repo = data.get("nameWithOwner")
    if not isinstance(repo, str) or not repo:
        raise RuntimeError("could not determine GitHub repository name")
    return repo


def find_task_path(repo_root: Path, task_id: str) -> Path:
    tasks_dir = repo_root / "tasks"
    for path in tasks_dir.rglob("*.md"):
        if ".worklogs" in path.parts:
            continue
        text = path.read_text()
        match = re.search(r'^id:\s*"?(.*?)"?\s*$', text, re.MULTILINE)
        if match and match.group(1) == task_id:
            return path
    raise RuntimeError(f"could not find task file for id {task_id}")


def get_task_id(args: list[str]) -> str:
    option_value = find_option_value(args, "--task-id")
    if option_value:
        return option_value

    options_with_values = {
        "--add-pr",
        "--add-tag",
        "--add-touches",
        "--depends-on",
        "--effort",
        "--owner",
        "--parent",
        "--phase",
        "--priority",
        "--remove-pr",
        "--remove-tag",
        "--remove-touches",
        "--status",
        "--type",
    }
    skip_next = False
    for arg in args:
        if skip_next:
            skip_next = False
            continue
        if arg in options_with_values:
            skip_next = True
            continue
        if any(arg.startswith(option + "=") for option in options_with_values):
            continue
        if arg.startswith("-"):
            continue
        return arg
    raise RuntimeError("task ID is required for `set`")


def is_completion_status(args: list[str]) -> bool:
    if has_flag(args, "--done"):
        return True
    status = find_option_value(args, "--status")
    return status == "completed"


def has_flag(args: list[str], flag: str) -> bool:
    return any(arg == flag for arg in args)


def find_option_value(args: list[str], option: str) -> str | None:
    for index, arg in enumerate(args):
        if arg == option:
            if index + 1 >= len(args):
                raise RuntimeError(f"missing value for {option}")
            return args[index + 1]
        if arg.startswith(option + "="):
            return arg.split("=", 1)[1]
    return None


def replace_or_append_option(args: list[str], option: str, value: str) -> list[str]:
    updated: list[str] = []
    skip_next = False
    replaced = False
    for index, arg in enumerate(args):
        if skip_next:
            skip_next = False
            continue
        if arg == option:
            updated.extend([option, value])
            skip_next = True
            replaced = True
            continue
        if arg.startswith(option + "="):
            updated.append(f"{option}={value}")
            replaced = True
            continue
        updated.append(arg)
    if not replaced:
        updated.extend([option, value])
    return updated


def require_string(data: dict[str, Any], key: str) -> str:
    value = data.get(key)
    if not isinstance(value, str) or not value:
        raise RuntimeError(f"missing expected field: {key}")
    return value


def run_json(command: list[str], cwd: Path) -> dict[str, Any]:
    completed = run_checked(command, cwd=cwd)
    try:
        data = json.loads(completed.stdout)
    except json.JSONDecodeError as exc:
        raise RuntimeError(f"invalid JSON output from {' '.join(command)}: {exc}") from exc
    if not isinstance(data, dict):
        raise RuntimeError(f"unexpected JSON output from {' '.join(command)}")
    return data


def resolve_task_path(repo_root: Path, task_path: str) -> Path:
    path = Path(task_path)
    if not path.is_absolute():
        path = repo_root / path
    return path.resolve()


def run_checked(command: list[str], cwd: Path) -> subprocess.CompletedProcess[str]:
    try:
        return subprocess.run(
            command,
            cwd=cwd,
            text=True,
            capture_output=True,
            check=True,
        )
    except subprocess.CalledProcessError as exc:
        raise RuntimeError(command_error_message(exc)) from exc


def command_error_message(exc: subprocess.CalledProcessError) -> str:
    command = format_command(exc.cmd)
    error_output = (exc.stderr or exc.stdout or "").strip()
    if error_output:
        return f"command failed ({exc.returncode}): {command}\n{error_output}"
    return f"command failed ({exc.returncode}): {command}"


def format_command(command: Any) -> str:
    if isinstance(command, (list, tuple)):
        return " ".join(str(part) for part in command)
    return str(command)


if __name__ == "__main__":
    raise SystemExit(main())
