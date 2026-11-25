"""Temporary-history coverage for committed and local regression guard diffs."""
from __future__ import annotations

import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))
from scripts import regression_check as rc  # noqa: E402


class RegressionDiffCoverageTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.repo = Path(self.tmp.name)
        self.old_cwd = Path.cwd()
        import os
        os.chdir(self.repo)
        self.git("init", "-q")
        self.git("config", "user.email", "fixture@example.invalid")
        self.git("config", "user.name", "Fixture")
        Path("sample.txt").write_text("base\n")
        self.git("add", "sample.txt")
        self.git("commit", "-qm", "fixture base")
        self.base = self.git("rev-parse", "HEAD").strip()
        self.registry = self.repo / "registry.jsonl"
        self.registry.write_text("")
        self.rules = self.repo / "rules"
        self.rules.mkdir()
        (self.rules / "pattern-rules.json").write_text(json.dumps({"rules": [{
            "rule_id": "FIXTURE-001", "name": "benign sentinel fixture",
            "pattern": "DENY_SENTINEL", "forbidden_context": "ALLOW_SENTINEL",
            "file_glob": ["*.txt"],
            "severity": "warning", "enabled": True,
        }]}))

    def tearDown(self):
        import os
        os.chdir(self.old_cwd)
        self.tmp.cleanup()

    def git(self, *args):
        return subprocess.run(["git", *args], check=True, text=True,
                              capture_output=True).stdout

    def check(self, **kwargs):
        return rc.run_regression_check(self.registry, self.rules, **kwargs)

    def test_committed_range_detects_added_sentinel_and_clean_control(self):
        Path("sample.txt").write_text("base\nDENY_SENTINEL\n")
        self.git("add", "sample.txt")
        self.git("commit", "-qm", "benign fixture change")
        head = self.git("rev-parse", "HEAD").strip()
        count, issues = self.check(base=self.base, head=head)
        self.assertEqual(count, 1)
        self.assertEqual(issues[0]["violations"][0]["rule_id"], "FIXTURE-001")
        result = subprocess.run([
            sys.executable, str(ROOT / "scripts/regression_check.py"),
            "--base", self.base, "--head", head,
            "--registry", str(self.registry), "--rules", str(self.rules), "--json",
        ], cwd=self.repo, text=True, capture_output=True, check=True)
        self.assertEqual(json.loads(result.stdout)["issue_count"], 1)

    def test_clean_range_and_invalid_ranges_fail_closed(self):
        Path("other.txt").write_text("unmatched clean control\n")
        self.git("add", "other.txt")
        self.git("commit", "-qm", "clean range control")
        head = self.git("rev-parse", "HEAD").strip()
        self.assertEqual(self.check(base=self.base, head=head), (0, []))
        with self.assertRaisesRegex(ValueError, "empty"):
            self.check(base=head, head=head)
        with self.assertRaisesRegex(ValueError, "unavailable"):
            self.check(base="missing-fixture-revision", head=head)
        with self.assertRaisesRegex(ValueError, "both"):
            rc.get_changed_files(base=self.base)

    def test_pr_range_uses_merge_base_and_excludes_base_only_content(self):
        self.git("checkout", "-qb", "pr-head", self.base)
        Path("pr.txt").write_text("head-only change\n")
        self.git("add", "pr.txt")
        self.git("commit", "-qm", "PR head")
        head = self.git("rev-parse", "HEAD").strip()

        self.git("checkout", "-qb", "advanced-base", self.base)
        Path("base-only.txt").write_text("DENY_SENTINEL\n")
        self.git("add", "base-only.txt")
        self.git("commit", "-qm", "base advanced after fork")
        base = self.git("rev-parse", "HEAD").strip()

        # PR mode scans common ancestor -> head. The marker exists only on the
        # advanced base side and must not contaminate the PR patch.
        self.assertEqual(self.check(base=base, head=head, merge_base=True), (0, []))
        result = subprocess.run([
            sys.executable, str(ROOT / "scripts/regression_check.py"),
            "--base", base, "--head", head, "--merge-base",
            "--registry", str(self.registry), "--rules", str(self.rules), "--json",
        ], cwd=self.repo, text=True, capture_output=True, check=True)
        self.assertEqual(json.loads(result.stdout)["issue_count"], 0)
        with self.assertRaisesRegex(ValueError, "not an ancestor"):
            self.check(base=base, head=head)

    def test_shallow_history_fails_closed(self):
        Path("later.txt").write_text("later\n")
        self.git("add", "later.txt")
        self.git("commit", "-qm", "later")
        head = self.git("rev-parse", "HEAD").strip()
        shallow_tmp = tempfile.TemporaryDirectory()
        shallow_repo = Path(shallow_tmp.name) / "clone"
        subprocess.run(["git", "clone", "--quiet", "--depth=1", "--no-local",
                        f"file://{self.repo}", str(shallow_repo)], check=True,
                       text=True, capture_output=True)
        old_cwd = Path.cwd()
        import os
        os.chdir(shallow_repo)
        try:
            self.assertEqual(self.git("rev-parse", "--is-shallow-repository").strip(), "true")
            with self.assertRaisesRegex(ValueError, "shallow"):
                rc.get_changed_files(base=head, head=head, merge_base=True)
        finally:
            os.chdir(old_cwd)
            shallow_tmp.cleanup()

    def test_pr_with_multiple_best_merge_bases_fails_closed(self):
        self.git("checkout", "-qb", "left", self.base)
        Path("left.txt").write_text("left side\n")
        self.git("add", "left.txt")
        self.git("commit", "-qm", "left")
        left = self.git("rev-parse", "HEAD").strip()
        left_tree = self.git("rev-parse", "HEAD^{tree}").strip()

        self.git("checkout", "-qb", "right", self.base)
        Path("right.txt").write_text("right side\n")
        self.git("add", "right.txt")
        self.git("commit", "-qm", "right")
        right = self.git("rev-parse", "HEAD").strip()
        right_tree = self.git("rev-parse", "HEAD^{tree}").strip()

        # Criss-cross commits with the same two parents have two best common
        # ancestors; neither side may choose one silently.
        merge_left = self.git("commit-tree", left_tree, "-p", left, "-p", right).strip()
        merge_right = self.git("commit-tree", right_tree, "-p", right, "-p", left).strip()
        bases = self.git("merge-base", "--all", merge_left, merge_right).split()
        self.assertEqual(set(bases), {left, right})
        with self.assertRaisesRegex(ValueError, "ambiguous"):
            rc.get_changed_files(base=merge_left, head=merge_right, merge_base=True)

    def test_local_staged_unstaged_and_mixed_content_are_scanned(self):
        Path("sample.txt").write_text("base\nDENY_SENTINEL staged\n")
        self.git("add", "sample.txt")
        self.assertEqual(self.check(unstaged=False)[0], 1)

        self.git("reset", "-q")
        Path("sample.txt").write_text("base\nDENY_SENTINEL unstaged\n")
        self.assertEqual(self.check(staged=False, unstaged=True)[0], 1)

        Path("sample.txt").write_text("base\nDENY_SENTINEL staged\n")
        self.git("add", "sample.txt")
        Path("sample.txt").write_text("base\nDENY_SENTINEL staged\nALLOW_SENTINEL unstaged\n")
        self.assertEqual(self.check(staged=True, unstaged=True)[0], 1)


if __name__ == "__main__":
    unittest.main()
