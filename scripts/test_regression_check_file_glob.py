#!/usr/bin/env python3
"""Red-on-drift self-test for the file_glob scope in scripts/regression_check.py.

The defect this locks: every shipped pattern rule declares ``file_glob``, and
this module read it ZERO times. A rule written for ``Dockerfile*`` with the
pattern ``.*`` therefore fired against every changed Markdown, Python and JSON
file - measured on a pristine tree with a one-line prose edit to
CONTRIBUTING.md, which reported 13 "Missing .dockerignore" violations and made
``--pre-commit`` exit 1. The Node scanner has always honoured the field, so the
two scanners disagreed about the same rule set.

Cases:
  * a prose file no longer trips a Dockerfile-scoped rule  (the reported red);
  * a Dockerfile still does                              (not fix-by-disabling);
  * basename and repo-relative matching both work        (file-kind rules);
  * ``*`` does not cross a path separator while ``**`` does;
  * a rule with no file_glob still applies to everything (backwards shape).

Run: python3 scripts/test_regression_check_file_glob.py
 or: python3 -m pytest scripts/test_regression_check_file_glob.py -v
"""
from __future__ import annotations

import os
import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
sys.path.insert(0, str(Path(__file__).resolve().parent.parent))

from scripts import regression_check as rc  # noqa: E402

DOCKER_RULE = {
    "rule_id": "PREVENT-027",
    "rule_type": "pattern",
    "pattern": ".*",
    "file_glob": ["Dockerfile", "Dockerfile.*"],
    "severity": "info",
    "message": "No .dockerignore file found",
}

PY_RULE = {
    "rule_id": "PREVENT-002",
    "rule_type": "pattern",
    "pattern": "eval\\(",
    "file_glob": ["*.py", "*.go", "*.js"],
    "severity": "critical",
    "message": "eval",
}


class FileGlobScopeTests(unittest.TestCase):

    def test_prose_file_does_not_trip_dockerfile_rule(self):
        """The reported red: a Markdown edit fired a Dockerfile-only rule."""
        self.assertFalse(rc.rule_applies_to(DOCKER_RULE, "CONTRIBUTING.md"))
        self.assertFalse(rc.rule_applies_to(DOCKER_RULE, "docs/specs/19.md"))

    def test_dockerfile_still_trips_its_rule(self):
        """Positive control: the fix must not disable the rule it scopes."""
        self.assertTrue(rc.rule_applies_to(DOCKER_RULE, "Dockerfile"))
        self.assertTrue(rc.rule_applies_to(DOCKER_RULE, "mcp-server/deploy/Dockerfile"))

    def test_file_kind_rule_matches_by_basename(self):
        self.assertTrue(rc.rule_applies_to(PY_RULE, "scripts/deep/nested/thing.py"))
        self.assertFalse(rc.rule_applies_to(PY_RULE, "scripts/deep/nested/thing.md"))

    def test_rule_without_glob_applies_everywhere(self):
        rule = dict(PY_RULE)
        rule.pop("file_glob")
        self.assertTrue(rc.rule_applies_to(rule, "anything/at/all.md"))

    def test_single_star_does_not_cross_separator(self):
        rule = dict(PY_RULE)
        rule["file_glob"] = ["src/*.py"]
        self.assertTrue(rc.rule_applies_to(rule, "src/a.py"))
        self.assertFalse(rc.rule_applies_to(rule, "src/deep/a.py"))

    def test_double_star_crosses_separator(self):
        rule = dict(PY_RULE)
        rule["file_glob"] = ["src/**/*.py"]
        self.assertTrue(rc.rule_applies_to(rule, "src/deep/a.py"))
        self.assertTrue(rc.rule_applies_to(rule, "src/a.py"))

    def test_content_violation_is_scoped_by_caller_path(self):
        """End to end through the real matcher: same diff, different files."""
        diff = "+eval('x')\n"
        hits_py = rc.check_diff_against_patterns(diff, [PY_RULE], "src/a.py")
        hits_md = rc.check_diff_against_patterns(diff, [PY_RULE], "docs/a.md")
        self.assertEqual(len(hits_py), 1)
        self.assertEqual(len(hits_md), 0)

    def test_shipped_rules_all_declare_a_glob(self):
        """Non-vacuity: if the rules stopped declaring globs, this fix is moot."""
        import json
        root = Path(__file__).resolve().parent.parent
        data = json.loads(
            (root / ".guardrails/prevention-rules/pattern-rules.json").read_text()
        )
        rules = data.get("rules", [])
        self.assertGreater(len(rules), 0)
        self.assertTrue(all(r.get("file_glob") for r in rules))


if __name__ == "__main__":
    unittest.main(verbosity=2)
