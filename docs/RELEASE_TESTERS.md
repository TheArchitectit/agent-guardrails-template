# MCP Server v1.9.5 - Testers Release

> **Release for Testers:** This release is ready for testing by external testers.

**Release Date:** 2026-02-08
**Branch:** `mcpserver`
**Tag:** `v1.9.5-testers`
**Status:** Production Ready - Testing Phase

---

## Quick Start for Testers

### Prerequisites

- Docker or Podman
- Go 1.23+ (for building from source)
- curl (for testing endpoints)

### Build and Run

```bash
# Clone the repository
git clone https://github.com/TheArchitectit/agent-guardrails-template.git
cd agent-guardrails-template

# Checkout the MCP branch
git checkout mcpserver

# Build the server
cd mcp-server
make build

# Run with Docker Compose
make docker-up
```

### Environment Setup

Copy and configure the environment:

```bash
cp .env.example .env
# Edit .env and set secure values for:
# - MCP_API_KEY
# - IDE_API_KEY
# - JWT_SECRET
# - DB_PASSWORD
# - REDIS_PASSWORD
```

---

## What to Test

### 1. MCP Protocol Testing

**SSE Endpoint Connection:**
```bash
curl -N http://localhost:8080/mcp/v1/sse
```

**Initialize Session:**
```bash
curl -X POST http://localhost:8080/mcp/v1/message \
  -H "Content-Type: application/json" \
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

**Expected:** JSON-RPC 2.0 response with server capabilities

### 2. Guardrail Validation Testing

**Test Bash Command Validation:**
```bash
curl -X POST http://localhost:8080/mcp/v1/message \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $MCP_API_KEY" \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/call",
    "params": {
      "name": "guardrail_validate_bash",
      "arguments": {
        "command": "rm -rf /",
        "context": "test-session"
      }
    }
  }'
```

**Expected:** Validation failure with reason (dangerous command detected)

### 3. Web UI Testing

Access the Web UI at: `http://localhost:8081`

**Test Items:**
- Document browsing and search
- Prevention rule management (CRUD)
- Project configuration
- Failure registry viewing
- Statistics dashboard

### 4. Health Check Testing

```bash
# Liveness probe
curl http://localhost:8081/health/live

# Readiness probe (checks DB and Redis)
curl http://localhost:8081/health/ready

# Metrics endpoint
curl http://localhost:8081/metrics
```

### 5. API Security Testing

**Test Authentication:**
```bash
# Without API key (should fail)
curl http://localhost:8080/mcp/v1/sse
# Expected: 401 Unauthorized

# With valid API key
curl -H "Authorization: Bearer $MCP_API_KEY" \
  http://localhost:8080/mcp/v1/sse
# Expected: 200 OK with SSE stream
```

**Test Rate Limiting:**
```bash
# Send 1000+ requests rapidly
for i in {1..1100}; do
  curl -s -o /dev/null -w "%{http_code}\n" \
    -H "Authorization: Bearer $MCP_API_KEY" \
    http://localhost:8081/api/documents
done
# Expected: First 1000 succeed, then 429 Too Many Requests
```

### 6. Database Operations Testing

**Test Document CRUD:**
```bash
# List documents
curl http://localhost:8081/api/documents

# Search documents
curl "http://localhost:8081/api/documents/search?q=guardrail"

# Get specific document
curl http://localhost:8081/api/documents/1
```

### 7. Resilience Testing

**Circuit Breaker Test:**
```bash
# Stop PostgreSQL container
podman stop guardrail-postgres

# Requests should fail fast with circuit breaker open
# Wait 30 seconds for recovery attempt

# Start PostgreSQL
podman start guardrail-postgres

# Service should recover automatically
```

### 8. Container Security Testing

```bash
# Verify non-root user
podman exec guardrail-mcp-server id
# Expected: uid=65532 gid=65532

# Verify read-only filesystem
docker exec guardrail-mcp-server touch /test 2>&1
# Expected: Read-only file system error

# Check dropped capabilities
podman inspect guardrail-mcp-server | grep -A 10 CapDrop
# Expected: ALL
```

---

## Bug Report Template

If you find issues, please report with:

```markdown
**Test Case:** [Which test above]
**Severity:** [Critical/High/Medium/Low]
**Expected:** [What should happen]
**Actual:** [What actually happened]
**Steps to Reproduce:**
1. Step one
2. Step two
3. ...

**Logs:**
```
[Relevant log output]
```

**Environment:**
- OS: [e.g., Ubuntu 22.04]
- Docker/Podman version: [e.g., Podman 4.9]
- Go version: [e.g., 1.23.2]
```

---

## Known Issues

None at this time.

---

## Feedback

Please submit feedback via:
- GitHub Issues: https://github.com/TheArchitectit/agent-guardrails-template/issues
- Email: [maintainer contact]

---

## Security Notes

- All API keys should be generated with: `openssl rand -hex 32`
- JWT secret should be at least 48 characters
- Database passwords should be strong and unique
- Never commit .env files with real credentials

---

**Co-Authored-By:** Claude Opus 4.5 <noreply@anthropic.com>
