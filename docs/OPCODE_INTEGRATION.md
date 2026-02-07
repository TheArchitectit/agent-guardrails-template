# OpenCode Integration

This guide explains how to integrate Agent Guardrails with OpenCode using custom agents, skills, and hooks.

## Overview

OpenCode supports:
- **Agents** - JSON files that define specialized agent behaviors
- **Skills** - Markdown files with tool definitions and prompts
- **Hooks** - Shell scripts that run at specific points

The setup script generates these configurations for you.

## Setup

### 1. Run Setup Script

```bash
python scripts/setup_agents.py --opencode --full
```

This creates:
```
.opencode/
├── oh-my-opencode.jsonc
├── agents/
│   ├── guardrails-enforcer.json
│   ├── commit-validator.json
│   └── env-separator.json
├── skills/
│   ├── guardrails-enforcer.md
│   ├── commit-validator.md
│   └── env-separator.md
└── hooks/
    ├── pre-execution.sh
    ├── post-execution.sh
    └── pre-commit.sh
```

### 2. Verify Installation

Check that agents are loaded:
```bash
ls -la .opencode/agents/
```

Check that skills are loaded:
```bash
ls -la .opencode/skills/
```

Check that hooks are executable:
```bash
ls -la .opencode/hooks/
```

## How It Works

### Agents

Agents are JSON files with:
- `name` - Unique identifier
- `description` - What the agent does
- `tools` - Allowed tools for this agent
- `prompt` - Instructions injected into context

**Example: guardrails-enforcer.json**
```json
{
  "name": "guardrails-enforcer",
  "description": "Enforces the Four Laws of Agent Safety",
  "tools": ["Read", "Grep", "Glob"],
  "prompt": "You MUST enforce these rules..."
}
```

### Skills

Skills are Markdown files with structured prompts:

**Example: guardrails-enforcer.md**
```markdown
# Guardrails Enforcer

## Description
Enforces the Four Laws of Agent Safety

## Tools
- Read
- Grep
- Glob

## Instructions
You MUST enforce these rules:
1. Read before editing
2. Stay in scope
3. Verify before committing
4. Halt when uncertain
```

### Hooks

Hooks are shell scripts that run automatically:

| Hook | When It Runs | Purpose |
|------|--------------|---------|
| `pre-execution.sh` | Before file modifications | Verify read-before-edit |
| `post-execution.sh` | After file modifications | Validate changes |
| `pre-commit.sh` | Before git commit | Validate commit message |

## Skill Details

### guardrails-enforcer

**Purpose:** Enforces the Four Laws of Agent Safety

**Rules Enforced:**
1. Read before editing
2. Stay in scope
3. Verify before committing
4. Halt when uncertain

**Halt Conditions:**
- Modifying unread code
- Unclear scope boundaries
- No rollback procedure
- Test/production mix
- Three failed attempts

### commit-validator

**Purpose:** Validates git commits

**Checks:**
- AI attribution present (`Co-Authored-By:`)
- Single focus per commit
- No secrets in diff
- Tests pass

**Commit Format:**
```
<type>: <description>

Co-Authored-By: Claude <noreply@anthropic.com>
```

### env-separator

**Purpose:** Enforces test/production separation

**Rules:**
- Production code before tests
- Separate service instances
- No test data in production

**Detection:**
- Production DB connections in tests
- Shared instances
- Hardcoded production credentials

## Customization

### Adding a Custom Agent

1. Create a new JSON file in `.opencode/agents/`:

```json
{
  "name": "my-agent",
  "description": "What it does",
  "tools": ["Read", "Bash"],
  "prompt": "Your instructions here..."
}
```

2. Create corresponding skill in `.opencode/skills/`:

```markdown
# My Agent

## Description
What it does

## Tools
- Read
- Bash

## Instructions
Your detailed instructions here...
```

3. Update `oh-my-opencode.jsonc` to include the new agent

### Modifying Hooks

Edit the shell scripts in `.opencode/hooks/`:

```bash
#!/bin/bash
# Custom pre-execution logic
echo "Running custom checks..."
# Add your validation here
```

Make sure hooks remain executable:
```bash
chmod +x .opencode/hooks/*.sh
```

## Advanced Configuration

### Agent Selection

By default, all agents are active. To disable an agent temporarily:

1. Move it out of the agents directory:
```bash
mv .opencode/agents/commit-validator.json .opencode/agents/disabled/
```

2. Update `oh-my-opencode.jsonc` to remove from active agents list

### Hook Chaining

Hooks can call other tools:

```bash
#!/bin/bash
# pre-commit.sh

# Run linter
npm run lint

# Run tests
npm test

# Check for secrets
trufflehog git file://. --since-commit HEAD
```

## Troubleshooting

### Agents Not Loading

**Check:**
- JSON syntax is valid: `python -m json.tool .opencode/agents/*.json`
- Files are in correct directory
- `oh-my-opencode.jsonc` references the agents correctly

### Skills Not Loading

**Check:**
- Markdown files have proper frontmatter headers
- Files are in `.opencode/skills/` directory
- Agent configs reference correct skill names

### Hooks Not Running

**Check:**
- Hooks are executable: `chmod +x .opencode/hooks/*.sh`
- Shell syntax is valid: `bash -n .opencode/hooks/pre-execution.sh`
- Hook names match expected patterns in config

### Permission Denied

**Fix:**
```bash
chmod +x .opencode/hooks/*.sh
```

## Best Practices

1. **Keep agents focused** - One agent = One responsibility
2. **Test hooks independently** - Run scripts manually to verify
3. **Document customizations** - Add comments to modified files
4. **Version control** - Commit `.opencode/` to share with team

## References

- [OpenCode Documentation](https://docs.opencode.ai)
- [AGENT_GUARDRAILS.md](AGENT_GUARDRAILS.md) - Core safety protocols
- [AGENTS_AND_SKILLS_SETUP.md](AGENTS_AND_SKILLS_SETUP.md) - General setup guide
