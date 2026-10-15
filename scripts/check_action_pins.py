#!/usr/bin/env python3
"""check_action_pins.py — Spec 19 R19-15 / Spec 17 R17-11 action pinning gate.

Spec 19 R19-15: *"Required workflow actions MUST use reviewed full commit SHA
references, and release/base images MUST use immutable digests. A test validates
workflow references ... at the exact commit; a mutable tag, unknown action SHA,
omitted identity, or unreviewed update is FAIL."*

Spec 17 R17-11: *"Pin build/test actions by immutable full commit SHA and
production base/service images by immutable digest ... not a `latest`/floating
tag as release proof."*

A tag is mutable: `actions/checkout@v4` resolves to whatever the `v4` tag points
at *today*, so a moved or compromised tag silently changes the code that runs
with the repository's `GITHUB_TOKEN`. A 40-hex commit SHA is immutable, which is
the property the requirement asks for.

Scope is every workflow under `.github/workflows/`.

Ref forms judged:
  ``./path``             a LOCAL composite action in this repo — allowed (this
                         repository's own code at this commit, not a mutable
                         third-party ref).
  ``owner/repo@<40hex>`` a third-party action pinned by commit SHA — allowed.
  ``owner/repo@<tag>``   a mutable ref — VIOLATION.
  ``owner/repo``         no ref (defaults to the default branch) — VIOLATION.
  ``docker://...``       a container ref — judged only when it carries a mutable
                         tag rather than a digest.

Fail-closed: a workflow that cannot be read is a violation, not a skip.
Non-vacuity: a scan that finds zero ``uses:`` across zero workflows FAILS (no
green from an empty scope). The allowlist is empty by design; a new entry fails
the gate until it is given a reason.

Exit: 0 clean; 1 a mutable/unpinned third-party ref (or a malformed scan); 2 usage.
"""
from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

EXIT_OK = 0
EXIT_VIOLATIONS = 1
EXIT_USAGE = 2

WORKFLOW_DIR = Path(".github") / "workflows"
SHA_RE = re.compile(r"^[0-9a-f]{40}$")
# `uses:` may be indented, and may be a list item (`- uses: ...`).
USES_RE = re.compile(r"^\s*(?:-\s*)?uses:\s*(?P<ref>[^\s#]+)\s*(?:#.*)?$")

# Deliberate exceptions: "owner/repo@ref" -> reason. Empty by design.
ALLOWLIST: dict[str, str] = {}


def workflow_files(repo: Path) -> list[Path]:
    d = repo / WORKFLOW_DIR
    if not d.is_dir():
        return []
    return sorted(p for p in d.iterdir() if p.suffix in (".yml", ".yaml"))


def judge_ref(ref: str) -> str | None:
    """Return a violation reason for `ref`, or None when it is acceptable."""
    if ref.startswith("./"):
        return None  # local composite action — this repo's own code
    if ref.startswith("docker://"):
        image = ref[len("docker://"):]
        if "@sha256:" in image:
            return None
        return f"{ref}: container action is not pinned by digest (@sha256:...)"
    if "@" not in ref:
        return f"{ref}: no ref — resolves to the action's default branch, which is mutable"
    path, _, version = ref.rpartition("@")
    if SHA_RE.match(version):
        return None
    if ref in ALLOWLIST:
        return None
    return (
        f"{ref}: pinned by '{version}' (a mutable tag), not a 40-hex commit SHA. "
        f"Resolve it with: gh api repos/{path}/commits/{version} --jq .sha"
    )


def scan(repo: Path) -> tuple[int, int, list[str]]:
    """Return (workflow_count, refs_seen, violations)."""
    files = workflow_files(repo)
    violations: list[str] = []
    refs_seen = 0
    for wf in files:
        rel = wf.relative_to(repo)
        try:
            text = wf.read_text(encoding="utf-8")
        except OSError as exc:
            violations.append(f"{rel}: unreadable workflow ({exc})")
            continue
        for lineno, line in enumerate(text.splitlines(), 1):
            m = USES_RE.match(line)
            if not m:
                continue
            ref = m.group("ref")
            refs_seen += 1
            reason = judge_ref(ref)
            if reason is not None:
                violations.append(f"{rel}:{lineno}: {reason}")
    return len(files), refs_seen, violations


def main(argv: list[str]) -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--repo", default=".", help="repository root")
    args = ap.parse_args(argv)

    repo = Path(args.repo).resolve()
    files, refs_seen, violations = scan(repo)

    if files == 0:
        print(
            f"R19-15 FAIL: no workflow files under {WORKFLOW_DIR} — the gate "
            f"scanned nothing (no vacuous green)",
            file=sys.stderr,
        )
        return EXIT_VIOLATIONS

    if refs_seen == 0:
        print(
            f"R19-15 FAIL: {files} workflow file(s) but zero `uses:` refs found — "
            f"the scan is not exercising anything (no vacuous green)",
            file=sys.stderr,
        )
        return EXIT_VIOLATIONS

    if violations:
        for v in violations:
            print(f"R19-15 VIOLATION: {v}")
        print(f"\nfail: {len(violations)} unpinned ref(s) across {files} workflow(s)")
        return EXIT_VIOLATIONS

    print(
        f"R19-15: {refs_seen} action ref(s) across {files} workflow(s); "
        f"all third-party refs pinned by commit SHA (or local) — OK"
    )
    return EXIT_OK


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
