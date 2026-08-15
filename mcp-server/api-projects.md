# Projects API

> See [API.md](api.md) for base URLs, authentication, and shared conventions.

## GET /api/projects

List all projects with pagination.

**Query Parameters**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| limit | integer | No | Items per page (default: 20, max: 100) |
| offset | integer | No | Offset for pagination (default: 0) |

**Response**
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440003",
      "name": "My Project",
      "slug": "my-project",
      "guardrail_context": "# Project Context...",
      "active_rules": ["PREVENT-001", "PREVENT-002"],
      "metadata": {},
      "created_at": "2026-01-14T10:00:00Z",
      "updated_at": "2026-01-14T10:00:00Z"
    }
  ],
  "pagination": {
    "total": 8,
    "limit": 20,
    "offset": 0
  }
}
```

## GET /api/projects/:id

Get a project by ID (UUID).

**Path Parameters**
| Name | Type | Description |
|------|------|-------------|
| id | UUID | Project ID |

**Response**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440003",
  "name": "My Project",
  "slug": "my-project",
  "guardrail_context": "# Project Context...",
  "active_rules": ["PREVENT-001", "PREVENT-002"],
  "metadata": {
    "repository": "https://github.com/org/repo"
  }
}
```

## POST /api/projects

Create a new project.

**Request Body**
```json
{
  "name": "New Project",
  "slug": "new-project",
  "guardrail_context": "# Context",
  "active_rules": ["PREVENT-001"],
  "metadata": {}
}
```

**Response (201)**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440004",
  "name": "New Project",
  "slug": "new-project",
  "guardrail_context": "# Context",
  "active_rules": ["PREVENT-001"],
  "metadata": {},
  "created_at": "2026-02-07T16:00:00Z",
  "updated_at": "2026-02-07T16:00:00Z"
}
```

## PUT /api/projects/:id

Update a project.

**Path Parameters**
| Name | Type | Description |
|------|------|-------------|
| id | UUID | Project ID |

**Request Body**
```json
{
  "name": "Updated Project Name",
  "guardrail_context": "# Updated Context",
  "active_rules": ["PREVENT-001", "PREVENT-003"]
}
```

**Response**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440003",
  "name": "Updated Project Name",
  "slug": "my-project",
  "guardrail_context": "# Updated Context",
  "active_rules": ["PREVENT-001", "PREVENT-003"],
  "metadata": {},
  "updated_at": "2026-02-07T16:00:00Z"
}
```

## DELETE /api/projects/:id

Delete a project.

**Path Parameters**
| Name | Type | Description |
|------|------|-------------|
| id | UUID | Project ID |

**Response (204)**
No content.
