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
| [15-secure-cross-product-method/spec.md](15-secure-cross-product-method/spec.md) | Cross-Product Evidence Auth v1: identity, signing, replay, both sides |
| [16-authentication-authorization-and-evidence-remediation.md](16-authentication-authorization-and-evidence-remediation.md) | Proposed source-audited remediation gates for auth, authorization, audit, lifecycle, and evidence; not shipped |
| [17-deployment-tls-and-secret-boundary.md](17-deployment-tls-and-secret-boundary.md) | Proposed deployment profiles, TLS, secret sources and rotation |
| [18-migration-startup-and-readiness.md](18-migration-startup-and-readiness.md) | Proposed migration ownership, startup ordering, readiness and outage behavior |
| [19-phase0-ci-security-gates.md](19-phase0-ci-security-gates.md) | Proposed same-commit blocking CI matrix, UCS03 isolation and negative controls |
| [20-mcp-parity-with-fastmcp.md](20-mcp-parity-with-fastmcp.md) | FastMCP parity closure: consolidates spec 10's required items into one closable unit (NEW 2026-10-10) |
| [21-tool-schema-handler-conformance-gate.md](21-tool-schema-handler-conformance-gate.md) | Blocking CI gate comparing advertised tool schemas to handler argument reads (NEW 2026-10-10) |
| [22-mcp-client-transport-compatibility-matrix.md](22-mcp-client-transport-compatibility-matrix.md) | Versioned client/transport compatibility matrix backed by live tests; **required row (official MCP Inspector) SUPPORTED 2026-10-10** (NEW 2026-10-10) |
| [22-evidence-2026-10-10.md](22-evidence-2026-10-10.md) | Raw official MCP Inspector step outputs for the spec 22 required row (initialize/tools-list/call/call-rejected all PASS, revision 2025-11-25) (NEW 2026-10-10) |
| [23-live-tool-resource-listing-sequence.md](23-live-tool-resource-listing-sequence.md) | Live tools/list + resources/list sequence over the production transport (NEW 2026-10-10, spin-out from spec 20 R20.3) |
| [24-pi-bridge-sse-transport-mismatch.md](24-pi-bridge-sse-transport-mismatch.md) | Pi-bridge targeted SSE `/mcp/v1/sse` the OMCP server does not serve; **fixed 2026-10-10** — switched to Streamable HTTP `/mcp/stream` + focused test (NEW 2026-10-10) |
| [25-omcp-protocol-revision-gap.md](25-omcp-protocol-revision-gap.md) | Official MCP Inspector offers MCP revision 2025-11-25, OMCP refuses it; blocks spec 22 required row (RESOLVED 2026-10-10 — OMCP adopts 2025-11-25) |
| [26-omcp-listener-self-terminates-after-grace.md](26-omcp-listener-self-terminates-after-grace.md) | OMCP HTTP serve loop self-terminates after `shutdown::GRACE` (10s) with no signal — `tokio::time::timeout` wraps the whole serve future (NEW 2026-10-10, spin-out from spec 22 run) |
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