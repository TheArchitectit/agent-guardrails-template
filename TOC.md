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
| **README.md** | ~150 | YES | Project overview and quick start |
| **INDEX_MAP.md** | 170 | YES | Master navigation - find docs by keyword |
| **HEADER_MAP.md** | 408 | YES | Section headers with line numbers |
| **CLAUDE.md** | 29 | Recommended | Optimized context for Claude Code CLI |
| **.claudeignore** | ~20 | Recommended | Token-saving ignore rules |
| **CHANGELOG.md** | 238 | YES | Release notes archive |
| **LICENSE** | - | YES | BSD-3-Clause license |
| **.gitignore** | ~100 | Recommended | Common ignore patterns |

---

## Documentation Directory

### Root Documentation (`docs/`)

| File | Lines | Sections | Purpose |
|------|-------|----------|---------|
| **AGENT_GUARDRAILS.md** | 267 | 13 | Core safety protocols (MANDATORY) |
| **HOW_TO_APPLY.md** | 432 | 5 | How to apply template with example prompts |

### Workflows (`docs/workflows/`)

| File | Lines | Key Sections | Purpose |
|------|-------|--------------|---------|
| **INDEX.md** | 126 | 5 | Workflow navigation hub |
| **AGENT_EXECUTION.md** | 280 | 6 | Execution protocol and rollback |
| **AGENT_ESCALATION.md** | 300 | 6 | Audit requirements and escalation |
| **TESTING_VALIDATION.md** | 303 | 9 | Validation protocols and checks |
| **COMMIT_WORKFLOW.md** | 328 | 8 | Commit timing and message format |
| **DOCUMENTATION_UPDATES.md** | ~250 | 5 | Post-sprint doc updates |
| **GIT_PUSH_PROCEDURES.md** | 323 | 8 | Push safety and verification |
| **BRANCH_STRATEGY.md** | ~200 | 6 | Git branching conventions |
| **CODE_REVIEW.md** | 348 | 7 | Code review and escalation |
| **MCP_CHECKPOINTING.md** | ~280 | 7 | MCP server checkpointing |

**Total:** 10 workflow documents (INDEX.md + 9 guides)

### Standards (`docs/standards/`)

| File | Lines | Key Sections | Purpose |
|------|-------|--------------|---------|
| **INDEX.md** | 89 | 4 | Standards navigation hub |
| **TEST_PRODUCTION_SEPARATION.md** | 558 | 12 | Test/production isolation (MANDATORY) |
| **MODULAR_DOCUMENTATION.md** | 330 | 8 | 500-line max rule and structure |
| **LOGGING_PATTERNS.md** | ~280 | 7 | Array-based logging format |
| **LOGGING_INTEGRATION.md** | ~250 | 7 | External logging hooks |
| **API_SPECIFICATIONS.md** | ~300 | 6 | OpenAPI vs OpenSpec guidance |

**Total:** 6 standards documents (INDEX.md + 5 guides)

### Sprints (`docs/sprints/`)

| File | Lines | Key Sections | Purpose |
|------|-------|--------------|---------|
| **INDEX.md** | 31 | 3 | Sprint navigation hub |
| **SPRINT_TEMPLATE.md** | 515 | 15 | Task execution template |
| **SPRINT_GUIDE.md** | 270 | 9 | How to write sprints |

**Total:** 3 sprint documents

### Overall Documentation Summary

| Category | Documents | Total Lines |
|----------|-----------|-------------|
| Root docs | 7 | ~1,050 |
| Workflows | 10 | ~3,000 |
| Standards | 6 | ~2,000 |
| Sprints | 3 | ~816 |
| **TOTAL** | **26** | **~6,866** |

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
| **AGENT_EXECUTION.md** | All AI agents | Task Flow, Rollback, Error Handling |
| **AGENT_ESCALATION.md** | All AI agents | Audit Requirements, When to Escalate |
| **HOW_TO_APPLY.md** | All agents | 4 Options with ready-to-use prompts |
| **INDEX_MAP.md** | All agents | Find docs by keyword (60-80% token savings) |
| **HEADER_MAP.md** | All agents | Section-level lookup for targeted reading |
| **SPRINT_TEMPLATE.md** | Agents creating tasks | Complete task execution format |

---

## File Size Summary

| Category | Files | Min Lines | Max Lines | Average Lines |
|----------|-------|-----------|-----------|--------------|
| Root | 7 | 29 | 408 | ~150 |
| docs/ | 3 | 238 | 432 | ~333 |
| docs/workflows/ | 10 | ~200 | ~350 | ~300 |
| docs/standards/ | 6 | ~250 | ~558 | ~333 |
| docs/sprints/ | 3 | 31 | 515 | ~272 |
| .github/ | 3 | ~50 | ~150 | ~100 |
| examples/ | 53 | ~30 | ~150 | ~40 |
| **TOTAL** | **85** | **29** | **558** | **~90** |

---

## Compliance Status

### 500-Line Maximum Compliance

All documents comply with the 500-line maximum rule:

| Document | Lines | Status |
|----------|-------|--------|
| README.md | ~150 | ✅ |
| AGENT_GUARDRAILS.md | 267 | ✅ |
| HOW_TO_APPLY.md | 432 | ✅ |
| TEST_PRODUCTION_SEPARATION.md | 558 | ⚠️ Exceeds - needs split |
| All workflows | ~280 average | ✅ |
| All standards | ~300 average | ✅ |
| All sprints | ~270 average | ✅ |

**Note:** TEST_PRODUCTION_SEPARATION.md is the only document exceeding 500 lines at 558 lines. It will be split in a future release.

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

---

## File Templates

All files follow these conventions:

- **Line limit:** 500 lines (except TEST_PRODUCTION_SEPARATION.md pending split)
- **Markdown:** CommonMark with GitHub extensions
- **Headers:** Level 1 (H1) for title, Level 2 (H2) for sections
- **Code blocks:** Backtick fences with language identifier
- **Tables:** GitHub-flavored Markdown tables
- **Lists:** Bullet and numbered lists for hierarchy

---

**Authored by:** TheArchitectit
**Document Owner:** Project Maintainers
**Last Updated:** 2026-01-18  
**Total Files:** 85  
**Total Lines:** ~7,500
