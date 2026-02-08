# MCP Server Changelog

All notable changes to the Guardrail MCP Server will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

---

## [Unreleased]

### Added

- Comprehensive API documentation (API.md)
- Troubleshooting guide in README.md
- Database migration instructions in README.md

### Changed

- Updated README.md with complete project structure
- Updated README.md with security features documentation
- Updated .env.example with better organization and documentation

---

## [1.9.2] - 2026-02-07

### Fixed

- **Web UI Authentication** - Removed API key requirement for Web UI routes
  - Web UI routes (`/`, `/index.html`, `/static/*`) are now publicly accessible
  - API endpoints still require valid API key
  - Health checks and metrics remain unauthenticated

---

## [1.9.1] - 2026-02-07

### Fixed

- **SSE Compatibility** - Fixed EOF errors with non-interactive clients
  - Added `WriteHeader(http.StatusOK)` for immediate header commit
  - Added `X-Accel-Buffering: no` for proxy compatibility
  - Added `Access-Control-Allow-Origin: *` for CORS
  - Send immediate ping event after endpoint to prevent client timeout
  - Better error handling on write/flush operations

- **PostgreSQL Array Scanning** - Fixed TEXT[] array scanning bug
  - Changed `AffectedFiles` from `pq.StringArray` to `pgtype.Array[string]`
  - Added `ToStringSlice()` and `ToTextArray()` helper functions
  - Compatible with pgx v5 driver

---

## [1.9.0] - 2026-02-07

### Added

- **MCP Protocol Implementation**
  - SSE transport for real-time client communication
  - JSON-RPC 2.0 message handling
  - Tools: `guardrail_init_session`, `guardrail_validate_bash`,
    `guardrail_validate_file_edit`, `guardrail_validate_git_operation`,
    `guardrail_pre_work_check`, `guardrail_get_context`
  - Resources: `guardrail://quick-reference`, `guardrail://rules/active`

- **Web UI API**
  - Document CRUD operations
  - Prevention rule management
  - Project configuration
  - Failure registry

- **IDE API**
  - File validation endpoint
  - Selection validation endpoint
  - Active rules endpoint
  - Quick reference endpoint

- **Infrastructure**
  - PostgreSQL 16 support with migrations
  - Redis 7 for caching and rate limiting
  - Circuit breaker pattern for resilience
  - Secrets scanning in document content
  - Audit logging infrastructure
  - Prometheus metrics

- **Security**
  - API key authentication (MCP and IDE keys)
  - JWT session tokens with configurable expiry
  - Rate limiting per API key
  - Content Security Policy headers
  - Non-root container execution
  - Read-only filesystem support

---

## Version Management

This MCP Server follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html):

- **MAJOR**: Incompatible API changes
- **MINOR**: Backwards-compatible functionality additions
- **PATCH**: Backwards-compatible bug fixes

---

*Last Updated: 2026-02-07*
