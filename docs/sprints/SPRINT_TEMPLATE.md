# Sprint Task: [TITLE]

**Sprint Date:** YYYY-MM-DD (Day)
**Archive After:** YYYY-MM-DD (Day) [+7 days]
**Sprint Focus:** [One-line description of what this sprint accomplishes]
**Priority:** P0 (Critical) | P1 (Blocking) | P2 (Normal) | P3 (Low)
**Estimated Effort:** [X minutes/hours]
**Status:** PENDING | IN_PROGRESS | COMPLETE | BLOCKED | FAILED

---

## SAFETY PROTOCOLS (MANDATORY)

### Pre-Execution Safety Checks

| Check | Requirement | Verify |
|-------|-------------|--------|
| **READ FIRST** | NEVER edit a file without reading it first | [ ] |
| **SCOPE LOCK** | Only modify files explicitly in scope | [ ] |
| **NO FEATURE CREEP** | Do NOT add features or "improve" unrelated code | [ ] |
| **BACKUP AWARENESS** | Know the rollback command before editing | [ ] |
| **TEST BEFORE COMMIT** | All tests must pass before committing | [ ] |

### Guardrails Reference

Full guardrails: [docs/AGENT_GUARDRAILS.md](../AGENT_GUARDRAILS.md)

### MCP Checkpoint (Optional)

If using MCP checkpointing, create checkpoint before starting:
```
[MCP: create_checkpoint("sprint-YYYY-MM-DD-before-start")]
```

See [MCP_CHECKPOINTING.md](../workflows/MCP_CHECKPOINTING.md) for details.

---

## PROBLEM STATEMENT

[Describe the problem in 2-3 sentences. Include error messages if applicable.]

```
[Include error output or symptoms here]
```

**Why:** [Root cause explanation]

**Where:** [File and line numbers]

---

## SCOPE BOUNDARY

```
IN SCOPE (may modify):
  - File: [path/to/file.ext]
  - Lines: [X-Y]
  - Change: [Brief description]

OUT OF SCOPE (DO NOT TOUCH):
  - All other files
  - All other methods/functions
  - Tests (read-only for verification)
  - Documentation (this file excluded)
```

---

## EXECUTION DIRECTIONS

### Overview

```
TASK SEQUENCE:

  STEP 1: [Action] ──────────────────────► [Purpose]
       │
       ▼
  STEP 2: [Action] ──────────────────────► [Purpose]
       │
       ▼
  STEP 3: [Action] ──────────────────────► [Purpose]
       │
       ▼
  DONE: Report to user ──────────────────► Summary
```

---

## STEP-BY-STEP EXECUTION

### STEP 1: [Title]

**Action:** [Describe what to do]

```
TOOL: [Read | Edit | Bash | etc.]
[Tool-specific parameters]
```

**Checkpoint:** [What to verify before proceeding]

**Decision Point:**
- [ ] Success → Proceed to STEP 2
- [ ] Failure → HALT and report to user

---

### STEP 2: [Title]

**Action:** [Describe what to do]

```
TOOL: [Read | Edit | Bash | etc.]
[Tool-specific parameters]
```

**Checkpoint:** [What to verify]

**Decision Point:**
- [ ] Success → Proceed to STEP 3
- [ ] Failure → ROLLBACK and report

**Rollback Command (if needed):**
```bash
git checkout HEAD -- [file]
```

---

### STEP 3: [Title]

**Action:** [Describe what to do]

```
TOOL: [Read | Edit | Bash | etc.]
[Tool-specific parameters]
```

**Expected Output:**
```
[What successful output looks like]
```

**Decision Point:**
- [ ] Success → Proceed to DONE
- [ ] Failure → ROLLBACK and report

---

### DONE: Commit and Report

**COMMIT AFTER VALIDATION** (see [COMMIT_WORKFLOW.md](../workflows/COMMIT_WORKFLOW.md)):
```bash
git add <modified-files>
git commit -m "<type>(<scope>): <description>

AI-assisted: Claude Code and Opus"
```

**MCP Checkpoint (Optional):**
```
[MCP: create_checkpoint("sprint-YYYY-MM-DD-after-complete")]
```

**Action:** Provide completion summary

```
REPORT FORMAT:

## Sprint Complete: [Title]

**Status:** SUCCESS / FAILED
**File Modified:** [path]
**Lines Changed:** [X-Y]
**Commit Hash:** [hash]

### Changes Made:
- [Change 1]
- [Change 2]

### Verification Results:
- Syntax check: PASSED
- Unit tests: PASSED / SKIPPED
- Manual verification: X/X PASSED

### Next Steps:
- Review commit with: git show HEAD
- Push when ready with: git push origin [branch]
```

---

## ACCEPTANCE CRITERIA

| # | Criterion | Test | Pass Condition |
|---|-----------|------|----------------|
| 1 | [Criterion 1] | [How to test] | [What passes] |
| 2 | [Criterion 2] | [How to test] | [What passes] |
| 3 | [Criterion 3] | [How to test] | [What passes] |

---

## ROLLBACK PROCEDURE

```bash
# Immediate rollback - discard all changes
git checkout HEAD -- [file]

# Verify rollback
git status

# Report to user
echo "Rollback complete. File restored to original state."
```

---

## REFERENCE

[Optional: Include reference code, documentation links, or examples]

---

## QUICK REFERENCE CARD

```
+------------------------------------------------------------------+
|                    SPRINT QUICK REFERENCE                        |
+------------------------------------------------------------------+
| TARGET FILE:  [path/to/file]                                     |
| TARGET LINES: [X-Y]                                              |
| CHANGE TYPE:  [Brief description]                                |
+------------------------------------------------------------------+
| SAFETY:                                                          |
|   - Read before edit                                             |
|   - Single file only                                             |
|   - Test before commit                                           |
|   - No push without permission                                   |
+------------------------------------------------------------------+
| HALT IF:                                                         |
|   - Conditions don't match                                       |
|   - Any check fails                                              |
|   - Uncertain about anything                                     |
+------------------------------------------------------------------+
| ROLLBACK: git checkout HEAD -- [file]                            |
+------------------------------------------------------------------+
```

---

**Created:** YYYY-MM-DD
**Author:** [Name/Agent]
**Archive Date:** YYYY-MM-DD
**Version:** 1.0
