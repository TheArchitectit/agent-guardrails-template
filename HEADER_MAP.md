# Header Map - All Sections Across All Documents

> **Quick Section Lookup** - Find exact sections without reading full files.
> Format: `file_path:line_number` → Header

---

## How to Use This Map

1. Find the document you need in the index below
2. Locate the specific header/section
3. Use the line number to read only that section:
   ```
   Read file_path with offset=line_number, limit=50
   ```

---

## AGENT_GUARDRAILS.md

| Line | Header |
|------|--------|
| 1 | # Agent Guardrails & Safety Protocols |
| 9 | ## Applicability |
| 26 | ## Purpose |
| 38 | ## CORE PRINCIPLES |
| 40 | ### The Four Laws of Agent Safety |
| 64 | ## SAFETY PROTOCOLS (MANDATORY) |
| 66 | ### Pre-Execution Checklist |
| 79 | ### Git Safety Rules |
| 94 | ### Code Safety Rules |
| 109 | ## GUARDRAILS |
| 111 | ### HALT CONDITIONS |
| 134 | ### FORBIDDEN ACTIONS |
| 179 | ### SCOPE BOUNDARIES |
| 203 | ## EXECUTION PROTOCOL |
| 205 | ### Standard Task Flow |
| 240 | ### Decision Matrix |
| 254 | ## ROLLBACK PROCEDURES |
| 256 | ### Immediate Rollback (Uncommitted Changes) |
| 269 | ### Rollback After Commit (Not Pushed) |
| 282 | ### Rollback After Push (REQUIRES USER PERMISSION) |
| 296 | ## COMMIT MESSAGE FORMAT |
| 308 | ### Commit Types |
| 321 | ### Co-Author Attribution |
| 333 | ## ERROR HANDLING PROTOCOLS |
| 335 | ### Syntax Error After Edit |
| 345 | ### Test Failure After Edit |
| 356 | ### Edit Operation Failed |
| 367 | ### Unknown Error |
| 379 | ## VERIFICATION CHECKLIST |
| 402 | ## AGENT-SPECIFIC GUIDELINES |
| 404 | ### For Claude (Anthropic) |
| 414 | ### For GPT Models (OpenAI) |
| 423 | ### For Gemini (Google) |
| 432 | ### For Open Source Models (LLaMA, Mistral, etc.) |
| 441 | ### For Autonomous Agents (CrewAI, LangChain, etc.) |
| 453 | ## AUDIT REQUIREMENTS |
| 480 | ## ESCALATION PROCEDURES |
| 482 | ### When to Escalate to Human |
| 496 | ### How to Escalate |
| 508 | ## QUICK REFERENCE CARD |
| 542 | ## COMPLIANCE |
| 544 | ### Acknowledgment |
| 552 | ### Reporting Violations |
| 563 | ## RELATED DOCUMENTS |

---

## SPRINT_TEMPLATE.md

| Line | Header |
|------|--------|
| 1 | # Sprint Task: [TITLE] |
| 12 | ## SAFETY PROTOCOLS (MANDATORY) |
| 14 | ### Pre-Execution Safety Checks |
| 24 | ### Guardrails Reference |
| 30 | ## PROBLEM STATEMENT |
| 44 | ## SCOPE BOUNDARY |
| 61 | ## EXECUTION DIRECTIONS |
| 63 | ### Overview |
| 82 | ## STEP-BY-STEP EXECUTION |
| 84 | ### STEP 1: [Title] |
| 101 | ### STEP 2: [Title] |
| 123 | ### STEP 3: [Title] |
| 143 | ### DONE: Report to User |
| 173 | ## ACCEPTANCE CRITERIA |
| 183 | ## ROLLBACK PROCEDURE |
| 198 | ## REFERENCE |
| 204 | ## QUICK REFERENCE CARD |

---

## SPRINT_GUIDE.md

| Line | Header |
|------|--------|
| 1 | # Sprint Documentation Guide |
| 8 | ## Purpose |
| 14 | ## When to Create a Sprint Document |
| 30 | ## Sprint Document Structure |
| 32 | ### Required Sections |
| 65 | ### Optional Sections |
| 76 | ## Writing Effective Steps |
| 78 | ### Good Step Example |
| 105 | ### Bad Step Example (Avoid) |
| 122 | ## Key Principles |
| 124 | ### 1. Be Explicit About Everything |
| 131 | ### 2. Provide Exact Code |
| 138 | ### 3. Include Decision Points |
| 145 | ### 4. Define Scope Clearly |
| 158 | ### 5. Make Rollback Easy |
| 167 | ## Naming Convention |
| 179 | ## Archive Policy |
| 190 | ## Priority Levels |
| 201 | ## Status Values |
| 213 | ## Checklist for Sprint Authors |
| 234 | ## Example: Minimal Sprint |
| 262 | ## Template Quick Copy |

---

## CLAUDE.md

| Line | Header |
|------|--------|
| 1 | # Project Guidelines |
| 3 | ## 1. Context & Setup |
| 7 | ## 2. Token-Saving Rules (STRICT) |
| 13 | ## 3. Workflow |

---

## docs/sprints/INDEX.md

| Line | Header |
|------|--------|
| 1 | # Sprint Index |
| 5 | ## Quick Links |
| 11 | ## Active Sprints |
| 19 | ## Archived Sprints |
| 25 | ## Creating a New Sprint |

---

## docs/workflows/INDEX.md (TO BE CREATED)

| Line | Header |
|------|--------|
| 1 | # Workflow Documentation Index |
| - | ## Overview |
| - | ## Quick Reference Table |
| - | ## Document Summaries |
| - | ## Integration with Guardrails |

---

## docs/workflows/TESTING_VALIDATION.md (TO BE CREATED)

| Line | Header |
|------|--------|
| 1 | # Testing & Validation Protocols |
| - | ## Overview |
| - | ## Validation Function Patterns |
| - | ### Pre-Edit Validation |
| - | ### Post-Edit Validation |
| - | ## Git Diff Verification Patterns |
| - | ### Reviewing Changes Before Commit |
| - | ### Double-Check Verification Protocol |
| - | ## Validation Checklists |
| - | ## Language-Specific Validation |
| - | ## Error Handling |

---

## docs/workflows/COMMIT_WORKFLOW.md (TO BE CREATED)

| Line | Header |
|------|--------|
| 1 | # Commit Workflow Guidelines |
| - | ## Overview |
| - | ## When to Commit |
| - | ### Commit Decision Matrix |
| - | ### After Each To-Do Rule |
| - | ## Commit Frequency Patterns |
| - | ## Commit Message Standards |
| - | ## Pre-Commit Checklist |
| - | ## Commit Verification |
| - | ## Integration with To-Do Lists |

---

## docs/workflows/DOCUMENTATION_UPDATES.md (TO BE CREATED)

| Line | Header |
|------|--------|
| 1 | # Documentation Update Procedures |
| - | ## Overview |
| - | ## Post-Sprint Documentation Updates |
| - | ## Documentation Review Checklist |
| - | ## Documentation Templates |
| - | ## Version Control for Docs |
| - | ## Cross-Reference Maintenance |

---

## docs/workflows/GIT_PUSH_PROCEDURES.md (TO BE CREATED)

| Line | Header |
|------|--------|
| 1 | # Git Push Procedures |
| - | ## Overview |
| - | ## Pre-Push Checklist |
| - | ## Push Decision Matrix |
| - | ## Standard Push Workflow |
| - | ## Branch-Specific Procedures |
| - | ## Push Safety Rules |
| - | ## Error Handling |
| - | ## Integration with CI/CD |

---

## docs/workflows/MCP_CHECKPOINTING.md (TO BE CREATED)

| Line | Header |
|------|--------|
| 1 | # MCP Auto Checkpoint Documentation |
| - | ## Overview |
| - | ## Checkpoint Concepts |
| - | ## Integrating with MCP Servers |
| - | ## Checkpoint Workflow |
| - | ## Checkpoint-Aware Sprint Design |
| - | ## Recovery Procedures |
| - | ## Configuration Templates |
| - | ## Best Practices |
| - | ## Troubleshooting |

---

## docs/workflows/BRANCH_STRATEGY.md (TO BE CREATED)

| Line | Header |
|------|--------|
| 1 | # Branch Strategy Guide |
| - | ## Overview |
| - | ## Branch Naming Conventions |
| - | ## Main/Master Protection Rules |
| - | ## Feature Branch Workflow |
| - | ## Hotfix Emergency Procedures |
| - | ## Release Branch Management |
| - | ## Merge vs Rebase Guidance |

---

## docs/workflows/CODE_REVIEW.md (TO BE CREATED)

| Line | Header |
|------|--------|
| 1 | # Code Review Process |
| - | ## Overview |
| - | ## Agent Self-Review Checklist |
| - | ## When to Request Human Review |
| - | ## Review Focus Areas |
| - | ## Review Comment Standards |
| - | ## Approval Requirements |
| - | ## Escalation Procedures |

---

## docs/workflows/ROLLBACK_PROCEDURES.md (TO BE CREATED)

| Line | Header |
|------|--------|
| 1 | # Rollback Procedures |
| - | ## Overview |
| - | ## Immediate Rollback (Uncommitted Changes) |
| - | ## Post-Commit Rollback (Not Pushed) |
| - | ## Post-Push Rollback (Requires Care) |
| - | ## Database Rollback Considerations |
| - | ## Service Rollback Procedures |
| - | ## Disaster Recovery Checklist |
| - | ## Communication Templates |

---

## docs/standards/INDEX.md (TO BE CREATED)

| Line | Header |
|------|--------|
| 1 | # Standards Documentation Index |
| - | ## Overview |
| - | ## Quick Reference Table |
| - | ## Document Summaries |
| - | ## Integration with Guardrails |

---

## docs/standards/MODULAR_DOCUMENTATION.md (TO BE CREATED)

| Line | Header |
|------|--------|
| 1 | # Modular Documentation Standards |
| - | ## Overview |
| - | ## The 500-Line Rule |
| - | ### Why 500 Lines? |
| - | ### How to Count Lines |
| - | ### Enforcement |
| - | ## Document Structure Standards |
| - | ## Breaking Up Large Documents |
| - | ## Directory Organization |
| - | ## Module Dependencies |
| - | ## Compliance Checklist |

---

## docs/standards/LOGGING_PATTERNS.md (TO BE CREATED)

| Line | Header |
|------|--------|
| 1 | # Logging Patterns for Agents |
| - | ## Overview |
| - | ## Array-Based Logging Pattern |
| - | ### Core Concept |
| - | ### Standard Log Entry Structure |
| - | ## Log Levels |
| - | ## Standard Log Categories |
| - | ## Log Array Management |
| - | ## Log Output Formats |
| - | ## Integration with Sprints |
| - | ## Anti-Patterns |

---

## docs/standards/LOGGING_INTEGRATION.md (TO BE CREATED)

| Line | Header |
|------|--------|
| 1 | # External Logging Integration Hooks |
| - | ## Overview |
| - | ## Integration Architecture |
| - | ## Standard Hook Interface |
| - | ## Supported Integration Types |
| - | ## Configuration Templates |
| - | ## Placeholder Implementations |
| - | ## Migration Path |
| - | ## Error Handling |
| - | ## Security Considerations |

---

## docs/standards/API_SPECIFICATIONS.md (TO BE CREATED)

| Line | Header |
|------|--------|
| 1 | # API Specification Standards |
| - | ## Overview |
| - | ## OpenAPI Overview and Use Cases |
| - | ## OpenSpec Overview and Use Cases |
| - | ## When to Use OpenAPI |
| - | ## When to Use OpenSpec |
| - | ## Hybrid Approach Guidance |
| - | ## Template Files |
| - | ## Validation Tools and Commands |

---

## .github/SECRETS_MANAGEMENT.md (TO BE CREATED)

| Line | Header |
|------|--------|
| 1 | # GitHub Secrets & Actions Management |
| - | ## Overview |
| - | ## GitHub Secrets Concepts |
| - | ## Setting Up Secrets |
| - | ## Naming Conventions |
| - | ## Accessing Secrets in Actions |
| - | ## Secret Rotation |
| - | ## Security Best Practices |
| - | ## Integration with Guardrails |
| - | ## Troubleshooting |

---

## .github/PULL_REQUEST_TEMPLATE.md

| Line | Header |
|------|--------|
| 1 | ## Summary |
| 5 | ## Related Issue |
| 9 | ## Type of Change |
| 17 | ## Checklist |
| 27 | ## Test Plan |
| 31 | ## Screenshots |

---

**Last Updated:** 2026-01-14
**Status:** Partial - line numbers will be updated as documents are created
**Note:** Lines marked with `-` are placeholders for documents not yet created
