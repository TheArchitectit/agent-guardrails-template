# Cursor Integration

This guide explains how to integrate Agent Guardrails with Cursor using markdown-based rules.

## Overview

Cursor supports:
- **Rules** - Markdown files that define AI behavior and constraints
- **Global Rules** - `.cursorrules` file in project root for universal settings

The setup script generates these configurations for you.

## Setup

### 1. Run Setup Script

```bash
python scripts/setup_agents.py --cursor --full
```

This creates:
```
.cursor/
├── rules/
│   ├── guardrails-enforcer.md
│   ├── commit-validator.md
│   └── env-separator.md
└── .cursorrules (optional root config)
```

### 2. Verify Installation

Check that rules are loaded:
```bash
ls -la .cursor/rules/
```

Check that `.cursorrules` exists (if using global config):
```bash
cat .cursorrules
```

## How It Works

### Rules

Rules are markdown files with:
- **Frontmatter** - Metadata (name, description, version)
- **Always Section** - Rules applied to all interactions
- **When Section** - Conditional rules for specific contexts

**Example: guardrails-enforcer.md**
```markdown
---
name: guardrails-enforcer
description: Enforces the Four Laws of Agent Safety
version: 1.0.0
---

## Always

You MUST enforce these rules:
1. Read before editing
2. Stay in scope
3. Verify before committing
4. Halt when uncertain

## When

- Before modifying any file
- Before running git commands
- When uncertain about scope
```

### Global Rules (.cursorrules)

The `.cursorrules` file in the project root applies to all Cursor sessions:

```markdown
# Project Rules

## Always

- Follow the Four Laws of Agent Safety
- Read files before editing
- Validate commits before creating

## When

- Editing code: Check scope boundaries
- Running commands: Verify environment separation
```

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

### Adding a Custom Rule

1. Create a new markdown file in `.cursor/rules/`:

```markdown
---
name: my-rule
description: What it does
version: 1.0.0
---

## Always

Your universal instructions here.

## When

- Context for conditional application
```

2. Cursor automatically loads rules from this directory

### Modifying Global Rules

Edit `.cursorrules` in the project root:

```markdown
# Custom Global Rules

## Always

- Your custom rule 1
- Your custom rule 2

## When

- Specific context: Apply specific rule
```

### Rule Priority

Rules are applied in order:
1. `.cursorrules` (global) - Applied first
2. `.cursor/rules/*.md` - Applied in alphabetical order

Later rules can override earlier ones for the same context.

## Advanced Configuration

### Rule Selection

By default, all rules in `.cursor/rules/` are active. To disable a rule temporarily:

1. Move it out of the rules directory:
```bash
mv .cursor/rules/commit-validator.md .cursor/rules/disabled/
```

2. Cursor will stop loading it immediately

### Rule Chaining

Rules can reference other rules in their instructions:

```markdown
---
name: custom-validator
---

## Always

Apply the commit-validator rules, then additionally:
- Check for TODO comments
- Verify no console.log statements
```

### Conditional Rules

Use the `When` section for context-specific behavior:

```markdown
## When

- Editing Python files: Follow PEP 8
- Editing JavaScript: Use ESLint rules
- Running tests: Ensure env-separator rules are active
```

## Troubleshooting

### Rules Not Loading

**Check:**
- Markdown syntax is valid
- Frontmatter is properly formatted with `---` delimiters
- Files are in `.cursor/rules/` directory
- File extension is `.md`

### .cursorrules Not Applied

**Check:**
- File is in project root (not `.cursor/`)
- File is named exactly `.cursorrules` (no extension)
- Cursor has indexed the project

### Rules Being Ignored

**Fix:**
- Ensure rule files are saved
- Restart Cursor to reload rules
- Check for conflicting rules (later rules override earlier ones)

## Best Practices

1. **Keep rules focused** - One rule = One responsibility
2. **Use frontmatter** - Always include name and description
3. **Document customizations** - Add comments explaining complex rules
4. **Version control** - Commit `.cursor/` and `.cursorrules` to share with team
5. **Test rules** - Verify rules work as expected in practice

## References

- [Cursor Rules Documentation](https://docs.cursor.com/context/rules)
- [AGENT_GUARDRAILS.md](AGENT_GUARDRAILS.md) - Core safety protocols
- [AGENTS_AND_SKILLS_SETUP.md](AGENTS_AND_SKILLS_SETUP.md) - General setup guide
