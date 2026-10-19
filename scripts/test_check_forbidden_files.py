#!/usr/bin/env python3
"""Red-on-drift self-test for scripts/check_forbidden_files.py.

The cases are the exact inputs that made the workflow's inline `grep -E` loop
wrong, plus the shapes a matcher must still refuse. Each refusal case is the
requirement's own artifact; each permission case is a path the OLD matcher
refused, so a regression to the previous behaviour turns this suite red.

Run with: python -m pytest scripts/test_check_forbidden_files.py -v
Or:       python scripts/test_check_forbidden_files.py
"""

import os
import sys
import unittest

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from scripts import check_forbidden_files as gate


class Refused(unittest.TestCase):
    """Paths a change set must be refused for."""

    def test_plain_env_file(self):
        self.assertTrue(gate.is_forbidden(".env"))

    def test_nested_env_file(self):
        self.assertTrue(gate.is_forbidden("mcp-server/.env"))

    def test_env_variant_with_values(self):
        # `.env.production` holds values; only the value-free templates are allowed.
        self.assertTrue(gate.is_forbidden("infra/.env.production"))
        self.assertTrue(gate.is_forbidden("deploy/.env.local"))

    def test_envrc(self):
        self.assertTrue(gate.is_forbidden(".envrc"))

    def test_pem_key_anywhere(self):
        self.assertTrue(gate.is_forbidden("certs/server.pem"))
        self.assertTrue(gate.is_forbidden("server.pem"))

    def test_key_suffix(self):
        self.assertTrue(gate.is_forbidden("certs/server.key"))

    def test_keystore_shapes(self):
        for p in ("a.p12", "a.pfx", "a.jks", "a.keystore"):
            with self.subTest(path=p):
                self.assertTrue(gate.is_forbidden(p))

    def test_credential_json_names(self):
        self.assertTrue(gate.is_forbidden("credentials.json"))
        self.assertTrue(gate.is_forbidden("nested/secrets.json"))


class Permitted(unittest.TestCase):
    """Paths the old `grep -E` loop refused, and must not."""

    def test_env_example_is_the_documented_alternative(self):
        # The job's own failure text says "use .env.example instead".
        # grep -E ".env" matched it, so the check refused its own remediation.
        self.assertFalse(gate.is_forbidden(".env.example"))
        self.assertFalse(gate.is_forbidden("mcp-server/.env.example"))

    def test_env_sample(self):
        self.assertFalse(gate.is_forbidden(".env.sample"))

    def test_prose_mentioning_env(self):
        # grep -E ".env" matched any char + "env": docs/.env.example,
        # src/config.env.md, .envrc...
        self.assertFalse(gate.is_forbidden("src/config.env.md"))
        self.assertFalse(gate.is_forbidden("docs/environment.md"))

    def test_ordinary_source_files(self):
        for p in ("main.go", "internal/auth/key.go", "README.md", "keyring.go"):
            with self.subTest(path=p):
                self.assertFalse(gate.is_forbidden(p))

    def test_blank_and_comment_lines_ignored(self):
        self.assertFalse(gate.is_forbidden(""))
        self.assertFalse(gate.is_forbidden("   "))
        self.assertFalse(gate.is_forbidden("# a comment"))


class NonVacuity(unittest.TestCase):
    """The gate must exercise its matcher, not report a clean scan of nothing."""

    def test_finds_violations_in_a_mixed_set(self):
        paths = [
            "README.md",
            "certs/server.pem",
            ".env.example",
            "infra/.env",
            "scripts/run.py",
        ]
        found = gate.find_violations(paths)
        self.assertEqual(sorted(found), ["certs/server.pem", "infra/.env"])

    def test_clean_set_reports_none(self):
        self.assertEqual(gate.find_violations(["a.go", "b.md", ".env.example"]), [])

    def test_case_from_the_real_tree_passes(self):
        # The tracked .env files at main today are exactly the two templates.
        self.assertEqual(
            gate.find_violations([".env.example", "mcp-server/.env.example"]), []
        )


if __name__ == "__main__":
    unittest.main(verbosity=2)
