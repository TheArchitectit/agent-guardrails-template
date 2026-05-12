# Multi-Agent Architecture

## Overview

This project uses a multi-agent system with specialized roles. Each agent type has a dedicated set of skills loaded from `./skills/` via platform-specific manifests.

## Agent Types

| Agent | Role | Skills |
|---|---|---|
| **Guardrails Enforcer** | Safety oversight on all operations | `guardrails-enforcer`, `four-laws`, `halt-conditions` |
| **Build Agent** | Build system, CI/CD, native format generation | `commit-validator`, `env-separator`, `production-first` |
| **GDUI Agent** | Game design, UI/UX, spatial computing | `3d-game-dev`, `scope-validator` |
| **Vibe Coder** | Rapid development within guardrails | `vibe-coding`, `error-recovery`, `three-strikes` |

## Safety Layer

All agents inherit the Four Laws:
1. Read Before Editing
2. Stay in Scope
3. Verify Before Committing
4. Halt When Uncertain

## Skill Distribution

- **Per-file platforms** (Claude, Cursor, OpenCode, OpenClaw): Each skill generates its own native file
- **Monolithic platforms** (Copilot, Windsurf, Gemini): All skills assembled into one instructions file

## Plugin Manifests

| Platform | Manifest | Generated Output |
|---|---|---|
| Claude | `.claude-plugin/plugin.json` | `.claude/skills/*.json` |
| Cursor | `.cursor-plugin/plugin.json` | `.cursor/rules/*.md` |
| Codex | `.codex-plugin/plugin.json` | `.github/copilot-instructions.md` |
| Gemini | `.gemini-extension/gemini-extension.json` | `.github/copilot-instructions.md` |

## Rebuilding

```bash
python scripts/build_skills.py         # Build all platforms
python scripts/build_skills.py --check # Verify no drift
```
