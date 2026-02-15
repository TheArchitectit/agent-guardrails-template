# Team Layout Management Tools

## Overview

The MCP Team Layout Management Tools provide programmatic access to the standardized enterprise team structure. These tools enable agents to initialize projects, manage team assignments, track phase progress, and validate phase gate requirements across the 5-phase development lifecycle.

## Team Structure

The enterprise team layout consists of **12 teams** organized across **5 phases**:

| Phase | Teams | Description |
|-------|-------|-------------|
| Phase 1 | Teams 1-3 | Strategy, Governance & Planning |
| Phase 2 | Teams 4-6 | Platform & Foundation |
| Phase 3 | Teams 7-8 | Build & Implementation |
| Phase 4 | Teams 9-10 | Validation & Security |
| Phase 5 | Teams 11-12 | Delivery & Operations |

See [TEAM_STRUCTURE.md](TEAM_STRUCTURE.md) for detailed team definitions and [../.guardrails/team-layout-rules.json](../.guardrails/team-layout-rules.json) for compliance rules.

## Available Tools

### guardrail_team_init

Initialize team structure for a project.

**Parameters:**
- `project_name` (string, required): Name of the project to initialize
  - Must be 64 characters or less
  - Allowed characters: letters, numbers, hyphens, underscores
  - Regex: `^[a-zA-Z0-9_-]+$`

**Example:**
```json
{
  "project_name": "my-awesome-project"
}
```

**Returns:**
- Success: Confirmation message with initialized team structure
- Error: Validation error if project_name is invalid

---

### guardrail_team_list

List all teams and their status for a project.

**Parameters:**
- `project_name` (string, required): Name of the project
- `phase` (string, optional): Filter by phase name (e.g., "Phase 1")

**Example:**
```json
{
  "project_name": "my-awesome-project",
  "phase": "Phase 1"
}
```

**Returns:**
- List of teams with their status (not_started, active, completed, blocked)
- Team members and role assignments
- Phase completion percentages

---

### guardrail_team_assign

Assign a person to a role in a team.

**Parameters:**
- `project_name` (string, required): Name of the project
- `team_id` (integer, required): Team ID (1-12)
- `role_name` (string, required): Name of the role to assign
- `person` (string, required): Name of the person to assign

**Example:**
```json
{
  "project_name": "my-awesome-project",
  "team_id": 7,
  "role_name": "Senior Backend Engineer",
  "person": "Alice Developer"
}
```

**Returns:**
- Success: Confirmation of assignment
- Error: Validation error if team/role doesn't exist

---

### guardrail_team_status

Get phase or project status.

**Parameters:**
- `project_name` (string, required): Name of the project
- `phase` (string, optional): Phase name (shows all phases if omitted)

**Example:**
```json
{
  "project_name": "my-awesome-project",
  "phase": "Phase 3"
}
```

**Returns:**
- Phase completion status
- Required vs completed deliverables
- Blocking issues

---

### guardrail_phase_gate_check

Check if phase gate requirements are met.

**Parameters:**
- `project_name` (string, required): Name of the project
- `from_phase` (integer, required): Starting phase (1-4)
- `to_phase` (integer, required): Target phase (2-5)

**Example:**
```json
{
  "project_name": "my-awesome-project",
  "from_phase": 2,
  "to_phase": 3
}
```

**Returns:**
- Gate name (e.g., "Environment Readiness")
- Required teams and their status
- Required deliverables checklist
- Approval status

**Phase Gates:**

| Transition | Gate Name | Required Teams | Deliverables |
|------------|-----------|----------------|--------------|
| 1 → 2 | Architecture Review Board | Teams 1, 2, 3 | Architecture Decision Records, Approved Tech List, Compliance Checklist |
| 2 → 3 | Environment Readiness | Teams 4, 5, 6 | Infrastructure Provisioned, CI/CD Pipelines, Data Models |
| 3 → 4 | Feature Complete + Code Review | Teams 7, 8 | Features Implemented, Code Reviewed, Documentation Complete |
| 4 → 5 | Security + QA Sign-off | Teams 9, 10 | Security Review Passed, Test Coverage Met, UAT Sign-off |

---

### guardrail_agent_team_map

Get the team assignment for an agent type.

**Parameters:**
- `agent_type` (string, required): Type of agent
  - Valid values: planner, architect, infrastructure, platform, backend, frontend, security, qa, sre, ops

**Example:**
```json
{
  "agent_type": "backend"
}
```

**Returns:**
- Assigned team ID
- Recommended roles
- Associated phase

**Agent Mappings:**

| Agent Type | Team | Phase | Roles |
|------------|------|-------|-------|
| planner | 2 | Phase 1 | Solution Architect |
| architect | 2 | Phase 1 | Chief Architect, Domain Architect |
| infrastructure | 4 | Phase 2 | Cloud Architect, IaC Engineer |
| platform | 5 | Phase 2 | CI/CD Architect, Kubernetes Administrator |
| backend | 7 | Phase 3 | Senior Backend Engineer |
| frontend | 7 | Phase 3 | Senior Frontend Engineer, Accessibility Expert |
| security | 9 | Phase 4 | Security Architect |
| qa | 10 | Phase 4 | QA Architect, SDET |
| sre | 11 | Phase 5 | SRE Lead, Observability Engineer |
| ops | 12 | Phase 5 | Release Manager, NOC Analyst |

## Security Considerations

**Input Validation:**
All `project_name` parameters are validated to prevent command injection:
- Must match pattern: `^[a-zA-Z0-9_-]+$`
- Maximum length: 64 characters
- Cannot be empty

**Behind the Scenes:**
Team tools delegate to `scripts/team_manager.py` for persistence. Project data is stored in `.teams/{project_name}.json`.

## Related Documentation

- [TEAM_STRUCTURE.md](TEAM_STRUCTURE.md) - Complete team structure documentation
- [.guardrails/team-layout-rules.json](../.guardrails/team-layout-rules.json) - Compliance rules
- [AGENT_GUARDRAILS.md](AGENT_GUARDRAILS.md) - Safety protocols
- [docs/workflows/](../docs/workflows/) - Workflow documentation

## Example Workflow

```
1. Initialize project:
   guardrail_team_init({"project_name": "web-platform"})

2. Check agent assignment:
   guardrail_agent_team_map({"agent_type": "backend"})
   → Returns: Team 7, Phase 3

3. Assign team members:
   guardrail_team_assign({
     "project_name": "web-platform",
     "team_id": 7,
     "role_name": "Senior Backend Engineer",
     "person": "Alice Developer"
   })

4. Check phase gate before transitioning:
   guardrail_phase_gate_check({
     "project_name": "web-platform",
     "from_phase": 2,
     "to_phase": 3
   })

5. Monitor progress:
   guardrail_team_status({"project_name": "web-platform"})
```

## Compliance

Team tools enforce compliance with:
- Phase gate requirements
- Required team assignments
- Approval workflows

See [.guardrails/team-layout-rules.json](../.guardrails/team-layout-rules.json) for the complete rule set.
