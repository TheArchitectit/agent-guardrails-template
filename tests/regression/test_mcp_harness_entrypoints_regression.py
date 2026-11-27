"""Regression coverage for the executable, fail-closed MCP smoke entry points."""

import os
from pathlib import Path
import subprocess


ROOT = Path(__file__).resolve().parents[2]


def run_entrypoint(name, *args, env=None):
    return subprocess.run(
        [str(ROOT / "scripts" / name), *args],
        cwd=ROOT,
        capture_output=True,
        text=True,
        env=env,
        check=False,
    )


def test_image_harness_is_executable_and_requires_a_receipt():
    help_result = run_entrypoint("run_mcp_image_codex_harness.sh", "--help")
    assert help_result.returncode == 0
    assert "--receipt FILE" in help_result.stdout

    result = run_entrypoint("run_mcp_image_codex_harness.sh")
    assert result.returncode == 2
    assert "--receipt is required" in result.stderr


def test_codex_smoke_is_executable_and_requires_a_synthetic_key():
    help_result = run_entrypoint("codex_mcp_runtime_smoke.sh", "--help")
    assert help_result.returncode == 0
    assert "GUARDRAILS_SMOKE_MCP_KEY" in help_result.stdout

    env = os.environ.copy()
    env.pop("GUARDRAILS_SMOKE_MCP_KEY", None)
    result = run_entrypoint("codex_mcp_runtime_smoke.sh", env=env)
    assert result.returncode == 2
    assert "GUARDRAILS_SMOKE_MCP_KEY is required" in result.stderr
