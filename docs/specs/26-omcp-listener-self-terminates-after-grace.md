# OpenSpec 26: OMCP HTTP Listener Self-Terminates After `shutdown::GRACE`

**Status:** CLOSED — 2026-10-14. Opened 2026-10-10 (spin-out from the spec 22 required-row
live run). **Option 1 was implemented the same day** in OMCP commit
`ca1db65` ("fix(server): arm GRACE only after shutdown signal, not at
serve-start (spec 26)", 2026-10-10) — an ancestor of `master@eec3d15` — with the
regression test `the_listener_stays_up_past_grace_when_no_signal_arrives`;
the GR spec doc had not recorded the closure until now.

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

## 4. Acceptance criteria — VERIFIED 2026-10-14

1. **MET (unit + live).** The regression test
   `the_listener_stays_up_past_grace_when_no_signal_arrives` (in
   `shutdown.test.rs`) passes at `master@eec3d15` (7/7 shutdown tests ok), and a
   live run on ucs03 (2026-10-14, `radical-mcp-server` debug build of the tip,
   `MCP_LISTEN_ADDR=127.0.0.1:8093`) answered `initialize` at t≈4 s and
   `tools/list` at t≈41 s and t≈50 s — serving ~5× past `GRACE = 10 s` with no
   signal (server log: requests logged, no shutdown line until SIGTERM).
2. **MET.** A real SIGTERM during the same live session drained cleanly:
   `shutdown signal received; finishing in-flight calls` → the in-flight
   `tools/call` completed → `stopped 0 plugin(s)` → `shutdown complete`.
   The unit tests `a_signal_stops_the_server_within_grace` and
   `a_call_that_outlives_the_deadline_does_not_hang_the_process` cover the
   bound paths.
3. **MET.** No gate touched; `serve_until_within`'s test-only parameter kept
   working (`the_deadline_is_a_constant_rather_than_configuration` and the
   signal test both pass, using the short stand-in duration).

## 5. Evidence trail

- Source: `crates/server/src/shutdown.rs` (`serve_until_within`, the
  `tokio::time::timeout(grace, axum::serve(...))` wrap).
- `shutdown::GRACE = Duration::from_secs(10)`.
- Observed 2026-10-10 during the spec 22 required-row run; see
  `22-evidence-2026-10-10.md` (OMCP HEAD
  `0a084de844254380789d1d6bf57f73bb8ba33191`).
- **Fix + closure evidence 2026-10-14:** OMCP `ca1db65` (ancestor of
  `master@eec3d15`, same-commit green ci + security-regression 04:20Z);
  `cargo test -p radical-mcp-server shutdown` at the tip → 7 passed / 0 failed;
  live serve-past-GRACE + SIGTERM drain as recorded in Sec. 4.
- Related: spec 17 R17.8 (graceful shutdown), spec 22 R22.1.
