# Documentation Index Map

> **READ THIS FIRST** - Find what you need without loading full documents.
> Estimated token savings: 60-80% when using targeted lookups.

---

## Quick Lookup Table

| Keyword | Document | Path | Purpose |
|---------|----------|------|---------|
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
| sprint | SPRINT_TEMPLATE.md | docs/sprints/ | Sprint task template |
| sprint-guide | SPRINT_GUIDE.md | docs/sprints/ | How to write sprints |
| validation | SPRINT_TEMPLATE.md | docs/sprints/ | Completion gate & validation loop |
| completion | SPRINT_TEMPLATE.md | docs/sprints/ | Pre-completion checklist |

---

## Document Summaries

| Document | Purpose (one line) | When to Use |
|----------|-------------------|-------------|
| **TOC.md** | Complete template contents and file listing | When exploring full template |
| **AGENT_GUARDRAILS.md** | Core safety protocols (mandatory) | Before ANY code changes |
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
| **SPRINT_TEMPLATE.md** | Copy-paste template for new sprints | When creating tasks |
| **SPRINT_GUIDE.md** | Best practices for writing sprints | When writing sprint docs |

---

## Category Index

### Git Operations
- `COMMIT_WORKFLOW.md` - Commit timing and format
- `GIT_PUSH_PROCEDURES.md` - Push safety and verification
- `BRANCH_STRATEGY.md` - Branch naming and workflow
- `ROLLBACK_PROCEDURES.md` - Undo and recovery

### Quality & Validation
- `TESTING_VALIDATION.md` - Pre/post validation checks
- `CODE_REVIEW.md` - Review process and escalation
- `AGENT_GUARDRAILS.md` - Safety protocols (MANDATORY)

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
│   │   └── ROLLBACK_PROCEDURES.md
│   ├── standards/
│   │   ├── INDEX.md
│   │   ├── TEST_PRODUCTION_SEPARATION.md  # Test/production isolation (MANDATORY)
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
│   ├── rust/
│   └── typescript/
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

---

**Last Updated:** 2026-01-18
**Document Count:** 28 (excluding INDEX files)
**Line Count:** ~170
