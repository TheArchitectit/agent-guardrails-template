# Guardrail MCP Server

A Model Context Protocol (MCP) server for enforcing guardrails across AI coding assistants and IDE extensions.

## Architecture

```
┌─────────────────────────────────────────────────────┐
│           Guardrail MCP Server Container            │
│  ┌─────────────┐  ┌─────────────┐                  │
│  │   MCP SSE   │  │   Web UI    │                  │
│  │   :8080     │  │   :8081     │                  │
│  └──────┬──────┘  └──────┬──────┘                  │
│         │                │                          │
│  ┌──────┴────────────────┴─────────────────┐        │
│  │              Redis Cache                │        │
│  └─────────────────────────────────────────┘        │
│         │                                           │
│  ┌──────┴────────────────┴─────────────────┐        │
│  │              PostgreSQL                 │        │
│  └─────────────────────────────────────────┘        │
└─────────────────────────────────────────────────────┘
```

## Quick Start

### Prerequisites

- Go 1.23+
- Podman or Docker
- PostgreSQL 16 (or use podman-compose)
- Redis 7 (or use podman-compose)

### Configuration

1. Copy `.env.example` to `.env` and fill in the values:

```bash
cp .env.example .env
# Edit .env with your values
```

2. Generate security keys:

```bash
export MCP_API_KEY=$(openssl rand -hex 32)
export IDE_API_KEY=$(openssl rand -hex 32)
export JWT_SECRET=$(openssl rand -hex 32)
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

```bash
# Build container
make docker-build

# Start all services (PostgreSQL, Redis, MCP Server)
make docker-up

# View logs
make docker-logs

# Stop services
make docker-down
```

## API Endpoints

### Health

- `GET /health/live` - Liveness probe
- `GET /health/ready` - Readiness probe (checks DB and Redis)
- `GET /metrics` - Prometheus metrics endpoint
- `GET /version` - Server version information

### MCP Protocol (Port 8080)

Server-Sent Events (SSE) endpoint for MCP clients.

- `GET /mcp/v1/sse` - SSE event stream endpoint
- `POST /mcp/v1/message` - JSON-RPC message endpoint

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
- **API Key Authentication** - All external endpoints require valid API key (MCP_API_KEY or IDE_API_KEY)
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

- `guardrail_init_session` - Initialize a validation session for a project
- `guardrail_validate_bash` - Validate bash command against forbidden patterns
- `guardrail_validate_file_edit` - Validate file edit operation
- `guardrail_validate_git_operation` - Validate git command against guardrails
- `guardrail_pre_work_check` - Run pre-work checklist from failure registry
- `guardrail_get_context` - Get guardrail context for the session's project

### MCP Resources

- `guardrail://quick-reference` - Quick reference card for guardrails
- `guardrail://rules/active` - Currently active prevention rules

### Connecting to MCP Server

```bash
# SSE endpoint for MCP clients
GET http://localhost:8080/mcp/v1/sse

# Send JSON-RPC messages
POST http://localhost:8080/mcp/v1/message?session_id=<session_id>
```

See [API.md](API.md) for complete API documentation.

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
│   ├── validation/      # Input validation utilities
│   └── web/             # HTTP server, handlers, middleware
├── deploy/              # Deployment files (Dockerfile, compose)
└── README.md            # This file
```

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

### SSE Connection Errors

**Problem:** EOF errors when connecting to `/mcp/v1/sse`

**Solution:**
- Verify MCP_API_KEY is set correctly in client headers
- Check that `Authorization: Bearer <key>` header is included
- Ensure no proxy is buffering SSE responses (check X-Accel-Buffering header)

### API Key Authentication Failures

**Problem:** `Missing authorization header` or `Invalid API key`

**Solution:**
- Verify `Authorization: Bearer <api_key>` header format
- Check that MCP_API_KEY or IDE_API_KEY environment variables are set
- For Web UI access, no API key is required (publicly accessible)

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

# Verify all required environment variables are set
cat .env | grep -E "(API_KEY|PASSWORD|SECRET)"

# Ensure PostgreSQL and Redis are healthy before starting MCP server
```

## License

MIT
