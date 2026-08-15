# Template Contents (Table of Contents)

> Complete list of all files and directories in the Agent Guardrails Template.

---

## Quick Navigation

- [Root Files](#root-files)
- [Documentation Directory](#documentation-directory)
- [GitHub Integration](#github-integration)
- [Examples Directory](#examples-directory)

---

## Root Files

| File | Lines | Required? | Purpose |
|------|-------|-----------|---------|
| **README.md** | 359 | YES | Project overview and quick start |
| **INDEX_MAP.md** | 395 | YES | Master navigation - find docs by keyword |
| **HEADER_MAP.md** | 1033 | YES | Section headers with line numbers |
| **TOC.md** | 280 | YES | Complete file listing and organization |
| **CLAUDE.md** | 59 | Recommended | Optimized context for Claude Code CLI |
| **CONTRIBUTING.md** | 370 | YES | How to contribute |
| **.claudeignore** | ~20 | Recommended | Token-saving ignore rules |
| **CHANGELOG.md** | 1034 | YES | Release notes archive |
| **LICENSE** | - | YES | BSD-3-Clause license |
| **.gitignore** | ~60 | Recommended | Common ignore patterns |

---

## Documentation Directory

### Root Documentation (`docs/`)

| File | Lines | Sections | Purpose |
|------|-------|----------|---------|
| **AGENT_GUARDRAILS.md** | 319 | 13 | Core safety protocols (MANDATORY) |
| **HOW_TO_APPLY.md** | 434 | 5 | How to apply template with example prompts |
| **QUICK_SETUP.md** | 344 | 5 | 5-minute setup guide |
| **STATUS.md** | 130 | 4 | Project status and version |
| **AGENTS_AND_SKILLS_SETUP.md** | ~200 | 6 | Setup guide for Claude Code/OpenCode |
| **CLCODE_INTEGRATION.md** | ~250 | 7 | Claude Code skills and hooks integration |
| **OPENCODE_INTEGRATION.md** | ~300 | 8 | OpenCode agents and skills integration |
| **PYTHON_TO_GO_MIGRATION.md** | ~350 | 11 | Python to Go migration guide (v2.6.0) |

### Workflows (`docs/workflows/`)

| File | Lines | Key Sections | Purpose |
|------|-------|--------------|---------|
| **INDEX.md** | 151 | 5 | Workflow navigation hub |
| **AGENT_EXECUTION.md** | 546 | 6 | Execution protocol and rollback |
| **AGENT_ESCALATION.md** | 415 | 6 | Audit requirements and escalation |
| **AGENT_REVIEW_PROTOCOL.md** | 638 | 12 | Post-work agent/LLM review |
| **TESTING_VALIDATION.md** | 304 | 9 | Validation protocols and checks |
| **COMMIT_WORKFLOW.md** | 333 | 8 | Commit timing and message format |
| **DOCUMENTATION_UPDATES.md** | 302 | 5 | Post-sprint doc updates |
| **GIT_PUSH_PROCEDURES.md** | 324 | 8 | Push safety and verification |
| **BRANCH_STRATEGY.md** | 341 | 6 | Git branching conventions |
| **CODE_REVIEW.md** | 359 | 7 | Code review and escalation |
| **MCP_CHECKPOINTING.md** | 372 | 7 | MCP server checkpointing |
| **REGRESSION_PREVENTION.md** | 536 | 10 | Bug tracking and regression prevention |

**Total:** 12 workflow documents (INDEX.md + 11 guides)

### Standards (`docs/standards/`)

| File | Lines | Key Sections | Purpose |
|------|-------|--------------|---------|
| **INDEX.md** | 128 | 4 | Standards navigation hub |
| **TEST_PRODUCTION_SEPARATION.md** | 559 | 12 | Test/production isolation (MANDATORY) |
| **PROJECT_CONTEXT_TEMPLATE.md** | 376 | 9 | Project Bible - stack, style, forbidden patterns |
| **ADVERSARIAL_TESTING.md** | 510 | 12 | Breaker agent, fuzz testing, attack checklists |
| **DEPENDENCY_GOVERNANCE.md** | 483 | 8 | Package allow-list, license compliance |
| **INFRASTRUCTURE_STANDARDS.md** | 546 | 11 | IaC, Terraform, no-ClickOps, drift detection |
| **OPERATIONAL_PATTERNS.md** | 667 | 12 | Health checks, circuit breakers, retry, rate limiting |
| **MODULAR_DOCUMENTATION.md** | 331 | 8 | 500-line max rule and structure |
| **LOGGING_PATTERNS.md** | 357 | 7 | Array-based logging format |
| **LOGGING_INTEGRATION.md** | 464 | 7 | External logging hooks |
| **API_SPECIFICATIONS.md** | 421 | 6 | OpenAPI vs OpenSpec guidance |
| **PROMPTING_GUIDE.md** | 984 | 10 | Master prompting techniques |
| **CROSS_CUTTING_2026.md** | 288 | 5 | Cross-cutting concerns for 2026 |

**Total:** 14 standards documents (INDEX.md + 13 guides)

### Sprints (`docs/sprints/`)

| File | Lines | Key Sections | Purpose |
|------|-------|--------------|---------|
| **INDEX.md** | 69 | 3 | Sprint navigation hub |
| **SPRINT_TEMPLATE.md** | 532 | 15 | Task execution template |
| **SPRINT_GUIDE.md** | 270 | 9 | How to write sprints |
| **SPRINT_001_MCP_GAP_IMPLEMENTATION.md** | 514 | 10 | Sprint: MCP gap implementation |
| **SPRINT_002_WEB_UI_IMPLEMENTATION.md** | 771 | 15 | Sprint: Web UI implementation |
| **SPRINT_003_DOCUMENTATION_PARITY.md** | 754 | 12 | Sprint: Documentation parity |
| **SPRINT_005_PRECOMMIT_SAFETY.md** | 396 | 8 | Sprint: Pre-commit safety |
| **SPRINT_006_CUSTOM_ADVISOR_ROLES.md** | 1059 | 15 | Sprint: Custom advisor roles |

**Total:** 8 sprint documents

### UI/UX & Accessibility (`docs/`)

| File | Path | Lines | Purpose |
|------|------|-------|---------|
| **2026_UI_UX_STANDARD.md** | `docs/ui-ux/` | ~350 | UI/UX component patterns, design tokens, responsive breakpoints |
| **ACCESSIBILITY_GUIDE.md** | `docs/accessibility/` | ~300 | WCAG 3.0+ conformance (Bronze/Silver/Gold), testing methods |
| **SPATIAL_COMPUTING_UI.md** | `docs/spatial/` | ~400 | XR/VR/AR layout patterns, comfort zones, latency requirements |
| **ETHICAL_ENGAGEMENT.md** | `docs/ethical/` | ~250 | Dark pattern taxonomy and prevention, ethical design principles |

**Total:** 4 documents (game-design docs moved to private repo)

### AI-First Development & Safety (`docs/`)

| File | Path | Lines | Purpose |
|------|------|-------|---------|
| **AI_ASSISTED_DEV.md** | `docs/ai-dev/` | 326 | AI development patterns, vibe coding, decision matrix |
| **STATE_MANAGEMENT.md** | `docs/state/` | 303 | State architecture, client/server state, CRDTs |
| **GENERATIVE_ASSET_SAFETY.md** | `docs/generative/` | 332 | AI content disclosure, procedural generation safety |
| **MONETIZATION_GUARDRAILS.md** | `docs/monetization/` | 263 | IAP ethics, loot box transparency, virtual economy |
| **MULTIPLAYER_SAFETY.md** | `docs/multiplayer/` | 276 | Social safety, chat moderation, matchmaking |
| **ANALYTICS_ETHICS.md** | `docs/analytics/` | 302 | Analytics consent, data minimization, A/B testing |
| **CROSS_PLATFORM_DEPLOYMENT.md** | `docs/deployment/` | 259 | App store compliance, CI/CD, feature flags |

**Total:** 7 documents covering AI-first development guardrails

### Overall Documentation Summary

| Category | Documents | Total Lines |
|----------|-----------|-------------|
| Root docs | 7 | ~1,050 |
| Workflows | 11 | ~3,500 |
| Standards | 11 | ~4,400 |
| Sprints | 3 | ~816 |
| UI/UX & Accessibility | 4 | ~1,300 |
| AI-First Development | 7 | ~2,061 |
| **TOTAL** | **43** | **~12,927** |

---

## GitHub Integration

### GitHub Root (`.github/`)

| File/Diretory | Purpose |
|--------------|---------|
| **SECRETS_MANAGEMENT.md** | GitHub Secrets setup and rotation guide |
| **PULL_REQUEST_TEMPLATE.md** | PR template with AI attribution |
| **ISSUE_TEMPLATE/bug_report.md** | Bug report template |

### GitHub Workflows (`.github/workflows/`)

| File | Purpose |
|------|---------|
| **secret-validation.yml** | Validate no secrets in commits |
| **documentation-check.yml** | Validate documentation format |
| **guardrails-lint.yml** | Enforce guardrails compliance |

---

## Examples Directory

### Language-Specific Examples (`examples/`)

| Directory | Files | Lines | Language | Purpose |
|-----------|-------|-------|----------|---------|
| **examples/** | 53 | ~2,000 | Mixed | Guardrails implementation examples |
| **go/** | 7 | ~300 | Go 1.19+ | Environment-specific config |
| **java/** | 15 | ~500 | Java 11+ | ConfigLoader with validation |
| **python/** | 8 | ~350 | Python 3.8+ | YAML config with type hints |
| **ruby/** | 7 | ~300 | Ruby 3.0+ | BDD-style testing |
| **rust/** | 4 | ~200 | Rust 1.70+ | Type-safe Serde config |
| **typescript/** | 10 | ~350 | TypeScript 5+ | Modular logging hooks |
| **scala/functional-ui/** | ~10 | ~400 | Scala 3.4+ | Functional composition, type-safe CSS, DDA telemetry |
| **r/game-analytics/** | ~8 | ~350 | R 4.3+ | ggplot2 4.0+, Shiny 2.0+, ethics auditing |
| **flutter/cross-platform/** | 4 | ~350 | Dart/Flutter | Ethical widgets, accessibility wrappers, guardrail config |

### Examples Structure

Each language example includes:
- Source code demonstrating guardrails patterns
- Tests validating separation requirements
- Environment-specific configuration files
- Build/test instructions
- Language-specific README

---

## Document Purpose Quick Reference

| Document | Primary Audience | Key Sections |
|----------|------------------|--------------|
| **AGENT_GUARDRAILS.md** | All AI agents | Four Laws, Pre-Execution Checklist, Forbidden Actions |
| **TEST_PRODUCTION_SEPARATION.md** | All AI agents | Three Laws, Blocking Violations, Uncertainty Protocol |
| **AGENT_EXECUTION.md** | All AI agents | Task Flow, Rollback, Error Handling, Three Strikes Rule |
| **AGENT_ESCALATION.md** | All AI agents | Audit Requirements, When to Escalate |
| **AGENT_REVIEW_PROTOCOL.md** | All AI agents | Dual-Agent Review, Cross-Model Review, Review Package |
| **PROJECT_CONTEXT_TEMPLATE.md** | Project setup | Tech Stack, Style Guide, Forbidden Patterns |
| **ADVERSARIAL_TESTING.md** | Security testing | Breaker Agent, Attack Vectors, Fuzz Testing |
| **DEPENDENCY_GOVERNANCE.md** | All AI agents | Package Allow-List, Forbidden Packages |
| **INFRASTRUCTURE_STANDARDS.md** | DevOps/IaC | Terraform, Drift Detection, No-ClickOps |
| **OPERATIONAL_PATTERNS.md** | Service developers | Health Checks, Circuit Breakers, Retry, Rate Limiting |
| **HOW_TO_APPLY.md** | All agents | 4 Options with ready-to-use prompts |
| **INDEX_MAP.md** | All agents | Find docs by keyword (60-80% token savings) |
| **HEADER_MAP.md** | All agents | Section-level lookup for targeted reading |
| **SPRINT_TEMPLATE.md** | Agents creating tasks | Complete task execution format |

---

## File Size Summary

| Category | Files | Min Lines | Max Lines | Average Lines |
|----------|-------|-----------|-----------|--------------|
| Root | 7 | 59 | 1034 | ~500 |
| docs/ | 8 | 130 | 984 | ~400 |
| docs/workflows/ | 12 | 151 | 638 | ~370 |
| docs/standards/ | 14 | 128 | 984 | ~450 |
| docs/sprints/ | 8 | 69 | 1059 | ~550 |
| .github/ | 3 | ~50 | ~150 | ~100 |
| examples/ | ~50 | ~30 | ~200 | ~50 |
| **TOTAL** | ~100 | 59 | 1059 | ~350 |

---

## Compliance Status

### 500-Line Maximum Compliance

All documents comply with the 500-line maximum rule:

| Document | Lines | Status |
|----------|-------|--------|
| README.md | 359 | ✅ |
| AGENT_GUARDRAILS.md | 319 | ✅ |
| HOW_TO_APPLY.md | 434 | ✅ |
| TEST_PRODUCTION_SEPARATION.md | 559 | Exceeds - needs split |
| AGENT_REVIEW_PROTOCOL.md | 638 | Exceeds - needs split |
| AGENT_EXECUTION.md | 546 | Exceeds - needs split |
| REGRESSION_PREVENTION.md | 536 | Exceeds - needs split |
| INFRASTRUCTURE_STANDARDS.md | 546 | Exceeds - needs split |
| OPERATIONAL_PATTERNS.md | 667 | Exceeds - needs split |
| ADVERSARIAL_TESTING.md | 510 | Exceeds - needs split |
| PROMPTING_GUIDE.md | 984 | Exceeds - needs split |
| SPRINT_002_WEB_UI_IMPLEMENTATION.md | 771 | Exceeds - needs split |
| SPRINT_003_DOCUMENTATION_PARITY.md | 754 | Exceeds - needs split |
| SPRINT_006_CUSTOM_ADVISOR_ROLES.md | 1059 | Exceeds - needs split |
| All other docs | <500 | ✅ |

**Note:** 11 documents exceed the 500-line limit. They will be split in a future release.

---

## Quick Lookup

**I need to...** → **Read this document:**

| Task | Document | Section |
|------|----------|---------|
| Find a document by keyword | INDEX_MAP.md | Quick Lookup Table |
| Understand safety rules | AGENT_GUARDRAILS.md | CORE PRINCIPLES (line 39) |
| Apply to existing repo | HOW_TO_APPLY.md | Option A (line 25) |
| Use example prompts | HOW_TO_APPLY.md | Option B (line 77) |
| Verify before committing | TESTING_VALIDATION.md | Post-Edit Validation (line 38) |
| Commit between to-dos | COMMIT_WORKFLOW.md | After Each To-Do (line 32) |
| Rollback changes | AGENT_EXECUTION.md | Rollback Procedures (line 51) |
| Review code | CODE_REVIEW.md | Self-Review Checklist (line 15) |
| Separate test/production | TEST_PRODUCTION_SEPARATION.md | CORE MANDATORY RULES (line 18) |
| Create task document | SPRINT_TEMPLATE.md | STEP-BY-STEP EXECUTION (line 91) |
| Write documentation | MODULAR_DOCUMENTATION.md | The 500-Line Rule (line 15) |
| Design UI components | 2026_UI_UX_STANDARD.md | Components, Design Tokens |
| Ensure accessibility | ACCESSIBILITY_GUIDE.md | WCAG 3.0+ Compliance |
| Build XR/VR/AR UIs | SPATIAL_COMPUTING_UI.md | Comfort Zones, Latency |
| Prevent dark patterns | ETHICAL_ENGAGEMENT.md | Dark Pattern Taxonomy |

---

## File Templates

All files follow these conventions:

- **Line limit:** 500 lines (11 docs currently exceed — see Compliance Status above)
- **Markdown:** CommonMark with GitHub extensions
- **Headers:** Level 1 (H1) for title, Level 2 (H2) for sections
- **Code blocks:** Backtick fences with language identifier
- **Tables:** GitHub-flavored Markdown tables
- **Lists:** Bullet and numbered lists for hierarchy

---

**Authored by:** TheArchitectit
**Document Owner:** Project Maintainers
**Last Updated:** 2026-08-14
**Note:** Game-design docs moved to private repo; counts reflect current repo contents
