#!/usr/bin/env python3
"""Red-on-drift self-test for scripts/check_python_tests.py (Spec 19 R19-10).

Proves the gate fails on an undocumented suite, a stale inventory row, a
duplicate row, a missing command/floor field and an empty scan scope, and passes
on a complete inventory — so a green run cannot come from a gate that never
exercises its assertion.

Run with: python -m pytest scripts/test_check_python_tests.py -v
Or: python scripts/test_check_python_tests.py
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

from scripts import check_python_tests as gate


def _entry(path: str, suite: str | None = None, **over) -> dict:
    base = {
        "suite": suite or path,
        "path": path,
        "test_command": "python3 -m pytest " + path + "/ -q",
        "min_tests": 1,
        "applicability": "test fixture suite",
    }
    base.update(over)
    return base


def _write_suite(root: Path, rel: str, fname: str = "test_example.py") -> None:
    d = root / rel
    d.mkdir(parents=True, exist_ok=True)
    (d / fname).write_text("def test_ok():\n    assert True\n", encoding="utf-8")


def _write_inv(root: Path, suites: list[dict]) -> None:
    d = root / ".guardrails"
    d.mkdir(parents=True, exist_ok=True)
    (d / "python-test-floor.json").write_text(
        json.dumps({"description": "fixture", "suites": suites}, indent=2),
        encoding="utf-8",
    )


class CheckPythonTestsTest(unittest.TestCase):
    def _run(self, root: Path) -> int:
        with contextlib.redirect_stdout(io.StringIO()):
            return gate.main(["--repo", str(root)])

    def test_clean_when_every_suite_covered(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _write_suite(root, "scripts")
            _write_suite(root, "examples/python/tests")
            _write_inv(root, [_entry("scripts"), _entry("examples/python/tests")])
            self.assertEqual(self._run(root), gate.EXIT_OK)

    def test_undocumented_suite_fails(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _write_suite(root, "scripts")
            _write_suite(root, "new/tests")
            _write_inv(root, [_entry("scripts")])
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)

    def test_stale_inventory_row_fails(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _write_suite(root, "scripts")
            _write_inv(root, [_entry("scripts"), _entry("gone/tests")])
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)

    def test_duplicate_row_fails(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _write_suite(root, "scripts")
            _write_inv(root, [_entry("scripts"), _entry("scripts")])
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)

    def test_missing_applicability_fails(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _write_suite(root, "scripts")
            _write_inv(root, [_entry("scripts", applicability="")])
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)

    def test_nonpositive_floor_fails(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _write_suite(root, "scripts")
            _write_inv(root, [_entry("scripts", min_tests=0)])
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)

    def test_missing_test_command_fails(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _write_suite(root, "scripts")
            _write_inv(root, [_entry("scripts", test_command="  ")])
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)

    def test_empty_setup_command_fails(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _write_suite(root, "scripts")
            _write_inv(root, [_entry("scripts", setup_command="  ")])
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)

    def test_omitted_setup_command_is_allowed(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _write_suite(root, "scripts")
            _write_inv(root, [_entry("scripts")])
            self.assertEqual(self._run(root), gate.EXIT_OK)

    def test_underscore_test_filename_is_discovered(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _write_suite(root, "svc", fname="loader_test.py")
            _write_inv(root, [_entry("svc")])
            self.assertEqual(self._run(root), gate.EXIT_OK)

    def test_illustrative_uppercase_template_is_not_discovered(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _write_suite(root, "scripts")
            # EXAMPLE_REGRESSION_TEST.py is not a pytest python_files match.
            _write_suite(root, "examples/regression-prevention", fname="EXAMPLE_REGRESSION_TEST.py")
            _write_inv(root, [_entry("scripts")])
            self.assertEqual(self._run(root), gate.EXIT_OK)

    def test_cached_or_vendored_suite_is_not_discovered(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _write_suite(root, "scripts")
            _write_suite(root, ".venv/lib/tests")
            _write_suite(root, "node_modules/pkg")
            _write_inv(root, [_entry("scripts")])
            self.assertEqual(self._run(root), gate.EXIT_OK)

    def test_non_vacuity_empty_scope_fails(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _write_inv(root, [_entry("scripts")])
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)

    def test_missing_inventory_fails(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _write_suite(root, "scripts")
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)


if __name__ == "__main__":
    unittest.main()
