# OpenSpec: Live Tool/Resource Listing Sequence over the Production Transport

**Status:** Implemented — opened 2026-10-10. NEW. The live listing sequence is
committed as `mcp-server/internal/mcp/streamable_http_listing_test.go`:
`TestStreamableHTTPListingSequence`.

**Priority:** High — it is the one remaining gap in spec 20 R20.3 (and spec 10
§4.2.1): the real transport is exercised for `initialize` and `tools/call` and
`resources/read`, but there is **no live `tools/list` or `resources/list`
sequence test** in `mcp-server/internal/mcp`.

**Purpose:** Add a same-commit test that drives the production Streamable HTTP
transport (`POST /mcp`) through the full protocol sequence a real client uses:
`initialize` → capability exchange → `tools/list` → `resources/list` →
`tools/call` (success and rejection) → shutdown. Discovery must be proven live,
not by enumerating the registry in-process.

## 1. Problem

Spec 20 R20.3 requires CI to exercise the production transport for
`initialize`, capability exchange, **tool/resource listing**, one successful and
one rejected `tools/call`, and shutdown. On 2026-10-10 the `internal/mcp`
package has:

- `initialize`: `TestMCPEndpointBehindBearer` (real `POST /mcp`).
- `tools/call` allow + deny and `resources/read`:
  `TestStreamableHTTPToolsCallAuthorizesAndDenies`,
  `TestStreamableHTTPResourceReadOmitsSecrets`.
- **No `tools/list` test** and **no `resources/list` test** anywhere over the
  transport.

`grep -rn "tools/list\|resources/list" mcp-server/internal/mcp` returns nothing.
Discovery is therefore only proven in-process (the conformance gate in spec 21
enumerates the registry from source), never over the wire a client actually
uses. A client that discovers zero tools, or a capability exchange that omits
`tools`, would not be caught.

## 2. Requirements

**R23.1** CI SHALL drive the production Streamable HTTP endpoint through
`initialize` and assert the returned protocol version and capabilities
(including `tools` and `resources`).

**R23.2** The test SHALL call `tools/list` over the transport and assert the
returned tool count and names match the live registry (`toolList()`), not a
hard-coded list.

**R23.3** The test SHALL call `resources/list` over the transport and assert the
registered fixed resources are returned; if the server advertises no resource
capability the assertion SHALL say so explicitly rather than pass silently.

**R23.4** The test SHALL perform one successful and one rejected `tools/call`
over the same session/transport path, distinguishing a tool-level `IsError`
result from a transport/HTTP error.

**R23.5** The sequence SHALL run unauthenticated-denied first (401 before
dispatch) and then authenticated, reusing the `requireBearerWithRegistry`
wiring the package already uses, and SHALL NOT call handlers directly.

**R23.6** The test SHALL be hermetic: no database, validator, guardrails engine
or network beyond the `httptest` server.

## 3. Acceptance criteria

1. `tools/list` and `resources/list` are asserted over the real transport in a
   committed test with a captured pass run id.
2. Counts asserted against the live registry, not a literal.
3. The rejected `tools/call` returns a stable permission-denial result with no
   side effect, matching the existing `streamable_http_authz_test.go` pattern.

## 4. Dependencies and notes

- Closes the gap recorded in spec 20 R20.3 and referenced by spec 22 §5.
- Reuses `newFullPathServer` / `rpcPost` / `decodeToolResult` from
  `streamable_http_authz_test.go`; the only new code is the `initialize`
  capability assertions and the `tools/list` / `resources/list` calls.
- Not a substitute for the spec 21 gate: that gate checks schema↔handler
  agreement from source; this spec checks that the registry is *discoverable*
  over the wire.

## 5. Closure status — 2026-10-10

**Implemented.** `TestStreamableHTTPListingSequence` drives the real
`requireBearerWithRegistry`-wrapped Streamable HTTP endpoint and asserts, in
subtests: (1) `initialize` returns a protocol version and advertises both
`tools` and `resources` capabilities; (2) `tools/list` returns exactly the live
registry (`s.toolList()`), with every returned tool present in the registry and
advertising an object input schema; (3) `resources/list` returns the registered
fixed resources including `guardrail://config`; (4) `tools/call` succeeds with
a side effect and a rejected `tools/call` returns the stable permission denial
with zero side effect; (5) unauthenticated `tools/list` is rejected 401 before
dispatch.

Live run (2026-10-10, branch `forge/spec-program-wave0`):

```
$ go test ./internal/mcp/ -run 'TestStreamableHTTPListingSequence' -count=1 -v
=== RUN   TestStreamableHTTPListingSequence
    --- PASS: initialize_advertises_tool_and_resource_capabilities
    --- PASS: tools/list_matches_the_live_registry
    --- PASS: resources/list_returns_the_registered_resources
    --- PASS: tools/call_success_then_rejection_over_the_same_transport
    --- PASS: tools/list_unauthenticated_is_rejected_before_dispatch
--- PASS: TestStreamableHTTPListingSequence (0.01s)
PASS
ok  github.com/thearchitectit/guardrail-mcp/internal/mcp
```

Honest limits. The capability assertions check only presence of the `tools` and
`resources` keys, not their exact sub-capabilities. `resources/templates/list`
and `prompts/list` are intentionally not tested because the server registers
neither (spec 22 §5 records them as intentionally unsupported). The registry is
still the process-local `toolList()`, so this run enumerates 37 always-on tools
(conditional tools are absent unless their store is set on the test server);
the spec 21 gate covers the full 58 both-states inventory.
