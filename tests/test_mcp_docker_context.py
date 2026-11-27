"""Regression check for the MCP build context boundary via explicit engines."""

from __future__ import annotations

import io
import json
import os
import re
import shutil
import subprocess
import tarfile
import tempfile
import unittest
import uuid
from pathlib import Path

from mcp_docker_context_fixture import EXCLUDED, REQUIRED, write_synthetic_context


REPO_ROOT = Path(__file__).resolve().parents[1]
MCP_ROOT = REPO_ROOT / "mcp-server"
DOCKERFILE = """FROM scratch
COPY . /context/
"""
OWNER_LABEL = "io.guardrails.test.context-owner"
DAEMON_SKIP_REASON = "Set MCP_DOCKER_CONTEXT_ENGINE=docker or podman to explicitly select a live engine"


class EngineUnavailable(RuntimeError):
    pass


def engine_requested(environment: dict[str, str] | None = None) -> bool:
    environment = os.environ if environment is None else environment
    return (environment.get("MCP_DOCKER_CONTEXT_DAEMON") == "available"
            or "MCP_DOCKER_CONTEXT_ENGINE" in environment)


def resolve_engine(environment: dict[str, str] | None = None, which=shutil.which) -> tuple[str, str]:
    """Select only Docker (legacy default) or Podman; explicit missing engines fail."""
    environment = os.environ if environment is None else environment
    engine = environment.get("MCP_DOCKER_CONTEXT_ENGINE", "docker").strip().lower()
    if engine not in {"docker", "podman"}:
        raise EngineUnavailable("MCP_DOCKER_CONTEXT_ENGINE must be docker or podman")
    executable = which(engine)
    if not executable:
        raise EngineUnavailable("explicitly selected container engine is unavailable: %s" % engine)
    return engine, executable


def labels_from_inspect(document: dict) -> dict:
    """Read Docker and Podman image/container label layouts."""
    config = document.get("Config") or {}
    return config.get("Labels") or document.get("Labels") or {}


def inspector_build_argv(context: Path, image: str, engine: str = "docker", owner: str | None = None) -> list[str]:
    context = context.resolve()
    command = [engine, "build", "--quiet", "--file", str((context / "Dockerfile").resolve())]
    if owner:
        command.extend(["--label", f"{OWNER_LABEL}={owner}"])
    command.extend(["--tag", image, str(context)])
    return command


def inspector_create_argv(image: str, engine: str = "docker", owner: str | None = None) -> list[str]:
    # create only: neither engine starts the synthetic scratch container.
    command = [engine, "create"]
    if owner:
        command.extend(["--label", f"{OWNER_LABEL}={owner}"])
    command.extend([image, "/context/.inspect-only-never-run"])
    return command


def inspect_json(executable: str, resource_type: str, resource: str):
    result = subprocess.run([executable, resource_type, "inspect", resource], capture_output=True, text=True)
    if result.returncode:
        details = result.stderr.lower()
        if resource_type == "image" and re.search(r"no such image|image not known|not found", details):
            return None
        raise AssertionError("%s inspect failed: %s" % (resource_type, result.stderr.strip()))
    data = json.loads(result.stdout)
    return data[0] if isinstance(data, list) else data


def rootless_podman(executable: str) -> bool:
    result = subprocess.run(
        [executable, "info", "--format", "{{.Host.Security.Rootless}}"],
        capture_output=True,
        text=True,
    )
    return result.returncode == 0 and result.stdout.strip().lower() == "true"


def engine_version(executable: str) -> str:
    result = subprocess.run([executable, "--version"], capture_output=True, text=True)
    if result.returncode:
        raise AssertionError("could not read engine version: %s" % result.stderr.strip())
    return result.stdout.strip()


class DockerContextTests(unittest.TestCase):
    def test_inspector_commands_use_absolute_dockerfile_and_unstarted_command(self) -> None:
        context = Path("/tmp/mcp-context-fixture")
        build = inspector_build_argv(context, "mcp-context-check:test")
        self.assertEqual(build[0], "docker")
        self.assertEqual(build[build.index("--file") + 1], str((context / "Dockerfile").resolve()))
        self.assertTrue(Path(build[build.index("--file") + 1]).is_absolute())
        self.assertEqual(
            inspector_create_argv("mcp-context-check:test"),
            ["docker", "create", "mcp-context-check:test", "/context/.inspect-only-never-run"],
        )

    def test_owner_labels_scope_cleanup_to_this_run(self) -> None:
        owner = "unique-owner-token"
        build = inspector_build_argv(Path("/tmp/context"), "image:test", "podman", owner)
        create = inspector_create_argv("image:test", "podman", owner)
        self.assertIn(f"{OWNER_LABEL}={owner}", build)
        self.assertIn(f"{OWNER_LABEL}={owner}", create)
        self.assertEqual(create[-1], "/context/.inspect-only-never-run")
        self.assertNotIn("start", create)

    def test_required_dockerfile_inputs_exist_and_have_explicit_keep_rules(self) -> None:
        ignore_lines = (MCP_ROOT / ".dockerignore").read_text(encoding="utf-8").splitlines()
        ignore_rules = set(ignore_lines)
        required_keep_rules = {
            "!go.mod", "!go.sum", "!**/*.go", "!web/", "!web/**", "!.guardrails/",
            "!.guardrails/prevention-rules/", "!.guardrails/prevention-rules/**",
        }
        self.assertTrue(required_keep_rules.issubset(ignore_rules))
        keep_positions = [ignore_lines.index(rule) for rule in required_keep_rules]
        deny_positions = [ignore_lines.index(rule) for rule in (".git", ".env", "**/.env", ".cache/", "**/.cache/")]
        self.assertLess(max(keep_positions), min(deny_positions))
        for relative in (
            "go.mod", "go.sum", "cmd/server/main.go", "web/index.html",
            ".guardrails/prevention-rules/extracted-rules.json",
        ):
            self.assertTrue((MCP_ROOT / relative).exists(), f"Dockerfile input is missing: {relative}")

    def test_engine_selector_preserves_docker_default_and_supports_podman(self) -> None:
        self.assertEqual(("docker", "/usr/bin/docker"),
                         resolve_engine({}, lambda name: "/usr/bin/docker" if name == "docker" else None))
        self.assertEqual(("podman", "/usr/bin/podman"),
                         resolve_engine({"MCP_DOCKER_CONTEXT_ENGINE": "podman"}, lambda name: "/usr/bin/podman" if name == "podman" else None))

    def test_explicit_missing_engine_is_an_error(self) -> None:
        with self.assertRaisesRegex(EngineUnavailable, "unavailable"):
            resolve_engine({"MCP_DOCKER_CONTEXT_ENGINE": "podman"}, lambda _name: None)

    def test_unknown_engine_is_an_error(self) -> None:
        with self.assertRaisesRegex(EngineUnavailable, "must be docker or podman"):
            resolve_engine({"MCP_DOCKER_CONTEXT_ENGINE": "dockerish"}, lambda _name: "/bin/nope")

    def test_engine_opt_in_is_required_unless_backend_is_explicit(self) -> None:
        self.assertFalse(engine_requested({}))
        self.assertTrue(engine_requested({"MCP_DOCKER_CONTEXT_DAEMON": "available"}))
        self.assertTrue(engine_requested({"MCP_DOCKER_CONTEXT_ENGINE": "podman"}))

    def test_owner_label_parser_handles_engine_inspect_shapes(self) -> None:
        self.assertEqual({OWNER_LABEL: "one"}, labels_from_inspect({"Config": {"Labels": {OWNER_LABEL: "one"}}}))
        self.assertEqual({OWNER_LABEL: "two"}, labels_from_inspect({"Labels": {OWNER_LABEL: "two"}}))

    @unittest.skipUnless(engine_requested(), DAEMON_SKIP_REASON)
    def test_local_only_files_are_excluded_and_build_inputs_remain(self) -> None:
        engine, executable = resolve_engine()
        version = engine_version(executable)
        rootless = None
        if engine == "podman":
            rootless = rootless_podman(executable)
            self.assertTrue(rootless, "selected Podman backend must be rootless")

        with tempfile.TemporaryDirectory(prefix="mcp-context-") as temporary:
            context = Path(temporary) / "context"
            context.mkdir()
            write_synthetic_context(context, MCP_ROOT / ".dockerignore", DOCKERFILE)

            owner = uuid.uuid4().hex
            image = f"mcp-context-check:{owner}"
            container = None
            build_attempted = False
            cleanup = {"container_removed": False, "image_removed": False}
            receipt_path = os.environ.get("MCP_DOCKER_CONTEXT_RECEIPT")
            try:
                existing = inspect_json(executable, "image", image)
                self.assertIsNone(existing, "refusing to overwrite an existing image tag")
                build_attempted = True
                build = subprocess.run(
                    inspector_build_argv(context, image, engine, owner),
                    capture_output=True,
                    text=True,
                )
                self.assertEqual(build.returncode, 0, "inspector build failed: " + build.stderr[-2000:])

                image_doc = inspect_json(executable, "image", image)
                self.assertIsNotNone(image_doc, "engine did not retain the built inspector image")
                self.assertEqual(labels_from_inspect(image_doc).get(OWNER_LABEL), owner)

                created = subprocess.run(
                    inspector_create_argv(image, engine, owner),
                    check=True,
                    capture_output=True,
                    text=True,
                )
                container = created.stdout.strip()
                self.assertTrue(container, "engine returned no container ID")
                container_doc = inspect_json(executable, "container", container)
                self.assertEqual(labels_from_inspect(container_doc).get(OWNER_LABEL), owner)
                self.assertFalse(container_doc.get("State", {}).get("Running", False), "inspector container must remain unstarted")

                exported = subprocess.run([executable, "export", container], check=True, capture_output=True)
                with tarfile.open(fileobj=io.BytesIO(exported.stdout), mode="r:") as archive:
                    paths = {member.name.removeprefix("./") for member in archive.getmembers()}
                for relative in EXCLUDED:
                    self.assertNotIn(f"context/{relative}", paths, f"excluded path leaked: {relative}")
                for relative in REQUIRED:
                    self.assertIn(f"context/{relative}", paths, f"required path missing: {relative}")
            finally:
                cleanup_errors = []
                if container:
                    try:
                        container_doc = inspect_json(executable, "container", container)
                        if labels_from_inspect(container_doc).get(OWNER_LABEL) != owner:
                            cleanup_errors.append("refused to remove a container without this run's ownership label")
                        else:
                            removed = subprocess.run([executable, "rm", container], capture_output=True, text=True)
                            if removed.returncode:
                                cleanup_errors.append("container cleanup failed: " + removed.stderr.strip())
                            else:
                                cleanup["container_removed"] = True
                    except (AssertionError, ValueError) as error:
                        cleanup_errors.append("container cleanup verification failed: " + str(error))
                if build_attempted:
                    try:
                        image_doc = inspect_json(executable, "image", image)
                        if image_doc is not None:
                            if labels_from_inspect(image_doc).get(OWNER_LABEL) != owner:
                                cleanup_errors.append("refused to remove an image without this run's ownership label")
                            else:
                                removed = subprocess.run([executable, "image", "rm", image], capture_output=True, text=True)
                                if removed.returncode:
                                    cleanup_errors.append("image cleanup failed: " + removed.stderr.strip())
                                else:
                                    cleanup["image_removed"] = True
                    except (AssertionError, ValueError) as error:
                        cleanup_errors.append("image cleanup verification failed: " + str(error))

                if receipt_path:
                    receipt = {
                        "engine": engine,
                        "engine_version": version,
                        "rootless": rootless,
                        "image_tag": image,
                        "container_id": container,
                        "container_was_started": False,
                        "cleanup": cleanup,
                        "cleanup_errors": cleanup_errors,
                    }
                    receipt_file = Path(receipt_path)
                    receipt_file.parent.mkdir(parents=True, exist_ok=True)
                    receipt_file.write_text(json.dumps(receipt, indent=2) + "\n", encoding="utf-8")
                self.assertFalse(cleanup_errors, "; ".join(cleanup_errors))


if __name__ == "__main__":
    unittest.main()
