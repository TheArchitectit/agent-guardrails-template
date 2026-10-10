# OpenSpec: MCP Client / Transport Compatibility Matrix

**Status:** Proposed — opened 2026-10-10 (spin-out from spec 20 / spec 10
§4.2.3, §5.4). NOT implemented.

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

## 4. Open decision (owner)

The target client for the required row — e.g. the reference-project "Phase A:
Inspector Bridge" vs an official vendor client — is **not yet recorded**. This
must be an owner call before R22.1 closes.
