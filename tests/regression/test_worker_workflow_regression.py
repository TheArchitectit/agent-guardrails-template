"""Protect the event coverage that dispatches exact-image acceptance to AI05."""
from pathlib import Path
import unittest

import yaml


WORKFLOWS = Path(__file__).resolve().parents[2] / ".github" / "workflows"


def workflow(name):
    return yaml.safe_load((WORKFLOWS / name).read_text())


class WorkerWorkflowCoverage(unittest.TestCase):
    def test_every_branch_tag_and_candidate_update_reaches_the_worker(self):
        caller = workflow("guardrails-build-call.yml")
        events = caller.get("on", caller.get(True))
        self.assertEqual(events["push"], {"branches": ["**"], "tags": ["**"]})
        self.assertEqual(events["pull_request"], {"types": ["opened", "synchronize", "reopened"]})
        self.assertIn("workflow_dispatch", events)

        job = caller["jobs"]["isolated-build"]
        self.assertEqual(
            job["if"],
            "(github.event_name != 'pull_request' || "
            "github.event.pull_request.head.repo.full_name == github.repository) && !github.event.deleted",
        )
        self.assertEqual(job["permissions"], {"contents": "read", "id-token": "write"})
        self.assertRegex(
            job["uses"],
            r"^TheArchitectit/agent-guardrails-template/\.github/workflows/guardrails-build\.yml@[0-9a-f]{40}$",
        )

    def test_manual_build_workflows_share_the_push_gate(self):
        canonical = workflow("guardrails-build-call.yml")["jobs"]["isolated-build"]
        cases = (
            ("team-validation.yml", "guardrails-build-manual", "github.event_name == 'workflow_dispatch'"),
            ("team-validation-fleet.yml", "guardrails-build-trusted-main-manual",
             "github.event_name == 'workflow_dispatch' && github.ref == 'refs/heads/main'"),
        )
        for filename, job_id, condition in cases:
            with self.subTest(workflow=filename):
                job = workflow(filename)["jobs"][job_id]
                self.assertEqual(job["uses"], canonical["uses"])
                self.assertEqual(job["permissions"], canonical["permissions"])
                self.assertEqual(job["if"], condition)
