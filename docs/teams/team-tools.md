# Team Layout Management Tools

> MCP tools for managing standardized team structure across projects

The Team Layout Management system provides MCP tools to initialize, manage, and validate team structures for software development projects. It enforces a standardized 12-team structure across 5 phases of the development lifecycle, ensuring proper governance, phase gates, and role assignments.

These tools are part of the 35 MCP tools and 11 resources exposed by the Agent Guardrails server. They use the Go `team` package (Go 1.25+, BSD-3-Clause licensed) in `mcp-server/internal/team/` to provide real-time team management capabilities through the MCP protocol. Project data is stored in `.teams/{project_name}.json`.

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

For complete team details, see [team-structure.md](./team-structure.md).

---

## Tool Reference

The 8 team-management tools are documented in focused sub-documents:

| Tool | Description | Document |
|------|-------------|----------|
| `guardrail_team_init` | Initialize team structure for a project | [Management](./team-tools-management.md) |
| `guardrail_team_list` | List all teams and their status | [Management](./team-tools-management.md) |
| `guardrail_team_assign` | Assign a person to a role in a team | [Management](./team-tools-management.md) |
| `guardrail_team_unassign` | Remove a person from a role in a team | [Management](./team-tools-management.md) |
| `guardrail_team_status` | Get phase or project status | [Management](./team-tools-management.md) |
| `guardrail_phase_gate_check` | Check if phase gate requirements are met | [Phase Gates](./team-tools-phase-gates.md) |
| `guardrail_agent_team_map` | Get the team assignment for an agent type | [Agent Mapping](./team-tools-agent-mapping.md) |
| `guardrail_team_size_validate` | Validate team sizes meet the 4-6 member requirement | [Validation](./team-tools-validation.md) |

---

## Sub-Documents

- **[team-tools-management.md](./team-tools-management.md)** - Reference for init, list, assign, unassign, and status tools
- **[team-tools-phase-gates.md](./team-tools-phase-gates.md)** - Phase gate check tool and detailed gate requirements for all 4 gates
- **[team-tools-agent-mapping.md](./team-tools-agent-mapping.md)** - Agent-to-team mapping tool and supported agent types
- **[team-tools-validation.md](./team-tools-validation.md)** - Input validation rules (project name, role name, person name, phase), TEAM-007 compliance, and implementation details
- **[team-tools-errors.md](./team-tools-errors.md)** - Error response format and full error code reference (TEAM-001 through SERV-002)
- **[team-tools-workflows.md](./team-tools-workflows.md)** - Workflow patterns, complete project initialization example, and batch operation scripts

---

## Phase Gates Summary

Phase gates ensure proper completion and approval before progressing to the next phase:

| Gate | Transition | Name | Document |
|------|-----------|------|----------|
| 1_to_2 | Phase 1 to Phase 2 | Architecture Review Board | [Phase Gates](./team-tools-phase-gates.md) |
| 2_to_3 | Phase 2 to Phase 3 | Environment Readiness | [Phase Gates](./team-tools-phase-gates.md) |
| 3_to_4 | Phase 3 to Phase 4 | Feature Complete + Code Review | [Phase Gates](./team-tools-phase-gates.md) |
| 4_to_5 | Phase 4 to Phase 5 | Security + QA Sign-off | [Phase Gates](./team-tools-phase-gates.md) |

---

## Related Documentation

- [team-structure.md](./team-structure.md) - Complete team structure and role definitions
- [team-layout-rules.json](../../.guardrails/team-layout-rules.json) - Machine-readable team layout rules
- [AGENT_GUARDRAILS.md](../getting-started/agent-guardrails.md) - Core safety protocols for agents
