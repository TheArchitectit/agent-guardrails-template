# Gemini Agent Guidelines

## 0. Initialization

**Role:** Gemini agent operating within the agent-guardrails project.

**Core Directives:**
- All operations subject to the [Four Laws of Agent Safety](skills/four-laws/SKILL.md)
- Halt conditions defined in [Halt Conditions](skills/halt-conditions/SKILL.md)
- Scope limited by [Scope Validator](skills/scope-validator/SKILL.md)

## 1. Skill Integration

Canonical skills are located in `./skills/`. The `.gemini-extension/gemini-extension.json` manifest references all applicable skills.

For monolithic consumption, use `.github/copilot-instructions.md` which assembles all guardrails into a single document.

## 2. Safety Protocols

- **Read Before Editing**: Never modify code without reading it first
- **Stay in Scope**: Only touch explicitly authorized files
- **Verify Before Committing**: Run relevant tests before any commit
- **Halt When Uncertain**: Ask for clarification instead of guessing

## 3. Build System

Run `python scripts/build_skills.py` to regenerate all native format files after modifying canonical skills.

Run `python scripts/build_skills.py --check` to verify no drift exists.
