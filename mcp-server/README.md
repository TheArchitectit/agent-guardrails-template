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

### Development

```bash
# Install dependencies
make deps

# Run tests
make test

# Run locally (requires PostgreSQL and Redis running)
make dev
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

### MCP Protocol (Port 8080)

Server-Sent Events (SSE) endpoint for MCP clients.

### Web UI API (Port 8081)

- `GET /api/documents` - List documents
- `GET /api/documents/:id` - Get document
- `PUT /api/documents/:id` - Update document
- `GET /api/documents/search?q={query}` - Search documents

- `GET /api/rules` - List prevention rules
- `POST /api/rules` - Create rule
- `PUT /api/rules/:id` - Update rule

- `GET /api/projects` - List projects
- `GET /api/projects/:slug` - Get project

- `GET /api/failures` - List failure registry entries

### IDE API (Port 8081)

- `POST /ide/validate/file` - Validate file content
- `POST /ide/validate/selection` - Validate code selection
- `GET /ide/rules` - Get active rules for project

## Security

- All external endpoints require API key authentication
- JWT tokens for MCP sessions (15-minute expiry)
- Redis AUTH password
- PostgreSQL SSL mode support
- Rate limiting per API key
- Secrets scanning in document content
- Content Security Policy headers

## Development

### Project Structure

```
.
├── cmd/server/          # Main application entry point
├── internal/
│   ├── audit/           # Audit logging
│   ├── cache/           # Redis client
│   ├── circuitbreaker/  # Circuit breakers
│   ├── config/          # Configuration
│   ├── database/        # PostgreSQL operations
│   ├── models/          # Data models
│   ├── security/        # Secrets scanning
│   ├── validation/      # Input validation
│   └── web/             # HTTP server
├── deploy/              # Deployment files
└── migrations/          # Database migrations
```

### Adding New Features

1. Update models in `internal/models/`
2. Add database operations in `internal/database/`
3. Add handlers in `internal/web/`
4. Update routes in `internal/web/server.go`
5. Add tests

## License

MIT
