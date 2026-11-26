"""Behavioral rehearsal of the Release Coverage workflow's actual run block."""

from __future__ import annotations

import importlib.util
import os
import shutil
import subprocess
from pathlib import Path

import pytest

REPO = Path(__file__).resolve().parent.parent
WORKFLOW = Path(
    os.environ.get(
        "RELEASE_COVERAGE_WORKFLOW",
        str(REPO / ".github" / "workflows" / "release-coverage.yml"),
    )
)
VERSION = os.environ.get("RELEASE_FIXTURE_VERSION", "3.8.11")


def _coverage_run_block() -> str:
    structural_reader = os.environ.get("RELEASE_WORKFLOW_READ_MODULE")
    if structural_reader:
        spec = importlib.util.spec_from_file_location("workflow_read", structural_reader)
        if spec is None or spec.loader is None:
            raise AssertionError(f"cannot load workflow reader: {structural_reader}")
        module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(module)
        workflow = module.read_workflow(WORKFLOW.read_text(encoding="utf-8"))
        steps = workflow.tree["jobs"]["coverage"]["steps"]
        for step in steps:
            if step.get("name") == "Verify release coverage":
                block = step.get("run")
                assert isinstance(block, str)
                assert 'git diff --name-only HEAD^1 HEAD' in block
                assert 'git cat-file -t "$tag"' in block
                assert 'CHANGELOG.md has no release heading' in block
                return block
        raise AssertionError("Verify release coverage step is missing")

    lines = WORKFLOW.read_text(encoding="utf-8").splitlines()
    for index, line in enumerate(lines):
        if line.strip() != "- name: Verify release coverage":
            continue
        step_indent = len(line) - len(line.lstrip())
        for run_index in range(index + 1, len(lines)):
            candidate = lines[run_index]
            if candidate.strip() and len(candidate) - len(candidate.lstrip()) <= step_indent:
                break
            if candidate.strip() != "run: |":
                continue
            indent = len(lines[run_index]) - len(lines[run_index].lstrip()) + 2
            body = []
            for raw in lines[run_index + 1 :]:
                if raw.strip() and len(raw) - len(raw.lstrip()) < indent:
                    break
                body.append(raw[indent:] if raw.strip() else "")
            block = "\n".join(body)
            assert 'git diff --name-only HEAD^1 HEAD' in block
            assert 'git cat-file -t "$tag"' in block
            assert 'CHANGELOG.md has no release heading' in block
            return block
    raise AssertionError("Verify release coverage workflow run block is missing")


def _git_fixture_environment() -> dict[str, str]:
    env = {key: value for key, value in os.environ.items() if not key.startswith("GIT_")}
    env.update(
        {
            "GIT_CONFIG_NOSYSTEM": "1",
            "GIT_CONFIG_GLOBAL": os.devnull,
            "GIT_TERMINAL_PROMPT": "0",
        }
    )
    return env


def _git(cwd: Path, *args: str) -> str:
    result = subprocess.run(
        ["git", *args],
        cwd=cwd,
        env=_git_fixture_environment(),
        check=True,
        capture_output=True,
        text=True,
    )
    return result.stdout.strip()


def _fixture(path: Path, scenario: str) -> Path:
    repo = path / scenario
    repo.mkdir()
    _git(repo, "init", "--quiet", "--initial-branch=main")
    _git(repo, "config", "user.name", "Release Fixture")
    _git(repo, "config", "user.email", "fixture@example.invalid")
    (repo / "VERSION").write_text(f"{VERSION}\n", encoding="utf-8")
    (repo / "CHANGELOG.md").write_text(f"## [{VERSION}]\n\nFixture notes.\n", encoding="utf-8")
    (repo / "product.txt").write_text("base\n", encoding="utf-8")
    _git(repo, "add", "--all")
    _git(repo, "commit", "--quiet", "-m", "fixture base")
    base = _git(repo, "rev-parse", "HEAD")

    if scenario == "generated-only":
        target = repo / "docs" / "qa" / "fleet-sweeps"
        target.mkdir(parents=True)
        (target / "trend.md").write_text("generated\n", encoding="utf-8")
    else:
        (repo / "product.txt").write_text("product change\n", encoding="utf-8")
        if scenario == "mixed-generated-product":
            target = repo / "docs" / "qa" / "fleet-sweeps"
            target.mkdir(parents=True)
            (target / "trend.md").write_text("generated\n", encoding="utf-8")
    if scenario == "missing-notes":
        (repo / "CHANGELOG.md").write_text(f"## [{VERSION}.0]\n", encoding="utf-8")
    _git(repo, "add", "--all")
    _git(repo, "commit", "--quiet", "-m", "fixture candidate")

    if scenario in {"annotated-head", "missing-notes"}:
        _git(repo, "tag", "-a", f"v{VERSION}", "-m", "fixture release")
    elif scenario == "lightweight":
        _git(repo, "tag", f"v{VERSION}")
    elif scenario in {"stale", "mixed-generated-product"}:
        _git(repo, "tag", "-a", f"v{VERSION}", base, "-m", "stale fixture release")
    return repo


def _run(repo: Path, script: str) -> subprocess.CompletedProcess[str]:
    if shutil.which("bash") is None:
        pytest.skip("Bash is required to execute the workflow run block")
    return subprocess.run(
        ["bash", "-e", "-o", "pipefail", "-c", script],
        cwd=repo,
        env=_git_fixture_environment(),
        capture_output=True,
        text=True,
        check=False,
    )


@pytest.mark.parametrize(
    ("scenario", "expected", "message"),
    [
        ("annotated-head", 0, "release coverage: HEAD is annotated"),
        ("missing-tag", 1, "main has no tagged release"),
        ("lightweight", 1, "must be annotated"),
        ("stale", 1, "is not release"),
        ("missing-notes", 1, "has no release heading"),
        ("generated-only", 0, "SKIPPED (not a product release)"),
        ("mixed-generated-product", 1, "is not release"),
    ],
)
def test_release_coverage_workflow_fixtures(
    tmp_path: Path, scenario: str, expected: int, message: str
) -> None:
    repo = _fixture(tmp_path, scenario)
    assert repo.is_relative_to(tmp_path)
    assert _git(repo, "remote") == ""
    fixture_refs_before = _git(repo, "show-ref", "--head")

    result = _run(repo, _coverage_run_block())

    assert result.returncode == expected, result.stdout + result.stderr
    assert message in result.stdout + result.stderr
    assert _git(repo, "show-ref", "--head") == fixture_refs_before
