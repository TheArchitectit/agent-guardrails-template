# Workflow Documentation Index

> Navigation hub for all workflow documentation.

---

## Overview

This directory contains operational workflow documentation for AI agents. These documents define **how** to perform common operations safely and consistently.

**Before using these workflows:** Always read [AGENT_GUARDRAILS.md](../AGENT_GUARDRAILS.md) first.

---

## Quick Reference Table

| Document | Purpose | When to Use |
|----------|---------|-------------|
| [TESTING_VALIDATION.md](./TESTING_VALIDATION.md) | Validation and verification protocols | Before committing changes |
| [COMMIT_WORKFLOW.md](./COMMIT_WORKFLOW.md) | Commit timing and message format | After completing to-dos |
| [GIT_PUSH_PROCEDURES.md](./GIT_PUSH_PROCEDURES.md) | Push safety and verification | Before pushing to remote |
| [BRANCH_STRATEGY.md](./BRANCH_STRATEGY.md) | Branch naming and workflows | When creating branches |
| [ROLLBACK_PROCEDURES.md](./ROLLBACK_PROCEDURES.md) | Undo and recovery operations | When errors occur |
| [CODE_REVIEW.md](./CODE_REVIEW.md) | Review process and escalation | After code changes |
| [MCP_CHECKPOINTING.md](./MCP_CHECKPOINTING.md) | MCP checkpoint integration | Before/after critical tasks |
| [DOCUMENTATION_UPDATES.md](./DOCUMENTATION_UPDATES.md) | Post-sprint doc updates | After completing sprints |

---

## Document Summaries

### TESTING_VALIDATION.md
Defines validation functions and git diff verification patterns. Use this to understand how to double-check all work before committing.

**Key sections:**
- Pre-edit validation patterns
- Post-edit validation
- Git diff verification protocol
- Language-specific validation

### COMMIT_WORKFLOW.md
Guidelines for when and how to commit changes, especially the rule for committing between to-do items.

**Key sections:**
- Commit decision matrix
- "Commit after each to-do" rule
- Commit message format
- Pre-commit checklist

### GIT_PUSH_PROCEDURES.md
Safety procedures for pushing to remote repositories, including pre-push checklists and error handling.

**Key sections:**
- Pre-push checklist
- Push decision matrix
- Branch-specific procedures
- Push safety rules

### BRANCH_STRATEGY.md
Git branching conventions including naming, workflows, and merge strategies.

**Key sections:**
- Branch naming conventions
- Feature branch workflow
- Hotfix emergency procedures
- Merge vs rebase guidance

### ROLLBACK_PROCEDURES.md
Comprehensive recovery and undo operations for all scenarios.

**Key sections:**
- Immediate rollback (uncommitted)
- Post-commit rollback
- Post-push rollback
- Disaster recovery checklist

### CODE_REVIEW.md
Code review process for agents including self-review and human escalation.

**Key sections:**
- Agent self-review checklist
- When to request human review
- Review focus areas
- Escalation procedures

### MCP_CHECKPOINTING.md
Integration with MCP servers for automatic checkpointing before and after tasks.

**Key sections:**
- Checkpoint concepts
- MCP server integration
- Checkpoint-aware sprints
- Recovery from checkpoints

### DOCUMENTATION_UPDATES.md
Procedures for updating documentation after sprints and code changes.

**Key sections:**
- Post-sprint updates
- Documentation review checklist
- Cross-reference maintenance

---

## Integration with Guardrails

All workflows in this directory **support** the safety protocols defined in [AGENT_GUARDRAILS.md](../AGENT_GUARDRAILS.md). They provide detailed implementation guidance for:

- **Verification** → TESTING_VALIDATION.md
- **Commits** → COMMIT_WORKFLOW.md
- **Rollback** → ROLLBACK_PROCEDURES.md
- **Error handling** → Referenced throughout

---

## Related Documents

- [AGENT_GUARDRAILS.md](../AGENT_GUARDRAILS.md) - Mandatory safety protocols
- [../standards/INDEX.md](../standards/INDEX.md) - Documentation standards
- [../sprints/INDEX.md](../sprints/INDEX.md) - Sprint task framework

---

**Last Updated:** 2026-01-14
**Document Count:** 8
