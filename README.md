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

## MCP Server (New in v1.9.0)

The **Model Context Protocol (MCP) Server** provides real-time guardrail enforcement via a standardized protocol for AI agents and IDEs.

### Features

**6 MCP Tools:**

| Tool | Purpose |
|------|---------|
| `guardrail_init_session` | Initialize validation session for a task |
| `guardrail_validate_bash` | Validate bash commands before execution |
| `guardrail_validate_file_edit` | Validate file edits against rules |
| `guardrail_validate_git_operation` | Validate git commands for safety |
| `guardrail_pre_work_check` | Run pre-work checklist validation |
| `guardrail_get_context` | Get project context and guardrail rules |

**2 MCP Resources:**

| Resource | Description |
|----------|-------------|
| `guardrail://quick-reference` | Quick reference card for agents |
| `guardrail://rules/active` | Active prevention rules for current session |

**Endpoints:**

- **SSE Stream:** `GET /mcp/v1/sse` - Real-time event streaming
- **Message Handler:** `POST /mcp/v1/message` - JSON-RPC 2.0 protocol

**Web UI (Port 8093):**

Browser-based guardrail management interface:
- Document and rule management
- Session monitoring
- Configuration management

**Infrastructure:**

- **PostgreSQL 16** - Persistent storage for rules and sessions
- **Redis 7** - Caching layer for performance
- **Production Deployment** - Running on AI01 (100.96.49.42)

### Project Structure with MCP Server

```
agent-guardrails-template/
├── mcp-server/            ← MCP Server implementation
│   ├── src/               # Server source code
│   ├── deploy/            # Docker and deployment configs
│   │   └── Dockerfile
│   ├── podman-compose.yml # Container orchestration
│   └── requirements.txt   # Python dependencies
├── ...
```

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

### 🤖 AI Tool Integration

**Claude Code Support:**
- `scripts/setup_agents.py` - Generate Claude Code skills and hooks
- Skills: guardrails-enforcer, commit-validator, env-separator
- Hooks: pre-execution, post-execution, pre-commit

**OpenCode Support:**
- `.opencode/oh-my-opencode.jsonc` configuration
- Skills: guardrails-enforcer, commit-validator, env-separator
- Agents: guardrails-auditor, doc-indexer

**Script-Based Workflows:**
- Large code review automation
- Batch execution with guardrails compliance
- CI/CD integration patterns

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

### Setup with AI Tools (v1.7.0+)

**For Claude Code or OpenCode users, run the setup script:**

```bash
# Full setup (all skills, agents, and hooks)
python scripts/setup_agents.py --claude --opencode --full

# Or minimal setup (guardrails-enforcer only)
python scripts/setup_agents.py --claude --minimal
python scripts/setup_agents.py --opencode --minimal
```

**What gets created:**
- `.claude/skills/` - Claude Code skills (JSON)
- `.claude/hooks/` - Pre/post execution hooks (shell)
- `.opencode/` - OpenCode configuration and skills
- `skills/shared-prompts/` - Reusable prompt components

**Documentation:**
- [AGENTS_AND_SKILLS_SETUP.md](docs/AGENTS_AND_SKILLS_SETUP.md) - Complete setup guide
- [CLCODE_INTEGRATION.md](docs/CLCODE_INTEGRATION.md) - Claude Code details
- [OPENCODE_INTEGRATION.md](docs/OPENCODE_INTEGRATION.md) - OpenCode details

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

## Installation and Testing (MCP Server)

### Prerequisites

- Docker or Podman
- Access to AI01 server (100.96.49.42) for deployment
- Environment variables configured (see below)

### Environment Variables

Required environment variables for MCP server operation:

```bash
# API Keys
export MCP_API_KEY="your-mcp-api-key"
export IDE_API_KEY="your-ide-api-key"
export JWT_SECRET="your-jwt-secret"

# Database (PostgreSQL 16)
export DB_HOST="localhost"
export DB_PORT="5432"
export DB_NAME="guardrail_mcp"
export DB_USER="guardrail_user"
export DB_PASSWORD="your-db-password"

# Cache (Redis 7)
export REDIS_HOST="localhost"
export REDIS_PORT="6379"
export REDIS_PASSWORD="your-redis-password"

# Service Ports
export MCP_PORT="8092"
export WEB_PORT="8093"
```

### Build and Deploy

```bash
# Build Docker image
cd mcp-server
docker build -t guardrail-mcp:latest -f deploy/Dockerfile .

# Save image for transfer
docker save -o guardrail-mcp.tar guardrail-mcp:latest

# Deploy to AI01 server
scp guardrail-mcp.tar user001@100.96.49.42:/opt/guardrail-mcp/
ssh user001@100.96.49.42
cd /opt/guardrail-mcp

# Load and start with podman-compose
sudo podman load -i guardrail-mcp.tar
sudo podman-compose up -d

# Verify deployment
sudo podman-compose ps
```

### Testing the MCP Endpoint

**Test initialization:**

```bash
curl -X POST http://100.96.49.42:8092/mcp/v1/message \
  -H 'Content-Type: application/json' \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "initialize",
    "params": {
      "protocolVersion": "2024-11-05",
      "capabilities": {},
      "clientInfo": {
        "name": "test-client",
        "version": "1.0"
      }
    }
  }'
```

**Expected response:**

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": "2024-11-05",
    "capabilities": {
      "tools": {},
      "resources": {}
    },
    "serverInfo": {
      "name": "guardrail-mcp-server",
      "version": "1.9.0"
    }
  }
}
```

**Test guardrail validation:**

```bash
curl -X POST http://100.96.49.42:8092/mcp/v1/message \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer $MCP_API_KEY' \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/call",
    "params": {
      "name": "guardrail_validate_bash",
      "arguments": {
        "command": "rm -rf /",
        "context": "test-session-001"
      }
    }
  }'
```

### Accessing the Web UI

Once deployed, access the guardrail management interface:

```
http://100.96.49.42:8093
```

Features available:
- View and manage active guardrail rules
- Monitor validation sessions
- Configure rule sets
- View validation logs

### Troubleshooting

**Connection refused:**
- Verify podman-compose services are running: `sudo podman-compose ps`
- Check firewall rules on AI01
- Verify ports 8092 and 8093 are accessible

**Authentication errors:**
- Ensure `MCP_API_KEY` is set correctly
- Verify JWT_SECRET matches between client and server

**Database connection issues:**
- Check PostgreSQL is running: `sudo podman ps | grep postgres`
- Verify DB_HOST and DB_PORT environment variables
- Check network connectivity between containers

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
├── mcp-server/            ← MCP Server implementation (v1.9.0)
│   ├── src/               # Server source code
│   ├── deploy/            # Docker deployment configs
│   └── requirements.txt   # Python dependencies
├── docs/                   ← Documentation
│   ├── AGENT_GUARDRAILS.md       # Core guardrails (MANDATORY)
│   ├── HOW_TO_APPLY.md             # How to apply template
│   ├── AGENTS_AND_SKILLS_SETUP.md  # AI tool setup guide
│   ├── CLCODE_INTEGRATION.md       # Claude Code integration
│   ├── OPCODE_INTEGRATION.md       # OpenCode integration
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
├── scripts/                ← Setup and utility scripts
│   └── setup_agents.py        # CLI tool for AI tool configuration
├── skills/                 ← Reusable skill components
│   └── shared-prompts/        # Shared prompts for agents
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
| [**AGENTS_AND_SKILLS_SETUP.md**](docs/AGENTS_AND_SKILLS_SETUP.md) | AI tool users | Setup guide for Claude Code/OpenCode |
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
| **Total Documentation Files** | 31 |
| **Total Lines** | ~9,000 |
| **Workflows** | 10 documents |
| **Standards** | 11 documents |
| **Examples** | 53 files (6 languages) |
| **500-Line Compliance** | 30/31 (97%) |
| **Supported AI Models** | 30+ LLM families |
| **Programming Languages** | Go, Java, Python, Ruby, Rust, TypeScript |
| **AI Tool Integrations** | Claude Code, OpenCode |
| **MCP Server** | 6 tools, 2 resources, SSE + HTTP endpoints |
| **Infrastructure** | PostgreSQL 16, Redis 7, Docker/Podman |

---

## Version History

See [CHANGELOG.md](CHANGELOG.md) for complete release history.

**Current Version:** v1.9.0 (2026-02-07)

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
- Help support my coding plan, with this referral : https://synthetic.new/?referral=UAWqkKQQLFkzMkY
"Invite your friends to Synthetic and both of you will receive

$10.00 for standard signups.
$20.00 for pro signups.
in subscription credit when they subscribe!
"


---

**Last Updated:** 2026-02-07
**Status:** v1.9.0 - Production Ready
