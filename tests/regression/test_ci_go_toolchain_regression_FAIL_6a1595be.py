"""Regression test for failure FAIL-6a1595be.

Bug: Legacy GitLab and Jenkins Go jobs relied on Go 1.21 while the modules
declared Go 1.26.11 and 1.23.2, with no exact-version guard before build.
Fix: Pin each job to its module version, prohibit automatic fallback, and fail
before test/build when the requested toolchain is missing or mismatched.
These tests read and execute the shell commands from the checked-in templates.
"""

from __future__ import annotations

import os
import re
import subprocess
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[2]
GITLAB = REPO_ROOT / "ci/gitlab-ci.yml"
JENKINS = REPO_ROOT / "ci/Jenkinsfile"
MISMATCH_GO_VERSION = "0.0.0"


def module_go_version(path: Path) -> str:
    match = re.search(r"(?m)^go\s+(\d+\.\d+(?:\.\d+)?)\s*$", path.read_text())
    assert match, f"missing Go directive in {path}"
    return match.group(1)


def gitlab_go_job() -> tuple[str, list[str], str, str]:
    text = GITLAB.read_text()
    version_match = re.search(r'(?m)^  GO_VERSION:\s*"([^"]+)"$', text)
    toolchain_match = re.search(r'(?m)^  GOTOOLCHAIN:\s*"([^"]+)"$', text)
    job_match = re.search(r"(?ms)^test_go:\n(.*?)(?=^\S[^\n]*:\s*$|\Z)", text)
    assert version_match and toolchain_match and job_match, "GitLab Go job selectors missing"
    image_match = re.search(r"(?m)^  image:\s*golang:\$GO_VERSION\s*$", job_match.group(1))
    script_match = re.search(r"(?ms)^  script:\n((?:    - .*(?:\n|$))+)", job_match.group(1))
    assert image_match and script_match, "GitLab Go image or script missing"
    commands = re.findall(r"(?m)^    - (.*)$", script_match.group(1))
    return version_match.group(1), commands, toolchain_match.group(1), text


def jenkins_stage(stage_name: str, version_name: str) -> tuple[str, str]:
    text = JENKINS.read_text()
    environment = re.search(rf"(?m)^\s*{version_name}\s*=\s*'([^']+)'", text)
    stage = re.search(
        rf"(?ms)stage\('{re.escape(stage_name)}'\).*?sh \"\"\"(.*?)\"\"\"",
        text,
    )
    assert environment and stage, f"Jenkins {stage_name} selector or shell block missing"
    # Apply the stage's Groovy version interpolation and unescape shell dollars.
    shell = stage.group(1).replace("${env." + version_name + "}", environment.group(1))
    shell = shell.replace("\\$", "$")
    return environment.group(1), shell


def fixture_root(tmp_path: Path) -> Path:
    root = tmp_path / "checkout"
    (root / "mcp-server").mkdir(parents=True)
    (root / "cmd/team-cli").mkdir(parents=True)
    return root


def run_shell(
    tmp_path: Path,
    shell: str,
    *,
    version: str,
    extra_env: dict[str, str] | None = None,
) -> tuple[subprocess.CompletedProcess[str], Path, Path, Path, Path]:
    root = fixture_root(tmp_path)
    bin_dir = tmp_path / "bin"
    bin_dir.mkdir()
    calls = tmp_path / "calls.log"
    test_marker = tmp_path / "test-ran"
    cover_marker = tmp_path / "cover-ran"
    build_marker = tmp_path / "build-ran"

    fake_go = bin_dir / "go"
    fake_go.write_text(
        "#!/bin/sh\n"
        "printf 'go %s; selector=%s\\n' \"$*\" \"${GOTOOLCHAIN-}\" >> \"$FAKE_CALLS\"\n"
        "if [ \"$1\" = env ] && [ \"$2\" = GOVERSION ]; then\n"
        "  case \"${GOTOOLCHAIN-}\" in\n"
        "    go*+path) wanted=${GOTOOLCHAIN#go}; wanted=${wanted%+path}; "
        "if [ \"$FAKE_GO_VERSION\" != \"$wanted\" ]; then "
        "echo \"go: cannot find \\\"go$wanted\\\" in PATH\" >&2; exit 1; fi ;;\n"
        "  esac\n"
        "  printf 'go%s\\n' \"$FAKE_GO_VERSION\"; exit 0\n"
        "fi\n"
        "if [ \"$1\" = version ]; then echo \"go version go$FAKE_GO_VERSION linux/amd64\"; exit 0; fi\n"
        "if [ \"$1\" = test ]; then touch \"$FAKE_TEST_MARKER\"; exit 0; fi\n"
        "if [ \"$1\" = tool ] && [ \"$2\" = cover ]; then touch \"$FAKE_COVER_MARKER\"; exit 0; fi\n"
        "echo \"unexpected fake go invocation: $*\" >&2; exit 90\n"
    )
    fake_make = bin_dir / "make"
    fake_make.write_text(
        "#!/bin/sh\n"
        "[ \"$*\" = build ] || exit 91\n"
        "touch \"$FAKE_BUILD_MARKER\" team\n"
    )
    fake_go.chmod(0o755)
    fake_make.chmod(0o755)

    env = os.environ.copy()
    env.update(
        {
            "PATH": f"{bin_dir}{os.pathsep}{env.get('PATH', '')}",
            "FAKE_GO_VERSION": version,
            "FAKE_CALLS": str(calls),
            "FAKE_TEST_MARKER": str(test_marker),
            "FAKE_COVER_MARKER": str(cover_marker),
            "FAKE_BUILD_MARKER": str(build_marker),
        }
    )
    if extra_env:
        env.update(extra_env)
    result = subprocess.run(
        ["bash", "-c", shell], cwd=root, env=env, text=True, capture_output=True, check=False
    )
    return result, root, test_marker, cover_marker, build_marker


def test_gitlab_selector_runs_tests_and_rejects_mismatch_before_test(tmp_path: Path) -> None:
    version, commands, toolchain, text = gitlab_go_job()
    assert version == module_go_version(REPO_ROOT / "mcp-server/go.mod")
    assert toolchain == "local"
    assert "golang:$GO_VERSION" in text
    assert "-race" in " ".join(commands)
    assert "-coverprofile=coverage.out" in " ".join(commands)
    assert "go tool cover -func=coverage.out" in commands
    assert "example-module" not in text and "test_example" not in text

    shell = "set -eu\n" + "\n".join(commands)
    matched, _, test_marker, cover_marker, _ = run_shell(
        tmp_path / "match", shell, version=version,
        extra_env={"GO_VERSION": version, "GOTOOLCHAIN": toolchain},
    )
    assert matched.returncode == 0, matched.stderr
    assert test_marker.exists() and cover_marker.exists()

    mismatched, _, test_marker, cover_marker, _ = run_shell(
        tmp_path / "mismatch", shell, version=MISMATCH_GO_VERSION,
        extra_env={"GO_VERSION": version, "GOTOOLCHAIN": toolchain},
    )
    assert mismatched.returncode != 0
    assert f"Required Go {version}, got go{MISMATCH_GO_VERSION}" in mismatched.stderr
    assert not test_marker.exists() and not cover_marker.exists()


def test_jenkins_mcp_selector_runs_race_tests_and_rejects_mismatch(tmp_path: Path) -> None:
    version, shell = jenkins_stage("Test Go", "MCP_GO_VERSION")
    assert version == module_go_version(REPO_ROOT / "mcp-server/go.mod")
    assert "GOTOOLCHAIN=go${env.MCP_GO_VERSION}+path" in JENKINS.read_text()
    assert "-race" in shell and "-timeout=30m" in shell

    matched, _, test_marker, _, _ = run_shell(
        tmp_path / "match", shell, version=version,
        extra_env={"MCP_GO_VERSION": version},
    )
    assert matched.returncode == 0, matched.stderr
    assert test_marker.exists()

    mismatched, _, test_marker, _, _ = run_shell(
        tmp_path / "mismatch", shell, version=MISMATCH_GO_VERSION,
        extra_env={"MCP_GO_VERSION": version},
    )
    assert mismatched.returncode != 0
    assert f"Required Go {version} unavailable" in mismatched.stderr
    assert not test_marker.exists()


def test_jenkins_cli_selector_builds_artifact_and_rejects_missing_toolchain(tmp_path: Path) -> None:
    version, shell = jenkins_stage("Build CLI", "CLI_GO_VERSION")
    assert version == module_go_version(REPO_ROOT / "cmd/team-cli/go.mod")
    assert "GOTOOLCHAIN=go${env.CLI_GO_VERSION}+path" in JENKINS.read_text()
    assert "make build" in shell
    assert "archiveArtifacts artifacts: 'cmd/team-cli/team'" in JENKINS.read_text()

    matched, root, _, _, build_marker = run_shell(
        tmp_path / "match", shell, version=version,
        extra_env={"CLI_GO_VERSION": version},
    )
    assert matched.returncode == 0, matched.stderr
    assert build_marker.exists()
    assert (root / "cmd/team-cli/team").is_file()

    unavailable, root, _, _, build_marker = run_shell(
        tmp_path / "unavailable", shell, version=MISMATCH_GO_VERSION,
        extra_env={"CLI_GO_VERSION": version},
    )
    assert unavailable.returncode != 0
    assert f"Required Go {version} unavailable" in unavailable.stderr
    assert not build_marker.exists()
    assert not (root / "cmd/team-cli/team").exists()
