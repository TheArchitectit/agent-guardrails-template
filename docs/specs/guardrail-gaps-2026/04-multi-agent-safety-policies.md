# OpenSpec: Multi-Agent Safety Policies

> **This is a proposal, not a description of the system.** The chain engine and three validators exist as an unwired library; none of the three proposed tools exists. See [STATUS.md](STATUS.md).

**Gap:** Important — No agent-validates-agent mechanism
**Priority:** 🟡 Important (Phase 2)
**Depends on:** 02-semantic-content-filtering
**Blocks:** None

---

## 1. Problem Statement

The agent-guardrails-template enforces the "Four Laws" on **individual agents** but has no mechanism for **inter-agent safety validation** — where one agent's output is verified by another agent (or guardrail system) before it reaches the user or downstream systems.

**Scenarios the template cannot handle:**
- Agent A generates code, Agent B reviews it for safety
- A planner agent delegates to executor agents with safety constraints
- Parallel agents produce conflicting or contradictory outputs
- An agent's output needs to be validated against the "Four Laws" before use

---

## 2. Proposed Solution

Add **Multi-Agent Safety Policies** that define how agents validate each other's outputs and enforce cross-agent safety constraints.

### 2.1 Architecture

```
┌──────────┐     ┌──────────────────┐     ┌──────────┐
│ Agent A  │────▶│ Safety Validator  │────▶│ Agent B  │
│ (output) │     │ (policy engine)   │     │ (input)  │
└──────────┘     └──────────────────┘     └──────────┘
                       │
                ┌──────▼──────┐
                │  Audit Log  │
                │  + Chain    │
                │  Tracking   │
                └─────────────┘
```

### 2.2 Safety Validation Chains

Define ordered chains of safety checks that must pass before agent output is used:

```yaml
safety_chains:
  code_generation:
    description: "Safety chain for code generation agents"
    steps:
      - validator: "injection_defense"
        action: "block"
        description: "Check for prompt injection in output"
      - validator: "content_filter"
        action: "block"
        description: "Classify content against safety taxonomy"
      - validator: "four_laws_check"
        action: "block"
        description: "Verify output follows Four Laws"
      - validator: "code_review_agent"
        action: "warn"  # warn, not block (human review)
        description: "Automated code review for safety"
    on_failure: "block_and_notify"

  research_output:
    description: "Safety chain for research agents"
    steps:
      - validator: "content_filter"
        action: "block"
        description: "Check for harmful content"
      - validator: "citation_check"
        action: "warn"
        description: "Verify citations are real"
      - validator: "fact_check_agent"
        action: "warn"
        description: "Cross-reference claims"
    on_failure: "warn_and_continue"
```

### 2.3 Cross-Agent Safety Policies

```yaml
cross_agent_policies:
  # One agent cannot override another agent's safety constraints
  no_override:
    enabled: true
    description: "Agents cannot instruct other agents to bypass guardrails"

  # Output of one agent must be validated before use by another
  chain_validation:
    enabled: true
    description: "Agent outputs pass through safety chain before next agent"

  # Parallel agents cannot produce conflicting instructions
  conflict_detection:
    enabled: true
    description: "Detect and resolve conflicts between parallel agent outputs"

  # Parent agents inherit safety constraints from their callers
  constraint_inheritance:
    enabled: true
    description: "Child agents inherit parent's safety policies"
```

---

## 3. Technical Requirements

### 3.1 New MCP Tools

```go
// guardrail_validate_agent_output — validates an agent's output against a safety chain
// Input:  { agent_id: string, output: string, chain_id: string, context?: string }
// Output: { passed: bool, steps: []StepResult, violations: []Violation }
guardrail_validate_agent_output(agent_id, output, chain_id, context?) → ChainResult

// guardrail_check_agent_constraints — checks if an agent respects its constraints
// Input:  { agent_id: string, policy_id: string }
// Output: { compliant: bool, constraints: []ConstraintResult }
guardrail_check_agent_constraints(agent_id, policy_id) → ConstraintResult

// guardrail_resolve_conflicts — resolves conflicts between parallel agent outputs
// Input:  { outputs: Array<{agent_id: string, output: string, priority: int}> }
// Output: { resolved: string, conflicts: []Conflict, resolution_method: string }
guardrail_resolve_conflicts(outputs) → ConflictResult
```

### 3.2 Agent Identity and Tracking

```yaml
agent_registry:
  agents:
    - id: "code-writer"
      type: "generator"
      safety_chain: "code_generation"
      constraints:
        - "four_laws"
        - "no_destructive_commands"
    - id: "code-reviewer"
      type: "validator"
      safety_chain: "code_review"
      constraints:
        - "four_laws"
        - "adversarial_review"
    - id: "planner"
      type: "planner"
      safety_chain: "planning"
      constraints:
        - "four_laws"
        - "scope_limitation"
```

### 3.3 Audit Trail for Agent Chains

```json
{
  "event": "agent_chain_validated",
  "timestamp": "2026-08-22T10:30:00Z",
  "chain_id": "code_generation",
  "agent_id": "code-writer",
  "parent_agent_id": "planner",
  "steps": [
    {
      "step": 1,
      "validator": "injection_defense",
      "passed": true,
      "latency_ms": 12
    },
    {
      "step": 2,
      "validator": "content_filter",
      "passed": true,
      "latency_ms": 45
    },
    {
      "step": 3,
      "validator": "four_laws_check",
      "passed": false,
      "reason": "Output modifies files outside declared scope",
      "latency_ms": 8
    }
  ],
  "overall_passed": false,
  "decision": "block",
  "violation": {
    "type": "scope_violation",
    "detail": "Agent attempted to modify /etc/hosts"
  }
}
```

---

## 4. Implementation Notes

### 4.1 Validator Interface

All validators implement a common interface:

```go
type SafetyValidator interface {
    Validate(ctx context.Context, input ValidatorInput) (*ValidatorResult, error)
    Name() string
    Description() string
}
```

Built-in validators:
- `injection_defense` — uses Spec 01
- `content_filter` — uses Spec 02
- `four_laws_check` — validates against the Four Laws
- `code_review_agent` — delegates to an LLM for code review

### 4.2 Conflict Resolution Strategies

| Strategy | Description | Use Case |
|----------|-------------|----------|
| `priority` | Highest-priority agent wins | Planner vs executor |
| `intersection` | Only actions all agents agree on | Parallel research agents |
| `union` | All actions from all agents | Non-conflicting parallel work |
| `human_escalate` | Ask human for resolution | Conflicting critical actions |

### 4.3 Constraint Inheritance

When Agent A delegates to Agent B:
- Agent B inherits all of Agent A's constraints
- Agent B cannot have fewer constraints than Agent A
- Additional constraints can be added but not removed

---

## 5. Testing Criteria

### 5.1 Unit Tests
- [ ] Safety chain executes validators in order
- [ ] Chain stops on failure when `on_failure: block`
- [ ] Constraint inheritance works correctly
- [ ] Conflict resolution strategies produce correct results

### 5.2 Integration Tests
- [ ] Agent A → safety chain → Agent B pipeline
- [ ] Multi-step chain with mixed pass/fail results
- [ ] Parallel agents with conflict detection
- [ ] Audit trail captures full chain history

### 5.3 Adversarial Tests
- [ ] Agent A tries to instruct Agent B to bypass guardrails
- [ ] Agent output contains injection that passes first validator but fails second
- [ ] Parallel agents produce contradictory code changes
- [ ] Constraint inheritance prevents downgrade

---

## 6. Dependencies

- **Internal:** 02-semantic-content-filtering, existing MCP server
- **External:** None (validators can be custom or delegate to external services)
- **Related:** 01-prompt-injection-defense (injection_defense validator)

---

## 7. References

- [CrewAI Safety](https://docs.crewai.com/) — multi-agent safety concepts
- [LangGraph Agent Supervision](https://langchain-ai.github.io/langgraph/) — agent graph safety
- [NeMo Guardrails](https://github.com/NVIDIA/NeMo-Guardrails) — Colang-based multi-turn safety
- [Anthropic Constitutional AI](https://www.anthropic.com/research/constitutional-ai-harmlessness-from-ai-feedback) — principle-based safety

---

## 8. Implementation Status (reconciled 2026-10-03)

Evidence: `docs/specs/guardrail-gaps-2026/STATUS.md`.

| § | Requirement | Status | Evidence |
|---|-------------|--------|----------|
| 3.1 | `guardrail_validate_agent_output` | **Not implemented** | name absent from every `.go` file |
| 3.1 | `guardrail_check_agent_constraints` | **Not implemented** | same |
| 3.1 | `guardrail_resolve_conflicts` | **Not implemented** | same |
| 2.x | Safety chain engine | Built, **not wired** | `multi_agent.go:131` `NewSafetyChain` is called only from `multi_agent_findings_test.go:13` |
| 2.x | Validators: injection_defense, content_filter, four_laws_check | Built, not wired | `multi_agent_validators.go:38,96,227` — no production call sites |
| 2.x | `code_review_agent` validator | **Not implemented** | no LLM code-review validator exists |
| 2.x | Conflict strategies | Built, not wired | `multi_agent_conflict.go:51-58`; shipped name is `escalate`, spec says `human_escalate` |
| 3.x | Constraint inheritance | Built, not wired | `ResolveConstraints` (`multi_agent.go:296`), test-only callers |
| 3.3 | Audit trail | Partial | `AuditEntry` mirrors the spec shape; the only logger writes to slog, not a durable store |
| 3.2 | Config keys `safety_chains`, `cross_agent_policies`, `agent_registry` | **Not implemented** | zero occurrences in any `.go` file; no loader exists, no shipped YAML uses them |

**Bug recorded here rather than silently fixed:** the `four_laws_check`
Law-2 scope test (`multi_agent_validators.go:172-181`) flags a violation when
output merely **contains** a configured scope keyword. Mentioning a declared
scope word therefore fails the check — that is not "output exceeds declared
scope", and it will misfire on compliant output once the chain is wired. It
also hard-fails as `scope_unverified` when `Context` is empty, which the spec
does not describe.

**Net position.** A complete domain model with no wiring and no configuration
path. Nothing in this spec runs today.

**Blocking decisions:**

1. §5.3 ("Agent A instructs Agent B to bypass guardrails") presupposes an
   agent-to-agent runtime this repo does not have. Either that transport
   becomes a requirement here or §5.3 moves to a spec that owns it.
2. Decide whether multi-agent validation runs in-process (chains) or via MCP
   tools (§3.1) — currently both are described and neither is built.
3. Fix the Law-2 keyword logic before wiring the chain, or the wiring ships a
   false-positive generator.
