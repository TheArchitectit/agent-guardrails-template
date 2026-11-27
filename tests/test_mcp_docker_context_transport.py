"""Exercise Docker CLI client-side build context filtering without an engine.

The receiver accepts only the Docker API ping and build request. It records the
client-uploaded tar and closes the connection without returning a build result.
This proves context transport/filtering only; it is not an image build test.
"""

from __future__ import annotations

import http.server
import io
import os
import shutil
import socket
import subprocess
import tarfile
import tempfile
import threading
import unittest
from pathlib import Path

from mcp_docker_context_fixture import EXCLUDED, REQUIRED, write_synthetic_context

REPO_ROOT = Path(__file__).resolve().parents[1]
MCP_ROOT = REPO_ROOT / "mcp-server"


class _DockerApiReceiver(http.server.BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"
    archive: bytes | None = None
    received_path: str | None = None
    error: str | None = None

    def log_message(self, *_args) -> None:
        pass

    def do_GET(self) -> None:
        if self.path.endswith("/_ping") or self.path == "/_ping":
            body = b"OK"
            self.send_response(200)
            self.send_header("Api-Version", "1.41")
            self.send_header("Docker-Experimental", "false")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)
            return
        if self.path.endswith("/version") or self.path == "/version":
            body = b'{"ApiVersion":"1.41","MinAPIVersion":"1.24","Version":"fixture"}'
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)
            return
        self.send_error(404)

    def do_POST(self) -> None:
        if not self.path.split("?", 1)[0].endswith("/build"):
            self.send_error(404)
            return
        type(self).received_path = self.path
        try:
            if self.headers.get("Transfer-Encoding", "").lower() == "chunked":
                chunks = []
                while True:
                    line = self.rfile.readline().strip().split(b";", 1)[0]
                    size = int(line, 16)
                    if size == 0:
                        while self.rfile.readline().strip():
                            pass
                        break
                    chunks.append(self.rfile.read(size))
                    self.rfile.read(2)
                type(self).archive = b"".join(chunks)
            else:
                length = int(self.headers.get("Content-Length", "0"))
                type(self).archive = self.rfile.read(length)
            if not type(self).archive:
                type(self).error = "Docker CLI sent an empty request body"
        except Exception as exc:  # surfaced to the test thread
            type(self).error = repr(exc)
        finally:
            self.close_connection = True
            try:
                self.connection.shutdown(socket.SHUT_RDWR)
            except OSError:
                pass
            self.connection.close()


class DockerClientContextTransportTests(unittest.TestCase):
    def test_docker_cli_transmits_filtered_context_archive_without_engine(self) -> None:
        if not shutil.which("docker"):
            if os.environ.get("MCP_DOCKER_CONTEXT_REQUIRE_CLI") == "1":
                self.fail("The CI runner must have the Docker CLI for context transport verification")
            self.skipTest("Docker CLI is not installed")

        handler = _DockerApiReceiver
        handler.archive = None
        handler.received_path = None
        handler.error = None
        server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), handler)
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        try:
            with tempfile.TemporaryDirectory(prefix="mcp-context-transport-") as temporary:
                root = Path(temporary)
                context = root / "context"
                context.mkdir()
                write_synthetic_context(
                    context,
                    MCP_ROOT / ".dockerignore",
                    "FROM scratch\nCOPY . /context/\n",
                )
                dockerfile = context / "Dockerfile"
                config = root / "docker-config"
                config.mkdir()
                env = os.environ.copy()
                for key in ("DOCKER_CONTEXT", "DOCKER_HOST", "DOCKER_TLS_VERIFY", "DOCKER_CERT_PATH", "DOCKER_API_VERSION"):
                    env.pop(key, None)
                env.update({
                    "DOCKER_CONFIG": str(config),
                    "DOCKER_HOST": "tcp://127.0.0.1:%d" % server.server_address[1],
                    "DOCKER_API_VERSION": "1.41",
                    "DOCKER_BUILDKIT": "0",
                })
                result = subprocess.run(
                    ["docker", "build", "--quiet", "--file", str(dockerfile),
                     "--tag", "mcp-context-transport:synthetic", str(context)],
                    env=env, capture_output=True, text=True, timeout=60,
                )
            self.assertNotEqual(result.returncode, 0, "receiver intentionally does not fabricate a successful build response")
            self.assertIsNotNone(handler.received_path, "Docker CLI never sent the build context")
            self.assertTrue(handler.received_path.split("?", 1)[0].endswith("/build"))
            self.assertIsNone(handler.error, handler.error)
            self.assertTrue(handler.archive, "Docker CLI sent no context tar bytes")
            with tarfile.open(fileobj=io.BytesIO(handler.archive), mode="r:*") as archive:
                paths = {member.name.removeprefix("./").rstrip("/") for member in archive.getmembers()}
            for relative in EXCLUDED:
                self.assertNotIn(relative, paths, "excluded path reached Docker API: " + relative)
            for relative in REQUIRED + ("Dockerfile",):
                self.assertIn(relative, paths, "required path absent from Docker API tar: " + relative)
        finally:
            server.shutdown()
            server.server_close()
            thread.join(timeout=2)


if __name__ == "__main__":
    unittest.main()
