"""Regression check for the MCP Docker build context boundary."""

from __future__ import annotations

import io
import os
import shutil
import subprocess
import tarfile
import tempfile
import unittest
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[1]
MCP_ROOT = REPO_ROOT / "mcp-server"
DOCKERFILE = """FROM scratch
COPY . /context/
"""


def inspector_build_argv(context: Path, image: str) -> list[str]:
    context = context.resolve()
    return [
        "docker",
        "build",
        "--quiet",
        "--file",
        str((context / "Dockerfile").resolve()),
        "--tag",
        image,
        str(context),
    ]


def inspector_create_argv(image: str) -> list[str]:
    # The container is exported without ever being started. This placeholder
    # command allows `docker create` to succeed for a FROM scratch image.
    return ["docker", "create", image, "/context/.inspect-only-never-run"]


class DockerContextTests(unittest.TestCase):
    def test_inspector_commands_use_absolute_dockerfile_and_unstarted_command(self) -> None:
        context = Path("/tmp/mcp-context-fixture")
        build = inspector_build_argv(context, "mcp-context-check:test")
        self.assertEqual(
            build[build.index("--file") + 1],
            str((context / "Dockerfile").resolve()),
        )
        self.assertTrue(Path(build[build.index("--file") + 1]).is_absolute())
        self.assertEqual(
            inspector_create_argv("mcp-context-check:test"),
            ["docker", "create", "mcp-context-check:test", "/context/.inspect-only-never-run"],
        )

    def test_required_dockerfile_inputs_exist_and_have_explicit_keep_rules(self) -> None:
        ignore_lines = (MCP_ROOT / ".dockerignore").read_text(encoding="utf-8").splitlines()
        ignore_rules = set(ignore_lines)
        required_keep_rules = {
            "!go.mod",
            "!go.sum",
            "!**/*.go",
            "!web/",
            "!web/**",
            "!.guardrails/",
            "!.guardrails/prevention-rules/",
            "!.guardrails/prevention-rules/**",
        }
        self.assertTrue(required_keep_rules.issubset(ignore_rules))
        keep_positions = [ignore_lines.index(rule) for rule in required_keep_rules]
        deny_positions = [
            ignore_lines.index(rule)
            for rule in (".git", ".env", "**/.env", ".cache/", "**/.cache/")
        ]
        self.assertLess(max(keep_positions), min(deny_positions))
        for relative in (
            "go.mod",
            "go.sum",
            "cmd/server/main.go",
            "web/index.html",
            ".guardrails/prevention-rules/extracted-rules.json",
        ):
            self.assertTrue((MCP_ROOT / relative).exists(), f"Dockerfile input is missing: {relative}")

    @unittest.skipUnless(
        shutil.which("docker") and os.environ.get("MCP_DOCKER_CONTEXT_DAEMON") == "available",
        "Docker context inspection requires an explicitly available daemon",
    )
    def test_local_only_files_are_excluded_and_build_inputs_remain(self) -> None:
        with tempfile.TemporaryDirectory(prefix="mcp-context-") as temporary:
            context = Path(temporary) / "context"
            context.mkdir()
            shutil.copy2(MCP_ROOT / ".dockerignore", context / ".dockerignore")

            excluded = [
                ".env",
                ".git/config",
                ".cache/fixture",
                "__pycache__/fixture.pyc",
                "web/.env",
                "web/.git/config",
                "web/.cache/fixture",
                ".guardrails/prevention-rules/.env",
                ".guardrails/prevention-rules/.git/config",
                ".guardrails/prevention-rules/.cache/fixture",
            ]
            required = [
                "go.mod",
                "go.sum",
                "cmd/server/main.go",
                "web/index.html",
                ".guardrails/prevention-rules/extracted-rules.json",
            ]
            for relative in excluded + required:
                path = context / relative
                path.parent.mkdir(parents=True, exist_ok=True)
                # Synthetic fixture content is deliberately inert. Never print it.
                path.write_text("synthetic-context-fixture\n", encoding="utf-8")

            dockerfile = context / "Dockerfile"
            dockerfile.write_text(DOCKERFILE, encoding="utf-8")
            image = f"mcp-context-check:{os.getpid()}"
            container = None
            try:
                build = subprocess.run(
                    inspector_build_argv(context, image),
                    stdout=subprocess.DEVNULL,
                    stderr=subprocess.DEVNULL,
                )
                self.assertEqual(build.returncode, 0, "Docker context inspector build failed")

                created = subprocess.run(
                    inspector_create_argv(image),
                    check=True,
                    capture_output=True,
                    text=True,
                )
                container = created.stdout.strip()
                exported = subprocess.run(
                    ["docker", "export", container], check=True, capture_output=True
                )
                with tarfile.open(fileobj=io.BytesIO(exported.stdout), mode="r:") as archive:
                    paths = {member.name.removeprefix("./") for member in archive.getmembers()}

                for relative in excluded:
                    self.assertNotIn(f"context/{relative}", paths, f"excluded path leaked: {relative}")
                for relative in required:
                    self.assertIn(f"context/{relative}", paths, f"required path missing: {relative}")
            finally:
                if container:
                    subprocess.run(["docker", "rm", container], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
                subprocess.run(["docker", "image", "rm", image], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)


if __name__ == "__main__":
    unittest.main()
