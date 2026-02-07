# Guardrail MCP Server Implementation Plan

> **Version:** 1.1
> **Status:** Planning (Post-Architecture Review)

## Overview

Build a Guardrail Platform that serves as a central authority for guardrail enforcement:
- **Database-backed**: All Markdown/guardrail data stored in PostgreSQL
- **Caching Layer**: Redis for rule caching and rate limiting
- **Web UI**: Browse and edit guardrails (replaces reading MD files directly)
- **MCP Endpoint**: TUI clients (Claude Code, OpenCode, etc.) connect for live validation
- **Deployment**: Runs as containers (see `.env` for deployment target)

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Deployment Server                        │
│  ┌─────────────────────────────────────────────────────┐   │
│  │           Guardrail MCP Server Container            │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐ │   │
│  │  │   MCP SSE   │  │   Web UI    │  │   Ingest    │ │   │
│  │  │   :8080     │  │   :8081     │  │   (Job)     │ │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘ │   │
│  │           │              │                         │   │
│  │           └──────────────┘                         │   │
│  │                      │                             │   │
│  │  ┌───────────────────┴───────────────────┐        │   │
│  │  │              Redis Cache              │        │   │
│  │  │  - Active rules cache (TTL: 5m)      │        │   │
│  │  │  - Rate limiting counters            │        │   │
│  │  │  - Session tokens                    │        │   │
│  │  └─────────────────────────────────────┘        │   │
│  │                      │                             │   │
│  │  ┌───────────────────┴───────────────────┐        │   │
│  │  │           PostgreSQL                  │        │   │
│  │  │  - documents                          │        │   │
│  │  │  - prevention_rules                   │        │   │
│  │  │  - failure_registry                   │        │   │
│  │  │  - projects                           │        │   │
│  │  └─────────────────────────────────────┘        │   │
│  └─────────────────────────────────────────────────────┘   │
│                            │                                │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Ingest Source: Markdown files from source repo      │   │
│  │  - docs/workflows/*.md                               │   │
│  │  - docs/standards/*.md                               │   │
│  │  - docs/*.md                                         │   │
│  │  - .guardrails/*                                     │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                            │
              ┌─────────────┴─────────────┐
              ▼                           ▼
    ┌─────────────────┐        ┌──────────────────┐
    │  TUI Clients    │        │  Web Browser     │
    │  (Claude Code,  │        │  (View/Edit      │
    │   OpenCode)     │        │   Guardrails)    │
    └─────────────────┘        └──────────────────┘
```

### Components

| Component | Port | Purpose |
|-----------|------|---------|
| MCP Server | 8080 | SSE endpoint for TUI clients |
| Web UI | 8081 | Browser-based guardrail management |
| PostgreSQL | internal | Data persistence |
| Redis | internal | Caching, rate limiting, sessions |

### Data Flow

1. **Ingest Phase**: Load all MD files into PostgreSQL (run as needed)
2. **Web UI**: Users browse/edit guardrails stored in database
3. **MCP Server**: Validates tool calls from external repos against stored guardrails
4. **Caching**: Active rules cached in Redis (5 min TTL)

---

## Tech Stack

- **Go 1.23+** - Server implementation
- **mark3labs/mcp-go** - MCP protocol
- **Echo** - HTTP framework for web UI
- **PostgreSQL 16** - Database
- **Redis 7** - Caching and rate limiting
- **caarlos0/env** - Configuration (no Viper)
- **slog** - Structured logging
- **golang-migrate** - Database migrations

---

## Project Structure

```
cmd/
├── server/
│   └── main.go          # MCP + Web server
└── ingest/
    └── main.go          # Ingest tool (run as job)

internal/
├── models/              # Data models
├── database/            # PostgreSQL operations + migrations
│   └── migrations/      # golang-migrate files
├── cache/               # Redis client
├── ingester/            # MD file ingestion
├── guardrails/          # Validation logic
├── web/                 # HTTP server + UI
│   ├── handlers.go      # REST API handlers
│   ├── middleware.go    # Auth, CSRF, rate limiting
│   └── static/          # Embedded web UI
├── mcp/                 # MCP protocol handlers
├── config/              # Configuration
└── version/

deploy/
├── Dockerfile
├── Dockerfile.ingest    # Separate image for ingest job
└── podman-compose.yml

scripts/
└── ingest.sh
```

---

## Database Schema

### Tables

**documents**
```sql
CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug VARCHAR(255) UNIQUE NOT NULL,
    title VARCHAR(500) NOT NULL,
    content TEXT NOT NULL,
    search_vector tsvector,
    category VARCHAR(50) NOT NULL CHECK (category IN ('workflow', 'standard', 'guide', 'reference')),
    path VARCHAR(500) NOT NULL,
    version INTEGER DEFAULT 1,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_documents_category ON documents(category);
CREATE INDEX idx_documents_slug ON documents(slug);
CREATE INDEX idx_documents_updated ON documents(updated_at DESC);
CREATE INDEX idx_documents_search ON documents USING GIN(search_vector);
CREATE INDEX idx_documents_metadata ON documents USING GIN(metadata);
```

**prevention_rules**
```sql
CREATE TABLE prevention_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id VARCHAR(50) UNIQUE NOT NULL CHECK (LENGTH(TRIM(rule_id)) > 0),
    name VARCHAR(255) NOT NULL,
    pattern TEXT NOT NULL,
    pattern_hash VARCHAR(64), -- For exact-match pre-filtering
    message TEXT NOT NULL,
    severity VARCHAR(10) NOT NULL CHECK (severity IN ('error', 'warning', 'info')),
    enabled BOOLEAN NOT NULL DEFAULT true,
    document_id UUID REFERENCES documents(id) ON DELETE SET NULL,
    category VARCHAR(50),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_rules_document ON prevention_rules(document_id);
CREATE INDEX idx_rules_enabled ON prevention_rules(enabled) WHERE enabled = true;
CREATE INDEX idx_rules_severity ON prevention_rules(severity);
CREATE INDEX idx_rules_category ON prevention_rules(category);
CREATE INDEX idx_rules_covering ON prevention_rules(document_id, rule_id, name, severity, enabled)
    INCLUDE (pattern, message);
```

**failure_registry**
```sql
CREATE TABLE failure_registry (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    failure_id VARCHAR(50) UNIQUE NOT NULL,
    category VARCHAR(50) NOT NULL,
    severity VARCHAR(10) NOT NULL CHECK (severity IN ('critical', 'high', 'medium', 'low')),
    error_message TEXT NOT NULL,
    root_cause TEXT,
    affected_files TEXT[],
    regression_pattern VARCHAR(255),
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'resolved', 'deprecated')),
    project_slug VARCHAR(100),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
) PARTITION BY RANGE (created_at);

-- Indexes
CREATE INDEX idx_failures_status ON failure_registry(status);
CREATE INDEX idx_failures_category ON failure_registry(category);
CREATE INDEX idx_failures_created ON failure_registry(created_at DESC);
CREATE INDEX idx_failures_files ON failure_registry USING GIN(affected_files);
CREATE INDEX idx_failures_project ON failure_registry(project_slug);
CREATE INDEX idx_failures_covering ON failure_registry(status, created_at DESC, severity)
    INCLUDE (failure_id, category, error_message);
```

**projects**
```sql
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) UNIQUE NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL CHECK (LENGTH(TRIM(slug)) > 0),
    guardrail_context TEXT,
    active_rules VARCHAR(50)[],
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_projects_slug ON projects(slug);
CREATE INDEX idx_projects_active_rules ON projects USING GIN(active_rules);
CREATE INDEX idx_projects_metadata ON projects USING GIN(metadata);
```

**schema_migrations** (auto-created by golang-migrate)
```sql
CREATE TABLE schema_migrations (
    version BIGINT PRIMARY KEY,
    dirty BOOLEAN NOT NULL
);
```

### Migration Files

```
internal/database/migrations/
├── 001_create_tables.up.sql
├── 001_create_tables.down.sql
├── 002_add_indexes.up.sql
├── 002_add_indexes.down.sql
├── 003_add_constraints.up.sql
├── 003_add_constraints.down.sql
├── 004_add_search_vector.up.sql
└── 004_add_search_vector.down.sql
```

---

## MCP Protocol Specification

### Transport

- **Server-to-Client**: SSE (Server-Sent Events) on `/mcp/v1/sse`
- **Client-to-Server**: HTTP POST on `/mcp/v1/message`
- **Session Management**: JWT tokens with 15-minute expiry

### Session Initialization

**Tool:** `guardrail_init_session`

```json
{
  "name": "guardrail_init_session",
  "description": "Initialize a validation session for a project",
  "inputSchema": {
    "type": "object",
    "properties": {
      "project_slug": { "type": "string" },
      "agent_type": { "type": "string", "enum": ["claude-code", "opencode", "cursor", "other"] },
      "client_version": { "type": "string" }
    },
    "required": ["project_slug"]
  }
}
```

**Response:**
```json
{
  "session_token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_at": "2026-02-07T11:30:00Z",
  "project_context": "...",
  "active_rules_count": 15,
  "capabilities": ["bash_validation", "git_validation", "edit_validation"]
}
```

### Validation Tools

#### 1. `guardrail_validate_bash`

```json
{
  "name": "guardrail_validate_bash",
  "description": "Validate bash command against forbidden patterns",
  "inputSchema": {
    "type": "object",
    "properties": {
      "session_token": { "type": "string" },
      "command": { "type": "string" },
      "working_directory": { "type": "string" }
    },
    "required": ["session_token", "command"]
  }
}
```

#### 2. `guardrail_validate_file_edit`

```json
{
  "name": "guardrail_validate_file_edit",
  "description": "Validate file edit operation",
  "inputSchema": {
    "type": "object",
    "properties": {
      "session_token": { "type": "string" },
      "file_path": { "type": "string" },
      "old_string": { "type": "string" },
      "new_string": { "type": "string" },
      "change_description": { "type": "string" }
    },
    "required": ["session_token", "file_path", "old_string", "new_string"]
  }
}
```

#### 3. `guardrail_validate_git_operation`

```json
{
  "name": "guardrail_validate_git_operation",
  "description": "Validate git command against guardrails",
  "inputSchema": {
    "type": "object",
    "properties": {
      "session_token": { "type": "string" },
      "command": { "type": "string", "enum": ["push", "commit", "merge", "rebase", "reset"] },
      "args": { "type": "array", "items": { "type": "string" } },
      "is_force": { "type": "boolean" }
    },
    "required": ["session_token", "command"]
  }
}
```

#### 4. `guardrail_validate_scope`

```json
{
  "name": "guardrail_validate_scope",
  "description": "Check if file path is in scope for session",
  "inputSchema": {
    "type": "object",
    "properties": {
      "session_token": { "type": "string" },
      "file_path": { "type": "string" }
    },
    "required": ["session_token", "file_path"]
  }
}
```

#### 5. `guardrail_pre_work_check`

```json
{
  "name": "guardrail_pre_work_check",
  "description": "Run pre-work checklist from failure registry",
  "inputSchema": {
    "type": "object",
    "properties": {
      "session_token": { "type": "string" },
      "affected_files": { "type": "array", "items": { "type": "string" } }
    },
    "required": ["session_token", "affected_files"]
  }
}
```

#### 6. `guardrail_batch_validate`

```json
{
  "name": "guardrail_batch_validate",
  "description": "Validate multiple operations at once",
  "inputSchema": {
    "type": "object",
    "properties": {
      "session_token": { "type": "string" },
      "operations": {
        "type": "array",
        "items": {
          "type": "object",
          "properties": {
            "tool": { "type": "string" },
            "args": { "type": "object" }
          }
        }
      }
    },
    "required": ["session_token", "operations"]
  }
}
```

### Response Format Standards

**Success Response (no violations):**
```json
{
  "valid": true,
  "violations": [],
  "meta": {
    "checked_at": "2026-02-07T10:30:00Z",
    "rules_evaluated": 15,
    "duration_ms": 12,
    "cached": true
  }
}
```

**Violation Response:**
```json
{
  "valid": false,
  "violations": [
    {
      "rule_id": "PREVENT-001",
      "rule_name": "No Force Push",
      "severity": "error",
      "message": "git push --force violates guardrail: NO FORCE PUSH",
      "category": "git_operation",
      "action": "halt",
      "suggested_alternative": "Use git push --force-with-lease instead",
      "documentation_uri": "guardrail://docs/AGENT_GUARDRAILS"
    }
  ],
  "meta": {
    "checked_at": "2026-02-07T10:30:00Z",
    "rules_evaluated": 15,
    "duration_ms": 12
  }
}
```

**Severity Actions:**

| Severity | Action | Client Behavior |
|----------|--------|-----------------|
| error | halt | MUST halt operation |
| warning | confirm | SHOULD show confirmation dialog |
| info | log | MAY log for awareness |

### Error Response Format

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "code": -32602,
    "message": "Invalid guardrail parameters",
    "data": {
      "guardrail_error": {
        "type": "validation_error",
        "code": "INVALID_SESSION",
        "message": "Session token expired or invalid",
        "suggestion": "Call guardrail_init_session to create new session"
      }
    }
  }
}
```

**Error Codes:**

| Code | Meaning | HTTP Equivalent |
|------|---------|-----------------|
| INVALID_SESSION | Session token invalid/expired | 401 |
| INVALID_API_KEY | API key invalid | 401 |
| RATE_LIMITED | Too many requests | 429 |
| RULE_VIOLATION | Guardrail violation found | 403 |
| INVALID_ARGUMENT | Bad parameters | 400 |
| INTERNAL_ERROR | Server error | 500 |

### Resources

| Resource | Description |
|----------|-------------|
| `guardrail://docs/{slug}` | Document content (markdown) |
| `guardrail://docs/search?q={query}` | Full-text search results |
| `guardrail://rules` | All prevention rules |
| `guardrail://rules/{rule_id}` | Specific rule |
| `guardrail://rules/active` | Active rules only |
| `guardrail://failures?status={status}&limit={n}` | Failure registry (paginated) |
| `guardrail://projects/{slug}` | Project configuration |
| `guardrail://projects/{slug}/active-rules` | Rules for project |
| `guardrail://quick-reference` | Quick reference card |
| `guardrail://health` | Health status |
| `guardrail://capabilities` | Server capabilities |

---

## Web UI REST API

### Authentication

All endpoints require `Authorization: Bearer {WEB_API_KEY}` header.

### Documents

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/documents` | List documents (paginated) |
| GET | `/api/documents/:id` | Get document |
| PUT | `/api/documents/:id` | Update document |
| GET | `/api/documents/search?q={query}` | Full-text search |
| GET | `/api/documents/category/{category}` | Filter by category |

### Rules

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/rules` | List rules |
| GET | `/api/rules/:id` | Get rule |
| POST | `/api/rules` | Create rule |
| PUT | `/api/rules/:id` | Update rule |
| DELETE | `/api/rules/:id` | Delete rule |
| POST | `/api/rules/:id/toggle` | Enable/disable rule |
| POST | `/api/rules/:id/test` | Test rule against input |

### Projects

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/projects` | List projects |
| GET | `/api/projects/:slug` | Get project |
| POST | `/api/projects` | Create project |
| PUT | `/api/projects/:slug` | Update project |
| DELETE | `/api/projects/:slug` | Delete project |

### Failures

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/failures` | List failures (paginated) |
| GET | `/api/failures/:id` | Get failure |
| POST | `/api/failures` | Log new failure |
| PUT | `/api/failures/:id` | Update failure status |

### System

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/health` | Health check |
| GET | `/api/stats` | Usage statistics |
| POST | `/api/ingest` | Trigger ingest job |

### CSRF Protection

- All state-changing operations require CSRF token
- Token provided in `X-CSRF-Token` header or cookie
- Double-submit cookie pattern

---

## Security Requirements

### API Authentication

```go
// internal/web/middleware/auth.go

func APIKeyAuth(keyType string) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            auth := c.Request().Header.Get("Authorization")
            if auth == "" {
                return echo.NewHTTPError(401, "Missing authorization")
            }

            token := strings.TrimPrefix(auth, "Bearer ")
            if !validateAPIKey(token, keyType) {
                return echo.NewHTTPError(401, "Invalid API key")
            }

            // Log hashed key only
            slog.Info("API request", "key_hash", hashKey(token))
            return next(c)
        }
    }
}

func hashKey(key string) string {
    h := sha256.Sum256([]byte(key))
    return hex.EncodeToString(h[:8])
}
```

### Rate Limiting

```go
// internal/web/middleware/ratelimit.go

var limiters = map[string]*rate.Limiter{
    "mcp": rate.NewLimiter(rate.Limit(1000/60), 100),      // 1000/min burst 100
    "web": rate.NewLimiter(rate.Limit(100/60), 20),        // 100/min burst 20
}
```

### CSRF Protection

```go
// internal/web/middleware/csrf.go

func CSRF() echo.MiddlewareFunc {
    return csrfMiddleware(csrf.Config{
        TokenLookup: "header:X-CSRF-Token",
        CookieName: "csrf_token",
        CookieSameSite: http.SameSiteStrictMode,
    })
}
```

### Container Security

```dockerfile
# deploy/Dockerfile

FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

FROM gcr.io/distroless/static:nonroot
USER 65532:65532
COPY --from=builder --chown=65532:65532 /app/server /server
EXPOSE 8080 8081
ENTRYPOINT ["/server"]
```

### podman-compose Security

```yaml
# deploy/podman-compose.yml
version: "3.8"

services:
  redis:
    image: redis:7-alpine
    networks:
      - backend
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL

  postgres:
    image: postgres:16-alpine
    environment:
      - POSTGRES_PASSWORD=${DB_PASSWORD}
    networks:
      - backend
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL

  mcp-server:
    image: guardrail-mcp:latest
    read_only: true
    user: "65532:65532"
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL
    networks:
      - frontend
      - backend
    environment:
      - DB_SSLMODE=require
      - DB_PASSWORD=${DB_PASSWORD}
    tmpfs:
      - /tmp:noexec,nosuid,size=100m

networks:
  frontend:
    internal: false
  backend:
    internal: true
```

### Input Validation

```go
// internal/validation/safe_regex.go

func SafeRegex(pattern string, input string, timeout time.Duration) (bool, error) {
    resultChan := make(chan bool, 1)

    go func() {
        re, err := regexp.Compile(pattern)
        if err != nil {
            resultChan <- false
            return
        }
        resultChan <- re.MatchString(input)
    }()

    select {
    case result := <-resultChan:
        return result, nil
    case <-time.After(timeout):
        return false, fmt.Errorf("regex timeout - possible ReDoS")
    }
}
```

---

## Deployment

### Phase 1: Build

```bash
make build-container
# Produces: guardrail-mcp:latest
```

### Phase 2: Deploy Infrastructure

```bash
# Copy .env with real values to target server
scp .env user@target:/opt/guardrail-mcp/
scp deploy/podman-compose.yml user@target:/opt/guardrail-mcp/

# Start services
ssh user@target "cd /opt/guardrail-mcp && podman-compose up -d"
```

### Phase 3: Run Migrations

```bash
# Apply database migrations
podman run --rm --env-file .env guardrail-mcp:latest \
    /usr/local/bin/migrate -path /migrations -database "postgres://..." up
```

### Phase 4: Ingest Data

```bash
# Run ingest job
podman run --rm --env-file .env \
    -v /path/to/repo:/data/repo:ro \
    guardrail-mcp:latest \
    /usr/local/bin/ingest --repo /data/repo
```

---

## Configuration

**Required in `.env`:**

```bash
# Security (generate strong values)
MCP_API_KEY=
WEB_API_KEY=
DB_PASSWORD=

# Database (use SSL in production)
DB_HOST=postgres
DB_PORT=5432
DB_NAME=guardrails
DB_USER=guardrails
DB_PASSWORD=
DB_SSLMODE=require

# Redis
REDIS_HOST=redis
REDIS_PORT=6379

# Server
MCP_PORT=8080
WEB_PORT=8081
LOG_LEVEL=info
```

---

## Security Checklist

### Authentication & Authorization
- [ ] JWT implementation with 15-minute expiration
- [ ] API key authentication for all endpoints
- [ ] API key masking in logs (hash only)
- [ ] Separate MCP/Web API keys
- [ ] Key rotation mechanism

### Network Security
- [ ] DB_SSLMODE=require enforced
- [ ] Internal network for PostgreSQL/Redis
- [ ] No external exposure of backend services

### Input Validation
- [ ] Regex timeout protection (ReDoS prevention)
- [ ] UUID validation for document IDs
- [ ] Pattern validation for rule regex

### Web Security
- [ ] CSRF protection on all state-changing endpoints
- [ ] SameSite=Strict cookies
- [ ] Content Security Policy headers
- [ ] XSS protection via output encoding

### Rate Limiting
- [ ] Per-API-key rate limits (MCP: 1000/min, Web: 100/min)
- [ ] Token bucket algorithm
- [ ] Rate limit headers in responses

### Container Security
- [ ] Distroless/minimal base image
- [ ] Non-root user (UID 65532)
- [ ] Read-only filesystem
- [ ] No new privileges flag
- [ ] Capability dropping

### Audit & Logging
- [ ] Structured audit logging
- [ ] Security event logging (auth failures, etc.)
- [ ] PII scrubbing in logs
- [ ] Log retention policy

---

## Files to Create

| File | Lines | Purpose |
|------|-------|---------|
| `go.mod` | 35 | Go module with dependencies |
| `cmd/server/main.go` | 100 | Server entry point |
| `cmd/ingest/main.go` | 80 | Ingest tool |
| `internal/config/config.go` | 60 | Configuration struct |
| `internal/models/*.go` | 150 | Data models |
| `internal/database/*.go` | 400 | Database layer + migrations |
| `internal/cache/redis.go` | 100 | Redis client |
| `internal/ingester/*.go` | 280 | Ingestion logic |
| `internal/guardrails/*.go` | 300 | Validation + enforcement |
| `internal/web/*.go` | 400 | HTTP handlers + middleware |
| `internal/mcp/*.go` | 350 | MCP handlers |
| `deploy/Dockerfile` | 25 | Server container |
| `deploy/Dockerfile.ingest` | 20 | Ingest job container |
| `deploy/podman-compose.yml` | 60 | Orchestration |
| `.env.example` | 50 | Config template |
| `Makefile` | 80 | Build automation |
| `README.md` | 250 | Setup guide |

---

## Performance Targets

| Metric | Target |
|--------|--------|
| Validation p99 latency | < 50ms (cached) |
| Validation p99 latency | < 200ms (uncached) |
| Database query time | < 10ms |
| Cache hit ratio | > 90% |
| Max concurrent sessions | 1000 |
| Ingest throughput | 100 docs/min |

---

**Last Updated:** 2026-02-07 (v1.1 - Post Architecture Review)
