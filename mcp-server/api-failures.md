# Failure Registry API

> See [API.md](api.md) for base URLs, authentication, and shared conventions.

## GET /api/failures

List failure registry entries with pagination.

**Query Parameters**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| status | string | No | Filter by status (active, resolved, deprecated) |
| category | string | No | Filter by category |
| project | string | No | Filter by project slug |
| limit | integer | No | Items per page (default: 20, max: 100) |
| offset | integer | No | Offset for pagination |

**Response**
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440004",
      "failure_id": "FAIL-001",
      "category": "deployment",
      "severity": "high",
      "error_message": "Production database overwritten",
      "root_cause": "Missing environment check",
      "affected_files": ["scripts/deploy.sh"],
      "status": "active",
      "project_slug": "my-project",
      "created_at": "2026-01-14T10:00:00Z"
    }
  ],
  "pagination": {
    "total": 42,
    "limit": 20,
    "offset": 0
  }
}
```

## GET /api/failures/:id

Get a specific failure entry.

**Path Parameters**
| Name | Type | Description |
|------|------|-------------|
| id | UUID | Failure ID |

**Response**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440004",
  "failure_id": "FAIL-001",
  "category": "deployment",
  "severity": "high",
  "error_message": "Production database overwritten",
  "root_cause": "Missing environment check",
  "affected_files": ["scripts/deploy.sh"],
  "status": "active",
  "project_slug": "my-project",
  "created_at": "2026-01-14T10:00:00Z"
}
```

## POST /api/failures

Create a new failure entry.

**Request Body**
```json
{
  "failure_id": "FAIL-002",
  "category": "security",
  "severity": "critical",
  "error_message": "Secret leaked in commit",
  "root_cause": "Pre-commit hook not installed",
  "affected_files": ["config/production.yml"],
  "regression_pattern": "password:\\s*['\"][^'\"]+['\"]",
  "status": "active",
  "project_slug": "my-project"
}
```

**Response (201)**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440005",
  "failure_id": "FAIL-002",
  "category": "security",
  "severity": "critical",
  "error_message": "Secret leaked in commit",
  "root_cause": "Pre-commit hook not installed",
  "affected_files": ["config/production.yml"],
  "regression_pattern": "password:\\s*['\"][^'\"]+['\"]",
  "status": "active",
  "project_slug": "my-project",
  "created_at": "2026-02-07T16:00:00Z"
}
```

## PUT /api/failures/:id

Update a failure entry (e.g., mark as resolved).

**Path Parameters**
| Name | Type | Description |
|------|------|-------------|
| id | UUID | Failure ID |

**Request Body**
```json
{
  "status": "resolved"
}
```

**Response**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440004",
  "failure_id": "FAIL-001",
  "category": "deployment",
  "severity": "high",
  "error_message": "Production database overwritten",
  "root_cause": "Missing environment check",
  "affected_files": ["scripts/deploy.sh"],
  "status": "resolved",
  "project_slug": "my-project",
  "created_at": "2026-01-14T10:00:00Z",
  "updated_at": "2026-02-07T16:00:00Z"
}
```
