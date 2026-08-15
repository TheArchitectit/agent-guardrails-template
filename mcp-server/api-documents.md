# Documents API

> See [API.md](API.md) for base URLs, authentication, and shared conventions.

## GET /api/documents

List all documents with pagination.

**Query Parameters**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| category | string | No | Filter by category (workflow, standard, guide, reference) |
| limit | integer | No | Items per page (default: 20, max: 100) |
| offset | integer | No | Offset for pagination (default: 0) |

**Response**
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "slug": "agent-guardrails",
      "title": "Agent Guardrails",
      "content": "# Agent Guardrails...",
      "category": "standard",
      "path": "docs/AGENT_GUARDRAILS.md",
      "version": 1,
      "metadata": {},
      "created_at": "2026-01-14T10:00:00Z",
      "updated_at": "2026-02-07T15:30:00Z"
    }
  ],
  "pagination": {
    "total": 25,
    "limit": 20,
    "offset": 0
  }
}
```

## GET /api/documents/:id

Get a specific document by ID (UUID).

**Path Parameters**
| Name | Type | Description |
|------|------|-------------|
| id | UUID | Document ID |

**Response**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "slug": "agent-guardrails",
  "title": "Agent Guardrails",
  "content": "# Agent Guardrails...",
  "category": "standard",
  "path": "docs/AGENT_GUARDRAILS.md",
  "version": 1,
  "metadata": {},
  "created_at": "2026-01-14T10:00:00Z",
  "updated_at": "2026-02-07T15:30:00Z"
}
```

## PUT /api/documents/:id

Update a document.

**Request Body**
```json
{
  "title": "Updated Title",
  "content": "# Updated Content",
  "category": "standard",
  "metadata": {
    "author": "user@example.com"
  }
}
```

**Response**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "slug": "agent-guardrails",
  "title": "Updated Title",
  "content": "# Updated Content",
  "category": "standard",
  "version": 2,
  "updated_at": "2026-02-07T16:00:00Z"
}
```

**Error Response (Secrets Detected)**
```json
{
  "error": "Potential secrets detected in content",
  "findings": [
    {
      "pattern": "AWS Access Key ID",
      "line": 15,
      "column": 23,
      "match": "AKIA****XXXX",
      "description": "AWS IAM access key"
    }
  ]
}
```

## GET /api/documents/search

Full-text search documents.

**Query Parameters**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| q | string | Yes | Search query (max 200 chars) |
| limit | integer | No | Max results (default: 20, max: 50) |

**Response**
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "slug": "agent-guardrails",
      "title": "Agent Guardrails",
      "content": "# Agent Guardrails...",
      "category": "standard",
      "path": "docs/AGENT_GUARDRAILS.md",
      "version": 1,
      "metadata": {},
      "created_at": "2026-01-14T10:00:00Z",
      "updated_at": "2026-02-07T15:30:00Z"
    }
  ],
  "query": "guardrail safety",
  "pagination": {
    "limit": 20
  }
}
```
