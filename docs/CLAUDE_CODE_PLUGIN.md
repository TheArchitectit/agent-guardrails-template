# Claude Code Plugin

This repository is a native Claude Code plugin. Install it directly via the Claude Code plugin system or the official Anthropic marketplace.

## Local Testing

Test the plugin without installing it from a marketplace:

```bash
claude --plugin-dir /path/to/agent-guardrails-template
```

Once loaded, skills are namespaced under the plugin name:

```
/agent-guardrails:guardrails-enforcer
/agent-guardrails:commit-validator
/agent-guardrails:env-separator
/agent-guardrails:scope-validator
/agent-guardrails:production-first
/agent-guardrails:three-strikes
/agent-guardrails:error-recovery
/agent-guardrails:3d-game-dev
/agent-guardrails:four-laws
/agent-guardrails:halt-conditions
/agent-guardrails:vibe-coding
```

### Reload After Changes

Run `/reload-plugins` inside Claude Code to pick up modifications without restarting.

## Official Marketplace Submission

Submit the plugin to Anthropic's official marketplace:

- **Claude.ai**: [claude.ai/settings/plugins/submit](https://claude.ai/settings/plugins/submit)
- **Console**: [platform.claude.com/plugins/submit](https://platform.claude.com/plugins/submit)

### Plugin Manifest

The manifest at `.claude-plugin/plugin.json` follows the official Anthropic schema:

```json
{
  "name": "agent-guardrails",
  "description": "Core safety skills for AI-assisted development",
  "version": "3.2.0",
  "author": { "name": "TheArchitectit" },
  "homepage": "https://github.com/TheArchitectit/agent-guardrails-template",
  "repository": "https://github.com/TheArchitectit/agent-guardrails-template",
  "license": "MIT",
  "keywords": ["skills", "guardrails", "safety", "ai-agents", "workflows"]
}
```

### Skill Format

Each skill lives in `skills/<id>/SKILL.md` with YAML frontmatter:

```markdown
---
id: guardrails-enforcer
name: Guardrails Enforcement Agent
description: Enforces the Four Laws of Agent Safety
version: 1.0.0
applies_to: [claude, cursor, opencode, openclaw, windsurf, copilot]
---

# Guardrails Enforcement Agent

Skill instructions here...
```

Claude Code uses the `description` field to know when to invoke the skill automatically. The remaining frontmatter fields are used by the cross-platform build pipeline.

## Cross-Platform Distribution

For non-Claude platforms (Cursor, Copilot, Windsurf, OpenCode), use the marketplace CLI:

```bash
python scripts/marketplace.py install guardrails-enforcer --platform cursor
```

See [MARKETPLACE.md](MARKETPLACE.md) for the full cross-platform marketplace guide.
