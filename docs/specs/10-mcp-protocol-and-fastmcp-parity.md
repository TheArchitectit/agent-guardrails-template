# OpenSpec: MCP Protocol Completeness and FastMCP-Informed Parity

**Status:** Proposed — comparative review completed 2026-10-04; no claim that
all listed protocol surfaces exist locally.
**Priority:** High for schema/transport correctness; optional for framework-
specific conveniences.
**Purpose:** Use the Python `PrefectHQ/fastmcp` and TypeScript
`punkpeye/fastmcp` projects as reference implementations to identify useful
MCP server capabilities, while preserving this repository’s Go architecture
and guardrail purpose. This is a parity specification, not a proposal to port
FastMCP itself.

## 1. Problem

The Go server has a real MCP endpoint and substantial tool/resource coverage,
but source and client audits show schema/handler drift, limited end-to-end
protocol tests, no prompt registration found in the inspected MCP package,
and client bridges that may use incompatible transport paths. FastMCP
provides useful examples of schemas, resources, prompts, authentication,
context/lifecycle, middleware, progress and client transports. Some are MCP
protocol capabilities; others are framework-specific conveniences and must
not be copied merely for feature-count parity.

## 2. Goals and non-goals

### Goals

1. Make every registered tool’s advertised input schema match handler
   extraction, validation, defaults, and output contract.
2. Test the actual MCP transport path: initialize, capabilities, discovery,
   invocation, errors, authentication, and shutdown.
3. Decide and document support for MCP resources, resource templates,
   prompts, notifications/progress, cancellation, and request context.
4. Make supported client adapters use the server’s actual transport and
   schemas; publish a versioned compatibility matrix.
5. Keep authentication/authorization, middleware and lifecycle behavior
   consistent across all entry points.

### Non-goals

- Rewriting the Go server in Python or TypeScript.
- Reproducing FastMCP decorators, generated Python docstring schemas, its
  generic client SDK, CLI, OpenAPI bridge, Edge/Hono integration, or JSR/npm
  packaging unless a product requirement independently justifies them.
- Adding OAuth/OIDC providers by default. The authorization contract belongs
  to the security workstream and must be decided separately.
- Treating “supports a FastMCP feature” as proof that a guardrail is enforced.

## 3. Reference findings

The comparison used public FastMCP Python (`PrefectHQ/fastmcp`) and
TypeScript (`punkpeye/fastmcp`) source trees. These are references, not
normative dependencies.

| Area | Reference capability | Local baseline | Relevance |
|------|----------------------|----------------|-----------|
| Tools/schemas | typed schema generation, validation, structured results, annotations, per-tool access hooks | 37 core tool schemas plus conditional tools; audit found schema/handler mismatches | **Required parity:** accurate contract, output/error semantics, authorization hook |
| Resources | static resources, templates, subscriptions in frameworks | 11 fixed resources; no templates/subscriptions found in inspected Go MCP paths | Resource templates are an optional protocol feature; assess client need |
| Prompts | prompt registration and arguments | no prompt registration found in inspected MCP package | Optional unless reusable MCP prompts are a product requirement |
| Auth | bearer/JWT/OAuth/OIDC and per-tool access hooks | bearer key on `/mcp`; no MCP role model | Authenticated baseline exists; roles/scopes are separately governed by Phase 0 |
| Transports | stdio and HTTP transports, SSE compatibility in framework versions | stateless Streamable HTTP at `/mcp` | HTTP works; additional transports are product choices, not mandatory parity |
| Context/lifecycle | request context, middleware, startup/shutdown lifecycle hooks | server lifecycle and separate web middleware; consistent MCP cross-cutting hooks not evident | Useful for cancellation, tracing and authorization consistency |
| Progress/tasks | progress notifications and framework task helpers | task-attempt tracking is application data, not proof of MCP protocol Tasks support | Protocol progress/cancellation should be evaluated separately |
| Clients/tests | generic clients and broad protocol tests | server-focused tests; full live MCP contract suite is not run in current CI | **Required parity:** discover/invoke/auth/error tests through real transport |

The Pi bridge’s documented SSE endpoint does not match this server’s
Streamable HTTP `/mcp` endpoint; see
[`platform-current-state.md`](../platform-current-state.md) §6. Compatibility
must be tested against the versioned Go contract, not inferred from similar
tool names.

## 4. Requirements

### 4.1 Tool contract integrity

1. For each registered tool, one canonical contract MUST define its name,
   input schema, required fields, types, defaults, output schema or documented
   result shape, error semantics, and authorization requirement.
2. A conformance test MUST compare schema declarations to handler argument
   reads. Undeclared required reads and declared-but-unused inputs MUST fail
   CI unless explicitly allowlisted with a deprecation/removal issue.
3. Tool handlers MUST validate boundary inputs, return stable structured
   errors, and never silently coerce malformed values into a successful
   default.
4. Generated documentation and client bindings MUST derive from or be
   validated against this canonical contract.
5. Conditional tools MUST declare their activation requirements. Startup
   diagnostics MUST report which tools are registered and which dependencies
   are unavailable; generated static counts MUST not be presented as a live
   inventory.

### 4.2 Protocol integration tests

1. CI MUST start an in-process or containerized server using the production
   transport implementation and test MCP `initialize`, capability exchange,
   tool/resource listing, a representative call, invalid arguments, and
   shutdown.
2. Authentication tests MUST prove unauthenticated requests fail before
   method dispatch and valid credentials can initialize and invoke an
   authorized read operation.
3. Every supported client/transport pair MUST have a compatibility test for
   endpoint path, transport version, authentication header, and schema use.
4. Protocol errors, tool-level `IsError`, JSON payload errors, HTTP status,
   and transport failures MUST be distinguished and documented.
5. Transport-level tests MUST not rely solely on direct handler calls or
   mocked MCP calls.

### 4.3 Optional protocol surface

Before implementing additional surface, record the product decision:

- **Prompts:** if added, define names/arguments and test list/get behavior.
- **Resource templates/subscriptions:** add only for genuinely dynamic
  resources; define URI matching and update notifications.
- **Progress/cancellation:** use protocol mechanisms for long-running checks;
  distinguish request cancellation from the domain-specific three-strikes
  attempt store.
- **stdio/SSE:** add only when a supported host requires it; maintain
  Streamable HTTP compatibility and test each enabled transport.
- **Client SDK:** a server does not need to reproduce FastMCP’s generic client
  library; publish a minimal example using an official supported SDK instead.

### 4.4 Cross-cutting middleware and context

1. Authentication, authorization, request IDs, timeout/cancellation, and
   structured error handling MUST apply consistently to every registered tool
   and resource.
2. Per-tool authorization MUST be checked before side effects. Confirmation
   parameters are intent signals, not authorization.
3. Request context MUST be propagated to database, network, model, and
   subprocess work where cancellation is supported.
4. Middleware failure or missing mandatory context MUST be explicit and must
   not yield a successful-looking result.

## 5. Acceptance criteria

1. CI enumerates every registered tool and proves each schema can be decoded
   by the handler contract test; drift count is zero or explicitly waived.
2. A Streamable HTTP integration test performs initialize, list-tools, list-
   resources, and one successful plus one rejected tool call.
3. Unauthorized and insufficient-role calls are rejected before the handler’s
   side effect; authorized calls succeed. Tests cover both MCP and REST paths
   according to the chosen Phase 0 auth model.
4. A client compatibility test proves the supported client can initialize,
   authenticate, discover a tool, invoke it, and parse both success and error
   results.
5. Cancellation/timeout tests prove long-running work stops or reports
   explicitly when cancelled; no silent success is returned.
6. Conditional tool registration tests cover each enabled and disabled state,
   and diagnostics report the active inventory.
7. Any protocol surface intentionally not supported (prompts, templates,
   notifications, extra transports) is listed with rationale in the
   compatibility matrix.

## 6. Sequencing and dependencies

- Phase 0 authorization model must precede tool-level `canAccess`-style rules.
- Phase 1 of `09-system-roadmap-and-phase-gates.md` owns schema/handler
  contract integrity; this spec supplies the cross-cutting protocol test
  requirements.
- Phase 2 owns wiring security capabilities into real request paths.
- Phase 5 owns client compatibility, including the Pi bridge and IDE
  integrations.
- Specs 07 and 08 may consume the canonical tool/evaluation contracts, but
  must not create a second, conflicting tool schema source.

## 7. Implementation status

The Go server has bearer-protected stateless Streamable HTTP, tools and fixed
resources. This satisfies basic server functionality, not full framework
parity. Schema drift, missing end-to-end protocol coverage, and client
transport mismatches remain. Prompts, templates, progress/cancellation and
additional transports are **unresolved optional features**, not assumed gaps
until a client/product requirement selects them.