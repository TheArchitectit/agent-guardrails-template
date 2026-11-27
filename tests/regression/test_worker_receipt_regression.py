"""Regression tests for the signed, SHA-bound worker receipt contract."""
from __future__ import annotations

import base64
import importlib.util
import os
import subprocess
import tempfile
import unittest
from pathlib import Path

CLIENT = Path(__file__).resolve().parents[2] / ".github" / "guardrails-build-client.py"
spec = importlib.util.spec_from_file_location("guardrails_build_client", CLIENT)
assert spec and spec.loader
client = importlib.util.module_from_spec(spec)
spec.loader.exec_module(client)


class WorkerReceiptContract(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="worker-receipt-regression-")
        root = Path(self.temp.name)
        self.private, self.public = root / "worker.key", root / "worker.pub"
        subprocess.run(["openssl", "genpkey", "-algorithm", "ED25519", "-out", str(self.private)], check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        subprocess.run(["openssl", "pkey", "-in", str(self.private), "-pubout", "-out", str(self.public)], check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        self.claims = {"repository": "TheArchitectit/agent-guardrails-template", "repository_id": "12345", "event_name": "push", "ref": "refs/heads/main", "run_id": "9876", "run_attempt": "2", "check_run_id": "6789", "sha": "a" * 40}
        self.env = {"GITHUB_REPOSITORY": self.claims["repository"], "GITHUB_EVENT_NAME": self.claims["event_name"], "GITHUB_SHA": self.claims["sha"], "GITHUB_RUN_ID": self.claims["run_id"], "GITHUB_RUN_ATTEMPT": self.claims["run_attempt"]}
        self.old_env = {key: os.environ.get(key) for key in self.env}
        os.environ.update(self.env)

    def tearDown(self):
        for key, value in self.old_env.items():
            if value is None:
                os.environ.pop(key, None)
            else:
                os.environ[key] = value
        self.temp.cleanup()

    def receipt(self):
        return {"schema": "guardrails-build-receipt/v1", "worker_host": "ai05", "vm_profile": "qemu-kvm-offline", "codex": "not-run", "repository": self.claims["repository"], "repository_id": self.claims["repository_id"], "event_name": self.claims["event_name"], "ref": self.claims["ref"], "run_id": self.claims["run_id"], "run_attempt": self.claims["run_attempt"], "check_run_id": self.claims["check_run_id"], "candidate_sha": self.claims["sha"], "result": {"status": "passed", "image_digest": "sha256:" + "b" * 64, "protocol": {"initialize": True, "tools_list_count": 7, "benign_tool": "guardrail_advisor_list", "benign_result_count": 9}, "cleanup": {"resources_created": ["guardrails-ci-9876-2-ephemeral"], "resources_removed": ["guardrails-ci-9876-2-ephemeral"]}}}

    def signed(self, receipt):
        payload, signature = Path(self.temp.name) / "receipt.json", Path(self.temp.name) / "receipt.sig"
        payload.write_bytes(client.canonical(receipt))
        subprocess.run(["openssl", "pkeyutl", "-sign", "-inkey", str(self.private), "-rawin", "-in", str(payload), "-out", str(signature)], check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        return base64.b64encode(signature.read_bytes()).decode()

    def test_accepts_nonzero_observed_tool_inventory_and_nine_advisors(self):
        receipt = self.receipt()
        client.verify_receipt(receipt, self.signed(receipt), str(self.public), self.claims)

    def test_rejects_wrong_candidate_sha(self):
        receipt = self.receipt(); receipt["candidate_sha"] = "c" * 40
        with self.assertRaisesRegex(RuntimeError, "identity"):
            client.verify_receipt(receipt, self.signed(receipt), str(self.public), self.claims)

    def test_rejects_wrong_execution_environment(self):
        for field, value in (("codex", "invoked"), ("vm_profile", "networked"), ("worker_host", "ucs03")):
            with self.subTest(field=field):
                receipt = self.receipt()
                receipt[field] = value
                with self.assertRaisesRegex(RuntimeError, "attest"):
                    client.verify_receipt(receipt, self.signed(receipt), str(self.public), self.claims)

    def test_rejects_wrong_run_identity(self):
        receipt = self.receipt(); receipt["run_id"] = "9999"
        with self.assertRaisesRegex(RuntimeError, "identity"):
            client.verify_receipt(receipt, self.signed(receipt), str(self.public), self.claims)

    def test_rejects_wrong_receipt_public_key(self):
        receipt = self.receipt(); wrong = Path(self.temp.name) / "wrong.pub"; other = Path(self.temp.name) / "other.key"
        subprocess.run(["openssl", "genpkey", "-algorithm", "ED25519", "-out", str(other)], check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        subprocess.run(["openssl", "pkey", "-in", str(other), "-pubout", "-out", str(wrong)], check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        with self.assertRaisesRegex(RuntimeError, "signature"):
            client.verify_receipt(receipt, self.signed(receipt), str(wrong), self.claims)


if __name__ == "__main__":
    unittest.main()
