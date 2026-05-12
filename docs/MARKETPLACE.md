# Plugin Marketplace

Cross-platform skill marketplace for distributing guardrails to Claude Code, Cursor, OpenCode, OpenClaw, and other AI coding tools.

## Quick Start

```bash
# Register the official marketplace
python scripts/marketplace.py add TheArchitectit/agent-guardrails-template

# Search available skills
python scripts/marketplace.py search guardrails
python scripts/marketplace.py search --platform cursor

# Install a skill for your platform
python scripts/marketplace.py install guardrails-enforcer --platform claude
python scripts/marketplace.py install guardrails-enforcer --platform cursor --target ~/myproject

# Preview without installing
python scripts/marketplace.py install guardrails-enforcer --platform claude --dry-run
```

## Commands

| Command | Description |
|---------|-------------|
| `add <owner/repo>` | Register a marketplace from a GitHub repo |
| `remove <id>` | Remove a registered marketplace |
| `list` | List all registered marketplaces |
| `search <term>` | Search skills across all marketplaces |
| `install <skill>` | Install a skill for a target platform |
| `refresh` | Refresh all marketplace catalogs |

### Install Syntax

```bash
# Install from default marketplace (auto-detects platform)
python scripts/marketplace.py install guardrails-enforcer

# Install for a specific platform
python scripts/marketplace.py install guardrails-enforcer --platform cursor

# Install from a specific marketplace
python scripts/marketplace.py install guardrails-enforcer@official --platform claude

# Install to a specific project
python scripts/marketplace.py install guardrails-enforcer --target ~/myproject --platform claude
```

### Setup Agents Integration

The existing setup script also supports marketplace installs:

```bash
python scripts/setup_agents.py --install-skill guardrails-enforcer@official --platform claude
```

## Supported Platforms

| Platform | Output Format | Location |
|----------|---------------|----------|
| Claude Code | JSON skill | `.claude/skills/<id>.json` |
| Cursor | Markdown rule | `.cursor/rules/<id>.md` |
| OpenCode | SKILL.md | `.opencode/skills/<id>/SKILL.md` |
| OpenClaw | SKILL.md | `.openclaw/skills/<id>/SKILL.md` |

For Claude Code, you can also install the plugin directly — see [CLAUDE_CODE_PLUGIN.md](CLAUDE_CODE_PLUGIN.md).

## Marketplace JSON Schema

Each marketplace is a GitHub repository with a `marketplace.json` at its root:

```json
{
  "name": "my-marketplace",
  "displayName": "My Marketplace",
  "description": "Custom skills for AI development",
  "version": "1.0.0",
  "author": "Your Name",
  "repository": "https://github.com/you/your-repo",
  "skills": [
    {
      "id": "my-skill",
      "name": "My Skill",
      "description": "What this skill does",
      "version": "1.0.0",
      "applies_to": ["claude", "cursor", "opencode", "openclaw"],
      "source": {
        "type": "github",
        "owner": "you",
        "repo": "your-repo",
        "path": "skills/my-skill/SKILL.md",
        "branch": "main"
      }
    }
  ]
}
```

## Publishing Your Own Marketplace

1. Create a GitHub repo with your skills in `skills/<id>/SKILL.md`
2. Add a `marketplace.json` at the root
3. Share the repo URL with users
4. Users register it with: `python scripts/marketplace.py add you/your-repo`

## References

- [Skills Architecture](SKILLS_ARCHITECTURE.md) — Build script and canonical format
- [AGENTS_AND_SKILLS_SETUP.md](AGENTS_AND_SKILLS_SETUP.md) — Installation quick start
- [CLAUDE_CODE_PLUGIN.md](CLAUDE_CODE_PLUGIN.md) — Claude Code native plugin
