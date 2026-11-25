"""Temporary-root regressions for the platform installer paths."""

import importlib.util
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch


MODULE_PATH = Path(__file__).resolve().parents[1] / "scripts" / "setup_agents.py"
SPEC = importlib.util.spec_from_file_location("setup_agents", MODULE_PATH)
setup_agents = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(setup_agents)


class SetupAgentsPathTests(unittest.TestCase):
    def test_clone_preserves_hidden_path_and_optional_dot_slash(self):
        for requested in (".claude/skills/guardrails-enforcer.json", "./.claude/skills/guardrails-enforcer.json"):
            with self.subTest(requested=requested), tempfile.TemporaryDirectory() as tmp:
                root = Path(tmp)

                def fake_download(url, target, dry_run):
                    target.parent.mkdir(parents=True, exist_ok=True)
                    target.write_text("fixture", encoding="utf-8")
                    return True

                with patch.object(setup_agents, "get_default_branch", return_value="main"), patch.object(
                    setup_agents, "download_file", side_effect=fake_download
                ):
                    self.assertTrue(setup_agents.clone_skill(requested, str(root), False))
                self.assertEqual((root / ".claude/skills/guardrails-enforcer.json").read_text(), "fixture")

    def test_clone_rejects_traversal_without_writing_outside_root(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp) / "root"
            root.mkdir()
            outside = Path(tmp) / "outside.txt"
            with patch.object(setup_agents, "download_file") as download:
                self.assertFalse(setup_agents.clone_skill("../outside.txt", str(root), False))
            download.assert_not_called()
            self.assertFalse(outside.exists())

    def test_clone_rejects_symlink_escape(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp) / "root"
            outside = Path(tmp) / "outside"
            root.mkdir()
            outside.mkdir()
            try:
                (root / ".claude").symlink_to(outside, target_is_directory=True)
            except (OSError, NotImplementedError):
                self.skipTest("directory symlinks are unavailable")
            with patch.object(setup_agents, "download_file") as download:
                self.assertFalse(setup_agents.clone_skill(".claude/skills/file.json", str(root), False))
            download.assert_not_called()
            self.assertEqual(list(outside.iterdir()), [])

    def test_named_skill_rejects_symlink_escape(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp) / "root"
            outside = Path(tmp) / "outside"
            root.mkdir()
            outside.mkdir()
            try:
                (root / ".claude").symlink_to(outside, target_is_directory=True)
            except (OSError, NotImplementedError):
                self.skipTest("directory symlinks are unavailable")
            self.assertFalse(setup_agents.install_skill("guardrails-enforcer", str(root), "copy", False))
            self.assertEqual(list(outside.iterdir()), [])

    def test_single_windsurf_skill_is_root_file(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.assertTrue(setup_agents.install_skill("windsurf-rules", str(root), "copy", False))
            target = root / ".windsurfrules"
            self.assertTrue(target.is_file())
            self.assertFalse((target / ".windsurfrules").exists())
            self.assertEqual(target.read_bytes(), (setup_agents.REPO_ROOT / ".windsurfrules").read_bytes())

    def test_directory_skill_and_bulk_windsurf_paths_remain_unchanged(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.assertTrue(setup_agents.install_skill("guardrails-enforcer", str(root), "copy", False))
            self.assertTrue((root / ".claude/skills/guardrails-enforcer.json").is_file())
            bulk_root = root / "bulk"
            self.assertTrue(setup_agents.install_platform(str(bulk_root), "windsurf", "copy", False))
            self.assertTrue((bulk_root / ".windsurfrules").is_file())

    def test_named_and_bulk_symlink_installs_are_idempotent(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.assertTrue(setup_agents.install_skill("guardrails-enforcer", str(root), "symlink", False))
            skill = root / ".claude/skills/guardrails-enforcer.json"
            self.assertTrue(skill.is_symlink())
            self.assertTrue(setup_agents.install_skill("guardrails-enforcer", str(root), "symlink", False))
            self.assertTrue(skill.is_symlink())

            bulk_root = root / "bulk"
            self.assertTrue(setup_agents.install_platform(str(bulk_root), "windsurf", "symlink", False))
            windsurf = bulk_root / ".windsurfrules"
            self.assertTrue(windsurf.is_symlink())
            self.assertTrue(setup_agents.install_platform(str(bulk_root), "windsurf", "symlink", False))
            self.assertTrue(windsurf.is_symlink())


if __name__ == "__main__":
    unittest.main()
