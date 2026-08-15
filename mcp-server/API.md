# Guardrail MCP Server - API Documentation

Complete API reference for the Guardrail MCP Server REST endpoints. Endpoint details are split into focused sub-documents by group.

---

## Base URLs

| Service | URL | Port |
|---------|-----|------|
| MCP Protocol | `http://localhost:8080` | 8080 |
| Web UI API | `http://localhost:8081` | 8081 |

---

## Authentication

All API endpoints (except health checks and Web UI) require authentication via API key.

### Header Format

```
Authorization: Bearer <api_key>
```

### API Key Types

| Key Type | Environment Variable | Purpose |
|----------|---------------------|---------|
| MCP | `MCP_API_KEY` | MCP protocol and general API access |
| IDE | `IDE_API_KEY` | IDE-specific endpoints |

### Authentication Errors

**401 Unauthorized**
```json
{
  "error": "Missing authorization header"
}
```

**401 Unauthorized**
```json
{
  "error": "Invalid API key"
}
```

---

## Endpoint Groups

| Group | Endpoints | Reference |
|-------|-----------|-----------|
| Health | Liveness, readiness, metrics, version | [api-health.md](api-health.md) |
| Documents | List, get, update, search | [api-documents.md](api-documents.md) |
| Rules | List, get, create, update, delete, patch | [api-rules.md](api-rules.md) |
| Projects | List, get, create, update, delete | [api-projects.md](api-projects.md) |
| Failure Registry | List, get, create, update | [api-failures.md](api-failures.md) |
| IDE | Health, validate file/selection, rules, quick reference | [api-ide.md](api-ide.md) |
| System | Stats, ingest | [api-system.md](api-system.md) |

---

## Error Responses

### Standard Error Format

All error responses use the following format:

```json
{
  "error": "Human-readable error message"
}
```

### HTTP Status Codes

| Status | Meaning |
|--------|---------|
| 200 | Success |
| 201 | Created |
| 204 | No Content |
| 400 | Bad Request |
| 401 | Unauthorized |
| 403 | Forbidden |
| 404 | Not Found |
| 429 | Rate Limit Exceeded |
| 500 | Internal Server Error |
| 503 | Service Unavailable |

### Rate Limit Response (429)

```json
{
  "error": "Rate limit exceeded"
}
```

---

## Rate Limits

| Endpoint Type | Limit | Window |
|--------------|-------|--------|
| MCP | 1000 | per minute |
| IDE | 500 | per minute |
| Session | 100 | per minute |

---

## Data Models

### Severity Levels

| Level | Description | Action |
|-------|-------------|--------|
| error | Critical violation | halt operation |
| warning | Potential issue | confirm before proceeding |
| info | Informational | log only |

### Failure Status

| Status | Description |
|--------|-------------|
| active | Currently relevant |
| resolved | Fixed and verified |
| deprecated | No longer applicable |

### Document Categories

| Category | Description |
|----------|-------------|
| workflow | Process documentation |
| standard | Coding standards |
| guide | How-to guides |
| reference | Quick reference |

---

## Pagination Standards

All list endpoints use consistent pagination:

### Request Parameters

| Parameter | Type | Default | Max | Description |
|-----------|------|---------|-----|-------------|
| limit | integer | 20 | 100 | Items per page |
| offset | integer | 0 | - | Number of items to skip |

### Response Format

```json
{
  "data": [...],
  "pagination": {
    "total": 100,
    "limit": 20,
    "offset": 0
  }
}
```

### Calculating Next Page

```
next_offset = current_offset + limit
has_more = (offset + limit) < total
```
