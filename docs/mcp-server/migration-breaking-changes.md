# Migration: Breaking Changes by Version

> What changed in each release and what you need to do about it

See the [Migration Overview](version-migration.md) for the compatibility matrix and upgrade procedures.

---

## v2.6.0 (Current) - Go Migration

**Release Date:** 2026-02-15

### Breaking Changes

1. **Language Migration: Python to Go**
   - **Old:** `scripts/team_manager.py` (Python)
   - **New:** `mcp-server/internal/team/` (Go package)
   - **Impact:** No runtime Python required
   - **API:** Unchanged from MCP client perspective

2. **Build Process**
   - Old: `pip install -r requirements.txt`
   - New: `go build ./cmd/server`
   - Binary: Single static binary vs Python interpreter

3. **Container**
   - Old: Python-based image (~500MB)
   - New: Distroless Go image (~50MB)
   - Security: Non-root, read-only filesystem, dropped capabilities

### Migration Benefits

| Metric | Python | Go | Improvement |
|--------|--------|-----|-------------|
| Container Size | ~500MB | ~50MB | **10x smaller** |
| Startup Time | ~3s | ~100ms | **30x faster** |
| Memory Usage | ~200MB | ~20MB | **10x less** |
| Security | Full OS | Distroless | **Hardened** |

### New Features

- Team size validation (TEAM-007 compliance)
- Phase gate automation
- Agent team mapping
- Extended MCP tools (5 new tools)
- Hot-reloadable configuration
- Circuit breaker patterns

---

## v2.0.0

**Release Date:** 2026-02-15

### Breaking Changes

1. **Team Configuration Schema v2**
   - New required field: `team_version`
   - Changed `members` from array to object structure
   - Added `metadata` field for custom properties

2. **API Endpoint Changes**
   - `/mcp/v1/message` - Now requires `session_id` parameter
   - `/mcp/v1/sse` - Changed event format

3. **Environment Variables**
   - `MCP_PORT` renamed to `MCP_SERVER_PORT`
   - `WEB_PORT` renamed to `WEB_UI_PORT`
   - New required: `TEAM_CONFIG_VERSION`

### New Features

- Team size validation (TEAM-007 compliance)
- Phase gate automation
- Agent team mapping
- Extended MCP tools (5 new tools)

---

## v1.10.0

**Release Date:** 2026-02-08

### Breaking Changes

None. This is a backward-compatible release.

### New Features

- 5 new MCP tools (`guardrail_validate_scope`, etc.)
- 6 new MCP resources
- Web UI Management Interface
- Documentation search functionality

### Migration Notes

- All changes are additive
- No configuration changes required
- New tools available immediately after upgrade

---

## v1.9.0

**Release Date:** 2026-02-07

### Breaking Changes

1. **MCP Protocol Migration**
   - Moved from custom protocol to standard MCP
   - Port changed: 8094 (SSE), 8095 (message)
   - Authentication now requires `Authorization` header

2. **Configuration Structure**
   - `.teams/` directory location changed
   - New required files: `.guardrails/rules.json`

3. **Tool Names**
   - `validate_bash` renamed to `guardrail_validate_bash`
   - `validate_git` renamed to `guardrail_validate_git_operation`
   - `validate_file` renamed to `guardrail_validate_file_edit`

### New Features

- Full MCP server implementation
- SSE transport support
- PostgreSQL and Redis backends
- Production deployment support

---

## v1.8.0 to v1.9.0

**Critical:** This is a major protocol change. Plan for downtime.

### Breaking Changes

1. **Port Configuration**
   - Old: Port 8094 (custom protocol)
   - New: Port 8092 (MCP SSE), 8093 (Web UI)

2. **Client Configuration**
   - Old: Direct HTTP calls
   - New: MCP protocol with SSE
