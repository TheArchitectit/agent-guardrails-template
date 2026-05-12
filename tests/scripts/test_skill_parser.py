"""Tests for skill_lib/parser.py."""

import json
from pathlib import Path

import pytest

from skill_lib.parser import Skill, parse_skill, validate_skill


SAMPLE_SKILL_MD = """\
---
id: my-skill
name: My Skill
description: Does something useful
version: 1.2.3
tags: [foo, bar]
applies_to: [claude, cursor]
author: alice
references:
  - skills/other/SKILL.md
tools: [Read, Grep]
globs: "**/*.py"
alwaysApply: true
---

# My Skill

Body content here.

See skills/other/SKILL.md.
"""

MINIMAL_SKILL_MD = """\
---
id: minimal
name: Minimal
description: Minimal skill
version: 1.0.0
applies_to: [claude]
---

Body text.
"""

NO_FRONTMATTER = "# No frontmatter\n\nJust body.\n"


class TestParseSkill:
    def test_parses_full_frontmatter(self, tmp_path):
        path = tmp_path / "SKILL.md"
        path.write_text(SAMPLE_SKILL_MD, encoding="utf-8")
        skill = parse_skill(path)

        assert skill.id == "my-skill"
        assert skill.name == "My Skill"
        assert skill.description == "Does something useful"
        assert skill.version == "1.2.3"
        assert skill.tags == ["foo", "bar"]
        assert skill.applies_to == ["claude", "cursor"]
        assert skill.author == "alice"
        assert skill.references == ["skills/other/SKILL.md"]
        assert skill.tools == ["Read", "Grep"]
        assert skill.globs == "**/*.py"
        assert skill.alwaysApply is True
        assert "Body content here." in skill.body
        assert "See skills/other/SKILL.md." in skill.body

    def test_parses_minimal_frontmatter(self, tmp_path):
        path = tmp_path / "SKILL.md"
        path.write_text(MINIMAL_SKILL_MD, encoding="utf-8")
        skill = parse_skill(path)

        assert skill.id == "minimal"
        assert skill.tags == []
        assert skill.tools is None
        assert skill.globs is None
        assert skill.alwaysApply is False

    def test_rejects_missing_frontmatter(self, tmp_path):
        path = tmp_path / "SKILL.md"
        path.write_text(NO_FRONTMATTER, encoding="utf-8")
        with pytest.raises(ValueError, match="Missing YAML frontmatter"):
            parse_skill(path)

    def test_rejects_invalid_frontmatter(self, tmp_path):
        path = tmp_path / "SKILL.md"
        path.write_text("---\n---\nBody", encoding="utf-8")
        skill = parse_skill(path)
        assert skill.id == ""
        assert skill.body == "Body"

    def test_accepts_str_or_path(self, tmp_path):
        path = tmp_path / "SKILL.md"
        path.write_text(MINIMAL_SKILL_MD, encoding="utf-8")
        skill_from_str = parse_skill(str(path))
        skill_from_path = parse_skill(path)
        assert skill_from_str.id == skill_from_path.id


class TestValidateSkill:
    def test_valid_skill_has_no_errors(self, minimal_skill):
        assert validate_skill(minimal_skill) == []

    def test_invalid_skill_reports_all_errors(self, invalid_skill):
        errors = validate_skill(invalid_skill)
        assert "missing 'id'" in errors
        assert "missing 'name'" in errors
        assert "missing 'description'" in errors
        assert "missing 'version'" in errors
        assert "missing 'applies_to'" in errors

    def test_missing_single_field(self):
        skill = Skill(
            id="x",
            name="X",
            description="X",
            version="1.0.0",
            applies_to=[],
            body="",
        )
        assert validate_skill(skill) == ["missing 'applies_to'"]
