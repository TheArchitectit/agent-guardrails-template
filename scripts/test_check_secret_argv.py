#!/usr/bin/env python3
"""Red-on-drift self-test for scripts/check_secret_argv.py (Spec 17 R17-08).

Proves the gate fails on a credential in argv (the ``redis-cli -a`` /
``redis-server --requirepass`` forms, inline and block-scalar) and passes on an
argv-free form — so a green run cannot come from a gate that never exercises its
assertion. Also proves non-vacuity (zero command sequences fails) and the named
limit (a container ``command:`` that writes a config file from the environment
is allowed; a *probe* expanding the same variable is not).

Run with: python -m pytest scripts/test_check_secret_argv.py -v
Or: python scripts/test_check_secret_argv.py
"""

import os
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from scripts import check_secret_argv as gate

# The gate's manifest list; a temp repo must contain every one (fail-closed).
MANIFESTS = [
    "docker-compose.yml",
    "mcp-server/deploy/podman-compose.yml",
    "mcp-server/deploy/docker-compose.example.yml",
    "mcp-server/deploy/k8s-deployment.yaml",
]

CLEAN = """\
services:
  redis:
    image: redis:7-alpine
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
  app:
    image: app:1
    healthcheck:
      test: ["CMD", "/server", "--health-check"]
"""

K8S_CLEAN = """\
spec:
  containers:
    - name: app
      livenessProbe:
        exec:
          command:
            - /server
            - --health-check
"""


class CheckSecretArgvTest(unittest.TestCase):
    def _repo(self, td: str, compose: str = CLEAN, k8s: str = K8S_CLEAN) -> Path:
        root = Path(td)
        for rel in MANIFESTS:
            p = root / rel
            p.parent.mkdir(parents=True, exist_ok=True)
            p.write_text(k8s if rel.endswith("k8s-deployment.yaml") else compose,
                         encoding="utf-8")
        return root

    def _run(self, root: Path) -> int:
        return gate.main(["--repo", str(root)])

    def test_clean_when_no_secret_in_argv(self):
        with tempfile.TemporaryDirectory() as td:
            self.assertEqual(self._run(self._repo(td)), gate.EXIT_OK)

    def test_fails_on_redis_cli_dash_a_inline(self):
        body = CLEAN.replace(
            '["CMD", "redis-cli", "ping"]',
            '["CMD", "redis-cli", "-a", "${REDIS_PASSWORD}", "ping"]',
        )
        with tempfile.TemporaryDirectory() as td:
            self.assertEqual(self._run(self._repo(td, body)), gate.EXIT_VIOLATIONS)

    def test_fails_on_requirepass_block_scalar(self):
        body = CLEAN + (
            "  extra:\n"
            "    command:\n"
            "      - redis-server\n"
            "      - --requirepass\n"
            "      - ${REDIS_PASSWORD}\n"
        )
        with tempfile.TemporaryDirectory() as td:
            self.assertEqual(self._run(self._repo(td, body)), gate.EXIT_VIOLATIONS)

    def test_fails_on_requirepass_equals_form(self):
        body = CLEAN + "  extra:\n    command: redis-server --requirepass=${REDIS_PASSWORD}\n"
        with tempfile.TemporaryDirectory() as td:
            self.assertEqual(self._run(self._repo(td, body)), gate.EXIT_VIOLATIONS)

    def test_fails_on_shell_string_redis_cli_dash_a(self):
        body = CLEAN.replace(
            '["CMD", "redis-cli", "ping"]',
            '["CMD-SHELL", "redis-cli -a \\"$REDIS_PASSWORD\\" ping | grep PONG"]',
        )
        with tempfile.TemporaryDirectory() as td:
            self.assertEqual(self._run(self._repo(td, body)), gate.EXIT_VIOLATIONS)

    def test_fails_on_k8s_probe_secret(self):
        k8s = K8S_CLEAN.replace(
            "            - --health-check\n",
            "            - --health-check\n            - ${DB_PASSWORD}\n",
        )
        with tempfile.TemporaryDirectory() as td:
            self.assertEqual(self._run(self._repo(td, k8s=k8s)), gate.EXIT_VIOLATIONS)

    def test_command_writing_config_from_env_is_allowed(self):
        # The accepted argv-free pattern: a container command that writes the
        # password into a config file from the environment. The value never
        # enters any program's argv, so this must PASS.
        body = CLEAN + (
            "  extra:\n"
            "    command: >\n"
            "      sh -c \"mkdir -p /tmp/redis && printf 'requirepass %s' \\\"$$REDIS_PASSWORD\\\" > /tmp/redis/redis.conf && exec redis-server /tmp/redis/redis.conf\"\n"
        )
        with tempfile.TemporaryDirectory() as td:
            self.assertEqual(self._run(self._repo(td, body)), gate.EXIT_OK)

    def test_probe_expanding_same_variable_still_fails(self):
        # Same variable, but in a probe: orchestrators inspect/log probe argv.
        body = CLEAN.replace(
            '["CMD", "redis-cli", "ping"]',
            '["CMD", "redis-cli", "-a", "${REDIS_PASSWORD}", "ping"]',
        )
        with tempfile.TemporaryDirectory() as td:
            self.assertEqual(self._run(self._repo(td, body)), gate.EXIT_VIOLATIONS)

    def test_non_vacuous_when_no_sequences(self):
        # No healthcheck/probe/command anywhere (compose AND k8s) must FAIL:
        # a green cannot come from an empty scan.
        with tempfile.TemporaryDirectory() as td:
            self.assertEqual(
                self._run(
                    self._repo(
                        td,
                        "services:\n  x:\n    image: y\n",
                        k8s="spec:\n  containers:\n    - name: app\n",
                    )
                ),
                gate.EXIT_VIOLATIONS,
            )

    def test_fail_closed_on_missing_manifest(self):
        with tempfile.TemporaryDirectory() as td:
            root = self._repo(td)
            (root / "docker-compose.yml").unlink()
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)


if __name__ == "__main__":
    unittest.main(verbosity=2)
