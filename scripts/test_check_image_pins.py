#!/usr/bin/env python3
"""Red-on-drift self-test for scripts/check_image_pins.py (Spec 19 R19-15 / Spec 17 R17-11).

Proves the gate fails on an unpinned (tag-only) image, a digest that disagrees
with the reviewed registry, and an unreviewed pin — and passes on a
digest-pinned ref that matches the registry. Also proves non-vacuity (zero
external image refs fails) and that a first-party build image is allowed.

Run with: python -m pytest scripts/test_check_image_pins.py -v
Or: python scripts/test_check_image_pins.py
"""

import json
import os
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from scripts import check_image_pins as gate

DIGEST_A = "sha256:" + "a" * 64
DIGEST_B = "sha256:" + "b" * 64


def _scaffold(root: Path, registry: dict, compose_image: str, extra_registry: dict | None = None) -> None:
    """Write a minimal but complete manifest set + registry under ``root``.

    The auxiliary manifests are always pinned to DIGEST_A so the only ref under
    test is the one in ``docker-compose.yml``.
    """
    reg = dict(registry)
    reg.setdefault("docker.io/library/busybox:1.36", DIGEST_A)
    reg.setdefault("docker.io/library/alpine:3.20", DIGEST_A)
    (root / "image-pins.json").write_text(
        json.dumps({"images": reg}), encoding="utf-8"
    )
    # The gate requires every manifest in MANIFESTS to exist (fail-closed).
    (root / "docker-compose.yml").write_text(
        f"services:\n  redis:\n    image: {compose_image}\n", encoding="utf-8"
    )
    for rel in gate.MANIFESTS:
        if rel == "docker-compose.yml":
            continue
        p = root / rel
        p.parent.mkdir(parents=True, exist_ok=True)
        if rel.endswith("Dockerfile"):
            p.write_text(f"FROM alpine:3.20@{DIGEST_A}\n", encoding="utf-8")
        elif rel.endswith("k8s-deployment.yaml"):
            p.write_text("containers:\n  - image: guardrail-mcp:${VERSION}\n", encoding="utf-8")
        else:
            p.write_text(f"services:\n  x:\n    image: busybox:1.36@{DIGEST_A}\n", encoding="utf-8")


class CheckImagePinsTest(unittest.TestCase):
    def _run(self, root: Path) -> int:
        return gate.main(["--repo", str(root)])

    def test_clean_when_pinned_and_matching(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _scaffold(
                root,
                {"docker.io/library/redis:7-alpine": DIGEST_A},
                f"redis:7-alpine@{DIGEST_A}",
            )
            self.assertEqual(self._run(root), gate.EXIT_OK)

    def test_tag_only_is_a_violation(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _scaffold(root, {}, "redis:7-alpine")
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)

    def test_digest_mismatch_is_a_violation(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _scaffold(
                root,
                {"docker.io/library/redis:7-alpine": DIGEST_A},
                f"redis:7-alpine@{DIGEST_B}",
            )
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)

    def test_unreviewed_pin_is_a_violation(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _scaffold(root, {}, f"redis:7-alpine@{DIGEST_A}")
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)

    def test_first_party_build_image_allowed(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            # The compose image is first-party; every other manifest is pinned.
            _scaffold(root, {}, "guardrail-mcp:${VERSION:-latest}")
            self.assertEqual(self._run(root), gate.EXIT_OK)

    def test_missing_registry_fails_closed(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            (root / "docker-compose.yml").write_text(
                "services:\n  r:\n    image: redis:7-alpine\n", encoding="utf-8"
            )
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)

    def test_missing_manifest_fails_closed(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            (root / "image-pins.json").write_text(
                json.dumps({"images": {"docker.io/library/redis:7-alpine": DIGEST_A}}),
                encoding="utf-8",
            )
            # No manifests at all -> non-vacuous failure.
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)

    def test_empty_registry_is_a_violation(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _scaffold(root, {}, "guardrail-mcp:${VERSION}")
            # Force an empty registry to prove the non-vacuity guard fires.
            (root / "image-pins.json").write_text(
                json.dumps({"images": {}}), encoding="utf-8"
            )
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)

    def test_canonical_normalizes_short_names(self):
        self.assertEqual(gate.canonical("redis:7-alpine")[0], "docker.io/library/redis")
        self.assertEqual(
            gate.canonical("gcr.io/distroless/static:nonroot")[0],
            "gcr.io/distroless/static",
        )
        self.assertEqual(
            gate.canonical("guardrail-mcp:${VERSION:-latest}")[0],
            "docker.io/library/guardrail-mcp",
        )


if __name__ == "__main__":
    unittest.main()
