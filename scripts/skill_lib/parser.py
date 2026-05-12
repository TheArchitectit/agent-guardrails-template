"""Parse canonical SKILL.md files into structured Skill dataclasses."""

from dataclasses import dataclass, field
from pathlib import Path

import yaml


@dataclass
class Skill:
    """Canonical skill representation from YAML frontmatter + markdown body."""

    id: str
    name: str
    description: str
    version: str
    tags: list[str] = field(default_factory=list)
    applies_to: list[str] = field(default_factory=list)
    author: str = ""
    references: list[str] = field(default_factory=list)
    tools: list[str] | None = None
    globs: str | None = None
    alwaysApply: bool = False
    body: str = ""


def parse_skill(path: str | Path) -> Skill:
    """Parse a SKILL.md file into a Skill dataclass.

    Args:
        path: Path to the SKILL.md file.

    Returns:
        A Skill dataclass with frontmatter fields and markdown body.

    Raises:
        ValueError: If the file lacks valid YAML frontmatter.
    """
    content = Path(path).read_text(encoding="utf-8")
    if not content.startswith("---"):
        raise ValueError(f"Missing YAML frontmatter in {path}")

    parts = content.split("---", 2)
    if len(parts) < 3:
        raise ValueError(f"Invalid YAML frontmatter in {path}")

    frontmatter = yaml.safe_load(parts[1])
    if frontmatter is None:
        frontmatter = {}

    return Skill(
        id=frontmatter.get("id", ""),
        name=frontmatter.get("name", ""),
        description=frontmatter.get("description", ""),
        version=frontmatter.get("version", ""),
        tags=frontmatter.get("tags", []),
        applies_to=frontmatter.get("applies_to", []),
        author=frontmatter.get("author", ""),
        references=frontmatter.get("references", []),
        tools=frontmatter.get("tools"),
        globs=frontmatter.get("globs"),
        alwaysApply=frontmatter.get("alwaysApply", False),
        body=parts[2].strip(),
    )


def validate_skill(skill: Skill) -> list[str]:
    """Validate a Skill dataclass and return a list of error messages.

    Args:
        skill: The Skill to validate.

    Returns:
        A list of human-readable error strings. Empty list means valid.
    """
    errors = []
    if not skill.id:
        errors.append("missing 'id'")
    if not skill.name:
        errors.append("missing 'name'")
    if not skill.description:
        errors.append("missing 'description'")
    if not skill.version:
        errors.append("missing 'version'")
    if not skill.applies_to:
        errors.append("missing 'applies_to'")
    return errors
