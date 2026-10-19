#!/usr/bin/env python3
"""Red-on-drift self-test for scripts/runner_isolation_probe.py (Spec 19 R19-13).

The probe is the clause-5 evidence: "a runner label alone is not isolation
evidence". Its value is entirely in refusing a workspace that is NOT isolated, so
each case here builds a real git tree and measures the probe's verdict against
it — a workspace carrying another commit, a sibling repository, a cross-repo
runner work root, an over-permissive mode, and a leftover `.env`.

Both directions are proved for every property: the clean shape passes, and the
violating shape fails. A probe that only ever printed OK would pass a
one-directional test while providing no evidence at all.

Run with: python -m pytest scripts/test_runner_isolation_probe.py -v
Or: python scripts/test_runner_isolation_probe.py
"""

import os
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from scripts import runner_isolation_probe as probe


def _git_repo(path: Path) -> str:
    path.mkdir(parents=True, exist_ok=True)
    subprocess.run(["git", "init", "-q", str(path)], check=True)
    subprocess.run(
        ["git", "-C", str(path), "-c", "user.email=t@example.invalid",
         "-c", "user.name=test", "commit", "-q", "--allow-empty", "-m", "init"],
        check=True,
    )
    return subprocess.run(
        ["git", "-C", str(path), "rev-parse", "HEAD"],
        capture_output=True, text=True, check=True,
    ).stdout.strip()


class CleanWorkspace(unittest.TestCase):
    def test_isolated_workspace_passes(self):
        with tempfile.TemporaryDirectory() as d:
            ws = Path(d) / "only" / "repo"
            sha = _git_repo(ws)
            os.chmod(ws, 0o700)
            record = probe.probe(ws, sha, "TheArchitectit/agent-guardrails-template")
            self.assertEqual(record["violations"], [])
            self.assertTrue(record["clean_checkout"])
            self.assertEqual(record["sibling_repositories"], [])
            self.assertEqual(record["cross_repository_checkouts"], [])

    def test_unverifiable_sha_still_measures(self):
        """No expected SHA means the commit check is skipped, not passed."""
        with tempfile.TemporaryDirectory() as d:
            ws = Path(d) / "only" / "repo"
            _git_repo(ws)
            os.chmod(ws, 0o700)
            record = probe.probe(ws, None, "x")
            self.assertEqual(record["violations"], [])
            self.assertTrue(record["clean_checkout"])


class DirtyWorkspace(unittest.TestCase):
    def test_wrong_commit_fails(self):
        with tempfile.TemporaryDirectory() as d:
            ws = Path(d) / "only" / "repo"
            sha = _git_repo(ws)
            os.chmod(ws, 0o700)
            record = probe.probe(ws, "0" * 40, "x")
            self.assertFalse(record["clean_checkout"])
            self.assertTrue(
                any("carries another commit's tree" in v for v in record["violations"]),
                record["violations"],
            )

    def test_sibling_repository_fails(self):
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            ws = root / "agent-guardrails-template"
            sha = _git_repo(ws)
            _git_repo(root / "other-project")
            os.chmod(ws, 0o700)
            record = probe.probe(ws, sha, "x")
            self.assertEqual(record["sibling_repositories"], ["other-project"])
            self.assertTrue(any("shared across repositories" in v for v in record["violations"]))

    def test_cross_repository_runner_root_fails(self):
        """The Actions layout `_work/<repo>/<repo>` hides siblings one level up."""
        with tempfile.TemporaryDirectory() as d:
            work = Path(d) / "_work"
            ws = work / "agent-guardrails-template" / "agent-guardrails-template"
            sha = _git_repo(ws)
            other = work / "openagentplatform" / "openagentplatform"
            _git_repo(other)
            os.chmod(ws, 0o700)
            record = probe.probe(ws, sha, "x")
            self.assertEqual(record["cross_repository_checkouts"], ["openagentplatform"])
            self.assertTrue(any("not repo-scoped" in v for v in record["violations"]))

    def test_repo_scoped_runner_root_passes(self):
        with tempfile.TemporaryDirectory() as d:
            work = Path(d) / "_work"
            ws = work / "agent-guardrails-template" / "agent-guardrails-template"
            sha = _git_repo(ws)
            os.chmod(ws, 0o700)
            record = probe.probe(ws, sha, "x")
            self.assertEqual(record["cross_repository_checkouts"], [])
            self.assertEqual(record["violations"], [])

    def test_group_writable_workspace_fails(self):
        with tempfile.TemporaryDirectory() as d:
            ws = Path(d) / "only" / "repo"
            sha = _git_repo(ws)
            os.chmod(ws, 0o770)
            record = probe.probe(ws, sha, "x")
            self.assertTrue(any("group/world writable" in v for v in record["violations"]))

    def test_env_file_with_values_fails(self):
        with tempfile.TemporaryDirectory() as d:
            ws = Path(d) / "only" / "repo"
            sha = _git_repo(ws)
            os.chmod(ws, 0o700)
            (ws / ".env").write_text("MCP_API_KEY=placeholder-value\n# comment\n", encoding="utf-8")
            record = probe.probe(ws, sha, "x")
            self.assertEqual(len(record["env_files"]), 1)
            self.assertTrue(any("credential material" in v for v in record["violations"]))

    def test_empty_env_file_passes(self):
        with tempfile.TemporaryDirectory() as d:
            ws = Path(d) / "only" / "repo"
            sha = _git_repo(ws)
            os.chmod(ws, 0o700)
            (ws / ".env").write_text("MCP_API_KEY=\n# only comments\n", encoding="utf-8")
            self.assertEqual(probe.probe(ws, sha, "x")["violations"], [])

    def test_not_a_checkout_fails(self):
        with tempfile.TemporaryDirectory() as d:
            ws = Path(d) / "plain"
            ws.mkdir()
            os.chmod(ws, 0o700)
            record = probe.probe(ws, "0" * 40, "x")
            self.assertFalse(record["clean_checkout"])
            self.assertTrue(any("not a readable git checkout" in v for v in record["violations"]))

    def test_missing_workspace_fails(self):
        with tempfile.TemporaryDirectory() as d:
            record = probe.probe(Path(d) / "absent", "0" * 40, "x")
            self.assertTrue(any("does not exist" in v for v in record["violations"]))


class ExitCodes(unittest.TestCase):
    def test_clean_run_exits_zero_and_writes_record(self):
        with tempfile.TemporaryDirectory() as d:
            ws = Path(d) / "only" / "repo"
            sha = _git_repo(ws)
            os.chmod(ws, 0o700)
            out = Path(d) / "attestation.json"
            rc = probe.main(
                ["--workspace", str(ws), "--sha", sha, "--repo", "x", "--out", str(out)]
            )
            self.assertEqual(rc, probe.EXIT_OK)
            self.assertTrue(out.is_file())

    def test_dirty_run_exits_one_and_still_writes_record(self):
        with tempfile.TemporaryDirectory() as d:
            ws = Path(d) / "only" / "repo"
            _git_repo(ws)
            os.chmod(ws, 0o770)
            out = Path(d) / "attestation.json"
            rc = probe.main(
                ["--workspace", str(ws), "--sha", "", "--repo", "x", "--out", str(out)]
            )
            self.assertEqual(rc, probe.EXIT_VIOLATIONS)
            self.assertTrue(out.is_file(), "the job must still get the record to inspect")

    def test_unmeasurable_workspace_exits_two(self):
        with tempfile.TemporaryDirectory() as d:
            out = Path(d) / "attestation.json"
            rc = probe.main(
                ["--workspace", str(Path(d) / "absent"), "--repo", "x", "--out", str(out)]
            )
            self.assertEqual(rc, probe.EXIT_UNUSABLE)


if __name__ == "__main__":
    unittest.main(verbosity=2)
