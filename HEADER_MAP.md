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

## TOC.md

| Line | Header |
|------|--------|
| 1 | # Template Contents (Table of Contents) |
| 8 | ## Quick Navigation |
| 19 | ## Root Files |
| 35 | ## Documentation Directory |
| 60 | ## GitHub Integration |
| 77 | ## Examples Directory |
| 91 | ## Document Purpose Quick Reference |
| 101 | ## Document Size Summary |
| 118 | ## Compliance Status |
| 122 | ## Quick Lookup |
| 142 | ## File Templates |

---

## AGENT_GUARDRAILS.md

| Line | Header |
|------|--------|
| 1 | # Agent Guardrails & Safety Protocols |
| 9 | ## Applicability |
| 27 | ## Purpose |
| 39 | ## CORE PRINCIPLES |
| 41 | ### The Four Laws of Agent Safety |
| 65 | ## SAFETY PROTOCOLS (MANDATORY) |
| 67 | ### Pre-Execution Checklist |
| 80 | ### Git Safety Rules |
| 95 | ### Code Safety Rules |
| 108 | ### Test/Production Separation Rules (MANDATORY) |
| 117 | ## GUARDRAILS |
| 119 | ### HALT CONDITIONS |
| 137 | ### FORBIDDEN ACTIONS |
| 190 | ### SCOPE BOUNDARIES |
| 215 | ## QUICK REFERENCE |
| 242 | ## RELATED DOCUMENTS |

---

## docs/standards/TEST_PRODUCTION_SEPARATION.md

| Line | Header |
|------|--------|
| 1 | # Test/Production Separation Standards |
| 9 | ## Overview |
| 16 | ## CORE MANDATORY RULES |
| 18 | ### The Three Laws of Test/Production Separation |
| 28 | ### Mandatory Pre-Code Checklist |
| 41 | ## ENVIRONMENT SEPARATION REQUIREMENTS |
| 43 | ### Database Separation |
| 57 | ### Service Separation |
| 82 | ### User Account Separation |
| 107 | ## CODE CREATION SEQUENCE |
| 109 | ### Mandatory Order of Operations |
| 131 | ## TEST CODE LABELING REQUIREMENTS |
| 133 | ### When to Label vs Remove |
| 144 | ### Labeling Standards |
| 159 | ## UNCERTAINTY HANDLING PROTOCOL |
| 161 | ### Mandatory Ask Triggers |
| 173 | ### Ask Template |
| 184 | ### Example Scenarios |
| 214 | ## VERIFICATION CHECKLISTS |
| 216 | ### Pre-Commit Verification |
| 228 | ### Pre-Push Verification |
| 243 | ### CI/CD Blocking Checks |
| 258 | ## EXAMPLES AND PATTERNS |
| 260 | ### Good Pattern: Environment-Specific Config |
| 281 | ### Good Pattern: Environment Loading |
| 293 | ### Anti-Pattern: Hardcoded Production URLs |
| 302 | ### Good Pattern: Environment Variable Loading |
| 313 | ## BLOCKING VIOLATIONS |
| 315 | ### Immediate Halt Conditions |
| 330 | ### Notification Protocol |
| 340 | ## QUICK REFERENCE |

---

## docs/workflows/AGENT_EXECUTION.md

| Line | Header |
|------|--------|
| 1 | # Agent Execution Protocol |
| 9 | ## Overview |
| 16 | ## EXECUTION PROTOCOL |
| 18 | ### Standard Task Flow |
| 43 | ### Decision Matrix |
| 51 | ## ROLLBACK PROCEDURES |
| 53 | ### Immediate Rollback (Uncommitted Changes) |
| 64 | ### Rollback After Commit (Not Pushed) |
| 75 | ### Rollback After Push (REQUIRES USER PERMISSION) |
| 87 | ### Database Rollback Considerations |
| 99 | ### Service Rollback Procedures |
| 111 | ## COMMIT MESSAGE FORMAT |
| 113 | ### Format Template |
| 121 | ### Commit Types |
| 130 | ### Good vs Bad Messages |
| 143 | ### Co-Author Attribution |
| 153 | ## ERROR HANDLING PROTOCOLS |
| 155 | ### Syntax Error After Edit |
| 163 | ### Test Failure After Edit |
| 173 | ### Edit Operation Failed |
| 182 | ### Unknown Error |
| 192 | ### Database Error |
| 206 | ### Service Error |
| 221 | ## VERIFICATION CHECKLIST |
| 223 | ### Before Marking Task Complete |
| 238 | ### Pre-Commit Verification |
| 247 | ### Post-Commit Verification |
| 257 | ## QUICK REFERENCE |

---

## docs/workflows/AGENT_ESCALATION.md

| Line | Header |
|------|--------|
| 1 | # Agent Escalation & Guidelines |
| 9 | ## Overview |
| 16 | ## AUDIT REQUIREMENTS |
| 18 | ### All Agents MUST Maintain Logs |
| 58 | ### Log Format Standard |
| 81 | ### Audit Log Storage |
| 92 | ## ESCALATION PROCEDURES |
| 94 | ### When to Escalate to Human |
| 108 | ### How to Escalate |
| 136 | ### Escalation Scenarios |
| 164 | ## AGENT-SPECIFIC GUIDELINES |
| 166 | | ### Universal Requirements (ALL LLMs and AI Agents) |
| 176 | ### By Category |
| 189 | ### Model Compatibility Note |
| 202 | ## COMPLIANCE |
| 204 | ### Acknowledgment |
| 212 | ### Reporting Violations |
| 225 | ### Violation Categories |
| 236 | ## QUICK REFERENCE |
| 242 | ## COMPLIANCE

---

## CHANGELOG.md

| Line | Header |
|------|--------|
| 1 | # Changelog |
| 8 | ## [Unreleased] |
| 12 | ## [1.5.0] - 2026-01-18 |
| 27 | ## [1.4.0] - 2026-01-16 |
| 41 | ## [1.3.0] - 2026-01-16 |
| 54 | ## [1.1.0] - 2026-01-15 |
| 61 | ## [1.0.0] - 2026-01-14 |
| 64 | ## Version Management |
| 76 | ## Links |

---

## SPRINT_TEMPLATE.md

| Line | Header |
|------|--------|
| 1 | # Sprint Task: [TITLE] |
| 12 | ## SAFETY PROTOCOLS (MANDATORY) |
| 14 | ### Pre-Execution Safety Checks |
| 24 | ### Guardrails Reference |
| 28 | ### MCP Checkpoint (Optional) |
| 39 | ## PROBLEM STATEMENT |
| 53 | ## SCOPE BOUNDARY |
| 70 | ## EXECUTION DIRECTIONS |
| 72 | ### Overview |
| 91 | ## STEP-BY-STEP EXECUTION |
| 93 | ### STEP 1: [Title] |
| 110 | ### STEP 2: [Title] |
| 132 | ### STEP 3: [Title] |
| 152 | ### DONE: Commit and Report |
| 195 | ## COMPLETION GATE (MANDATORY) |
| 200 | ### Validation Loop Rules |
| 217 | ### Core Validation Checklist |
| 235 | ### Language-Specific Validation Commands |
| 383 | ### CLI-Specific Notes |
| 403 | ### Validation Loop Template |
| 428 | ### Completion Report Template |
| 453 | ## ACCEPTANCE CRITERIA |
| 463 | ## ROLLBACK PROCEDURE |
| 478 | ## REFERENCE |
| 484 | ## QUICK REFERENCE CARD |

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

**Last Updated:** 2026-01-18
**Status:** Complete - all documents and headers accurately mapped
