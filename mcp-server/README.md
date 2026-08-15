# Guardrail MCP Server

An MCP (Model Context Protocol) server that enforces guardrails across AI
coding assistants and IDE extensions. Written in Go — the old Python
implementation was retired back in v2.6.0.

[![Go Implementation](https://img.shields.io/badge/Implementation-Go-blue.svg?style=flat&logo=go)](https://golang.org)
[![Version](https://img.shields.io/badge/version-v3.4.0-blue.svg)](../CHANGELOG.md)

## Things worth knowing before you deploy

A few requirements the config validation will enforce anyway, listed here so
you don't get blindsided by error messages:

- **MCP_API_KEY and IDE_API_KEY** need 32+ characters with a mix of uppercase,
  lowercase, and digits. `openssl rand -hex` won't cut it (lowercase only) —
  use `openssl rand -base64`.
- **JWT_SECRET** needs 32+ bytes with real entropy. `openssl rand -hex 64`
  works fine.
- **JWT_ROTATION_HOURS** takes a duration with a unit suffix, e.g. `168h`.
- Containers talk to each other over the compose network by service name —
  don't point `DB_HOST` at `localhost`.

The full walkthrough is in [DEPLOYMENT_GUIDE.md](./deployment-guide.md).

## Architecture

```
Deployment host (or local VM)
|
|-- guardrail-mcp-server (app container)
|   |-- :8080 MCP StreamableHTTP endpoint (POST /mcp)
|   |-- :8081 Web UI + REST API + health + metrics
|   |-- attached networks: frontend, backend
|   |-- host bindings: 127.0.0.1:${MCP_PORT}->8080, 127.0.0.1:${WEB_PORT}->8081
|
|-- guardrail-postgres (state container)
|   |-- :5432 backend network only
|   |-- attached networks: backend
|   |-- volume: pg_data
|
|-- guardrail-redis (cache/rate-limiting container)
|   |-- :6379 backend network only
|   |-- attached networks: backend
|   |-- volume: redis_data
```

## Quick Start

### Prerequisites

- Go 1.25+
- Podman or Docker
- PostgreSQL 16 (if running without compose)
- Redis 7 (if running without compose)

### Configuration

1. Copy `.env.example` to `.env` and fill in the values:

```bash
cp .env.example .env
# Edit .env with your values
```

2. Generate security keys:

```bash
# base64, not hex — the API key validator wants mixed case + digits
export MCP_API_KEY=$(openssl rand -base64 48)
export IDE_API_KEY=$(openssl rand -base64 48)
export JWT_SECRET=$(openssl rand -hex 64)
export DB_PASSWORD=$(openssl rand -base64 32)
export REDIS_PASSWORD=$(openssl rand -base64 32)
```

### Database Migrations

Database migrations use golang-migrate.

```bash
# Set DATABASE_URL environment variable
export DATABASE_URL="postgresql://guardrails:password@localhost:5432/guardrails?sslmode=disable"

# Run migrations up
make migrate-up

# Run migrations down
make migrate-down
```

Migration files are located in `internal/database/migrations/`.

### Development

```bash
# Install dependencies
make deps

# Run tests
make test

# Run locally (requires PostgreSQL and Redis running and migrations applied)
make dev

# Format code
make fmt

# Run linter
make lint

# Check for vulnerabilities
make vuln
```

### Deployment

The short version: copy `.env.example` to `.env`, fill in the required values,
and bring the stack up. PostgreSQL, Redis, and the MCP server all run as
containers. For production hardening and the long-form walkthrough, see
[DEPLOYMENT_GUIDE.md](./deployment-guide.md).

The compose file reads variables from the `.env` in the same directory. If you
keep your `.env` somewhere else, pass it explicitly with `--env-file /path/to/.env`
— otherwise you'll get confusing "variable is not set" warnings.

One thing worth knowing up front: by default the stack only listens on
localhost. That's deliberate. If you need the server reachable from another
machine (a Tailscale IP, a second network interface, etc.), set `BIND_ADDR` in
your `.env` — but don't expose it on `0.0.0.0` without putting real auth in
front of it.

#### If you're a human doing this by hand

Docker:

```bash
cp .env.example .env        # edit in your secrets, ports, etc.

docker compose -f deploy/podman-compose.yml up -d --build
docker compose -f deploy/podman-compose.yml ps      # wait for everything to be healthy
docker compose -f deploy/podman-compose.yml logs -f mcp-server
```

Podman works with the same commands (the Docker-compatible CLI reads the same
compose file):

```bash
podman compose -f deploy/podman-compose.yml up -d --build
podman compose -f deploy/podman-compose.yml ps
```

Or if you prefer `podman-compose`:

```bash
podman-compose -f deploy/podman-compose.yml up -d --build
```

Give it a minute on first run — PostgreSQL has to initialize its data volume
and the migrations need to apply before the MCP server's health check passes.

#### If you're an agent doing this

Same commands, but mind the details that trip you up:

1. **Never commit or log the `.env`.** Secrets go in environment variables at runtime, not in files you write into the repo.
2. The compose file lives in `deploy/`, so relative-path assumptions break — always pass `--env-file` if your `.env` isn't next to it.
3. Ports are bound to `BIND_ADDR` (default `127.0.0.1`). Verify with `curl` against that address, not against a hostname you're guessing at.
4. Check container health before testing endpoints: `docker compose -f deploy/podman-compose.yml ps`. If `mcp-server` isn't `healthy`, read its logs before retrying blindly.

#### Verifying it works

```bash
# Web UI health (default ports: MCP 8080, Web 8081)
curl -s http://localhost:8081/health/ready

# MCP endpoint — stateless JSON-RPC, no session setup needed
curl -s -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "initialize",
    "params": {
      "protocolVersion": "2024-11-05",
      "capabilities": {},
      "clientInfo": { "name": "test", "version": "1.0" }
    }
  }'
```

You should get a JSON-RPC response naming the server and listing its
capabilities. If you get a 404 on `/mcp`, you're running an older build — the
stateless endpoint landed in v3.3.0.

#### Useful commands

```bash
# Stop everything (keeps your data volumes)
docker compose -f deploy/podman-compose.yml down

# Stop and wipe data volumes too
docker compose -f deploy/podman-compose.yml down -v

# Restart just the MCP server after rebuilding
docker compose -f deploy/podman-compose.yml up -d --build mcp-server
```

## API Endpoints

### Health

- `GET /health/live` - Liveness probe
- `GET /health/ready` - Readiness probe (checks DB and Redis)
- `GET /metrics` - Prometheus metrics endpoint
- `GET /version` - Server version information

### MCP Protocol (Port 8080)

Stateless StreamableHTTP endpoint for MCP clients.

- `POST /mcp` - JSON-RPC request/response over HTTP

No session ID required. Each request is independent and self-contained.

### Web UI API (Port 8081)

- `GET /api/documents` - List documents (paginated)
- `GET /api/documents/:id` - Get document by ID
- `PUT /api/documents/:id` - Update document
- `GET /api/documents/search?q={query}` - Full-text search documents

- `GET /api/rules` - List prevention rules
- `GET /api/rules/:id` - Get rule by ID
- `POST /api/rules` - Create rule
- `PUT /api/rules/:id` - Update rule
- `DELETE /api/rules/:id` - Delete rule
- `PATCH /api/rules/:id` - Enable/disable rule (partial update)

- `GET /api/projects` - List projects
- `GET /api/projects/:id` - Get project by ID
- `POST /api/projects` - Create project
- `PUT /api/projects/:id` - Update project
- `DELETE /api/projects/:id` - Delete project

- `GET /api/failures` - List failure registry entries
- `GET /api/failures/:id` - Get failure by ID
- `POST /api/failures` - Create failure entry
- `PUT /api/failures/:id` - Update failure status

- `GET /api/stats` - Get system statistics
- `POST /api/ingest` - Trigger document ingestion

### IDE API (Port 8081)

- `GET /ide/health` - IDE API health check
- `POST /ide/validate/file` - Validate file content
- `POST /ide/validate/selection` - Validate code selection
- `GET /ide/rules` - Get active rules for project
- `GET /ide/quick-reference` - Get quick reference documentation

## Security Features

### Authentication & Authorization
- **API Key Authentication** - Write and IDE endpoints require valid API key (MCP_API_KEY or IDE_API_KEY)
- **Public Read-Only Web Routes** - `/api/documents*`, `/api/rules*`, and `/version` are browsable without API key
- **JWT Tokens** - Session tokens for MCP clients with 15-minute expiry
- **Hashed Key Logging** - API keys are hashed in logs for audit purposes

### Infrastructure Security
- **Redis AUTH** - Password-protected Redis connections
- **PostgreSQL SSL** - TLS support for database connections
- **Non-root Container** - Runs as UID 65532 (distroless image)
- **Read-only Filesystem** - Container root is read-only
- **Dropped Capabilities** - ALL capabilities dropped for minimal attack surface

### Application Security
- **Rate Limiting** - Per-API-key rate limiting (MCP: 1000/min, IDE: 500/min)
- **Secrets Scanning** - Automatic detection of secrets in document content (AWS keys, GitHub tokens, private keys, etc.)
- **Content Security Policy** - Strict CSP headers to prevent XSS
- **Security Headers** - X-Content-Type-Options, X-Frame-Options, X-XSS-Protection, Referrer-Policy
- **Input Validation** - UUID validation, parameterized queries to prevent SQL injection
- **Regex Timeouts** - Protection against ReDoS attacks

### Resilience Patterns
- **Circuit Breakers** - Automatic failure detection for database and Redis
- **Graceful Degradation** - Service continues operating when cache is unavailable
- **Health Checks** - Liveness and readiness probes for orchestration
- **Graceful Shutdown** - 30-second timeout for in-flight requests

## MCP Protocol

The MCP server implements the Model Context Protocol for AI assistant integration.

### MCP Tools

The server registers 35 tools. The core validation set:

- `guardrail_init_session` - Initialize a validation session for a project
- `guardrail_validate_bash` - Validate bash commands against forbidden patterns
- `guardrail_validate_file_edit` - Validate file edit operations
- `guardrail_validate_git_operation` - Validate git commands against guardrails
- `guardrail_validate_commit` / `guardrail_validate_push` - Commit and push validation
- `guardrail_validate_scope` - Scope-check file changes against the task
- `guardrail_pre_work_check` - Run the pre-work checklist from the failure registry
- `guardrail_get_context` - Get guardrail context for the session's project
- `guardrail_prevent_regression` / `guardrail_verify_fixes_intact` - Regression prevention
- `guardrail_check_halt_conditions` / `guardrail_acknowledge_halt` - Halt conditions
- `guardrail_check_test_prod_separation` - Test/production isolation checks
- `guardrail_team_*` - Team management (init, list, assign, remove, health, config)
- `guardrail_record_*` / `guardrail_reset_attempts` - Attempt and halt recording

Use the `tools/list` method over `/mcp` for the full list with schemas.

### MCP Resources

11 resources exposed via `resources/list`:

- `guardrail://agent-guardrails` - Core guardrail rules
- `guardrail://four-laws` - The Four Laws of Agent Safety
- `guardrail://halt-conditions` - When to stop and ask
- `guardrail://quick-reference` - Quick reference card
- `guardrail://git-safety` - Git safety rules
- `guardrail://standards`, `guardrail://workflows` - Standards and workflow docs
- `guardrail://config`, `guardrail://stats` - Server configuration and stats
- `guardrail://advisors`, `guardrail://pre-work-check`, `guardrail://test-prod-separation`

### Connecting to MCP Server

```bash
# Send a JSON-RPC initialize request to the stateless StreamableHTTP endpoint
curl -i -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test-client","version":"1.0"}}}'

# JSON-RPC response arrives directly in the HTTP response body
```

See [API.md](api.md) for complete API documentation.

## Development

### Project Structure

```
.
├── cmd/
│   └── server/          # Main application entry point
├── internal/
│   ├── audit/           # Audit logging infrastructure
│   ├── cache/           # Redis client and cache management
│   ├── circuitbreaker/  # Circuit breaker pattern for resilience
│   ├── config/          # Configuration management
│   ├── database/        # PostgreSQL operations and migrations
│   │   └── migrations/  # golang-migrate migration files
│   ├── mcp/             # MCP protocol implementation
│   ├── models/          # Data models (Document, Rule, Project, Failure)
│   ├── security/        # Secrets scanning and detection
│   ├── team/            # Team management (migrated from Python v2.6.0)
│   │   ├── manager.go   # Core team operations
│   │   ├── encryption.go # Data encryption at rest
│   │   ├── rules.go     # Team layout rules
│   │   └── types.go     # Data structures
│   ├── validation/      # Input validation utilities
│   └── web/             # HTTP server, handlers, middleware
├── deploy/              # Deployment files (Dockerfile, compose)
└── README.md            # This file
```

**Note:** As of v2.6.0, all team management functionality has been migrated from Python (`scripts/team_manager.py`) to Go (`internal/team/`). See [../docs/PYTHON_TO_GO_MIGRATION.md](../docs/mcp-server/python-to-go-migration.md) for details.

### Adding New Features

1. Update models in `internal/models/`
2. Add database operations in `internal/database/`
3. Add handlers in `internal/web/`
4. Update routes in `internal/web/server.go`
5. Add tests

## Troubleshooting

### Database Connection Issues

**Problem:** `failed to connect to database`

**Solution:**
- Verify PostgreSQL is running: `docker ps | grep postgres`
- Check credentials in `.env` file
- Ensure database exists: `createdb guardrails`
- Verify SSL mode settings match your environment

### Redis Connection Issues

**Problem:** `failed to connect to Redis`

**Solution:**
- Verify Redis is running: `docker ps | grep redis`
- Check REDIS_PASSWORD matches between `.env` and Redis container
- For local development without Redis, set `REDIS_PASSWORD=` (empty)

### MCP Connection Errors

**Problem:** Connection refused or 404 when connecting to `/mcp`

**Solution:**
- Verify the MCP server is running on port 8080: `curl http://localhost:8080/mcp -X POST -H "Content-Type: application/json" -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}'`
- Ensure clients use `POST /mcp` (not the legacy `/mcp/v1/sse` SSE endpoint)
- The transport is stateless — no session IDs or persistent connections required

### API Key Authentication Failures

**Problem:** `Missing authorization header` or `Invalid API key`

**Solution:**
- Verify `Authorization: Bearer <api_key>` header format
- Check that MCP_API_KEY or IDE_API_KEY environment variables are set
- For Web UI access and read-only browsing APIs, no API key is required

### Guardrails Not Enforcing (`rules_evaluated=0` or dangerous commands allowed)

**Problem:** MCP tool calls return permissive results even for dangerous commands.

**Cause:** Runtime rule/project data is missing, or rule categories do not match validator categories.

**Solution:**
- Check data state:
  - `curl -s http://localhost:8081/api/stats`
  - If `rules_count` or `projects_count` is `0`, run rule sync and seed a project.
- Trigger rule sync:
  - `curl -X POST http://localhost:8081/api/rules/sync -H "Authorization: Bearer $MCP_API_KEY" -H "Content-Type: application/json" -d '{"force":true}'`
  - `curl -s http://localhost:8081/api/rules/sync/status`
- Ensure the project used by `guardrail_init_session` has `active_rules` populated.
- Verify categories for command enforcement:
  - `guardrail_validate_bash` evaluates `bash` (and compatible legacy categories) plus `all`.
  - `guardrail_validate_git_operation` evaluates `git` (and compatible legacy categories) plus `all`.
  - Rules intended to apply globally should use category `all`.
- Re-test using MCP `initialize` -> `guardrail_init_session` -> `guardrail_validate_bash`/`guardrail_validate_git_operation` and confirm `rules_evaluated > 0`.

### Database Migration Failures

**Problem:** `no schema has been selected to create in`

**Solution:**
```bash
# Connect to PostgreSQL and create schema
psql -U guardrails -d guardrails -c "CREATE SCHEMA IF NOT EXISTS public;"
```

### Container Won't Start

**Problem:** Container exits immediately

**Solution:**
```bash
# Check logs
make docker-logs
# or: docker compose -f deploy/podman-compose.yml logs -f

# Verify all required environment variables are set
cat .env | grep -E "(API_KEY|PASSWORD|SECRET)"

# Ensure PostgreSQL and Redis are healthy before starting MCP server
```

## License

BSD-3-Clause — see [../LICENSE](../LICENSE).

---

### Client configuration example

Once the server is running, point your MCP client at it:

```jsonc
{
  "mcpServers": {
    "guardrails": {
      "type": "remote",
      "url": "http://your-server-host:8080/mcp",
      "headers": {
        "Authorization": "Bearer <your-MCP_API_KEY>"
      }
    }
  }
}
```

Replace `your-server-host` and the API key with your own values.

See [DEPLOYMENT_GUIDE.md](./deployment-guide.md) for the full deployment
walkthrough and more troubleshooting.
