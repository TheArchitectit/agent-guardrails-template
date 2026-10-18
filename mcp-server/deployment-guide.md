# Guardrail MCP Server Deployment Guide

**Version:** v3.4.0
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
  --build-arg VERSION=v3.4.0 \
  --build-arg BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  --build-arg GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown') \
  -f deploy/Dockerfile \
  -t guardrail-mcp:v3.4.0 .

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
  localhost/guardrail-mcp:v3.4.0
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
VERSION=v3.4.0
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
  localhost/guardrail-mcp:v3.4.0
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

---

## Continued

Configuration, Compose, deployment testing and troubleshooting are in [deployment-guide-configuration.md](./deployment-guide-configuration.md).
