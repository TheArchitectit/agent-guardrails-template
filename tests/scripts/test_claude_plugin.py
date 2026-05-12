"""Tests for Claude Code plugin compatibility."""

import json
from pathlib import Path

import pytest


class TestClaudePluginManifest:
    def test_manifest_exists(self):
        manifest_path = Path(".claude-plugin/plugin.json")
        assert manifest_path.exists(), "Plugin manifest must exist"

    def test_manifest_has_required_fields(self):
        manifest = json.loads(Path(".claude-plugin/plugin.json").read_text())
        assert "name" in manifest, "Manifest must have 'name'"
        assert "description" in manifest, "Manifest must have 'description'"
        assert "version" in manifest, "Manifest must have 'version'"
        assert "author" in manifest, "Manifest must have 'author'"

    def test_manifest_no_nonstandard_fields(self):
        manifest = json.loads(Path(".claude-plugin/plugin.json").read_text())
        # These fields are not part of the official Anthropic schema
        nonstandard = ["displayName", "skills", "agents", "commands", "hooks"]
        for field in nonstandard:
            assert field not in manifest, f"Non-standard field '{field}' should not be in manifest"

    def test_manifest_valid_json(self):
        text = Path(".claude-plugin/plugin.json").read_text()
        manifest = json.loads(text)
        assert isinstance(manifest, dict)

    def test_version_is_semver(self):
        manifest = json.loads(Path(".claude-plugin/plugin.json").read_text())
        version = manifest["version"]
        parts = version.split(".")
        assert len(parts) == 3, "Version should be semver (x.y.z)"
        assert all(p.isdigit() for p in parts), "Version parts should be numeric"


class TestSkillStructure:
    def test_all_skills_have_skill_md(self):
        skills_dir = Path("skills")
        skill_dirs = [d for d in skills_dir.iterdir() if d.is_dir() and d.name != "shared-prompts"]
        for skill_dir in skill_dirs:
            skill_file = skill_dir / "SKILL.md"
            assert skill_file.exists(), f"Skill {skill_dir.name} must have SKILL.md"

    def test_all_skills_have_description(self):
        import yaml

        skills_dir = Path("skills")
        skill_dirs = [d for d in skills_dir.iterdir() if d.is_dir() and d.name != "shared-prompts"]
        for skill_dir in skill_dirs:
            skill_file = skill_dir / "SKILL.md"
            content = skill_file.read_text()
            assert content.startswith("---"), f"Skill {skill_dir.name} must have YAML frontmatter"
            parts = content.split("---", 2)
            frontmatter = yaml.safe_load(parts[1])
            assert frontmatter.get("description"), f"Skill {skill_dir.name} must have 'description' in frontmatter"
