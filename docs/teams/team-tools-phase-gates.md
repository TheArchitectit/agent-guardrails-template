# Phase Gate Tools

> MCP tool and reference for phase gate validation across the development lifecycle

Phase gates ensure proper completion and approval before progressing from one phase to the next. The `guardrail_phase_gate_check` tool validates that all requirements are satisfied before a phase transition.

---

## guardrail_phase_gate_check

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

## Gate Details

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

## Related Documentation

- [team-tools.md](./team-tools.md) - Overview of all team tools
- [team-tools-management.md](./team-tools-management.md) - Team management tools
- [team-tools-workflows.md](./team-tools-workflows.md) - Workflow patterns including phase transitions
- [team-structure.md](./team-structure.md) - Complete team structure and role definitions
