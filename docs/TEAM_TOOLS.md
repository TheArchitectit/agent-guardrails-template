# Team Layout Management Tools

> MCP tools for managing standardized team structure across projects

**Version:** 1.0
**Applies To:** All projects using the Agent Guardrails Template

---

## Overview

The Team Layout Management system provides MCP tools to initialize, manage, and validate team structures for software development projects. It enforces a standardized 12-team structure across 5 phases of the development lifecycle, ensuring proper governance, phase gates, and role assignments.

These tools integrate with the `team_manager.py` script to provide real-time team management capabilities through the MCP protocol.

---

## Team Structure

The system manages 12 teams across 5 phases of the software development lifecycle:

### Phase 1: Strategy, Governance & Planning
- **Team 1:** Business & Product Strategy (The "Why")
- **Team 2:** Enterprise Architecture (The "Standards")
- **Team 3:** GRC (Governance, Risk, & Compliance)

### Phase 2: Platform & Foundation
- **Team 4:** Infrastructure & Cloud Ops
- **Team 5:** Platform Engineering (The "Internal Tools")
- **Team 6:** Data Governance & Analytics

### Phase 3: The Build Squads
- **Team 7:** Core Feature Squad (The "Devs")
- **Team 8:** Middleware & Integration

### Phase 4: Validation & Hardening
- **Team 9:** Cybersecurity (AppSec)
- **Team 10:** Quality Engineering (SDET)

### Phase 5: Delivery & Sustainment
- **Team 11:** Site Reliability Engineering (SRE)
- **Team 12:** IT Operations & Support (NOC)

For complete team details, see [TEAM_STRUCTURE.md](./TEAM_STRUCTURE.md).

---

## Available Tools

### guardrail_team_init

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

### guardrail_team_list

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

### guardrail_team_assign

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

### guardrail_team_status

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

### guardrail_phase_gate_check

Check if phase gate requirements are met.

**Purpose:** Validate that all requirements are satisfied before transitioning from one phase to the next, enforcing the phase gate process.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `project_name` | string | Yes | Name of the project |
| `from_phase` | number | Yes | Source phase number (1-4) |
| `to_phase` | number | Yes | Target phase number (2-5) |

**Phase Gates:**

| Gate | From | To | Name |
|------|------|-----|------|
| 1_to_2 | Phase 1 | Phase 2 | Architecture Review Board |
| 2_to_3 | Phase 2 | Phase 3 | Environment Readiness |
| 3_to_4 | Phase 3 | Phase 4 | Feature Complete + Code Review |
| 4_to_5 | Phase 4 | Phase 5 | Security + QA Sign-off |

**Example:**

```json
{
  "method": "tools/call",
  "params": {
    "name": "guardrail_phase_gate_check",
    "arguments": {
      "project_name": "my-project",
      "from_phase": 1,
      "to_phase": 2
    }
  }
}
```

**Response:** Gate name, required teams, and deliverables checklist.

---

### guardrail_agent_team_map

Get the team assignment for an agent type.

**Purpose:** Map AI agent types to their appropriate teams and roles, ensuring agents work within their designated scope.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `agent_type` | string | Yes | Type of agent (see supported types below) |

**Supported Agent Types:**

| Agent Type | Assigned Team | Phase | Roles |
|------------|---------------|-------|-------|
| `planner` | Team 2 | Phase 1 | Solution Architect, Business Systems Analyst |
| `architect` | Team 2 | Phase 1 | Chief Architect, Domain Architect |
| `infrastructure` | Team 4 | Phase 2 | Cloud Architect, IaC Engineer |
| `platform` | Team 5 | Phase 2 | CI/CD Architect, Kubernetes Administrator |
| `backend` | Team 7 | Phase 3 | Senior Backend Engineer, Technical Lead |
| `frontend` | Team 7 | Phase 3 | Senior Frontend Engineer, Accessibility Expert |
| `security` | Team 9 | Phase 4 | Security Architect, Vulnerability Researcher |
| `qa` | Team 10 | Phase 4 | QA Architect, SDET |
| `sre` | Team 11 | Phase 5 | SRE Lead, Observability Engineer |
| `ops` | Team 12 | Phase 5 | Release Manager, NOC Analyst |

**Example:**

```json
{
  "method": "tools/call",
  "params": {
    "name": "guardrail_agent_team_map",
    "arguments": {
      "agent_type": "backend"
    }
  }
}
```

**Response:** Assigned team ID, phase, and applicable roles.

---

### guardrail_team_size_validate

Validate team sizes meet the 4-6 member requirement.

**Purpose:** Ensures all teams have between 4 and 6 members (inclusive) per TEAM-007 compliance rule.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `project_name` | string | Yes | Name of the project |
| `team_id` | number | No | Optional: Specific team ID to validate |

**Example:**

```json
{
  "method": "tools/call",
  "params": {
    "name": "guardrail_team_size_validate",
    "arguments": {
      "project_name": "my-project"
    }
  }
}
```

**Response:**

```
✅ All 12 teams have valid size (4-6 members)
```

Or if violations found:

```
❌ Team size violations found:
   Team 3 (GRC) has 3 members, minimum is 4
   Team 7 (Core Feature Squad) has 8 members, maximum is 6
```

---

## Phase Gates

Phase gates ensure proper completion and approval before progressing to the next phase of development.

### Gate 1: Architecture Review Board (Phase 1 to Phase 2)

**Required Teams:** 1, 2, 3
**Approval Required:** Team 2

**Deliverables:**
- Architecture Decision Records
- Approved Tech List
- Compliance Checklist

**Purpose:** Validate that business case, architecture, and compliance requirements are established before infrastructure work begins.

---

### Gate 2: Environment Readiness (Phase 2 to Phase 3)

**Required Teams:** 4, 5, 6
**Approval Required:** Teams 4, 5

**Deliverables:**
- Infrastructure Provisioned
- CI/CD Pipelines
- Data Models

**Purpose:** Ensure platform and infrastructure are ready before development teams begin building features.

---

### Gate 3: Feature Complete + Code Review (Phase 3 to Phase 4)

**Required Teams:** 7, 8
**Approval Required:** Team 7

**Deliverables:**
- Features Implemented
- Code Reviewed
- Documentation Complete

**Purpose:** Confirm that all features are developed and reviewed before entering validation and hardening phase.

---

### Gate 4: Security + QA Sign-off (Phase 4 to Phase 5)

**Required Teams:** 9, 10
**Approval Required:** Teams 9, 10

**Deliverables:**
- Security Review Passed
- Test Coverage Met
- UAT Sign-off

**Purpose:** Ensure security clearance and quality assurance approval before production deployment.

---

## Security

### Project Name Validation

All team tools validate the `project_name` parameter to prevent command injection and ensure consistent naming:

- **Maximum Length:** 64 characters
- **Allowed Characters:**
  - Letters (a-z, A-Z)
  - Numbers (0-9)
  - Hyphens (`-`)
  - Underscores (`_`)

**Valid Examples:**
- `my-project`
- `project_123`
- `team-alpha-v2`

**Invalid Examples:**
- `my project` (contains space)
- `project;rm -rf /` (contains special characters)
- `../etc/passwd` (path traversal attempt)

### Error Handling

If validation fails, tools return an error response:

```json
{
  "IsError": true,
  "Content": [{
    "Type": "text",
    "Text": "project_name must contain only letters, numbers, hyphens, and underscores"
  }]
}
```

### Team Size Compliance (TEAM-007)

All teams **MUST** comply with the 4-6 member size requirement:

- **Minimum:** 4 members per team
- **Maximum:** 6 members per team
- **Rule ID:** TEAM-007
- **Severity:** Error

**Validation:**
Use `guardrail_team_size_validate` to check compliance:

```json
{
  "method": "tools/call",
  "params": {
    "name": "guardrail_team_size_validate",
    "arguments": {
      "project_name": "my-project"
    }
  }
}
```

**Why This Matters:**
- Teams with fewer than 4 members lack adequate role coverage
- Teams with more than 6 members suffer from coordination overhead
- This rule applies to human teams, AI agent teams, and mixed teams

### Implementation Details

Team tools delegate to `scripts/team_manager.py` for persistence. Project data is stored in `.teams/{project_name}.json`.

---

## Workflow Integration

### Typical Project Setup Workflow

```
1. Initialize team structure
   └─ guardrail_team_init → Creates all 12 teams

2. Assign team members to roles
   └─ guardrail_team_assign → Assign people to specific roles

3. Check phase status
   └─ guardrail_team_status → Verify team readiness

4. Progress through phase gates
   └─ guardrail_phase_gate_check → Validate gate requirements
```

### Agent Assignment Workflow

```
1. Determine agent type (e.g., "backend", "security")

2. Get team mapping
   └─ guardrail_agent_team_map → Identify assigned team

3. Check team status
   └─ guardrail_team_status → Verify team is active

4. Begin work within assigned scope
```

### Example: Complete Project Initialization

```bash
# Initialize project
curl -X POST "http://localhost:8094/mcp/v1/message?session_id=abc123" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"guardrail_team_init","arguments":{"project_name":"web-platform"}}}'

# Assign backend lead
curl -X POST "http://localhost:8094/mcp/v1/message?session_id=abc123" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"guardrail_team_assign","arguments":{"project_name":"web-platform","team_id":7,"role_name":"Technical Lead","person":"Alice Developer"}}}'

# Check phase gate
curl -X POST "http://localhost:8094/mcp/v1/message?session_id=abc123" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"guardrail_phase_gate_check","arguments":{"project_name":"web-platform","from_phase":2,"to_phase":3}}}'
```

---

## Related Documentation

- [TEAM_STRUCTURE.md](./TEAM_STRUCTURE.md) - Complete team structure and role definitions
- [../.guardrails/team-layout-rules.json](../.guardrails/team-layout-rules.json) - Machine-readable team layout rules
- [AGENT_GUARDRAILS.md](./AGENT_GUARDRAILS.md) - Core safety protocols for agents

---

**Last Updated:** 2026-02-15
**Version:** 1.0
