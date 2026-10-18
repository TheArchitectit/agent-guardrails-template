# Guardrail MCP Server Deployment — Configuration and Troubleshooting

> Configuration requirements, Compose reference, deployment testing and the troubleshooting guide.

**Part of:** [Guardrail MCP Server Deployment Guide](./deployment-guide.md)

---

## Configuration Requirements

### Critical Settings

These settings were identified as critical during real production deployments:

1. **API Keys must be 32+ characters with mixed case and digits**
   ```bash
   # GOOD — generate with openssl (base64 gives mixed case + digits):
   MCP_API_KEY=$(openssl rand -base64 48)

   # BAD (lowercase-only hex fails the mixed-case check):
   MCP_API_KEY=$(openssl rand -hex 32)

   # BAD (too short, no digits):
   MCP_API_KEY=dev-key-short
   ```

2. **JWT_SECRET must be at least 32 bytes with real entropy**
   ```bash
   # GOOD — random hex passes the Shannon-entropy check:
   JWT_SECRET=$(openssl rand -hex 64)

   # BAD (human-readable phrase fails the entropy check):
   JWT_SECRET=this-is-my-jwt-secret-phrase
   ```

3. **JWT_ROTATION_HOURS must include 'h' unit**
   ```bash
   # GOOD:
   JWT_ROTATION_HOURS=168h

   # BAD (missing 'h'):
   JWT_ROTATION_HOURS=168
   ```

4. **Postgres and Redis need specific security settings in compose**
   ```yaml
   # postgres: no no-new-privileges (its entrypoint drops root via setuid);
   # minimal capability set for first-run volume init:
   cap_drop: [ALL]
   cap_add: [CHOWN, SETGID, SETUID, DAC_OVERRIDE, FOWNER]

   # redis: run as the image's redis user, config via CLI flags
   user: "999:1000"
   ```

5. **Server name must use underscores, not dashes**
   ```bash
   # In internal/mcp/server.go:
   # GOOD:
   server.NewMCPServer("guardrail_mcp", ...)

   # BAD (causes schema validation error):
   server.NewMCPServer("guardrail-mcp", ...)
   ```

### Environment Variables Reference

```bash
# Server Configuration
MCP_PORT=8080                    # External MCP port (maps to internal 8080)
WEB_PORT=8081                    # External Web UI port (maps to internal 8081)
WEB_ENABLED=true                 # Enable Web UI
LOG_LEVEL=info                   # Log level: debug, info, warn, error
REQUEST_TIMEOUT=30s              # Request timeout
SHUTDOWN_TIMEOUT=30s             # Graceful shutdown timeout
# BIND_ADDR=127.0.0.1            # Interface to bind ports on (default: localhost-only)

# Database Configuration
DB_HOST=postgres                 # Database host (service name in compose; localhost for pod networking)
DB_PORT=5432                     # Database port
DB_NAME=guardrails               # Database name
DB_USER=guardrail                # Database user
DB_PASSWORD=<generate>           # openssl rand -base64 24 — never check this in
DB_SSLMODE=disable               # SSL mode: disable, require, verify-full

# Redis Configuration
REDIS_HOST=redis                 # Redis host (service name in compose; localhost for pod networking)
REDIS_PORT=6379                  # Redis port
REDIS_PASSWORD=<generate>        # openssl rand -base64 24 — never check this in
REDIS_USE_TLS=false              # Use TLS for Redis

# Security Configuration (Critical: must meet requirements)
MCP_API_KEY=<generate>           # openssl rand -base64 48 — 32+ chars, mixed case+digits
IDE_API_KEY=<generate>           # openssl rand -base64 48 — 32+ chars, mixed case+digits
JWT_SECRET=<generate>            # openssl rand -hex 64 — 32+ bytes of entropy
JWT_ISSUER=guardrail-mcp      # JWT issuer
JWT_EXPIRY=15m                # JWT expiration
JWT_ROTATION_HOURS=168h       # JWT rotation (MUST include 'h')

# Rate Limiting
RATE_LIMIT_MCP=1000           # MCP API rate limit (req/min)
RATE_LIMIT_IDE=500            # IDE API rate limit (req/min)
RATE_LIMIT_SESSION=100        # Per-session rate limit (req/min)
RATE_LIMIT_WINDOW=1m          # Rate limit window
RATE_LIMIT_BURST_FACTOR=1.5   # Burst factor for rate limiting

# Cache TTL
CACHE_TTL_RULES=5m            # Rules cache TTL
CACHE_TTL_DOCS=10m            # Documents cache TTL
CACHE_TTL_SEARCH=2m           # Search cache TTL

# Feature Flags
ENABLE_VALIDATION=true        # Enable validation endpoint
ENABLE_METRICS=true           # Enable metrics collection
ENABLE_AUDIT_LOGGING=true     # Enable audit logging
ENABLE_CACHE=true             # Enable Redis caching

# CORS Configuration
CORS_ALLOWED_ORIGINS=*        # Allowed origins (use specific domains in production)
CORS_MAX_AGE=86400            # CORS max age
```

## Docker Compose Configuration

### Working Configuration (Reference)

```yaml
services:
  redis:
    image: docker.io/library/redis:7-alpine
    restart: unless-stopped
    # Spec 17 R17-08: no credential in argv. The password reaches redis only
    # through the config file written from the environment at startup, and the
    # healthcheck authenticates with REDISCLI_AUTH from the environment —
    # neither the server nor the probe exposes the secret in process arguments
    # (docker/podman inspect, /proc/<pid>/cmdline) or in logs.
    # `mkdir -p` is required: redis:7-alpine ships no /usr/local/etc/redis dir.
    command: >
      sh -c "
        mkdir -p /tmp/redis &&
        printf 'requirepass %s\nmaxmemory 256mb\nmaxmemory-policy allkeys-lru\n' \"$$REDIS_PASSWORD\" > /tmp/redis/redis.conf &&
        exec redis-server /tmp/redis/redis.conf
      "
    environment:
      - REDIS_PASSWORD=${REDIS_PASSWORD}
      - REDISCLI_AUTH=${REDIS_PASSWORD}
    user: "999:1000"
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 10s

  postgres:
    image: docker.io/library/postgres:16-alpine
    restart: unless-stopped
    environment:
      - POSTGRES_USER=${DB_USER}
      - POSTGRES_PASSWORD=${DB_PASSWORD}
      - POSTGRES_DB=${DB_NAME}
    volumes:
      - pg_data:/var/lib/postgresql/data
    # NOTE: no-new-privileges must NOT be set — the entrypoint drops
    # root -> postgres via setuid. These caps cover first-run volume init:
    cap_drop: [ALL]
    cap_add: [CHOWN, SETGID, SETUID, DAC_OVERRIDE, FOWNER]
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER} -d ${DB_NAME}"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 30s

  mcp-server:
    image: guardrail-mcp:${VERSION:-latest}
    restart: unless-stopped
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    ports:
      - "${BIND_ADDR:-127.0.0.1}:${MCP_PORT:-8080}:8080"  # MCP
      - "${BIND_ADDR:-127.0.0.1}:${WEB_PORT:-8081}:8081"  # Web UI
    environment:
      - DB_HOST=postgres
      - REDIS_HOST=redis
      # ... remaining config from .env (see deploy/podman-compose.yml)
    read_only: true
    user: "65532:65532"

volumes:
  pg_data:
    driver: local
  redis_data:
    driver: local
```

### Common Pitfalls

❌ **DON'T use dashes in server name:**
```go
// WRONG - causes schema validation error:
server.NewMCPServer("guardrail-mcp", ...)
```

✅ **DO use underscores:**
```go
// CORRECT:
server.NewMCPServer("guardrail_mcp", ...)
```

❌ **DON'T set no-new-privileges on postgres:**
```yaml
# WRONG - blocks the entrypoint's setuid drop, container crash-loops:
postgres:
  security_opt:
    - no-new-privileges:true
```

✅ **DO let postgres drop privileges itself:**
```yaml
# CORRECT - minimal caps, no no-new-privileges:
postgres:
  cap_drop: [ALL]
  cap_add: [CHOWN, SETGID, SETUID, DAC_OVERRIDE, FOWNER]
```

❌ **DON'T mount a redis config file:**
```yaml
# WRONG - redis:7-alpine has no /usr/local/etc/redis directory:
redis:
  volumes:
    - ./redis.conf:/usr/local/etc/redis/redis.conf
```

✅ **DO pass config as CLI flags:**
```yaml
# CORRECT:
redis:
  command: [redis-server, --requirepass, "${REDIS_PASSWORD}"]
```

❌ **DON'T use short/weak API keys:**
```bash
# WRONG - too short, no digits:
MCP_API_KEY=dev-key-short
# WRONG - hex-only fails the mixed-case requirement:
MCP_API_KEY=$(openssl rand -hex 32)
```

✅ **DO generate strong mixed-case keys:**
```bash
# CORRECT:
MCP_API_KEY=$(openssl rand -base64 48)
```

## Testing the Deployment

### Test MCP Protocol

```bash
# From the host (localhost):
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
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

# Expected: JSON-RPC response with server capabilities
```

### Test Guardrail Tools

```bash
# Test guardrail_validate_bash (replace YOUR_MCP_API_KEY with your key)
curl -X POST "http://localhost:8080/mcp" \
  -H "Authorization: Bearer YOUR_MCP_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/call",
    "params": {
      "name": "guardrail_validate_bash",
      "arguments": {
        "command": "rm -rf /",
        "session_token": "test-session"
      }
    }
  }'
```

### Test Web UI

```bash
# Test Web UI is responding
curl -s http://localhost:8081/ | head -10

# Test API endpoints
curl -s http://localhost:8081/api/rules | jq .
curl -s http://localhost:8081/api/documents | jq .
curl -s http://localhost:8081/api/stats | jq .
```

---

## Continued

Deployment troubleshooting is in [deployment-guide-troubleshooting.md](./deployment-guide-troubleshooting.md); verification, maintenance and hardening are in [deployment-guide-operations.md](./deployment-guide-operations.md).
