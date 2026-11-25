"""Behavioral tests for complete event-scoped commit attribution records."""

from __future__ import annotations

import os
import shutil
import subprocess
import sys
from pathlib import Path

import pytest

REPO = Path(__file__).resolve().parent.parent
WORKFLOW = REPO / ".github" / "workflows" / "guardrails-lint.yml"
HELPER = REPO / "scripts" / "changed_commits.sh"


def _attribution_run_block() -> str:
    lines = WORKFLOW.read_text(encoding="utf-8").splitlines()
    for index, line in enumerate(lines):
        if line.strip() != "- name: Check for AI Attribution":
            continue
        for run_index in range(index + 1, len(lines)):
            if lines[run_index].strip() != "run: |":
                continue
            indent = len(lines[run_index]) - len(lines[run_index].lstrip()) + 2
            body = []
            for raw in lines[run_index + 1 :]:
                if raw.strip() and len(raw) - len(raw.lstrip()) < indent:
                    break
                body.append(raw[indent:] if raw.strip() else "")
            block = "\n".join(body)
            assert 'scripts/changed_commits.sh --format "%H"' in block
            assert "git show -s --format=%B" in block
            return block
    raise AssertionError("AI attribution workflow step with a run block is missing")


def _git(cwd: Path, *args: str) -> str:
    env = os.environ.copy()
    env.update(
        {
            "GIT_CONFIG_NOSYSTEM": "1",
            "GIT_CONFIG_GLOBAL": os.devnull,
            "GIT_TERMINAL_PROMPT": "0",
        }
    )
    result = subprocess.run(
        ["git", *args], cwd=cwd, env=env, check=True, capture_output=True, text=True
    )
    return result.stdout.strip()


def _consumer_repo(tmp_path: Path, *, publish_target: bool = True) -> Path:
    repo = tmp_path / "consumer"
    bare = tmp_path / "origin.git"
    repo.mkdir()
    bare.mkdir()
    _git(bare, "init", "--quiet", "--bare", "--initial-branch=main")
    _git(repo, "init", "--quiet", "--initial-branch=main")
    _git(repo, "config", "user.name", "Attribution Fixture")
    _git(repo, "config", "user.email", "fixture@example.invalid")

    scripts = repo / "scripts"
    scripts.mkdir()
    helper = scripts / "changed_commits.sh"
    # The Windows checkout may use CRLF; the Linux CI fixture needs the script
    # bytes GitHub's Linux runner receives from the repository's LF blob.
    helper.write_text(HELPER.read_text(encoding="utf-8"), encoding="utf-8")
    helper.chmod(0o755)
    (repo / "base.txt").write_text("base\n", encoding="utf-8")
    _git(repo, "add", "--all")
    _git(repo, "commit", "--quiet", "-m", "chore: fixture base")
    _git(repo, "remote", "add", "origin", str(bare))

    if publish_target:
        _git(repo, "push", "--quiet", "origin", "main")
        _git(repo, "fetch", "--quiet", "origin", "main:refs/remotes/origin/main")
    return repo


def _commit_message(repo: Path, tmp_path: Path, index: int, message: str) -> str:
    content = repo / f"event-{index}.txt"
    content.write_text(f"event {index}\n", encoding="utf-8")
    _git(repo, "add", content.name)
    message_file = tmp_path / f"message-{index}.txt"
    message_file.write_text(message.rstrip("\n") + "\n", encoding="utf-8")
    _git(repo, "commit", "--quiet", "-F", str(message_file))
    return _git(repo, "rev-parse", "HEAD")


def _run_step(
    repo: Path, *, path_prefix: Path | None = None
) -> subprocess.CompletedProcess[str]:
    if sys.platform == "win32":
        pytest.skip("the workflow run block uses Linux Bash semantics")
    if shutil.which("bash") is None:
        pytest.skip("Bash is required to execute the workflow run block")

    env = os.environ.copy()
    env.update(
        {
            "EVENT_NAME": "pull_request",
            "BASE_REF": "main",
            "REF_NAME": "feature/attribution-fixture",
            "BEFORE_SHA": "",
            "REF_SHA": _git(repo, "rev-parse", "HEAD"),
            "GIT_CONFIG_NOSYSTEM": "1",
            "GIT_CONFIG_GLOBAL": os.devnull,
            "GIT_TERMINAL_PROMPT": "0",
        }
    )
    if path_prefix is not None:
        env["PATH"] = str(path_prefix) + os.pathsep + env.get("PATH", "")
        env["REAL_GIT"] = shutil.which("git") or "git"
    return subprocess.run(
        ["bash", "-c", _attribution_run_block()],
        cwd=repo,
        env=env,
        capture_output=True,
        text=True,
        timeout=30,
    )


def test_multiline_and_delimiter_like_messages_stay_with_their_commit(
    tmp_path: Path,
) -> None:
    repo = _consumer_repo(tmp_path)
    _commit_message(
        repo,
        tmp_path,
        1,
        "feat: multiline attribution\n\nClaude assisted with the implementation.\n"
        "Authored by TheArchitectit",
    )
    _commit_message(
        repo,
        tmp_path,
        2,
        "fix: separator-like body\n\nAI-assisted change.\n---COMMIT_SEP---\n"
        "Authored by TheArchitectit",
    )
    uncredited = _commit_message(
        repo, tmp_path, 3, "docs: uncredited update\n\nCopilot assisted with this note."
    )

    result = _run_step(repo)

    assert result.returncode == 0, result.stdout + result.stderr
    assert "3 commit(s) checked; 1 may need attribution" in result.stdout
    assert f"Commit {uncredited} may need AI attribution." in result.stdout
    assert "---COMMIT_SEP---" not in result.stdout


def test_missing_target_branch_fails_the_attribution_step(tmp_path: Path) -> None:
    repo = _consumer_repo(tmp_path, publish_target=False)

    result = _run_step(repo)

    assert result.returncode != 0
    assert "Could not compute the event-scoped commit range" in result.stdout
    assert "0 commits available" not in result.stdout


def test_successful_empty_range_is_reported_explicitly(tmp_path: Path) -> None:
    repo = _consumer_repo(tmp_path)

    result = _run_step(repo)

    assert result.returncode == 0, result.stdout + result.stderr
    assert "0 commits available (empty event range)" in result.stdout


def test_large_messages_do_not_trigger_pipefail_false_negatives(tmp_path: Path) -> None:
    repo = _consumer_repo(tmp_path)
    large_body = "x" * 65536
    attributed = _commit_message(
        repo,
        tmp_path,
        1,
        "Claude-assisted change\n\nAuthored by TheArchitectit\n" + large_body,
    )
    uncredited = _commit_message(
        repo,
        tmp_path,
        2,
        "Copilot-assisted change\n\n" + large_body,
    )

    result = _run_step(repo)

    assert result.returncode == 0, result.stdout + result.stderr
    assert f"Commit {attributed} may need AI attribution." not in result.stdout
    assert f"Commit {uncredited} may need AI attribution." in result.stdout
    assert "2 commit(s) checked; 1 may need attribution" in result.stdout


def test_unreadable_commit_message_fails_closed(tmp_path: Path) -> None:
    repo = _consumer_repo(tmp_path)
    commit_sha = _commit_message(repo, tmp_path, 1, "feat: Claude-assisted change")
    fake_bin = tmp_path / "fake-bin"
    fake_bin.mkdir()
    git_shim = fake_bin / "git"
    git_shim.write_text(
        "#!/bin/sh\n"
        "if [ \"${1:-}\" = show ]; then echo 'fixture cannot read commit body' >&2; exit 64; fi\n"
        "exec \"$REAL_GIT\" \"$@\"\n",
        encoding="utf-8",
    )
    git_shim.chmod(0o755)

    result = _run_step(repo, path_prefix=fake_bin)

    assert result.returncode != 0
    assert f"Could not read the complete message for commit {commit_sha}" in result.stdout
    assert "Attribution check complete." not in result.stdout


def test_malformed_commit_record_fails_before_inspection(tmp_path: Path) -> None:
    repo = _consumer_repo(tmp_path)
    helper = repo / "scripts" / "changed_commits.sh"
    helper.write_text("#!/bin/sh\nprintf '%s\\n' 'not-a-full-sha'\n", encoding="utf-8")
    helper.chmod(0o755)

    result = _run_step(repo)

    assert result.returncode != 0
    assert "expected a full lowercase SHA-1" in result.stdout
    assert "Attribution check complete." not in result.stdout
