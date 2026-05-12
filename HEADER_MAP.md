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
| 82 | ### Git Safety Rules |
| 97 | ### Code Safety Rules |
| 110 | ### Test/Production Separation Rules (MANDATORY) |
| 119 | ## GUARDRAILS |
| 121 | ### HALT CONDITIONS |
| 139 | ### FORBIDDEN ACTIONS |
| 192 | ### SCOPE BOUNDARIES |
| 217 | ## QUICK REFERENCE |
| 244 | ## RELATED DOCUMENTS |
| 248 | ### Regression Prevention |

---

## .guardrails/pre-work-check.md

| Line | Header |
|------|--------|
| 1 | # Pre-Work Regression Check |
| 7 | ## Quick Checklist |
| 19 | ## Active Failures Relevant to Current Work |
| 34 | ## Known Bug Patterns by Category |
| 63 | ## Prevention Rules in Effect |
| 78 | ## Files with Known Bug History |
| 92 | ## Required Verification Steps |
| 125 | ## When You Find a New Bug |
| 139 | ## Quick Commands Reference |
| 163 | ## Remember |

---

## docs/workflows/REGRESSION_PREVENTION.md

| Line | Header |
|------|--------|
| 1 | # Regression Prevention Protocol |
| 14 | ## Overview |
| 27 | ## Core Philosophy |
| 41 | ## Failure Registry |
| 72 | ## Using the Registry |
| 120 | ## Prevention Rules |
| 159 | ## Pre-Work Check Protocol |
| 192 | ## Regression Testing Requirements |
| 249 | ## CI/CD Integration |
| 285 | ## Review Protocol |
| 313 | ## Common Scenarios |
| 360 | ## Metrics and Success Criteria |
| 385 | ## Best Practices |
| 405 | ## Troubleshooting |
| 425 | ## Quick Reference |

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

## docs/standards/PROJECT_CONTEXT_TEMPLATE.md

| Line | Header |
|------|--------|
| 1 | # Project Context Template (Project Bible) |
| 9 | ## Overview |
| 17 | ## HOW TO USE THIS TEMPLATE |
| 26 | ## TEMPLATE START |
| 37 | ## 1. TECH STACK CONSTRAINTS (Hard Limits) |
| 39 | ### Primary Stack |
| 49 | ### Version Lock Directive |
| 56 | ### Package Manager |
| 64 | ## 2. CODING STYLE GUIDE (The "Vibe") |
| 66 | ### Naming Conventions |
| 77 | ### Export Patterns |
| 88 | ### Function Style |
| 100 | ### Comment Standards |
| 118 | ## 3. ARCHITECTURAL PATTERNS (Enforced) |
| 120 | ### Directory Structure |
| 140 | ### Barrel Pattern (MANDATORY) |
| 150 | ### Dependency Flow (One-Way Street) |
| 160 | ### File Size Limits |
| 171 | ## 4. FORBIDDEN PATTERNS (No-Go Zone) |
| 173 | ### TypeScript Forbidden |
| 189 | ### React Forbidden |
| 210 | ### Database Forbidden |
| 226 | ### Security Forbidden |
| 244 | ## 5. CHAIN OF THOUGHT MANDATE |
| 246 | ### Protocol: Plan Before Execution |
| 272 | ## 6. VALIDATION REQUIREMENTS |
| 274 | ### Before Committing |
| 285 | ### Code Review Checklist |
| 299 | ## 7. APPROVED DEPENDENCIES |
| 312 | ## QUICK REFERENCE CARD |
| 342 | ## EXAMPLE: Filled Template (Next.js Project) |

---

## docs/standards/ADVERSARIAL_TESTING.md

| Line | Header |
|------|--------|
| 1 | # Adversarial Testing Protocol (Breaker Agent) |
| 9 | ## Overview |
| 17 | ## THE BREAKER AGENT PERSONA |
| 19 | ### Agent Configuration |
| 37 | ### Breaker vs Builder Separation |
| 58 | ## ATTACK VECTOR CATEGORIES |
| 60 | ### 1. Input Validation Attacks |
| 62 | #### String Attacks |
| 74 | #### XSS (Cross-Site Scripting) Attacks |
| 94 | #### SQL Injection Attacks |
| 113 | #### Number Attacks |
| 125 | ### 2. Boundary Condition Attacks |
| 127 | #### Array/Collection Attacks |
| 140 | #### Object Attacks |
| 153 | ### 3. State-Based Attacks |
| 155 | #### Race Conditions |
| 173 | #### Session Attacks |
| 186 | ### 4. Resource Exhaustion Attacks |
| 188 | #### Memory Exhaustion |
| 203 | #### CPU Exhaustion (ReDoS) |
| 219 | ## FUZZ TESTING PROTOCOL |
| 221 | ### Automated Fuzzing Setup |
| 278 | ### Fuzz Test Directive |
| 302 | ## COMPONENT-SPECIFIC ATTACK CHECKLISTS |
| 304 | ### Form Component Attacks |
| 322 | ### API Endpoint Attacks |
| 340 | ### File Upload Attacks |
| 357 | ### Authentication Attacks |
| 376 | ## BREAKER AGENT PROMPT TEMPLATE |
| 418 | ## INTEGRATION WITH CI/CD |
| 420 | ### Automated Adversarial Tests |
| 457 | ### Blocking Gate |
| 477 | ## QUICK REFERENCE |

---

## docs/standards/DEPENDENCY_GOVERNANCE.md

| Line | Header |
|------|--------|
| 1 | # Dependency Governance |
| 9 | ## Overview |
| 17 | ## WHY DEPENDENCY GOVERNANCE |
| 19 | ### The Risks of Uncontrolled Dependencies |
| 45 | ## ALLOW-LIST STRUCTURE |
| 47 | ### Package Categories |
| 63 | ### Allow-List Template |
| 266 | ## AGENT DIRECTIVE |
| 268 | ### When Agent Wants to Add a Package |
| 300 | ### Forbidden Package Detection |
| 317 | ## VALIDATION WORKFLOW |
| 319 | ### Pre-Install Check |
| 349 | ### CI/CD Integration |
| 391 | ## MAINTENANCE |
| 393 | ### Adding New Packages |
| 420 | ### Removing Packages |
| 448 | ## QUICK REFERENCE |

---

## docs/standards/INFRASTRUCTURE_STANDARDS.md

| Line | Header |
|------|--------|
| 1 | # Infrastructure Standards (IaC) |
| 9 | ## Overview |
| 17 | ## THE NO-CLICKOPS MANDATE |
| 19 | ### Why ClickOps is Forbidden |
| 45 | ### The IaC Mandate |
| 66 | ## TERRAFORM STANDARDS |
| 68 | ### Directory Structure |
| 97 | ### Required File Structure |
| 153 | ## THE PLAN-BEFORE-APPLY PROTOCOL |
| 155 | ### Never Apply Without Plan Review |
| 190 | ### Agent IaC Directive |
| 216 | ## DRIFT DETECTION |
| 218 | ### What is Drift? |
| 235 | ### Drift Response Protocol |
| 256 | ### Automated Drift Detection |
| 297 | ## STATE FILE MANAGEMENT |
| 299 | ### State File Security |
| 316 | ### Backend Configuration |
| 331 | ### State File Agent Rules |
| 350 | ## RESOURCE NAMING CONVENTIONS |
| 352 | ### Standard Naming Pattern |
| 364 | ### Tagging Standards |
| 390 | ## SECURITY CONSTRAINTS |
| 392 | ### Forbidden Configurations |
| 420 | ### Required Security Controls |
| 437 | ## CI/CD INTEGRATION |
| 439 | ### Terraform CI Pipeline |
| 507 | ## QUICK REFERENCE |

---

## docs/standards/OPERATIONAL_PATTERNS.md

| Line | Header |
|------|--------|
| 1 | # Operational Patterns |
| 9 | ## Overview |
| 17 | ## HEALTH CHECK PATTERNS |
| 19 | ### The /health Endpoint |
| 58 | ### Health Check Implementation |
| 127 | ### Liveness vs Readiness |
| 162 | ## CIRCUIT BREAKER PATTERN |
| 164 | ### Why Circuit Breakers? |
| 188 | ### Circuit Breaker States |
| 215 | ### Circuit Breaker Implementation |
| 299 | ## RETRY PATTERNS |
| 301 | ### Exponential Backoff |
| 358 | ### Retry vs Circuit Breaker |
| 383 | ## GRACEFUL DEGRADATION |
| 385 | ### Fallback Strategies |
| 455 | ## RATE LIMITING |
| 457 | ### Token Bucket Implementation |
| 518 | ### Rate Limit Headers |
| 532 | ## TIMEOUT PATTERNS |
| 534 | ### Request Timeouts |
| 559 | ### Timeout Hierarchy |
| 576 | ## OBSERVABILITY |
| 578 | ### Metrics to Track |
| 604 | ### Structured Error Logging |
| 631 | ## QUICK REFERENCE |

---

## docs/workflows/AGENT_REVIEW_PROTOCOL.md

| Line | Header |
|------|--------|
| 1 | # Agent Review Protocol |
| 9 | ## Overview |
| 17 | ## WHY AGENT REVIEW IS MANDATORY |
| 19 | ### The Hallucination Problem |
| 33 | ### The Context Contamination Problem |
| 49 | ## REVIEW MODELS |
| 51 | ### Model 1: Dual-Agent Review (Recommended) |
| 74 | ### Model 2: Cross-Model Review |
| 99 | ### Model 3: Specialized Agent Review |
| 122 | ### Model 4: Automated + Agent Hybrid |
| 157 | ## REVIEWER AGENT PROMPTS |
| 159 | ### General Code Reviewer Prompt |
| 216 | ### Security-Focused Reviewer Prompt |
| 248 | ### Test Quality Reviewer Prompt |
| 283 | ### Architecture Reviewer Prompt |
| 320 | ## REVIEW WORKFLOW |
| 322 | ### Standard Review Flow |
| 365 | ### Review Package Template |
| 409 | ## REVIEW DECISION MATRIX |
| 411 | ### When to APPROVE |
| 428 | ### When to REQUEST_CHANGES |
| 446 | ### When to REJECT |
| 467 | ## REVIEW CYCLE LIMITS |
| 469 | ### Three Strikes Rule |
| 492 | ### Context Reset Between Cycles |
| 509 | ## AUTOMATION INTEGRATION |
| 511 | ### GitHub Actions Review Gate |
| 567 | ## QUICK REFERENCE |

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
| 143 | ### AI Attribution |
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

## docs/CLCODE_INTEGRATION.md

| Line | Header |
|------|--------|
| 1 | # Claude Code Integration |
| 5 | ## Overview |
| 13 | ## Setup |
| 15 | ### 1. Run Setup Script |
| 34 | ### 2. Verify Installation |
| 46 | ## How It Works |
| 48 | ### Skills |
| 66 | ### Hooks |
| 76 | ## Skill Details |
| 78 | ### guardrails-enforcer |
| 95 | ### commit-validator |
| 112 | ### env-separator |
| 126 | ## Customization |
| 128 | ### Adding a Custom Skill |
| 143 | ### Modifying Hooks |
| 159 | ## Advanced Configuration |
| 161 | ### Skill Selection |
| 172 | ### Hook Chaining |
| 190 | ## Troubleshooting |
| 192 | ### Skills Not Loading |
| 199 | ### Hooks Not Running |
| 206 | ### Permission Denied |
| 213 | ## Best Practices |
| 220 | ## References |

---

## docs/OPCODE_INTEGRATION.md

| Line | Header |
|------|--------|
| 1 | # OpenCode Integration |
| 5 | ## Overview |
| 13 | ## Setup |
| 15 | ### 1. Run Setup Script |
| 34 | ### 2. Verify Installation |
| 46 | ## How It Works |
| 48 | ### Agents |
| 61 | ### Skills |
| 74 | ### Hooks |
| 84 | ## Skill Details |
| 86 | ### guardrails-enforcer |
| 103 | ### commit-validator |
| 120 | ### env-separator |
| 134 | ## Customization |
| 136 | ### Adding a Custom Agent |
| 158 | ### Modifying Hooks |
| 174 | ## Advanced Configuration |
| 176 | ### Agent Selection |
| 187 | ### Hook Chaining |
| 205 | ## Troubleshooting |
| 207 | ### Agents Not Loading |
| 214 | ### Skills Not Loading |
| 221 | ### Hooks Not Running |
| 228 | ### Permission Denied |
| 235 | ## Best Practices |
| 242 | ## References |

---

---

## mcp-server/DEPLOYMENT_GUIDE.md

| Line | Header |
|------|--------|
| 1 | # Guardrail MCP Server Deployment Guide |
| 7 | ## Overview |
| 11 | ## Prerequisites |
| 15 | ## Deployment Summary |
| 19 | ## Quick Deploy |
| 41 | ### 1. Update AI01 IP in .env |
| 49 | ### 2. Build and Deploy |
| 95 | ## Detailed Deployment Steps |
| 99 | ### Step 1: Environment Setup |
| 119 | ### Step 2: Apply Schema Fix |
| 134 | ### Step 3: Build Docker Image |
| 160 | ### Step 4: Create Pod and Start Containers |
| 208 | ### Step 5: Verify Deployment |
| 222 | ## Configuration Requirements |
| 227 | ### Critical Settings |
| 232 | ### Environment Variables Reference |
| 298 | ## Docker Compose Configuration |
| 302 | ### Working Configuration (AI01 Deployment) |
| 370 | ### Common Pitfalls |
| 416 | ## Testing the Deployment |
| 420 | ### Test MCP Protocol |
| 458 | ### Test Guardrail Tools |
| 486 | ### Test Web UI |
| 500 | ## Troubleshooting Guide |
| 505 | ### Problem: Schema Validation Error |
| 530 | ### Problem: Postgres Permission Errors |
| 560 | ### Problem: Database Authentication Failed |
| 590 | ### Problem: Redis Connection Refused |
| 620 | ### Problem: Connection Timeout from Remote Machine |
| 650 | ### Problem: Container Exits Immediately |
| 680 | ### Problem: Ports Already in Use |
| 710 | ### Problem: YAML Syntax Errors in Compose File |
| 740 | ## Verification Checklist |
| 760 | ## Maintenance |
| 764 | ### Viewing Logs |
| 780 | ### Restarting Services |
| 800 | ### Updating Configuration |
| 820 | ### Backup and Restore |
| 840 | ## Production Hardening |
| 844 | ### Security Recommendations |
| 860 | ### Performance Tuning |
| 890 | ## OpenCode Configuration |
| 894 | ### MCP Server Configuration |
| 910 | ### Environment Variables |
| 930 | ## Troubleshooting |

---

## docs/PYTHON_TO_GO_MIGRATION.md

| Line | Header |
|------|--------|
| 1 | # Python to Go Migration Guide |
| 5 | > Complete guide for migrating from Python team_manager.py to Go team package |
| 8 | ## Overview |
| 13 | ## What Was Migrated |
| 17 | ## Developer Migration Guide |
| 37 | ### API Changes |
| 49 | ### Error Handling |
| 62 | ## Container Changes |
| 78 | ## Deployment Migration |
| 104 | ## Testing Changes |
| 123 | ## Performance Improvements |
| 133 | ## Backward Compatibility |
| 145 | ## Contributing |

---

## AI_ASSISTED_DEV.md

| Line | Header |
|------|--------|
| 1 | # AI-Assisted Development Patterns |
| 9 | ## Purpose |
| 17 | ## The Vibe Coding Workflow |
| 21 | ### The Speed Equation |
| 34 | ## Decision Matrix: Ask vs Decide vs Halt |
| 38 | ### Risk Level: LOW — Decide Autonomously |
| 48 | ### Risk Level: MEDIUM — Ask Before Proceeding |
| 58 | ### Risk Level: HIGH — Halt and Confirm |
| 70 | ## Design-Intent Preservation |
| 74 | ### Style Anchors |
| 108 | ### Intent Logs |
| 121 | ## Prompt-to-UI Scaffolding |
| 161 | ## Iteration Safety |
| 199 | ## Human Approval Gates |
| 214 | ## Design Tool Integration |
| 234 | ## HALT CONDITIONS |
| 251 | ## Language Patterns |
| 316 | ## RELATED DOCUMENTS |

---

## STATE_MANAGEMENT.md

| Line | Header |
|------|--------|
| 1 | # State Management & Data Patterns |
| 9 | ## Purpose |
| 17 | ## State Architecture Decision Tree |
| 31 | ## Client State Patterns |
| 33 | ### Local State (Single Component) |
| 55 | ### Global Client State (Shared Across Components) |
| 99 | ### Atomic State (Fine-Grained Reactivity) |
| 116 | ## Server State Patterns |
| 148 | ## Offline-First & Local Persistence |
| 180 | ## Real-Time & CRDT Collaboration |
| 205 | ## Forbidden Patterns |
| 218 | ## HALT CONDITIONS |
| 231 | ## Language Patterns |
| 296 | ## RELATED DOCUMENTS |

---

## GENERATIVE_ASSET_SAFETY.md

| Line | Header |
|------|--------|
| 1 | # Generative Asset Safety |
| 9 | ## Purpose |
| 17 | ## AI Content Disclosure |
| 19 | ### Mandatory Labeling |
| 32 | ### C2PA Metadata (Content Provenance) |
| 75 | ## Procedural Generation Guardrails |
| 77 | ### Seed Reproducibility |
| 102 | ### Output Bounding |
| 115 | ### Safety Filters |
| 140 | ## Asset Attribution |
| 183 | ## Synthetic Media Ethics |
| 207 | ## Content Filtering Pipeline |
| 229 | ## HALT CONDITIONS |
| 244 | ## Language Patterns |
| 324 | ## RELATED DOCUMENTS |

---

## MONETIZATION_GUARDRAILS.md

| Line | Header |
|------|--------|
| 1 | # Monetization & Economy Guardrails |
| 9 | ## Purpose |
| 17 | ## In-App Purchase (IAP) Ethics |
| 42 | ## Loot Box Transparency |
| 87 | ## Subscription Fairness |
| 100 | ## Virtual Economy Balance |
| 131 | ## Battle Pass Patterns |
| 154 | ## Age-Gated Spending |
| 184 | ## HALT CONDITIONS |
| 200 | ## Language Patterns |
| 255 | ## RELATED DOCUMENTS |

---

## MULTIPLAYER_SAFETY.md

| Line | Header |
|------|--------|
| 1 | # Multiplayer & Social Safety |
| 9 | ## Purpose |
| 17 | ## Presence & Social Graph |
| 41 | ## Matchmaking Fairness |
| 72 | ## Chat & Communication Moderation |
| 104 | ## Harassment Prevention |
| 128 | ## Content Moderation (User-Generated Content) |
| 161 | ## Trust & Safety Operations |
| 185 | ## HALT CONDITIONS |
| 201 | ## Language Patterns |
| 268 | ## RELATED DOCUMENTS |

---

## ANALYTICS_ETHICS.md

| Line | Header |
|------|--------|
| 1 | # Analytics & Telemetry Ethics |
| 9 | ## Purpose |
| 17 | ## Event Tracking Consent |
| 62 | ## Data Minimization |
| 101 | ## Behavioral Targeting Limits |
| 125 | ## A/B Testing Ethics |
| 166 | ## Algorithmic Transparency |
| 191 | ## HALT CONDITIONS |
| 207 | ## Language Patterns |
| 294 | ## RELATED DOCUMENTS |

---

## CROSS_PLATFORM_DEPLOYMENT.md

| Line | Header |
|------|--------|
| 1 | # Cross-Platform Deployment |
| 9 | ## Purpose |
| 17 | ## App Store Compliance Matrix |
| 73 | ## CI/CD for Games & Apps |
| 121 | ## Feature Flags |
| 156 | ## Progressive Rollout |
| 178 | ## HALT CONDITIONS |
| 193 | ## Language Patterns |
| 251 | ## RELATED DOCUMENTS |

---

## docs/game-design/3D_GAME_DEVELOPMENT.md

| Line | Header |
|------|--------|
| 1 | # 3D Game Development Guardrails |
| 9 | ## Purpose |
| 21 | ## Agent-3DDev-2026 Role Definition |
| 33 | ### AI-Optimized Development |
| 42 | ## CORE PRINCIPLES |
| 44 | ### The Four Laws of 3D Development |
| 53 | ## SAFETY PROTOCOLS (MANDATORY) |
| 55 | ### Pre-Implementation Checklist |
| 69 | ### Scope Enforcement Rules |
| 80 | ### Asset Pipeline Rules |
| 91 | ### LOD Thresholds |
| 100 | ### Texture Size Budgets |
| 110 | ## ENGINE CONVENTIONS (Godot 4.x) |
| 112 | ### Scene Architecture |
| 129 | ### Node Naming Conventions |

---

## docs/game-design/3D_GUARDREL_PROPOSALS_V1.2.md

| Line | Header |
|------|--------|
| 1 | # 3D Game Development Guardrails — v1.2 Proposed Additions |
| 9 | ## SUMMARY OF GAPS IDENTIFIED |
| 29 | ## NEW GUARDRAIL PROPOSALS |
| 31 | ### 1. NEURAL RENDERING GUARDRAILS |
| 45 | ### 2. AI ANIMATION & MOTION GUARDRAILS |
| 61 | ### 3. AI CODE GENERATION GUARDRAILS (CRITICAL) |
| 77 | ### 4. NEURAL PHYSICS GUARDRAILS (CRITICAL) |
| 93 | ### 5. AI QA/TESTING/BALANCE GUARDRAILS |
| 109 | ### 6. RUNTIME INFERENCE GUARDRAILS (CRITICAL) |
| 125 | ### 7. AI NPC & DIALOGUE GUARDRAILS (CRITICAL) |
| 141 | ### 8. WORLD/LEVEL GENERATION GUARDRAILS |
| 151 | ### 9. BUSINESS & VENDOR GUARDRAILS |
| 161 | ### 10. LEGAL/ETHICAL ENHANCEMENTS |
| 173 | ### 11. SOCIAL/ETHICAL GUARDRAILS |
| 183 | ## INTEGRATION PLAN |

---

## docs/game-design/3D_MATHEMATICAL_FOUNDATIONS.md

| Line | Header |
|------|--------|
| 1 | # 3D Mathematical Foundations for Game Development |
| 9 | ## 1. Coordinate Systems and Points |
| 13 | ### Handedness |
| 25 | ### Points vs. Vectors |
| 32 | ### Homogeneous Coordinates |
| 41 | ## 2. Vectors: The Engine of Movement |
| 45 | ### Vector Normalization |
| 65 | ### The Dot Product |
| 87 | ### The Cross Product |
| 107 | ## 3. Transformations: Matrix Mathematics |
| 113 | ### The Identity Matrix |
| 124 | ### Translation Matrix |
| 135 | ### Scaling Matrix |
| 146 | ### Rotation Matrix |
| 157 | ### Matrix Concatenation |

---

## docs/game-design/3D_MODULE_ARCHITECTURE.md

| Line | Header |
|------|--------|
| 1 | # 3D Game Design Module Architecture |
| 9 | ## Overview |
| 15 | ## 1. Architectural Expansion of .guardrails/ for 3D Environments |
| 19 | ### A. 3D Asset and Geometry Generation |
| 36 | ### B. Material and Shader Constraints |
| 51 | ### C. Physics and Spatial Logic Safety |
| 71 | ## 2. Extending the Go MCP Server |
| 75 | ### A. Spatial Analysis Tools |
| 90 | ### B. Scene Graph Traversal |
| 103 | ## 3. Engine-Specific Engineering Standards |
| 107 | ### A. Unity (C#) Standards |
| 120 | ### B. Unreal Engine (C++) Standards |
| 131 | ### C. Godot 4 (GDScript / C++) Standards |
| 143 | ## 4. 3D Accessibility Standards |
| 164 | ## 5. "Vibe Coding" & Shared Prompts |

---

## docs/game-design/AI_DEBUGGABLE_3D_ARCHITECTURE.md

| Line | Header |
|------|--------|
| 1 | # AI-Debuggable 3D Game Architecture |
| 9 | ## The Core Problem |
| 19 | ## 1. The Death of Deep Inheritance: Embracing ECS |
| 29 | ### Composition Over Inheritance: Entity Component System |
| 46 | ### The AI Debugging Advantage |
| 58 | ## 2. Decoupled State and Dependency Injection |
| 64 | ### Implementing Dependency Injection (DI) |
| 101 | ## 3. Designing for AI Observability (Semantic Telemetry) |
| 105 | ### Headless Execution and State Dumping |
| 137 | ## 4. The Spatial Query API (MCP Integration) |
| 141 | ### Required MCP Endpoints |
| 168 | ## 5. Deterministic Execution |
| 172 | ### Fixed Time Steps |
| 182 | ### Seeded Randomness |
| 194 | ## 6. Defensive Coding and Assertions |

---

## docs/game-design/AI_DEV_2026_COMPLETE_GUIDE.md

| Line | Header |
|------|--------|
| 1 | # AI-Powered Development in 2026: From Intro to Master |
| 3 | ## A Comprehensive Guide for the Modern Developer |
| 10 | # Chapter 1: The AI Development Landscape in 2026 |
| 12 | ## The Transformation Is Complete |
| 20 | ## The Three-Year Revolution: 2023 to 2026 |
| 22 | ### 2023: The Copilot Era |
| 28 | ### 2024: The Context Window Wars |
| 36 | ### 2025: The Agentic Breakthrough |
| 44 | ### 2026: The Orchestration Layer |
| 50 | ## The 2026 Tool Ecosystem |
| 54 | ### AI-Native IDEs |
| 64 | ### Agent-First Interfaces |
| 74 | ### Orchestration and Multi-Agent Platforms |
| 86 | ### Specialized Tools |
| 96 | ## The New Developer Role |

---

## docs/game-design/HERMES_2026_AI_DOSSIER.md

| Line | Header |
|------|--------|
| 1 | # AI IN 3D GAME DEVELOPMENT: THE 2026 DOSSIER |
| 2 | ## A Comprehensive Intelligence Report |
| 11 | ## TABLE OF CONTENTS |
| 32 | ## 1. EXECUTIVE SUMMARY |
| 60 | ## 2. AI-POWERED 3D ASSET GENERATION |
| 62 | ### 2.1 The State of the Art in Text-to-3D |
| 90 | ### 2.2 Architecture of Modern Text-to-3D Systems |
| 116 | ### 2.3 Photogrammetry and Neural Capture |
| 134 | ### 2.4 Commercial Rights and Licensing |
| 144 | ### 2.5 Quality Benchmarks and Limitations |
| 169 | ## 3. GAME ENGINE AI INTEGRATION |
| 171 | ### 3.1 Unity 6 and the Sentis/Muse Stack |
| 194 | ### 3.2 Unreal Engine 5.5/6 and Epic's AI Trajectory |
| 224 | ### 3.3 Godot 4.x and Open-Source AI Integration |
| 241 | ### 3.4 NVIDIA Omniverse and the OpenUSD Ecosystem |

---

**Authored by:** TheArchitectit
**Document Owner:** Project Maintainers
**Last Updated:** 2026-05-12
**Status:** Complete - all documents and headers accurately mapped
