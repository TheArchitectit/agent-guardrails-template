# Guardrail MCP Server Deployment Guide

**Version:** v3.3.0
**Last Updated:** 2026-08-15
**Tested On:** RHEL server with Podman, Docker Desktop (Windows 11), Ubuntu 24.04

## Overview

This guide provides step-by-step instructions for deploying the Guardrail MCP Server to production, including all fixes discovered while deploying to production.

## Prerequisites

- RHEL or compatible Linux distribution
- Podman or Docker installed
- Access to deployment server via SSH
- Go 1.25+ (for building from source)
- PostgreSQL 16 (or use containerized version)
- Redis 7 (or use containerized version)

## Deployment Summary

### What Was Fixed During Production Deployment

1. **Schema Validation Error** - Changed server name from "guardrail-mcp" to "guardrail_mcp" to fix MCP framework validation issues
2. **Postgres Permission Issues** - Removed security constraints and added `user: "70:70"` for postgres
3. **Configuration Variables** - Corrected ports, API keys, and JWT settings
4. **Container Networking** - Used pod networking to ensure containers can communicate

## Quick Deploy

### 1. Set the bind address in .env

By default the stack binds to `127.0.0.1` (localhost only). If you need the
server reachable from another machine, set `BIND_ADDR` to the interface you
want to expose — for example a Tailscale IP or a private-network address.

```bash
# Only do this if you need remote access; leave unset for localhost-only
sed -i 's/^BIND_ADDR=.*/BIND_ADDR=127.0.0.1/' .env
```

### 2. Build and Deploy

Generate real secrets first — never use checked-in or guessable values:

```bash
DB_PASSWORD=$(openssl rand -base64 24)
REDIS_PASSWORD=$(openssl rand -base64 24)
MCP_API_KEY=$(openssl rand -base64 48)   # mixed case + digits required
IDE_API_KEY=$(openssl rand -base64 48)
JWT_SECRET=$(openssl rand -hex 64)
```

Then build and run:

```bash
cd /opt/guardrail-mcp

# Build image
podman build \
  --build-arg VERSION=v3.3.0 \
  --build-arg BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  --build-arg GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown') \
  -f deploy/Dockerfile \
  -t guardrail-mcp:v3.3.0 .

# Create pod with port mappings (MCP and Web UI)
podman pod create --name guardrail-pod -p 8080:8080 -p 8081:8081

# Start postgres (user 70:70 matches the postgres user in the alpine image)
podman run -d --pod guardrail-pod --name guardrail-postgres \
  --user 70:70 \
  -e POSTGRES_USER=guardrail \
  -e POSTGRES_PASSWORD="$DB_PASSWORD" \
  -e POSTGRES_DB=guardrails \
  -v guardrail_pg_data:/var/lib/postgresql/data \
  docker.io/library/postgres:16-alpine

# Wait for postgres to be ready
sleep 5

# Start redis (config via CLI flags — no config file in the alpine image)
podman run -d --pod guardrail-pod --name guardrail-redis \
  docker.io/library/redis:7-alpine \
  redis-server --requirepass "$REDIS_PASSWORD" --maxmemory 256mb --maxmemory-policy allkeys-lru

# Start MCP server (pod networking: containers reach each other via localhost)
podman run -d --pod guardrail-pod --name guardrail-mcp-server \
  -e MCP_PORT=8080 \
  -e WEB_PORT=8081 \
  -e DB_HOST=localhost \
  -e DB_PORT=5432 \
  -e DB_NAME=guardrails \
  -e DB_USER=guardrail \
  -e DB_PASSWORD="$DB_PASSWORD" \
  -e DB_SSLMODE=disable \
  -e REDIS_HOST=localhost \
  -e REDIS_PORT=6379 \
  -e REDIS_PASSWORD="$REDIS_PASSWORD" \
  -e MCP_API_KEY="$MCP_API_KEY" \
  -e IDE_API_KEY="$IDE_API_KEY" \
  -e JWT_SECRET="$JWT_SECRET" \
  -e JWT_ISSUER=guardrail-mcp \
  -e JWT_EXPIRY=15m \
  -e JWT_ROTATION_HOURS=168h \
  localhost/guardrail-mcp:v3.3.0
```

> Prefer not to manage pods by hand? The compose path in
> [README.md](./README.md#deployment) handles all of the above in one command.

## Windows Docker Desktop Deployment

### Prerequisites

- Windows 10/11 with WSL2 enabled
- Docker Desktop installed with WSL2 backend
- Git for Windows or WSL2 Git

### Step 1: Clone and Configure

```powershell
# In PowerShell or Windows Terminal
cd C:\Users\YourName\Projects
git clone https://github.com/TheArchitectit/agent-guardrails-template.git
cd agent-guardrails-template/mcp-server

# Copy environment template
copy .env.example .env

# Generate secure keys (requires OpenSSL for Windows or use WSL2)
# Or manually generate 32+ character strings
```

### Step 2: Deploy with Docker Compose

```powershell
# Build and start all services
docker compose -f deploy/docker-compose.example.yml up -d --build

# Verify containers are running
docker compose -f deploy/docker-compose.example.yml ps

# View logs
docker compose -f deploy/docker-compose.example.yml logs -f
```

### Step 3: Access the Services

| Service | URL |
|---------|-----|
| Web UI | http://localhost:8081 |
| Health Check | http://localhost:8081/health/ready |
| MCP StreamableHTTP | http://localhost:8080/mcp |

### Windows-Specific Notes

- **Firewall:** Docker Desktop may prompt for firewall rules. Allow private network access.
- **Port conflicts:** If ports 8080/8081 are in use, edit `.env` to change `MCP_PORT` and `WEB_PORT`.
- **WSL2 file paths:** For best performance, keep project files inside WSL2 filesystem (`\\wsl$\Ubuntu\home\...`) rather than Windows mounts.
- **PowerShell execution policy:** If scripts fail, run `Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser`.

---

## Detailed Deployment Steps

### Step 1: Environment Setup

```bash
# Create deployment directory
mkdir -p /opt/guardrail-mcp
cd /opt/guardrail-mcp

# Copy code from repository (if not already there)
scp -r /path/to/agent-guardrails-template/mcp-server/* deploy@your-server:/opt/guardrail-mcp/

# Create .env file — generate real secrets, never use checked-in values
DB_PASSWORD=$(openssl rand -base64 24)
REDIS_PASSWORD=$(openssl rand -base64 24)
MCP_API_KEY=$(openssl rand -base64 48)
IDE_API_KEY=$(openssl rand -base64 48)
JWT_SECRET=$(openssl rand -hex 64)

cat > .env << EOF
# =============================================================================
# Server Configuration
# =============================================================================
MCP_PORT=8080
WEB_PORT=8081
WEB_ENABLED=true
LOG_LEVEL=info
REQUEST_TIMEOUT=30s
SHUTDOWN_TIMEOUT=30s
# Bind address — leave unset (or 127.0.0.1) for localhost-only.
# Set to a specific interface IP if you need the server reachable off-box.
# BIND_ADDR=127.0.0.1

# =============================================================================
# Database Configuration
# =============================================================================
DB_HOST=postgres
DB_PORT=5432
DB_NAME=guardrails
DB_USER=guardrail
DB_PASSWORD=${DB_PASSWORD}
DB_SSLMODE=disable

# =============================================================================
# Redis Configuration
# =============================================================================
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=${REDIS_PASSWORD}
REDIS_USE_TLS=false

# =============================================================================
# Security Configuration
# =============================================================================
# IMPORTANT: These keys MUST be at least 32 characters long
# and contain a mix of uppercase, lowercase, and digits.
MCP_API_KEY=${MCP_API_KEY}
IDE_API_KEY=${IDE_API_KEY}

# JWT Configuration — must be at least 32 bytes of real entropy
JWT_SECRET=${JWT_SECRET}
JWT_ISSUER=guardrail-mcp
JWT_EXPIRY=15m
JWT_ROTATION_HOURS=168h
EOF

chmod 600 .env
```

### Step 2: Apply Schema Fix

**Critical Fix:** The MCP framework has historically had issues with dashes/hyphens in server names — this causes schema validation errors on tool registration. The server name should use underscores.

```bash
# Check current server name
grep 'NewMCPServer' internal/mcp/server.go

# Should show: server.NewMCPServer("guardrail_mcp", ...)
# NOT: server.NewMCPServer("guardrail-mcp", ...)

# If it shows "guardrail-mcp", change it:
sed -i 's/server.NewMCPServer("guardrail-mcp"/server.NewMCPServer("guardrail_mcp"/' internal/mcp/server.go
```

### Step 3: Build Docker Image

```bash
cd /opt/guardrail-mcp

# Set build variables
VERSION=v3.3.0
BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')

# Build the image
podman build \
  --build-arg VERSION=$VERSION \
  --build-arg BUILD_TIME=$BUILD_TIME \
  --build-arg GIT_COMMIT=$GIT_COMMIT \
  -f deploy/Dockerfile \
  -t guardrail-mcp:$VERSION .

# Verify image was created
podman images | grep guardrail-mcp
```

### Step 4: Create Pod and Start Containers

Using pod networking ensures containers can communicate via localhost. Export
the secrets you generated in the Quick Deploy section first
(`DB_PASSWORD`, `REDIS_PASSWORD`, `MCP_API_KEY`, `IDE_API_KEY`, `JWT_SECRET`).

```bash
# Remove existing containers (if any)
podman stop guardrail-postgres guardrail-redis guardrail-mcp-server 2>/dev/null
podman rm guardrail-postgres guardrail-redis guardrail-mcp-server 2>/dev/null
podman pod rm guardrail-pod 2>/dev/null

# Create new pod with port mappings
podman pod create --name guardrail-pod -p 8080:8080 -p 8081:8081

# Start postgres (critical: use user 70:70 to avoid permission issues)
podman run -d --pod guardrail-pod --name guardrail-postgres \
  --user 70:70 \
  -e POSTGRES_USER=guardrail \
  -e POSTGRES_PASSWORD="$DB_PASSWORD" \
  -e POSTGRES_DB=guardrails \
  -v guardrail_pg_data:/var/lib/postgresql/data \
  docker.io/library/postgres:16-alpine

# Wait for postgres to initialize (important!)
echo "Waiting for postgres to be ready..."
sleep 5

# Verify postgres is running
podman ps | grep guardrail-postgres

# Start redis
podman run -d --pod guardrail-pod --name guardrail-redis \
  docker.io/library/redis:7-alpine \
  redis-server --requirepass "$REDIS_PASSWORD" --maxmemory 256mb --maxmemory-policy allkeys-lru

# Wait for redis to start
sleep 3

# Start MCP server
podman run -d --pod guardrail-pod --name guardrail-mcp-server \
  -e MCP_PORT=8080 \
  -e WEB_PORT=8081 \
  -e WEB_ENABLED=true \
  -e LOG_LEVEL=info \
  -e DB_HOST=localhost \
  -e DB_PORT=5432 \
  -e DB_NAME=guardrails \
  -e DB_USER=guardrail \
  -e DB_PASSWORD="$DB_PASSWORD" \
  -e DB_SSLMODE=disable \
  -e REDIS_HOST=localhost \
  -e REDIS_PORT=6379 \
  -e REDIS_PASSWORD="$REDIS_PASSWORD" \
  -e MCP_API_KEY="$MCP_API_KEY" \
  -e IDE_API_KEY="$IDE_API_KEY" \
  -e JWT_SECRET="$JWT_SECRET" \
  -e JWT_ISSUER=guardrail-mcp \
  -e JWT_EXPIRY=15m \
  -e JWT_ROTATION_HOURS=168h \
  localhost/guardrail-mcp:v3.3.0
```

### Step 5: Verify Deployment

```bash
# Check all containers are running
podman ps -a | grep guardrail

# Expected output showing all three containers UP
# guardrail-postgres, guardrail-redis, guardrail-mcp-server

# Check MCP server logs
podman logs guardrail-mcp-server 2>&1 | tail -20

# Should show:
# - "Database connected"
# - "Redis connected"
# - "Starting web server" on :8081
# - "Starting MCP server" on :8080

# Test the MCP endpoint — initialize over stateless StreamableHTTP
curl -s -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}'

# Should return a JSON-RPC response with server capabilities

# Test Web UI health
curl -s http://localhost:8081/health/ready
```

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
    # redis:7-alpine has no config dir — pass settings as CLI flags.
    # Run as the image's redis user so no setuid drop is needed.
    command:
      - redis-server
      - --requirepass
      - ${REDIS_PASSWORD}
      - --maxmemory
      - 256mb
      - --maxmemory-policy
      - allkeys-lru
    user: "999:1000"
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "-a", "${REDIS_PASSWORD}", "ping"]
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

## Troubleshooting Guide

### Problem: Schema Validation Error

**Error:**
```
Invalid schema for function 'guardrails_guardrail_pre_work_check':
In context=('properties', 'affected_files'), array schema missing items
```

**Cause:** Server name contains dashes/hyphens

**Solution:**
```bash
# Check server name
grep 'NewMCPServer' internal/mcp/server.go

# Should show: server.NewMCPServer("guardrail_mcp", "v3.3.0")
# If not, fix it:
sed -i 's/server.NewMCPServer("guardrail-mcp"/server.NewMCPServer("guardrail_mcp"/' internal/mcp/server.go

# Rebuild and redeploy
podman build -f deploy/Dockerfile -t guardrail-mcp:fixed .
podman stop guardrail-mcp-server
podman run -d --pod guardrail-pod --name guardrail-mcp-server [environment variables] localhost/guardrail-mcp:fixed
```

### Problem: Postgres Permission Errors

**Error:**
```
chmod: /var/run/postgresql: Operation not permitted
find: /var/lib/postgresql/data/pgdata: Permission denied
```

**Cause:** Missing user specification for postgres container

**Solution:**
```bash
# Add user to postgres service in compose file:
# In deploy/podman-compose.yml:
services:
  postgres:
    image: postgres:16-alpine
    user: "70:70"  # ADD THIS LINE
    # ... rest of config

# Or when running directly:
podman run -d --user 70:70 postgres:16-alpine [other options]
```

### Problem: Database Authentication Failed

**Error:**
```
failed to connect to database: password authentication failed for user "guardrail"
```

**Cause:** Wrong database credentials or postgres not ready

**Solution:**
```bash
# Check postgres is running
podman ps | grep guardrail-postgres

# Check postgres logs
podman logs guardrail-postgres

# Verify credentials in .env match postgres environment
# .env should have:
DB_USER=guardrail
DB_PASSWORD=<same value used when the postgres container was created>
# And postgres should have:
POSTGRES_USER=guardrail
POSTGRES_PASSWORD=<same value>

# Test connection from within container
podman exec guardrail-postgres psql -U guardrail -d guardrail -c 'SELECT 1'
```

### Problem: Redis Connection Refused

**Error:**
```
failed to connect to Redis: dial tcp [::1]:6379: connect: connection refused
```

**Cause:** Redis not running or not accessible

**Solution:**
```bash
# Check redis is running
podman ps | grep guardrail-redis

# Check redis logs
podman logs guardrail-redis

# Verify redis is in same pod (for localhost access)
podman pod inspect guardrail-pod | grep -A20 containers

# Test redis connection
podman exec guardrail-redis redis-cli -a "$REDIS_PASSWORD" ping
# Should return: PONG
```

### Problem: Container Exits Immediately

**Error:** Container shows "Exited (1)" status immediately after start

**Cause:** Missing or incorrect environment variables

**Solution:**
```bash
# Check container logs
podman logs guardrail-mcp-server

# Common issues:
# - Missing MCP_API_KEY or IDE_API_KEY
# - API keys too short (< 32 characters)
# - JWT_SECRET too short (< 32 bytes)
# - JWT_ROTATION_HOURS missing 'h' unit
# - Database credentials incorrect
# - Redis password incorrect

# Verify all required environment variables are set
podman inspect guardrail-mcp-server | grep -A100 '"Env"'
```

### Problem: Ports Already in Use

**Error:**
```
bind: address already in use
```

**Cause:** Ports 8080 or 8081 are already in use

**Solution:**
```bash
# Check what's using the ports
ss -tln | grep -E '8080|8081'

# Change ports in .env if needed:
MCP_PORT=8097
WEB_PORT=8098

# Update pod creation:
podman pod create --name guardrail-pod -p 8097:8097 -p 8098:8098

# Update environment in container:
-e MCP_PORT=8097 -e WEB_PORT=8098
```

### Problem: Connection Timeout from Remote Machine

**Error:** Cannot connect to MCP server from another machine

**Cause:** Firewall or network configuration

**Solution:**
```bash
# Check firewall status
firewall-cmd --state

# Open ports (if needed)
sudo firewall-cmd --permanent --add-port=8080/tcp
sudo firewall-cmd --permanent --add-port=8081/tcp
sudo firewall-cmd --reload

# Verify ports are open
sudo firewall-cmd --list-ports

# Test from remote machine
curl -s http://your-server:8080/mcp
```

### Problem: YAML Syntax Errors in Compose File

**Error:**
```
ERROR: yaml.scanner.ScannerError: mapping values are not allowed here
```

**Cause:** Incorrect indentation or orphaned lines

**Solution:**
```bash
# Validate YAML syntax
python3 -c "import yaml; yaml.safe_load(open('deploy/podman-compose.yml'))"

# Common issues:
# - Orphaned container_name line at file start
# - Incorrect indentation (use spaces, not tabs)
# - Duplicate keys
```

## Verification Checklist

After deployment, verify:

- [ ] All three containers running: `podman ps | grep guardrail`
- [ ] Postgres healthy: `podman logs guardrail-postgres | tail -5`
- [ ] Redis healthy: `podman logs guardrail-redis | tail -5`
- [ ] MCP server started: `podman logs guardrail-mcp-server | grep "Starting Guardrail MCP Server"`
- [ ] Database connected: `podman logs guardrail-mcp-server | grep "Database connected"`
- [ ] Redis connected: `podman logs guardrail-mcp-server | grep "Redis connected"`
- [ ] MCP endpoint responding: `curl -s -X POST http://localhost:8080/mcp`
- [ ] Web UI responding: `curl -s http://localhost:8081/health/ready`
- [ ] API key authentication working: `curl -H 'Authorization: Bearer YOUR_KEY' http://localhost:8080/version`
- [ ] Ports accessible from network: `curl http://your-server:8080/mcp`

## Maintenance

### Viewing Logs

```bash
# View all logs
podman logs guardrail-mcp-server

# Follow logs in real-time
podman logs -f guardrail-mcp-server

# View specific number of lines
podman logs --tail 50 guardrail-mcp-server

# View logs for all containers
podman logs guardrail-postgres
podman logs guardrail-redis
```

### Restarting Services

```bash
# Restart MCP server only
podman restart guardrail-mcp-server

# Restart all containers
podman restart guardrail-postgres guardrail-redis guardrail-mcp-server

# Recreate entire pod
podman pod stop guardrail-pod
podman pod rm guardrail-pod
# Then run deployment steps again
```

### Updating Configuration

```bash
# Update .env file
vim .env

# Restart containers to pick up changes
podman restart guardrail-mcp-server

# For database changes, also restart postgres
podman restart guardrail-postgres
```

### Backup and Restore

```bash
# Backup postgres data
podman run --rm \
  -v guardrail_pg_data:/data \
  -v $(pwd):/backup \
  alpine \
  tar czf /backup/postgres-backup.tar.gz /data

# Restore postgres data
podman run --rm \
  -v guardrail_pg_data:/data \
  -v $(pwd):/backup \
  alpine \
  tar xzf /backup/postgres-backup.tar.gz -C /
```

## Production Hardening

### Security Recommendations

1. **Change all default secrets** before production use
2. **Use strong random passwords**:
   ```bash
   openssl rand -hex 32  # For API keys
   openssl rand -base64 32  # For passwords
   ```
3. **Enable TLS** for database connections
4. **Set specific CORS origins** instead of "*"
5. **Enable production mode** for stricter security checks
6. **Use network policies** to restrict container communication
7. **Enable audit logging** and ship logs to central system
8. **Set resource limits** to prevent resource exhaustion

### Performance Tuning

```bash
# Database connection pooling
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=30m

# Redis connection pooling
REDIS_POOL_SIZE=10
REDIS_MIN_IDLE_CONNS=2

# Rate limiting (adjust based on load)
RATE_LIMIT_MCP=1000
RATE_LIMIT_IDE=500

# Cache TTL (adjust based on data volatility)
CACHE_TTL_RULES=5m
CACHE_TTL_DOCS=10m
```

## OpenCode Configuration

### MCP Server Configuration

Add to `.opencode/oh-my-opencode.jsonc`:

```jsonc
{
  "mcpServers": {
    "guardrails": {
      "type": "remote",
      "url": "http://your-server:8080/mcp",
      "headers": {
        "Authorization": "Bearer YOUR_MCP_API_KEY"
      }
    }
  }
}
```

### Environment Variables

Create `.env.opencode`:

```bash
# MCP Server Connection
export MCP_SERVER_URL=http://your-server:8080
export MCP_API_KEY=YOUR_MCP_API_KEY
export IDE_API_KEY=YOUR_IDE_API_KEY

# Local Configuration (for OpenCode)
export GUARDRAILS_PROJECT_SLUG=your-project
export GUARDRAILS_AGENT_TYPE=opencode
```

## References

- [MCP Server README](./README.md)
- [API Documentation](./API.md)
- [Security Review](./OBSERVABILITY_REVIEW.md)
- [Dockerfile](./deploy/Dockerfile)
- [Podman Compose](./deploy/podman-compose.yml)
- [Kubernetes Deployment](./deploy/k8s-deployment.yaml)

## Support

For issues or questions:
1. Check troubleshooting section above
2. Review container logs: `podman logs guardrail-mcp-server`
3. Verify configuration against this guide
4. Check [GitHub Issues](https://github.com/TheArchitectit/agent-guardrails-template/issues)

## Changelog

### 2026-02-13 - Initial Deployment Guide
- Documented production deployment process
- Added schema validation error fix
- Added postgres permission fix
- Added configuration requirements
- Added troubleshooting guide
- Added verification checklist