# Skill Registry

All canonical skills live in `skills/<id>/SKILL.md` and are generated into native IDE formats by `scripts/build_skills.py`.

## Available Skills

### guardrails-enforcer

Enforces the **Four Laws of Agent Safety**:
1. Read before editing
2. Stay in scope
3. Verify before committing
4. Halt when uncertain

**Applies to:** Claude, Cursor, OpenCode, OpenClaw, Windsurf, Copilot

### commit-validator

Validates commits against `COMMIT_WORKFLOW.md`:
- AI attribution required
- Single focus per commit
- No secrets in diff
- Tests pass before commit

**Applies to:** Claude, Cursor, OpenCode, OpenClaw, Windsurf, Copilot

### env-separator

Enforces `TEST_PRODUCTION_SEPARATION.md`:
- Production code before test code
- Separate service instances
- No test data in production

**Applies to:** Claude, Cursor, OpenCode, OpenClaw, Windsurf, Copilot

### scope-validator

Validates file modifications stay within authorized scope. Escalates when scope is unclear.

**Applies to:** Claude, Cursor, OpenCode, OpenClaw, Windsurf, Copilot

### production-first

Enforces production code is created before test or infrastructure code.

**Applies to:** Claude, Cursor, OpenCode, OpenClaw, Windsurf, Copilot

### three-strikes

Tracks failure attempts and halts after 3 strikes on any task.

**Applies to:** Claude, Cursor, OpenCode, OpenClaw, Windsurf, Copilot

### error-recovery

Guides recovery from failures without making things worse.

**Applies to:** Claude, Cursor, OpenCode, OpenClaw, Windsurf, Copilot

### 3d-game-dev

3D game development guardrails: mathematical correctness, asset safety, shader constraints, Godot/Unity conventions.

**Applies to:** Claude, Cursor, OpenCode, OpenClaw, Windsurf, Copilot

### vibe-coding

Principles for AI-driven rapid development: guardrails enable speed, decide by risk level, preserve design intent, ship accessible by default, iterate don't rebuild.

**Applies to:** Claude, Cursor, OpenCode, OpenClaw, Windsurf, Copilot

## Customization

### Modifying Skills

Edit the **canonical source** in `skills/<id>/SKILL.md`, then run:

```bash
python scripts/build_skills.py
```

This regenerates all native IDE files. Do not edit generated files directly — they have `GENERATED` headers.

### Creating Custom Skills

Create a new directory in `skills/` with a `SKILL.md`:

```markdown
---
id: my-custom-skill
name: My Custom Skill
description: "What this skill does"
version: 1.0.0
applies_to: [claude, cursor, opencode, openclaw, windsurf, copilot]
---

# My Custom Skill

Your skill instructions here...
```

Run `python scripts/build_skills.py` to generate native formats.

## References

- [Superpowers Architecture](SUPERPOWERS_ARCHITECTURE.md) - Build script and CI/CD details
- [AGENTS_AND_SKILLS_SETUP.md](AGENTS_AND_SKILLS_SETUP.md) - Installation quick start
