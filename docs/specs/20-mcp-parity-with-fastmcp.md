# OpenSpec: FastMCP Parity Closure (consolidating spec)

**Status:** Proposed — opened 2026-10-10. Consolidates and takes over the
closure obligation of `10-mcp-protocol-and-fastmcp-parity.md`. Adds no new
capability beyond what §4.1–4.4 of spec 10 already requires; it exists so the
parity work has one closable unit with acceptance criteria and a status table.

**Priority:** High — this is the spec Roger named on 2026-10-10 ("parity with
FastMCP") and the container for closing the MCP protocol surface.

**Purpose:** Close every *required* FastMCP-derived parity item against the Go
server, without porting the Python/TypeScript frameworks. Spec 10 supplies the
doctrine; this spec supplies the closure list, the evidence rule, and the
spin-out register for items discovered during closure.

## 1. Problem

Spec 10 states the parity goal, but it is a comparative review, not a closable
worklist: its requirements are phrased as goals, several optional surfaces are
deliberately unresolved, and nothing binds a requirement to a closing test.
Two items are genuinely required (schema↔handler conformance in CI; the
versioned client/transport compatibility matrix); the rest are either optional
surface or cross-cutting hardening. Without a single closing unit, "parity with
FastMCP" cannot be declared done or not-done.

## 2. Scope

### In scope (required parity)

1. **Tool-contract integrity** — every registered tool's advertised input
   schema matches handler extraction, validation, defaults, and output shape.
2. **Conformance gate in CI** — a machine check that fails on drift (spec 21).
3. **Real-transport protocol tests** — initialize, capabilities, discovery,
   invocation, error, auth, shutdown over the production transport.
4. **Client/transport compatibility matrix** — versioned, per supported client
   (spec 22).

### Out of scope (unchanged from spec 10)

Porting FastMCP decorators, generated docstring schemas, its generic client
SDK, CLI, OpenAPI bridge, Edge/Hono integration, or JSR/npm packaging;
adding OAuth/OIDC providers by default; treating "supports a FastMCP feature"
as proof a guardrail is enforced.

## 3. Requirements

**R20.1** Every registered tool SHALL have exactly one canonical contract
(name, input schema, required fields, types, defaults, output shape, error
semantics, authorization requirement).

**R20.2** A same-commit CI gate SHALL compare schema declarations to handler
argument reads and SHALL fail on un-allowlisted drift (spec 21).

**R20.3** CI SHALL exercise the production transport for `initialize`,
capability exchange, tool/resource listing, one successful and one rejected
`tools/call`, and shutdown.

**R20.4** Authentication SHALL be proven to fail before method dispatch and
valid credentials SHALL initialize and invoke an authorized read.

**R20.5** A versioned compatibility matrix SHALL record endpoint path,
transport version, auth header, and schema use for every supported client, and
SHALL list intentionally-unsupported surfaces with rationale (spec 22).

**R20.6** A spec SHALL NOT be closed on intent: closure requires the same-commit
green CI run id and a live artifact read in the closing turn.

## 4. Spin-out register (new items become their own openspec)

Discovered items are recorded here and given their own spec file; this table is
the index only.

| Item | Owned by | State |
|---|---|---|
| Schema↔handler conformance gate | `21-tool-schema-handler-conformance-gate.md` | NEW 2026-10-10 |
| Client/transport compatibility matrix | `22-mcp-client-transport-compatibility-matrix.md` | NEW 2026-10-10 |
| OMCP private FastMCP phase package (specs 30–58, v4.0.2) | OMCP `28-fastmcp-parity-closure.md` | NEW 2026-10-10 |
| OAP guardrail-checker MCP surface | OAP `mcp-fastmcp-parity` | NEW 2026-10-10 |

## 5. Acceptance criteria

1. R20.1 contract inventory exists and drift count is zero or explicitly waived.
2. R20.2 CI gate is blocking and demonstrated red-on-drift, green-on-clean.
3. R20.3/R20.4 real-transport tests pass in the same commit that closes them.
4. R20.5 matrix is published and each row is backed by a test.
5. Every spin-out item is either closed or carries a dated owner decision.

## 6. Supersession

On closure, `10-mcp-protocol-and-fastmcp-parity.md` is annotated
"closure tracked by 20" and its unresolved-optional list is copied into the
matrix (R20.5). Spec 10 is not deleted; it remains the reference review.
