#!/usr/bin/env python3
"""check_secret_inventory.py — Spec 17 R17-10 secret-inventory coverage gate.

Spec 17 R17-10: *"Maintain a secret inventory (owner, consumer, source, expiry,
last rotation, revocation procedure) without values."*

An inventory that can silently fall out of date is not an inventory. This gate
makes ``docs/security/SECRET-INVENTORY.md`` fail closed against the tree:

  1. **Coverage.** Every secret-bearing environment name discovered in the
     Go tree — an ``env:"NAME"`` struct tag or an ``os.Getenv("NAME")`` /
     ``getEnv("NAME", ...)`` read whose name matches the reviewed secret
     pattern — MUST have a row in the inventory table. A newly added secret
     that is not documented fails the gate.
  2. **Required columns.** Each inventory row MUST carry the five R17-10 fields
     (owner, consumer, source, expiry, last rotation, revocation) — a row with
     an empty cell fails.
  3. **No values.** A cell containing an assignment of the form ``NAME=<value>``
     (a plausible pasted secret) fails; the inventory must hold policy, not
     credential material.

Fail-closed: a missing/unreadable inventory or a scan that discovers zero
secret names FAILS (no green from an empty scope). Non-vacuity is asserted.

Exit: 0 clean; 1 a coverage/format/vacuity violation; 2 usage.
"""
from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

EXIT_OK = 0
EXIT_VIOLATIONS = 1
EXIT_USAGE = 2

INVENTORY = Path("docs/security/SECRET-INVENTORY.md")
GO_ROOT = Path("mcp-server")

# Reviewed secret-name pattern: a name is secret-bearing when it ends with or
# contains one of these tokens (mirrors scripts/check_secret_argv.py).
SECRET_TOKEN_RE = re.compile(
    r"(?:^|_)(?:PASSWORD|PASSWD|SECRET|TOKEN|APIKEY|API_KEY|PRIVATE_KEY|"
    r"VERIFIER|ENCRYPTION_KEY|DSN)(?:_|$)"
)
# Names that look secret-bearing by token but are deliberately public
# (non-secret) and therefore need no inventory row. Each entry carries a reason.
NON_SECRET_ALLOWLIST: dict[str, str] = {
    "TEST_API_KEY": (
        "test-only fixture name referenced in test code; not a deployment secret"
    ),
}

ENV_TAG_RE = re.compile(r'env:"([A-Z][A-Z0-9_]+)"')
GETENV_RE = re.compile(r'(?:os\.Getenv|getEnv)\("([A-Z][A-Z0-9_]+)"')
# A pasted assignment in an inventory cell, e.g. ``JWT_SECRET=abc123``. Only a
# *secret-shaped* LHS is a violation: a documented non-secret flag such as
# ``DB_SSLMODE=verify-full`` is policy, not credential material.
ASSIGN_RE = re.compile(r"\b([A-Z][A-Z0-9_]{3,})\s*=\s*\S")

# A ``<NAME>_FILE`` source is documented by its base ``<NAME>`` row: the file
# variant is an alternative source for the same secret, not a separate secret.
FILE_SUFFIX = "_FILE"

# The five R17-10 fields every row must carry, in table-column order.
REQUIRED_COLUMNS = ["Owner", "Consumer", "Source", "Expiry", "Last rotation", "Revocation"]


def is_secret_name(name: str) -> bool:
    return bool(SECRET_TOKEN_RE.search(name)) and name not in NON_SECRET_ALLOWLIST


def documented_name(name: str) -> str:
    """Map a discovered env name to the inventory row that documents it.

    ``JWT_SECRET_FILE`` -> ``JWT_SECRET`` (the file variant is an alternative
    source for the same secret, documented once under the base name).
    """
    if name.endswith(FILE_SUFFIX):
        return name[: -len(FILE_SUFFIX)]
    return name


def discover_secret_names(repo: Path) -> tuple[set[str], list[str]]:
    """Return (secret env names found in the tree, fail-closed errors)."""
    names: set[str] = set()
    errors: list[str] = []
    root = repo / GO_ROOT
    if not root.is_dir():
        return names, [f"{GO_ROOT}: not found (fail-closed)"]
    for path in sorted(root.rglob("*.go")):
        try:
            text = path.read_text(encoding="utf-8")
        except OSError as exc:  # pragma: no cover - unreadable file
            errors.append(f"{path}: cannot read ({exc})")
            continue
        for m in ENV_TAG_RE.finditer(text):
            if is_secret_name(m.group(1)):
                names.add(m.group(1))
        for m in GETENV_RE.finditer(text):
            if is_secret_name(m.group(1)):
                names.add(m.group(1))
    return names, errors


def parse_inventory(path: Path) -> tuple[dict[str, list[str]], list[str], list[str]]:
    """Return (rows, header, errors).

    rows: {secret-name -> [cells]} from the first markdown table whose header
    carries the R17-10 columns.
    """
    rows: dict[str, list[str]] = {}
    errors: list[str] = []
    if not path.is_file():
        return rows, [], [f"{path}: inventory not found (fail-closed)"]
    try:
        text = path.read_text(encoding="utf-8")
    except OSError as exc:  # pragma: no cover
        return rows, [], [f"{path}: cannot read ({exc})"]

    header: list[str] = []
    for line in text.splitlines():
        stripped = line.strip()
        if not stripped.startswith("|"):
            continue
        cells = [c.strip() for c in stripped.strip("|").split("|")]
        if not header:
            if all(col in cells for col in REQUIRED_COLUMNS):
                header = cells
            continue
        if set(cells) and all(set(c) <= {"-", ":", " "} for c in cells):
            continue  # separator row
        if not cells or not cells[0]:
            continue
        name = cells[0].strip("`").strip()
        if name in rows:
            errors.append(f"{path}: duplicate row for {name}")
        rows[name] = cells
    if not header:
        errors.append(
            f"{path}: no table with the R17-10 columns {REQUIRED_COLUMNS}"
        )
    return rows, header, errors


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", default=".", help="repository root")
    args = parser.parse_args(argv)

    repo = Path(args.repo).resolve()
    found, errors = discover_secret_names(repo)
    rows, header, inv_errors = parse_inventory(repo / INVENTORY)
    errors.extend(inv_errors)

    if not found:
        errors.append(
            "non-vacuity: no secret-bearing environment names discovered "
            "(the scan scope is empty)"
        )

    missing = sorted(n for n in found if documented_name(n) not in rows)
    for name in missing:
        errors.append(f"{INVENTORY}: no inventory row for secret {name}")

    if header:
        for name, cells in sorted(rows.items()):
            for idx, col in enumerate(REQUIRED_COLUMNS):
                pos = header.index(col)
                if pos >= len(cells) or not cells[pos]:
                    errors.append(f"{INVENTORY}: {name} row is missing '{col}'")
            for cell in cells:
                for m in ASSIGN_RE.finditer(cell):
                    if is_secret_name(m.group(1)):
                        errors.append(
                            f"{INVENTORY}: {name} row contains a secret value "
                            f"assignment (inventory must hold policy, not "
                            f"secrets): {cell[:40]!r}"
                        )

    if errors:
        print("SECRET INVENTORY GATE: FAIL")
        for err in errors:
            print(f"  - {err}")
        return EXIT_VIOLATIONS

    print(
        f"SECRET INVENTORY GATE: OK "
        f"({len(found)} secret names covered, {len(rows)} rows)"
    )
    return EXIT_OK


if __name__ == "__main__":
    sys.exit(main())
