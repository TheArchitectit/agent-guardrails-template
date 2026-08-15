# Error Handling Reference

> Error response format and error code reference for team management tools

Team tools use standard HTTP status codes and structured error responses. All errors follow a consistent format with error code, message, and troubleshooting guidance.

---

## Error Response Format

```json
{
  "IsError": true,
  "Content": [{
    "Type": "text",
    "Text": "TEAM-001: Team not found"
  }],
  "error_code": "TEAM-001",
  "error_message": "Team with ID 99 does not exist",
  "documentation_url": "https://docs.example.com/errors/TEAM-001"
}
```

---

## Error Code Reference

| HTTP Code | Error Code | Description |
|-----------|------------|-------------|
| 400 | TEAM-001 | Team not found |
| 400 | TEAM-002 | Invalid team ID (must be 1-12) |
| 400 | TEAM-003 | Role not found in team |
| 400 | TEAM-004 | Person already assigned to role |
| 400 | TEAM-005 | Team size violation (TEAM-007) |
| 401 | AUTH-001 | Authentication required |
| 401 | AUTH-002 | Invalid API key |
| 403 | AUTH-003 | Insufficient permissions |
| 404 | PROJ-001 | Project not found |
| 404 | PROJ-002 | Project configuration missing |
| 429 | RATE-001 | Rate limit exceeded |
| 500 | SERV-001 | Internal server error |
| 500 | SERV-002 | Team manager script failure |

---

## 400 Bad Request Errors

### TEAM-001: Team Not Found

**Cause:** The specified team ID does not exist for the project.

**Example:**
```json
{
  "IsError": true,
  "Content": [{
    "Type": "text",
    "Text": "TEAM-001: Team not found"
  }],
  "error_code": "TEAM-001",
  "error_message": "Team with ID 99 does not exist in project 'my-project'"
}
```

**Troubleshooting:**
1. Verify the team ID is between 1 and 12
2. Run `guardrail_team_list` to see available teams
3. Check that the project was initialized with `guardrail_team_init`

---

### TEAM-002: Invalid Team ID

**Cause:** Team ID is outside the valid range (1-12).

**Example:**
```json
{
  "IsError": true,
  "Content": [{
    "Type": "text",
    "Text": "TEAM-002: Invalid team ID"
  }],
  "error_code": "TEAM-002",
  "error_message": "Team ID must be between 1 and 12, got: 15"
}
```

**Troubleshooting:**
1. Use team IDs 1-12 only (see Team Structure section)
2. Verify your mapping logic for team assignments

---

### TEAM-003: Role Not Found

**Cause:** Attempted to assign/unassign a role that does not exist in the team.

**Example:**
```json
{
  "IsError": true,
  "Content": [{
    "Type": "text",
    "Text": "TEAM-003: Role not found"
  }],
  "error_code": "TEAM-003",
  "error_message": "Role 'Junior Developer' not found in Team 7 (Core Feature Squad)"
}
```

**Troubleshooting:**
1. Check TEAM_STRUCTURE.md for valid role names per team
2. Use exact role names (case-sensitive)
3. Run `guardrail_team_list` to see assigned roles

---

### TEAM-004: Person Already Assigned

**Cause:** Attempted to assign a person to a role that is already filled.

**Example:**
```json
{
  "IsError": true,
  "Content": [{
    "Type": "text",
    "Text": "TEAM-004: Person already assigned"
  }],
  "error_code": "TEAM-004",
  "error_message": "Role 'Technical Lead' in Team 7 already has 'Alice Johnson' assigned"
}
```

**Troubleshooting:**
1. Unassign the current person first with `guardrail_team_unassign`
2. Or assign the new person to a different role
3. Check current assignments with `guardrail_team_list`

---

### TEAM-005: Team Size Violation

**Cause:** Operation would violate TEAM-007 compliance (4-6 members per team).

**Example:**
```json
{
  "IsError": true,
  "Content": [{
    "Type": "text",
    "Text": "TEAM-005: Team size violation"
  }],
  "error_code": "TEAM-005",
  "error_message": "Team 7 has 6 members (maximum). Cannot add more members."
}
```

**Troubleshooting:**
1. Check current team size with `guardrail_team_size_validate`
2. Unassign a member before adding a new one
3. Verify team size requirements in TEAM_STRUCTURE.md

---

## 401 Unauthorized Errors

### AUTH-001: Authentication Required

**Cause:** Request missing authentication token.

**Example:**
```json
{
  "IsError": true,
  "Content": [{
    "Type": "text",
    "Text": "AUTH-001: Authentication required"
  }],
  "error_code": "AUTH-001",
  "error_message": "API key required for this endpoint"
}
```

**Troubleshooting:**
1. Include `Authorization: Bearer YOUR_API_KEY` header
2. Verify API key is valid and not expired
3. Check API key permissions

---

### AUTH-002: Invalid API Key

**Cause:** Provided API key is invalid or revoked.

**Example:**
```json
{
  "IsError": true,
  "Content": [{
    "Type": "text",
    "Text": "AUTH-002: Invalid API key"
  }],
  "error_code": "AUTH-002",
  "error_message": "The provided API key is not valid"
}
```

**Troubleshooting:**
1. Generate a new API key from the dashboard
2. Ensure the key has not been revoked
3. Check for typos in the Authorization header

---

## 403 Forbidden Errors

### AUTH-003: Insufficient Permissions

**Cause:** Authenticated user lacks permission for the operation.

**Example:**
```json
{
  "IsError": true,
  "Content": [{
    "Type": "text",
    "Text": "AUTH-003: Insufficient permissions"
  }],
  "error_code": "AUTH-003",
  "error_message": "User 'viewer@example.com' cannot modify team assignments"
}
```

**Troubleshooting:**
1. Verify user has appropriate role (admin, team-lead)
2. Check project permissions in admin panel
3. Contact project administrator for access

---

## 404 Not Found Errors

### PROJ-001: Project Not Found

**Cause:** Project name does not exist.

**Example:**
```json
{
  "IsError": true,
  "Content": [{
    "Type": "text",
    "Text": "PROJ-001: Project not found"
  }],
  "error_code": "PROJ-001",
  "error_message": "Project 'nonexistent-project' does not exist"
}
```

**Troubleshooting:**
1. Initialize project first with `guardrail_team_init`
2. Verify project name spelling (case-sensitive)
3. Check project exists: `guardrail_team_list --project-name <name>`

---

### PROJ-002: Project Configuration Missing

**Cause:** Project was partially initialized or config file corrupted.

**Example:**
```json
{
  "IsError": true,
  "Content": [{
    "Type": "text",
    "Text": "PROJ-002: Project configuration missing"
  }],
  "error_code": "PROJ-002",
  "error_message": "Team configuration file missing for project 'my-project'"
}
```

**Troubleshooting:**
1. Re-initialize project with `guardrail_team_init`
2. Check `.teams/` directory for configuration files
3. Restore from backup if available

---

## 429 Rate Limit Exceeded

### RATE-001: Rate Limit Exceeded

**Cause:** Too many requests in a short time period.

**Example:**
```json
{
  "IsError": true,
  "Content": [{
    "Type": "text",
    "Text": "RATE-001: Rate limit exceeded"
  }],
  "error_code": "RATE-001",
  "error_message": "Rate limit exceeded. Retry after 60 seconds."
}
```

**Troubleshooting:**
1. Implement exponential backoff in batch scripts
2. Reduce request frequency (default limit: 100 req/min)
3. Contact support to increase rate limits

**Retry Strategy:**
```bash
# Example with exponential backoff
for i in 1 2 4 8; do
    response=$(curl -s ...)
    if ! echo "$response" | grep -q "RATE-001"; then
        break
    fi
    echo "Rate limited. Retrying in ${i}s..."
    sleep $i
done
```

---

## 500 Internal Server Error

### SERV-001: Internal Server Error

**Cause:** Unexpected server error.

**Example:**
```json
{
  "IsError": true,
  "Content": [{
    "Type": "text",
    "Text": "SERV-001: Internal server error"
  }],
  "error_code": "SERV-001",
  "error_message": "An unexpected error occurred. Incident ID: abc-123-xyz"
}
```

**Troubleshooting:**
1. Retry the request after a brief delay
2. Check service status page for outages
3. Contact support with the incident ID

---

### SERV-002: Team Manager Execution Failure

**Cause:** Backend team management operation failed.

**Example:**
```json
{
  "IsError": true,
  "Content": [{
    "Type": "text",
    "Text": "SERV-002: Team manager execution failure"
  }],
  "error_code": "SERV-002",
  "error_message": "Team operation failed: unable to initialize team"
}
```

**Troubleshooting:**
1. Check server logs for error details
2. Verify `.teams/` directory has write permissions
3. Ensure project name is valid (alphanumeric, hyphens, underscores only)

---

## Validation Errors

If parameter validation fails, tools return an error response:

```json
{
  "IsError": true,
  "Content": [{
    "Type": "text",
    "Text": "project_name must contain only letters, numbers, hyphens, and underscores"
  }],
  "error_code": "VALID-001",
  "error_message": "Invalid project_name format",
  "validation_errors": [{
    "field": "project_name",
    "code": "INVALID_CHARS",
    "message": "Contains invalid characters"
  }]
}
```

---

## Related Documentation

- [team-tools.md](./team-tools.md) - Overview of all team tools
- [team-tools-validation.md](./team-tools-validation.md) - Input validation rules and compliance
- [team-tools-workflows.md](./team-tools-workflows.md) - Batch operations with error handling
