# Agent Guardrails - Agents and Skills Setup

This guide explains how to install pre-committed guardrails configurations for Claude Code, Cursor, OpenCode, Windsurf, and GitHub Copilot — full platforms or individual skills.

## Quick Start

```bash
# Install all platforms (copy mode)
python scripts/setup_agents.py --install

# Install specific platforms
python scripts/setup_agents.py --install --platform claude,cursor,windsurf

# Preview what would be installed
python scripts/setup_agents.py --install --dry-run

# Install to a different project directory
python scripts/setup_agents.py --install --platform claude --target ~/myproject

# Symlink instead of copy (updates track the repo)
python scripts/setup_agents.py --install --platform claude,cursor --mode symlink
```

### Clone Individual Skill Files (No Repo Clone)

Download any single skill file directly from GitHub — no git clone needed:

```bash
# Clone a skill file by repo path
python scripts/setup_agents.py --clone .claude/skills/guardrails-enforcer.json

# Clone to a specific project
python scripts/setup_agents.py --clone .cursor/rules/guardrails-enforcer.md --target ~/myproject
```

### Install Per-Skill (Named Install)

Install one specific skill at a time by name:

```bash
# List all available skills
python scripts/setup_agents.py --list-skills

# Install a single Claude Code skill
python scripts/setup_agents.py --install-skill guardrails-enforcer

# Install a shared prompt to any project
python scripts/setup_agents.py --install-skill four-laws --target ~/myproject
```

### Available Platforms

| Platform | Config Location | Description |
|----------|----------------|-------------|
| claude | `.claude/` | Claude Code skills and hooks |
| cursor | `.cursor/rules/` | Cursor rules |
| opencode | `.opencode/` | OpenCode agents and skills |
| windsurf | `.windsurfrules` | Windsurf rules |
| copilot | `.github/copilot-instructions.md` | GitHub Copilot instructions |

### MCP Tool

The `guardrail_install_skills` MCP tool wraps the install script with all three modes:

```javascript
// Full platform install
guardrail_install_skills({ platforms: "claude,cursor", target_path: "/path/to/project" })

// Clone a single file (downloads from GitHub)
guardrail_install_skills({ action: "clone", path: ".claude/skills/guardrails-enforcer.json" })

// Install a single skill by name
guardrail_install_skills({ action: "install", skill: "guardrails-enforcer" })

// List available skills
guardrail_install_skills({ list_skills: true })

// List available platforms
guardrail_install_skills({ list_platforms: true })
```

## What Gets Created

### Claude Code Configuration (`.claude/`)

**Minimal:**
- `.claude/skills/guardrails-enforcer.json` - Core safety enforcement

**Full:**
- `.claude/skills/guardrails-enforcer.json` - Four Laws enforcement
- `.claude/skills/commit-validator.json` - Commit message validation
- `.claude/skills/env-separator.json` - Test/production separation
- `.claude/hooks/pre-execution.sh` - Pre-operation checks
- `.claude/hooks/post-execution.sh` - Post-operation validation
- `.claude/hooks/pre-commit.sh` - Pre-commit validation

### OpenCode Configuration (`.opencode/`)

**Minimal:**
- `.opencode/oh-my-opencode.jsonc` - Main configuration
- `.opencode/skills/guardrails-enforcer/SKILL.md` - Core safety skill

**Full:**
- `.opencode/oh-my-opencode.jsonc` - Main configuration with all agents
- `.opencode/skills/guardrails-enforcer/SKILL.md` - Four Laws enforcement
- `.opencode/skills/commit-validator/SKILL.md` - Commit validation
- `.opencode/skills/env-separator/SKILL.md` - Environment separation
- `.opencode/agents/guardrails-auditor.json` - Post-work auditor
- `.opencode/agents/doc-indexer.json` - Documentation indexer

## Platform-Specific Guides

- [Claude Code Integration](CLCODE_INTEGRATION.md) - Detailed Claude Code setup
- [OpenCode Integration](OPENCODE_INTEGRATION.md) - Detailed OpenCode setup

## Removing Configuration

To remove the setup:

```bash
# Remove Claude Code configuration
rm -rf .claude/

# Remove OpenCode configuration
rm -rf .opencode/

# Remove both
rm -rf .claude/ .opencode/
```

## Troubleshooting

### Skills Not Loading

**Claude Code:**
- Ensure `.claude/skills/` directory exists
- Check JSON syntax in skill files
- Restart Claude Code

**OpenCode:**
- Verify `.opencode/oh-my-opencode.jsonc` syntax
- Check that skills are listed in the `enable` array
- Restart OpenCode

### Hooks Not Running

- Ensure hooks are executable: `chmod +x .claude/hooks/*.sh`
- Check hook syntax with `bash -n <hook-file>`
- Verify hook paths in your configuration

## References

- [Skill Registry](SKILL_REGISTRY.md) - All available skills and customization
- [Skills Architecture](SKILLS_ARCHITECTURE.md) - Build script, CI/CD, and advanced workflows
- [AGENT_GUARDRAILS.md](AGENT_GUARDRAILS.md) - Core safety protocols
- [TEST_PRODUCTION_SEPARATION.md](standards/TEST_PRODUCTION_SEPARATION.md) - Environment rules
- [COMMIT_WORKFLOW.md](workflows/COMMIT_WORKFLOW.md) - Commit standards
- [AGENT_EXECUTION.md](workflows/AGENT_EXECUTION.md) - Execution protocols
