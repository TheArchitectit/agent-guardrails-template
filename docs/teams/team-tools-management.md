# Team Management Tools

> MCP tools for initializing, listing, assigning, and checking team structures

These tools manage the core lifecycle of a project's team structure: creating teams, assigning and unassigning people to roles, and checking status. They are part of the 35 MCP tools exposed by the Agent Guardrails server (Go 1.25+, BSD-3-Clause licensed).

Project data is stored in `.teams/{project_name}.json` via the native Go `team` package.

---

## guardrail_team_init

Initialize team structure for a project.

**Purpose:** Creates the initial team structure configuration for a new project, setting up all 12 teams with their default roles and states.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `project_name` | string | Yes | Name of the project (alphanumeric, hyphen, underscore only) |

**Constraints:**
- Project name must be 64 characters or less
- Allowed characters: letters, numbers, hyphens (`-`), underscores (`_`)
- No spaces or special characters permitted

**Example:**

```json
{
  "method": "tools/call",
  "params": {
    "name": "guardrail_team_init",
    "arguments": {
      "project_name": "my-project"
    }
  }
}
```

**Response:** Confirmation of initialized 12-team structure for the project.

---

## guardrail_team_list

List all teams and their status.

**Purpose:** Display all teams for a project, including their assigned roles, completion status, and current state.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `project_name` | string | Yes | Name of the project |
| `phase` | string | No | Filter by phase (e.g., "Phase 1", "Phase 2") |

**Example (All Teams):**

```json
{
  "method": "tools/call",
  "params": {
    "name": "guardrail_team_list",
    "arguments": {
      "project_name": "my-project"
    }
  }
}
```

**Example (Filtered by Phase):**

```json
{
  "method": "tools/call",
  "params": {
    "name": "guardrail_team_list",
    "arguments": {
      "project_name": "my-project",
      "phase": "Phase 1"
    }
  }
}
```

**Response:** List of teams with role assignments and completion status.

---

## guardrail_team_assign

Assign a person to a role in a team.

**Purpose:** Assign team members to specific roles within a team, enabling proper resource allocation and responsibility tracking.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `project_name` | string | Yes | Name of the project |
| `team_id` | number | Yes | Team ID (1-12) |
| `role_name` | string | Yes | Name of the role to assign |
| `person` | string | Yes | Name of the person to assign |

**Example:**

```json
{
  "method": "tools/call",
  "params": {
    "name": "guardrail_team_assign",
    "arguments": {
      "project_name": "my-project",
      "team_id": 7,
      "role_name": "Technical Lead",
      "person": "Jane Developer"
    }
  }
}
```

**Response:** Confirmation of role assignment with updated team roster.

---

## guardrail_team_unassign

Remove a person from a role in a team.

**Purpose:** Unassign team members from specific roles, enabling role reassignment and team restructuring.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `project_name` | string | Yes | Name of the project |
| `team_id` | number | Yes | Team ID (1-12) |
| `role_name` | string | Yes | Name of the role to unassign |

**Example:**

```json
{
  "method": "tools/call",
  "params": {
    "name": "guardrail_team_unassign",
    "arguments": {
      "project_name": "my-project",
      "team_id": 7,
      "role_name": "Technical Lead"
    }
  }
}
```

**Response:** Confirmation of role unassignment.

---

## guardrail_team_status

Get phase or project status.

**Purpose:** Check the completion status of a specific phase or the entire project, showing which roles are assigned and which teams are ready.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `project_name` | string | Yes | Name of the project |
| `phase` | string | No | Specific phase to check (e.g., "Phase 1") |

**Example (Project Status):**

```json
{
  "method": "tools/call",
  "params": {
    "name": "guardrail_team_status",
    "arguments": {
      "project_name": "my-project"
    }
  }
}
```

**Example (Phase Status):**

```json
{
  "method": "tools/call",
  "params": {
    "name": "guardrail_team_status",
    "arguments": {
      "project_name": "my-project",
      "phase": "Phase 2"
    }
  }
}
```

**Response:** Phase status with team completion percentages and role assignments.

---

## Related Documentation

- [team-tools.md](./team-tools.md) - Overview of all team tools
- [team-tools-phase-gates.md](./team-tools-phase-gates.md) - Phase gate tools
- [team-tools-validation.md](./team-tools-validation.md) - Input validation and compliance rules
- [team-tools-errors.md](./team-tools-errors.md) - Error code reference
- [team-structure.md](./team-structure.md) - Complete team structure and role definitions
