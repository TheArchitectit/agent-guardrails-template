"""Rewrite cross-skill references for target platforms."""

import re

_REF_PATTERN = re.compile(r"skills/([a-z0-9_-]+)/SKILL\.md")


def rewrite_references(body: str, platform: str) -> str:
    """Rewrite cross-skill references for the target platform.

    For monolithic platforms (copilot, windsurf), skill paths become section
    anchors. For per-file platforms, paths are rewritten to the generated
    shared-prompts location for backward compatibility.

    Args:
        body: The markdown body containing cross-skill references.
        platform: Target platform name (e.g. "claude", "copilot").

    Returns:
        The body with references rewritten for the platform.
    """
    if platform in ("copilot", "windsurf"):
        return _REF_PATTERN.sub(r"#\1", body)
    return _REF_PATTERN.sub(r"skills/shared-prompts/\1.md", body)
