# OpenSpec 25: Official MCP Inspector ⇄ OMCP Protocol-Revision Gap

**Status:** Proposed — opened 2026-10-10 (spin-out from the spec 22 required-row
live run).

**Priority:** High — blocks the single required compatibility row (spec 22
R22.1).

**Purpose:** Record that the official MCP Inspector cannot handshake with OMCP
because the Inspector offers MCP protocol revision `2025-11-25` and OMCP refuses
it, and specify the resolution options.

## 1. Problem

The required client/transport row (owner decision 2026-10-10) is the **official
MCP Inspector** against OMCP Streamable HTTP. A live run on ucs03 on 2026-10-10
(OMCP HEAD `faad1328b3ebf06da0ab25717caba0a5e90097ca`, evidence doc
`docs/qa/mcp-inspector-compat-2026-10-10.md`, OMCP commit
`837e917d1845d54f187c06af2c844392f88409d9`) shows **every** Inspector step fails
at `initialize`.

Raw Inspector error (identical for `initialize`, `tools/list`, and both
`tools/call` variants):

```
{"error":{"code":"error","message":"Unsupported MCP protocol version: 2025-11-25. This server speaks: 2025-06-18, 2025-03-26, 2024-11-05"}}
```

Server-side proof (`curl`, no Inspector — the endpoint itself is healthy):

```
POST /mcp/stream {"protocolVersion":"2025-11-25"} ->
{"jsonrpc":"2.0","error":{"code":-32602,"message":"Unsupported MCP protocol version: 2025-11-25. ...","data":{"error_type":"unsupported_protocol_version","requested":"2025-11-25","supported_protocol_versions":["2025-06-18","2025-03-26","2024-11-05"]}}}

POST /mcp/stream {"protocolVersion":"2025-03-26"} -> result (capabilities.tools, serverInfo radical-mcp 0.1.0)
```

### Root cause

- OMCP: `crates/mcp/src/state.rs:271` —
  `pub const SUPPORTED_PROTOCOL_VERSIONS: &[&str] = &["2025-06-18", "2025-03-26", "2024-11-05"];`
  and `state.rs:286-315` refuses any revision outside the set (deliberately: "a
  handshake that silently downgrades hides a real incompatibility").
- Inspector: the current v2 line (v2.10.1, `@modelcontextprotocol/*` core
  2.2.0) and the deprecated v1 line (1.0.2) both request `2025-11-25`. The
  Inspector CLI exposes **no flag** to pin the requested protocol revision;
  `--protocol-era legacy|auto|modern` all still request `2025-11-25` (or a
  pinned `2026-07-28` in `modern`, which also fails).

The gap is generation-independent and not a URL/transport bug.

## 2. Requirements

**R25.1** OMCP SHALL either (a) add the revision(s) the target Inspector
requests to `SUPPORTED_PROTOCOL_VERSIONS` once the server can actually speak the
surface that revision requires, or (b) record the row as **unsupported** for
that Inspector version with the exact failing step and date. No silent
downgrade; no weakening of the version-negotiation gate.

**R25.2** Adding a revision SHALL be paired with the capability gating the
revision implies (elicitation, structured tool output, `_meta`, `server/discover`
as applicable); a revision must not be advertised while only half-spoken.

**R25.3** The compatibility matrix (spec 22) SHALL track the Inspector's
negotiated revision so the row can flip to supported when the Inspector offers a
compatible revision (or OMCP adds it).

## 3. Resolution options

1. **OMCP adopts `2025-11-25`** once it implements the surface that revision
   requires, adding it to `SUPPORTED_PROTOCOL_VERSIONS` and capability-gating the
   new features. Highest value (unblocks the required row) but largest change.
2. **Pin the Inspector's protocol revision.** Not currently possible via the
   CLI; would require an Inspector-side change (out of this repo's control).
3. **Record as unsupported for now** and revisit when either side changes.
   Mirror the finding into spec 22 §5.

Recommended: option 3 now (row stays FAILED/unsupported), with option 1 as the
tracked remediation.

## 4. Acceptance criteria

1. Either the Inspector completes the full end-to-end sequence (row flips to
   supported with same-commit evidence), or spec 22 records the row as
   unsupported with the exact failing step.
2. No gate is weakened and no version outside `SUPPORTED_PROTOCOL_VERSIONS` is
   accepted by a silent downgrade.

## 5. Evidence trail

- OMCP evidence: `docs/qa/mcp-inspector-compat-2026-10-10.md` (OMCP commit
  `837e917d1845d54f187c06af2c844392f88409d9`).
- OMCP source: `crates/mcp/src/state.rs:271`, `:286-315`.
- Related: spec 22 R22.1 (required row FAILED), spec 23 (Streamable HTTP listing
  sequence).
