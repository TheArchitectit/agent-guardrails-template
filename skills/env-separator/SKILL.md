---
id: env-separator
name: Environment Separator
description: Enforces strict separation between test and production environments
version: 1.0.0
tags: [safety, core, mandatory]
applies_to: [claude, cursor, opencode, openclaw, windsurf, copilot]
author: TheArchitectit
tools: [Read, Grep, Glob, AskUserQuestion]
globs: "**/*"
alwaysApply: true
---

# Environment Separator

Enforce strict separation between test and production environments.

## The Three Laws of Environment Separation

1. **Production Code First** - Production code MUST be created before test code
2. **Separate Instances** - Test and production MUST use separate service instances
3. **No Data Mixing** - Test data must NEVER contaminate production databases

## Pre-Flight Checklist

Before creating test code or running tests:

- [ ] Production implementation exists and is functional
- [ ] Test environment uses separate database/service instances
- [ ] Test data will not leak to production
- [ ] Separate user accounts for test vs production

## Forbidden Patterns (NEVER ALLOW)

1. Tests writing to production databases
2. Test fixtures in production code paths
3. Shared database instances (even separate schemas)
4. Test credentials in production configs
5. Production data in tests without sanitization

## When Uncertain

If you cannot verify environment separation:

1. HALT the operation immediately
2. Ask the user to confirm environment boundaries
3. Do NOT proceed until separation is guaranteed

## References

- `docs/standards/TEST_PRODUCTION_SEPARATION.md` - Full environment rules
- `skills/production-first/SKILL.md` - Production-first mandate
- `docs/AGENT_GUARDRAILS.md` - Core safety protocols
