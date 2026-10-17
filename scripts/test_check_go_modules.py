#!/usr/bin/env python3
"""Red-on-drift self-test for scripts/check_go_modules.py (Spec 19 R19-10).

Proves the gate fails on an undocumented module, a stale inventory row, a
duplicate row, a missing command/floor field and an empty scan scope, and passes
on a complete inventory — so a green run cannot come from a gate that never
exercises its assertion.

Run with: python -m pytest scripts/test_check_go_modules.py -v
Or: python scripts/test_check_go_modules.py
"""

import contextlib
import io
import json
import os
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from scripts import check_go_modules as gate


def _entry(path: str, module: str | None = None, **over) -> dict:
    base = {
        "module": module or path,
        "path": path,
        "test_command": "go test ./... -count=1",
        "build_command": "go build ./...",
        "min_tests": 1,
        "applicability": "test fixture module",
    }
    base.update(over)
    return base


def _write_module(root: Path, rel: str) -> None:
    d = root / rel
    d.mkdir(parents=True, exist_ok=True)
    (d / "go.mod").write_text("module example.test/x\n\ngo 1.25\n", encoding="utf-8")


def _write_inv(root: Path, modules: list[dict]) -> None:
    d = root / ".guardrails"
    d.mkdir(parents=True, exist_ok=True)
    (d / "test-floor.json").write_text(
        json.dumps({"description": "fixture", "modules": modules}, indent=2),
        encoding="utf-8",
    )


class CheckGoModulesTest(unittest.TestCase):
    def _run(self, root: Path) -> int:
        with contextlib.redirect_stdout(io.StringIO()):
            return gate.main(["--repo", str(root)])

    def test_clean_when_every_module_covered(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _write_module(root, "svc-a")
            _write_module(root, "svc-b")
            _write_inv(root, [_entry("svc-a"), _entry("svc-b")])
            self.assertEqual(self._run(root), gate.EXIT_OK)

    def test_undocumented_module_fails(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _write_module(root, "svc-a")
            _write_module(root, "svc-new")
            _write_inv(root, [_entry("svc-a")])
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)

    def test_stale_inventory_row_fails(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _write_module(root, "svc-a")
            _write_inv(root, [_entry("svc-a"), _entry("svc-gone")])
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)

    def test_duplicate_row_fails(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _write_module(root, "svc-a")
            _write_inv(root, [_entry("svc-a"), _entry("svc-a")])
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)

    def test_missing_applicability_fails(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _write_module(root, "svc-a")
            _write_inv(root, [_entry("svc-a", applicability="")])
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)

    def test_nonpositive_floor_fails(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _write_module(root, "svc-a")
            _write_inv(root, [_entry("svc-a", min_tests=0)])
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)

    def test_missing_test_command_fails(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _write_module(root, "svc-a")
            _write_inv(root, [_entry("svc-a", test_command="  ")])
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)

    def test_vendored_module_is_not_discovered(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _write_module(root, "svc-a")
            _write_module(root, "vendor/example.test/dep")
            _write_inv(root, [_entry("svc-a")])
            self.assertEqual(self._run(root), gate.EXIT_OK)

    def test_non_vacuity_empty_scope_fails(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _write_inv(root, [_entry("svc-a")])
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)

    def test_missing_inventory_fails(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _write_module(root, "svc-a")
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)


if __name__ == "__main__":
    unittest.main()
