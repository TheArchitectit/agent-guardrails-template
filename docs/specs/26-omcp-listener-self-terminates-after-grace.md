# OpenSpec 26: OMCP HTTP Listener Self-Terminates After `shutdown::GRACE`

**Status:** Proposed — opened 2026-10-10 (spin-out from the spec 22 required-row
live run).

**Priority:** Medium — does not affect any compatibility verdict, but caps every
HTTP server instance at ~10 s and is a real defect in the serve loop.

**Purpose:** Record that `radical-mcp-server` stops accepting connections
`shutdown::GRACE` (10 s) after start **even when no SIGTERM/SIGINT is received**,
with cited source, and specify the fix.

## 1. Problem

During the spec 22 required-row run (2026-10-10, OMCP HEAD
`0a084de844254380789d1d6bf57f73bb8ba33191`) the OMCP server shut down its
listener ~10 s after every start, unconditionally:

```
INFO  radical_mcp_server: Starting MCP server on http://127.0.0.1:8081 ...
(warn) radical_mcp_server::shutdown: in-flight calls did not finish within 10s; stopping anyway
INFO  radical_mcp_server::shutdown: shutdown complete
```

No signal was sent and no client call was in flight. Every Inspector step had to
be run against a freshly started instance, and the server is unusable for a
long-lived session as built here.

### Root cause

`crates/server/src/shutdown.rs::serve_until_within` wraps the **entire**
`axum::serve(...).with_graceful_shutdown(announced)` future in
`tokio::time::timeout(grace, ...)`:

```rust
let served = tokio::time::timeout(
    grace,
    axum::serve(listener, app).with_graceful_shutdown(announced),
)
.await;
```

`with_graceful_shutdown` only begins draining once `announced` resolves, but the
outer `timeout` starts ticking **immediately** at `serve_until_within` entry. So
after `GRACE` (10 s) the serve future is cancelled whether or not a shutdown
signal ever arrived, and `serve_until_within` then logs the
"in-flight calls did not finish" warning and returns — the intended deadline is
meant to bound post-signal draining, not total server uptime.

## 2. Requirements

**R26.1** The `GRACE` deadline SHALL bound only the post-signal drain, not the
server's lifetime. A server with no shutdown signal SHALL serve indefinitely.

**R26.2** The fix SHALL preserve the bounded-drain contract (a wedged in-flight
call must not hang a supervisor past `GRACE` after a real signal).

**R26.3** No gate SHALL be weakened. A regression test SHALL assert that a
server with no signal keeps serving past `GRACE`.

## 3. Fix options

1. **Move the timeout inside the graceful-shutdown future (preferred).** Await the
   shutdown signal first, then race `axum::serve` against `timeout(GRACE, drain)`.
   Equivalently, keep `axum::serve(...).with_graceful_shutdown(signal)` unwrapped
   and apply the deadline only to the post-signal drain path. This is the
   smallest change that restores indefinite serving while keeping the bound.
2. **Raise/parameterise `GRACE`.** Does not fix the defect — the timeout still
   fires without a signal; it only moves the artificial deadline.
3. **Send a keep-alive/self-signal.** Treats the symptom, not the cause.

Recommended: **option 1.**

## 4. Acceptance criteria

1. A server started without a signal serves past `GRACE` (e.g. a request at
   `> 2 × GRACE` succeeds).
2. After a real SIGTERM/SIGINT, in-flight calls are still bounded by `GRACE`.
3. No gate is weakened; `serve_until_within`'s test-only parameter keeps working.

## 5. Evidence trail

- Source: `crates/server/src/shutdown.rs` (`serve_until_within`, the
  `tokio::time::timeout(grace, axum::serve(...))` wrap).
- `shutdown::GRACE = Duration::from_secs(10)`.
- Observed 2026-10-10 during the spec 22 required-row run; see
  `22-evidence-2026-10-10.md` (OMCP HEAD
  `0a084de844254380789d1d6bf57f73bb8ba33191`).
- Related: spec 17 R17.8 (graceful shutdown), spec 22 R22.1.
