#!/usr/bin/env python3
"""check_forbidden_files.py — refused-artifact paths for a change set.

This replaces the inline bash loop that lived in
`.github/workflows/guardrails-lint.yml` (job `check-forbidden-files`). That loop
built its list with `grep -E "$pattern"` over a newline-separated path list and
passed **glob** patterns to a **regular-expression** engine:

    FORBIDDEN_PATTERNS=(".env" "*.pem" "*.key" "credentials.json" "secrets.json")
    MATCHES=$(echo "$CHANGED_FILES" | grep -E "$pattern" || true)

Measured against a real change list, that has two defects and they point in
opposite directions:

  * `.env` is an ERE meaning "any character followed by `env`", so it matched
    `docs/.env.example`, `.envrc`, and `src/config.env.md`. `.env.example` is the
    exact file the same job's own failure message tells contributors to use
    ("use .env.example instead"), so the check refused the remediation it
    recommended.
  * `*.pem` and `*.key` begin with `*`, which GNU grep warns about
    (`warning: * at start of expression`) and handles as undefined behaviour
    rather than as a glob. A gate whose matcher is undefined is not a gate.

The decision this encodes is the one the job claims: a file is refused when its
**basename** is a secret-bearing artifact. That is exact, warning-free, and
testable. `.env.example` / `.env.sample` are explicitly permitted because they
carry no values and are the documented alternative.

Exit: 0 no forbidden path; 1 at least one forbidden path; 2 usage.
"""
from __future__ import annotations

import argparse
import sys
from pathlib import Path
from typing import Iterable

EXIT_OK = 0
EXIT_VIOLATIONS = 1
EXIT_USAGE = 2

# Basenames that are refused outright.
FORBIDDEN_BASENAMES = ("credentials.json", "secrets.json", ".envrc")

# Suffixes that are refused outright.
FORBIDDEN_SUFFIXES = (".pem", ".key", ".p12", ".pfx", ".jks", ".keystore")

# `.env`-family files are refused, except the value-free templates that are the
# documented contributor alternative. Listed positively so a new `.env.*` variant
# is refused by default rather than silently permitted.
ENV_BASENAME = ".env"
ENV_ALLOWED = (".env.example", ".env.sample")


def is_forbidden(path: str) -> bool:
    """True when this repository-relative path names a refused artifact."""
    candidate = path.strip()
    if not candidate or candidate.startswith("#"):
        return False
    base = Path(candidate).name

    if base in ENV_ALLOWED:
        return False
    if base == ENV_BASENAME or base.startswith(ENV_BASENAME + "."):
        return True
    if base in FORBIDDEN_BASENAMES:
        return True
    return base.endswith(FORBIDDEN_SUFFIXES)


def find_violations(paths: Iterable[str]) -> list[str]:
    return [p for p in (raw.strip() for raw in paths) if p and is_forbidden(p)]


def main(argv=None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--paths-file",
        help="file holding one changed path per line (default: stdin)",
    )
    args = parser.parse_args(argv)

    if args.paths_file:
        try:
            lines = Path(args.paths_file).read_text(encoding="utf-8").splitlines()
        except OSError as exc:
            print(f"check_forbidden_files: cannot read {args.paths_file}: {exc}", file=sys.stderr)
            return EXIT_USAGE
    else:
        lines = sys.stdin.read().splitlines()

    violations = find_violations(lines)
    if violations:
        print("FORBIDDEN FILES DETECTED:")
        for v in violations:
            print(f"  - {v}")
        print()
        print("These files must not be committed:")
        print("  - .env files with values (use .env.example / .env.sample instead)")
        print("  - Private keys and keystores (*.pem, *.key, *.p12, *.pfx, *.jks, *.keystore)")
        print("  - Credential files (credentials.json, secrets.json, .envrc)")
        return EXIT_VIOLATIONS

    print(f"No forbidden files in changes ({len([l for l in lines if l.strip()])} paths checked).")
    return EXIT_OK


if __name__ == "__main__":
    sys.exit(main())
