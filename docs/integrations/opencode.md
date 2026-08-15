# OpenCode Integration

How to wire Agent Guardrails into OpenCode using agents, skills, and hooks.

## Overview

OpenCode uses a multi-agent architecture built on a JSONC config file
(`.opencode/oh-my-opencode.jsonc`). The guardrails integration gives you:

- **Agents** — specialized workers (enforcer, auditor, doc-indexer) with model
  selection and permissions.
- **Skills** — markdown files in `.opencode/skills/` that inject guardrail rules
  into the agent's context.
- **Hooks** — shell scripts at lifecycle points (pre-execution, post-execution,
  pre-commit).
- **MCP server** — a remote Guardrail MCP server for live validation.

## Setup

### Run the setup script

```bash
python scripts/setup_agents.py --install --platform opencode
```

This creates:

```
.opencode/
├── oh-my-opencode.jsonc
├── agents/
│   ├── guardrails-auditor.json
│   └── doc-indexer.json
├── skills/
│   ├── guardrails-enforcer/SKILL.md
│   ├── commit-validator/SKILL.md
│   └── env-separator/SKILL.md
└── hooks/
    ├── pre-execution.sh
    ├── post-execution.sh
    └── pre-commit.sh
```

### Verify

```bash
ls -la .opencode/agents/      # agents present
ls -la .opencode/skills/*/    # skills present
chmod +x .opencode/hooks/*.sh # hooks executable
python -m json.tool .opencode/oh-my-opencode.jsonc  # config valid
```

Restart OpenCode — it loads config on startup.

## Connecting to a Guardrail MCP server

If you have a deployed Guardrail MCP server, point OpenCode at it in
`oh-my-opencode.jsonc`:

```jsonc
{
  "$schema": "https://raw.githubusercontent.com/code-yeongyu/oh-my-opencode/master/assets/oh-my-opencode.schema.json",
  "mcpServers": {
    "guardrails": {
      "type": "remote",
      "url": "http://your-server:8080/mcp",
      "headers": {
        "Authorization": "Bearer YOUR_MCP_API_KEY"
      }
    }
  }
}
```

Use the external port your server exposes. The `Authorization` header must use
`Bearer` format — `X-API-Key` won't work. Get the key from your server's `.env`
(`MCP_API_KEY`).

Verify the connection:

```bash
curl http://your-server:8081/health/ready   # {"status":"ready",...}
curl http://your-server:8080/mcp -X POST \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}'
```

## Configuration format

OpenCode uses **JSONC** (JSON with comments). A full example:

```jsonc
{
  "$schema": "https://raw.githubusercontent.com/code-yeongyu/oh-my-opencode/master/assets/oh-my-opencode.schema.json",
  "agents": {
    "guardrails-enforcer": {
      "model": "anthropic/claude-sonnet-4",
      "temperature": 0.1,
      "prompt_append": "Before ANY operation verify: 1) File read, 2) Scope authorized, 3) Rollback known, 4) No forbidden patterns. HALT and ask if uncertain.",
      "permissions": { "edit": "ask", "bash": "ask", "read": "allow", "webfetch": "allow" }
    },
    "guardrails-auditor": {
      "model": "anthropic/claude-sonnet-4",
      "temperature": 0.1,
      "permissions": { "edit": "deny", "bash": "deny", "read": "allow" }
    }
  },
  "skills": {
    "sources": [{ "path": "./.opencode/skills", "recursive": true }],
    "enable": ["guardrails-enforcer", "commit-validator", "env-separator"]
  }
}
```

### Agent fields

| Field | Type | Description |
|-------|------|-------------|
| `model` | string | Model identifier, e.g. `anthropic/claude-sonnet-4` |
| `temperature` | number | 0.0 deterministic → 1.0 creative. Use 0.0–0.1 for guardrails. |
| `prompt_append` | string | Instructions appended to the system prompt |
| `permissions` | object | Tool permissions (`allow` / `ask` / `deny`) |

| Permission | Behavior |
|------------|----------|
| `allow` | Always permitted, no prompt |
| `ask` | Prompt the user first |
| `deny` | Never permitted |

### Cost-based categories

OpenCode routes work by cost tier:

| Category | Cost | Use for |
|----------|------|---------|
| `quick` | $ | Fast queries, greps |
| `unspecified-low` | $$ | Standard operations |
| `unspecified-high` | $$$ | Complex analysis |
| `visual-engineering` | $$$ | UI/UX work |

Override the model per category:

```jsonc
{ "categories": { "quick": { "model": "opencode/gpt-5-nano" } } }
```

## Included agents

### guardrails-enforcer
Real-time safety enforcement. Validates read-before-edit, enforces scope
boundaries, checks halt conditions. Temperature 0.1, asks before edit/bash.

### guardrails-auditor
Post-execution compliance review. Read-only — no edit or bash access. Reviews
completed work, reports violations, suggests corrections.

### doc-indexer
Keeps documentation maps updated. Runs on document changes. Uses a fast, cheap
model; edit permissions for docs only.

## Included skills

| Skill | Activates | Enforces |
|-------|-----------|----------|
| `guardrails-enforcer` | All operations | Four Laws, pre-operation checklist, halt conditions, three strikes |
| `commit-validator` | Before commits | AI attribution, single focus, no secrets, tests passing |
| `env-separator` | Test code creation | Production code first, separate instances, no data mixing |

### Skill markdown format

```markdown
---
name: skill-name
description: "What this skill does"
---

# Skill Title

## Tools
- Read
- Grep

## Instructions
You MUST enforce these rules...
```

## Hooks

Shell scripts that run automatically at lifecycle points:

| Hook | When | Purpose |
|------|------|---------|
| `pre-execution.sh` | Before file modifications | Verify read-before-edit |
| `post-execution.sh` | After file modifications | Validate changes |
| `pre-commit.sh` | Before git commit | Validate commit message, run checks |

Custom hook example:

```bash
#!/bin/bash
# .opencode/hooks/pre-commit.sh
npm run lint
npm test
trufflehog git file://. --since-commit HEAD
```

## Shared prompts

Skills and agents draw rules from `skills/shared-prompts/`:

| Shared prompt | Used by |
|---------------|---------|
| `four-laws.md` | guardrails-enforcer |
| `halt-conditions.md` | guardrails-enforcer |
| `three-strikes.md` | three-strikes skill |
| `production-first.md` | production-first skill |
| `clean-architecture.md` | guardrails-enforcer |
| `scope-validation.md` | scope-validator skill |
| `error-recovery.md` | error-recovery skill |

Re-run the setup script after updating shared prompts:

```bash
python scripts/setup_agents.py --install --platform opencode
```

## Customization

### Add a custom agent

1. Add an entry to the `agents` object in `oh-my-opencode.jsonc`.
2. Create a matching skill in `.opencode/skills/my-skill/SKILL.md`.
3. Add the skill name to `skills.enable`.

### Install a single skill

```bash
python scripts/setup_agents.py --install-skill guardrails-enforcer --platform opencode
```

### Installation modes

| Mode | Command | Behavior |
|------|---------|----------|
| Copy | `--mode copy` (default) | Standalone copies in the project |
| Symlink | `--mode symlink` | Symlinks back to this repo |

## Troubleshooting

**Agents not loading** — validate JSON: `python -m json.tool .opencode/oh-my-opencode.jsonc`; confirm entries exist in the `agents` object; confirm the file is at `.opencode/`.

**Skills not activating** — check `skills.enable` lists the skill; confirm `SKILL.md` has valid YAML frontmatter; confirm the directory name matches the skill name.

**Hooks not running** — `chmod +x .opencode/hooks/*.sh`; `bash -n .opencode/hooks/pre-execution.sh` to check syntax.

## Differences from Claude Code

| Feature | Claude Code | OpenCode |
|---------|-------------|----------|
| Config format | JSON | JSONC |
| Skills location | `.claude/skills/*.json` | `.opencode/skills/*/SKILL.md` |
| Agents | Implicit | Explicit definition |
| Hooks | Shell scripts | Shell scripts |
| Cost routing | N/A | Category-based |

## References

- [OpenCode](https://github.com/code-yeongyu/opencode)
- [Oh My OpenCode schema](https://raw.githubusercontent.com/code-yeongyu/oh-my-opencode/master/assets/oh-my-opencode.schema.json)
- [Agent guardrails](../getting-started/agent-guardrails.md) — core safety protocols
- [Agents and skills setup](agents-and-skills-setup.md) — general setup guide
