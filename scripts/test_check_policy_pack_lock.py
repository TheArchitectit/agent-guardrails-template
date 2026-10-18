#!/usr/bin/env python3
"""Red-on-drift self-test for scripts/check_policy_pack_lock.py (Spec 07).

The verifier this replaces had two defects worth pinning as tests:

  1. **Nothing ran it.** `validate-policy-packs.mjs` was invoked by no workflow;
     the repository's own audit says so. A gate outside CI is a comment, so one
     case here asserts the workflow actually invokes this file.
  2. **It passed on an empty set.** Zero packs + `{}` lock printed `lock ok`,
     exit 0. Cases below therefore prove the gate FAILS when discovery finds
     nothing, when the lock cannot be read at all, and when the lock shape is
     unrecognised — not merely when a digest differs.

Run with: python -m pytest scripts/test_check_policy_pack_lock.py -v
Or: python scripts/test_check_policy_pack_lock.py
"""

import json
import os
import shutil
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from scripts import check_policy_pack_lock as gate

REPO = Path(__file__).resolve().parent.parent

TOOLS = ["guardrail_alpha", "guardrail_beta"]


def _pack(pack="core/demo", version="1.0.0", rules=None, **extra):
    if rules is None:
        rules = [
            {"id": "R1", "law": "read first", "enforcement": "advisory",
             "tool": "guardrail_alpha", "on_unknown": "stop"},
        ]
    data = {"pack": pack, "version": version, "rules": rules}
    data.update(extra)
    return data


class _Fixture:
    """Minimal repo: a tools manifest and N packs, locked from whatever parsed.

    The lock is written even when the packs are deliberately invalid: several
    cases below assert that the *gate* rejects a bad pack, so the fixture must
    not raise before the gate gets to look at it.
    """

    def __init__(self, packs=None):
        self.dir = tempfile.mkdtemp()
        self.root = Path(self.dir)
        tools_dir = self.root / "site/src/data"
        tools_dir.mkdir(parents=True)
        (tools_dir / "tools.json").write_text(
            json.dumps({"count": len(TOOLS), "tools": TOOLS, "source": "test"})
        )
        (self.root / "policy-packs/core").mkdir(parents=True)
        if packs is None:
            packs = {"core/demo": _pack()}
        for name, data in packs.items():
            path = self.root / "policy-packs" / f"{name}.pack.json"
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(json.dumps(data, indent=2) + "\n")
        gate.write_lock(self.root, *self._resolve())

    def _resolve(self):
        packs = gate.discover_packs(self.root)
        rows, _violations, digest = gate.resolve(self.root, packs, set(TOOLS))
        return rows, digest

    def relock(self):
        rows, digest = self._resolve()
        gate.write_lock(self.root, rows, digest)

    def clean(self):
        shutil.rmtree(self.dir, ignore_errors=True)

    def run(self, relock=False):
        return gate.run(self.root, relock=relock)


class PolicyPackLockTest(unittest.TestCase):
    def setUp(self):
        self.fx = None

    def tearDown(self):
        if self.fx:
            self.fx.clean()

    def _run(self, packs=None, relock=False):
        self.fx = _Fixture(packs)
        violations = self.fx.run(relock=relock)
        return [v.render() for v in violations]

    # --- the vacuity defects ------------------------------------------------

    def test_zero_packs_fails_instead_of_passing_vacuously(self):
        """The exact bug in validate-policy-packs.mjs: nothing to check == ok."""
        self.fx = _Fixture({})
        (self.fx.root / "policy-packs/LOCK.json").write_text(
            json.dumps({"packs": {}, "effective_digest": "sha256:0"})
        )
        rendered = [v.render() for v in self.fx.run()]
        self.assertTrue(any("zero" in r for r in rendered), rendered)

    def test_missing_lock_record_fails(self):
        self.fx = _Fixture()
        (self.fx.root / "policy-packs/LOCK.json").unlink()
        rendered = [v.render() for v in self.fx.run()]
        self.assertTrue(any("must exist" in r for r in rendered), rendered)

    def test_unrecognised_lock_shape_fails_rather_than_passing(self):
        """The legacy flat {path: {version, sha256}} map must not read as valid."""
        self.fx = _Fixture()
        (self.fx.root / "policy-packs/LOCK.json").write_text(
            json.dumps({"policy-packs/core/demo.pack.json": {"version": "1.0.0",
                                                            "sha256": "dead"}})
        )
        rendered = [v.render() for v in self.fx.run()]
        self.assertTrue(any("'packs' is missing" in r for r in rendered), rendered)

    def test_malformed_lock_json_fails(self):
        self.fx = _Fixture()
        (self.fx.root / "policy-packs/LOCK.json").write_text("{not json")
        rendered = [v.render() for v in self.fx.run()]
        self.assertTrue(any("valid UTF-8 JSON" in r for r in rendered), rendered)

    def test_zero_known_tools_fails(self):
        self.fx = _Fixture()
        (self.fx.root / "site/src/data/tools.json").write_text(
            json.dumps({"count": 0, "tools": [], "source": "test"})
        )
        rendered = [v.render() for v in self.fx.run()]
        self.assertTrue(any("zero known tools" in r for r in rendered), rendered)

    # --- lock drift ---------------------------------------------------------

    def test_clean_tree_passes(self):
        self.assertEqual(self._run(), [])

    def test_pack_byte_change_without_relock_fails(self):
        """Acceptance 2: one byte of pack content changes the digest."""
        self.assertEqual(self._run(), [])
        path = self.fx.root / "policy-packs/core/demo.pack.json"
        data = json.loads(path.read_text())
        data["rules"][0]["law"] = "read first (edited)"
        path.write_text(json.dumps(data, indent=2) + "\n")
        rendered = [v.render() for v in self.fx.run()]
        self.assertTrue(any("digest is" in r for r in rendered), rendered)
        self.assertTrue(any("effective_digest" in r for r in rendered), rendered)

    def test_version_change_without_relock_fails(self):
        self.assertEqual(self._run(), [])
        path = self.fx.root / "policy-packs/core/demo.pack.json"
        data = json.loads(path.read_text())
        data["version"] = "1.0.1"
        path.write_text(json.dumps(data, indent=2) + "\n")
        rendered = [v.render() for v in self.fx.run()]
        self.assertTrue(any("version is" in r for r in rendered), rendered)

    def test_extra_pack_not_in_lock_fails(self):
        self.assertEqual(self._run(), [])
        path = self.fx.root / "policy-packs/core/second.pack.json"
        path.write_text(json.dumps(_pack(pack="core/second"), indent=2) + "\n")
        rendered = [v.render() for v in self.fx.run()]
        self.assertTrue(any("missing entry" in r for r in rendered), rendered)

    def test_stale_lock_entry_fails(self):
        self.assertEqual(self._run(), [])
        lock = json.loads((self.fx.root / "policy-packs/LOCK.json").read_text())
        lock["packs"]["policy-packs/core/gone.pack.json"] = dict(
            lock["packs"]["policy-packs/core/demo.pack.json"]
        )
        (self.fx.root / "policy-packs/LOCK.json").write_text(json.dumps(lock))
        rendered = [v.render() for v in self.fx.run()]
        self.assertTrue(any("stale entry" in r for r in rendered), rendered)

    def test_resolved_at_must_be_utc_iso8601(self):
        self.assertEqual(self._run(), [])
        lock = json.loads((self.fx.root / "policy-packs/LOCK.json").read_text())
        lock["packs"]["policy-packs/core/demo.pack.json"]["resolved_at"] = "yesterday"
        (self.fx.root / "policy-packs/LOCK.json").write_text(json.dumps(lock))
        rendered = [v.render() for v in self.fx.run()]
        self.assertTrue(any("UTC ISO-8601" in r for r in rendered), rendered)

    # --- schema as data (R07-4.3.2) ----------------------------------------

    def test_unknown_critical_field_fails(self):
        rendered = self._run({"core/demo": _pack(script="./deploy.sh")})
        self.assertTrue(any("unknown critical field 'script'" in r for r in rendered), rendered)

    def test_annotation_prefix_is_allowed(self):
        self.assertEqual(self._run({"core/demo": _pack(**{"x-owner": "security"})}), [])

    def test_bad_semver_fails(self):
        rendered = self._run({"core/demo": _pack(version="v1")})
        self.assertTrue(any("must be semver" in r for r in rendered), rendered)

    def test_pack_name_must_match_path(self):
        rendered = self._run({"core/demo": _pack(pack="core/other")})
        self.assertTrue(any("path implies" in r for r in rendered), rendered)

    def test_blocking_rule_requires_evidence(self):
        rule = {"id": "R1", "law": "x", "enforcement": "blocking",
                "tool": "guardrail_alpha", "on_unknown": "stop"}
        rendered = self._run({"core/demo": _pack(rules=[rule])})
        self.assertTrue(any("requires a non-empty 'evidence'" in r for r in rendered), rendered)

    def test_blocking_rule_with_evidence_passes(self):
        rule = {"id": "R1", "law": "x", "enforcement": "blocking",
                "tool": "guardrail_alpha", "on_unknown": "stop", "evidence": "test_x"}
        self.assertEqual(self._run({"core/demo": _pack(rules=[rule])}), [])

    def test_unknown_tool_fails(self):
        rule = {"id": "R1", "law": "x", "enforcement": "advisory",
                "tool": "guardrail_not_a_tool", "on_unknown": "stop"}
        rendered = self._run({"core/demo": _pack(rules=[rule])})
        self.assertTrue(any("is not registered by the server" in r for r in rendered), rendered)

    def test_duplicate_rule_id_fails(self):
        rules = [
            {"id": "R1", "law": "a", "enforcement": "advisory",
             "tool": "guardrail_alpha", "on_unknown": "stop"},
            {"id": "R1", "law": "b", "enforcement": "advisory",
             "tool": "guardrail_beta", "on_unknown": "stop"},
        ]
        rendered = self._run({"core/demo": _pack(rules=rules)})
        self.assertTrue(any("duplicate rule id" in r for r in rendered), rendered)

    def test_on_unknown_must_be_stop(self):
        rule = {"id": "R1", "law": "a", "enforcement": "advisory",
                "tool": "guardrail_alpha", "on_unknown": "continue"}
        rendered = self._run({"core/demo": _pack(rules=[rule])})
        self.assertTrue(any("'on_unknown' must be 'stop'" in r for r in rendered), rendered)

    def test_empty_non_seed_pack_fails(self):
        rendered = self._run({"core/demo": _pack(rules=[])})
        self.assertTrue(any("requires 'seed': true" in r for r in rendered), rendered)

    def test_empty_seed_pack_passes(self):
        self.assertEqual(self._run({"core/demo": _pack(rules=[], seed=True)}), [])

    # --- determinism (R07-4.1.2 / acceptance 1) ----------------------------

    def test_effective_digest_is_reproducible_across_resolutions(self):
        self.fx = _Fixture()
        packs = gate.discover_packs(self.fx.root)
        known = set(TOOLS)
        _, v1, d1 = gate.resolve(self.fx.root, packs, known)
        _, v2, d2 = gate.resolve(self.fx.root, packs, known)
        self.assertEqual(v1, [])
        self.assertEqual(v2, [])
        self.assertEqual(d1, d2)
        # The effective digest is over identity + content only; ``resolved_at`` is
        # metadata, so two resolutions seconds apart must agree exactly.
        self.assertTrue(d1.startswith("sha256:"))

    def test_effective_digest_changes_with_content(self):
        self.fx = _Fixture()
        packs = gate.discover_packs(self.fx.root)
        _, _, before = gate.resolve(self.fx.root, packs, set(TOOLS))
        path = self.fx.root / "policy-packs/core/demo.pack.json"
        data = json.loads(path.read_text())
        data["version"] = "2.0.0"
        path.write_text(json.dumps(data, indent=2) + "\n")
        _, _, after = gate.resolve(self.fx.root, gate.discover_packs(self.fx.root), set(TOOLS))
        self.assertNotEqual(before, after)


class WiringTest(unittest.TestCase):
    """A verifier nothing invokes is the defect this change exists to fix."""

    def test_the_gate_is_invoked_by_a_workflow(self):
        workflows = sorted((REPO / ".github/workflows").glob("*.yml"))
        self.assertTrue(workflows, "no workflows found")
        hits = [
            wf.name for wf in workflows
            if "check_policy_pack_lock.py" in wf.read_text(encoding="utf-8", errors="replace")
        ]
        self.assertTrue(hits, "check_policy_pack_lock.py is not run by any workflow")

    def test_the_real_tree_passes_the_gate(self):
        """Guards against shipping a gate that fails on its own landing."""
        violations = gate.run(REPO)
        self.assertEqual([v.render() for v in violations], [])


if __name__ == "__main__":
    unittest.main(verbosity=2)
