#!/usr/bin/env python3
"""check_runner_isolation.py — Spec 19 R19-13 runner trust and repository isolation.

Spec 19 R19-13, verbatim:

    All CI MUST execute on the repository's own UCS03 runner
    (`guardrails-ucs03`); GitHub-hosted runners MUST NOT be used.
    Same-repository pull requests MAY run on the fleet only when every job
    carries a fork guard evaluated before runner assignment, so fork PR code
    never executes on it; `pull_request_target` MUST NOT be used. The UCS03
    job MUST be repo-scoped and isolated: clean checkout, no cross-repository
    workspace/artifact/cache/credential reuse, least-privilege token, no
    production credentials, bounded concurrency, and post-job cleanup. A runner
    label alone is not isolation evidence. Release acceptance MUST include an
    isolation probe or recorded host attestation.

That last sentence is the reason this gate exists in this shape. Before it, the
only thing asserting the self-hosted posture was prose in a comment above the
`runs-on:` line — and a comment cannot fail a build. A `runs-on:` label says
where a job was *routed*; it says nothing about whether the job shared a
workspace, a cache, or a token with a previous job on the same host.

What is checked, per file and per job:

1. **All CI runs on the repo-scoped UCS03 fleet.** Every job must target the
   `guardrails-ucs03` label. A literal GitHub-hosted label (`ubuntu-*`,
   `windows-*`, `macos-*`) is a VIOLATION, and so is an expression `runs-on:`
   (e.g. `${{ inputs.runner }}`), because its value is not decidable at review
   time.
2. **Fork code never reaches the fleet.** Same-repository pull requests run on
   the fleet, but every job in a workflow that declares `pull_request` MUST
   carry the fork guard

       github.event_name != 'pull_request' ||
       github.event.pull_request.head.repo.full_name == github.repository

   which GitHub evaluates before assigning a runner, so a fork PR's jobs are
   skipped rather than dispatched. `pull_request_target` is a VIOLATION
   outright: it runs a base-repository workflow with a PR-controlled checkout
   and a writable token.
3. **The trusted-main job's isolation posture is declared, not implied.** The
   job in `team-validation-fleet.yml` must declare least-privilege
   `permissions` (no `write` scope, nothing omitted), a `concurrency` group
   with an explicit `cancel-in-progress` (bounded concurrency), a checkout with
   no `path:` and no `repository:`, no `actions/cache` use (a cache on a
   persistent host is restored across runs and repositories by key), and a
   final `if: always()` cleanup step (post-job cleanup). The no-cache and
   repo-scoped-checkout rules apply to every fleet job, not only the trusted
   one.
4. **Non-vacuity.** Zero workflow files, zero `pull_request` workflows, or zero
   fleet jobs each FAIL. A green run must mean the rules were exercised against
   a tree that actually contains what they govern.

Clause 5 (isolation probe / recorded host attestation) cannot be decided by
reading YAML, so it is not faked here: `scripts/runner_isolation_probe.py` runs
on the host and writes the attestation, and this gate's `--attestation` mode
fails the job when that record is missing, malformed, or reports a violated
property.

Exit: 0 clean; 1 a violation (or a vacuous scan); 2 usage.
"""
from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

import yaml

EXIT_OK = 0
EXIT_VIOLATIONS = 1
EXIT_USAGE = 2

WORKFLOW_DIR = Path(".github") / "workflows"

# The workflow whose job must carry the full isolation posture and the host
# attestation (R19-13 clause 3 / clause 5).
TRUSTED_WORKFLOW = ".github/workflows/team-validation-fleet.yml"
# The label that identifies the fleet host pool. Every job targets it.
TRUSTED_LABEL = "guardrails-ucs03"
# GitHub-hosted runner label prefixes. Any of these is a violation.
HOSTED_PREFIXES = ("ubuntu-", "windows-", "macos-")

# Read by GitHub BEFORE a runner is assigned. A job carrying this condition is
# skipped for fork pull requests, so untrusted contribution code never reaches
# the fleet. The gate requires the canonical contiguous expression rather than
# accepting "something about forks": an inverted operator (`&&`) would skip
# every push run and must not pass as a guard.
FORK_GUARD = (
    "github.event_name != 'pull_request' || "
    "github.event.pull_request.head.repo.full_name == github.repository"
)

# Scopes a read-only validation job never needs. `contents: read` is the whole
# requirement; anything else must be justified by a reviewer, not inherited.
WRITE_SCOPES = ("write", "admin")

ATTESTATION_REQUIRED_KEYS = (
    "schema",
    "repo",
    "runner_name",
    "workspace",
    "clean_checkout",
    "sibling_repositories",
    "cross_repository_checkouts",
    "violations",
)


def workflow_files(repo: Path) -> list[Path]:
    d = repo / WORKFLOW_DIR
    if not d.is_dir():
        return []
    return sorted(p for p in d.iterdir() if p.suffix in (".yml", ".yaml"))


def _triggers(doc: dict) -> set[str]:
    """Return the normalised trigger names.

    PyYAML resolves the bare key `on` to the boolean True (YAML 1.1), so both
    spellings are read — reading only one would silently scan zero triggers and
    turn the pull_request rule into an empty pass.
    """
    raw = doc.get("on")
    if raw is None and True in doc:
        raw = doc[True]
    if raw is None:
        return set()
    if isinstance(raw, str):
        return {raw}
    if isinstance(raw, list):
        return {str(x) for x in raw}
    if isinstance(raw, dict):
        return {str(k) for k in raw}
    return set()


def _as_labels(runs_on: object) -> list[str]:
    if isinstance(runs_on, str):
        return [runs_on]
    if isinstance(runs_on, list):
        return [str(x) for x in runs_on]
    return []


def _on_fleet(labels: list[str]) -> bool:
    """True when the job targets the fleet label (directly or in a list)."""
    return any(label.strip() == TRUSTED_LABEL for label in labels)


def _normalise(expr: object) -> str:
    return " ".join(str(expr or "").split())


def _fork_guarded(job: dict) -> bool:
    return _normalise(FORK_GUARD) in _normalise(job.get("if"))


def _is_literal_hosted(labels: list[str]) -> bool:
    """True only for a literal GitHub-hosted label.

    A `${{ ... }}` expression is NOT hosted-by-assumption: it is unresolvable
    at review time and is reported as a violation by the caller.
    """
    return (
        len(labels) == 1
        and not labels[0].startswith("${{")
        and labels[0].startswith(HOSTED_PREFIXES)
    )


def _has_expression(labels: list[str]) -> bool:
    return any("${{" in label for label in labels)


def _jobs(doc: dict) -> dict:
    jobs = doc.get("jobs")
    return jobs if isinstance(jobs, dict) else {}


def _steps(job: dict) -> list[dict]:
    steps = job.get("steps")
    return [s for s in steps if isinstance(s, dict)] if isinstance(steps, list) else []


def _step_uses(step: dict) -> str:
    return str(step.get("uses") or "")


def _step_name(step: dict) -> str:
    return str(step.get("name") or "")


def check_permissions(doc: dict, job: dict, where: str) -> list[str]:
    """Least privilege: an explicit block with no write scope.

    An omitted block is not accepted, because the omission resolves to the
    repository default (which is writable in many repositories) rather than to
    the read-only token this job is supposed to hold.
    """
    block = job.get("permissions", doc.get("permissions"))
    if not isinstance(block, dict) or not block:
        return [
            f"{where}: no explicit `permissions` block — the job inherits the "
            f"repository default token instead of a least-privilege one (R19-13)"
        ]
    problems: list[str] = []
    for scope, access in block.items():
        text = str(access).strip()
        if text in WRITE_SCOPES:
            problems.append(
                f"{where}: permissions grants `{scope}: {text}` — a read-only "
                f"validation job must not hold a write scope (R19-13)"
            )
        elif text not in ("read", "none"):
            problems.append(
                f"{where}: permissions `{scope}: {text}` is not a decidable "
                f"least-privilege value (R19-13)"
            )
    return problems


def check_reuse(job: dict, where: str) -> list[str]:
    """No cross-repository reuse in any job that runs on the fleet.

    A shared action cache on a persistent host is restored across runs and
    repositories by key, and a checkout `path:`/`repository:` is shared or
    foreign workspace state rather than a repo-scoped checkout.
    """
    problems: list[str] = []
    for step in _steps(job):
        uses = _step_uses(step)
        label = f"{where}: step `{_step_name(step) or uses or '<unnamed>'}`"

        if uses.startswith("actions/cache"):
            problems.append(
                f"{label}: uses a shared action cache — on a persistent host a "
                f"cache key can be restored across runs and repositories "
                f"(R19-13)"
            )
        if uses.startswith("actions/checkout"):
            with_ = step.get("with") or {}
            if not isinstance(with_, dict):
                with_ = {}
            if "path" in with_:
                problems.append(
                    f"{label}: checkout sets `path: {with_['path']}` — a "
                    f"non-default workspace path is shared state, not a "
                    f"repo-scoped checkout (R19-13)"
                )
            if "repository" in with_:
                problems.append(
                    f"{label}: checkout pulls `repository: "
                    f"{with_['repository']}` — cross-repository checkout in "
                    f"a fleet job (R19-13)"
                )
    return problems


def check_trusted_job(
    doc: dict, job: dict, triggers: set[str], where: str
) -> list[str]:
    """Isolation posture of the attested trusted-main job."""
    problems: list[str] = []

    guard = str(job.get("if") or "")
    if "refs/heads/main" not in guard:
        problems.append(
            f"{where}: runs on the self-hosted pool without a "
            f"`refs/heads/main` guard (`if: {guard or '<absent>'}`) — an "
            f"untrusted ref would then execute on the private host (R19-13)"
        )
    if "push" not in triggers:
        problems.append(
            f"{where}: trusted-main workflow does not declare a `push` "
            f"trigger (triggers: {sorted(triggers)}) (R19-13)"
        )

    problems.extend(check_permissions(doc, job, where))

    concurrency = doc.get("concurrency")
    if not isinstance(concurrency, dict) or "group" not in concurrency:
        problems.append(
            f"{where}: no `concurrency.group` — concurrent runs on one host "
            f"are unbounded (R19-13)"
        )
    elif "cancel-in-progress" not in concurrency:
        problems.append(
            f"{where}: `concurrency` omits `cancel-in-progress`, so the "
            f"bound is implied rather than declared (R19-13)"
        )

    steps = _steps(job)
    if not steps:
        problems.append(f"{where}: job has no steps (R19-13)")
        return problems

    problems.extend(check_reuse(job, where))

    names = [_step_name(s) for s in steps]
    cleanup = [
        s
        for s in steps
        if "if" in s and "always()" in str(s["if"]) and "clean" in _step_name(s).lower()
    ]
    if not cleanup:
        problems.append(
            f"{where}: no final `if: always()` cleanup step (steps: {names}) — "
            f"post-job cleanup is required so a failed run cannot leave "
            f"workspace or artifact residue for the next one (R19-13)"
        )
    else:
        last = steps[-1]
        if last not in cleanup:
            problems.append(
                f"{where}: the `if: always()` cleanup step is not the last "
                f"step (`{_step_name(last)}` follows it) — cleanup that runs "
                f"before other work does not clean up after it (R19-13)"
            )

    return problems


def scan(repo: Path) -> tuple[int, int, int, list[str]]:
    """Return (workflows, pull_request_workflows, fleet_jobs, violations)."""
    files = workflow_files(repo)
    violations: list[str] = []
    pr_workflows = 0
    fleet_jobs = 0

    for wf in files:
        rel = wf.relative_to(repo).as_posix()
        try:
            doc = yaml.safe_load(wf.read_text(encoding="utf-8"))
        except (OSError, yaml.YAMLError) as exc:
            violations.append(f"{rel}: unreadable/unparsable workflow ({exc})")
            continue
        if not isinstance(doc, dict):
            violations.append(f"{rel}: workflow is not a mapping")
            continue

        triggers = _triggers(doc)
        is_pr = "pull_request" in triggers
        if is_pr:
            pr_workflows += 1
        if "pull_request_target" in triggers:
            violations.append(
                f"{rel}: declares `pull_request_target` — it runs a "
                f"base-repository workflow against PR-controlled code with a "
                f"writable token and must not be used (R19-13)"
            )

        for job_id, job in _jobs(doc).items():
            if not isinstance(job, dict):
                continue
            where = f"{rel}:{job_id}"
            labels = _as_labels(job.get("runs-on"))

            if _has_expression(labels):
                violations.append(
                    f"{where}: `runs-on: {labels}` is an expression — its "
                    f"value cannot be reviewed, and an input can resolve it to "
                    f"a private or cloud runner (R19-13)"
                )
                continue
            if _is_literal_hosted(labels):
                violations.append(
                    f"{where}: `runs-on: {labels}` is a GitHub-hosted label — "
                    f"all CI must run on the repo-scoped UCS03 fleet (R19-13)"
                )
                continue
            if not _on_fleet(labels):
                violations.append(
                    f"{where}: `runs-on: {labels}` does not target the fleet "
                    f"label `{TRUSTED_LABEL}` (R19-13)"
                )
                continue

            fleet_jobs += 1
            violations.extend(check_reuse(job, where))

            if is_pr and not _fork_guarded(job):
                violations.append(
                    f"{where}: workflow declares `pull_request` and this job "
                    f"lacks the fork guard (`if: {job.get('if') or '<absent>'}`) "
                    f"— a fork PR would execute on the private runner (R19-13)"
                )

            if rel == TRUSTED_WORKFLOW:
                violations.extend(check_trusted_job(doc, job, triggers, where))

    return len(files), pr_workflows, fleet_jobs, violations


def check_attestation(path: Path) -> list[str]:
    """Clause 5: a recorded host attestation, or the release says so."""
    if not path.exists():
        return [
            f"{path}: no runner isolation attestation was recorded — R19-13 "
            f"requires an isolation probe or recorded host attestation as "
            f"release acceptance, and a runner label is not evidence"
        ]
    try:
        record = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        return [f"{path}: attestation is unreadable/unparsable ({exc})"]
    if not isinstance(record, dict):
        return [f"{path}: attestation is not a JSON object"]

    problems = [
        f"{path}: attestation is missing `{key}`" for key in ATTESTATION_REQUIRED_KEYS
        if key not in record
    ]
    if problems:
        return problems

    if record.get("schema") != "runner-isolation/v1":
        problems.append(
            f"{path}: schema is {record.get('schema')!r}, expected "
            f"'runner-isolation/v1'"
        )
    if record.get("clean_checkout") is not True:
        problems.append(
            f"{path}: probe reported `clean_checkout: "
            f"{record.get('clean_checkout')!r}` — the workspace carried "
            f"residue from a previous run (R19-13)"
        )
    siblings = record.get("sibling_repositories")
    if not isinstance(siblings, list) or siblings:
        problems.append(
            f"{path}: probe found sibling repositories in the workspace root "
            f"({siblings!r}) — the checkout is not repo-scoped (R19-13)"
        )
    cross = record.get("cross_repository_checkouts")
    if not isinstance(cross, list) or cross:
        problems.append(
            f"{path}: probe found other repositories under the runner work "
            f"root ({cross!r}) — the runner is not repo-scoped (R19-13)"
        )
    recorded = record.get("violations")
    if not isinstance(recorded, list) or recorded:
        problems.append(
            f"{path}: probe recorded violations: {recorded!r} (R19-13)"
        )
    return problems


def main(argv: list[str]) -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--repo", default=".", help="repository root")
    ap.add_argument(
        "--attestation",
        default=None,
        help="recorded host attestation to require and validate (R19-13 clause 5)",
    )
    args = ap.parse_args(argv)

    repo = Path(args.repo).resolve()
    files, pr_workflows, fleet_jobs, violations = scan(repo)

    if files == 0:
        print(
            f"R19-13 FAIL: no workflow files under {WORKFLOW_DIR} — the gate "
            f"scanned nothing (no vacuous green)",
            file=sys.stderr,
        )
        return EXIT_VIOLATIONS
    if pr_workflows == 0:
        print(
            f"R19-13 FAIL: {files} workflow(s) but none declares `pull_request` "
            f"— the fork-guard rule was never exercised (no vacuous green)",
            file=sys.stderr,
        )
        return EXIT_VIOLATIONS
    if fleet_jobs == 0:
        print(
            f"R19-13 FAIL: no job targets the fleet label `{TRUSTED_LABEL}`, so "
            f"the fleet rules were not exercised against the tree (no vacuous "
            f"green). If the fleet lane was retired, remove this assertion "
            f"deliberately rather than letting it pass on an empty set.",
            file=sys.stderr,
        )
        return EXIT_VIOLATIONS

    if args.attestation:
        violations.extend(check_attestation(Path(args.attestation)))

    if violations:
        for v in violations:
            print(f"R19-13 VIOLATION: {v}")
        print(f"\nfail: {len(violations)} violation(s)")
        return EXIT_VIOLATIONS

    print(
        f"R19-13 OK: {files} workflow(s), {fleet_jobs} job(s) on `{TRUSTED_LABEL}`, "
        f"{pr_workflows} pull_request workflow(s) with a fork guard on every job, "
        f"and {TRUSTED_WORKFLOW} with least-privilege permissions, bounded "
        "concurrency, repo-scoped checkout and post-job cleanup"
        + (f"; attestation verified at {args.attestation}" if args.attestation else "")
    )
    return EXIT_OK


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
