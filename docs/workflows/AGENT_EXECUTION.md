# Agent Execution Protocol

> Standard task execution flow, rollback procedures, and error handling for AI agents.

**Related:** [../AGENT_GUARDRAILS.md](../AGENT_GUARDRAILS.md) | [TESTING_VALIDATION.md](./TESTING_VALIDATION.md)

---

## Overview

This document defines the standard execution protocol for AI agents, including task flow, decision matrices, rollback procedures, and error handling protocols. All agent operations must follow this protocol to ensure safe, predictable behavior.

---

## EXECUTION PROTOCOL

### Standard Task Flow

```
┌─────────────────────────────────────────────────────────────┐
│              UNIVERSAL AGENT EXECUTION FLOW                  │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  PHASE 1: PREPARATION                                        │
│  ├── Read task requirements                                  │
│  ├── Identify scope boundaries                               │
│  └── Plan execution steps                                    │
│                                                              │
│  PHASE 2: VERIFICATION                                       │
│  ├── Read target file(s)                                     │
│  ├── Verify preconditions match                              │
│  └── Confirm rollback procedure known                        │
│                                                              │
│  PHASE 3: EXECUTION                                          │
│  ├── Apply single, focused change                            │
│  ├── Syntax check immediately                                │
│  └── If error: HALT and rollback                             │
│                                                              │
│  PHASE 4: VALIDATION                                         │
│  ├── Run related tests                                       │
│  ├── Perform manual verification                             │
│  └── If failure: HALT and rollback                           │
│                                                              │
│  PHASE 5: COMPLETION                                         │
│  ├── Commit with proper message                              │
│  ├── Generate completion report                              │
│  └── Await user review before push                           │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Decision Matrix

| Condition | Action |
|-----------|--------|
| Preconditions match | PROCEED to next step |
| Preconditions don't match | HALT and report |
| Test passes | PROCEED to next step |
| Test fails | ROLLBACK and report |
| Uncertain about anything | HALT and ask |
| User requests stop | STOP immediately |
| Error encountered | ROLLBACK and report |

---

## ROLLBACK PROCEDURES

### Immediate Rollback (Uncommitted Changes)

```bash
# Discard changes to specific file
git checkout HEAD -- path/to/file.py

# Discard all uncommitted changes
git checkout HEAD -- .

# Verify clean state
git status
```

### Rollback After Commit (Not Pushed)

```bash
# Undo last commit, keep changes staged
git reset --soft HEAD~1

# Undo last commit, discard changes
git reset --hard HEAD~1

# Verify state
git log --oneline -3
```

### Rollback After Push (REQUIRES USER PERMISSION)

```bash
# Create revert commit (safe, preferred method)
git revert HEAD

# Push revert
git push origin main
```

**CRITICAL:** Never use `git push --force` without explicit user permission and understanding of consequences.

### Database Rollback Considerations

When rolling back database changes:

```
IF DATABASE CHANGES MADE:
  1. Check if migration was applied
  2. If migration NOT applied → rollback is safe
  3. If migration WAS applied → ask user for direction
  4. Never revert migrations without explicit permission
  5. Ask user: Should I create rollback migration?

IF DATA CHANGES MADE:
  1. Check if production data affected
  2. If production data → HALT and escalate to user
  3. If test data only → can rollback safely
  4. Consider data backup if uncertain
```

### Service Rollback Procedures

When rolling back service deployments:

```
IF SERVICE DEPLOYED:
  1. Check current service version
  2. Identify previous stable version
  3. Ask user: Revert to previous version?
  4. Use service rollback command (e.g., kubectl, docker)
  5. Verify service health after rollback

IF CONFIGURATION CHANGED:
  1. Reload previous configuration
  2. Restart service if needed
  3. Verify configuration applied correctly
  4. Monitor service logs for errors
```

---

## COMMIT MESSAGE FORMAT

### Format Template

```
<type>(<scope>): <short description>

<optional body - explain why, not what>

Authored by TheArchitectit
```

### Commit Types

| Type | Use For |
|------|---------|
| `fix` | Bug fixes |
| `feat` | New features |
| `docs` | Documentation only |
| `refactor` | Code change that doesn't fix bug or add feature |
| `test` | Adding or updating tests |
| `chore` | Maintenance tasks |
| `perf` | Performance improvements |
| `security` | Security fixes |

### Good vs Bad Messages

```
BAD:
- "fix bug"
- "update code"
- "changes"
- "WIP"

GOOD:
- "fix(parser): handle empty string input gracefully"
- "feat(api): add rate limiting to public endpoints"
- "docs(readme): add troubleshooting section"
```

### AI Attribution

**All AI-generated commits MUST include:**

```
Authored by TheArchitectit
```

Use HEREDOC for proper formatting:

```bash
git commit -m "$(cat <<'EOF'
fix(parser): handle null input gracefully

Added null check to prevent TypeError when input is undefined.

Authored by TheArchitectit
EOF
)"
```

---

## ERROR HANDLING PROTOCOLS

### Syntax Error After Edit

```
1. IMMEDIATELY execute: git checkout HEAD -- <file>
2. Report exact error message to user
3. Report line number if available
4. DO NOT attempt additional fixes without user guidance
5. Mark task as FAILED
```

### Test Failure After Edit

```
1. IMMEDIATELY execute: git checkout HEAD -- <file>
2. Capture full test output
3. Report which test(s) failed
4. Report failure reason if determinable
5. DO NOT attempt fixes without user guidance
6. Mark task as FAILED
```

### Edit Operation Failed

```
1. Re-read the target file
2. Compare expected vs actual content
3. Report the mismatch to user
4. Request updated instructions
5. DO NOT guess at alternative edits
6. Mark task as BLOCKED
```

### Unknown Error

```
1. Capture error message and stack trace
2. Rollback any partial changes
3. Report full error context to user
4. DO NOT retry without user guidance
5. Mark task as ERROR
```

### Database Error

```
1. Check if production database affected
2. If production error → HALT and escalate IMMEDIATELY
3. If test database error → can attempt recovery
4. Check connection string and environment
5. Ask user: Should I retry or rollback?

IF PRODUCTION DATABASE ERROR:
  → STOP ALL OPERATIONS
  → Report error with full context
  → DO NOT make any manual corrections
  → Wait for user guidance
```

### Service Error

```
1. Check service logs for details
2. Check if service is running
3. Check configuration is correct
4. Check dependencies are available
5. Ask user: Should I restart service or rollback?

IF PRODUCTION SERVICE ERROR:
  → Check service health status
  → Report error with full context
  → Wait for user guidance
  → DO NOT restart without permission
```

---

## VERIFICATION CHECKLIST

### Before Marking Task Complete

**Before marking ANY task complete, verify ALL items:**

```
PRE-COMPLETION CHECKLIST:

[ ] Target file(s) modified correctly
[ ] No unintended changes (reviewed git diff)
[ ] Syntax check passes
[ ] All related tests pass
[ ] Manual verification confirms expected behavior
[ ] Commit message follows format
[ ] AI attribution included
[ ] No files outside scope were modified
[ ] No new files created (unless required)
[ ] No dependencies changed
[ ] No secrets or credentials in changes
[ ] User has been given completion report
```

### Pre-Commit Verification

```
[ ] Pre-edit validations passed
[ ] Post-edit validations passed
[ ] Git diff shows only expected changes
[ ] No commented-out code in commit
[ ] No debug/logging statements in commit
[ ] No whitespace-only changes
[ ] Files staged for commit are correct
[ ] Commit message follows convention
```

### Post-Commit Verification

```
[ ] git status shows clean working tree
[ ] git log shows commit created
[ ] git show HEAD confirms commit contents
[ ] No accidental commits of test files to production
[ ] No accidental commits of production credentials
```

---

## QUICK REFERENCE

```
+------------------------------------------------------------------+
|              AGENT EXECUTION QUICK REFERENCE                      |
+------------------------------------------------------------------+
| EXECUTION FLOW:                                                  |
|   1. Preparation → Verify preconditions → Execute → Validate     |
|   2. If any check fails → ROLLBACK → Report → Stop               |
|   3. If all checks pass → Commit → Report → Done                |
+------------------------------------------------------------------+
| ROLLBACK COMMANDS:                                               |
|   Uncommitted: git checkout HEAD -- <file>                       |
|   Committed:  git reset --soft HEAD~1                            |
|   Pushed:    git revert HEAD                                     |
+------------------------------------------------------------------+
| ERROR HANDLING:                                                  |
|   Syntax error → Rollback immediately                            |
|   Test failure → Rollback immediately                            |
|   Edit failed → Re-read and ask                                 |
|   Unknown error → Rollback and report                            |
+------------------------------------------------------------------+
| COMMIT FORMAT:                                                   |
|   <type>(<scope>): <description>                                |
|   Authored by TheArchitectit                            |
+------------------------------------------------------------------+
```

---

**Related Documents:**
- [AGENT_GUARDRAILS.md](../AGENT_GUARDRAILS.md) - Core safety protocols
- [TESTING_VALIDATION.md](./TESTING_VALIDATION.md) - Validation protocols
- [ROLLBACK_PROCEDURES.md](./ROLLBACK_PROCEDURES.md) - Recovery operations
- [COMMIT_WORKFLOW.md](./COMMIT_WORKFLOW.md) - Commit guidelines

---

**Last Updated:** 2026-01-16
**Authored by:** TheArchitectit
**Document Owner:** Project Maintainers
**Line Count:** ~280
