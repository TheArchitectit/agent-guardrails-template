# Guardrail MCP Server Deployment — Troubleshooting

> Deployment troubleshooting guide from the Guardrail MCP Server deployment documentation.

**Part of:** [Guardrail MCP Server Deployment Guide](./deployment-guide.md)

---

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

# Should show: server.NewMCPServer("guardrail_mcp", "v3.4.0")
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

# Test redis connection (Spec 17 R17-08: no credential in argv — the
# password is read from REDISCLI_AUTH in the environment, never from -a)
podman exec -e REDISCLI_AUTH="$REDIS_PASSWORD" guardrail-redis redis-cli ping
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

---

## Continued

Verification, maintenance and production hardening are in [deployment-guide-operations.md](./deployment-guide-operations.md).
