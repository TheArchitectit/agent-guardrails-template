#!/usr/bin/env python3
"""check_python_tests.py — Spec 19 R19-10 Python suite discovery gate.

Spec 19 R19-10: *"The package gate MUST enumerate modules from the repository
and execute each Go module, rather than assuming a ``go.work`` workspace. It MUST
run Python tests and all applicable Go tests/builds, record discovered and
executed test counts, and fail on compilation, race, test, or package-discovery
errors."*

The Go half is enforced by ``scripts/check_go_modules.py``. The Python half was
not: CI ran a single hardcoded ``pytest scripts/`` invocation, so any other
Python test suite in the tree — a new ``tests/`` directory, a nested tool, the
``examples/python`` loader suite — silently escaped every Python test run. That
is the same defect class the Go gate exists to close, one language over: a
coverage list hardcoded in the workflow instead of derived from the tree.

This gate closes that gap fail-closed:

  1. **Discovery.** Every directory in the repository that directly contains a
     pytest-collectable file (``test_*.py`` or ``*_test.py``, pytest's default
     ``python_files``) is discovered from the tree, excluding VCS metadata,
     dependency/vendor trees and caches.
  2. **Coverage.** Each discovered suite MUST have an inventory entry in
     ``.guardrails/python-test-floor.json``. An undocumented suite — one CI
     would not run — is a FAIL.
  3. **No stale rows.** Each inventory entry MUST name a suite that still
     exists. A row for a removed suite is a FAIL (the inventory may not drift
     away from the tree in either direction).
  4. **Declared command + positive floor.** Each row MUST declare a non-empty
     ``test_command``, a positive integer ``min_tests`` and a non-empty
     ``applicability`` reason (R19-12's "declare each required package, test
     command, positive minimum count and applicability condition"). A
     ``setup_command`` is optional but, when present, must be a non-empty
     string; it is how a suite declares its own test dependencies, so the
     runner does not have to know any suite's imports in advance.
  5. **Non-vacuity.** A discovery that finds zero suites FAILS; a green result
     may never come from an empty scan scope.

Fail-closed: a missing or unreadable inventory FAILS.

Exit: 0 clean; 1 a coverage/format/vacuity violation; 2 usage.
"""
from __future__ import annotations

import argparse
import fnmatch
import json
import sys
from pathlib import Path

EXIT_OK = 0
EXIT_VIOLATIONS = 1
EXIT_USAGE = 2

INVENTORY = Path(".guardrails/python-test-floor.json")

# Directory names never descended into: VCS metadata, JS dependencies, vendored
# source, virtualenvs and caches (a cache is not a test suite of this repo).
SKIP_DIRS = {
    ".git",
    "node_modules",
    "vendor",
    "testdata",
    "__pycache__",
    ".pytest_cache",
    ".mypy_cache",
    ".ruff_cache",
    ".venv",
    "venv",
    ".tox",
}

# pytest's default python_files, matched case-sensitively so illustrative
# templates (e.g. EXAMPLE_REGRESSION_TEST.py) are not mistaken for live suites.
TEST_FILE_PATTERNS = ("test_*.py", "*_test.py")

REQUIRED_STRING_KEYS = ("suite", "path", "test_command", "applicability")


def _is_test_file(name: str) -> bool:
    return any(fnmatch.fnmatchcase(name, pat) for pat in TEST_FILE_PATTERNS)


def discover_suites(repo: Path) -> tuple[set[str], list[str]]:
    """Return (relative suite dirs holding collectable tests, errors)."""
    found: set[str] = set()
    errors: list[str] = []
    if not repo.is_dir():
        return found, [f"{repo}: not a directory (fail-closed)"]

    for path in sorted(repo.rglob("*.py")):
        rel = path.relative_to(repo)
        if any(part in SKIP_DIRS for part in rel.parts[:-1]):
            continue
        if not _is_test_file(path.name):
            continue
        suite_dir = rel.parent.as_posix()
        found.add("." if suite_dir == "." else suite_dir)
    return found, errors


def load_inventory(path: Path) -> tuple[dict[str, dict], list[str]]:
    """Return (suite path -> entry, fail-closed errors)."""
    rows: dict[str, dict] = {}
    errors: list[str] = []
    if not path.is_file():
        return rows, [f"{path}: inventory not found (fail-closed)"]
    try:
        doc = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        return rows, [f"{path}: cannot read/parse ({exc})"]

    suites = doc.get("suites")
    if not isinstance(suites, list) or not suites:
        return rows, [f"{path}: 'suites' must be a non-empty list (fail-closed)"]

    for idx, entry in enumerate(suites):
        if not isinstance(entry, dict):
            errors.append(f"{path}: suites[{idx}] is not an object")
            continue
        missing = [k for k in REQUIRED_STRING_KEYS if not str(entry.get(k, "")).strip()]
        if missing:
            errors.append(
                f"{path}: suites[{idx}] ({entry.get('suite', '?')}) is missing "
                f"required field(s): {', '.join(missing)}"
            )
        floor = entry.get("min_tests")
        if not isinstance(floor, int) or isinstance(floor, bool) or floor <= 0:
            errors.append(
                f"{path}: suites[{idx}] ({entry.get('suite', '?')}) must declare "
                f"a positive integer min_tests (got {floor!r})"
            )
        if "setup_command" in entry and not str(entry.get("setup_command", "")).strip():
            errors.append(
                f"{path}: suites[{idx}] ({entry.get('suite', '?')}) has an empty "
                f"setup_command (omit the key or give a non-empty command)"
            )
        raw_path = entry.get("path")
        if not isinstance(raw_path, str) or not raw_path.strip():
            continue
        norm = raw_path.strip().rstrip("/")
        if norm in rows:
            errors.append(f"{path}: duplicate row for suite path {norm}")
        rows[norm] = entry
    return rows, errors


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", default=".", help="repository root")
    args = parser.parse_args(argv)

    repo = Path(args.repo).resolve()
    discovered, errors = discover_suites(repo)
    rows, inv_errors = load_inventory(repo / INVENTORY)
    errors.extend(inv_errors)

    if not discovered:
        errors.append(
            "non-vacuity: no Python test suite discovered in the repository "
            "(the scan scope is empty)"
        )

    for suite_dir in sorted(discovered):
        if suite_dir not in rows:
            errors.append(
                f"{INVENTORY}: suite {suite_dir} has no inventory row "
                f"(a discovered Python test suite would escape CI)"
            )

    for suite_dir in sorted(rows):
        if suite_dir not in discovered:
            errors.append(
                f"{INVENTORY}: row for suite {suite_dir} has no collectable test "
                f"files in the repository (stale inventory row)"
            )

    if errors:
        print("PYTHON TEST DISCOVERY GATE: FAIL")
        for err in errors:
            print(f"  - {err}")
        return EXIT_VIOLATIONS

    print(
        f"PYTHON TEST DISCOVERY GATE: OK "
        f"({len(discovered)} discovered suites, all covered by {INVENTORY})"
    )
    return EXIT_OK


if __name__ == "__main__":
    sys.exit(main())
