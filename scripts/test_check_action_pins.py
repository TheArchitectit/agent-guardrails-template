#!/usr/bin/env python3
"""Red-on-drift self-test for scripts/check_action_pins.py (Spec 19 R19-15).

Proves the gate fails on a mutable tag / missing ref / unpinned container, and
passes on a full-SHA pin — so a green run cannot come from a gate that never
exercises its assertion. Also proves non-vacuity (zero workflows fails).

Run with: python -m pytest scripts/test_check_action_pins.py -v
Or: python scripts/test_check_action_pins.py
"""

import os
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from scripts import check_action_pins as gate


def _write_workflow(root: Path, body: str, name: str = "ci.yml") -> None:
    d = root / ".github" / "workflows"
    d.mkdir(parents=True, exist_ok=True)
    (d / name).write_text(body, encoding="utf-8")


PINNED = "actions/checkout@11d5960a326750d5838078e36cf38b85af677262"
TAGGED = "actions/checkout@v4"


class CheckActionPinsTest(unittest.TestCase):
    def _run(self, root: Path) -> int:
        return gate.main(["--repo", str(root)])

    def test_clean_when_all_pinned(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _write_workflow(root, f"jobs:\n  a:\n    steps:\n      - uses: {PINNED}\n")
            self.assertEqual(self._run(root), gate.EXIT_OK)

    def test_local_composite_action_allowed(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _write_workflow(root, "jobs:\n  a:\n    steps:\n      - uses: ./.github/actions/x\n")
            self.assertEqual(self._run(root), gate.EXIT_OK)

    def test_mutable_tag_is_a_violation(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _write_workflow(root, f"jobs:\n  a:\n    steps:\n      - uses: {TAGGED}\n")
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)

    def test_missing_ref_is_a_violation(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _write_workflow(root, "jobs:\n  a:\n    steps:\n      - uses: actions/checkout\n")
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)

    def test_container_without_digest_is_a_violation(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _write_workflow(root, "jobs:\n  a:\n    steps:\n      - uses: docker://alpine:3.20\n")
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)

    def test_container_with_digest_allowed(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _write_workflow(
                root,
                "jobs:\n  a:\n    steps:\n      - uses: docker://alpine@sha256:"
                + "a" * 64 + "\n",
            )
            self.assertEqual(self._run(root), gate.EXIT_OK)

    def test_no_workflows_is_non_vacuous_failure(self):
        with tempfile.TemporaryDirectory() as td:
            self.assertEqual(self._run(Path(td)), gate.EXIT_VIOLATIONS)

    def test_workflow_with_zero_uses_is_non_vacuous_failure(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _write_workflow(root, "jobs:\n  a:\n    steps:\n      - run: echo hi\n")
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)


if __name__ == "__main__":
    unittest.main()
