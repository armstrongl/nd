from __future__ import annotations

import importlib.util
import json
import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest import mock


SCRIPT_PATH = Path(__file__).resolve().parents[1] / "taskmd-issue-sync.py"
SPEC = importlib.util.spec_from_file_location("taskmd_issue_sync", SCRIPT_PATH)
assert SPEC is not None and SPEC.loader is not None
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class TaskmdIssueSyncTests(unittest.TestCase):
    def test_parse_issue_reference_accepts_suffixes(self) -> None:
        repo, number = MODULE.parse_issue_reference(
            "https://GitHub.com/owner/repo/issues/42/?foo=bar#comments"
        )
        self.assertEqual("owner/repo", repo)
        self.assertEqual(42, number)

    def test_resolve_task_path_handles_repo_relative_path(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            repo_root = Path(temp_dir)
            task_file = repo_root / "tasks" / "abc123-task.md"
            task_file.parent.mkdir(parents=True)
            task_file.write_text("id: abc123\n")

            resolved = MODULE.resolve_task_path(repo_root, "tasks/abc123-task.md")
            self.assertEqual(task_file.resolve(), resolved)

    def test_run_set_uses_repo_from_issue_url(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            repo_root = Path(temp_dir)
            task_file = repo_root / "tasks" / "abc123-task.md"
            task_file.parent.mkdir(parents=True)
            task_file.write_text(
                "---\nid: abc123\n---\n\n### References\n\n- GitHub issue: https://github.com/other-owner/other-repo/issues/9\n"
            )

            def fake_run(command: list[str], **_: object) -> subprocess.CompletedProcess[str]:
                if command[:2] == ["taskmd", "set"]:
                    return subprocess.CompletedProcess(command, 0)
                if command[:3] == ["gh", "issue", "close"]:
                    return subprocess.CompletedProcess(command, 0, "", "")
                raise AssertionError(f"unexpected command: {command}")

            with (
                mock.patch.object(MODULE, "get_repo_root", return_value=repo_root),
                mock.patch.object(MODULE, "find_task_path", return_value=task_file),
                mock.patch.object(MODULE, "ensure_gh_auth"),
                mock.patch.object(
                    MODULE,
                    "run_json",
                    return_value={
                        "state": "OPEN",
                        "url": "https://github.com/other-owner/other-repo/issues/9",
                    },
                ) as run_json_mock,
                mock.patch.object(MODULE.subprocess, "run", side_effect=fake_run) as run_mock,
            ):
                result = MODULE.run_set(["abc123", "--status", "completed"])

            self.assertEqual(0, result)
            self.assertEqual(
                [
                    "gh",
                    "issue",
                    "view",
                    "9",
                    "--repo",
                    "other-owner/other-repo",
                    "--json",
                    "state,url",
                ],
                run_json_mock.call_args.args[0],
            )
            close_call = run_mock.call_args_list[-1].args[0]
            self.assertEqual(
                [
                    "gh",
                    "issue",
                    "close",
                    "9",
                    "--repo",
                    "other-owner/other-repo",
                    "--comment",
                    "Closed automatically after taskmd task `abc123` was marked completed.",
                ],
                close_call,
            )

    def test_run_json_surfaces_subprocess_error_output(self) -> None:
        error = subprocess.CalledProcessError(
            returncode=1,
            cmd=["gh", "issue", "view", "1"],
            stderr="boom",
            output="",
        )
        with mock.patch.object(MODULE.subprocess, "run", side_effect=error):
            with self.assertRaises(RuntimeError) as context:
                MODULE.run_json(["gh", "issue", "view", "1"], cwd=Path.cwd())

        self.assertIn("command failed (1): gh issue view 1", str(context.exception))
        self.assertIn("boom", str(context.exception))

    def test_run_json_rejects_non_object_payloads(self) -> None:
        completed = subprocess.CompletedProcess(
            args=["taskmd", "add", "--format", "json"],
            returncode=0,
            stdout=json.dumps([1, 2, 3]),
            stderr="",
        )
        with mock.patch.object(MODULE.subprocess, "run", return_value=completed):
            with self.assertRaises(RuntimeError) as context:
                MODULE.run_json(["taskmd", "add", "--format", "json"], cwd=Path.cwd())
        self.assertIn("unexpected JSON output", str(context.exception))


if __name__ == "__main__":
    unittest.main()
