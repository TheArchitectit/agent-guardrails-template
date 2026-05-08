#!/usr/bin/env python3
"""
Agent Guardrails Setup Script

Installs pre-committed guardrails configurations for various AI coding platforms.

This script reads from the canonical .claude/, .cursor/, .opencode/, and .windsurfrules
files in this repository and copies or symlinks them to the target project directory.

Usage:
    python scripts/setup_agents.py --install --platform claude,cursor,opencode,windsurf,copilot [--target /path/to/project] [--mode copy|symlink] [--dry-run]

Examples:
    python scripts/setup_agents.py --install --platform claude --target ~/myproject
    python scripts/setup_agents.py --install --platform claude,cursor,opencode --mode symlink --dry-run
    python scripts/setup_agents.py --list-platforms
"""

import argparse
import json
import os
import shutil
import sys
from pathlib import Path
from typing import Optional


SCRIPT_DIR = Path(__file__).parent.resolve()
REPO_ROOT = SCRIPT_DIR.parent

PLATFORM_CONFIGS = {
    "claude": {
        "source": REPO_ROOT / ".claude",
        "description": "Claude Code skills and hooks",
        "target_name": ".claude",
    },
    "cursor": {
        "source": REPO_ROOT / ".cursor" / "rules",
        "description": "Cursor rules",
        "target_name": ".cursor/rules",
    },
    "opencode": {
        "source": REPO_ROOT / ".opencode",
        "description": "OpenCode agents and skills",
        "target_name": ".opencode",
    },
    "windsurf": {
        "source": REPO_ROOT / ".windsurfrules",
        "description": "Windsurf rules",
        "target_name": ".windsurfrules",
    },
    "copilot": {
        "source": REPO_ROOT / ".github" / "copilot-instructions.md",
        "description": "GitHub Copilot instructions",
        "target_name": ".github/copilot-instructions.md",
    },
}


def resolve_target(target_path: Optional[str], platform: str) -> Path:
    """Resolve target directory for a platform."""
    if target_path:
        base = Path(target_path).resolve()
    else:
        base = REPO_ROOT
    config = PLATFORM_CONFIGS[platform]
    return base / config["target_name"]


def ensure_parent_dirs(path: Path) -> None:
    """Ensure parent directories exist for a path."""
    parent = path.parent
    if parent != path:
        parent.mkdir(parents=True, exist_ok=True)


def install_platform(target_root: Path, platform: str, mode: str = "copy", dry_run: bool = False) -> bool:
    """Install guardrails config for a single platform."""
    config = PLATFORM_CONFIGS[platform]
    source = config["source"]
    target = resolve_target(str(target_root), platform)

    if dry_run:
        action = "symlink" if mode == "symlink" else "copy"
        print(f"[DRY-RUN] Would {action}: {source} -> {target}")
        return True

    if not source.exists():
        print(f"[ERROR] Source not found: {source}")
        return False

    if target.exists():
        print(f"[WARN] Target exists, skipping: {target}")
        return True

    ensure_parent_dirs(target)

    try:
        if mode == "symlink":
            rel_source = os.path.relpath(source, target.parent)
            target.symlink_to(rel_source)
            print(f"[OK] Symlinked: {target} -> {rel_source}")
        else:
            if source.is_dir():
                shutil.copytree(source, target, dirs_exist_ok=False)
            else:
                shutil.copy2(source, target)
            print(f"[OK] Copied: {source} -> {target}")
        return True
    except Exception as e:
        print(f"[ERROR] Failed to install {platform}: {e}")
        return False


def install_all(target_root: Optional[str], platforms: list[str], mode: str, dry_run: bool) -> bool:
    """Install configs for all specified platforms."""
    target = Path(target_root) if target_root else REPO_ROOT
    success = True
    for platform in platforms:
        if platform not in PLATFORM_CONFIGS:
            print(f"[ERROR] Unknown platform: {platform}")
            success = False
            continue
        if not install_platform(target, platform, mode, dry_run):
            success = False
    return success


def list_platforms() -> None:
    """List available platforms."""
    print("Available platforms:")
    for name, config in PLATFORM_CONFIGS.items():
        exists = " [exists]" if config["source"].exists() else " [missing]"
        print(f"  {name}: {config['description']}{exists}")


def validate_sources(platforms: list[str]) -> bool:
    """Validate that all source files exist."""
    missing = []
    for platform in platforms:
        if platform not in PLATFORM_CONFIGS:
            missing.append(platform)
            continue
        source = PLATFORM_CONFIGS[platform]["source"]
        if not source.exists():
            missing.append(f"{platform} ({source})")
    if missing:
        print("[ERROR] Missing source files:")
        for m in missing:
            print(f"  {m}")
        return False
    return True


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Install agent guardrails for Claude Code, Cursor, OpenCode, Windsurf, and Copilot."
    )
    parser.add_argument(
        "--install",
        action="store_true",
        help="Install guardrails configs (required to perform installation)",
    )
    parser.add_argument(
        "--platform",
        type=str,
        default="claude,cursor,opencode,windsurf,copilot",
        help="Comma-separated list of platforms (default: all)",
    )
    parser.add_argument(
        "--target",
        type=str,
        default=None,
        help="Target project directory (default: this repository)",
    )
    parser.add_argument(
        "--mode",
        type=str,
        choices=["copy", "symlink"],
        default="copy",
        help="Installation mode: copy files or create symlinks (default: copy)",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Preview what would be installed without making changes",
    )
    parser.add_argument(
        "--list-platforms",
        action="store_true",
        help="List available platforms and exit",
    )

    args = parser.parse_args()

    if args.list_platforms:
        list_platforms()
        return 0

    if not args.install and not args.dry_run:
        parser.print_help()
        print("\n[ERROR] Use --install to perform installation, or --dry-run to preview.")
        return 1

    platforms = [p.strip() for p in args.platform.split(",")]
    if "all" in platforms:
        platforms = list(PLATFORM_CONFIGS.keys())

    if not validate_sources(platforms):
        return 1

    if not install_all(args.target, platforms, args.mode, args.dry_run):
        return 1

    if args.dry_run:
        print("\n[INFO] Dry-run complete. Re-run without --dry-run to install.")
    else:
        print(f"\n[OK] Installed guardrails for: {', '.join(platforms)}")

    return 0


if __name__ == "__main__":
    sys.exit(main())
