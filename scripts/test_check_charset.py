#!/usr/bin/env python3
"""Red-on-drift self-test for scripts/check_charset.py (fw-charset-01).

Every case drives the REAL gate as a subprocess against a REAL git tree, so a
green result cannot come from a helper the production path never calls.

The cases exist because each one is a way this class of gate goes quiet:
  * the reported defect (U+00A7) must FAIL;
  * a strict-zero path must FAIL even though a baseline row would cover it;
  * the substitution artifact (the section marker written twice in a row)
    must FAIL in a pure-ASCII
    file - the exact defect a codepoint check is blind to;
  * an artifact must NOT be grandfathered when its file is baselined;
  * a missing allowlist, a row without a reason, or a codepoint both allowed
    and denied each REFUSE TO RUN (exit 2) rather than banning nothing;
  * a zero-file scan FAILS (no vacuous green);
  * allowlisted typography (box drawing) still passes, so the gate is not
    fix-by-banning-everything.

Run: python3 scripts/test_check_charset.py
 or: python3 -m pytest scripts/test_check_charset.py -v
"""
from __future__ import annotations

import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

SCRIPT = Path(__file__).resolve().parent / "check_charset.py"
PYTHON = sys.executable

POLICY = {
    "api_version": "guardrails.charset/v1",
    "deny": [
        {"codepoint": "0x00A7", "reason": "section sign"},
        {"codepoint": "0xFE0F", "reason": "variation selector"},
        {"range": ["0x1F000", "0x1FAFF"], "reason": "emoji"},
    ],
    "strict_zero": [
        {"path": "specs", "reason": "published corpus"},
    ],
    "allow": [
        {"range": ["0x2500", "0x257F"], "reason": "box drawing"},
        {"codepoint": "0x2014", "reason": "em dash"},
    ],
}

SEC_SIGN = "\u00a7"
# Built from parts on purpose: writing the degenerate form literally would
# make this test file the gate's own first hit (fw-charset-01).
ARTIFACT = "Sec." + " " + "Sec." + " 3, 5"
EMOJI = "\U0001F534"
BOX = "\u2500"


def make_repo(tmp: Path, files: dict[str, str], policy=POLICY,
              baseline: dict[str, int] | None = None) -> Path:
    subprocess.run(["git", "init", "-q", str(tmp)], check=True)
    subprocess.run(["git", "-C", str(tmp), "config", "user.email", "t@t"],
                   check=True)
    subprocess.run(["git", "-C", str(tmp), "config", "user.name", "t"],
                   check=True)
    gd = tmp / ".guardrails"
    gd.mkdir(exist_ok=True)
    if policy is not None:
        (gd / "charset-allowlist.json").write_text(
            json.dumps(policy), encoding="utf-8")
    if baseline is not None:
        (gd / "charset-baseline.json").write_text(
            json.dumps({"api_version": "guardrails.charset-baseline/v1",
                        "files": baseline}), encoding="utf-8")
    for rel, content in files.items():
        p = tmp / rel
        p.parent.mkdir(parents=True, exist_ok=True)
        p.write_text(content, encoding="utf-8")
    subprocess.run(["git", "-C", str(tmp), "add", "-A"], check=True)
    return tmp


def run(root: Path, *args: str):
    return subprocess.run([PYTHON, str(SCRIPT), "--repo", str(root), *args],
                          capture_output=True, text=True)


class CharsetGateTests(unittest.TestCase):

    def test_section_sign_fails(self):
        with tempfile.TemporaryDirectory() as d:
            root = make_repo(Path(d), {"specs/08-x.md": f"see {SEC_SIGN}3.2\n"})
            r = run(root)
            self.assertEqual(r.returncode, 1, r.stdout)
            self.assertIn("U+00A7", r.stdout)

    def test_ascii_section_reference_passes(self):
        with tempfile.TemporaryDirectory() as d:
            root = make_repo(Path(d), {"specs/08-x.md": "see Sec. 3.2\n"})
            self.assertEqual(run(root).returncode, 0)

    def test_emoji_label_fails(self):
        with tempfile.TemporaryDirectory() as d:
            root = make_repo(Path(d), {"specs/08-x.md": f"Priority: {EMOJI}\n"})
            r = run(root)
            self.assertEqual(r.returncode, 1)
            self.assertIn("U+1F534", r.stdout)

    def test_allowlisted_box_drawing_passes(self):
        with tempfile.TemporaryDirectory() as d:
            root = make_repo(Path(d), {"specs/08-x.md": f"a {BOX} b\n"})
            self.assertEqual(run(root).returncode, 0)

    def test_strict_zero_scope_defeats_baseline(self):
        """A baseline row must not exempt the corpus the standard protects."""
        with tempfile.TemporaryDirectory() as d:
            root = make_repo(
                Path(d),
                {"specs/08-x.md": f"see {SEC_SIGN}3.2\n"},
                baseline={"specs/08-x.md": 10},
            )
            r = run(root)
            self.assertEqual(r.returncode, 1, r.stdout)
            self.assertIn("strict-zero", r.stdout)

    def test_baseline_still_covers_non_strict_path(self):
        """The ratchet is real elsewhere: legacy debt outside strict scopes."""
        with tempfile.TemporaryDirectory() as d:
            root = make_repo(
                Path(d),
                {"docs/legacy.md": f"old {SEC_SIGN}1\n"},
                baseline={"docs/legacy.md": 1},
            )
            self.assertEqual(run(root).returncode, 0, run(root).stdout)

    def test_baseline_blocks_new_debt(self):
        with tempfile.TemporaryDirectory() as d:
            root = make_repo(
                Path(d),
                {"docs/legacy.md": f"old {SEC_SIGN}1 {SEC_SIGN}2\n"},
                baseline={"docs/legacy.md": 1},
            )
            self.assertEqual(run(root).returncode, 1)

    def test_substitution_artifact_fails_in_ascii_file(self):
        """The core post-condition: pure ASCII, still a defect."""
        with tempfile.TemporaryDirectory() as d:
            root = make_repo(Path(d), {"specs/08-x.md":
                                       "applies to " + ARTIFACT + "\n"})
            r = run(root)
            self.assertEqual(r.returncode, 1, r.stdout)
            self.assertIn("substitution", r.stdout.lower())

    def test_substitution_artifact_is_not_grandfathered(self):
        with tempfile.TemporaryDirectory() as d:
            root = make_repo(
                Path(d),
                {"docs/legacy.md": "applies to " + ARTIFACT + "\n"},
                baseline={"docs/legacy.md": 5},
            )
            self.assertEqual(run(root).returncode, 1)

    def test_doubled_ellipsis_fails(self):
        with tempfile.TemporaryDirectory() as d:
            # Built from parts so this file is not the gate's own hit.
            root = make_repo(Path(d), {"docs/x.md": "wait" + "." * 6 + "\n"})
            self.assertEqual(run(root).returncode, 1)

    def test_sec_without_space_fails(self):
        """Second degenerate output of the same substitution.

        Replacing the marker where the source wrote it against a digit
        produced a token with no separating space. The gate prescribes
        "Sec.", so it must also reject the malformed form its own
        prescription can create.
        """
        with tempfile.TemporaryDirectory() as d:
            bad = "Sec." + "6" + " Implementation status\n"
            root = make_repo(Path(d), {"specs/08-x.md": bad})
            r = run(root)
            self.assertEqual(r.returncode, 1, r.stdout + r.stderr)
            self.assertIn("substitution", (r.stdout + r.stderr).lower())

    def test_missing_allowlist_refuses(self):
        with tempfile.TemporaryDirectory() as d:
            root = make_repo(Path(d), {"docs/x.md": "plain\n"}, policy=None)
            r = run(root)
            self.assertEqual(r.returncode, 2, r.stdout)

    def test_row_without_reason_refuses(self):
        with tempfile.TemporaryDirectory() as d:
            bad = dict(POLICY)
            bad["allow"] = [{"codepoint": "0x2014"}]
            root = make_repo(Path(d), {"docs/x.md": "plain\n"}, policy=bad)
            self.assertEqual(run(root).returncode, 2)

    def test_codepoint_both_allowed_and_denied_refuses(self):
        with tempfile.TemporaryDirectory() as d:
            bad = dict(POLICY)
            bad["allow"] = [{"codepoint": "0x00A7", "reason": "oops"}]
            root = make_repo(Path(d), {"docs/x.md": "plain\n"}, policy=bad)
            self.assertEqual(run(root).returncode, 2)

    def test_zero_text_files_fails(self):
        with tempfile.TemporaryDirectory() as d:
            root = make_repo(Path(d), {"data.bin": "x"})
            r = run(root)
            self.assertEqual(r.returncode, 1)
            # The refusal is a gate error, so it is written to stderr; assert
            # on the combined streams rather than pinning the wrong one.
            self.assertIn("zero text files", (r.stdout + r.stderr).lower())

    def test_ascii_only_passes(self):
        with tempfile.TemporaryDirectory() as d:
            root = make_repo(Path(d), {"docs/x.md": "plain ascii\n"})
            self.assertEqual(run(root).returncode, 0)


if __name__ == "__main__":
    unittest.main(verbosity=2)
