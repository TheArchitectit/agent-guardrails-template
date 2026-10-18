#!/usr/bin/env python3
"""check_doc_length.py — Spec 19 R19-11 Markdown 500-line cap gate.

Spec 19 R19-11: *"The documentation gate MUST run on PR and protected
push/release events. It MUST check internal Markdown links, including links in
the new spec, and enforce a hard maximum of 500 lines per Markdown file."*

Before this gate the cap existed only as an unenforced sentence in
`docs/standards/modular-documentation.md` and as an inline shell loop in
`documentation-check.yml` that ran **only on pull requests, path-filtered to
`docs/**` and `*.md`**. Every push to `main` therefore skipped the check
entirely, and the tree carried 15 files over the cap — including files the cap
had never once been measured against.

Failure modes this gate exists to make impossible:

  1. **Vacuous pass.** A checker that finds no files, or counts nothing, must
     FAIL. "I matched nothing" is not "everything is correct".
  2. **Silent drift.** A new over-limit document must fail on the commit that
     adds it, not on a later audit.
  3. **Stale exemptions.** An exemption naming a file that is gone, or that is
     no longer over the cap, is itself a violation — otherwise the exemption
     list rots into a permanent blanket.
  4. **Unreasoned exemptions.** Every exemption MUST carry a scope class and a
     written reason. An unexplained exemption is a violation.

Exemption scope is deliberately narrow: **historical records only** (archive
trees, release notes, the append-only changelog, dated audit/plan snapshots).
Those are immutable evidence — splitting them would rewrite the record. The
release tree (docs outside `archive/`, `specs`, `mcp-server`, `ide`, top level)
is enforced with no exemption.

Exit: 0 clean; 1 one or more violations; 2 usage error.
"""
from __future__ import annotations

import argparse
import json
import subprocess
import sys
from pathlib import Path

EXIT_OK = 0
EXIT_VIOLATIONS = 1
EXIT_USAGE = 2

MAX_LINES = 500
MANIFEST = Path(".guardrails/doc-length-exemptions.json")

# An exemption must declare one of these scope classes. The class is what makes
# the exemption auditable: it states *why this kind of file* is out of scope,
# not merely that somebody was allowed to skip it.
ALLOWED_SCOPES = {
    "immutable-record",   # append-only or historical evidence; splitting rewrites the record
    "dated-snapshot",     # a point-in-time audit/review whose length is part of its identity
}


def tracked_markdown(repo: Path) -> list[Path]:
    """Tracked Markdown files, relative to repo. Falls back to a tree walk."""
    try:
        out = subprocess.run(
            ["git", "-C", str(repo), "ls-files", "-z", "--", "*.md"],
            check=True, capture_output=True,
        ).stdout
        names = [n for n in out.decode().split("\0") if n]
        return sorted(repo / n for n in names)
    except (subprocess.CalledProcessError, FileNotFoundError):
        return sorted(
            p for p in repo.rglob("*.md")
            if ".git" not in p.parts and "node_modules" not in p.parts
        )


def line_count(path: Path) -> int:
    with path.open("rb") as fh:
        return sum(1 for _ in fh)


def load_manifest(repo: Path) -> tuple[dict[str, dict], list[str]]:
    """Return (exemptions keyed by repo-relative posix path, errors)."""
    path = repo / MANIFEST
    if not path.is_file():
        # Fail-closed: an absent manifest means every file is enforced, which is
        # a valid (strictest) posture, not an error. Nothing may be exempted by
        # accident, and nothing is skipped silently either.
        return {}, []

    try:
        raw = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        return {}, [f"{MANIFEST}: unreadable or invalid JSON: {exc}"]

    entries = raw.get("exemptions")
    if not isinstance(entries, list):
        return {}, [f"{MANIFEST}: missing 'exemptions' list"]

    exemptions: dict[str, dict] = {}
    errors: list[str] = []
    for i, entry in enumerate(entries):
        if not isinstance(entry, dict):
            errors.append(f"{MANIFEST}: exemptions[{i}] is not an object")
            continue
        rel = entry.get("path")
        if not isinstance(rel, str) or not rel.strip():
            errors.append(f"{MANIFEST}: exemptions[{i}] has no 'path'")
            continue
        rel = rel.strip().lstrip("./")
        if rel in exemptions:
            errors.append(f"{MANIFEST}: duplicate exemption for {rel}")
            continue
        scope = entry.get("scope")
        if scope not in ALLOWED_SCOPES:
            errors.append(
                f"{MANIFEST}: {rel}: scope must be one of "
                f"{sorted(ALLOWED_SCOPES)}, got {scope!r}"
            )
            continue
        reason = entry.get("reason")
        if not isinstance(reason, str) or len(reason.strip()) < 20:
            errors.append(f"{MANIFEST}: {rel}: needs a written reason (>=20 chars)")
            continue
        exemptions[rel] = entry
    return exemptions, errors


def check(repo: Path, max_lines: int = MAX_LINES) -> tuple[list[str], list[str], dict]:
    """Return (violations, notes, stats)."""
    violations: list[str] = []
    notes: list[str] = []

    files = tracked_markdown(repo)
    if not files:
        return (["no Markdown files found — a gate that scans nothing cannot pass"], notes,
                {"files": 0, "over": 0, "exempt": 0})

    exemptions, manifest_errors = load_manifest(repo)
    violations.extend(manifest_errors)

    over: list[tuple[str, int]] = []
    for f in files:
        try:
            n = line_count(f)
        except OSError as exc:
            violations.append(f"{f}: unreadable ({exc})")
            continue
        if n > max_lines:
            over.append((f.relative_to(repo).as_posix(), n))

    over_paths = {p for p, _ in over}

    # Rule 1: every over-limit file must be a declared exemption.
    for rel, n in over:
        if rel not in exemptions:
            violations.append(
                f"{rel}: {n} lines exceeds the {max_lines}-line cap "
                f"(split it, or declare an exemption in {MANIFEST} with a scope and reason)"
            )

    # Rule 2: no stale exemption (deleted file, or file now within the cap).
    existing = {f.relative_to(repo).as_posix() for f in files}
    for rel in exemptions:
        if rel not in existing:
            violations.append(f"{MANIFEST}: {rel} is exempt but is not a tracked Markdown file")
        elif rel not in over_paths:
            violations.append(
                f"{MANIFEST}: {rel} is exempt but is within the {max_lines}-line cap "
                f"— remove the stale exemption"
            )

    stats = {
        "files": len(files),
        "over": len(over),
        "exempt": len(exemptions),
        "largest": max((n for _, n in over), default=0),
    }
    for rel, n in sorted(over, key=lambda t: -t[1]):
        notes.append(f"  exempt  {n:>5}  {rel}  [{exemptions[rel]['scope']}]"
                     if rel in exemptions else f"  OVER    {n:>5}  {rel}")
    return violations, notes, stats


def main(argv: list[str] | None = None) -> int:
    ap = argparse.ArgumentParser(description="Spec 19 R19-11 Markdown line-cap gate")
    ap.add_argument("path", nargs="?", default=".", help="repository root (default: .)")
    ap.add_argument("--max-lines", type=int, default=MAX_LINES)
    args = ap.parse_args(argv)

    repo = Path(args.path).resolve()
    if not repo.is_dir():
        print(f"DOC LENGTH: usage error — {repo} is not a directory")
        return EXIT_USAGE

    violations, notes, stats = check(repo, args.max_lines)

    if stats["files"] == 0:
        print("DOC LENGTH: FAIL — scanned no Markdown files (non-vacuous scan required)")
        return EXIT_VIOLATIONS

    if violations:
        print("==========================================")
        print("DOC LENGTH VIOLATIONS:")
        for v in violations:
            print(f"  - {v}")
        print("==========================================")
        print(f"DOC LENGTH: FAIL — {len(violations)} violation(s); "
              f"{stats['files']} files scanned, {stats['over']} over {args.max_lines} lines, "
              f"{stats['exempt']} declared exemptions")
        return EXIT_VIOLATIONS

    for n in notes:
        print(n)
    print(f"DOC LENGTH: OK — {stats['files']} Markdown files; "
          f"{stats['over']} over {args.max_lines} lines and all declared with a scope + reason; "
          f"{stats['files'] - stats['over']} within the cap")
    return EXIT_OK


if __name__ == "__main__":
    sys.exit(main())
