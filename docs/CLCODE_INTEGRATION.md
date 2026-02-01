# Claude Code Integration

This guide explains how to integrate Agent Guardrails with Claude Code using custom skills and hooks.

## Overview

Claude Code supports:
- **Skills** - JSON files that define specialized behaviors
- **Hooks** - Shell scripts that run at specific points

The setup script generates these configurations for you.

## Setup

### 1. Run Setup Script

```bash
python scripts/setup_agents.py --claude --full
```

This creates:
```
.claude/
├── skills/
│   ├── guardrails-enforcer.json
│   ├── commit-validator.json
│   └── env-separator.json
└── hooks/
    ├── pre-execution.sh
    ├── post-execution.sh
    └── pre-commit.sh
```

### 2. Verify Installation

Check that skills are loaded:
```bash
ls -la .claude/skills/
```

Check that hooks are executable:
```bash
ls -la .claude/hooks/
```

## How It Works

### Skills

Skills are JSON files with:
- `name` - Unique identifier
- `description` - What the skill does
- `tools` - Allowed tools for this skill
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

### Adding a Custom Skill

1. Create a new JSON file in `.claude/skills/`:

```json
{
  "name": "my-skill",
  "description": "What it does",
  "tools": ["Read", "Bash"],
  "prompt": "Your instructions here..."
}
```

2. Restart Claude Code to load the skill

### Modifying Hooks

Edit the shell scripts in `.claude/hooks/`:

```bash
#!/bin/bash
# Custom pre-execution logic
echo "Running custom checks..."
# Add your validation here
```

Make sure hooks remain executable:
```bash
chmod +x .claude/hooks/*.sh
```

## Advanced Configuration

### Skill Selection

By default, all skills are active. To disable a skill temporarily:

1. Move it out of the skills directory:
```bash
mv .claude/skills/commit-validator.json .claude/skills/disabled/
```

2. Restart Claude Code

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

### Skills Not Loading

**Check:**
- JSON syntax is valid: `python -m json.tool .claude/skills/*.json`
- Files are in correct directory
- Claude Code has been restarted

### Hooks Not Running

**Check:**
- Hooks are executable: `chmod +x .claude/hooks/*.sh`
- Shell syntax is valid: `bash -n .claude/hooks/pre-execution.sh`
- Hook names match expected patterns

### Permission Denied

**Fix:**
```bash
chmod +x .claude/hooks/*.sh
```

## Best Practices

1. **Keep skills focused** - One skill = One responsibility
2. **Test hooks independently** - Run scripts manually to verify
3. **Document customizations** - Add comments to modified files
4. **Version control** - Commit `.claude/` to share with team

## References

- [Claude Code Documentation](https://docs.anthropic.com/en/docs/agents-and-tools/claude-code)
- [AGENT_GUARDRAILS.md](AGENT_GUARDRAILS.md) - Core safety protocols
- [AGENTS_AND_SKILLS_SETUP.md](AGENTS_AND_SKILLS_SETUP.md) - General setup guide
