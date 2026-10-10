# OpenSpec: MCP Client / Transport Compatibility Matrix

**Status:** Proposed — opened 2026-10-10 (spin-out from spec 20 / spec 10
§4.2.3, §5.4); closure block added 2026-10-10. NOT implemented: no supported
client has a passing end-to-end compatibility test yet, and the required-row
owner decision is still open.

**Priority:** High for the one required row (an official supported client); the
remaining rows are product choices.

**Purpose:** Publish a versioned matrix recording, per supported client and
transport, the endpoint path, transport version, authentication header, and
schema use — each backed by a live compatibility test — and list
intentionally-unsupported surfaces with rationale.

## 1. Problem

The AUDIT (§10-4.2.3) records client/transport compatibility tests as
**NOT_RUN**. The Pi bridge's documented SSE endpoint does not match this
server's Streamable HTTP `/mcp` endpoint, so compatibility cannot be inferred
from similar tool names (spec 10 §3). Until a supported client is proven to
initialize, authenticate, discover, invoke, and parse both success and error
results against the real transport, parity is unproven.

## 2. Requirements

**R22.1** The matrix SHALL name at least one official/supported MCP client and
prove end-to-end: initialize → authenticate → discover a tool → invoke →
parse success and error results.

### R22.1 required row — recorded 2026-10-10 (owner decision: OFFICIAL MCP
Inspector)

The owner decided 2026-10-10 that the **required** client/transport row is the
**official MCP Inspector** (`@modelcontextprotocol/inspector`) against the OMCP
server's **Streamable HTTP** transport. The row was driven live on this machine
(ucs03) on 2026-10-10; evidence doc:
`radicalopenmcpplatform docs/qa/mcp-inspector-compat-2026-10-10.md` (OMCP
commit `837e917d1845d54f187c06af2c844392f88409d9`).

| Field | Value |
|---|---|
| Client | **Official MCP Inspector** (`@modelcontextprotocol/inspector`; tested v2.10.1 and v1.0.2) |
| Transport | **Streamable HTTP** — `POST /mcp/stream` (MCP `2025-03-26`) |
| Auth header | none required on loopback (local run: `127.0.0.1:8081`); `RADICAL_API_KEY` / `Authorization: Bearer <key>` is required for non-loopback binds |
| Schema revision | OMCP HEAD at run time: `faad1328b3ebf06da0ab25717caba0a5e90097ca` |
| Evidence | OMCP `docs/qa/mcp-inspector-compat-2026-10-10.md`; raw Inspector step outputs recorded there |
| Raw step results | `initialize` **FAIL**, `tools/list` **FAIL**, `tools/call` success **FAIL**, `tools/call` error **FAIL** |
| Verdict | **FAILED — NOT SUPPORTED.** Must not be published as supported (R22.3). |

**Exact failing step:** `initialize`. The official Inspector requests MCP
protocol revision `2025-11-25`; OMCP speaks `2025-06-18`, `2025-03-26`,
`2024-11-05` (`crates/mcp/src/state.rs:271`) and refuses anything else
(`state.rs:286-315`). Both the v2 line and the deprecated v1 line request
`2025-11-25`, and the Inspector CLI exposes no flag to pin the protocol
revision (`--protocol-era legacy|auto|modern` all still request
`2025-11-25`/pinned `2026-07-28`). Because the handshake is refused, no later
method is ever reached. Raw Inspector output:

```
{"error":{"code":"error","message":"Unsupported MCP protocol version: 2025-11-25. This server speaks: 2025-06-18, 2025-03-26, 2024-11-05"}}
```

Control: a raw `curl` `initialize` with `protocolVersion: "2025-03-26"` against
the same endpoint succeeds, so the server and the Streamable HTTP endpoint are
healthy — the gap is the negotiated protocol revision. The mismatch is spun out
as **spec 25** (OMCP-side); see `25-omcp-protocol-revision-gap.md`.

**R22.2** Each row SHALL record endpoint path, transport version, auth header,
and schema revision.

**R22.3** Each supported client/transport pair SHALL have a same-commit
compatibility test; rows without a test SHALL NOT be listed as supported.

**R22.4** Intentionally-unsupported surfaces (prompts, resource templates,
notifications/progress, additional transports) SHALL be listed with rationale.

**R22.5** The matrix SHALL be versioned and its revision SHALL be published
alongside the server's protocol-version advertisement (spec 10 R10.5 / spec 24
in OMCP).

## 3. Acceptance criteria

1. One official client passes the full end-to-end sequence on the production
   transport, evidenced by a run id.
2. Every "supported" row has a test; every untested pair is marked unsupported
   or pending with a date.
3. The unsupported-surface list is complete and rationaled.

## 4. Open decision (owner) — RESOLVED 2026-10-10

The target client for the required row — e.g. the reference-project "Phase A:
Inspector Bridge" vs an official vendor client — is **not yet recorded**. This
must be an owner call before R22.1 closes.

**Resolved 2026-10-10:** the required row is the **official MCP Inspector**
(`@modelcontextprotocol/inspector`). The "Phase A: Inspector Bridge" candidate is
dropped as the required row. The row is recorded under R22.1 above, but it is
**FAILED — not supported**: the live run blocks at `initialize` on a protocol
revision the server refuses. R22.1 therefore does **not** close on this run; the
blocking mismatch is spin-out spec 24.

## 5. Closure status — 2026-10-10

No row is marked supported. Every candidate below is **pending** or
**unsupported**, because a row may not be listed as supported without a
same-commit passing test (R22.3) and none exists yet.

Candidate clients enumerated from the repository on 2026-10-10:

| Candidate | Source of truth in repo | Documented transport | Server transport | Status |
|---|---|---|---|---|
| Pi extension MCP bridge | `pi-extension/mcp-bridge/mcp-client.ts` | `SSEClientTransport` at `${url}/mcp/v1/sse` (line 43: `new URL(url.endsWith("/sse") ? url : `${url}/mcp/v1/sse`)`) | stateless Streamable HTTP `POST /mcp` | **Unsupported mismatch** — no SSE endpoint exists (`docs/platform-current-state.md` §6); compatibility cannot be inferred from tool-name overlap |
| VS Code extension | `ide/vscode-extension/src/utils/client.ts` (`serverUrl` default `http://localhost:8095`) | plain HTTP REST to the web service, not MCP JSON-RPC | web REST (separate auth path) | **Not an MCP client** — does not initialize/speak MCP; cannot be a compat-matrix row |
| JetBrains plugin | `ide/jetbrains-plugin/src/main/kotlin/com/guardrail/plugin/GuardrailService.kt` (`serverUrl` default `http://localhost:8095`) | OkHttp REST to the web service | web REST | **Not an MCP client** |
| Vim / Neovim plugins | `ide/vim-plugin`, `ide/neovim-plugin` | thin wrappers over the same REST server | web REST | **Not MCP clients** |
| MCP Inspector (official, `@modelcontextprotocol/inspector`) | external, not vendored | Streamable HTTP `/mcp` | stateless Streamable HTTP `/mcp` | **Pending test** — the strongest candidate for the required row; speaks the same Streamable HTTP transport the server exposes |
| pi-extension "Phase A: Inspector Bridge" | referenced in spec 22 §4 only; no source in this repo | — | — | **Pending owner decision** |

What a compatibility test would need (recorded so the row is testable, not
asserted):

1. An MCP client that speaks **stateless Streamable HTTP** to `POST /mcp`
   with `Accept: application/json, text/event-stream` and
   `Authorization: Bearer <credential>` — the same transport the server exposes
   and the same shape `streamable_http_authz_test.go` already drives.
2. A full sequence: `initialize` → capability exchange → `tools/list` discover
   a tool → `tools/call` success → `tools/call` rejected result → parse both.
3. A recorded row: endpoint path (`/mcp`), transport version (Streamable HTTP),
   auth header (`Authorization: Bearer`), schema revision (`cfg.SchemaVersion`).
4. A same-commit test artifact (e.g. a Go test driving the real endpoint with
   the client's request shape, or a scripted Node/Python client run captured
   with its output).

Blocking gaps already visible: the Pi bridge's SSE path does not exist on this
server, so the Pi bridge cannot be the required row as-is. The earlier missing
live `tools/list` sequence test has since been added (spec 23,
`TestStreamableHTTPListingSequence`), so discovery over the wire is now proven
for the Streamable HTTP transport a candidate row would use.

**Open owner decision (unchanged and required before R22.1 closes):** which
client is the *required* supported row — the official MCP Inspector, or the
reference-project "Phase A: Inspector Bridge", or a vendor client. Until that
is recorded, no row may be published as supported.

Intentionally-unsupported surfaces (R22.4), with rationale: MCP **prompts**
(no prompt registration in the inspected package; no product requirement);
**resource templates/subscriptions** (11 fixed resources only);
**notifications/progress and Tasks** (application task-attempt tracking is not
protocol Tasks support); **stdio and SSE transports** (server exposes only
stateless Streamable HTTP); no vendored generic client SDK (a server need not
reproduce FastMCP's client library). These are copied from spec 10 §3/§4.3 and
the `platform-current-state.md` baseline, and should be re-confirmed when a
row is added.
