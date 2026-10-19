#!/usr/bin/env python3
"""check_mandatory_rows.py — Spec 19 R19-14 path-filter safeguards.

Spec 19 R19-14, verbatim:

    A path filter MAY select a narrower fast suite only when a dependency
    manifest maps changed paths to every affected gate. Security, secret,
    registry, deployment/readiness, package inventory, and test-floor checks
    MUST run on every protected push and release candidate. A workflow edit,
    script/tool change, configuration/schema/migration change, shared handler,
    auth/route/transport change, dependency-file change, or uncertain path MUST
    select the full matrix. A filter error, missing diff, shallow checkout, or
    unclassified path is `NOT_RUN`/FAIL, not an automatic skip.

Why this gate exists in this shape
----------------------------------
The automated half of R19-14 was unenforced. Nothing in the tree read a `paths:`
filter, and nothing asserted that the six named mandatory classes were reachable
from a protected push. Two separable failures were live at `main@58148a16`, and
the second is the worse one:

  1. **No dependency manifest existed.** A `paths:` key added to any workflow
     tomorrow would silently narrow a gate and no check would notice, because
     the filter is expressed in YAML that only GitHub reads.
  2. **A "mandatory" row can be routed somewhere that never runs.** A gate step
     living in a workflow whose only trigger is `pull_request` satisfies a reader
     and never executes on a protected push at all. That is enforcement in
     appearance only — the exact failure this repository has now paid for four
     times in the F001 tool-count gate.

So the gate asserts three independent properties:

  * **Filter discipline.** No workflow may declare `paths:` or `paths-ignore:`.
    Narrowing is permitted by the requirement only behind a dependency manifest
    that maps the changed path to *every* affected gate; no such manifest can be
    written for a filter expressed in `on:`, so a filter is refused outright
    rather than assumed to be sound. A declared full-matrix escalation list is
    checked for the paths the requirement names (workflow edit, script/tool,
    config/schema/migration, shared handler, auth/route/transport, dependency
    files), and an escalation list that omits any of them is a violation.
  * **Reachability.** Every mandatory row named in `.guardrails/mandatory-ci-rows.json`
    must resolve to a step that exists in a workflow triggered by `push` on a
    protected branch. A step that exists only in a `pull_request`-triggered
    workflow is a VIOLATION, not a pass: the requirement says "on every
    protected push and release candidate".
  * **Event-correct diffing.** A job that computes a changed-file list must not
    hardcode a base ref that its own triggers contradict (a workflow that also
    runs on `develop` diffing against the main branch compares the wrong pair),
    and must not silently narrow to a single tip commit when the real diff
    fails. The fallback exists precisely because the primary command failed —
    "a filter error ... is NOT_RUN/FAIL, not an automatic skip" — so a silent
    narrowing fallback is refused and the job must fail instead.

Scope of the diff check: only **executable** `run:` blocks are inspected. A
comment that quotes the old broken command is documentation of a defect that was
fixed, not a recurrence of it; scanning prose would make the gate red on its own
changelog and teach the next maintainer to delete the explanation. The YAML
parser is what draws that line — comments never reach the parsed document.

Non-vacuity, because a green run must mean something was exercised:
  * zero workflow files                          -> FAIL
  * zero mandatory rows declared                 -> FAIL
  * a mandatory row naming a step that no workflow contains -> FAIL
  * an empty mandatory-class set                 -> FAIL

Exit: 0 clean; 1 a violation (or a vacuous scan); 2 usage.
"""
from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path

import yaml

EXIT_OK = 0
EXIT_VIOLATIONS = 1
EXIT_USAGE = 2

WORKFLOW_DIR = Path(".github") / "workflows"
MANIFEST = Path(".guardrails") / "mandatory-ci-rows.json"

# The six classes R19-14 names by hand. A manifest that declares fewer than these
# is not a narrower policy, it is an incomplete one — the requirement lists them
# with "MUST".
REQUIRED_CLASSES = (
    "security",
    "secret",
    "registry",
    "deployment-readiness",
    "package-inventory",
    "test-floor",
)

# Path shapes the requirement says MUST select the full matrix. A declared
# escalation list has to cover every one of these by name.
REQUIRED_ESCALATION_KEYS = (
    "workflow",
    "script_tool",
    "config_schema_migration",
    "shared_handler",
    "auth_route_transport",
    "dependency_file",
    "uncertain",
)

# Triggers that mean "a protected push runs this".
PUSH_EVENTS = ("push",)

# A base ref hardcoded in a diff command.
HARDCODED_BASE_RE = re.compile(r"origin/(main|master)\b")
# The narrowing fallback: scan only the tip commit if the real diff failed.
TIP_ONLY_FALLBACK_RE = re.compile(r"HEAD~1\b")


def load_workflows(root: Path):
    """Return [(path, parsed_or_None, text)] for every workflow file."""
    out = []
    directory = root / WORKFLOW_DIR
    if not directory.is_dir():
        return out
    for path in sorted(directory.glob("*.yml")) + sorted(directory.glob("*.yaml")):
        text = path.read_text(encoding="utf-8", errors="replace")
        try:
            parsed = yaml.safe_load(text)
        except Exception:
            parsed = None
        out.append((path.relative_to(root).as_posix(), parsed, text))
    return out


def triggers_of(doc):
    """Normalise an `on:` block to a dict of event -> config (or None)."""
    if not isinstance(doc, dict):
        return {}
    raw = doc.get(True, doc.get("on"))
    if raw is None:
        return {}
    if isinstance(raw, str):
        return {raw: None}
    if isinstance(raw, list):
        return {str(e): None for e in raw}
    if isinstance(raw, dict):
        return {str(k): v for k, v in raw.items()}
    return {}


def branches_of(config):
    """Branch list for a push event config, or None when unbounded."""
    if not isinstance(config, dict):
        return None
    branches = config.get("branches")
    if branches is None:
        return None
    if isinstance(branches, str):
        return [branches]
    return [str(b) for b in branches]


def has_path_filter(config) -> bool:
    if not isinstance(config, dict):
        return False
    return "paths" in config or "paths-ignore" in config


def iter_steps(doc):
    """Yield (job_id, step_name, step) across every job in a workflow."""
    if not isinstance(doc, dict):
        return
    jobs = doc.get("jobs")
    if not isinstance(jobs, dict):
        return
    for job_id, job in jobs.items():
        if not isinstance(job, dict):
            continue
        steps = job.get("steps")
        if not isinstance(steps, list):
            continue
        for step in steps:
            if not isinstance(step, dict):
                continue
            yield str(job_id), str(step.get("name", "")), step


def run_blocks(doc):
    """Yield (job_id, step_name, script) for every executable `run:` block.

    This is the scan surface for behavioural checks: YAML comments never reach
    the parsed document, so prose about a past defect cannot be mistaken for a
    live one.
    """
    for job_id, name, step in iter_steps(doc):
        script = step.get("run")
        if isinstance(script, str):
            yield job_id, name, script


def step_matches(step_name: str, step: dict, needle: str) -> bool:
    if needle.lower() in step_name.lower():
        return True
    run = step.get("run")
    return isinstance(run, str) and needle.lower() in run.lower()


def check(root: Path) -> list[str]:
    problems: list[str] = []
    workflows = load_workflows(root)
    if not workflows:
        problems.append(
            "VACUOUS: no workflow files found under .github/workflows — nothing was checked"
        )
        return problems

    # --- 1. filter discipline ------------------------------------------------
    push_reachable = {}  # path -> set of branches (or {"*"})
    for rel, doc, _text in workflows:
        if doc is None:
            problems.append(f"{rel}: not parseable as YAML — cannot be verified (FAIL, not skip)")
            continue
        trig = triggers_of(doc)
        if not trig:
            problems.append(f"{rel}: declares no `on:` triggers — dead workflow")
        for event, config in trig.items():
            if has_path_filter(config):
                problems.append(
                    f"{rel}: `{event}` declares a path filter "
                    f"({', '.join(k for k in ('paths', 'paths-ignore') if k in config)}). "
                    "R19-14 permits narrowing only behind a dependency manifest that maps "
                    "every changed path to every affected gate; a filter in `on:` cannot "
                    "express that mapping, so it is refused."
                )
            if event in PUSH_EVENTS:
                branches = branches_of(config)
                push_reachable.setdefault(rel, set())
                if branches is None:
                    push_reachable[rel].add("*")
                else:
                    push_reachable[rel].update(branches)

    protected_push = {
        rel for rel, br in push_reachable.items() if "*" in br or br & {"main", "master"}
    }
    if not protected_push:
        problems.append(
            "VACUOUS: no workflow runs on a protected push (main/master) — the "
            "mandatory rows cannot be reachable"
        )

    # --- 2. mandatory-row reachability --------------------------------------
    manifest_path = root / MANIFEST
    if not manifest_path.is_file():
        problems.append(
            f"{MANIFEST.as_posix()}: missing. R19-14 requires a dependency manifest "
            "declaring how each mandatory class is reached; without it nothing maps "
            "a gate to a guaranteed trigger."
        )
        rows = []
        escalation = {}
    else:
        try:
            data = json.loads(manifest_path.read_text(encoding="utf-8"))
        except Exception as exc:  # malformed manifest is an error, not an empty set
            problems.append(f"{MANIFEST.as_posix()}: unparseable ({exc}) — FAIL, not skip")
            data = {}
        rows = data.get("rows") or []
        escalation = data.get("full_matrix_escalation") or {}
        if not isinstance(rows, list) or not rows:
            problems.append(f"{MANIFEST.as_posix()}: declares zero rows — a vacuous manifest")

        declared_classes = {str(r.get("class", "")) for r in rows if isinstance(r, dict)}
        for cls in REQUIRED_CLASSES:
            if cls not in declared_classes:
                problems.append(
                    f"{MANIFEST.as_posix()}: mandatory class `{cls}` is not declared. "
                    "R19-14 names it with MUST; an absent row is an unenforced requirement."
                )

    for row in rows:
        if not isinstance(row, dict):
            problems.append(f"{MANIFEST.as_posix()}: a row is not an object")
            continue
        cls = str(row.get("class", "<unnamed>"))
        workflow = str(row.get("workflow", ""))
        needle = str(row.get("step", ""))
        if not workflow or not needle:
            problems.append(f"row `{cls}`: must name both `workflow` and `step`")
            continue
        match = next((w for w in workflows if w[0] == workflow), None)
        if match is None:
            problems.append(f"row `{cls}`: names workflow `{workflow}` which does not exist")
            continue
        rel, doc, _text = match
        if doc is None:
            problems.append(f"row `{cls}`: workflow `{rel}` is unparseable")
            continue
        if not any(step_matches(name, step, needle) for _job, name, step in iter_steps(doc)):
            problems.append(
                f"row `{cls}`: step `{needle}` does not exist in `{rel}` — the row "
                "names a check nobody implements"
            )
            continue
        if rel not in protected_push:
            problems.append(
                f"row `{cls}`: `{rel}` carries the step but never runs on a protected "
                "push. R19-14 requires the check on every protected push; a "
                "pull_request-only workflow satisfies a reader, not the requirement."
            )

    # --- 3. escalation coverage ---------------------------------------------
    if manifest_path.is_file():
        for key in REQUIRED_ESCALATION_KEYS:
            value = escalation.get(key) if isinstance(escalation, dict) else None
            if not value:
                problems.append(
                    f"{MANIFEST.as_posix()}: `full_matrix_escalation.{key}` is absent or "
                    "empty. R19-14 names this path class as MUST-select-full-matrix; an "
                    "escalation list that omits it cannot select it."
                )

    # --- 4. event-correct diffing -------------------------------------------
    for rel, doc, text in workflows:
        trig = triggers_of(doc) if doc is not None else {}
        # Only executable content counts. `text` is the raw file (comments and
        # all); `run_blocks` is what actually executes. A comment quoting the old
        # broken command is a record of a fixed defect, not a live one — the
        # requirement governs behaviour, and scanning prose would make the gate
        # fail on its own explanation.
        for _job, _name, script in run_blocks(doc):
            base_match = HARDCODED_BASE_RE.search(script)
            if base_match:
                base = base_match.group(1)
                other_branches = set()
                for event, config in trig.items():
                    for br in branches_of(config) or []:
                        other_branches.add(br)
                if other_branches - {base}:
                    problems.append(
                        f"{rel}: diffs against a hardcoded `origin/{base}` but also runs on "
                        f"{sorted(other_branches - {base})} — the comparison base is the "
                        "wrong branch for those events, so the changed set is wrong "
                        "(R19-14 'missing diff ... is FAIL'). Derive the base from the event."
                    )
            if TIP_ONLY_FALLBACK_RE.search(script):
                problems.append(
                    f"{rel}: falls back to a single-tip-commit diff when the real diff "
                    "fails. That fallback scans one commit, so a multi-commit change is "
                    "partially measured and the run still reports green — R19-14 says a "
                    "filter error is NOT_RUN/FAIL, not an automatic narrowing."
                )

    return problems


def main(argv=None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", default=".", help="repository root")
    args = parser.parse_args(argv)

    root = Path(args.repo).resolve()
    if not root.is_dir():
        print(f"check_mandatory_rows: not a directory: {root}", file=sys.stderr)
        return EXIT_USAGE

    problems = check(root)
    if problems:
        print("MANDATORY CI ROWS: FAIL")
        for p in problems:
            print(f"  - {p}")
        return EXIT_VIOLATIONS

    print("MANDATORY CI ROWS: OK — every mandatory class reaches a protected push, "
          "no path filter narrows a gate, escalation covers each required path class, "
          "and no workflow diffs against a base its triggers contradict")
    return EXIT_OK


if __name__ == "__main__":
    sys.exit(main())
