# Specs

Design specifications and change proposals for the guardrails MCP server.

| Document | Purpose |
|----------|---------|
| [09-system-roadmap-and-phase-gates.md](09-system-roadmap-and-phase-gates.md) | Master roadmap: transform the repo into a verified MCP guardrail system |
| [10-mcp-protocol-and-fastmcp-parity.md](10-mcp-protocol-and-fastmcp-parity.md) | MCP contract/test parity informed by FastMCP implementations |
| [11-authorization-scopes-and-roles.md](11-authorization-scopes-and-roles.md) | Proposed intersection of scoped API keys and named roles; migration plan |
| [12-web-exposure-boundary.md](12-web-exposure-boundary.md) | Public routes, CORS, trusted proxies, and deployment exposure profiles |
| [13-webhook-ssrf-hardening.md](13-webhook-ssrf-hardening.md) | Delivery-time DNS pinning, redirect policy, and bounded webhook egress |
| [14-optional-oap-guardrail-checker/spec.md](14-optional-oap-guardrail-checker/spec.md) | Conditional OAP checker boundary; no identity or effect authority |
| [AUTH-01-mcp-endpoint-auth.md](AUTH-01-mcp-endpoint-auth.md) | Bearer authentication on `/mcp` — implemented and merged |
| [guardrail-gaps-2026/index.md](guardrail-gaps-2026/index.md) | Six gap-analysis specs vs 2026 AI safety systems |
| [07-versioned-policy-pack-governance.md](07-versioned-policy-pack-governance.md) | Proposed immutable policy-pack resolution, overlays, trust boundaries, and lock records |
| [08-control-plane-composition-and-evidence.md](08-control-plane-composition-and-evidence.md) | Proposed component composition, integration states, and replayable evidence |

## Guardrail gaps 2026

Six OpenSpecs written against a 2026 gap analysis (NeMo, Llama Guard,
Lakera, Constitutional AI, NIST AI RMF, EU AI Act).

| # | Spec | Priority |
|---|------|----------|
| 1 | [Prompt injection defense](guardrail-gaps-2026/01-prompt-injection-defense.md) | Critical |
| 2 | [Semantic content filtering](guardrail-gaps-2026/02-semantic-content-filtering.md) | Critical |
| 3 | [Runtime sandbox isolation](guardrail-gaps-2026/03-runtime-sandbox-isolation.md) | Important |
| 4 | [Multi-agent safety policies](guardrail-gaps-2026/04-multi-agent-safety-policies.md) | Important |
| 5 | [Indirect prompt injection](guardrail-gaps-2026/05-indirect-prompt-injection.md) | Important |
| 6 | [Regulatory compliance mapping](guardrail-gaps-2026/06-regulatory-compliance-mapping.md) | Nice-to-have |

**These specs are proposals, not descriptions of shipped behaviour.** Part of
the underlying code exists in `mcp-server/internal/guardrails/`, but several
of the tools these specs propose were never exposed over MCP. Before acting
on one, verify the tool exists — see
[guardrail-gaps-2026/STATUS.md](guardrail-gaps-2026/STATUS.md) for a
requirement-by-requirement reconciliation, and
[../mcp-server/tools-reference.md](../mcp-server/tools-reference.md) for
what the server actually exposes.

## Verifying a proposed tool

```bash
grep -n "case \"<tool_name>\"" mcp-server/internal/mcp/server.go
grep -rn "<tool_name>" -g "*.go" mcp-server/
```

No hits in Go means the library may exist, but nothing can call it.