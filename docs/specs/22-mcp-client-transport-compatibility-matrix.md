# OpenSpec: MCP Client / Transport Compatibility Matrix

**Status:** Proposed — opened 2026-10-10 (spin-out from spec 20 / spec 10
§4.2.3, §5.4); closure block added 2026-10-10; **required row PASSES
2026-10-10** (R22.1 SUPPORTED — see §R22.1 and `22-evidence-2026-10-10.md`).
The remaining candidate rows are product choices and stay pending/unsupported;
the required row is closed.

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
Inspector); **PASSES 2026-10-10**

The owner decided 2026-10-10 that the **required** client/transport row is the
**official MCP Inspector** (`@modelcontextprotocol/inspector`) against the OMCP
server's **Streamable HTTP** transport. The first live run (same day) **failed**
at `initialize` because the Inspector requested MCP revision `2025-11-25` and
OMCP refused it (spun out as spec 25). OMCP has since accepted `2025-11-25`, and
the row was re-driven live on ucs03 on 2026-10-10; it now passes every step.
Evidence: `22-evidence-2026-10-10.md` (raw Inspector step outputs).

| Field | Value |
|---|---|
| Client | **Official MCP Inspector** (`@modelcontextprotocol/inspector`) v2.10.1 (`@modelcontextprotocol/core` 2.2.0) |
| Transport | **Streamable HTTP** — `POST /mcp/stream`; MCP revision **`2025-11-25`** negotiated |
| Auth header | none required on loopback (run: `127.0.0.1:8081`); `RADICAL_API_KEY` / `Authorization: Bearer <key>` is required for non-loopback binds |
| Schema revision | OMCP HEAD at run time: `0a084de844254380789d1d6bf57f73bb8ba33191` |
| Evidence | GR `docs/specs/22-evidence-2026-10-10.md`; raw Inspector step outputs recorded there |
| Raw step results | `initialize` **PASS**, `tools/list` **PASS**, `tools/call` success **PASS**, `tools/call` rejected **PASS** (structured error) |
| Verdict | **SUPPORTED.** |

Raw `initialize` result (verbatim excerpt):

```json
{
  "serverInfo": { "name": "radical-mcp", "version": "0.1.0" },
  "protocolVersion": "2025-11-25",
  "capabilities": {
    "prompts": { "listChanged": false },
    "resources": { "subscribe": false, "listChanged": false },
    "tools": { "listChanged": false }
  }
}
```

Raw rejected-call result (verbatim):

```json
{"error":{"code":"error","message":"Invalid arguments"}}
```

Revision resolution: OMCP now lists `2025-11-25` first in
`SUPPORTED_PROTOCOL_VERSIONS` (`crates/mcp/src/state.rs`), so the Inspector's
requested revision is negotiated rather than refused. See
`25-omcp-protocol-revision-gap.md` (option 1 adopted) and
`22-evidence-2026-10-10.md`.

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
dropped as the required row. The row is recorded under R22.1 above and
**PASSES** as of 2026-10-10: the negotiated revision is `2025-11-25`, the
revision OMCP now accepts. R22.1 **closes** on this run; the previously blocking
mismatch was spin-out **spec 25**, now resolved (option 1 adopted).

## 5. Closure status — 2026-10-10

The **required row is SUPPORTED** (official MCP Inspector, see §R22.1 and
`22-evidence-2026-10-10.md`). The remaining candidate rows below are product
choices and stay **pending** or **unsupported**, because a row may not be listed
as supported without a same-commit passing test (R22.3).

Candidate clients enumerated from the repository on 2026-10-10:

| Candidate | Source of truth in repo | Documented transport | Server transport | Status |
|---|---|---|---|---|
| Pi extension MCP bridge | `pi-extension/mcp-bridge/mcp-client.ts` | **fixed 2026-10-10** — `StreamableHTTPClientTransport` at `/mcp/stream` (was `SSEClientTransport` at `${url}/mcp/v1/sse`, spec 24); SDK loaded via real subpath entrypoints (`client/index.js`, `client/streamableHttp.js`, `client/stdio.js`), not the package root (spec 27) | stateless Streamable HTTP `POST /mcp/stream` | **SUPPORTED (live)** — re-driven 2026-10-10 after the spec 27 fix: the real bridge module `tryConnect` → `true`, discovered 4 tools, `tools/call` success parsed, `tools/call` rejected parsed (structured `-32602`). Earlier same-day live run was **BLOCKED** by the SDK root-import defect (spec 27, now resolved). Raw output: §6 below + `22-evidence-pi-bridge-2026-10-10.md` |
| VS Code extension | `ide/vscode-extension/src/utils/client.ts` (`serverUrl` default `http://localhost:8095`) | plain HTTP REST to the web service, not MCP JSON-RPC | web REST (separate auth path) | **Not an MCP client** — does not initialize/speak MCP; cannot be a compat-matrix row |
| JetBrains plugin | `ide/jetbrains-plugin/src/main/kotlin/com/guardrail/plugin/GuardrailService.kt` (`serverUrl` default `http://localhost:8095`) | OkHttp REST to the web service | web REST | **Not an MCP client** |
| Vim / Neovim plugins | `ide/vim-plugin`, `ide/neovim-plugin` | thin wrappers over the same REST server | web REST | **Not MCP clients** |
| MCP Inspector (official, `@modelcontextprotocol/inspector`) | external, not vendored | Streamable HTTP `/mcp/stream` | stateless Streamable HTTP `/mcp/stream` | **SUPPORTED (required row)** — passed end-to-end 2026-10-10; see §R22.1 and `22-evidence-2026-10-10.md` |
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

The earlier missing live `tools/list` sequence test has since been added (spec 23,
`TestStreamableHTTPListingSequence`), so discovery over the wire is now proven
for the Streamable HTTP transport the required row uses. The Pi bridge SSE path
mismatch is resolved in code by spec 24 (Streamable HTTP `/mcp/stream`).

**Owner decision — closed 2026-10-10:** the required supported row is the
official MCP Inspector, and it passes (§R22.1).

Intentionally-unsupported surfaces (R22.4), with rationale: MCP **prompts**
(no prompt registration in the inspected package; no product requirement);
**resource templates/subscriptions** (11 fixed resources only);
**notifications/progress and Tasks** (application task-attempt tracking is not
protocol Tasks support); **stdio and SSE transports** (server exposes only
stateless Streamable HTTP); no vendored generic client SDK (a server need not
reproduce FastMCP's client library). These are copied from spec 10 §3/§4.3 and
the `platform-current-state.md` baseline, and should be re-confirmed when a
row is added.

## 6. Pi-bridge row — live run 2026-10-10 (SUPPORTED after spec 27 fix)

**First run (BLOCKED).** A live run of the real bridge module against a running
OMCP on 2026-10-10 reached `connect` and stopped: `MCPClient.tryConnect(<url>)`
returned `false` and no request reached the server. Cause: `mcp-client.ts`
imported `@modelcontextprotocol/sdk` from its **package root**, but the
published package ships no root `dist/esm/index.js`, so the optional-dependency
import threw and the bare `catch {}` left the SDK `null`. The same run's control
client (SDK subpath entrypoints) passed `initialize` → `tools/list` → success
call → rejected call over `POST /mcp/stream`, proving the row's server/transport
legs were good and isolating the failure to the bridge import. Spun into
**spec 27**.

**Re-run (SUPPORTED).** Spec 27 fixed the bridge to import the SDK's real
subpath entrypoints (`client/index.js`, `client/streamableHttp.js`,
`client/stdio.js`) and to report — not swallow — an unexpected subpath
resolution error. The real bridge module was then re-driven live
(2026-10-10, ucs03) against a freshly started OMCP (`MCP_TRANSPORT=http`,
`MCP_LISTEN_ADDR=127.0.0.1:8081`, `RADICAL_ROOT_DIR=/tmp/rmp-w4`) via
`pi-extension/live-bridge-run.ts`. Raw output:

```
=== connect (tryConnect) ===
endpoint: http://127.0.0.1:8081
tryConnect: true
isConnected: true
getTools: ["git_diff","git_status","list_files","read_file"]

=== tools/call success: read_file {path: Cargo.toml} ===
{
  "content": [
    {
      "type": "text",
      "text": "{\"content\":\"[workspace]\\nmembers = [\n...\"…\",\"path\":\"/tmp/rmp-w4/Cargo.toml\"}"
    }
  ]
}

=== tools/call rejected: read_file {} (missing required path) ===
{
  "error": "MCP call failed: MCP error -32602: Invalid arguments"
}

STEP RESULTS: success=true rejected=true
```

Every step ran through the actual bridge code (`MCPClient.tryConnect` →
`connectHttp` → `resolveMcpEndpoint` → `StreamableHTTPClientTransport` →
`Client.connect`/`listTools`/`callTool`). Verdict: **SUPPORTED (live)**, R22.3
satisfied. Full raw output: `22-evidence-pi-bridge-2026-10-10.md`.

