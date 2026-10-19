#!/usr/bin/env python3
"""Red-on-drift self-test for scripts/check_runner_isolation.py (Spec 19 R19-13).

Every case below is a shape this gate must refuse, taken from the requirement's
own words rather than invented: fork code on the private pool, the label in the
wrong file, a missing main guard, an inherited token, an unbounded concurrency
group, a shared cache, a foreign checkout, and cleanup that is absent or not
last. Each one is measured PASS->FAIL against a tree that otherwise passes, so a
green run cannot come from a gate that never exercises its assertion.

Non-vacuity is proved separately: zero workflows, zero pull_request workflows and
zero self-hosted jobs each FAIL instead of reporting a clean scan of nothing.

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

HOSTED = """\
name: Team Validation
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]
jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: %s # v4.4.0
      - name: Test
        run: echo ok
""" % PINNED


def _write(root: Path, rel: str, body: str) -> None:
    p = root / rel
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(body, encoding="utf-8")


def _tree(root: Path) -> None:
    _write(root, gate.TRUSTED_WORKFLOW, TRUSTED)
    _write(root, ".github/workflows/team-validation.yml", HOSTED)


class CleanTreePasses(unittest.TestCase):
    def test_clean_tree_passes(self):
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            _tree(root)
            files, pr, sh, violations = gate.scan(root)
            self.assertEqual(violations, [])
            self.assertEqual((files, pr, sh), (2, 1, 1))

    def test_main_runs_the_gate_against_this_repository(self):
        repo = Path(__file__).resolve().parent.parent
        _, pr, sh, violations = gate.scan(repo)
        self.assertEqual(violations, [], violations)
        self.assertGreater(pr, 0)
        self.assertEqual(sh, 1)


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
            _write(
                root,
                ".github/workflows/other.yml",
                "name: Other\non:\n  push:\n    branches: [main]\n"
                "jobs:\n  j:\n    runs-on: ubuntu-latest\n    steps:\n"
                "      - run: echo ok\n",
            )
            self.assertEqual(gate.main(["--repo", str(root)]), gate.EXIT_VIOLATIONS)

    def test_zero_self_hosted_jobs_fails(self):
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            _write(root, ".github/workflows/team-validation.yml", HOSTED)
            self.assertEqual(gate.main(["--repo", str(root)]), gate.EXIT_VIOLATIONS)


class TrustedLaneViolations(unittest.TestCase):
    def _violations(self, trusted: str = TRUSTED, hosted: str = HOSTED):
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            _write(root, gate.TRUSTED_WORKFLOW, trusted)
            _write(root, ".github/workflows/team-validation.yml", hosted)
            return gate.scan(root)[3]

    def test_label_in_wrong_file_fails(self):
        v = self._violations(
            hosted=HOSTED.replace("ubuntu-latest", "[self-hosted, Linux, X64, guardrails-ucs03]")
        )
        self.assertTrue(any("may only be used by" in x for x in v), v)

    def test_pull_request_workflow_reaching_self_hosted_fails(self):
        trusted = TRUSTED.replace("    branches: [main]\npermissions",
                                  "    branches: [main]\n  pull_request:\n    branches: [main]\npermissions")
        v = self._violations(trusted=trusted)
        self.assertTrue(any("fork code must never reach" in x for x in v), v)

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


class HostedRuleViolations(unittest.TestCase):
    def test_expression_runner_in_pull_request_fails(self):
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            _write(root, gate.TRUSTED_WORKFLOW, TRUSTED)
            _write(
                root,
                ".github/workflows/team-validation.yml",
                HOSTED.replace("runs-on: ubuntu-latest", "runs-on: ${{ inputs.runner }}"),
            )
            v = gate.scan(root)[3]
            self.assertTrue(any("is an expression" in x for x in v), v)

    def test_unrecognised_label_in_pull_request_fails(self):
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            _write(root, gate.TRUSTED_WORKFLOW, TRUSTED)
            _write(
                root,
                ".github/workflows/team-validation.yml",
                HOSTED.replace("runs-on: ubuntu-latest", "runs-on: my-custom-pool"),
            )
            v = gate.scan(root)[3]
            self.assertTrue(any("not a literal GitHub-hosted label" in x for x in v), v)


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
