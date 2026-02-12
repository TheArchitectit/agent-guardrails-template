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

## MCP Server (Updated in v1.10.0)

The **Model Context Protocol (MCP) Server** provides real-time guardrail enforcement via a standardized protocol for AI agents and IDEs.

### Features

**11 MCP Tools:**

| Tool | Purpose |
|------|---------|
| `guardrail_init_session` | Initialize validation session for a task |
| `guardrail_validate_bash` | Validate bash commands before execution |
| `guardrail_validate_file_edit` | Validate file edits against rules |
| `guardrail_validate_git_operation` | Validate git commands for safety |
| `guardrail_pre_work_check` | Run pre-work checklist validation |
| `guardrail_get_context` | Get project context and guardrail rules |
| `guardrail_validate_scope` | Check if file path is within authorized scope |
| `guardrail_validate_commit` | Validate conventional commit format |
| `guardrail_prevent_regression` | Check failure registry for pattern matches |
| `guardrail_check_test_prod_separation` | Verify test/production isolation |
| `guardrail_validate_push` | Validate git push safety conditions |

**8 MCP Resources:**

| Resource | Description |
|----------|-------------|
| `guardrail://quick-reference` | Quick reference card for agents |
| `guardrail://rules/active` | Active prevention rules for current session |
| `guardrail://docs/agent-guardrails` | Core safety protocols documentation |
| `guardrail://docs/four-laws` | Four Laws of Agent Safety (canonical) |
| `guardrail://docs/halt-conditions` | When to stop and ask for help |
| `guardrail://docs/workflows` | Workflow documentation index |
| `guardrail://docs/standards` | Standards documentation index |
| `guardrail://docs/pre-work-checklist` | Pre-work regression checklist |

**Endpoints:**

- **SSE Stream:** `GET /mcp/v1/sse` - Real-time event streaming
- **Message Handler:** `POST /mcp/v1/message?session_id=<session_id>` - JSON-RPC 2.0 protocol
- **Web UI:** `GET /web` - Complete management interface

**Web UI (Port 8080/8081):**

Browser-based guardrail management interface:
- Dashboard with system stats
- Document browser with search
- Rules management (CRUD + toggle)
- Projects management
- Failure registry viewer
- IDE Tools validation interface

**Infrastructure:**

- **PostgreSQL 16** - Persistent storage for rules and sessions
- **Redis 7** - Caching layer for performance
- **Production Deployment** - Deploy to your infrastructure (see deployment guide)

### Project Structure with MCP Server

```
agent-guardrails-template/
├── mcp-server/            ← MCP Server implementation
│   ├── cmd/server/        # Go application entry point
│   ├── internal/          # MCP, web API, DB, cache, security modules
│   ├── deploy/            # Deployment manifests and container config
│   │   ├── Dockerfile
│   │   ├── podman-compose.yml
│   │   └── k8s-deployment.yaml
│   ├── API.md             # REST/API contract
│   └── README.md          # MCP server docs
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

---

## How to Use This Platform

### For Different User Types

#### 1. AI Agent Developers (Using Claude Code/OpenCode)

**Quick Start:**

```bash
# 1. Set up MCP server connection
# Add to your .opencode/oh-my-opencode.jsonc or Claude Code config:
{
  "mcpServers": {
    "guardrails": {
      "type": "remote",
      "url": "http://your-server:8094/mcp/v1/sse",
      "headers": {
        "Authorization": "Bearer YOUR_MCP_API_KEY"
      }
    }
  }
}

# 2. The MCP tools are now available to validate your actions
```

**What Happens Automatically:**
- Every bash command is validated before execution
- File edits are checked against scope boundaries
- Git operations are validated for safety
- Pre-work checklist runs before starting tasks
- Prevents common mistakes like editing unread files or mixing environments

**Example Workflow:**

```
User: "Add a new feature to the auth system"

↓ AI Agent uses guardrail_init_session
↓ Session created with project context

↓ AI Agent attempts to edit src/auth/login.js
↓ guardrail_validate_file_edit checks:
   ✓ File was read first (Read Before Edit)
   ✓ File is within authorized scope
   ✓ No forbidden patterns detected

↓ AI Agent runs tests
↓ guardrail_check_test_prod_separation verifies:
   ✓ Test database used, not production

↓ AI Agent commits changes
↓ guardrail_validate_commit checks:
   ✓ Conventional commit format
   ✓ No secrets in commit message

↓ Changes pushed with guardrail_validate_push
↓ guardrail_prevent_regression checks:
   ✓ No patterns matching past failures
```

#### 2. DevOps/SRE Teams (Deploying MCP Server)

**Production Deployment:**

```bash
# 1. Clone and build
git clone https://github.com/TheArchitectit/agent-guardrails-template.git
cd agent-guardrails-template/mcp-server

# 2. Configure environment
cp .env.example .env
# Edit .env with your production values:
# - Generate secure API keys
# - Set database credentials
# - Configure Redis

# 3. Deploy with Docker/Podman
docker compose -f deploy/podman-compose.yml up -d

# 4. Verify deployment
curl http://your-server:8095/health/ready
```

**Monitoring:**

```bash
# Check health
curl http://your-server:8095/health/ready

# View metrics
curl http://your-server:8095/metrics

# Check version
curl http://your-server:8095/version
```

**Access the Web UI:**

Open `http://your-server:8095` in your browser to:
- View active guardrail rules
- Monitor validation sessions
- Manage projects
- View failure registry
- Configure rule sets

#### 3. Development Teams (Using Web UI)

**Dashboard Overview:**

1. **Home/Dashboard** - System stats and health
2. **Documents** - Browse and search guardrail documentation
3. **Rules** - View and manage prevention rules
   - Toggle rules on/off
   - Create custom rules
   - Import/export rule sets
4. **Projects** - Manage project-specific configurations
5. **Failures** - View and update failure registry
   - Log new failures
   - Mark failures as resolved
   - See prevention rules created from failures

**Managing Prevention Rules:**

```javascript
// Example: Create a rule via Web UI API
fetch('http://your-server:8095/api/rules', {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer YOUR_MCP_API_KEY',
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    name: 'No Console Logs in Production',
    pattern: 'console\\.(log|debug)',
    language: 'javascript',
    severity: 'warning',
    description: 'Prevent console.log in production code'
  })
})
```

### MCP Tools in Detail

#### 1. guardrail_init_session

**Purpose:** Initialize a validation session before starting work

**When to use:** At the beginning of every task

**Example:**
```json
{
  "method": "tools/call",
  "params": {
    "name": "guardrail_init_session",
    "arguments": {
      "project_slug": "my-project",
      "agent_type": "claude-code",
      "client_version": "1.0.0"
    }
  }
}
```

**Returns:** Session token for subsequent validations

#### 2. guardrail_validate_bash

**Purpose:** Validate bash commands before execution

**When to use:** Before running any bash command

**Example:**
```json
{
  "method": "tools/call",
  "params": {
    "name": "guardrail_validate_bash",
    "arguments": {
      "command": "rm -rf /important/data",
      "session_token": "sess_abc123"
    }
  }
}
```

**Blocks:** Dangerous commands like `rm -rf /`, `dd if=/dev/zero`, etc.

#### 3. guardrail_validate_file_edit

**Purpose:** Validate file edits before applying them

**When to use:** Before modifying any file

**Example:**
```json
{
  "method": "tools/call",
  "params": {
    "name": "guardrail_validate_file_edit",
    "arguments": {
      "file_path": "src/config/production.js",
      "old_string": "const debug = true",
      "new_string": "const debug = false",
      "session_token": "sess_abc123"
    }
  }
}
```

**Validates:**
- File was read before editing
- File is within authorized scope
- No forbidden patterns in changes
- Secrets not being added

#### 4. guardrail_prevent_regression

**Purpose:** Check code against failure registry patterns

**When to use:** Before committing, after completing work

**Example:**
```json
{
  "method": "tools/call",
  "params": {
    "name": "guardrail_prevent_regression",
    "arguments": {
      "file_paths": ["src/auth.js", "src/login.js"],
      "code_content": "// ...code to check...",
      "session_token": "sess_abc123"
    }
  }
}
```

**Prevents:** Repeating past failures by pattern matching

### Common Use Cases

#### Use Case 1: Preventing Production Accidents

**Problem:** AI agent accidentally deletes production database

**Solution:**
```bash
# MCP server validates all bash commands
# This would be blocked:
$ psql -c "DROP DATABASE production"
# Error: Dangerous command detected
```

**Prevention Rule:**
```json
{
  "name": "No DROP DATABASE",
  "pattern": "DROP\\s+DATABASE",
  "severity": "critical",
  "action": "block"
}
```

#### Use Case 2: Ensuring Code Review

**Problem:** AI agent commits directly to main without review

**Solution:**
```bash
# MCP validates git operations
$ git push origin main
# Error: Direct push to main requires approval
```

**Prevention Rule:**
```json
{
  "name": "Require PR for main",
  "pattern": "push.*main",
  "severity": "critical",
  "action": "block"
}
```

#### Use Case 3: Test/Production Separation

**Problem:** Tests accidentally use production database

**Solution:**
```javascript
// MCP validates test code
const db = process.env.NODE_ENV === 'test' 
  ? testDb 
  : productionDb; // This would be flagged!

// Correct:
const db = testDb; // MCP validates test-only code
```

### Web UI Walkthrough

#### Dashboard

The dashboard shows:
- **System Health** - Database, Redis, and MCP server status
- **Validation Statistics** - Total validations, blocked actions, failures prevented
- **Active Sessions** - Current AI agent sessions
- **Recent Activity** - Latest validations and rule triggers

#### Documents Browser

Search and view all guardrail documentation:
```
Search: "git push"
Results:
- docs/workflows/GIT_PUSH_PROCEDURES.md
- docs/AGENT_GUARDRAILS.md (section on push safety)
```

#### Rules Management

**Active Rules View:**
```
┌─────────────────────────────────────────┐
│ Prevention Rules                        │
├─────────────────────────────────────────┤
│ ☑ No Force Push                         │
│ ☑ Read Before Edit                      │
│ ☑ No Secrets in Code                    │
│ ☐ Custom Rule (disabled)                │
└─────────────────────────────────────────┘
```

**Creating a Rule:**
1. Click "New Rule"
2. Enter pattern (regex or literal)
3. Select language (optional)
4. Set severity (info/warning/critical)
5. Save and activate

#### Failure Registry

Track and prevent recurring issues:
```
┌─────────────────────────────────────────┐
│ Recent Failures                         │
├─────────────────────────────────────────┤
│ 🔴 Database connection leak            │
│    Status: Active | Created: 2026-02-10│
│    Prevention: Added connection check  │
├─────────────────────────────────────────┤
│ 🟢 Missing await in async function     │
│    Status: Resolved | Fixed: 2026-02-09│
│    Prevention: ESLint rule added      │
└─────────────────────────────────────────┘
```

### Integration Examples

#### GitHub Actions Integration

```yaml
# .github/workflows/guardrails.yml
name: Guardrails Validation

on: [push, pull_request]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Validate with MCP
        run: |
          curl -X POST http://your-mcp-server:8095/api/ingest \
            -H "Authorization: Bearer ${{ secrets.MCP_API_KEY }}" \
            -d "{\"repo_path\": \".\", \"project_slug\": \"${{ github.repository }}\"}"
```

#### IDE Integration (VS Code)

Configure VS Code to use MCP validation:
```json
// .vscode/settings.json
{
  "guardrails.mcpServerUrl": "http://your-server:8094",
  "guardrails.apiKey": "${env:MCP_API_KEY}"
}
```

#### Custom Client Integration

```python
# Python client example
import requests

class GuardrailClient:
    def __init__(self, server_url, api_key):
        self.url = server_url
        self.headers = {"Authorization": f"Bearer {api_key}"}
    
    def validate_bash(self, command):
        response = requests.post(
            f"{self.url}/mcp/v1/message",
            headers=self.headers,
            json={
                "method": "tools/call",
                "params": {
                    "name": "guardrail_validate_bash",
                    "arguments": {"command": command}
                }
            }
        )
        return response.json()

# Usage
client = GuardrailClient("http://server:8094", "api-key")
result = client.validate_bash("rm -rf /")
# Returns: {allowed: false, reason: "Dangerous command"}
```

### Troubleshooting

**Connection refused:**
- Verify podman-compose services are running: `sudo podman-compose ps`
- Docker-only equivalent: `docker compose -f deploy/podman-compose.yml ps`
- Check firewall rules on your server
- Verify ports 8092 and 8093 are accessible
- **Port confusion:** Remember external ports (8094/8095) vs internal ports (8080/8081). Use external ports from outside the container.

**Authentication errors:**
- Ensure `MCP_API_KEY` is set correctly
- Verify JWT_SECRET matches between client and server
- Use `Authorization: Bearer <key>` format (not `X-API-Key` header)
- Check you're connecting to the MCP port (8094), not the Web UI port (8095)

**Database connection issues:**
- Check PostgreSQL is running: `sudo podman ps | grep postgres`
- Docker-only equivalent: `docker ps | grep postgres`
- Verify DB_HOST and DB_PORT environment variables
- Check network connectivity between containers

**MCP tools not responding:**
- Check SSE connection: `curl -sN http://server:8094/mcp/v1/sse`
- Verify session ID is being used correctly
- Check MCP server logs: `docker logs guardrail-mcp-server`

**Web UI not loading:**
- Check Web UI port (8095) is accessible
- Verify CORS settings if accessing from different origin
- Check browser console for JavaScript errors

---

## Installation and Testing (MCP Server)

### Prerequisites

- Docker or Podman
- Access to your deployment server for production use
- Environment variables configured (see below)

### Port Mapping

The MCP server uses two ports. When deploying with Docker/Podman, you map **external ports** to the container's **internal ports**:

| Service | Internal Port | External Port | Purpose |
|---------|--------------|---------------|---------|
| MCP Protocol | 8080 | 8094 (configurable) | SSE + JSON-RPC endpoint for AI agents |
| Web UI/API | 8081 | 8095 (configurable) | Web interface + REST API + health checks |

**In your configuration files:**
- Use **external ports** (8094/8095) when connecting from outside the container
- Use **internal ports** (8080/8081) only inside the container or when using host networking

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
# Example deployment convention: 8092/8093
# Defaults in compose: 8080/8081
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

# Deploy to your server
scp guardrail-mcp.tar user@your-server:/opt/guardrail-mcp/
ssh user@your-server
cd /opt/guardrail-mcp

# Load and start with podman-compose
sudo podman load -i guardrail-mcp.tar
sudo podman-compose up -d

# Verify deployment
sudo podman-compose ps
```

**Docker-only alternative (no Podman):**

```bash
cd mcp-server

# Build and start directly with Docker Compose
docker compose -f deploy/podman-compose.yml up -d --build

# Verify deployment
docker compose -f deploy/podman-compose.yml ps
```

If you need the tester-validated Docker variant:

```bash
cd mcp-server
docker compose -f deploy/docker-compose.example.yml up -d --build
docker compose -f deploy/docker-compose.example.yml ps
```

### Testing the MCP Endpoint

**Get session endpoint and initialize:**

```bash
# 1) Open SSE stream and capture endpoint event
curl -sN http://localhost:8092/mcp/v1/sse
# event: endpoint
# data: http://localhost:8092/mcp/v1/message?session_id=<session_id>

# 2) In another terminal, send initialize to the session endpoint
curl -i -X POST "http://localhost:8092/mcp/v1/message?session_id=<session_id>" \
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

**Expected behavior:**

- The POST returns `202 Accepted`
- The JSON-RPC initialize result is delivered on the SSE stream as `event: message`

Example SSE message payload:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": "2024-11-05",
    "capabilities": {
      "resources": {}
    },
    "serverInfo": {
      "name": "guardrail-mcp",
      "version": "1.9.6"
    }
  }
}
```

**Test guardrail validation:**

```bash
curl -X POST "http://localhost:8092/mcp/v1/message?session_id=<session_id>" \
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

### OpenCode MCP Configuration

To connect OpenCode to the remote MCP server, add this to your `.opencode/oh-my-opencode.jsonc`:

```jsonc
{
  "mcpServers": {
    "guardrails": {
      "type": "remote",
      "url": "http://100.96.49.42:8094/mcp/v1/sse",
      "headers": {
        "Authorization": "Bearer YOUR_MCP_API_KEY_HERE"
      }
    }
  }
}
```

**Important:**
- Replace `100.96.49.42:8094` with your actual server IP and external MCP port
- Replace `YOUR_MCP_API_KEY_HERE` with the value from your `.env` file (MCP_API_KEY)
- Use the **external port** (8094), not the internal container port (8080)
- The `Authorization` header must use `Bearer` format (not `X-API-Key`)

### Accessing the Web UI

Once deployed, access the guardrail management interface:

```
http://localhost:8093
```

Features available:
- View and manage active guardrail rules
- Monitor validation sessions
- Configure rule sets
- View validation logs

### Troubleshooting

**Connection refused:**
- Verify podman-compose services are running: `sudo podman-compose ps`
- Docker-only equivalent: `docker compose -f deploy/podman-compose.yml ps`
- Check firewall rules on your server
- Verify ports 8092 and 8093 are accessible
- **Port confusion:** Remember external ports (8094/8095) vs internal ports (8080/8081). Use external ports from outside the container.

**Authentication errors:**
- Ensure `MCP_API_KEY` is set correctly
- Verify JWT_SECRET matches between client and server
- Use `Authorization: Bearer <key>` format (not `X-API-Key` header)
- Check you're connecting to the MCP port (8094), not the Web UI port (8095)

**Database connection issues:**
- Check PostgreSQL is running: `sudo podman ps | grep postgres`
- Docker-only equivalent: `docker ps | grep postgres`
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
├── mcp-server/            ← MCP Server implementation (v1.9.6)
│   ├── cmd/server/        # Go entry point
│   ├── internal/          # Core server modules
│   ├── deploy/            # Docker deployment configs
│   └── README.md          # MCP server docs
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

**Current Version:** v1.9.6 (2026-02-08)

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

**Last Updated:** 2026-02-08
**Status:** v1.9.6 - Production Ready
