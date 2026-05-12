"""Tests for skill_lib/rewriter.py."""

import pytest

from skill_lib.rewriter import rewrite_references


class TestRewriteReferences:
    def test_rewrites_to_anchor_for_copilot(self):
        body = "See skills/four-laws/SKILL.md and skills/halt-conditions/SKILL.md."
        result = rewrite_references(body, "copilot")
        assert result == "See #four-laws and #halt-conditions."

    def test_rewrites_to_anchor_for_windsurf(self):
        body = "See skills/four-laws/SKILL.md for details."
        result = rewrite_references(body, "windsurf")
        assert result == "See #four-laws for details."

    def test_rewrites_to_shared_prompts_for_claude(self):
        body = "See skills/four-laws/SKILL.md."
        result = rewrite_references(body, "claude")
        assert result == "See skills/shared-prompts/four-laws.md."

    def test_rewrites_to_shared_prompts_for_cursor(self):
        body = "See skills/four-laws/SKILL.md."
        result = rewrite_references(body, "cursor")
        assert result == "See skills/shared-prompts/four-laws.md."

    def test_rewrites_to_shared_prompts_for_opencode(self):
        body = "See skills/four-laws/SKILL.md."
        result = rewrite_references(body, "opencode")
        assert result == "See skills/shared-prompts/four-laws.md."

    def test_rewrites_to_shared_prompts_for_openclaw(self):
        body = "See skills/four-laws/SKILL.md."
        result = rewrite_references(body, "openclaw")
        assert result == "See skills/shared-prompts/four-laws.md."

    def test_no_change_when_no_refs(self):
        body = "Just some plain text."
        result = rewrite_references(body, "copilot")
        assert result == body

    def test_multiple_refs_all_replaced(self):
        body = "A: skills/a/SKILL.md B: skills/b/SKILL.md C: skills/c/SKILL.md"
        result = rewrite_references(body, "copilot")
        assert result == "A: #a B: #b C: #c"

    def test_leaves_non_skill_paths_intact(self):
        body = "See docs/guide.md and skills/four-laws/SKILL.md."
        result = rewrite_references(body, "copilot")
        assert "docs/guide.md" in result
        assert "#four-laws" in result
