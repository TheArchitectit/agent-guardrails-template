"""Regression for TPL-53. A Service without ports passed the routing preflight."""

from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

import yaml


ROOT = Path(__file__).resolve().parents[2]
SCRIPT = ROOT / "scripts/render_k8s_profile.py"
TEMPLATE = ROOT / "mcp-server/deploy/k8s-deployment.yaml"
PROFILE = ROOT / "mcp-server/deploy/profiles/process-local-single-instance.yaml"


class ProcessLocalRoutingRegression(unittest.TestCase):
    def test_unroutable_service_preserves_previous_output(self):
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary)
            output = directory / "rendered.yaml"
            command = [sys.executable, str(SCRIPT), "--profile", str(PROFILE)]
            subprocess.run(command + ["--template", str(TEMPLATE), "--output", str(output)],
                           check=True, capture_output=True)
            previous = output.read_bytes()
            for ports in (None, [], {}):
                with self.subTest(ports=ports):
                    documents = list(yaml.safe_load_all(previous))
                    service = next(doc for doc in documents if doc["kind"] == "Service")
                    if ports is None:
                        del service["spec"]["ports"]
                    else:
                        service["spec"]["ports"] = ports
                    invalid = directory / "unroutable.yaml"
                    invalid.write_text(yaml.safe_dump_all(documents), encoding="utf-8")
                    for arguments in (["--validate", str(invalid)],
                                      ["--template", str(invalid), "--output", str(output)]):
                        result = subprocess.run(command + arguments, capture_output=True, text=True)
                        self.assertEqual(result.returncode, 2, result.stdout + result.stderr)
                        self.assertIn("Service ports", result.stderr)
                        self.assertEqual(output.read_bytes(), previous)


if __name__ == "__main__":
    unittest.main()
