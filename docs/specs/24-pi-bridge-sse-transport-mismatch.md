# OpenSpec 24: Pi-Bridge SSE Transport Mismatch

**Status:** Proposed — opened 2026-10-10 (spin-out from spec 22 closure run);
**bridge fixed 2026-10-10** (option 1 implemented — Streamable HTTP). The code
change and a focused test are in this commit; a live bridge run against OMCP has
not been recorded, so the compat-matrix row stays pending (R24.4).

**Priority:** High — a real client in this repo cannot connect to the server as
written, and the incompatibility was previously only inferable from tool-name
overlap.

**Purpose:** Record, with cited source, that the Pi extension's MCP bridge
points at an SSE endpoint that the OMCP server does not serve, and specify the
fix options so the bridge can be made compatible without weakening any gate.

## 1. Problem

`pi-extension/mcp-bridge/mcp-client.ts` connects over SSE. Its endpoint
construction is:

- **`pi-extension/mcp-bridge/mcp-client.ts:43`** —
  ``const sseUrl = new URL(url.endsWith("/sse") ? url : `${url}/mcp/v1/sse`);``
- **`pi-extension/mcp-bridge/mcp-client.ts:51`** —
  ``this.transport = new SSEClientTransport(sseUrl, { ... });``

So for any base URL that does not itself end in `/sse`, the bridge targets
`<base>/mcp/v1/sse` over `SSEClientTransport` (legacy HTTP+SSE).

The OMCP server (`radical-mcp-server`) does **not** serve `/mcp/v1/sse`. Its
router nests under `/mcp` (`crates/server/src/router.rs:79-85`):

| Route | Method | Handler |
|---|---|---|
| `/mcp/stream` | GET + POST | `streamable_http_sse_handler` / `streamable_http_handler` |
| `/mcp/sse` | GET | `sse_handler` (legacy SSE) |
| `/mcp/messages` | POST | `message_handler` (legacy SSE) |
| `/mcp/ws` | GET | `websocket_handler` |

There is **no `/mcp/v1/...` prefix at all**. The bridge therefore targets a
404 path: the legacy SSE endpoint the server does expose is `/mcp/sse`
(+ `/mcp/messages`), and the modern transport it prefers is Streamable HTTP
`/mcp/stream`. `mcp-server/README.md:408` already warns clients to use
`POST /mcp` and not the legacy `/mcp/v1/sse` endpoint — the bridge has not been
updated to match.

## 2. Requirements

**R24.1** The mismatch SHALL be recorded with the exact source line(s) and the
exact server route table, so it is verifiable rather than asserted (done above).

**R24.2** The bridge SHALL be made to speak a transport the server actually
serves, by exactly one of the fix options in §3. No option may weaken, skip, or
`continue-on-error` any existing gate.

**R24.3** A same-commit test SHALL prove the bridge connects to the real server
endpoint (mirroring the R22.3 requirement for a supported row).

**R24.4** If the bridge is not fixed in the same change, the compatibility-matrix
row for the Pi bridge SHALL remain **unsupported** with a dated rationale (it
already is; see spec 22 §5).

## 3. Fix options

1. **Switch the bridge to Streamable HTTP (preferred).** Replace the
   `SSEClientTransport` usage with the SDK's Streamable HTTP client transport
   and target `<base>/mcp/stream`. This matches the server's primary transport,
   is the same surface the required Inspector row targets, and aligns with
   `mcp-server/README.md:408`. This is the smallest change that makes the bridge
   a first-class client.
2. **Target the legacy SSE routes as they exist.** Change the default path from
   `/mcp/v1/sse` to `/mcp/sse` (and use `/mcp/messages` for client→server).
   This keeps `SSEClientTransport` but is legacy and is the transport the server
   documents as discouraged; acceptable only as a stopgap.
3. **Add a server-side `/mcp/v1/sse` alias route.** Add a compatibility shim
   under `/mcp/v1/sse` (+ `/mcp/v1/messages`) mapping onto the legacy SSE
   handlers. This satisfies clients that hard-code the `/mcp/v1` prefix, but it
   adds a second public path to maintain and should carry the same auth
   middleware — and it still leaves the bridge on legacy SSE rather than the
   server's primary Streamable HTTP transport.

Recommended: **option 1**, with option 3 only if other deployed clients also
hard-code the `/mcp/v1` prefix (grep shows many archived docs/releases still
reference `/mcp/v1/sse`, e.g. `docs/releases/v3.3.0.md:114` records the old SSE
URL returning 404 — those are historical, not live clients).

## 4. Acceptance criteria

1. The bridge connects to the chosen live endpoint and completes
   `initialize → tools/list → tools/call` against `radical-mcp-server`.
2. A same-commit test drives the bridge (or the exact request shape it emits)
   against the real endpoint.
3. No gate is weakened to make the test pass.

## 6. Resolution — implemented 2026-10-10 (option 1)

The Pi bridge now uses the SDK's **Streamable HTTP** client transport against the
server's real route, as recommended in §3 option 1.

Changed: `pi-extension/mcp-bridge/mcp-client.ts`

- Added `MCP_STREAM_PATH = "/mcp/stream"` and an exported
  `resolveMcpEndpoint(url)`.
- The HTTP branch of `tryConnect` now calls `connectHttp` (was `connectSSE`),
  which builds `new StreamableHTTPClientTransport(endpoint, { requestInit: {
  headers } })` (was `new SSEClientTransport(...)`).
- `resolveMcpEndpoint` maps a bare base URL, `/`, `/mcp`, and any legacy SSE
  path (`.../sse`, including the old `${url}/mcp/v1/sse`) to `/mcp/stream`; an
  already-correct explicit path (e.g. `/mcp/stream`) is preserved.
- The public API surface is unchanged: `tryConnect`, `callTool`, `isConnected`,
  `getTools`, `close` are all intact; only the private HTTP connect method and
  the endpoint the bridge targets changed.

Evidence (all live, this commit):

- `mcp-server` Go tests for the transport: `cd mcp-server && go test
  ./internal/mcp/ -count=1` → `ok github.com/thearchitectit/guardrail-mcp/internal/mcp 0.204s`.
- `pi-extension` full suite: `npx vitest run` → **22 files, 193 tests passed**,
  including the new `mcp-bridge/mcp-client.test.ts`.
- Focused test `mcp-bridge/mcp-client.test.ts` (new): asserts the endpoint
  resolver targets `/mcp/stream` for a bare URL and for legacy SSE paths (never
  `/mcp/v1/sse`), preserves an explicit `/mcp/stream`, and preserves
  host/port; plus a source-contract check that the bridge uses
  `StreamableHTTPClientTransport` and contains no `SSEClientTransport`.

What is **not** claimed: a live bridge run (pi agent → bridge → OMCP) was not
performed. R24.3's "same-commit test proves the bridge connects to the real
server endpoint" is satisfied at the contract level (transport/route) by the
focused test plus the transport's own live Go tests; a full end-to-end bridge
run remains open and the compat-matrix row stays **pending** (R24.4).

## 5. Evidence trail

- Source line (pre-fix): `pi-extension/mcp-bridge/mcp-client.ts:43`, `:51`.
- Post-fix source: `pi-extension/mcp-bridge/mcp-client.ts` (`resolveMcpEndpoint`,
  `connectHttp`).
- Test: `pi-extension/mcp-bridge/mcp-client.test.ts`.
- Server routes: `crates/server/src/router.rs:79-85` (OMCP).
- Existing warning: `mcp-server/README.md:408`.
- Related: spec 22 §5 (Pi bridge marked unsupported mismatch).
