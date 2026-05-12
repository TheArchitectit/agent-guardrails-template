#!/usr/bin/env python3
"""Build native IDE skill files from canonical SKILL.md sources."""

import argparse
import sys
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
sys.path.insert(0, str(SCRIPT_DIR))

from skill_lib.assembler import assemble_content, generate_copilot, generate_windsurf
from skill_lib.generator import (
    claude_content,
    cursor_content,
    generate_claude,
    generate_cursor,
    generate_openclaw,
    generate_opencode,
    oc_content,
)
from skill_lib.parser import parse_skill, validate_skill

SKILLS_DIR = Path("skills")
REPO_ROOT = SCRIPT_DIR.parent

PLATFORM_HANDLERS = {
    "claude": (generate_claude, ".claude/skills"),
    "cursor": (generate_cursor, ".cursor/rules"),
    "opencode": (generate_opencode, ".opencode/skills"),
    "openclaw": (generate_openclaw, ".openclaw/skills"),
    "copilot": (generate_copilot, ".github/copilot-instructions.md"),
    "windsurf": (generate_windsurf, ".windsurfrules"),
}


def discover_skills(skills_dir: Path) -> list:
    """Discover and parse all SKILL.md files under the given directory."""
    skills = []
    for path in skills_dir.rglob("SKILL.md"):
        try:
            skill = parse_skill(path)
            errors = validate_skill(skill)
            if errors:
                print(f"Validation errors in {path}: {', '.join(errors)}", file=sys.stderr)
                continue
            skills.append(skill)
        except Exception as e:
            print(f"Failed to parse {path}: {e}", file=sys.stderr)
    return skills


def filter_skills(skills, single_skill: str | None = None) -> list:
    """Filter skills to a single id if requested."""
    if single_skill:
        skills = [s for s in skills if s.id == single_skill]
        if not skills:
            print(f"Skill '{single_skill}' not found.", file=sys.stderr)
            sys.exit(1)
    return skills


def _collect_outputs(skills) -> dict[str, str]:
    """Map relative output paths to expected content strings."""
    outputs: dict[str, str] = {}

    for skill in skills:
        if "claude" in skill.applies_to:
            outputs[f".claude/skills/{skill.id}.json"] = claude_content(skill)

    for skill in skills:
        if "cursor" in skill.applies_to:
            outputs[f".cursor/rules/{skill.id}.md"] = cursor_content(skill)

    for skill in skills:
        if "opencode" in skill.applies_to:
            outputs[f".opencode/skills/{skill.id}/SKILL.md"] = oc_content(skill, "opencode")

    for skill in skills:
        if "openclaw" in skill.applies_to:
            outputs[f".openclaw/skills/{skill.id}/SKILL.md"] = oc_content(skill, "openclaw")

    outputs[".github/copilot-instructions.md"] = assemble_content(skills, "copilot")
    outputs[".windsurfrules"] = assemble_content(skills, "windsurf")

    return outputs


def _filter_outputs_by_platform(outputs: dict[str, str], platforms: list[str]) -> dict[str, str]:
    """Keep only output entries belonging to the requested platforms."""
    filtered = {}
    for rel_path, content in outputs.items():
        for platform in platforms:
            out_spec = PLATFORM_HANDLERS[platform][1]
            if rel_path.startswith(out_spec) or rel_path == out_spec:
                filtered[rel_path] = content
                break
    return filtered


def main() -> int:
    parser = argparse.ArgumentParser(description="Build IDE skill files from canonical SKILL.md")
    parser.add_argument("--check", action="store_true", help="Drift check: exit 1 if generated files differ")
    parser.add_argument("--dry-run", action="store_true", help="Preview changes without writing")
    parser.add_argument("--platform", choices=list(PLATFORM_HANDLERS.keys()), help="Build only one platform")
    parser.add_argument("--skill", help="Rebuild a single skill by id")
    args = parser.parse_args()

    skills = discover_skills(REPO_ROOT / SKILLS_DIR)
    skills = filter_skills(skills, args.skill)

    if not skills:
        print("No skills found.", file=sys.stderr)
        return 1

    platforms = [args.platform] if args.platform else list(PLATFORM_HANDLERS.keys())

    if args.check or args.dry_run:
        outputs = _collect_outputs(skills)
        outputs = _filter_outputs_by_platform(outputs, platforms)
        drift = False

        for rel_path, expected in outputs.items():
            actual_path = REPO_ROOT / rel_path
            if not actual_path.exists():
                print(f"MISSING: {rel_path}")
                drift = True
                continue
            actual = actual_path.read_text(encoding="utf-8")
            if actual != expected:
                print(f"DIFFERS: {rel_path}")
                drift = True

        if args.check:
            return 1 if drift else 0
        return 0

    # Normal build
    for platform in platforms:
        handler, out_spec = PLATFORM_HANDLERS[platform]
        if platform in ("copilot", "windsurf"):
            out_path = REPO_ROOT / out_spec
            handler(skills, str(out_path))
        else:
            handler(skills, str(REPO_ROOT))
        print(f"Built {platform}")

    print("Done.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
