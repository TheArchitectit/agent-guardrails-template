#!/usr/bin/env python3
"""check_go_modules.py — Spec 19 R19-10 Go module discovery gate.

Spec 19 R19-10: *"The package gate MUST enumerate modules from the repository
and execute each Go module, rather than assuming a ``go.work`` workspace."*

The gate is only as wide as its module list. `.guardrails/test-floor.json` names
the modules CI tests, but nothing tied that list to the tree: a newly added
``go.mod`` (a new service, a new example, a nested tool) silently escaped every
Go test, build and floor check, because the workflow steps and the inventory both
hardcoded the same three paths. There is no ``go.work`` in this repository, so a
"workspace" assumption is not available either.

This gate closes that gap fail-closed:

  1. **Discovery.** Every ``go.mod`` in the repository (excluding ``.git``,
     ``node_modules`` and ``vendor`` trees) is discovered from the tree.
  2. **Coverage.** Each discovered module MUST have an inventory entry in
     ``.guardrails/test-floor.json``. An undocumented module — one CI would not
     test, build or floor-check — is a FAIL.
  3. **No stale rows.** Each inventory entry MUST name a module that still
     exists. A row for a deleted module is a FAIL (the inventory may not drift
     away from the tree in either direction).
  4. **Declared command + positive floor.** Each row MUST declare a non-empty
     ``test_command``, a non-empty ``build_command``, a positive integer
     ``min_tests`` and a non-empty ``applicability`` reason (R19-12's "declare
     each required package, test command, positive minimum count and
     applicability condition").
  5. **Non-vacuity.** A discovery that finds zero modules FAILS; a green result
     may never come from an empty scan scope.

Fail-closed: a missing or unreadable inventory FAILS.

Exit: 0 clean; 1 a coverage/format/vacuity violation; 2 usage.
"""
from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

EXIT_OK = 0
EXIT_VIOLATIONS = 1
EXIT_USAGE = 2

INVENTORY = Path(".guardrails/test-floor.json")

# Directory names never descended into: VCS metadata, JS dependencies and
# vendored Go source (a vendored module is not a module of this repository).
SKIP_DIRS = {".git", "node_modules", "vendor", "testdata"}

REQUIRED_STRING_KEYS = ("module", "path", "test_command", "build_command", "applicability")


def discover_modules(repo: Path) -> tuple[set[str], list[str]]:
    """Return (relative module dirs containing go.mod, fail-closed errors)."""
    found: set[str] = set()
    errors: list[str] = []
    if not repo.is_dir():
        return found, [f"{repo}: not a directory (fail-closed)"]

    for path in sorted(repo.rglob("go.mod")):
        rel = path.relative_to(repo)
        if any(part in SKIP_DIRS for part in rel.parts[:-1]):
            continue
        module_dir = rel.parent.as_posix()
        found.add("." if module_dir == "." else module_dir)
    return found, errors


def load_inventory(path: Path) -> tuple[dict[str, dict], list[str]]:
    """Return (path -> entry, fail-closed errors)."""
    rows: dict[str, dict] = {}
    errors: list[str] = []
    if not path.is_file():
        return rows, [f"{path}: inventory not found (fail-closed)"]
    try:
        doc = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        return rows, [f"{path}: cannot read/parse ({exc})"]

    modules = doc.get("modules")
    if not isinstance(modules, list) or not modules:
        return rows, [f"{path}: 'modules' must be a non-empty list (fail-closed)"]

    for idx, entry in enumerate(modules):
        if not isinstance(entry, dict):
            errors.append(f"{path}: modules[{idx}] is not an object")
            continue
        missing = [k for k in REQUIRED_STRING_KEYS if not str(entry.get(k, "")).strip()]
        if missing:
            errors.append(
                f"{path}: modules[{idx}] ({entry.get('module', '?')}) is missing "
                f"required field(s): {', '.join(missing)}"
            )
        floor = entry.get("min_tests")
        if not isinstance(floor, int) or isinstance(floor, bool) or floor <= 0:
            errors.append(
                f"{path}: modules[{idx}] ({entry.get('module', '?')}) must declare "
                f"a positive integer min_tests (got {floor!r})"
            )
        raw_path = entry.get("path")
        if not isinstance(raw_path, str) or not raw_path.strip():
            continue
        norm = raw_path.strip().rstrip("/")
        if norm in rows:
            errors.append(f"{path}: duplicate row for module path {norm}")
        rows[norm] = entry
    return rows, errors


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", default=".", help="repository root")
    args = parser.parse_args(argv)

    repo = Path(args.repo).resolve()
    discovered, errors = discover_modules(repo)
    rows, inv_errors = load_inventory(repo / INVENTORY)
    errors.extend(inv_errors)

    if not discovered:
        errors.append(
            "non-vacuity: no go.mod discovered in the repository "
            "(the scan scope is empty)"
        )

    for module_dir in sorted(discovered):
        if module_dir not in rows:
            errors.append(
                f"{INVENTORY}: module {module_dir} has no inventory row "
                f"(a discovered Go module would escape test/build/floor CI)"
            )

    for module_dir in sorted(rows):
        if module_dir not in discovered:
            errors.append(
                f"{INVENTORY}: row for module {module_dir} has no go.mod in the "
                f"repository (stale inventory row)"
            )

    if errors:
        print("GO MODULE DISCOVERY GATE: FAIL")
        for err in errors:
            print(f"  - {err}")
        return EXIT_VIOLATIONS

    print(
        f"GO MODULE DISCOVERY GATE: OK "
        f"({len(discovered)} discovered modules, all covered by {INVENTORY})"
    )
    return EXIT_OK


if __name__ == "__main__":
    sys.exit(main())
