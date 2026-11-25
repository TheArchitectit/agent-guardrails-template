"""Regression coverage for the Gitleaks runner-temp collision.

The old action used a fixed ``/tmp/gitleaks.tmp`` download path, which could
already exist on a persistent runner. The workflow now allocates a private
child under canonicalized ``RUNNER_TEMP`` and removes only that child. These
tests exercise allocation and cleanup with synthetic paths; they never run a
scanner or touch global ``/tmp/gitleaks.tmp``.
"""

from __future__ import annotations

import os
import subprocess
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[1]
ALLOCATE = REPO_ROOT / ".github/scripts/allocate-gitleaks-temp.sh"
CLEANUP = REPO_ROOT / ".github/scripts/cleanup-gitleaks.sh"


def allocate(runner_temp: Path, output: Path) -> Path:
    env = os.environ | {
        "RUNNER_TEMP": str(runner_temp),
        "GITHUB_OUTPUT": str(output),
    }
    subprocess.run(["bash", str(ALLOCATE)], env=env, check=True, capture_output=True)
    key, value = output.read_text(encoding="utf-8").strip().split("=", 1)
    assert key == "install_dir"
    return Path(value)


def cleanup(runner_temp: Path, install_dir: Path) -> subprocess.CompletedProcess[str]:
    env = os.environ | {
        "RUNNER_TEMP": str(runner_temp),
        "GITLEAKS_TEMP_DIR": str(install_dir),
    }
    return subprocess.run(
        ["bash", str(CLEANUP)], env=env, check=False, capture_output=True, text=True
    )


def test_concurrent_temp_allocations_are_canonical_and_cleanup_is_scoped(tmp_path: Path) -> None:
    workflow = (REPO_ROOT / ".github/workflows/secret-validation.yml").read_text(
        encoding="utf-8"
    )
    assert "run: bash .github/scripts/allocate-gitleaks-temp.sh" in workflow
    assert "TMPDIR: ${{ steps.gitleaks-temp.outputs.install_dir }}" in workflow

    real_parent = tmp_path / "runner-temp"
    real_parent.mkdir()
    runner_temp_link = tmp_path / "runner-temp-link"
    runner_temp_link.symlink_to(real_parent, target_is_directory=True)

    stale_sentinel = real_parent / "gitleaks.tmp"
    stale_value = "synthetic stale-path sentinel"
    stale_sentinel.write_text(stale_value, encoding="utf-8")

    outputs = [tmp_path / "job-a.out", tmp_path / "job-b.out"]
    with ThreadPoolExecutor(max_workers=2) as pool:
        jobs = [pool.submit(allocate, runner_temp_link, output) for output in outputs]
        install_dirs = [job.result() for job in jobs]

    assert install_dirs[0] != install_dirs[1]
    assert all(path.parent == real_parent.resolve() for path in install_dirs)
    assert all(path.is_dir() for path in install_dirs)

    # Model the pinned action's fixed child names, without downloading or scanning.
    for install_dir in install_dirs:
        (install_dir / "gitleaks.tmp").write_text("mock archive", encoding="utf-8")
        (install_dir / "gitleaks-8.24.3").mkdir()

    result = cleanup(runner_temp_link, install_dirs[0])
    assert result.returncode == 0, result.stderr
    assert not install_dirs[0].exists()
    assert install_dirs[1].is_dir()
    assert stale_sentinel.read_text(encoding="utf-8") == stale_value

    result = cleanup(runner_temp_link, install_dirs[1])
    assert result.returncode == 0, result.stderr
    assert not install_dirs[1].exists()
    assert stale_sentinel.read_text(encoding="utf-8") == stale_value


def test_cleanup_refuses_paths_outside_canonical_runner_temp(tmp_path: Path) -> None:
    runner_temp = tmp_path / "runner-temp"
    runner_temp.mkdir()
    other_dir = tmp_path / "other"
    other_dir.mkdir()

    result = cleanup(runner_temp, other_dir)

    assert result.returncode != 0
    assert other_dir.is_dir()
