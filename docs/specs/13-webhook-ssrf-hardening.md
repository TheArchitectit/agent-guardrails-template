# OpenSpec: Webhook Delivery SSRF Hardening

**Status:** Proposed. Describes intended behavior; current delivery is not
protected against DNS rebinding or redirect-based SSRF.
**Owner:** Project maintainer.
**Date:** 2026-10-04.
**Roadmap:** Phase 0, work package 07
([`09-system-roadmap-and-phase-gates.md`](09-system-roadmap-and-phase-gates.md)).
**Related:** [`11-authorization-scopes-and-roles.md`](11-authorization-scopes-and-roles.md)
(authorization to manage webhook configuration),
[`12-web-exposure-boundary.md`](12-web-exposure-boundary.md)
(deployment network boundary).

## 1. Problem and goal

`validateWebhookURL` in `mcp-server/internal/mcp/tools_notifications.go`
checks scheme, hostname, and resolved addresses when a webhook is configured.
The dispatcher later uses `http.Client.Do` for event deliveries and test
 deliveries. The standard transport resolves the hostname again at connection
time, and the client follows redirects by default. An attacker who controls a
DNS name can therefore change its answer after configuration, or redirect a
permitted public URL to a private target. Configuration-time validation does
not constrain the actual socket destination.

The goal is to make every outbound webhook request safe at the point of
connection, including retries and test events. DNS results must be checked and
pinned for each new connection; every redirect must either be rejected or
independently subjected to the same policy. Any uncertainty or resolution
failure denies the request. No webhook delivery may reach loopback, private,
link-local, special-purpose, or otherwise disallowed destinations.

This spec covers the outbound network boundary, not webhook authorization,
secret storage, queue durability, or general egress policy.

## 2. Current observed behavior (audit snapshot, 2026-10-04)

Re-verify before implementation.

- `validateWebhookURL` allows `http` and `https`, requires a host, rejects
  `localhost` and `.localhost`, and checks literal IPs or all IPs returned by
  `net.LookupIP` at configuration time. It rejects loopback, private,
  unspecified, and selected link-local/multicast addresses. It does not pin
  those addresses for delivery and is outside the notifications package.
- `notifications.NewDispatcher` creates a default `http.Client` with a
  10-second timeout and no `CheckRedirect`, custom dialer, or destination
  policy.
- `Dispatcher.doRequest` and `Dispatcher.SendTestEvent` both call
  `d.client.Do`. The normal delivery path retries up to three times. The test
  delivery path is separate, so fixing only `doRequest` would leave a bypass.
- Webhook creation and update call the URL validation function, but the stored
  URL can also predate that check or be changed outside these handlers.
- Redirect response bodies are not drained/closed by the caller-controlled
  logic because redirects are followed inside `http.Client.Do`.
- Delivery records store a truncated response body; failed delivery logs
  include the configured URL.

The current validation reduces trivial abuse but is not an SSRF-safe egress
boundary.

## 3. Normative delivery policy

### 3.1 URL parsing and canonical checks

Before every delivery attempt, parse the configured URL and require:

1. Scheme is exactly `https` by default. Plain HTTP is denied unless an
   explicit deployment policy allows it for a reviewed internal destination
   (§3.5). HTTPS must be used for public endpoints so payloads and signatures
   are not exposed in transit.
2. A non-empty DNS hostname or IP literal is present. Reject userinfo,
   fragments, opaque URLs, control characters, invalid ports, IPv6 zone
   identifiers, and ambiguous/noncanonical address spellings.
3. The effective port is 443 for HTTPS, or the configured permitted port for
   the narrowly scoped internal exception. Arbitrary ports are denied unless
   explicitly allowed by owner policy.
4. Host comparison and IP parsing use canonical forms; trailing-dot DNS names
   are handled consistently and cannot evade hostname policy.

Validation applies to both newly configured and already-stored URLs. A
configuration-time check may provide fast feedback, but it is not authorization
to connect and cannot replace the checks below.

### 3.2 DNS resolution and address policy

For each new TCP connection:

1. Resolve the hostname using the configured resolver and obtain the complete
   answer set. Resolution errors, empty sets, malformed addresses, or partial
   uncertainty fail closed.
2. Normalize IPv4, IPv6, IPv4-mapped IPv6, and any address representation
   before classification. Reject the entire answer set if **any** address is
   disallowed; do not select only a public answer from a mixed public/private
   response.
3. Deny loopback, unspecified, private, link-local, multicast, broadcast,
   reserved, documentation, benchmarking, carrier-grade NAT, IPv4-mapped
   disallowed addresses, IPv6 unique-local, IPv6 link-local, and other
   IANA-special-purpose ranges. Maintain the range policy as tested data, not a
   few ad hoc `net.IP` predicates.
4. Deny cloud metadata and platform control-plane destinations, including
   well-known metadata IPs and names. Metadata protection supplements address
   classification; it does not replace it.
5. Select only from the validated answer set and pin the selected IP to the
   actual socket connection. Preserve the original hostname for HTTP `Host`,
   TLS SNI, and certificate verification. Do not perform a second unvalidated
   hostname lookup between validation and connect.
6. Repeat resolution and validation for every fresh connection, including
   retries and connections after keep-alive expiry. Reuse is permitted only for
   a connection whose pinned peer was validated under the current policy.

IP literals undergo the same classification and connect-time checks without
DNS. IPv4-mapped IPv6 addresses are classified by their embedded IPv4 value.
Transition/tunneling ranges that can route to otherwise blocked IPv4 targets
(e.g. NAT64 or 6to4) are denied unless the implementation can prove and enforce
the embedded destination policy.

### 3.3 Redirect policy

The default policy is **no redirects**. Any 3xx response is recorded as a
failed delivery and is not followed. This avoids forwarding signed payloads to
an unintended host and eliminates redirect chains as an SSRF bypass.

If a future owner-approved policy permits redirects, each hop must:

- be limited to a small fixed maximum (at most three);
- be re-parsed and pass all URL, scheme, port, DNS, address, and TLS checks;
- be independently pinned at connection time;
- never downgrade HTTPS to HTTP;
- never forward authorization, cookies, or webhook-signature headers across
  origins; and
- stop on malformed `Location`, loops, or any denied destination.

No redirect exception is in scope for this spec; enabling redirects requires an
amendment and dedicated tests.

### 3.4 Network and response bounds

- Use request context deadlines and a bounded client timeout. Bound DNS lookup,
  connect, TLS handshake, response-header wait, and total delivery time.
- Bound response-body reads and close bodies on every response, including
  rejected redirects and errors. Never buffer an unbounded response.
- Retries must be bounded, respect cancellation/deadline, and re-run the
  destination policy for each new connection. A retry does not turn a denied
  address into a transient success.
- Do not log URL userinfo, query secrets, full response bodies, or webhook
  secrets. Persist only bounded, sanitized response data. Delivery errors must
  distinguish policy denial from transport failure without exposing sensitive
  URL components.
- Concurrency and retries remain bounded per webhook and globally so a large
  configuration cannot create unbounded outbound work.

### 3.5 Internal webhook exception

The default is no private-network destination exception. If deployments require
webhooks to internal services, use an explicit allowlist with all of the
following properties:

- It is configured by an operator, not supplied by a webhook creator.
- Entries bind an exact canonical hostname to specific CIDRs and ports; broad
  private CIDRs, suffix wildcards, and implicit cluster/service discovery are
  prohibited.
- DNS answers must remain within the configured CIDRs and the connected peer
  must match the validated answer. Public addresses do not automatically
  inherit the exception.
- The exception is disabled by default, auditable, documented per deployment,
  and covered by positive and negative tests.
- It cannot include loopback, unspecified, link-local, multicast, broadcast,
  metadata, or control-plane destinations.

The owner must decide whether an internal exception is needed before
implementation. Without that decision, implement the no-private-destination
policy and do not add a permissive escape hatch.

## 4. Application and lifecycle requirements

1. Apply the same safe request-construction and connection policy to
   `Dispatcher.doRequest` and `Dispatcher.SendTestEvent`; preferably both use
   one delivery function.
2. Apply checks to create and update for fast user feedback, but also enforce
   them at every delivery so old rows, direct database changes, and DNS changes
   cannot bypass policy.
3. The delivery boundary belongs in the notifications/egress layer, not only
   in the MCP tool package. No caller may reach the raw default client for a
   webhook URL.
4. On policy denial, do not attempt the socket connection, do not follow a
   redirect, and do not retry as though it were a transient network failure.
   Record a failed delivery with a sanitized reason.
5. Keep webhook identifiers, event types, status code, timestamp, and bounded
   outcome evidence available to operators. Do not record full signed payloads
   or secrets as delivery diagnostics.
6. A blocked destination must produce a failure state, never success or an
   ambiguous “delivered” result.

## 5. Acceptance criteria

Each criterion is executable in CI with deterministic resolver/transport
fixtures; no public DNS or external network access is required.

1. **Configuration-time feedback.** URLs with unsupported scheme, missing host,
   userinfo, invalid/forbidden port, zone identifier, localhost, or a literal
   prohibited IP are rejected. *Expected:* creation/update returns a clear
   validation error. *Failure:* URL is stored as enabled.
2. **Stored URL revalidation.** Seed an otherwise valid webhook row, then
   change its DNS answer to a prohibited range before delivery. *Expected:*
   delivery is denied and the test server is never contacted. *Failure:* event
   payload arrives.
3. **DNS rebinding/pinning.** Use a controlled resolver that returns a public
   address at initial validation and a private address at connection time.
   *Expected:* connection policy checks the answer used for the connection and
   blocks it. *Failure:* the private listener receives a request.
4. **Mixed answer set.** Resolver returns one public and one prohibited address
   in either order. *Expected:* entire destination is denied. *Failure:* client
   connects to the public member while accepting the mixed set.
5. **Address matrix.** Table-driven tests cover IPv4/IPv6 loopback, unspecified,
   private, link-local, multicast, broadcast/reserved/special-purpose, mapped
   IPv4, unique-local, and transition addresses. *Expected:* every prohibited
   class is denied and representative public addresses pass. *Failure:* any
   prohibited class can connect.
6. **Literal IP enforcement.** Test prohibited IPv4 and IPv6 literals,
   including mapped forms and zone identifiers. *Expected:* denied before
   dialing. *Failure:* dial callback observes an attempt.
7. **No redirect.** A public test endpoint responds with 301/302 to a private
   listener. *Expected:* the first response is recorded as failure; private
   listener receives zero requests. *Failure:* redirect is followed.
8. **Both delivery paths.** Run equivalent blocked-target and redirect tests
   through event delivery and `SendTestEvent`. *Expected:* identical policy
   enforcement. *Failure:* either path reaches the target or reports success.
9. **Redirect headers.** If redirects are ever enabled by an amended spec,
   cross-origin hops do not receive signatures or credentials. In this spec’s
   default no-redirect policy, assert no second request is issued.
10. **Cancellation and bounds.** A stalled resolver, connect, TLS handshake, or
    response body respects the request deadline; an oversized response is
    truncated to the configured cap and closed. *Failure:* goroutine/socket
    remains active past the deadline or response memory is unbounded.
11. **Retry behavior.** Transient transport failures retry only within the
    configured cap; policy denials do not retry; every fresh dial revalidates
    and pins. *Failure:* a denied destination is retried or bypassed.
12. **Sanitized evidence.** Delivery logs/records contain no URL userinfo,
    secret query values, HMAC secret, or unbounded response data. *Failure:*
    sentinel secret appears in logs or stored diagnostics.
13. **No external network dependency.** The package test suite passes with DNS
    and HTTP entirely stubbed. *Failure:* SSRF policy tests require internet
    access or real DNS.

## 6. Migration and rollout

1. Implement a single egress policy/transport seam with an injectable resolver
   and dialer. Keep it internal to webhook delivery; do not expose a generic
   arbitrary-URL fetch tool.
2. Enforce at delivery first, including the test-event path. Treat existing
   webhook rows that violate the selected scheme/destination policy as disabled
   for delivery and record a clear denial; do not delete or silently rewrite
   them.
3. Retain configuration-time validation and align its result with the delivery
   policy. Report how many stored configurations need operator remediation
   without logging their full URLs.
4. Roll out the default HTTPS-only and no-redirect policy with a migration note.
   Existing HTTP webhooks require endpoint upgrade or an explicitly approved
   narrow exception; they must not be silently upgraded/downgraded.
5. Add regression tests before changing the old validation behavior. Verify
   both normal delivery and test delivery before enabling webhook delivery in a
   release.
6. Update the platform current-state document and webhook operator docs only
   after the tests demonstrate enforcement through the real delivery paths.

## 7. Open decisions and first-implementation defaults

For the first deployment, select **no internal webhook exceptions**, HTTPS
only, port 443 only, and no redirects. Keep denied configurations for operator
review but never dial them. The implementation must pin a reviewed IANA IPv4/
IPv6 special-purpose registry snapshot by version and digest, with a test
fixture for every denied range; updating it is a reviewed policy change, not
a live DNS fallback. These are proposed implementation choices, not shipped
delivery-time enforcement. A deployment requiring an internal destination must
stop and amend §3.5 before allowing it.

1. Is any internal webhook required? If yes, name the exact deployment use case
   and approve the exact-host/CIDR/port allowlist model in §3.5. Default: no.
2. Is HTTPS-only acceptable for all existing deployments? Default: yes; plain
   HTTP is denied.
3. Are nonstandard public HTTPS ports required? Default: no; allow only 443.
4. Which maintained IANA special-purpose address registry snapshot will the
   implementation use, and how will updates be reviewed and tested?
5. Should policy denials disable the webhook immediately or leave it enabled
   while recording each failed attempt? Default: keep configuration but never
   send to the denied destination; expose a clear operational state.

## 8. Non-goals

- Role/scope authorization to configure or delete webhooks — Spec 11.
- Public route/CORS/proxy exposure policy — Spec 12.
- General outbound egress firewall, arbitrary URL fetching, or DNS resolver
  infrastructure beyond the webhook egress boundary.
- Webhook receiver authentication or replay-window design.
- CI phase-gate wiring, specified in
  [Spec 19](19-phase0-ci-security-gates.md) (`T37.2.6`).
