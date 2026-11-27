"""Fixture-only checks for the Kubernetes session-topology preflight."""

from copy import deepcopy
from pathlib import Path
import sys
import unittest

import yaml

sys.path.insert(0, str(Path(__file__).parents[1] / "scripts"))
from render_k8s_profile import PreflightError, preflight, render, validate_serialized  # noqa: E402


ROOT = Path(__file__).parents[1]
TEMPLATE = ROOT / "mcp-server/deploy/k8s-deployment.yaml"
PROFILE = ROOT / "mcp-server/deploy/profiles/process-local-single-instance.yaml"


class ProcessLocalProfileTests(unittest.TestCase):
    def setUp(self):
        self.documents = render(TEMPLATE, PROFILE, {"VERSION": "fixture", "BUILD_TIME": "fixture", "GIT_COMMIT": "fixture"})
        self.profile = yaml.safe_load(PROFILE.read_text(encoding="utf-8"))

    def test_actual_rendered_objects_are_qualified(self):
        preflight(self.documents, self.profile)
        deployment = next(d for d in self.documents if d["kind"] == "Deployment")
        service = next(d for d in self.documents if d["kind"] == "Service")
        hpa = next(d for d in self.documents if d["kind"] == "HorizontalPodAutoscaler")
        self.assertEqual(deployment["spec"]["replicas"], 1)
        self.assertEqual(deployment["spec"]["strategy"], {"type": "Recreate"})
        self.assertEqual(service["spec"]["type"], "ClusterIP")
        self.assertEqual(hpa["spec"]["maxReplicas"], 1)
        self.assertNotIn("MCP_SESSION_MODE", [item["name"] for item in deployment["spec"]["template"]["spec"]["containers"][0]["env"]])

    def assert_rejected(self, mutate, message):
        documents = deepcopy(self.documents)
        mutate(documents)
        with self.assertRaisesRegex(PreflightError, message):
            preflight(documents, self.profile)

    def test_rejects_more_than_one_replica(self):
        self.assert_rejected(lambda d: next(x for x in d if x["kind"] == "Deployment")["spec"].update(replicas=2), "replicas=1")

    def test_rejects_boolean_replica(self):
        self.assert_rejected(lambda d: next(x for x in d if x["kind"] == "Deployment")["spec"].update(replicas=True), "replicas=1")

    def test_rejects_hpa_max_greater_than_one(self):
        self.assert_rejected(lambda d: next(x for x in d if x["kind"] == "HorizontalPodAutoscaler")["spec"].update(maxReplicas=2), "minReplicas=maxReplicas=1")

    def test_rejects_overlapping_rollout(self):
        def mutate(documents):
            next(x for x in documents if x["kind"] == "Deployment")["spec"]["strategy"] = {"type": "RollingUpdate", "rollingUpdate": {"maxSurge": 1}}
        self.assert_rejected(mutate, "Recreate")

    def test_rejects_inconsistent_session_mode(self):
        def mutate(documents):
            deployment = next(x for x in documents if x["kind"] == "Deployment")
            deployment["spec"]["template"]["metadata"]["annotations"]["guardrails.dev/session-mode"] = "stateless"
        self.assert_rejected(mutate, "session mode")

    def test_serialized_validation_reads_actual_wire_yaml(self):
        path = ROOT / "tests/.tmp-process-local.yaml"
        try:
            path.write_text("---\n" + yaml.safe_dump_all(self.documents, sort_keys=False), encoding="utf-8")
            validate_serialized(path, PROFILE)
        finally:
            path.unlink(missing_ok=True)


if __name__ == "__main__":
    unittest.main()
