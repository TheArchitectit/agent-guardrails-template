# OpenSpec 27: Pi-bridge SDK Root Import Cannot Resolve (`@modelcontextprotocol/sdk`)

**Status:** Resolved — 2026-10-10. Fix applied (option 1: import the SDK's
real subpath entrypoints) and live-verified; the spec 22 Pi-bridge row is now
**SUPPORTED (live)**. Opened 2026-10-10 (spin-out from the spec 22 Pi-bridge
live run).

**Priority:** High — the Pi bridge is permanently unavailable as shipped; the
Pi-bridge row in spec 22 cannot reach a supported verdict until this is fixed.

**Purpose:** Record that `pi-extension/mcp-bridge/mcp-client.ts` imports the MCP
SDK from its **package root**, which no published version of
`@modelcontextprotocol/sdk` exposes, so the optional-dependency import always
throws and the bridge is permanently offline — with cited source and a live run
that proves the server/transport are fine.

## 1. Problem

The bridge loads the SDK in a top-level try/catch:

```ts
let MCP_SDK: typeof import("@modelcontextprotocol/sdk") | null = null;
try {
  MCP_SDK = await import("@modelcontextprotocol/sdk");
} catch {
  // @modelcontextprotocol/sdk is an optional dependency
  // MCP bridge is permanently unavailable when not installed
}
```

The import target is the **package root**. `@modelcontextprotocol/sdk`'s
`exports` map points `.` at `./dist/esm/index.js`, but the published package does
**not ship that file** — it ships only subpath entrypoints (`client`,
`client/streamableHttp`, `server`, `server/...`). A clean `npm install` +
root `import("@modelcontextprotocol/sdk")` therefore fails:

```
ERR_MODULE_NOT_FOUND: Cannot find module
  '.../node_modules/@modelcontextprotocol/sdk/dist/esm/index.js'
```

Because the failure is caught by the bare `catch {}`, the bridge swallows it:
`MCP_SDK` stays `null`, `tryConnect()` returns `false`, and `callTool()` returns
the generic "server not connected" string. Nothing surfaces the real cause, and
the fallback comment ("permanently unavailable when not installed") is
misleading — the SDK **is** installed; the import target is wrong.

Verified 2026-10-10 on ucs03 (live run, encode as evidence doc):
- A clean scratch install of `@modelcontextprotocol/sdk@1.17.5` has
  `dist/esm/` with `cli.js`, `inMemory.js`, `types.js`, `client/`, `server/`,
  `shared/`, `experimental/` — **no root `index.js`**.
- Every SDK install present on the host (`1.29.0` in openclaw/kanban, `1.25.2`
  in octofriend, `1.17.5` scratch) lacks a root `dist/esm/index.js`.
- The real bridge module, driven live against a running OMCP, reports
  `tryConnect -> false | isConnected -> false | tools -> []`, and the direct
  reproduction of its own import fails with the `ERR_MODULE_NOT_FOUND` above.

## 2. Requirements

**R27.1** The bridge SHALL import the MCP SDK from entrypoints that the
published package actually exposes, not the package root
(e.g. `@modelcontextprotocol/sdk/client/index.js` for `Client` and
`@modelcontextprotocol/sdk/client/streamableHttp.js` for
`StreamableHTTPClientTransport`, and `.../client/stdio.js` for
`StdioClientTransport`).

**R27.2** The SDK-unavailable fallback SHALL NOT swallow an unexpected
resolution/host error as "SDK not installed"; a failed import SHALL be
distinguishable from an absent optional dependency (log/report the cause so the
bridge is diagnosable).

**R27.3** A same-commit test SHALL assert the bridge can resolve its SDK
entrypoints and construct its transports when the SDK is installed.

**R27.4** No gate SHALL be weakened, and the transport fix from spec 24
(Streamable HTTP `/mcp/stream`) SHALL be preserved.

## 3. Fix options

1. **Import the real subpaths (preferred).** Replace
   `import("@modelcontextprotocol/sdk")` with the three documented subpath
   imports (`client/index.js`, `client/streamableHttp.js`, `client/stdio.js`).
   This matches how the SDK is designed to be consumed and how the MCP Inspector
   and other clients import it.
2. **Keep the root import and vendor/alias a root shim.** Adds a build step and
   diverges from the SDK's own entrypoint contract; not recommended.
3. **Drop the bridge.** Not warranted — the transport is correct (spec 24) and
   the server side is proven (this run's control client).

Recommended: **option 1.** — **adopted 2026-10-10.**

## 4. Acceptance criteria

1. With the SDK installed, `MCPClient.tryConnect(<OMCP URL>)` returns `true` and
   discovers tools over `POST /mcp/stream`.
2. A full sequence (initialize → tools/list → tools/call success → tools/call
   rejected) runs through the real bridge and both results parse.
3. A failed SDK import is reported with its cause, not silently mapped to
   "not connected".
4. The spec 22 Pi-bridge row can then be re-driven to a verdict.

## 5. Resolution (2026-10-10)

Implemented option 1 in `pi-extension/mcp-bridge/mcp-client.ts`:

- The top-level loader now imports the SDK's real subpath entrypoints —
  `@modelcontextprotocol/sdk/client/index.js` (`Client`),
  `.../client/streamableHttp.js` (`StreamableHTTPClientTransport`), and
  `.../client/stdio.js` (`StdioClientTransport`) — and never the package root
  (R27.1).
- The catch now distinguishes an **absent optional dependency**
  (`ERR_MODULE_NOT_FOUND` with `Cannot find package '@modelcontextprotocol/sdk'`)
  from an unexpected failure to load a subpath that should exist. Only the
  former degrades quietly; the latter is recorded and reported via
  `getMcpSdkLoadError()` (R27.2).
- `pi-extension/mcp-bridge/mcp-client.test.ts` asserts the bridge does not
  import the package root, does import the three subpaths, resolves the SDK on
  this install (no load error), and maps a bare URL to `/mcp/stream` (R27.3).
- No gate was weakened and the spec 24 Streamable HTTP `/mcp/stream` transport
  is preserved (R27.4).

**Live verification (R27 acceptance).** Real bridge module driven against a
freshly started OMCP (`MCP_TRANSPORT=http`, `MCP_LISTEN_ADDR=127.0.0.1:8081`,
`RADICAL_ROOT_DIR=/tmp/rmp-w4`) via `pi-extension/live-bridge-run.ts`:

```
=== connect (tryConnect) ===
tryConnect: true
isConnected: true
getTools: ["git_diff","git_status","list_files","read_file"]

=== tools/call success: read_file {path: Cargo.toml} ===
{ "content": [ { "type": "text", "text": "{\"content\":\"[workspace]…", "path": "/tmp/rmp-w4/Cargo.toml" } ] }

=== tools/call rejected: read_file {} (missing required path) ===
{ "error": "MCP call failed: MCP error -32602: Invalid arguments" }

STEP RESULTS: success=true rejected=true
```

Acceptance 1–4 all pass: `tryConnect` returns `true` and discovers tools over
`POST /mcp/stream`; the full sequence (initialize → tools/list → success →
rejected) parses; a failed subpath load is now reported rather than silently
mapped to "not connected"; and the spec 22 Pi-bridge row is re-driven to
**SUPPORTED (live)**.

## 6. Evidence trail

- Source: `pi-extension/mcp-bridge/mcp-client.ts` (`let MCP_SDK` /
  `await import("@modelcontextprotocol/sdk")`; `connectHttp`,
  `connectStdio`, `performConnection`).
- Live run 2026-10-10 (ucs03, OMCP HEAD
  `274a3d652df757d9b4b000c105d997420eb8ec49`, server
  `radical-mcp-server --release`, `MCP_LISTEN_ADDR=127.0.0.1:8081`):
  `docs/specs/22-evidence-pi-bridge-2026-10-10.md`.
- Related: spec 22 R22.3 (Pi-bridge row), spec 24 (Pi-bridge transport fix).
- Non-blocking companion defect: spec 26 (listener self-terminates after
  `shutdown::GRACE`), which is why the bridge was run against a freshly started
  instance.
