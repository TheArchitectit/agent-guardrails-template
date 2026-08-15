# Rules API

> See [API.md](API.md) for base URLs, authentication, and shared conventions.

## GET /api/rules

List prevention rules with pagination.

**Query Parameters**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| enabled | boolean | No | Filter by enabled status |
| category | string | No | Filter by category |
| limit | integer | No | Items per page (default: 20, max: 100) |
| offset | integer | No | Offset for pagination (default: 0) |

**Response**
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440001",
      "rule_id": "PREVENT-001",
      "name": "No Force Push",
      "pattern": "git push --force",
      "pattern_hash": "abc123...",
      "message": "Force push is not allowed",
      "severity": "error",
      "enabled": true,
      "category": "git",
      "created_at": "2026-01-14T10:00:00Z",
      "updated_at": "2026-01-14T10:00:00Z"
    }
  ],
  "pagination": {
    "total": 15,
    "limit": 20,
    "offset": 0
  }
}
```

## GET /api/rules/:id

Get a specific rule by ID (UUID).

**Path Parameters**
| Name | Type | Description |
|------|------|-------------|
| id | UUID | Rule ID |

**Response**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440001",
  "rule_id": "PREVENT-001",
  "name": "No Force Push",
  "pattern": "git push --force",
  "message": "Force push is not allowed",
  "severity": "error",
  "enabled": true,
  "category": "git"
}
```

## POST /api/rules

Create a new prevention rule.

**Request Body**
```json
{
  "rule_id": "PREVENT-002",
  "name": "No rm -rf /",
  "pattern": "rm -rf /",
  "message": "Dangerous command detected",
  "severity": "error",
  "category": "bash",
  "enabled": true
}
```

**Response (201)**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440002",
  "rule_id": "PREVENT-002",
  "name": "No rm -rf /",
  "pattern": "rm -rf /",
  "message": "Dangerous command detected",
  "severity": "error",
  "enabled": true,
  "category": "bash",
  "created_at": "2026-02-07T16:00:00Z"
}
```

## PUT /api/rules/:id

Update a rule.

**Path Parameters**
| Name | Type | Description |
|------|------|-------------|
| id | UUID | Rule ID |

**Request Body**
```json
{
  "name": "Updated Rule Name",
  "pattern": "updated pattern",
  "message": "Updated message",
  "severity": "warning",
  "enabled": true
}
```

**Response**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440001",
  "rule_id": "PREVENT-001",
  "name": "Updated Rule Name",
  "pattern": "updated pattern",
  "message": "Updated message",
  "severity": "warning",
  "enabled": true,
  "category": "git",
  "updated_at": "2026-02-07T16:00:00Z"
}
```

## DELETE /api/rules/:id

Delete a rule.

**Path Parameters**
| Name | Type | Description |
|------|------|-------------|
| id | UUID | Rule ID |

**Response (204)**
No content.

## PATCH /api/rules/:id

Partially update a rule (e.g., enable/disable).

**Path Parameters**
| Name | Type | Description |
|------|------|-------------|
| id | UUID | Rule ID |

**Request Body**
```json
{
  "enabled": false,
  "name": "Optional new name",
  "message": "Optional new message",
  "pattern": "Optional new pattern",
  "severity": "warning"
}
```

**Response**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440001",
  "rule_id": "PREVENT-001",
  "name": "No Force Push",
  "pattern": "git push --force",
  "message": "Force push is not allowed",
  "severity": "error",
  "enabled": false,
  "category": "git",
  "updated_at": "2026-02-07T16:00:00Z"
}
```
