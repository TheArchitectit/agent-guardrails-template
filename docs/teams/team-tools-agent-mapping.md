# Agent-Team Mapping Tool

> MCP tool for mapping AI agent types to their designated teams and roles

The `guardrail_agent_team_map` tool maps AI agent types to their appropriate teams and roles, ensuring agents work within their designated scope.

---

## guardrail_agent_team_map

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

## Agent Assignment Workflow

```
1. Determine agent type (e.g., "backend", "security")

2. Get team mapping
   └─ guardrail_agent_team_map → Identify assigned team

3. Check team status
   └─ guardrail_team_status → Verify team is active

4. Begin work within assigned scope
```

---

## Related Documentation

- [team-tools.md](./team-tools.md) - Overview of all team tools
- [team-tools-management.md](./team-tools-management.md) - Team management tools including status
- [team-structure.md](./team-structure.md) - Complete team structure and role definitions
