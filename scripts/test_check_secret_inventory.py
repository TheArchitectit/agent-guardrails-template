#!/usr/bin/env python3
"""Red-on-drift self-test for scripts/check_secret_inventory.py (Spec 17 R17-10).

Proves the gate fails on a missing row, an empty required cell and a pasted
secret value, and passes on a complete inventory — so a green run cannot come
from a gate that never exercises its assertion. Also proves non-vacuity (an
empty scan scope fails) and the ``<NAME>_FILE`` mapping.

Run with: python -m pytest scripts/test_check_secret_inventory.py -v
Or: python scripts/test_check_secret_inventory.py
"""

import contextlib
import io
import os
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from scripts import check_secret_inventory as gate

HEADER = (
    "| Secret | Owner | Consumer | Source | Expiry | Last rotation | Revocation |\n"
    "|---|---|---|---|---|---|---|\n"
)


def _row(name: str, cells: list[str] | None = None) -> str:
    cells = cells or ["owner", "consumer", "source", "expiry", "rotation", "revocation"]
    return f"| `{name}` | " + " | ".join(cells) + " |\n"


def _write_go(root: Path, body: str) -> None:
    d = root / "mcp-server" / "internal" / "x"
    d.mkdir(parents=True, exist_ok=True)
    (d / "cfg.go").write_text(body, encoding="utf-8")


def _write_inv(root: Path, body: str) -> None:
    d = root / "docs" / "security"
    d.mkdir(parents=True, exist_ok=True)
    (d / "SECRET-INVENTORY.md").write_text(body, encoding="utf-8")


GO = 'package x\n\ntype C struct {\n\tJWTSecret string `env:"JWT_SECRET"`\n\tJWTSecretFile string `env:"JWT_SECRET_FILE"`\n}\n'


class CheckSecretInventoryTest(unittest.TestCase):
    def _run(self, root: Path) -> int:
        # The gate prints its verdict; swallow it so the test output stays clean.
        with contextlib.redirect_stdout(io.StringIO()):
            return gate.main(["--repo", str(root)])

    def test_clean_when_covered(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _write_go(root, GO)
            _write_inv(root, "# inv\n\n" + HEADER + _row("JWT_SECRET"))
            self.assertEqual(self._run(root), gate.EXIT_OK)

    def test_missing_row_fails(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _write_go(root, GO)
            _write_inv(root, "# inv\n\n" + HEADER + _row("SOMETHING_ELSE"))
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)

    def test_empty_required_cell_fails(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _write_go(root, GO)
            _write_inv(
                root,
                "# inv\n\n" + HEADER + _row("JWT_SECRET", ["owner", "", "source", "expiry", "rotation", "revocation"]),
            )
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)

    def test_pasted_value_fails(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _write_go(root, GO)
            _write_inv(
                root,
                "# inv\n\n"
                + HEADER
                + _row("JWT_SECRET", ["owner", "consumer", "JWT_SECRET=deadbeef", "expiry", "rotation", "revocation"]),
            )
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)

    def test_non_secret_assignment_is_policy_not_a_value(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _write_go(root, GO)
            _write_inv(
                root,
                "# inv\n\n"
                + HEADER
                + _row("JWT_SECRET", ["owner", "consumer", "DB_SSLMODE=verify-full required", "expiry", "rotation", "revocation"]),
            )
            self.assertEqual(self._run(root), gate.EXIT_OK)

    def test_non_vacuity_empty_scope_fails(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _write_go(root, "package x\n\nvar y = 1\n")
            _write_inv(root, "# inv\n\n" + HEADER + _row("JWT_SECRET"))
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)

    def test_file_variant_documented_by_base_row(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _write_go(root, GO)
            _write_inv(root, "# inv\n\n" + HEADER + _row("JWT_SECRET"))
            self.assertEqual(self._run(root), gate.EXIT_OK)

    def test_missing_inventory_fails(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            _write_go(root, GO)
            self.assertEqual(self._run(root), gate.EXIT_VIOLATIONS)


if __name__ == "__main__":
    unittest.main()
