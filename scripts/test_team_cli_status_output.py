"""Regression coverage for text and JSON status output."""

import json
import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[1]
BACKEND = REPO_ROOT / "scripts" / "team_manager.py"
FIXTURE = REPO_ROOT / ".teams" / "fixtures" / "full-project.json"


class StatusOutputTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.root = Path(self.temp.name)
        (self.root / ".teams").mkdir()
        shutil.copyfile(FIXTURE, self.root / ".teams" / "full-project.json")

    def tearDown(self):
        self.temp.cleanup()

    def run_status(self, *args):
        return subprocess.run(
            [sys.executable, str(BACKEND), "--project", "full-project", "status", *args],
            cwd=self.root,
            text=True,
            capture_output=True,
            check=True,
        )

    def test_selected_phase_json(self):
        result = self.run_status("--phase", "Phase 1: Strategy, Governance & Planning", "--format", "json")
        payload = json.loads(result.stdout)
        self.assertEqual(payload["phase"], "Phase 1: Strategy, Governance & Planning")
        self.assertEqual(payload["total_teams"], 3)
        self.assertIn("progress_pct", payload)

    def test_all_phases_json(self):
        result = self.run_status("--format", "json")
        payload = json.loads(result.stdout)
        self.assertEqual(len(payload), 5)
        self.assertEqual(payload[0]["phase"], "Phase 1: Strategy, Governance & Planning")
        self.assertTrue(all("total_teams" in phase for phase in payload))

    def test_selected_phase_text_unchanged(self):
        result = self.run_status("--phase", "Phase 1: Strategy, Governance & Planning")
        self.assertIn("\nPhase 1: Strategy, Governance & Planning\n", result.stdout)
        self.assertIn("Progress:", result.stdout)
        self.assertNotEqual(result.stdout[0], "{")

    def test_all_phases_text_unchanged(self):
        result = self.run_status()
        self.assertIn("Phase 1: Strategy, Governance & Planning:", result.stdout)
        self.assertIn("Phase 5: Delivery & Sustainment:", result.stdout)
        self.assertNotEqual(result.stdout[0], "[")


if __name__ == "__main__":
    unittest.main()
