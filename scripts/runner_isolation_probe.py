#!/usr/bin/env python3
"""runner_isolation_probe.py — Spec 19 R19-13 clause 5, executed ON the runner.

    "A runner label alone is not isolation evidence. Release acceptance MUST
    include an isolation probe or recorded host attestation."

The static gate (`scripts/check_runner_isolation.py`) reads YAML and can prove
that the *definition* is repo-scoped: hosted-only PR jobs, one self-hosted job,
least-privilege token, bounded concurrency, cleanup last. It cannot prove
anything about the *host*, because YAML is not the host. This probe runs inside
the trusted job, measures the things the requirement names, and writes a record
that the gate then validates with `--attestation`.

Measured, not asserted:

- **clean checkout** — the workspace holds exactly one git top-level, and its
  HEAD is the commit this run was dispatched for. A workspace that also holds a
  previous run's tree, or a tree for a different SHA, is residue.
- **repo-scoped workspace** — the parent of the checkout directory contains no
  other repository working tree. On a persistent host, `<runner>/_work/<repo>/
  <repo>` shares `<runner>/_work` with every other repository the runner has
  hosted; a sibling there is cross-repository reuse reachable by a plain `..`.
- **no credential material in the workspace** — no `.env`-shaped file holding a
  non-empty assignment at the top of the tree. This is a heuristic and it is
  reported as such: it catches the copy-pasted `.env`, not every leak.
- **bounded workspace permissions** — the checkout root is not group- or
  world-writable, because a shared writable directory on a persistent host is
  how one run edits the next one's tree.

Exit: 0 attestation written and clean; 1 attestation written but a property was
violated (the job still gets the record to inspect); 2 the probe could not run
(no git, no workspace) — which is a failure, not a pass.

Write to a path outside the checkout by default (`--out`), so the attestation
does not itself become workspace residue.
"""
from __future__ import annotations

import argparse
import json
import os
import shutil
import stat
import subprocess
import sys
from pathlib import Path

EXIT_OK = 0
EXIT_VIOLATIONS = 1
EXIT_UNUSABLE = 2

SCHEMA = "runner-isolation/v1"
# A `.env`-shaped file at the tree root with a real assignment. Bounded depth on
# purpose: a full-tree scan would walk node_modules and vendor trees for no gain.
ENV_CANDIDATES = (".env", ".env.local", ".env.production")
ENV_SKIP_SUFFIXES = (".example", ".sample", ".template", ".dist")


def _git(workspace: Path, *args: str) -> str | None:
    git = shutil.which("git")
    if git is None:
        return None
    try:
        out = subprocess.run(
            [git, *args],
            cwd=workspace,
            capture_output=True,
            text=True,
            timeout=30,
            check=False,
        )
    except (OSError, subprocess.SubprocessError):
        return None
    if out.returncode != 0:
        return None
    return out.stdout.strip()


def _looks_like_git_tree(path: Path) -> bool | None:
    """True/False, or None when the entry cannot be inspected.

    A permission error on one unrelated entry in the parent directory is not
    evidence of cross-repository reuse, and letting it abort the whole scan
    makes the probe report a violation the host never committed. It is
    recorded as unmeasured instead (see `unreadable_entries`), because a probe
    that guesses is the failure this requirement is about.
    """
    try:
        return (path / ".git").exists() or (path / "HEAD").is_file()
    except OSError:
        return None


def probe(workspace: Path, expected_sha: str | None, expected_repo: str | None) -> dict:
    violations: list[str] = []
    record: dict = {
        "schema": SCHEMA,
        "repo": expected_repo or "",
        "runner_name": os.environ.get("RUNNER_NAME", ""),
        "workspace": str(workspace),
        "expected_sha": expected_sha or "",
        "actual_sha": "",
        "clean_checkout": False,
        "sibling_repositories": [],
        "cross_repository_checkouts": [],
        "unreadable_entries": [],
        "workspace_mode": "",
        "env_files": [],
        "violations": violations,
        "limits": [
            "the credential scan is a root-level .env heuristic, not a full "
            "secret sweep (secret-validation.yml owns that)",
            "the probe measures the workspace, not the host kernel, network "
            "namespace or container runtime",
        ],
    }

    if not workspace.is_dir():
        violations.append(f"workspace {workspace} does not exist")
        return record

    actual = _git(workspace, "rev-parse", "HEAD")
    record["actual_sha"] = actual or ""
    if actual is None:
        violations.append("workspace is not a readable git checkout")
    else:
        toplevel = _git(workspace, "rev-parse", "--show-toplevel")
        if toplevel and Path(toplevel).resolve() != workspace.resolve():
            violations.append(
                f"workspace {workspace} is nested inside the git tree at "
                f"{toplevel}, so the checkout is not the workspace root"
            )
        if expected_sha and actual != expected_sha:
            violations.append(
                f"HEAD is {actual[:12]} but this run is for {expected_sha[:12]} "
                f"— the workspace carries another commit's tree"
            )
        else:
            record["clean_checkout"] = True

    # Siblings: another repository tree beside this one under the same parent.
    # The parent itself being unreadable is a genuine violation (the scope cannot
    # be established); an individual unreadable entry is recorded, not assumed.
    siblings: list[str] = []
    unreadable: list[str] = []
    try:
        entries = sorted(workspace.parent.iterdir())
    except OSError as exc:
        entries = []
        violations.append(
            f"cannot list the workspace parent {workspace.parent} ({exc}) — the "
            f"repo scope of the workspace cannot be established"
        )
    for entry in entries:
        try:
            if not entry.is_dir() or entry.resolve() == workspace.resolve():
                continue
        except OSError:
            unreadable.append(entry.name)
            continue
        looks = _looks_like_git_tree(entry)
        if looks is None:
            unreadable.append(entry.name)
        elif looks:
            siblings.append(entry.name)
    record["sibling_repositories"] = sorted(siblings)
    record["unreadable_entries"] = sorted(unreadable)
    if siblings:
        violations.append(
            f"sibling repository trees in {workspace.parent}: {siblings} — the "
            f"workspace root is shared across repositories"
        )

    # Cross-repository reuse in the Actions layout. A self-hosted runner lays
    # checkouts out as `<runner>/_work/<repo>/<repo>`, so the parent of the
    # workspace holds only this repository's own checkout and every OTHER
    # repository sits one level higher. Scanning only the parent would report a
    # clean scope on a runner that hosts the whole fleet — which is exactly the
    # cross-repository reuse R19-13 forbids. The grandparent is examined only in
    # that recognised shape, so an unrelated `/tmp` full of scratch clones does
    # not masquerade as a runner workspace.
    cross_repo: list[str] = []
    if workspace.parent.name == workspace.name:
        work_root = workspace.parent.parent
        try:
            for entry in sorted(work_root.iterdir()):
                if entry.name == workspace.name:
                    continue
                try:
                    if not entry.is_dir():
                        continue
                    nested = [c for c in entry.iterdir() if c.is_dir() and _looks_like_git_tree(c)]
                except OSError:
                    unreadable.append(f"{work_root.name}/{entry.name}")
                    continue
                if nested:
                    cross_repo.append(entry.name)
        except OSError as exc:
            violations.append(
                f"cannot list the runner work root {work_root} ({exc}) — the "
                f"runner's repository scope cannot be established"
            )
    record["cross_repository_checkouts"] = sorted(cross_repo)
    if cross_repo:
        violations.append(
            f"the runner work root {workspace.parent.parent} holds checkouts for "
            f"other repositories ({cross_repo}) — the runner is not repo-scoped "
            f"(R19-13)"
        )

    try:
        mode = stat.S_IMODE(workspace.stat().st_mode)
        record["workspace_mode"] = oct(mode)
        if mode & (stat.S_IWGRP | stat.S_IWOTH):
            violations.append(
                f"workspace {workspace} is {oct(mode)} — group/world writable on "
                f"a persistent host lets one run edit the next one's tree"
            )
    except OSError as exc:
        violations.append(f"cannot stat the workspace ({exc})")

    for name in ENV_CANDIDATES:
        candidate = workspace / name
        if not candidate.is_file():
            continue
        try:
            text = candidate.read_text(encoding="utf-8", errors="replace")
        except OSError:
            continue
        assignments = [
            line.split("=", 1)[0].strip()
            for line in text.splitlines()
            if "=" in line
            and not line.lstrip().startswith("#")
            and line.split("=", 1)[1].strip()
        ]
        if assignments:
            record["env_files"].append({"path": str(candidate), "keys": sorted(assignments)})
            violations.append(
                f"{name} in the checkout root holds {len(assignments)} non-empty "
                f"assignment(s) ({', '.join(sorted(assignments))}) — credential "
                f"material left in a shared workspace"
            )

    return record


def main(argv: list[str]) -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument(
        "--workspace",
        default=os.environ.get("GITHUB_WORKSPACE", "."),
        help="checkout directory to probe (default: $GITHUB_WORKSPACE)",
    )
    ap.add_argument(
        "--sha",
        default=os.environ.get("GITHUB_SHA", ""),
        help="commit this run is for (default: $GITHUB_SHA)",
    )
    ap.add_argument(
        "--repo",
        default=os.environ.get("GITHUB_REPOSITORY", ""),
        help="owner/name (default: $GITHUB_REPOSITORY)",
    )
    ap.add_argument(
        "--out",
        default=None,
        help="attestation path (default: $RUNNER_TEMP/runner-isolation.json, "
        "outside the checkout)",
    )
    args = ap.parse_args(argv)

    workspace = Path(args.workspace or ".").resolve()
    out = args.out or os.path.join(
        os.environ.get("RUNNER_TEMP") or "/tmp", "runner-isolation.json"
    )

    record = probe(workspace, args.sha or None, args.repo or None)

    try:
        Path(out).parent.mkdir(parents=True, exist_ok=True)
        Path(out).write_text(json.dumps(record, indent=2) + "\n", encoding="utf-8")
    except OSError as exc:
        print(f"R19-13 probe FAIL: cannot write attestation to {out} ({exc})", file=sys.stderr)
        return EXIT_UNUSABLE

    print(json.dumps({k: v for k, v in record.items() if k != "limits"}, indent=2))

    if not record["actual_sha"]:
        print("R19-13 probe FAIL: the probe could not measure the workspace", file=sys.stderr)
        return EXIT_UNUSABLE

    if record["violations"]:
        for v in record["violations"]:
            print(f"R19-13 VIOLATION: {v}")
        print(f"\nfail: {len(record['violations'])} violation(s); attestation at {out}")
        return EXIT_VIOLATIONS

    print(f"\nR19-13 OK: host attestation written to {out}")
    return EXIT_OK


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
