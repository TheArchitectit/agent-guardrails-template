#!/usr/bin/env python3
"""
Plugin Marketplace CLI for Guardrails Skills

Manages marketplace registries and installs skills for any supported platform.
Works alongside Claude Code's native plugin system for cross-platform distribution.

Usage:
    python scripts/marketplace.py add TheArchitectit/agent-guardrails-template
    python scripts/marketplace.py list
    python scripts/marketplace.py search guardrails
    python scripts/marketplace.py search --platform cursor
    python scripts/marketplace.py install guardrails-enforcer --platform claude
    python scripts/marketplace.py install guardrails-enforcer --platform cursor --dry-run
    python scripts/marketplace.py refresh
    python scripts/marketplace.py remove official
"""

import argparse
import json
import os
import sys
import urllib.error
import urllib.request
from dataclasses import dataclass, field
from datetime import datetime, timezone
from pathlib import Path
from typing import Optional

SCRIPT_DIR = Path(__file__).parent.resolve()
REPO_ROOT = SCRIPT_DIR.parent
sys.path.insert(0, str(SCRIPT_DIR))

from skill_lib import generator, parser


# ---------------------------------------------------------------------------
# Local config paths
# ---------------------------------------------------------------------------
GUARDRAILS_DIR = REPO_ROOT / ".guardrails"
MARKETPLACES_JSON = GUARDRAILS_DIR / "marketplaces.json"

DEFAULT_MARKETPLACE = {
    "id": "official",
    "name": "agent-guardrails-official",
    "url": "https://github.com/TheArchitectit/agent-guardrails-template",
    "marketplace_json_url": "https://raw.githubusercontent.com/TheArchitectit/agent-guardrails-template/main/marketplace.json",
    "added_at": datetime.now(timezone.utc).isoformat(),
}


# ---------------------------------------------------------------------------
# Data models
# ---------------------------------------------------------------------------
@dataclass
class MarketplaceEntry:
    id: str
    name: str
    url: str
    marketplace_json_url: str
    added_at: str


@dataclass
class MarketplaceSkill:
    id: str
    name: str
    description: str
    version: str
    applies_to: list[str] = field(default_factory=list)
    source_owner: str = ""
    source_repo: str = ""
    source_path: str = ""
    source_branch: str = "main"


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
def _now() -> str:
    return datetime.now(timezone.utc).isoformat()


def _ensure_dir(path: Path) -> None:
    path.mkdir(parents=True, exist_ok=True)


def _raw_github_url(owner: str, repo: str, branch: str, path: str) -> str:
    return f"https://raw.githubusercontent.com/{owner}/{repo}/{branch}/{path}"


def _fetch_json(url: str) -> dict:
    with urllib.request.urlopen(url) as response:
        return json.loads(response.read())


def _fetch_text(url: str) -> str:
    with urllib.request.urlopen(url) as response:
        return response.read().decode("utf-8")


def _download(url: str, target: Path, dry_run: bool) -> bool:
    if dry_run:
        print(f"[DRY-RUN] Would download: {url}")
        print(f"[DRY-RUN]            -> {target}")
        return True
    try:
        _ensure_dir(target.parent)
        text = _fetch_text(url)
        target.write_text(text, encoding="utf-8")
        print(f"[OK] Downloaded: {url}")
        print(f"     -> {target}")
        return True
    except urllib.error.HTTPError as e:
        print(f"[ERROR] HTTP {e.code}: {url}")
        return False
    except Exception as e:
        print(f"[ERROR] Download failed: {e}")
        return False


def _parse_repo_spec(spec: str) -> tuple[str, str]:
    """Convert 'owner/repo' or a GitHub URL into (owner, repo)."""
    spec = spec.strip().rstrip("/")
    if spec.startswith("https://github.com/"):
        parts = spec.replace("https://github.com/", "").split("/")
        return parts[0], parts[1]
    if "/" in spec:
        parts = spec.split("/")
        return parts[0], parts[1]
    raise ValueError(f"Invalid repo spec: {spec}")


# ---------------------------------------------------------------------------
# MarketplaceRegistry
# ---------------------------------------------------------------------------
class MarketplaceRegistry:
    """Local registry of subscribed marketplaces."""

    def __init__(self, path: Path = MARKETPLACES_JSON):
        self.path = path
        self.data: dict = {"version": "1.0.0", "marketplaces": []}
        self._load()

    def _load(self) -> None:
        if self.path.exists():
            self.data = json.loads(self.path.read_text(encoding="utf-8"))

    def _save(self) -> None:
        _ensure_dir(self.path.parent)
        self.path.write_text(json.dumps(self.data, indent=2) + "\n", encoding="utf-8")

    def entries(self) -> list[MarketplaceEntry]:
        return [
            MarketplaceEntry(
                id=m["id"],
                name=m.get("name", m["id"]),
                url=m["url"],
                marketplace_json_url=m["marketplace_json_url"],
                added_at=m.get("added_at", ""),
            )
            for m in self.data.get("marketplaces", [])
        ]

    def get(self, entry_id: str) -> Optional[MarketplaceEntry]:
        for e in self.entries():
            if e.id == entry_id:
                return e
        return None

    def add(self, entry: MarketplaceEntry) -> bool:
        for m in self.data["marketplaces"]:
            if m["id"] == entry.id:
                print(f"[WARN] Marketplace already registered: {entry.id}")
                return False
        self.data["marketplaces"].append(
            {
                "id": entry.id,
                "name": entry.name,
                "url": entry.url,
                "marketplace_json_url": entry.marketplace_json_url,
                "added_at": entry.added_at,
            }
        )
        self._save()
        print(f"[OK] Added marketplace: {entry.id}")
        return True

    def remove(self, entry_id: str) -> bool:
        before = len(self.data["marketplaces"])
        self.data["marketplaces"] = [
            m for m in self.data["marketplaces"] if m["id"] != entry_id
        ]
        if len(self.data["marketplaces"]) == before:
            print(f"[ERROR] Marketplace not found: {entry_id}")
            return False
        self._save()
        print(f"[OK] Removed marketplace: {entry_id}")
        return True

    def ensure_default(self) -> None:
        """Seed the official marketplace if none exist."""
        if not self.data["marketplaces"]:
            self.add(
                MarketplaceEntry(
                    id=DEFAULT_MARKETPLACE["id"],
                    name=DEFAULT_MARKETPLACE["name"],
                    url=DEFAULT_MARKETPLACE["url"],
                    marketplace_json_url=DEFAULT_MARKETPLACE["marketplace_json_url"],
                    added_at=DEFAULT_MARKETPLACE["added_at"],
                )
            )


# ---------------------------------------------------------------------------
# MarketplaceClient
# ---------------------------------------------------------------------------
class MarketplaceClient:
    """Fetches and parses marketplace.json from remote repositories."""

    @staticmethod
    def fetch(entry: MarketplaceEntry) -> dict:
        print(f"[INFO] Fetching marketplace: {entry.marketplace_json_url}")
        try:
            return _fetch_json(entry.marketplace_json_url)
        except urllib.error.HTTPError as e:
            # If the marketplace points to the local repo and the remote fetch
            # fails (e.g. file not yet on main), fall back to the local copy.
            if e.code == 404 and entry.id == "official":
                local_path = REPO_ROOT / "marketplace.json"
                if local_path.exists():
                    print(f"[INFO] Falling back to local: {local_path}")
                    return json.loads(local_path.read_text(encoding="utf-8"))
            raise

    @staticmethod
    def parse_skills(data: dict) -> list[MarketplaceSkill]:
        skills = []
        for s in data.get("skills", []):
            source = s.get("source", {})
            skills.append(
                MarketplaceSkill(
                    id=s["id"],
                    name=s.get("name", s["id"]),
                    description=s.get("description", ""),
                    version=s.get("version", ""),
                    applies_to=s.get("applies_to", []),
                    source_owner=source.get("owner", ""),
                    source_repo=source.get("repo", ""),
                    source_path=source.get("path", ""),
                    source_branch=source.get("branch", "main"),
                )
            )
        return skills


# ---------------------------------------------------------------------------
# MarketplaceInstaller
# ---------------------------------------------------------------------------
class MarketplaceInstaller:
    """Downloads a skill and generates native IDE format."""

    PLATFORM_OUTPUTS = {
        "claude": (".claude/skills/{id}.json", generator.claude_content),
        "cursor": (".cursor/rules/{id}.md", generator.cursor_content),
        "opencode": (".opencode/skills/{id}/SKILL.md", generator.oc_content),
        "openclaw": (".openclaw/skills/{id}/SKILL.md", generator.oc_content),
    }

    def __init__(self, target_root: Path):
        self.target_root = target_root

    def _skill_url(self, skill: MarketplaceSkill) -> str:
        return _raw_github_url(
            skill.source_owner,
            skill.source_repo,
            skill.source_branch,
            skill.source_path,
        )

    def install(self, skill: MarketplaceSkill, platform: str, dry_run: bool) -> bool:
        if platform not in self.PLATFORM_OUTPUTS:
            print(f"[ERROR] Unknown platform: {platform}")
            print(f"[INFO] Supported: {', '.join(self.PLATFORM_OUTPUTS.keys())}")
            return False

        if platform not in skill.applies_to:
            print(f"[ERROR] Skill does not support platform: {platform}")
            return False

        rel_path, content_fn = self.PLATFORM_OUTPUTS[platform]
        target = self.target_root / rel_path.format(id=skill.id)

        if dry_run:
            print(f"[DRY-RUN] Would download SKILL.md and generate {platform} format")
            print(f"[DRY-RUN] Would write: {target}")
            return True

        # Download SKILL.md to a temp location for parsing
        url = self._skill_url(skill)
        try:
            raw_skill = _fetch_text(url)
        except Exception as e:
            # Fallback to local copy if the skill is from this repo
            local_skill = REPO_ROOT / skill.source_path
            if local_skill.exists():
                print(f"[INFO] Falling back to local skill: {local_skill}")
                raw_skill = local_skill.read_text(encoding="utf-8")
            else:
                print(f"[ERROR] Failed to download skill: {e}")
                return False

        # Write temp file so parser can read it
        temp_path = GUARDRAILS_DIR / "tmp" / f"{skill.id}.md"
        _ensure_dir(temp_path.parent)
        temp_path.write_text(raw_skill, encoding="utf-8")

        try:
            parsed = parser.parse_skill(temp_path)
        except Exception as e:
            print(f"[ERROR] Failed to parse skill: {e}")
            return False
        finally:
            temp_path.unlink(missing_ok=True)

        # Generate native content
        if platform in ("opencode", "openclaw"):
            content = content_fn(parsed, platform)
        else:
            content = content_fn(parsed)

        _ensure_dir(target.parent)
        target.write_text(content, encoding="utf-8")
        print(f"[OK] Installed {skill.id} for {platform}: {target}")
        return True


# ---------------------------------------------------------------------------
# CLI
# ---------------------------------------------------------------------------
def cmd_add(args) -> int:
    registry = MarketplaceRegistry()
    try:
        owner, repo = _parse_repo_spec(args.repo_spec)
    except ValueError as e:
        print(f"[ERROR] {e}")
        return 1

    entry_id = args.id or repo
    url = f"https://github.com/{owner}/{repo}"
    marketplace_json_url = f"https://raw.githubusercontent.com/{owner}/{repo}/main/marketplace.json"

    entry = MarketplaceEntry(
        id=entry_id,
        name=entry_id,
        url=url,
        marketplace_json_url=marketplace_json_url,
        added_at=_now(),
    )

    ok = registry.add(entry)
    if not ok:
        # Try fetching to verify the marketplace exists
        try:
            client = MarketplaceClient()
            client.fetch(entry)
        except Exception as e:
            print(f"[WARN] Could not verify marketplace.json: {e}")
    return 0 if ok else 1


def cmd_remove(args) -> int:
    registry = MarketplaceRegistry()
    ok = registry.remove(args.id)
    return 0 if ok else 1


def cmd_list(_args) -> int:
    registry = MarketplaceRegistry()
    registry.ensure_default()
    entries = registry.entries()
    if not entries:
        print("No marketplaces registered.")
        return 0
    print("Registered marketplaces:")
    for e in entries:
        print(f"  {e.id:20s} {e.url}")
    return 0


def cmd_search(args) -> int:
    registry = MarketplaceRegistry()
    registry.ensure_default()
    client = MarketplaceClient()
    term = (args.term or "").lower()
    platform = args.platform
    found_any = False

    for entry in registry.entries():
        try:
            data = client.fetch(entry)
            skills = client.parse_skills(data)
        except Exception as e:
            print(f"[WARN] Failed to fetch {entry.id}: {e}")
            continue

        matched = []
        for s in skills:
            if platform and platform not in s.applies_to:
                continue
            if term and term not in s.name.lower() and term not in s.description.lower():
                continue
            matched.append(s)

        if matched:
            found_any = True
            print(f"\n  [{entry.id}]")
            for s in matched:
                platforms = ", ".join(s.applies_to)
                print(f"    {s.id:30s} {s.name}")
                print(f"      {s.description}")
                print(f"      platforms: {platforms}")

    if not found_any:
        print("No skills found.")
    return 0


def cmd_install(args) -> int:
    registry = MarketplaceRegistry()
    registry.ensure_default()
    client = MarketplaceClient()
    installer = MarketplaceInstaller(Path(args.target or ".").resolve())

    # Parse skill@marketplace syntax
    skill_id = args.skill
    marketplace_id = args.marketplace
    if "@" in skill_id:
        skill_id, marketplace_id = skill_id.split("@", 1)

    # Find the skill
    skill: Optional[MarketplaceSkill] = None
    entry: Optional[MarketplaceEntry] = None

    # If marketplace specified, search only there
    if marketplace_id:
        entry = registry.get(marketplace_id)
        if not entry:
            print(f"[ERROR] Marketplace not found: {marketplace_id}")
            return 1
        try:
            data = client.fetch(entry)
            skills = client.parse_skills(data)
        except Exception as e:
            print(f"[ERROR] Failed to fetch marketplace: {e}")
            return 1
        for s in skills:
            if s.id == skill_id:
                skill = s
                break
    else:
        # Search all marketplaces
        for e in registry.entries():
            try:
                data = client.fetch(e)
                skills = client.parse_skills(data)
            except Exception as ex:
                print(f"[WARN] Failed to fetch {e.id}: {ex}")
                continue
            for s in skills:
                if s.id == skill_id:
                    skill = s
                    entry = e
                    break
            if skill:
                break

    if not skill:
        print(f"[ERROR] Skill not found: {skill_id}")
        return 1

    # Determine platform
    platform = args.platform
    if not platform:
        # Default to the first applicable platform
        for p in ("claude", "cursor", "opencode", "openclaw"):
            if p in skill.applies_to:
                platform = p
                break
        if not platform:
            platform = skill.applies_to[0] if skill.applies_to else "claude"
        print(f"[INFO] Auto-selected platform: {platform}")

    ok = installer.install(skill, platform, args.dry_run)
    return 0 if ok else 1


def cmd_refresh(_args) -> int:
    registry = MarketplaceRegistry()
    registry.ensure_default()
    print("Refreshing marketplace catalogs...")
    for entry in registry.entries():
        try:
            client = MarketplaceClient()
            data = client.fetch(entry)
            skill_count = len(data.get("skills", []))
            print(f"  [OK] {entry.id}: {skill_count} skills")
        except Exception as e:
            print(f"  [ERROR] {entry.id}: {e}")
    return 0


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
def main() -> int:
    p = argparse.ArgumentParser(
        description="Plugin Marketplace CLI for Guardrails Skills"
    )
    sub = p.add_subparsers(dest="command", required=True)

    # add
    add_p = sub.add_parser("add", help="Register a marketplace")
    add_p.add_argument("repo_spec", help="GitHub repo as owner/repo or full URL")
    add_p.add_argument("--id", help="Custom marketplace ID (default: repo name)")

    # remove
    remove_p = sub.add_parser("remove", help="Remove a marketplace")
    remove_p.add_argument("id", help="Marketplace ID")

    # list
    sub.add_parser("list", help="List registered marketplaces")

    # search
    search_p = sub.add_parser("search", help="Search skills across marketplaces")
    search_p.add_argument("term", nargs="?", help="Search term")
    search_p.add_argument("--platform", help="Filter by platform")

    # install
    install_p = sub.add_parser("install", help="Install a skill")
    install_p.add_argument("skill", help="Skill ID or skill@marketplace")
    install_p.add_argument("--platform", help="Target platform (claude, cursor, opencode, openclaw)")
    install_p.add_argument("--marketplace", help="Marketplace ID (optional)")
    install_p.add_argument("--target", help="Target project directory")
    install_p.add_argument("--dry-run", action="store_true", help="Preview only")

    # refresh
    sub.add_parser("refresh", help="Refresh all marketplace catalogs")

    args = p.parse_args()
    handlers = {
        "add": cmd_add,
        "remove": cmd_remove,
        "list": cmd_list,
        "search": cmd_search,
        "install": cmd_install,
        "refresh": cmd_refresh,
    }
    return handlers[args.command](args)


if __name__ == "__main__":
    sys.exit(main())
