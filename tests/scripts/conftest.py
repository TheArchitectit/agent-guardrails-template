"""Shared fixtures for build_skills tests."""

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent.parent / "scripts"))

import pytest

from skill_lib.parser import Skill


@pytest.fixture
def sample_skill():
    """Return a fully populated Skill for testing."""
    return Skill(
        id="test-skill",
        name="Test Skill",
        description="A skill for testing.",
        version="1.0.0",
        tags=["test"],
        applies_to=["claude", "cursor", "copilot"],
        author="tester",
        references=["skills/other/SKILL.md"],
        tools=["Read", "Grep"],
        globs="**/*.py",
        alwaysApply=True,
        body="# Test Skill\n\nSome instructions.\n\nSee skills/other/SKILL.md for more.",
    )


@pytest.fixture
def minimal_skill():
    """Return a minimal valid Skill."""
    return Skill(
        id="minimal",
        name="Minimal",
        description="Minimal skill.",
        version="1.0.0",
        applies_to=["claude"],
        body="Body text.",
    )


@pytest.fixture
def invalid_skill():
    """Return a Skill missing required fields."""
    return Skill(
        id="",
        name="",
        description="",
        version="",
        applies_to=[],
        body="Body text.",
    )
