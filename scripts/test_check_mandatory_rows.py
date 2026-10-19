#!/usr/bin/env python3
"""Red-on-drift self-test for scripts/check_mandatory_rows.py (Spec 19 R19-14).

Each case builds a throwaway tree containing only what the gate reads, then
mutates ONE thing and asserts the gate goes red. A green run therefore means the
assertion was exercised, not skipped.

Run with: python -m pytest scripts/test_check_mandatory_rows.py -v
Or:       python scripts/test_check_mandatory_rows.py
"""

import json
import os
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from scripts import check_mandatory_rows as gate

PINNED = "actions/checkout@11d5960a326750d5838078e36cf38b85af677262"

# A workflow that runs on a protected push and carries one named step.
GOOD_WORKFLOW = f"""\
name: Team Validation
on:
  push:
    branches: [main, master]
permissions:
  contents: read
jobs:
  security:
    runs-on: ubuntu-latest
    steps:
      - uses: {PINNED}
      - name: Web fail-closed middleware + REST authorization row
        run: go test ./internal/web
      - name: Registry fail-closed + lifecycle row
        run: go test ./internal/auth
      - name: Readiness and lifecycle row
        run: go test ./internal/health
      - name: Run every enumerated Python test suite
        run: python -m pytest scripts/
"""

SECRET_WORKFLOW = f"""\
name: Secret Validation
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]
permissions:
  contents: read
jobs:
  scan:
    runs-on: ubuntu-latest
    steps:
      - uses: {PINNED}
      - name: Run Gitleaks
        run: gitleaks detect
"""

GUARD_WORKFLOW = f"""\
name: Regression Guard
on:
  push:
    branches: [main]
permissions:
  contents: read
jobs:
  guard:
    runs-on: ubuntu-latest
    steps:
      - uses: {PINNED}
      - name: Go module discovery (Spec 19 R19-10)
        run: python scripts/check_go_modules.py
"""


def good_manifest():
    return {
        "version": 1,
        "rows": [
            {"class": "security", "workflow": ".github/workflows/team-validation.yml",
             "step": "Web fail-closed middleware + REST authorization row"},
            {"class": "secret", "workflow": ".github/workflows/secret-validation.yml",
             "step": "Run Gitleaks"},
            {"class": "registry", "workflow": ".github/workflows/team-validation.yml",
             "step": "Registry fail-closed + lifecycle row"},
            {"class": "deployment-readiness",
             "workflow": ".github/workflows/team-validation.yml",
             "step": "Readiness and lifecycle row"},
            {"class": "package-inventory",
             "workflow": ".github/workflows/regression-guard.yml",
             "step": "Go module discovery (Spec 19 R19-10)"},
            {"class": "test-floor", "workflow": ".github/workflows/team-validation.yml",
             "step": "Run every enumerated Python test suite"},
        ],
        "full_matrix_escalation": {k: ["x"] for k in gate.REQUIRED_ESCALATION_KEYS},
    }


class Tree:
    """A throwaway repo root holding only the files the gate reads."""

    def __init__(self, workflows=None, manifest=None):
        self._tmp = tempfile.TemporaryDirectory()
        root = Path(self._tmp.name) / "repo"
        wf = root / ".github" / "workflows"
        wf.mkdir(parents=True)
        (root / ".guardrails").mkdir()

        files = workflows if workflows is not None else {
            "team-validation.yml": GOOD_WORKFLOW,
            "secret-validation.yml": SECRET_WORKFLOW,
            "regression-guard.yml": GUARD_WORKFLOW,
        }
        for name, text in files.items():
            (wf / name).write_text(text, encoding="utf-8")

        payload = good_manifest() if manifest is None else manifest
        if payload is not False:  # False = write no manifest at all
            (root / gate.MANIFEST).write_text(json.dumps(payload), encoding="utf-8")
        self.root = root

    def check(self):
        return gate.check(self.root)

    def close(self):
        self._tmp.cleanup()


class CleanTree(unittest.TestCase):
    def test_clean_tree_passes(self):
        t = Tree()
        try:
            self.assertEqual(t.check(), [])
        finally:
            t.close()
        # The real tree must also pass, or this gate is unusable.
        real = Path(__file__).resolve().parent.parent
        self.assertEqual(gate.check(real), [])


class FilterDiscipline(unittest.TestCase):
    def test_paths_filter_is_refused(self):
        broken = GUARD_WORKFLOW.replace(
            "  push:\n    branches: [main]\n",
            "  push:\n    branches: [main]\n    paths: ['docs/**']\n",
        )
        t = Tree(workflows={
            "team-validation.yml": GOOD_WORKFLOW,
            "secret-validation.yml": SECRET_WORKFLOW,
            "regression-guard.yml": broken,
        })
        try:
            problems = t.check()
            self.assertTrue(any("path filter" in p for p in problems), problems)
        finally:
            t.close()

    def test_paths_ignore_is_refused(self):
        broken = GOOD_WORKFLOW.replace(
            "  push:\n    branches: [main, master]\n",
            "  push:\n    branches: [main, master]\n    paths-ignore: ['**/*.md']\n",
        )
        t = Tree(workflows={
            "team-validation.yml": broken,
            "secret-validation.yml": SECRET_WORKFLOW,
            "regression-guard.yml": GUARD_WORKFLOW,
        })
        try:
            self.assertTrue(any("paths-ignore" in p for p in t.check()))
        finally:
            t.close()

    def test_missing_escalation_class_is_refused(self):
        m = good_manifest()
        del m["full_matrix_escalation"]["dependency_file"]
        t = Tree(manifest=m)
        try:
            problems = t.check()
            self.assertTrue(
                any("dependency_file" in p for p in problems), problems
            )
        finally:
            t.close()

    def test_empty_escalation_list_is_refused(self):
        m = good_manifest()
        m["full_matrix_escalation"]["workflow"] = []
        t = Tree(manifest=m)
        try:
            self.assertTrue(
                any("full_matrix_escalation.workflow" in p for p in t.check())
            )
        finally:
            t.close()


class Reachability(unittest.TestCase):
    def test_pr_only_workflow_cannot_satisfy_a_mandatory_row(self):
        # The defect this gate exists for: the step exists, the name matches,
        # and it never runs on a protected push.
        pr_only = SECRET_WORKFLOW.replace(
            "  push:\n    branches: [main]\n",
            "",
        )
        t = Tree(workflows={
            "team-validation.yml": GOOD_WORKFLOW,
            "secret-validation.yml": pr_only,
            "regression-guard.yml": GUARD_WORKFLOW,
        })
        try:
            problems = t.check()
            self.assertTrue(
                any("secret" in p and "protected push" in p for p in problems),
                problems,
            )
        finally:
            t.close()

    def test_named_step_that_does_not_exist_is_refused(self):
        m = good_manifest()
        m["rows"][0]["step"] = "Web fail-closed middleware row that nobody wrote"
        t = Tree(manifest=m)
        try:
            problems = t.check()
            self.assertTrue(any("does not exist" in p for p in problems), problems)
        finally:
            t.close()

    def test_named_workflow_that_does_not_exist_is_refused(self):
        m = good_manifest()
        m["rows"][0]["workflow"] = ".github/workflows/ghost.yml"
        t = Tree(manifest=m)
        try:
            self.assertTrue(any("does not exist" in p for p in t.check()))
        finally:
            t.close()

    def test_missing_mandatory_class_is_refused(self):
        m = good_manifest()
        m["rows"] = [r for r in m["rows"] if r["class"] != "test-floor"]
        t = Tree(manifest=m)
        try:
            problems = t.check()
            self.assertTrue(
                any("`test-floor`" in p and "not declared" in p for p in problems),
                problems,
            )
        finally:
            t.close()


class DiffCorrectness(unittest.TestCase):
    def test_wrong_hardcoded_base_is_refused(self):
        broken = GUARD_WORKFLOW.replace(
            "  push:\n    branches: [main]\n",
            "  push:\n    branches: [main, develop]\n",
        ).replace(
            "        run: python scripts/check_go_modules.py",
            "        run: git diff --name-only origin/main...HEAD",
        )
        t = Tree(workflows={
            "team-validation.yml": GOOD_WORKFLOW,
            "secret-validation.yml": SECRET_WORKFLOW,
            "regression-guard.yml": broken,
        })
        try:
            problems = t.check()
            self.assertTrue(any("hardcoded" in p for p in problems), problems)
        finally:
            t.close()

    def test_tip_only_fallback_is_refused(self):
        broken = GUARD_WORKFLOW.replace(
            "        run: python scripts/check_go_modules.py",
            "        run: git diff --name-only origin/main...HEAD || git diff --name-only HEAD~1",
        )
        t = Tree(workflows={
            "team-validation.yml": GOOD_WORKFLOW,
            "secret-validation.yml": SECRET_WORKFLOW,
            "regression-guard.yml": broken,
        })
        try:
            problems = t.check()
            self.assertTrue(
                any("single-tip-commit" in p or "narrowing" in p for p in problems),
                problems,
            )
        finally:
            t.close()

    def test_comment_quoting_the_old_command_is_not_a_violation(self):
        # A record of a fixed defect must not fail the gate, or the next
        # maintainer deletes the explanation instead of the defect.
        annotated = GUARD_WORKFLOW.replace(
            "  push:\n    branches: [main]\n",
            "  push:\n    branches: [main]\n"
            "#   previously: git diff --name-only origin/main...HEAD || git diff --name-only HEAD~1\n",
        )
        t = Tree(workflows={
            "team-validation.yml": GOOD_WORKFLOW,
            "secret-validation.yml": SECRET_WORKFLOW,
            "regression-guard.yml": annotated,
        })
        try:
            self.assertEqual(t.check(), [])
        finally:
            t.close()


class NonVacuity(unittest.TestCase):
    def test_no_workflows_fails(self):
        t = Tree(workflows={})
        try:
            problems = t.check()
            self.assertTrue(any("VACUOUS" in p for p in problems), problems)
        finally:
            t.close()

    def test_no_manifest_fails(self):
        t = Tree(manifest=False)
        try:
            self.assertTrue(any("missing" in p for p in t.check()))
        finally:
            t.close()

    def test_empty_rows_fails(self):
        m = good_manifest()
        m["rows"] = []
        t = Tree(manifest=m)
        try:
            self.assertTrue(any("zero rows" in p for p in t.check()))
        finally:
            t.close()

    def test_malformed_manifest_fails_not_empty(self):
        t = Tree()
        try:
            (t.root / gate.MANIFEST).write_text("{not json", encoding="utf-8")
            self.assertTrue(any("unparseable" in p for p in t.check()))
        finally:
            t.close()

    def test_every_mandatory_row_off_a_protected_push_is_refused(self):
        # Aggregate vacuity only fires when NO workflow runs on a protected
        # push. Here other workflows still do, so the defect must surface per
        # row: each mandatory class that lost its protected-push carrier is
        # named. Asserting the aggregate here would be asserting the wrong
        # branch of the gate.
        pr_only = GOOD_WORKFLOW.replace(
            "  push:\n    branches: [main, master]\n", ""
        ).replace("on:\n", "on:\n  pull_request:\n    branches: [main]\n")
        t = Tree(workflows={
            "team-validation.yml": pr_only,
            "secret-validation.yml": SECRET_WORKFLOW,
            "regression-guard.yml": GUARD_WORKFLOW,
        })
        try:
            problems = t.check()
            for cls in ("security", "registry", "deployment-readiness", "test-floor"):
                with self.subTest(cls=cls):
                    self.assertTrue(
                        any(f"row `{cls}`" in p and "protected push" in p for p in problems),
                        problems,
                    )
            # And no protected push at all IS vacuous.
            self.assertFalse(any("VACUOUS" in p for p in problems), problems)
        finally:
            t.close()

    def test_no_workflow_on_a_protected_push_is_vacuous(self):
        pr_only_good = GOOD_WORKFLOW.replace(
            "  push:\n    branches: [main, master]\n", ""
        ).replace("on:\n", "on:\n  pull_request:\n    branches: [main]\n")
        pr_only_secret = SECRET_WORKFLOW.replace(
            "  push:\n    branches: [main]\n", ""
        )
        pr_only_guard = GUARD_WORKFLOW.replace(
            "  push:\n    branches: [main]\n", ""
        ).replace("on:\n", "on:\n  pull_request:\n    branches: [main]\n")
        t = Tree(workflows={
            "team-validation.yml": pr_only_good,
            "secret-validation.yml": pr_only_secret,
            "regression-guard.yml": pr_only_guard,
        })
        try:
            self.assertTrue(any("VACUOUS" in p for p in t.check()))
        finally:
            t.close()


if __name__ == "__main__":
    unittest.main(verbosity=2)
