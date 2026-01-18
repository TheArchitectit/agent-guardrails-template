# Agent Guardrails Template

> Comprehensive safety protocol framework for AI agents, LLMs, and automated systems working with codebases.

---

## What Is This?

**The Agent Guardrails Template** is a standardized framework that defines safety protocols, guardrails, and operating procedures for AI agents interacting with code repositories.

It ensures that any AI system (Claude, GPT, Gemini, LLaMA, etc.) follows strict safety rules when performing tasks like:

- Reading and editing code
- Running tests and validations
- Making commits and pushing changes
- Accessing databases and infrastructure
- Creating or modifying documentation

### The Problem It Solves

**Without guardrails, AI agents may:**
- Modify code they haven't read
- Push untested changes to production
- Accidentally delete files
- Mix test and production data
- Forget to verify changes before committing
- Expose sensitive credentials

**With guardrails, agents:**
- Must read files before editing
- Validate changes before committing
- Separate test and production environments
- Ask for help when uncertain
- Follow consistent commit patterns
- Maintain clean, reversible git history

---

## Why Use This Template?

### For Human Developers

**Consistency:** All agents follow the same rules, regardless of which AI tool you use.

**Safety:** Prevents common mistakes like test/production mix-ups or accidental deletions.

**Predictability:** Always know what to expect when working with AI agents.

**Quality:** Ensures code is always tested and validated before committing.

### For AI Agents

**Clear Expectations:** Explicit rules for what can and cannot be done.

**Validation Protocols:** Step-by-step checks before committing.

**Escalation Guidance:** When and how to ask for human help.

**Audit Trail:** Standardized logging of all actions.

### For Teams

**Onboarding:** New agents and developers understand expectations immediately.

**Collaboration:** Consistent patterns across all repositories.

**Troubleshooting:** Clear rollback and error recovery procedures.

**Compliance:** Documented safety processes for audits.

---

## Key Features

### 🛡️ Four Laws of Agent Safety

1. **Read before editing** - Never modify code without reading it first
2. **Stay in scope** - Only touch files explicitly authorized
3. **Verify before committing** - Test and check all changes
4. **Halt when uncertain** - Ask for clarification instead of guessing

### 🚫 Forbidden Actions

Clear list of actions agents must never perform:
- Force pushing (destroys history)
- Modifying git config
- Creating test users in production
- Using production databases for tests
- Editing files outside declared scope
- And 20+ more critical prohibitions

### ✅ Mandatory Protocols

- **Pre-Execution Checklist** - 7 checks before any work
- **Test/Production Separation** - Isolated environments required
- **Validation Protocols** - Double-check work before committing
- **Commit Workflow** - When and how to commit (after each to-do)
- **Code Review** - Self-review and when to ask for human review

### 📋 Sprint Task Framework

- Ready-to-use task templates for agents
- Step-by-step execution instructions
- Validation gates and completion checklists
- Rollback procedures for every scenario

### 📊 Token Efficiency

- **INDEX_MAP.md** - Find docs by keyword (saves 60-80% tokens)
- **HEADER_MAP.md** - Section-level lookup for targeted reading
- **MAX 500 lines per document** - Fast context loading
- **.claudeignore** - Skip irrelevant files

---

## Quick Start

### For AI Agents: Apply This Template

**If you're an AI agent (Claude, GPT, Gemini, etc.), see [docs/HOW_TO_APPLY.md](docs/HOW_TO_APPLY.md) for:**

- Detailed step-by-step instructions
- 5 ready-to-use prompts (copy-paste)
- How to add to existing repository
- Migration examples
- Verification checklists

**Quick Options:**
- **Add to existing repo** → [Option A](docs/HOW_TO_APPLY.md#option-a-apply-to-an-existing-repository)
- **Use example prompts** → [Option B](docs/HOW_TO_APPLY.md#option-b-example-ai-agent-prompts)
- **Create new repo** → [Option C](docs/HOW_TO_APPLY.md#option-c-create-a-new-repository-with-standards)
- **Migrate existing docs** → [Option D](docs/HOW_TO_APPLY.md#option-d-migrate-existing-documentation-to-guardrails-structure)

### For Humans: Use This Template via GitHub

**Create new repository from template:**

1. Click the green **"Use this template"** button above
2. Select **"Create a new repository"**
3. Name your repository and set visibility
4. Click **"Create repository"**

**Or use CLI:**

```bash
gh repo create my-new-project \
  --template TheArchitectit/agent-guardrails-template \
  --private \
  --clone
```

---

## Project Structure

```
agent-guardrails-template/
├── README.md              ← What you're reading now
├── TOC.md                 ← Complete file listing
├── INDEX_MAP.md           ← Find docs by keyword (start here)
├── HEADER_MAP.md          ← Section-level lookup
├── CLAUDE.md               ← Claude Code CLI guidelines
├── CHANGELOG.md           ← Release notes archive
├── docs/                   ← Documentation
│   ├── AGENT_GUARDRAILS.md       # Core guardrails (MANDATORY)
│   ├── HOW_TO_APPLY.md             # How to apply template
│   ├── workflows/                   # Operational procedures (10 docs)
│   │   ├── INDEX.md
│   │   ├── AGENT_EXECUTION.md       # Execution protocol
│   │   ├── AGENT_ESCALATION.md      # Audit & escalation
│   │   ├── TESTING_VALIDATION.md
│   │   ├── COMMIT_WORKFLOW.md
│   │   ├── GIT_PUSH_PROCEDURES.md
│   │   ├── BRANCH_STRATEGY.md
│   │   ├── CODE_REVIEW.md
│   │   ├── ROLLBACK_PROCEDURES.md
│   │   ├── MCP_CHECKPOINTING.md
│   │   └── DOCUMENTATION_UPDATES.md
│   ├── standards/                   # Coding standards (6 docs)
│   │   ├── INDEX.md
│   │   ├── TEST_PRODUCTION_SEPARATION.md  # Test/production isolation (MANDATORY)
│   │   ├── MODULAR_DOCUMENTATION.md
│   │   ├── LOGGING_PATTERNS.md
│   │   ├── LOGGING_INTEGRATION.md
│   │   └── API_SPECIFICATIONS.md
│   └── sprints/                     # Task framework (3 docs)
│       ├── INDEX.md
│       ├── SPRINT_TEMPLATE.md      # Task execution template
│       └── SPRINT_GUIDE.md          # How to write sprints
├── examples/               ← Real-world implementations
│   ├── go/                    # Go examples
│   ├── java/                  # Java examples
│   ├── python/                # Python examples
│   ├── ruby/                  # Ruby examples
│   ├── rust/                  # Rust examples
│   └── typescript/            # TypeScript examples
└── .github/                ← GitHub integration
    ├── SECRETS_MANAGEMENT.md  # GitHub Secrets guide
    ├── PULL_REQUEST_TEMPLATE.md
    ├── ISSUE_TEMPLATE/bug_report.md
    └── workflows/
        ├── secret-validation.yml
        ├── documentation-check.yml
        └── guardrails-lint.yml
```

---

## Documentation Guide

### Start Here

1. **New to this project?** Read this README (what you're reading now)
2. **Need the full list?** See [TOC.md](TOC.md) - complete file listing
3. **Find a specific document?** Use [INDEX_MAP.md](INDEX_MAP.md) - keyword search
4. **Jump to a section?** Use [HEADER_MAP.md](HEADER_MAP.md) - line-number lookup
5. ** applying to your repo?** See [docs/HOW_TO_APPLY.md](docs/HOW_TO_APPLY.md) - detailed instructions

### Core Documents

| Document | Who Needs It | What It Covers |
|----------|-------------|---------------|
| [**AGENT_GUARDRAILS.md**](docs/AGENT_GUARDRAILS.md) | EVERYONE | Core safety protocols (MANDATORY) |
| [**TEST_PRODUCTION_SEPARATION.md**](docs/standards/TEST_PRODUCTION_SEPARATION.md) | EVERYONE | Test/production isolation (MANDATORY) |
| [**HOW_TO_APPLY.md**](docs/HOW_TO_APPLY.md) | Applying template | Step-by-step instructions with prompts |
| [**TOC.md**](TOC.md) | Everyone | Complete file listing and organization |
| [**INDEX_MAP.md**](INDEX_MAP.md) | Everyone | Find docs by keyword (saves 60-80% tokens) |

### Operational Documents

| Document | When to Read |
|----------|-------------|
| [COMMIT_WORKFLOW.md](docs/workflows/COMMIT_WORKFLOW.md) | Before committing changes |
| [TESTING_VALIDATION.md](docs/workflows/TESTING_VALIDATION.md) | Before committing changes |
| [CODE_REVIEW.md](docs/workflows/CODE_REVIEW.md) | After making changes |
| [ROLLBACK_PROCEDURES.md](docs/workflows/ROLLBACK_PROCEDUTES.md) | When errors occur |
| [GIT_PUSH_PROCEDURES.md](docs/workflows/GIT_PUSH_PROCEDURES.md) | Before pushing to remote |

---

## Statistics

| Metric | Count |
|--------|-------|
| **Total Documentation Files** | 26 |
| **Total Lines** | ~6,866 |
| **Workflows** | 10 documents |
| **Standards** | 6 documents  
| **Examples** | 53 files (6 languages) |
| **500-Line Compliance** | 25/26 (96%) |
| **Supported AI Models** | 30+ LLM families |
| **Programming Languages** | Go, Java, Python, Ruby, Rust, TypeScript |

---

## Version History

See [CHANGELOG.md](CHANGELOG.md) for complete release history.

**Current Version:** v1.5.0 (2026-01-18)

---

## License

BSD-3-Clause - See [LICENSE](LICENSE) file for details.

---

## Getting Help

- **Documentation:** Start with [INDEX_MAP.md](INDEX_MAP.md)
- **Applying to your repo:** See [docs/HOW_TO_APPLY.md](docs/HOW_TO_APPLY.md)
- **Examples:** [examples/](examples/) - real-world implementations
- **Issues:** [GitHub Issues](https://github.com/TheArchitectit/agent-guardrails-template/issues)

---

## Credits

- **Maintainer:** TheArchitectit
- **AI Tooling:** Created with Claude Code and Opus

---

**Last Updated:** 2026-01-18  
**Status:** v1.5.0 - Production Ready
