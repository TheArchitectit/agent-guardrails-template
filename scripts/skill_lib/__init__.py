"""skill_lib — build-tool library for canonical SKILL.md → native IDE files."""

from .assembler import assemble_content, generate_copilot, generate_windsurf
from .generator import (
    claude_content,
    cursor_content,
    generate_claude,
    generate_cursor,
    generate_openclaw,
    generate_opencode,
    oc_content,
)
from .parser import Skill, parse_skill, validate_skill
from .rewriter import rewrite_references

__all__ = [
    "Skill",
    "parse_skill",
    "validate_skill",
    "rewrite_references",
    "claude_content",
    "cursor_content",
    "oc_content",
    "generate_claude",
    "generate_cursor",
    "generate_opencode",
    "generate_openclaw",
    "assemble_content",
    "generate_copilot",
    "generate_windsurf",
]
