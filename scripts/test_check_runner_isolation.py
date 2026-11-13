#!/usr/bin/env python3
"""Red-on-drift self-test for scripts/check_runner_isolation.py (Spec 19 R19-13).

Every case below is a shape this gate must refuse, taken from the requirement's
own words rather than invented: a GitHub-hosted label, an undecidable
`runs-on` expression, a pull_request workflow whose jobs lack the fork guard, an
inverted fork guard that would skip every push, `pull_request_target`, a missing
main guard on the trusted workflow, an inherited token, an unbounded
concurrency group, a shared cache, a foreign checkout, and cleanup that is
absent or not last. Each one is measured PASS->FAIL against a tree that
otherwise passes, so a green run cannot come from a gate that never exercises
its assertion.

Non-vacuity is proved separately: zero workflows, zero pull_request workflows
and zero fleet jobs each FAIL instead of reporting a clean scan of nothing.

Run with: python -m pytest scripts/test_check_runner_isolation.py -v
Or: python scripts/test_check_runner_isolation.py
"""

import json
import os
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from scripts import check_runner_isolation as gate

PINNED = "actions/checkout@11d5960a326750d5838078e36cf38b85af677262"
CLEANUP = "      - name: Post-job cleanup (Spec 19 R19-13)\n        if: always()\n        run: rm -f out.txt\n"

TRUSTED = """\
name: Trusted Main Validation
on:
  push:
    branches: [main]
permissions:
  contents: read
concurrency:
  group: guardrails-ucs03-main
  cancel-in-progress: false
jobs:
  validate-trusted-main:
    if: github.ref == 'refs/heads/main'
    runs-on: [self-hosted, Linux, X64, guardrails-ucs03]
    steps:
      - uses: %s # v4.4.0
      - name: Test
        run: echo ok
%s""" % (PINNED, CLEANUP)

FLEET_PR = """\
name: Team Validation
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]
permissions:
  contents: read
jobs:
  validate:
    if: %s
    runs-on: [self-hosted, Linux, X64, guardrails-ucs03]
    steps:
      - uses: %s # v4.4.0
      - name: Test
        run: echo ok
""" % (gate.FORK_GUARD, PINNED)

GUARD_LINE = "    if: %s\n" % gate.FORK_GUARD

PUSH_ONLY_FLEET = """\
name: Release
on:
  push:
    branches: [main]
jobs:
  release:
    runs-on: [self-hosted, Linux, X64, guardrails-ucs03]
    steps:
      - run: echo ok
"""

HOSTED_PR = """\
name: Hosted
on:
  pull_request:
    branches: [main]
jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - run: echo ok
"""


def _write(root: Path, rel: str, body: str) -> None:
    p = root / rel
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(body, encoding="utf-8")


def _tree(root: Path) -> None:
    _write(root, gate.TRUSTED_WORKFLOW, TRUSTED)
    _write(root, ".github/workflows/team-validation.yml", FLEET_PR)


class CleanTreePasses(unittest.TestCase):
    def test_clean_tree_passes(self):
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            _tree(root)
            files, pr, fleet, violations = gate.scan(root)
            self.assertEqual(violations, [])
            self.assertEqual((files, pr, fleet), (2, 1, 2))

    def test_main_runs_the_gate_against_this_repository(self):
        repo = Path(__file__).resolve().parent.parent
        _, pr, fleet, violations = gate.scan(repo)
        self.assertEqual(violations, [], violations)
        self.assertGreater(pr, 0)
        self.assertGreater(fleet, 0)


class NonVacuity(unittest.TestCase):
    def test_zero_workflows_fails(self):
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            (root / ".github" / "workflows").mkdir(parents=True)
            self.assertEqual(gate.main(["--repo", str(root)]), gate.EXIT_VIOLATIONS)

    def test_zero_pull_request_workflows_fails(self):
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            _write(root, gate.TRUSTED_WORKFLOW, TRUSTED)
            _write(root, ".github/workflows/release.yml", PUSH_ONLY_FLEET)
            self.assertEqual(gate.main(["--repo", str(root)]), gate.EXIT_VIOLATIONS)

    def test_zero_fleet_jobs_fails(self):
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            _write(root, ".github/workflows/team-validation.yml", HOSTED_PR)
            self.assertEqual(gate.main(["--repo", str(root)]), gate.EXIT_VIOLATIONS)


class TrustedLaneViolations(unittest.TestCase):
    def _violations(self, trusted: str = TRUSTED, pr: str = FLEET_PR):
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            _write(root, gate.TRUSTED_WORKFLOW, trusted)
            _write(root, ".github/workflows/team-validation.yml", pr)
            return gate.scan(root)[3]

    def test_fleet_label_in_another_file_is_allowed(self):
        # The owner's routing change: every workflow runs on the fleet, so the
        # label is no longer confined to one file. A same-repo PR run on the
        # fleet with the guard is the intended shape.
        self.assertEqual(self._violations(), [])

    def test_missing_main_guard_fails(self):
        v = self._violations(
            trusted=TRUSTED.replace("    if: github.ref == 'refs/heads/main'\n", "")
        )
        self.assertTrue(any("refs/heads/main" in x for x in v), v)

    def test_inherited_token_fails(self):
        v = self._violations(
            trusted=TRUSTED.replace("permissions:\n  contents: read\n", "")
        )
        self.assertTrue(any("no explicit `permissions`" in x for x in v), v)

    def test_write_scope_fails(self):
        v = self._violations(
            trusted=TRUSTED.replace("  contents: read", "  contents: write")
        )
        self.assertTrue(any("must not hold a write scope" in x for x in v), v)

    def test_missing_concurrency_fails(self):
        v = self._violations(
            trusted=TRUSTED.replace(
                "concurrency:\n  group: guardrails-ucs03-main\n  cancel-in-progress: false\n", ""
            )
        )
        self.assertTrue(any("concurrency.group" in x for x in v), v)

    def test_implied_cancel_in_progress_fails(self):
        v = self._violations(
            trusted=TRUSTED.replace("  cancel-in-progress: false\n", "")
        )
        self.assertTrue(any("cancel-in-progress" in x for x in v), v)

    def test_shared_cache_fails(self):
        v = self._violations(
            trusted=TRUSTED.replace(
                "      - name: Test\n",
                "      - uses: actions/cache@0057852bfaa89a56745cba8c7296529d2fc39830\n"
                "      - name: Test\n",
            )
        )
        self.assertTrue(any("shared action cache" in x for x in v), v)

    def test_shared_cache_fails_in_any_fleet_workflow(self):
        v = self._violations(
            pr=FLEET_PR.replace(
                "      - name: Test\n",
                "      - uses: actions/cache@0057852bfaa89a56745cba8c7296529d2fc39830\n"
                "      - name: Test\n",
            )
        )
        self.assertTrue(any("shared action cache" in x for x in v), v)

    def test_foreign_repository_checkout_fails(self):
        v = self._violations(
            trusted=TRUSTED.replace(
                "        run: echo ok\n      - name: Post",
                "        run: echo ok\n"
                "      - uses: " + PINNED + "\n"
                "        with:\n          repository: TheArchitectit/other\n"
                "      - name: Post",
            )
        )
        self.assertTrue(any("cross-repository checkout" in x for x in v), v)

    def test_non_default_workspace_path_fails(self):
        v = self._violations(
            trusted=TRUSTED.replace(
                "        run: echo ok\n      - name: Post",
                "        run: echo ok\n"
                "      - uses: " + PINNED + "\n"
                "        with:\n          path: shared\n"
                "      - name: Post",
            )
        )
        self.assertTrue(any("non-default workspace path" in x for x in v), v)

    def test_missing_cleanup_fails(self):
        v = self._violations(trusted=TRUSTED.replace(CLEANUP, ""))
        self.assertTrue(any("post-job cleanup is required" in x for x in v), v)

    def test_cleanup_not_last_fails(self):
        moved = TRUSTED.replace(CLEANUP, "") + CLEANUP.replace(
            "      - name: Post-job cleanup (Spec 19 R19-13)\n"
            "        if: always()\n"
            "        run: rm -f out.txt\n",
            "      - name: Post-job cleanup (Spec 19 R19-13)\n"
            "        if: always()\n"
            "        run: rm -f out.txt\n"
            "      - name: Something after\n"
            "        run: echo late\n",
        )
        v = self._violations(trusted=moved)
        self.assertTrue(any("not the last step" in x for x in v), v)


class ForkGuardViolations(unittest.TestCase):
    """A pull_request workflow may use the fleet only behind the fork guard."""

    def _violations(self, workflow: str):
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            _write(root, gate.TRUSTED_WORKFLOW, TRUSTED)
            _write(root, ".github/workflows/team-validation.yml", workflow)
            return gate.scan(root)[3]

    def test_missing_fork_guard_fails(self):
        v = self._violations(FLEET_PR.replace(GUARD_LINE, ""))
        self.assertTrue(any("lacks the fork guard" in x for x in v), v)

    def test_wrong_fork_guard_operator_fails(self):
        # `&&` in place of `||` would skip every push run; it is not a guard.
        v = self._violations(FLEET_PR.replace(GUARD_LINE, GUARD_LINE.replace("||", "&&")))
        self.assertTrue(any("lacks the fork guard" in x for x in v), v)

    def test_every_job_needs_the_guard(self):
        two_jobs = FLEET_PR.replace(
            "      - name: Test\n        run: echo ok\n",
            "      - name: Test\n        run: echo ok\n"
            "  unguarded:\n"
            "    runs-on: [self-hosted, Linux, X64, guardrails-ucs03]\n"
            "    steps:\n"
            "      - run: echo second\n",
        )
        v = self._violations(two_jobs)
        self.assertTrue(any("unguarded" in x and "lacks the fork guard" in x for x in v), v)

    def test_pull_request_target_fails(self):
        v = self._violations(
            FLEET_PR.replace("  pull_request:\n", "  pull_request_target:\n")
        )
        self.assertTrue(any("pull_request_target" in x for x in v), v)


class FleetRoutingViolations(unittest.TestCase):
    def _violations(self, workflow: str):
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            _write(root, gate.TRUSTED_WORKFLOW, TRUSTED)
            _write(root, ".github/workflows/team-validation.yml", workflow)
            return gate.scan(root)[3]

    def test_hosted_label_fails(self):
        v = self._violations(
            FLEET_PR.replace(
                "runs-on: [self-hosted, Linux, X64, guardrails-ucs03]",
                "runs-on: ubuntu-latest",
            )
        )
        self.assertTrue(any("GitHub-hosted label" in x for x in v), v)

    def test_expression_runner_fails(self):
        v = self._violations(
            FLEET_PR.replace(
                "runs-on: [self-hosted, Linux, X64, guardrails-ucs03]",
                "runs-on: ${{ inputs.runner }}",
            )
        )
        self.assertTrue(any("is an expression" in x for x in v), v)

    def test_unrecognised_label_fails(self):
        v = self._violations(
            FLEET_PR.replace(
                "runs-on: [self-hosted, Linux, X64, guardrails-ucs03]",
                "runs-on: my-custom-pool",
            )
        )
        self.assertTrue(any("does not target the fleet" in x for x in v), v)


class Attestation(unittest.TestCase):
    def _record(self, root: Path, **overrides) -> Path:
        record = {
            "schema": "runner-isolation/v1",
            "repo": "TheArchitectit/agent-guardrails-template",
            "runner_name": "ucs03-gr",
            "workspace": "/home/runner/_work/agent-guardrails-template",
            "clean_checkout": True,
            "sibling_repositories": [],
            "cross_repository_checkouts": [],
            "violations": [],
        }
        record.update(overrides)
        p = root / "attestation.json"
        p.write_text(json.dumps(record), encoding="utf-8")
        return p

    def test_clean_attestation_passes(self):
        with tempfile.TemporaryDirectory() as d:
            self.assertEqual(gate.check_attestation(self._record(Path(d))), [])

    def test_missing_attestation_fails(self):
        with tempfile.TemporaryDirectory() as d:
            problems = gate.check_attestation(Path(d) / "absent.json")
            self.assertTrue(any("no runner isolation attestation" in x for x in problems))

    def test_malformed_attestation_fails(self):
        with tempfile.TemporaryDirectory() as d:
            p = Path(d) / "bad.json"
            p.write_text("{not json", encoding="utf-8")
            self.assertTrue(gate.check_attestation(p))

    def test_dirty_checkout_fails(self):
        with tempfile.TemporaryDirectory() as d:
            problems = gate.check_attestation(self._record(Path(d), clean_checkout=False))
            self.assertTrue(any("carried residue" in x for x in problems))

    def test_sibling_repository_fails(self):
        with tempfile.TemporaryDirectory() as d:
            problems = gate.check_attestation(
                self._record(Path(d), sibling_repositories=["other-repo"])
            )
            self.assertTrue(any("sibling repositories" in x for x in problems))

    def test_cross_repository_checkout_fails(self):
        with tempfile.TemporaryDirectory() as d:
            problems = gate.check_attestation(
                self._record(Path(d), cross_repository_checkouts=["openagentplatform"])
            )
            self.assertTrue(any("runner is not repo-scoped" in x for x in problems))

    def test_recorded_violation_fails(self):
        with tempfile.TemporaryDirectory() as d:
            problems = gate.check_attestation(
                self._record(Path(d), violations=["workspace is group-writable"])
            )
            self.assertTrue(any("recorded violations" in x for x in problems))

    def test_missing_key_fails(self):
        with tempfile.TemporaryDirectory() as d:
            record = self._record(Path(d))
            data = json.loads(record.read_text(encoding="utf-8"))
            del data["runner_name"]
            record.write_text(json.dumps(data), encoding="utf-8")
            self.assertTrue(any("missing `runner_name`" in x for x in gate.check_attestation(record)))


if __name__ == "__main__":
    unittest.main(verbosity=2)
