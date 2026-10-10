# Spec 22 — Pi-bridge Row Evidence: pi-extension MCP bridge ⇄ OMCP (2026-10-10)

**Result:** the spec 22 **Pi-bridge row is BLOCKED** at `connect`: the bridge
module cannot resolve its SDK import, so it never reaches the transport. The
server and the Streamable HTTP transport are proven good in the same run by an
isolated control client that uses the SDK's real entrypoints. Spun the defect
into **spec 27** (`27-pi-bridge-sdk-root-import-unresolvable.md`).

This run does **not** flip the row to SUPPORTED. Per R22.3 a row without a
passing same-commit test is not listed as supported; the bridge is not
functional as shipped.

## Run environment

| Field | Value |
|---|---|
| Host | ucs03 |
| Date | 2026-10-10 |
| GR tree | `/tmp/gr-w`, branch `forge/spec-program-wave0`, base `db0aff34` |
| OMCP tree | `/tmp/rmp-w4`, branch `forge/spec-program-wave3-residues`, HEAD `274a3d652df757d9b4b000c105d997420eb8ec49` |
| OMCP build | `cargo build -p radical-mcp-server --release` (Finished in 12.54s; binary `target/release/radical-mcp-server`) |
| Server start | `MCP_TRANSPORT=http MCP_LISTEN_ADDR=127.0.0.1:8081 RADICAL_ROOT_DIR=/tmp/rmp-w4 ./target/release/radical-mcp-server` |
| Endpoint | `POST /mcp/stream` (Streamable HTTP; `stateless=true`) |
| Auth | none on loopback (`RADICAL_API_KEY` unset) |
| SDK used by control | `@modelcontextprotocol/sdk` 1.17.5 (installed `--no-save`, gitignored) |
| Runner | `tsx` 4 (bridge is TS; native Node 24 type-stripping rejects the module's `!` type assertions) |
| Node | v24.21.0 |

### Spec-26 note (which instance)

The OMCP listener self-terminates ~10 s after start even with no signal (spec
26). The bridge and control were therefore run **against a freshly started
instance** — server started, health polled until `{"status":"ok"}`, then the
client run immediately. Recorded method: **freshly-started instance** (not
restart-during-run).

## Raw bridge output (verbatim)

`node_modules/.bin/tsx live-bridge-run.ts` (drives the REAL module
`mcp-bridge/mcp-client.ts`), against the freshly started server:

```
=== connect (tryConnect) ===
endpoint: http://127.0.0.1:8081
tryConnect: false
isConnected: false
getTools: []
BLOCKER: bridge could not connect
BRIDGE_EXIT=2
server alive after: {"status":"ok"}
```

The bridge returns `false` and swallows the cause. Directly reproducing the
bridge's own module-top import (`diag-bridge.ts`) surfaces it:

```
--- REAL bridge module import ---
typeof client.tryConnect: function
root import FAILED: ERR_MODULE_NOT_FOUND - Cannot find module
  '/tmp/gr-w/pi-extension/node_modules/@modelcontextprotocol/sdk/dist/esm/index.js'
  imported from /tmp/gr-w/pi-extension/diag-bridge.ts
bridge tryConnect -> false | isConnected -> false | tools -> []
```

`mcp-client.ts` does `await import("@modelcontextprotocol/sdk")`; the published
package's `exports["."].import` points at `./dist/esm/index.js`, which is not
shipped. A clean scratch install of `1.17.5` ships `dist/esm/` with `cli.js`,
`inMemory.js`, `types.js`, `client/`, `server/`, `shared/`, `experimental/` —
**no root `index.js`**. Every SDK install on the host (`1.29.0`, `1.25.2`,
`1.17.5`) likewise has no root `dist/esm/index.js`. The bare `catch {}` in the
bridge then leaves `MCP_SDK = null`, so `tryConnect` returns `false` with no
diagnostic. The bridge's SDK import target — not the server, transport, or
endpoint — is the blocker.

## Isolated control (server + transport proven good)

`control-client.ts` uses the SDK's **real** entrypoints
(`@modelcontextprotocol/sdk/client/index.js`,
`.../client/streamableHttp.js`) — the same transport and endpoint the bridge
intends — so the failure is isolated to the bridge's import, not the server:

```
control initialize: OK
control tools: list_files, git_status, read_file, git_diff
control call success content types: text
control rejected call -> error: MCP error -32602: Invalid arguments
control done
```

Server log for the control run (verbatim tail):

```
INFO Streamable HTTP request: method=notifications/initialized
INFO Streamable HTTP request: method=tools/list
INFO Streamable HTTP request: method=tools/call
INFO ...execute{path="Cargo.toml"}: radical_tool_api::fs: Reading file: "/tmp/rmp-w4/Cargo.toml"
INFO Streamable HTTP request: method=tools/call
```

So `initialize` → `tools/list` (4 tools) → `tools/call` success (content type
`text`) → `tools/call` rejected (`-32602 Invalid arguments`) all work over
`POST /mcp/stream`. Had the bridge imported the same subpaths, it would have
connected; the transport fix from spec 24 is correct and the defect is purely
the root import.

## Why the row is BLOCKED, not SUPPORTED

R22.1 requires the client to initialize, discover, invoke, and parse both
success and error results **through the actual bridge code**. The actual bridge
never leaves `tryConnect` (returns `false`); no request reaches the server from
the bridge (the server log shows requests only from the control client). No
step passed through the bridge, so no verdict of SUPPORTED is possible. The
defect is recorded as spec 27; once fixed, the row can be re-driven.

## Verdict

**BLOCKED** — the Pi-bridge row stays **pending/unsupported**. Failing step:
`connect` (`tryConnect`), because `mcp-client.ts` imports
`@modelcontextprotocol/sdk` from a package root that no published version
exposes, so `MCP_SDK` is `null` and the bridge is permanently offline.
Server, endpoint (`POST /mcp/stream`), and Streamable HTTP transport verified
working in the same run by the control client. Spun into **spec 27**.
