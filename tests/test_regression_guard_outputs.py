"""Regression coverage for Regression Guard step output records."""

from __future__ import annotations

import os
import subprocess
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[1]
HELPER = REPO_ROOT / ".github/scripts/count-bug-fix-commits.sh"
WORKFLOW = REPO_ROOT / ".github/workflows/regression-guard.yml"


def test_mixed_subjects_emit_only_the_consumed_numeric_output(tmp_path: Path) -> None:
    subjects = tmp_path / "subjects.txt"
    subjects.write_text(
        "docs: clarify setup\nfix(api): correct response handling\n"
        "chore: update metadata\nhotfix: prevent startup failure\n",
        encoding="utf-8",
    )
    output = tmp_path / "github-output.txt"

    env = os.environ.copy()
    env["GITHUB_OUTPUT"] = str(output)
    counted = subprocess.run(
        ["bash", str(HELPER), str(subjects)],
        env=env,
        check=True,
        capture_output=True,
        text=True,
    )

    records = output.read_text(encoding="utf-8").splitlines()
    assert records == ["bug_fix_count=2"]
    assert counted.stdout == "2\n"
    assert all(record.startswith("bug_fix_count=") for record in records)
    assert all(record.partition("=")[2].isdecimal() for record in records)

    workflow = WORKFLOW.read_text(encoding="utf-8")
    assert "BUG_FIX_COUNT=$(bash .github/scripts/count-bug-fix-commits.sh commit_messages.txt)" in workflow
    assert "steps.check_bug_fixes.outputs.bug_fix_count" in workflow
    assert "steps.check_bug_fixes.outputs.commits" not in workflow
