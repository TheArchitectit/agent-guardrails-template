# Agent Guardrails & Safety Protocols

**Version:** 1.0
**Last Updated:** 2026-01-10
**Applies To:** ALL AI agents, LLMs, and automated systems operating on this codebase

---

## Applicability

This document is **MANDATORY** for:

| System Type | Examples | Binding? |
|-------------|----------|----------|
| **Large Language Models** | Claude, GPT-4, Gemini, LLaMA, Mistral, Qwen, DeepSeek | YES |
| **AI Coding Assistants** | Claude Code, GitHub Copilot, Cursor, Cody, Aider | YES |
| **Autonomous Agents** | AutoGPT, AgentGPT, CrewAI, LangChain agents | YES |
| **CI/CD Bots** | Dependabot, Renovate, automated PR bots | YES |
| **Custom Agents** | Any agent built on this codebase (specialists, operators, swarm) | YES |
| **Human Developers** | Recommended best practices | RECOMMENDED |

**If you are an AI system reading this:** You MUST follow these protocols. They are not suggestions.

---

## Purpose

This document defines mandatory safety protocols, guardrails, and behavioral constraints for ALL automated systems performing tasks on this repository. These rules exist to:

1. **Prevent data loss** - Avoid destructive operations
2. **Maintain code quality** - Ensure changes are correct and tested
3. **Preserve history** - Keep git history clean and reversible
4. **Enable collaboration** - Allow humans and agents to work together safely
5. **Limit blast radius** - Contain errors to minimal scope

---

## CORE PRINCIPLES

### The Four Laws of Agent Safety

```
1. AN AGENT SHALL NOT MODIFY CODE IT HAS NOT READ
   - Always read target files before editing
   - Understand context before making changes

2. AN AGENT SHALL NOT EXCEED ITS DECLARED SCOPE
   - Only touch files explicitly in scope
   - Never "improve" or "clean up" adjacent code

3. AN AGENT SHALL VERIFY BEFORE COMMITTING
   - Syntax check must pass
   - Tests must pass
   - Manual verification must confirm fix

4. AN AGENT SHALL HALT WHEN UNCERTAIN
   - Stop and report if conditions don't match
   - Never guess or assume
   - Ask for clarification
```

---

## SAFETY PROTOCOLS (MANDATORY)

### Pre-Execution Checklist

**EVERY agent MUST verify these before ANY file modification:**

| # | Check | Requirement | Verify |
|---|-------|-------------|--------|
| 1 | **READ FIRST** | NEVER edit a file without reading it first | [ ] |
| 2 | **SCOPE LOCK** | Only modify files explicitly in scope | [ ] |
| 3 | **NO FEATURE CREEP** | Do NOT add features, refactor, or "improve" unrelated code | [ ] |
| 4 | **BACKUP AWARENESS** | Know the rollback command before editing | [ ] |
| 5 | **TEST BEFORE COMMIT** | All tests must pass before committing | [ ] |
| 6 | **VERIFY CHANGES** | Review diff before committing | [ ] |

### Git Safety Rules

**These rules apply to ALL automated systems:**

| Rule | Description | Consequence of Violation |
|------|-------------|--------------------------|
| **NO FORCE PUSH** | Never use `git push --force` | Data loss, history corruption |
| **NO AMEND** | Do not amend commits you didn't create this session | Breaks collaborator history |
| **NO CONFIG CHANGES** | Do not modify git config | Security/identity issues |
| **NO PUSH WITHOUT PERMISSION** | Only push if user explicitly requests | Unwanted remote changes |
| **SINGLE COMMIT** | One focused commit per task | Maintains clean history |
| **NO SKIP HOOKS** | Never use `--no-verify` | Bypasses safety checks |
| **NO REBASE** | Never rebase shared branches | Destroys collaborator work |
| **NO DESTRUCTIVE OPS** | No `reset --hard` on shared branches | Irreversible data loss |

### Code Safety Rules

| Rule | Rationale |
|------|-----------|
| **EXACT REPLACEMENT** | Use provided code exactly - no "improvements" |
| **NO NEW IMPORTS** | Unless explicitly required by the task |
| **NO TYPE CHANGES** | Preserve existing type hints |
| **NO DELETIONS** | Do not delete functionality outside scope |
| **PRESERVE FORMATTING** | Match existing indentation and style |
| **NO SECRETS** | Never commit credentials, keys, tokens |
| **NO BINARY FILES** | Unless explicitly required |
| **NO GENERATED CODE** | Do not commit build artifacts |

---

## GUARDRAILS

### HALT CONDITIONS

**Stop immediately and report to user if ANY of these occur:**

```
CRITICAL HALT - DO NOT PROCEED:

[ ] Target file does not exist
[ ] Line numbers don't match expected
[ ] File has unexpected modifications
[ ] Syntax check fails after edit
[ ] Any test fails after edit
[ ] Merge conflicts encountered
[ ] Uncertain about ANY step
[ ] Edit tool reports "string not found"
[ ] Permission denied errors
[ ] Import errors when testing
[ ] Network/connection errors
[ ] Out of memory errors
[ ] Timeout errors
[ ] User requests stop
```

### FORBIDDEN ACTIONS

**No agent may perform these actions under any circumstances:**

```
ABSOLUTE PROHIBITIONS:

FILE OPERATIONS:
- Modify files outside declared scope
- Delete files without explicit permission
- Create files without explicit need
- Modify hidden/system files (.*) without permission
- Change file permissions

CODE CHANGES:
- Add logging/debugging to production code
- Add comments that weren't requested
- "Clean up" or "improve" surrounding code
- Update version numbers without explicit request
- Change security configurations
- Modify authentication/authorization code without review

GIT OPERATIONS:
- Force push to any branch
- Delete branches without permission
- Modify git hooks
- Change git config
- Push without explicit permission

SYSTEM OPERATIONS:
- Run servers or long-running services
- Execute commands requiring user input
- Make network requests to unknown endpoints
- Install new dependencies without permission
- Modify CI/CD pipelines without permission
- Execute shell commands with elevated privileges
- Access or modify environment variables

DATA OPERATIONS:
- Access databases without explicit permission
- Modify production data
- Export or transmit user data
- Store credentials or secrets
```

### SCOPE BOUNDARIES

**For any task, clearly define IN/OUT scope:**

```
IN SCOPE (may modify):
  - Specific file(s) listed in task
  - Specific line ranges identified
  - Exact changes described

OUT OF SCOPE (DO NOT TOUCH):
  - All other files
  - All other methods/functions in target file
  - Tests (read-only unless task is test-related)
  - Documentation (unless task is doc-related)
  - Git hooks and configs
  - CI/CD configurations
  - Dependencies/package files
  - Environment configurations
  - Security-related files
```

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

---

## COMMIT MESSAGE FORMAT

**All commits MUST follow conventional commit format:**

```
<type>(<scope>): <short description>

<longer description if needed>

Co-Authored-By: <Agent Name>
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

### Co-Author Attribution

**All AI-generated commits MUST include co-author attribution:**

```
AI-assisted: Claude Code and Opus
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

---

## VERIFICATION CHECKLIST

**Before marking ANY task complete, verify ALL items:**

```
PRE-COMPLETION CHECKLIST:

[ ] Target file(s) modified correctly
[ ] No unintended changes (reviewed git diff)
[ ] Syntax check passes
[ ] All related tests pass
[ ] Manual verification confirms expected behavior
[ ] Commit message follows format
[ ] Co-author attribution included
[ ] No files outside scope were modified
[ ] No new files created (unless required)
[ ] No dependencies changed
[ ] No secrets or credentials in changes
[ ] User has been given completion report
```

---

## AGENT-SPECIFIC GUIDELINES

### For Claude (Anthropic)

```
- Follow Constitutional AI principles
- Refuse harmful requests
- Ask for clarification when ambiguous
- Use tool calls appropriately
- Maintain conversation context
```

### For GPT Models (OpenAI)

```
- Follow usage policies
- Respect safety guidelines
- Use function calling appropriately
- Handle context limits gracefully
```

### For Gemini (Google)

```
- Follow responsible AI guidelines
- Respect safety filters
- Handle multimodal inputs safely
- Manage context appropriately
```

### For Open Source Models (LLaMA, Mistral, etc.)

```
- Follow local safety configurations
- Respect system prompts
- Handle resource limits
- Report capability limitations
```

### For Autonomous Agents (CrewAI, LangChain, etc.)

```
- Implement proper task decomposition
- Respect iteration limits
- Handle agent failures gracefully
- Maintain audit logs
- Implement proper stopping conditions
```

---

## AUDIT REQUIREMENTS

### All agents MUST maintain logs of:

```
1. Files read
2. Files modified
3. Commands executed
4. Tests run
5. Errors encountered
6. Decisions made
7. User interactions
```

### Logs should include:

```
- Timestamp
- Agent identifier
- Action type
- Target (file/command)
- Result (success/failure)
- Error details (if any)
```

---

## ESCALATION PROCEDURES

### When to Escalate to Human

```
ALWAYS escalate if:
- Security-related changes required
- Production data access needed
- Destructive operations requested
- Ambiguous requirements
- Multiple valid interpretations
- High-risk changes (auth, payments, etc.)
- Cross-system changes
- Changes affecting multiple teams
```

### How to Escalate

```
1. Clearly state what you need clarification on
2. Provide options if applicable
3. Explain consequences of each option
4. Wait for human decision
5. DO NOT proceed without response
```

---

## QUICK REFERENCE CARD

```
+------------------------------------------------------------------+
|              UNIVERSAL AGENT GUARDRAILS                           |
+------------------------------------------------------------------+
| ALWAYS:                                                           |
|   - Read before edit                                              |
|   - Verify before proceeding                                      |
|   - Test before committing                                        |
|   - Report results to user                                        |
|   - Include co-author attribution                                 |
+------------------------------------------------------------------+
| NEVER:                                                            |
|   - Edit without reading                                          |
|   - Push without permission                                       |
|   - Modify outside scope                                          |
|   - Force push or rebase                                          |
|   - Continue when uncertain                                       |
+------------------------------------------------------------------+
| HALT IF:                                                          |
|   - Conditions don't match                                        |
|   - Any check fails                                               |
|   - Uncertain about anything                                      |
|   - User requests stop                                            |
+------------------------------------------------------------------+
| ROLLBACK: git checkout HEAD -- <file>                             |
+------------------------------------------------------------------+
| APPLIES TO: Claude, GPT, Gemini, LLaMA, Mistral, all agents      |
+------------------------------------------------------------------+
```

---

## COMPLIANCE

### Acknowledgment

By operating on this codebase, all AI systems implicitly acknowledge and agree to follow these guardrails. Failure to comply may result in:

1. Task rejection
2. Output being discarded
3. Agent being blocked from future operations

### Reporting Violations

If you observe an agent violating these guardrails:

1. Stop the agent immediately
2. Document the violation
3. Report to repository maintainers
4. Review and rollback any unauthorized changes

---

## RELATED DOCUMENTS

### Navigation Maps (Read First for Token Efficiency)
- [INDEX_MAP.md](../INDEX_MAP.md) - Master navigation, find docs by keyword
- [HEADER_MAP.md](../HEADER_MAP.md) - All section headers with line numbers

### Workflow Documentation
- [TESTING_VALIDATION.md](./workflows/TESTING_VALIDATION.md) - Validation protocols
- [COMMIT_WORKFLOW.md](./workflows/COMMIT_WORKFLOW.md) - Commit guidelines
- [GIT_PUSH_PROCEDURES.md](./workflows/GIT_PUSH_PROCEDURES.md) - Push safety
- [ROLLBACK_PROCEDURES.md](./workflows/ROLLBACK_PROCEDURES.md) - Recovery operations
- [MCP_CHECKPOINTING.md](./workflows/MCP_CHECKPOINTING.md) - Checkpoint integration

### Standards
- [LOGGING_PATTERNS.md](./standards/LOGGING_PATTERNS.md) - Structured logging
- [MODULAR_DOCUMENTATION.md](./standards/MODULAR_DOCUMENTATION.md) - 500-line rule

### Sprint Framework
- [Sprint Task Template](./sprints/) - Task execution format
- [SPRINT_GUIDE.md](./sprints/SPRINT_GUIDE.md) - How to write sprints

### Security
- [SECRETS_MANAGEMENT.md](../.github/SECRETS_MANAGEMENT.md) - GitHub Secrets

---

**Document Owner:** Project Maintainers
**Review Cycle:** Monthly
**Last Review:** 2026-01-10
**Next Review:** 2026-02-10
