# Guardrail Gaps 2026 — OpenSpec Index

**Created:** 2026-08-22
**Status:** Reconciled 2026-10-03 — see [STATUS.md](STATUS.md)
**Context:** Gap analysis of agent-guardrails-template vs 2026 AI safety guardrail systems (NeMo, Llama Guard, Lakera, Constitutional AI, NIST AI RMF, EU AI Act)

> **These specs describe work that was proposed, not work that shipped.**
> Of the 13 MCP tools they specify, 2 exist. Much of the underlying library
> code was written but is unreachable from any tool. Read
> [STATUS.md](STATUS.md) before acting on any requirement here.

---

## Overview

This directory contains full OpenSpecs to close the 6 identified gaps in agent-guardrails-template. Each spec defines the problem, proposed solution, technical requirements, implementation approach, and testing criteria.

The specs are ordered by priority (critical → important → nice-to-have).

---

## Spec Documents

Status is from the 2026-10-03 reconciliation, not from the original drafting.

| # | Gap | Priority | Spec File | Status |
|---|-----|----------|-----------|--------|
| 1 | [Prompt Injection Defense](01-prompt-injection-defense.md) | 🔴 Critical | `01-prompt-injection-defense.md` | Library only — no tool, unwired |
| 2 | [Semantic Content Filtering](02-semantic-content-filtering.md) | 🔴 Critical | `02-semantic-content-filtering.md` | **Shipped** — both tools live |
| 3 | [Runtime Sandbox Isolation](03-runtime-sandbox-isolation.md) | 🟡 Important | `03-runtime-sandbox-isolation.md` | Library only — no tool, unwired |
| 4 | [Multi-Agent Safety Policies](04-multi-agent-safety-policies.md) | 🟡 Important | `04-multi-agent-safety-policies.md` | Library only — no tool, unwired |
| 5 | [Indirect Prompt Injection Handling](05-indirect-prompt-injection.md) | 🟡 Important | `05-indirect-prompt-injection.md` | Config shipped; tracker unwired |
| 6 | [Regulatory Compliance Mapping](06-regulatory-compliance-mapping.md) | 🟢 Nice-to-have | `06-regulatory-compliance-mapping.md` | Library only — and scores are simulated |

Full requirement-by-requirement position: **[STATUS.md](STATUS.md)**.

---

## Follow-up proposals — separate from the six gap specs

Two adjacent ideas came from reviewing the private
`guardrails-control-plane` and `guardrail-policy-packs` repositories. They are
separate proposals, not requirements silently folded into the six gap specs:

- [07 — Versioned Policy-Pack Governance](../07-versioned-policy-pack-governance.md)
  defines immutable policy identity, overlays, exceptions, and trust handling.
  The private policy-pack repo is prior art; its domain packs are empty seeds,
  not implemented policy coverage.
- [08 — Control-Plane Composition and Evidence](../08-control-plane-composition-and-evidence.md)
  defines how verified components could be composed and how evaluations could
  prove which controls ran. DevGate's game-quality OpenSpecs remain outside
  this guardrail scope.

Both are **proposed only**. Neither is evidence of an existing loader,
control plane, or evidence service.

---

## Design Principles

1. **Backward-compatible** — All specs extend the existing MCP server; no breaking changes to current tools.
2. **Opt-in by default** — New capabilities are disabled unless explicitly configured.
3. **Pluggable backends** — Content safety can use Llama Guard, NeMo, or custom classifiers.
4. **Fail-closed** — When a guardrail cannot determine safety, it blocks the action.
5. **Observable** — Every guardrail decision is logged with reasoning for audit trails.

---

## Cross-Cutting Concerns — Proposed vs Shipped

These were design assumptions when the specs were drafted. They are **not**
all current system guarantees:

| Concern | Spec assumption | Current position |
|---------|-----------------|------------------|
| Configuration | `guardrails.yaml` extends existing config | No `guardrails.yaml` exists; several YAML loaders are library-only and have no production caller |
| Logging | Structured JSON events to PostgreSQL | Go `slog` and audit stores exist, but the injection pipeline only logs via slog; no unified guardrail-decision trail |
| Metrics | Prometheus counters for guardrail decisions | HTTP/MCP and circuit-breaker metrics exist in `internal/metrics`; the guardrail subsystems do not emit per-decision metrics |
| Testing | Unit + integration + cross-spec tests | Unit tests exist, but several end-to-end acceptance paths have no harness or request-path wiring |

See [STATUS.md](STATUS.md) for the detailed evidence and
[the implementation status section](01-prompt-injection-defense.md#8-implementation-status-reconciled-2026-10-03)
in each spec for its requirement-level position.

---

## Original Proposed Order

The phase order below records the original proposal, **not shipped status**.

- **Phase 1:** 01 Prompt Injection Defense (foundation for 05); 02 Semantic
  Content Filtering (foundation for 04).
- **Phase 2:** 03 Runtime Sandbox Isolation; 05 Indirect Prompt Injection
  (builds on 01); 04 Multi-Agent Safety Policies (builds on 02).
- **Phase 3:** 06 Regulatory Compliance Mapping (builds on all).

For delivery status see [the reconciliation](STATUS.md). The key distinction:
02 has two registered tools; the other five specs' proposed tools do not
exist, even where library code is present.
