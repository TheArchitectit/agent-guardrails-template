# Documentation Index Map

> **READ THIS FIRST** - Find what you need without loading full documents.
> Estimated token savings: 60-80% when using targeted lookups.

---

## Quick Lookup Table

| Keyword | Document | Path | Purpose |
|---------|----------|------|---------|
| quick-setup | QUICK_SETUP.md | ./ | **5-minute setup guide** ⭐ |
| prompting | PROMPTING_GUIDE.md | ./ | **Master prompting techniques** ⭐ |
| toc | TOC.md | ./ | Complete template contents and file listing |
| safety | AGENT_GUARDRAILS.md | docs/ | Mandatory safety protocols |
| test-prod | TEST_PRODUCTION_SEPARATION.md | docs/standards/ | Test/production isolation (MANDATORY) |
| execution | AGENT_EXECUTION.md | docs/workflows/ | Standard execution protocol |
| escalation | AGENT_ESCALATION.md | docs/workflows/ | Audit & escalation procedures |
| how-to-apply | HOW_TO_APPLY.md | docs/ | How to apply guardrails to repos |
| commit | COMMIT_WORKFLOW.md | docs/workflows/ | When/how to commit |
| push | GIT_PUSH_PROCEDURES.md | docs/workflows/ | Push safety procedures |
| branch | BRANCH_STRATEGY.md | docs/workflows/ | Git branching conventions |
| rollback | ROLLBACK_PROCEDURES.md | docs/workflows/ | Recovery and undo |
| test | TESTING_VALIDATION.md | docs/workflows/ | Validation protocols |
| review | CODE_REVIEW.md | docs/workflows/ | Code review process |
| checkpoint | MCP_CHECKPOINTING.md | docs/workflows/ | MCP auto-checkpoint |
| docs | DOCUMENTATION_UPDATES.md | docs/workflows/ | Post-sprint doc updates |
| logging | LOGGING_PATTERNS.md | docs/standards/ | Array-based logging |
| hooks | LOGGING_INTEGRATION.md | docs/standards/ | External logging hooks |
| modular | MODULAR_DOCUMENTATION.md | docs/standards/ | 500-line rule |
| api | API_SPECIFICATIONS.md | docs/standards/ | OpenAPI/OpenSpec guidance |
| secrets | SECRETS_MANAGEMENT.md | .github/ | GitHub Secrets setup |
| examples | examples/ | examples/ | Multi-language implementation examples |
| regression-examples | regression-prevention/ | examples/regression-prevention/ | Practical regression prevention examples |
| sprint | SPRINT_TEMPLATE.md | docs/sprints/ | Sprint task template |
| sprint-guide | SPRINT_GUIDE.md | docs/sprints/ | How to write sprints |
| validation | SPRINT_TEMPLATE.md | docs/sprints/ | Completion gate & validation loop |
| completion | SPRINT_TEMPLATE.md | docs/sprints/ | Pre-completion checklist |
| context | PROJECT_CONTEXT_TEMPLATE.md | docs/standards/ | Project Bible - stack constraints, style guide |
| adversarial | ADVERSARIAL_TESTING.md | docs/standards/ | Breaker agent, fuzz testing, attack checklists |
| agent-review | AGENT_REVIEW_PROTOCOL.md | docs/workflows/ | Post-work verification by another agent |
| dependencies | DEPENDENCY_GOVERNANCE.md | docs/standards/ | Package allow-list, forbidden packages |
| infrastructure | INFRASTRUCTURE_STANDARDS.md | docs/standards/ | IaC, Terraform, drift detection |
| operational | OPERATIONAL_PATTERNS.md | docs/standards/ | Health checks, circuit breakers, retry |
| retry | AGENT_EXECUTION.md | docs/workflows/ | Three Strikes Rule, retry limits |
| scope-freeze | SPRINT_TEMPLATE.md | docs/sprints/ | Scope Freeze Protocol |
| deployment | DEPLOYMENT_GUIDE.md | mcp-server/ | MCP server deployment instructions (critical fixes) |
| schema-error | DEPLOYMENT_GUIDE.md | mcp-server/ | Fix schema validation error (guardrail-mcp → guardrail_mcp) |
| postgres-perm | DEPLOYMENT_GUIDE.md | mcp-server/ | Fix postgres permission errors (user 70:70) |
| container-networking | DEPLOYMENT_GUIDE.md | mcp-server/ | Pod networking for container communication |
| skills | AGENTS_AND_SKILLS_SETUP.md | docs/ | Setup agents and skills for all platforms |
| claude-code | CLCODE_INTEGRATION.md | docs/ | Claude Code skills and hooks integration |
| opencode | OPCODE_INTEGRATION.md | docs/ | OpenCode agents and skills integration |
| cursor | CURSOR_INTEGRATION.md | docs/ | Cursor rules and configuration |
| copilot | CLCODE_INTEGRATION.md | docs/ | GitHub Copilot instructions (see Claude Code) |
| cody | CLCODE_INTEGRATION.md | docs/ | Cody context configuration (see Claude Code) |
| mcp-server | MCP_SERVER_PLAN.md | docs/plans/ | MCP server implementation plan |
| mcp-api | API.md | mcp-server/ | MCP server REST API documentation |
| mcp-changelog | CHANGELOG.md | mcp-server/ | MCP server version history |
| guardrail-platform | MCP_SERVER_PLAN.md | docs/plans/ | Guardrail enforcement platform |
| team-tools | TEAM_TOOLS.md | docs/ | Team layout management MCP tools reference (Go implementation) |
| team-structure | TEAM_STRUCTURE.md | docs/ | 12-team enterprise structure documentation |
| python-migration | PYTHON_TO_GO_MIGRATION.md | docs/ | Python to Go migration guide for developers |
| go-migration | PYTHON_TO_GO_MIGRATION.md | docs/ | Python to Go migration guide for developers |
| team-cli | cmd/team-cli/README.md | cmd/team-cli/ | Team management CLI tool |
| phase-gate | TEAM_TOOLS.md | docs/ | Phase transition requirements and deliverables |
| aider | CLCODE_INTEGRATION.md | docs/ | Aider YAML configuration (see Claude Code) |
| continue | CLCODE_INTEGRATION.md | docs/ | Continue IDE configuration (see Claude Code) |
| windsurf | CLCODE_INTEGRATION.md | docs/ | Windsurf rules configuration (see Claude Code) |
| generic | GENERIC_LLM_INTEGRATION.md | docs/ | Generic/local LLM configuration guide |
| setup | setup_agents.py | scripts/ | CLI tool to generate agent configurations |
| regression | REGRESSION_PREVENTION.md | docs/workflows/ | Bug tracking and regression prevention protocol |
| failure-registry | failure-registry.jsonl | .guardrails/ | Append-only bug database (JSONL format) |
| pre-work-check | pre-work-check.md | .guardrails/ | MANDATORY pre-work regression checklist |
| log-failure | log_failure.py | scripts/ | CLI tool to log bugs to failure registry |
| regression-check | regression_check.py | scripts/ | Pre-commit regression pattern scanner |
| prevention-rules | pattern-rules.json | .guardrails/prevention-rules/ | Regex patterns to prevent regressions |
| semantic-rules | semantic-rules.json | .guardrails/prevention-rules/ | AST-based prevention rules |
| extracted-rules | extracted-rules.json | .guardrails/prevention-rules/ | Rules extracted from AGENT_GUARDRAILS.md |
| bug-fix | REGRESSION_PREVENTION.md | docs/workflows/ | Requirements for bug fixes (regression tests) |
| known-bugs | failure-registry.jsonl | .guardrails/ | Active/resolved/deprecated bug history |
| four-laws | four-laws.md | skills/shared-prompts/ | Canonical Four Laws of Agent Safety |
| halt-conditions | halt-conditions.md | skills/shared-prompts/ | When to stop and ask for help |
| sprint-001 | SPRINT_001_MCP_GAP_IMPLEMENTATION.md | docs/sprints/ | Sprint: MCP Gap Implementation |
| sprint-002 | SPRINT_002_WEB_UI_IMPLEMENTATION.md | docs/sprints/ | Sprint: Web UI Implementation |
| sprint-003 | SPRINT_003_DOCUMENTATION_PARITY.md | docs/sprints/ | Sprint: Documentation Parity (this sprint) |
| rules-from-md | RULES_FROM_MD.md | docs/ | Extracting prevention rules from markdown |
| rules-index | RULES_INDEX_MAP.md | docs/ | Master index of all prevention rules |
| mcp-tools | MCP_TOOLS_REFERENCE.md | docs/ | MCP validation tools documentation |
| rule-patterns | RULE_PATTERNS_GUIDE.md | docs/ | Pattern authoring guide |

---

## Document Summaries

| Document | Purpose (one line) | When to Use |
|----------|-------------------|-------------|
| **TOC.md** | Complete template contents and file listing | When exploring full template |
| **AGENT_GUARDRAILS.md** | Core safety protocols (mandatory) | Before ANY code changes |
| **RULES_FROM_MD.md** | Extracting prevention rules from markdown | When working with MCP rules |
| **RULES_INDEX_MAP.md** | Master index of all prevention rules | When searching for specific prevention rules |
| **MCP_TOOLS_REFERENCE.md** | MCP validation tools documentation | When using MCP validation tools |
| **RULE_PATTERNS_GUIDE.md** | Pattern authoring guide | When writing new prevention rules |
| **TEST_PRODUCTION_SEPARATION.md** | Test/production isolation standards (MANDATORY) | Before ANY deployment |
| **AGENT_EXECUTION.md** | Execution protocol and rollback procedures | During task execution |
| **AGENT_ESCALATION.md** | Audit requirements and escalation procedures | When uncertain or errors occur |
| **HOW_TO_APPLY.md** | how to apply guardrails to repositories | When setting up agent guardrails |
| **TESTING_VALIDATION.md** | Validation functions and git diff verification | Before committing changes |
| **COMMIT_WORKFLOW.md** | Guidelines for commits between to-dos | After completing each task |
| **GIT_PUSH_PROCEDURES.md** | Pre-push checklist and safety rules | Before pushing to remote |
| **BRANCH_STRATEGY.md** | Git branching conventions (feature/hotfix/release) | When creating branches |
| **ROLLBACK_PROCEDURES.md** | Recovery commands for all scenarios | When errors occur |
| **MCP_CHECKPOINTING.md** | MCP server checkpoint integration | Before/after critical tasks |
| **DOCUMENTATION_UPDATES.md** | Post-sprint documentation procedures | After completing sprints |
| **MODULAR_DOCUMENTATION.md** | 500-line max rule and splitting strategies | When writing docs |
| **LOGGING_PATTERNS.md** | Array-based structured logging format | When implementing logging |
| **LOGGING_INTEGRATION.md** | Webhook/file/queue integration hooks | When adding external logging |
| **API_SPECIFICATIONS.md** | OpenAPI vs OpenSpec guidance | When documenting APIs |
| **SECRETS_MANAGEMENT.md** | GitHub Secrets setup and rotation | When handling credentials |
| **examples/** | Multi-language guardrails implementation examples | When exploring code examples |
| **regression-prevention/** | Bug tracking & regression prevention examples | When logging bugs or creating prevention rules |
| **mcp-server/API.md** | Complete REST API reference for MCP server | When integrating with MCP server |
| **mcp-server/CHANGELOG.md** | MCP server version history | When tracking MCP server updates |
| **SPRINT_TEMPLATE.md** | Copy-paste template for new sprints | When creating tasks |
| **SPRINT_GUIDE.md** | Best practices for writing sprints | When writing sprint docs |
| **PROJECT_CONTEXT_TEMPLATE.md** | Project Bible - stack, style, forbidden patterns | When setting up new project |
| **ADVERSARIAL_TESTING.md** | Breaker agent, fuzz testing, attack vectors | When security testing |
| **AGENT_REVIEW_PROTOCOL.md** | Post-work verification by another agent/LLM | After completing major work |
| **DEPENDENCY_GOVERNANCE.md** | Package allow-list, license compliance | When adding dependencies |
| **INFRASTRUCTURE_STANDARDS.md** | IaC, Terraform, no-ClickOps | When managing infrastructure |
| **OPERATIONAL_PATTERNS.md** | Health checks, circuit breakers, retry | When implementing services |
| **AGENTS_AND_SKILLS_SETUP.md** | Setup agents and skills for all AI platforms | When configuring AI tools |
| **CLCODE_INTEGRATION.md** | Claude Code skills and hooks integration | When using Claude Code |
| **OPENCODE_INTEGRATION.md** | OpenCode agents and skills integration | When using OpenCode |
| **CURSOR_INTEGRATION.md** | Cursor rules and guardrails integration | When using Cursor |
| **GENERIC_LLM_INTEGRATION.md** | Generic/local LLM configuration (Ollama, vLLM, etc.) | When using custom LLMs |

---

## Category Index

### AI Tools Integration
- `AGENTS_AND_SKILLS_SETUP.md` - Setup guide for all AI platforms (Claude Code, OpenCode, Cursor, Copilot, etc.)
- `CLCODE_INTEGRATION.md` - Claude Code skills and hooks
- `OPENCODE_INTEGRATION.md` - OpenCode agents and skills
- `CURSOR_INTEGRATION.md` - Cursor rules configuration
- `GENERIC_LLM_INTEGRATION.md` - Generic/local LLM setup (Ollama, vLLM, etc.)

### Git Operations
- `COMMIT_WORKFLOW.md` - Commit timing and format
- `GIT_PUSH_PROCEDURES.md` - Push safety and verification
- `BRANCH_STRATEGY.md` - Branch naming and workflow
- `ROLLBACK_PROCEDURES.md` - Undo and recovery

### Quality & Validation
- `TESTING_VALIDATION.md` - Pre/post validation checks
- `CODE_REVIEW.md` - Review process and escalation
- `AGENT_GUARDRAILS.md` - Safety protocols (MANDATORY)
- `AGENT_REVIEW_PROTOCOL.md` - Post-work agent/LLM review
- `ADVERSARIAL_TESTING.md` - Breaker agent and fuzz testing
- `AGENTS_AND_SKILLS_SETUP.md` - Setup guide for Claude Code/OpenCode
- `RULES_FROM_MD.md` - Extracting prevention rules from markdown
- `RULES_INDEX_MAP.md` - Master index of all prevention rules
- `MCP_TOOLS_REFERENCE.md` - MCP validation tools documentation
- `RULE_PATTERNS_GUIDE.md` - Pattern authoring guide

### Logging & Monitoring
- `LOGGING_PATTERNS.md` - Structured log format
- `LOGGING_INTEGRATION.md` - External system hooks
- `MCP_CHECKPOINTING.md` - State checkpoints

### Documentation Standards
- `MODULAR_DOCUMENTATION.md` - 500-line rule
- `DOCUMENTATION_UPDATES.md` - Post-sprint updates
- `API_SPECIFICATIONS.md` - API doc formats

### Security
- `SECRETS_MANAGEMENT.md` - GitHub Secrets
- `AGENT_GUARDRAILS.md` - Forbidden actions
- `ADVERSARIAL_TESTING.md` - Security attack checklists
- `DEPENDENCY_GOVERNANCE.md` - Package allow-list

### Infrastructure & Operations
- `INFRASTRUCTURE_STANDARDS.md` - IaC and Terraform standards
- `OPERATIONAL_PATTERNS.md` - Health checks, circuit breakers, retry

### Project Setup
- `PROJECT_CONTEXT_TEMPLATE.md` - Project Bible template

### Sprint Framework
- `SPRINT_TEMPLATE.md` - Task template
- `SPRINT_GUIDE.md` - Writing guide
- `INDEX.md` (sprints/) - Sprint navigation

---

## Directory Structure

```
agent-guardrails-template/
├── INDEX_MAP.md              ← YOU ARE HERE
├── TOC.md                   ← Complete file listing and contents
├── HEADER_MAP.md             # Section-level lookup
├── CLAUDE.md                 # Claude Code CLI config
├── .claudeignore             # Token-saving ignores
├── CHANGELOG.md              # Release notes archive
├── docs/
│   ├── AGENT_GUARDRAILS.md   # Core safety (MANDATORY)
│   ├── HOW_TO_APPLY.md       # How to apply to repos
│   ├── AGENTS_AND_SKILLS_SETUP.md         # Setup guide for Claude Code/OpenCode
│   ├── CLCODE_INTEGRATION.md              # Claude Code integration
│   ├── OPCODE_INTEGRATION.md              # OpenCode integration
│   ├── workflows/
│   │   ├── INDEX.md
│   │   ├── AGENT_EXECUTION.md       # Execution protocol
│   │   ├── AGENT_ESCALATION.md      # Audit & escalation
│   │   ├── TESTING_VALIDATION.md
│   │   ├── COMMIT_WORKFLOW.md
│   │   ├── DOCUMENTATION_UPDATES.md
│   │   ├── GIT_PUSH_PROCEDURES.md
│   │   ├── MCP_CHECKPOINTING.md
│   │   ├── BRANCH_STRATEGY.md
│   │   ├── CODE_REVIEW.md
│   │   ├── AGENT_REVIEW_PROTOCOL.md       # Post-work agent review
│   │   └── ROLLBACK_PROCEDURES.md
│   ├── standards/
│   │   ├── INDEX.md
│   │   ├── TEST_PRODUCTION_SEPARATION.md  # Test/production isolation (MANDATORY)
│   │   ├── PROJECT_CONTEXT_TEMPLATE.md    # Project Bible template
│   │   ├── ADVERSARIAL_TESTING.md         # Breaker agent, fuzz testing
│   │   ├── DEPENDENCY_GOVERNANCE.md       # Package allow-list
│   │   ├── INFRASTRUCTURE_STANDARDS.md    # IaC, Terraform, drift
│   │   ├── OPERATIONAL_PATTERNS.md        # Health checks, circuit breakers
│   │   ├── MODULAR_DOCUMENTATION.md
│   │   ├── LOGGING_PATTERNS.md
│   │   ├── LOGGING_INTEGRATION.md
│   │   └── API_SPECIFICATIONS.md
│   └── sprints/
│       ├── INDEX.md
│       ├── SPRINT_TEMPLATE.md
│       ├── SPRINT_GUIDE.md
│       └── archive/
├── examples/               ← Multi-language implementation examples
│   ├── go/
│   ├── java/
│   ├── python/
│   ├── ruby/
│   ├── regression-prevention/  # Bug tracking examples
│   ├── rust/
│   └── typescript/
├── scripts/                ← Setup and utility scripts
│   └── setup_agents.py     # CLI tool to generate agent configs
├── .github/
│   ├── SECRETS_MANAGEMENT.md
│   ├── PULL_REQUEST_TEMPLATE.md
│   ├── workflows/
│   │   ├── secret-validation.yml
│   │   ├── documentation-check.yml
│   │   └── guardrails-lint.yml
│   └── ISSUE_TEMPLATE/
│       └── bug_report.md
└── README.md
```

---

## Usage Instructions

### For AI Agents

1. **Always read INDEX_MAP.md first** before exploring documentation
2. Use the Quick Lookup Table to find relevant documents by keyword
3. Check HEADER_MAP.md for specific section line numbers
4. Read only the sections you need using line offset parameters
5. For mandatory safety protocols, always read AGENT_GUARDRAILS.md

### For Humans

1. Use Category Index to browse by topic
2. Document Summaries tell you when to use each doc
3. Directory Structure shows the full file layout

---

## Cross-Reference Quick Links

| If you need... | Read... |
|----------------|---------|
| Safety rules before editing | AGENT_GUARDRAILS.md |
| How to validate changes | TESTING_VALIDATION.md |
| When to commit | COMMIT_WORKFLOW.md |
| How to handle errors | ROLLBACK_PROCEDURES.md |
| Logging format | LOGGING_PATTERNS.md |
| Secret handling | SECRETS_MANAGEMENT.md |
| Creating a new task | SPRINT_TEMPLATE.md |
| Setting up AI tools | AGENTS_AND_SKILLS_SETUP.md |
| Claude Code integration | CLCODE_INTEGRATION.md |
| OpenCode integration | OPCODE_INTEGRATION.md |
| Cursor integration | CURSOR_INTEGRATION.md |
| Generic LLM integration | GENERIC_LLM_INTEGRATION.md |
| MCP rule extraction | RULES_FROM_MD.md |
| Prevention rules index | RULES_INDEX_MAP.md |
| MCP tools reference | MCP_TOOLS_REFERENCE.md |
| Pattern authoring | RULE_PATTERNS_GUIDE.md |

---

**Authored by:** TheArchitectit
**Document Owner:** Project Maintainers
**Last Updated:** 2026-02-11
**Document Count:** 74 (excluding INDEX files)
**Line Count:** ~260

---

## Canonical Sources

To avoid duplication, always reference these canonical sources:

| Content | Canonical Location | Reference In |
|---------|-------------------|--------------|
| Four Laws | skills/shared-prompts/four-laws.md | docs/AGENT_GUARDRAILS.md |
| Halt Conditions | skills/shared-prompts/halt-conditions.md | Workflows, integration docs |

---

## Oversized Documents

The following files exceed the 500-line limit and should be split per MODULAR_DOCUMENTATION.md:

| File | Lines | Action Needed |
|------|-------|---------------|
| docs/plans/MCP_SERVER_PLAN.md | 2093 | Split into multiple files |
| docs/sprints/SPRINT_002_WEB_UI_IMPLEMENTATION.md | 768 | Split or archive |
| docs/sprints/SPRINT_003_DOCUMENTATION_PARITY.md | 754 | Split or archive after completion |
| HEADER_MAP.md | 822 | Navigation file - exempt |
| docs/standards/OPERATIONAL_PATTERNS.md | 667 | Split |
| docs/workflows/AGENT_REVIEW_PROTOCOL.md | 638 | Split |
| docs/security/SECURITY_AUDIT_CONFIG.md | 597 | Split |
| README.md | 565 | Landing page - exempt |
