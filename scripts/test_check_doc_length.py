#!/usr/bin/env python3
"""Red-on-drift self-test for scripts/check_doc_length.py (Spec 19 R19-11).

Proves the cap gate fails when it must and passes when it should, so a green
run cannot come from a gate that never exercises its assertion:

  * an over-limit file with no exemption FAILS
  * the same file with a declared exemption PASSES
  * a stale exemption (file now within the cap) FAILS
  * an exemption naming a deleted file FAILS
  * an exemption with no reason / a bogus scope FAILS
  * a duplicate exemption FAILS
  * an empty scan scope FAILS (non-vacuity)
  * a file exactly at the cap PASSES (the boundary is inclusive)

Run with: python -m pytest scripts/test_check_doc_length.py -v
Or: python scripts/test_check_doc_length.py
"""

import json
import os
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from scripts import check_doc_length as gate


def _write(root: Path, rel: str, lines: int) -> None:
    p = root / rel
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text("\n".join(f"line {i}" for i in range(lines)) + "\n", encoding="utf-8")


def _manifest(root: Path, entries: list[dict]) -> None:
    p = root / gate.MANIFEST
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(json.dumps({"exemptions": entries}, indent=2), encoding="utf-8")


def _entry(path: str, scope: str = "immutable-record",
           reason: str = "historical record preserved verbatim for the audit trail", **over) -> dict:
    e = {"path": path, "scope": scope, "reason": reason}
    e.update(over)
    return e


class DocLengthGateTest(unittest.TestCase):
    def setUp(self) -> None:
        self.tmp = tempfile.TemporaryDirectory()
        self.root = Path(self.tmp.name)

    def tearDown(self) -> None:
        self.tmp.cleanup()

    def test_over_limit_without_exemption_fails(self) -> None:
        _write(self.root, "docs/big.md", 501)
        violations, _, _ = gate.check(self.root)
        self.assertTrue(any("exceeds" in v for v in violations), violations)

    def test_at_cap_passes(self) -> None:
        _write(self.root, "docs/exact.md", 500)
        violations, _, stats = gate.check(self.root)
        self.assertEqual(violations, [])
        self.assertEqual(stats["over"], 0)

    def test_declared_exemption_passes(self) -> None:
        _write(self.root, "docs/big.md", 501)
        _manifest(self.root, [_entry("docs/big.md")])
        violations, _, stats = gate.check(self.root)
        self.assertEqual(violations, [])
        self.assertEqual(stats["exempt"], 1)

    def test_stale_exemption_fails(self) -> None:
        _write(self.root, "docs/small.md", 10)
        _manifest(self.root, [_entry("docs/small.md")])
        violations, _, _ = gate.check(self.root)
        self.assertTrue(any("stale exemption" in v for v in violations), violations)

    def test_exemption_for_missing_file_fails(self) -> None:
        _write(self.root, "docs/ok.md", 10)
        _manifest(self.root, [_entry("docs/gone.md")])
        violations, _, _ = gate.check(self.root)
        self.assertTrue(any("not a tracked Markdown file" in v for v in violations), violations)

    def test_exemption_without_reason_fails(self) -> None:
        _write(self.root, "docs/big.md", 501)
        _manifest(self.root, [_entry("docs/big.md", reason="short")])
        violations, _, _ = gate.check(self.root)
        self.assertTrue(any("written reason" in v for v in violations), violations)

    def test_exemption_with_bogus_scope_fails(self) -> None:
        _write(self.root, "docs/big.md", 501)
        _manifest(self.root, [_entry("docs/big.md", scope="because-i-said-so")])
        violations, _, _ = gate.check(self.root)
        self.assertTrue(any("scope must be one of" in v for v in violations), violations)

    def test_duplicate_exemption_fails(self) -> None:
        _write(self.root, "docs/big.md", 501)
        _manifest(self.root, [_entry("docs/big.md"), _entry("docs/big.md")])
        violations, _, _ = gate.check(self.root)
        self.assertTrue(any("duplicate exemption" in v for v in violations), violations)

    def test_empty_scan_fails_non_vacuously(self) -> None:
        violations, _, stats = gate.check(self.root)
        self.assertEqual(stats["files"], 0)
        self.assertTrue(any("scans nothing" in v for v in violations), violations)

    def test_unreadable_manifest_fails(self) -> None:
        _write(self.root, "docs/ok.md", 10)
        p = self.root / gate.MANIFEST
        p.parent.mkdir(parents=True, exist_ok=True)
        p.write_text("{not json", encoding="utf-8")
        violations, _, _ = gate.check(self.root)
        self.assertTrue(any("invalid JSON" in v for v in violations), violations)

    def test_missing_manifest_is_strict_not_permissive(self) -> None:
        """No manifest => every file enforced (strict), never a silent skip."""
        _write(self.root, "docs/big.md", 501)
        violations, _, _ = gate.check(self.root)
        self.assertTrue(any("exceeds" in v for v in violations), violations)


if __name__ == "__main__":
    unittest.main(verbosity=2)
