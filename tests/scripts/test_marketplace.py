"""Tests for scripts/marketplace.py."""

import json
import sys
from pathlib import Path
from unittest.mock import MagicMock, patch

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parent.parent.parent / "scripts"))

from marketplace import (
    MarketplaceClient,
    MarketplaceEntry,
    MarketplaceInstaller,
    MarketplaceRegistry,
    MarketplaceSkill,
    _parse_repo_spec,
)


class TestParseRepoSpec:
    def test_owner_repo(self):
        assert _parse_repo_spec("TheArchitectit/agent-guardrails-template") == (
            "TheArchitectit",
            "agent-guardrails-template",
        )

    def test_full_url(self):
        assert _parse_repo_spec(
            "https://github.com/TheArchitectit/agent-guardrails-template"
        ) == ("TheArchitectit", "agent-guardrails-template")

    def test_invalid(self):
        with pytest.raises(ValueError):
            _parse_repo_spec("invalid")


class TestMarketplaceRegistry:
    def test_ensure_default_creates_entry(self, tmp_path):
        registry = MarketplaceRegistry(path=tmp_path / "marketplaces.json")
        registry.ensure_default()
        entries = registry.entries()
        assert len(entries) == 1
        assert entries[0].id == "official"

    def test_add_and_list(self, tmp_path):
        registry = MarketplaceRegistry(path=tmp_path / "marketplaces.json")
        entry = MarketplaceEntry(
            id="custom",
            name="custom-marketplace",
            url="https://github.com/custom/repo",
            marketplace_json_url="https://example.com/marketplace.json",
            added_at="2026-01-01T00:00:00Z",
        )
        assert registry.add(entry) is True
        assert len(registry.entries()) == 1
        # Duplicate add should fail
        assert registry.add(entry) is False

    def test_remove(self, tmp_path):
        registry = MarketplaceRegistry(path=tmp_path / "marketplaces.json")
        entry = MarketplaceEntry(
            id="custom",
            name="custom",
            url="https://github.com/custom/repo",
            marketplace_json_url="https://example.com/marketplace.json",
            added_at="2026-01-01T00:00:00Z",
        )
        registry.add(entry)
        assert registry.remove("custom") is True
        assert registry.remove("missing") is False


class TestMarketplaceClient:
    def test_parse_skills(self):
        data = {
            "skills": [
                {
                    "id": "test-skill",
                    "name": "Test Skill",
                    "description": "A test skill",
                    "version": "1.0.0",
                    "applies_to": ["claude", "cursor"],
                    "source": {
                        "owner": "test",
                        "repo": "repo",
                        "path": "skills/test/SKILL.md",
                        "branch": "main",
                    },
                }
            ]
        }
        skills = MarketplaceClient.parse_skills(data)
        assert len(skills) == 1
        s = skills[0]
        assert s.id == "test-skill"
        assert s.name == "Test Skill"
        assert s.applies_to == ["claude", "cursor"]
        assert s.source_owner == "test"
        assert s.source_branch == "main"


class TestMarketplaceInstaller:
    def test_unknown_platform(self, tmp_path):
        installer = MarketplaceInstaller(tmp_path)
        skill = MarketplaceSkill(
            id="test",
            name="Test",
            description="desc",
            version="1.0.0",
            applies_to=["claude"],
        )
        assert installer.install(skill, "unknown", False) is False

    def test_unsupported_platform(self, tmp_path):
        installer = MarketplaceInstaller(tmp_path)
        skill = MarketplaceSkill(
            id="test",
            name="Test",
            description="desc",
            version="1.0.0",
            applies_to=["cursor"],
        )
        assert installer.install(skill, "claude", False) is False

    @patch("marketplace._fetch_text")
    def test_install_claude(self, mock_fetch, tmp_path):
        mock_fetch.return_value = (
            "---\n"
            "id: test-skill\n"
            "name: Test Skill\n"
            "description: A test skill\n"
            "version: 1.0.0\n"
            "applies_to: [claude]\n"
            "---\n\n"
            "# Test Skill\n\nBody here.\n"
        )
        installer = MarketplaceInstaller(tmp_path)
        skill = MarketplaceSkill(
            id="test-skill",
            name="Test Skill",
            description="A test skill",
            version="1.0.0",
            applies_to=["claude"],
            source_owner="test",
            source_repo="repo",
            source_path="skills/test/SKILL.md",
        )
        assert installer.install(skill, "claude", False) is True
        output = tmp_path / ".claude" / "skills" / "test-skill.json"
        assert output.exists()
        data = json.loads(output.read_text())
        assert data["name"] == "test-skill"
        assert data["description"] == "A test skill"

    @patch("marketplace._fetch_text")
    def test_install_cursor(self, mock_fetch, tmp_path):
        mock_fetch.return_value = (
            "---\n"
            "id: test-skill\n"
            "name: Test Skill\n"
            "description: A test skill\n"
            "version: 1.0.0\n"
            "applies_to: [cursor]\n"
            "globs: '**/*.py'\n"
            "alwaysApply: true\n"
            "---\n\n"
            "# Test Skill\n\nBody here.\n"
        )
        installer = MarketplaceInstaller(tmp_path)
        skill = MarketplaceSkill(
            id="test-skill",
            name="Test Skill",
            description="A test skill",
            version="1.0.0",
            applies_to=["cursor"],
            source_owner="test",
            source_repo="repo",
            source_path="skills/test/SKILL.md",
        )
        assert installer.install(skill, "cursor", False) is True
        output = tmp_path / ".cursor" / "rules" / "test-skill.md"
        assert output.exists()
        content = output.read_text()
        assert "GENERATED" in content
        assert "Test Skill" in content
