"""Regression controls for the verified direct Gitleaks CLI wrapper."""

from __future__ import annotations

import json
import os
import subprocess
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[1]
SCRIPT = REPO_ROOT / ".github/scripts/gitleaks-verified.sh"
WORKFLOW = REPO_ROOT / ".github/workflows/secret-validation.yml"
PIN = "9991e0b2903da4c8f6122b5c3186448b927a5da4deef1fe45271c3793f4ee29c"


def git(*args: str, cwd: Path) -> str:
    return subprocess.run(
        ["git", *args], cwd=cwd, check=True, capture_output=True, text=True
    ).stdout.strip()


def make_repo(root: Path) -> tuple[str, str]:
    root.mkdir()
    git("init", "-q", cwd=root)
    git("config", "user.email", "test@example.invalid", cwd=root)
    git("config", "user.name", "Test", cwd=root)
    (root / "payload.txt").write_text("benign fixture\n", encoding="utf-8")
    git("add", "payload.txt", cwd=root)
    git("commit", "-qm", "base", cwd=root)
    base = git("rev-parse", "HEAD", cwd=root)
    (root / "payload.txt").write_text("advanced fixture\n", encoding="utf-8")
    git("commit", "-qam", "head", cwd=root)
    return base, git("rev-parse", "HEAD", cwd=root)


def mock_tools(root: Path, *, fake_digest: bool = True) -> Path:
    tools = root / "mock-bin"
    tools.mkdir(parents=True)
    (tools / "curl").write_text(
        "#!/usr/bin/env bash\n"
        "while (($#)); do if [[ $1 == --output ]]; then out=$2; shift 2; else shift; fi; done\n"
        "printf 'synthetic archive bytes' > \"$out\"\n",
        encoding="utf-8",
    )
    (tools / "tar").write_text(
        "#!/usr/bin/env bash\n"
        "while (($#)); do if [[ $1 == -C ]]; then dest=$2; shift 2; else shift; fi; done\n"
        "cat > \"$dest/gitleaks\" <<'SCANNER'\n"
        "#!/usr/bin/env bash\n"
        "if [[ ${1:-} == version ]]; then printf '%s\\n' \"${FAKE_VERSION:-8.24.3}\"; exit 0; fi\n"
        "printf '%s\\n' \"$@\" > \"$CAPTURE_ARGS\"\n"
        "printf '%s' \"${FAKE_OUTPUT:-synthetic-canary-DO-NOT-PRINT}\"\n"
        "exit \"${FAKE_STATUS:-0}\"\n"
        "SCANNER\nchmod +x \"$dest/gitleaks\"\n",
        encoding="utf-8",
    )
    if fake_digest:
        (tools / "sha256sum").write_text(
            "#!/usr/bin/env bash\nprintf '%s  %s\\n' \"$GITLEAKS_TEST_SHA256\" \"$2\"\n",
            encoding="utf-8",
        )
    for tool in tools.iterdir():
        tool.chmod(0o755)
    return tools


def run_wrapper(root: Path, tools: Path, base: str, head: str, tmp: Path,
                **extra_env: str) -> subprocess.CompletedProcess[str]:
    env = os.environ | {
        "PATH": f"{tools}{os.pathsep}{os.environ['PATH']}",
        "RUNNER_TEMP": str(tmp),
        "GITLEAKS_TEST_SHA256": PIN,
        "FAKE_VERSION": "8.24.3",
        "FAKE_STATUS": "0",
        "CAPTURE_ARGS": str(tmp / "scanner-args.txt"),
    } | extra_env
    return subprocess.run(
        ["bash", str(SCRIPT), base, head], cwd=root, env=env,
        capture_output=True, text=True, check=False,
    )


def test_workflow_uses_verified_cli_and_preserves_read_only_guards() -> None:
    workflow = WORKFLOW.read_text(encoding="utf-8")
    scan_job = workflow.split("  check-env-files:", 1)[0]
    assert "pull_request:" in workflow and "push:" in workflow
    assert "contents: read" in workflow
    assert "runs-on: [self-hosted, Linux, X64, guardrails-ucs03]" in scan_job
    assert "fetch-depth: 0" in scan_job
    assert "persist-credentials: false" in scan_job
    assert "run: bash .github/scripts/gitleaks-verified.sh" in scan_job
    assert "gitleaks/gitleaks-action" not in scan_job
    assert "GITLEAKS_LICENSE" not in scan_job and "GITHUB_TOKEN" not in scan_job
    assert "GITLEAKS_ENABLE_UPLOAD_ARTIFACT" not in scan_job


def test_corrupt_archive_fails_before_extraction_or_path_scanner(tmp_path: Path) -> None:
    root = tmp_path / "repo"
    base, head = make_repo(root)
    tools = mock_tools(tmp_path, fake_digest=False)
    marker = tmp_path / "path-scanner-used"
    (tools / "gitleaks").write_text(f"#!/usr/bin/env bash\ntouch '{marker}'\n", encoding="utf-8")
    (tools / "gitleaks").chmod(0o755)
    (tmp_path / "runner-temp").mkdir()

    result = run_wrapper(root, tools, base, head, tmp_path / "runner-temp")

    assert result.returncode != 0
    assert "digest mismatch" in result.stderr
    assert not marker.exists()
    assert not list((tmp_path / "runner-temp").glob("gitleaks-verified.*"))


def test_wrong_version_and_scanner_status_are_fail_closed_and_redacted(tmp_path: Path) -> None:
    root = tmp_path / "repo"
    base, head = make_repo(root)
    tools = mock_tools(tmp_path)
    runner_temp = tmp_path / "runner-temp"
    runner_temp.mkdir()
    path_scanner_marker = tmp_path / "path-scanner-used"
    (tools / "gitleaks").write_text(
        f"#!/usr/bin/env bash\ntouch '{path_scanner_marker}'\nexit 99\n", encoding="utf-8"
    )
    (tools / "gitleaks").chmod(0o755)

    wrong = run_wrapper(root, tools, base, head, runner_temp, FAKE_VERSION="8x24y3")
    assert wrong.returncode != 0
    assert "unexpected scanner version" in wrong.stderr
    assert not path_scanner_marker.exists()

    clean = run_wrapper(root, tools, base, head, runner_temp)
    assert clean.returncode == 0, clean.stderr
    assert "clean" in clean.stdout

    detected = run_wrapper(
        root, tools, base, head, runner_temp,
        FAKE_STATUS="2", FAKE_OUTPUT="synthetic-canary-DO-NOT-PRINT",
    )
    assert detected.returncode == 2
    assert "findings detected" in detected.stderr
    assert "synthetic-canary" not in detected.stdout + detected.stderr
    assert not path_scanner_marker.exists()
    assert not list(runner_temp.glob("gitleaks-verified.*"))


def test_malformed_range_fails_before_network_or_path_scanner(tmp_path: Path) -> None:
    root = tmp_path / "repo"
    base, _ = make_repo(root)
    tools = mock_tools(tmp_path)
    curl_marker = tmp_path / "curl-used"
    (tools / "curl").write_text(
        f"#!/usr/bin/env bash\ntouch '{curl_marker}'\nexit 9\n", encoding="utf-8"
    )
    (tools / "curl").chmod(0o755)
    runner_temp = tmp_path / "runner-temp"
    runner_temp.mkdir()

    result = run_wrapper(root, tools, base, "not-a-full-commit", runner_temp)

    assert result.returncode != 0
    assert "local head must resolve" in result.stderr
    assert not curl_marker.exists()


def test_pinned_local_range_and_private_scratch_are_isolated_concurrently(tmp_path: Path) -> None:
    root = tmp_path / "repo"
    base, head = make_repo(root)
    tools = mock_tools(tmp_path)
    runner_temp = tmp_path / "runner-temp"
    runner_temp.mkdir()
    stale = runner_temp / "gitleaks.tmp"
    stale.write_text("old scratch sentinel", encoding="utf-8")

    def invoke() -> subprocess.CompletedProcess[str]:
        return run_wrapper(root, tools, base, head, runner_temp)

    with ThreadPoolExecutor(max_workers=2) as pool:
        results = list(pool.map(lambda _: invoke(), range(2)))

    assert all(result.returncode == 0 for result in results)
    args = (runner_temp / "scanner-args.txt").read_text(encoding="utf-8").splitlines()
    assert f"--log-opts={base}..{head}" in args
    assert "--redact=100" in args
    assert stale.read_text(encoding="utf-8") == "old scratch sentinel"
    assert not list(runner_temp.glob("gitleaks-verified.*"))


def test_equal_local_boundary_scans_one_commit(tmp_path: Path) -> None:
    root = tmp_path / "repo"
    _, head = make_repo(root)
    tools = mock_tools(tmp_path)
    runner_temp = tmp_path / "runner-temp"
    runner_temp.mkdir()

    result = run_wrapper(root, tools, head, head, runner_temp)

    assert result.returncode == 0, result.stderr
    args = (runner_temp / "scanner-args.txt").read_text(encoding="utf-8").splitlines()
    assert f"--log-opts=-1 {head}" in args


def test_shallow_local_history_fails_closed(tmp_path: Path) -> None:
    root = tmp_path / "repo"
    base, head = make_repo(root)
    shallow_file = root / git("rev-parse", "--git-path", "shallow", cwd=root)
    shallow_file.write_text(f"{base}\n{head}\n", encoding="ascii")
    tools = mock_tools(tmp_path)
    runner_temp = tmp_path / "runner-temp"
    runner_temp.mkdir()

    result = run_wrapper(root, tools, base, head, runner_temp)

    assert result.returncode != 0
    assert "shallow checkout" in result.stderr


def test_local_commits_without_merge_base_fail_closed(tmp_path: Path) -> None:
    root = tmp_path / "repo"
    _, head = make_repo(root)
    tree = git("write-tree", cwd=root)
    unrelated = git("commit-tree", tree, "-m", "unrelated history", cwd=root)
    tools = mock_tools(tmp_path)
    runner_temp = tmp_path / "runner-temp"
    runner_temp.mkdir()

    result = run_wrapper(root, tools, unrelated, head, runner_temp)

    assert result.returncode != 0
    assert "no merge base" in result.stderr


def test_github_ranges_validate_events_and_fall_back_to_pinned_head_history(tmp_path: Path) -> None:
    root = tmp_path / "repo"
    base, head = make_repo(root)
    tools = mock_tools(tmp_path)
    runner_temp = tmp_path / "runner-temp"
    runner_temp.mkdir()
    event_file = tmp_path / "event.json"

    def run_event(name: str, event: dict[str, str]) -> subprocess.CompletedProcess[str]:
        event_file.write_text(json.dumps(event), encoding="utf-8")
        env = os.environ | {
            "PATH": f"{tools}{os.pathsep}{os.environ['PATH']}",
            "RUNNER_TEMP": str(runner_temp),
            "GITHUB_EVENT_NAME": name,
            "GITHUB_EVENT_PATH": str(event_file),
            "GITLEAKS_TEST_SHA256": PIN,
            "FAKE_VERSION": "8.24.3",
            "CAPTURE_ARGS": str(runner_temp / "scanner-args.txt"),
        }
        return subprocess.run(["bash", str(SCRIPT)], cwd=root, env=env,
                              capture_output=True, text=True, check=False)

    push = run_event("push", {"before": base, "after": head})
    assert push.returncode == 0, push.stderr
    assert f"--log-opts={base}..{head}" in (runner_temp / "scanner-args.txt").read_text()

    new_ref = run_event("push", {"before": "0" * 40, "after": head})
    assert new_ref.returncode == 0, new_ref.stderr
    assert f"--log-opts={head}" in (runner_temp / "scanner-args.txt").read_text()

    missing_ancestor = run_event("push", {"before": "f" * 40, "after": head})
    assert missing_ancestor.returncode == 0, missing_ancestor.stderr
    assert f"--log-opts={head}" in (runner_temp / "scanner-args.txt").read_text()

    tree = git("write-tree", cwd=root)
    unrelated = git("commit-tree", tree, "-m", "rewritten history", cwd=root)
    rewritten = run_event("push", {"before": unrelated, "after": head})
    assert rewritten.returncode == 0, rewritten.stderr
    assert f"--log-opts={head}" in (runner_temp / "scanner-args.txt").read_text()

    malformed = run_event("push", {"before": "bad", "after": head})
    assert malformed.returncode != 0
    assert "push before boundary is malformed" in malformed.stderr

    current_branch = git("branch", "--show-current", cwd=root)
    git("checkout", "-qb", "feature", base, cwd=root)
    (root / "feature.txt").write_text("merged fixture\n", encoding="utf-8")
    git("add", "feature.txt", cwd=root)
    git("commit", "-qm", "feature commit", cwd=root)
    git("checkout", current_branch, cwd=root)
    git("merge", "--no-ff", "feature", "-qm", "merge feature", cwd=root)
    merged_head = git("rev-parse", "HEAD", cwd=root)
    pull = run_event("pull_request", {
        "pull_request": {"base": {"sha": base}, "head": {"sha": merged_head}}
    })
    assert pull.returncode == 0, pull.stderr
    assert f"--log-opts={base}..{merged_head}" in (runner_temp / "scanner-args.txt").read_text()
