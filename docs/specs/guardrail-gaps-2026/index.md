# Guardrail Gaps 2026 — OpenSpec Index

**Created:** 2026-08-22
**Status:** Draft
**Context:** Gap analysis of agent-guardrails-template vs 2026 AI safety guardrail systems (NeMo, Llama Guard, Lakera, Constitutional AI, NIST AI RMF, EU AI Act)

---

## Overview

This directory contains full OpenSpecs to close the 6 identified gaps in agent-guardrails-template. Each spec defines the problem, proposed solution, technical requirements, implementation approach, and testing criteria.

The specs are ordered by priority (critical → important → nice-to-have).

---

## Spec Documents

| # | Gap | Priority | Spec File | Status |
|---|-----|----------|-----------|--------|
| 1 | [Prompt Injection Defense](01-prompt-injection-defense.md) | 🔴 Critical | `01-prompt-injection-defense.md` | Draft |
| 2 | [Semantic Content Filtering](02-semantic-content-filtering.md) | 🔴 Critical | `02-semantic-content-filtering.md` | Draft |
| 3 | [Runtime Sandbox Isolation](03-runtime-sandbox-isolation.md) | 🟡 Important | `03-runtime-sandbox-isolation.md` | Draft |
| 4 | [Multi-Agent Safety Policies](04-multi-agent-safety-policies.md) | 🟡 Important | `04-multi-agent-safety-policies.md` | Draft |
| 5 | [Indirect Prompt Injection Handling](05-indirect-prompt-injection.md) | 🟡 Important | `05-indirect-prompt-injection.md` | Draft |
| 6 | [Regulatory Compliance Mapping](06-regulatory-compliance-mapping.md) | 🟢 Nice-to-have | `06-regulatory-compliance-mapping.md` | Draft |

---

## Design Principles

1. **Backward-compatible** — All specs extend the existing MCP server; no breaking changes to current tools.
2. **Opt-in by default** — New capabilities are disabled unless explicitly configured.
3. **Pluggable backends** — Content safety can use Llama Guard, NeMo, or custom classifiers.
4. **Fail-closed** — When a guardrail cannot determine safety, it blocks the action.
5. **Observable** — Every guardrail decision is logged with reasoning for audit trails.

---

## Cross-Cutting Concerns

All specs share:
- **Configuration**: `guardrails.yaml` (extends existing config)
- **Logging**: Structured JSON events to PostgreSQL (extends existing audit trail)
- **Metrics**: Prometheus-compatible counters for guardrail decisions
- **Testing**: Unit + integration tests per spec, plus cross-spec integration tests

---

## Implementation Order

```
Phase 1 (Critical):
  → 01-prompt-injection-defense (foundation for 05)
  → 02-semantic-content-filtering (foundation for 04)

Phase 2 (Important):
  → 03-runtime-sandbox-isolation (independent)
  → 05-indirect-prompt-injection (builds on 01)
  → 04-multi-agent-safety-policies (builds on 02)

Phase 3 (Nice-to-have):
  → 06-regulatory-compliance-mapping (builds on all)
```
