# OpenSpec: Web Exposure Boundary — Public Routes, CORS, Trusted Proxies, and Exposure Profiles

**Status:** Proposed. Describes intended behavior; the current code does not yet
enforce everything below. §3.4 (trusted-proxy and client-IP stance) is
**IMPLEMENTED** — see the implementation note at the end of this file.
**Owner:** Project maintainer.
**Date:** 2026-10-03.
**Roadmap:** Phase 0, work package 06
([`09-system-roadmap-and-phase-gates.md`](09-system-roadmap-and-phase-gates.md)).
**Related:** [`11-authorization-scopes-and-roles.md`](11-authorization-scopes-and-roles.md)
(principal identity), [`07-versioned-policy-pack-governance.md`](07-versioned-policy-pack-governance.md)
(trust boundaries).

## 1. Problem

The HTTP surface exposes three classes of route on a single listener that binds
`0.0.0.0` by default:

1. Fully public routes (health, metrics, API docs, the web UI, and a read-only
   API subset).
2. Authenticated routes keyed only by a bearer API key.
3. A static-asset exemption that skips authentication based on the request
   **path suffix and method**.

Three defects make the boundary hard to reason about and easy to widen by
accident:

- **The public set is implicit.** It is a hand-maintained list inside
  `APIKeyAuth`, duplicated again in `RateLimitMiddleware`. Nothing fails when
  the two drift apart, and a new route is public only if someone remembers to
  exclude it.
- **Client-IP trust is unspecified.** `c.RealIP()` is used for rate-limit keys
  and log attribution with no notion of a trusted proxy, so any caller can set
  `X-Forwarded-For`.
- **CORS default is permissive and silently rewritten.** The default origin is
  `*`, and `PRODUCTION_MODE` rewrites it to a small localhost set at wiring
  time rather than rejecting the misconfiguration.

The goal is a **declared, deny-by-default exposure contract**: every route has
an explicit exposure class, the trusted-proxy stance is stated, CORS is pinned
to named origins, and deployment profiles bound what the listener can reach.

## 2. Current observed behavior (audit snapshot, 2026-10-03)

Re-verify before starting; line references are to the state at this date.

**Listener binding.** Both servers bind all interfaces unconditionally:
`mcp-server/cmd/server/main.go` sets `0.0.0.0:%d` for the web listener
(`cfg.WebPort`, default 8081) and the MCP listener (`cfg.MCPPort`, default
8080). There is no bind-address configuration and no split between loopback and
external exposure.

**Public routes (authentication skipped).** From
`mcp-server/internal/web/middleware.go` `APIKeyAuth`:

| Route / pattern | Condition | Skip reason |
|---|---|---|
| `/health/live`, `/health/ready` | any | liveness/readiness |
| `/metrics` | any | Prometheus scrape |
| `/docs`, `/openapi.yaml` | any | API documentation |
| `/`, `/index.html`, `/web`, `/web/*`, `/static/*`, `/assets/*`, `/js/*`, `/css/*` | any | Web UI shell |
| path ending `.js/.css/.html/.svg/.png/.jpg/.ico/.woff/.woff2/.ttf` | `GET` or `HEAD` and path not under `/api/` | static asset |
| `/api/documents`, `/api/documents/search`, `/api/documents/*` | `GET` or `OPTIONS` | public read |
| `/api/rules`, `/api/rules/*` | `GET` or `OPTIONS` | public read |
| `/api/stats`, `/api/stats/*` | `GET` or `OPTIONS` | public read |
| `/api/projects`, `/api/projects/*` | `GET` or `OPTIONS` | public read |
| `/api/failures`, `/api/failures/*` | `GET` or `OPTIONS` | public read |
| `/api/ingest/status`, `/api/ingest/orphans` | `GET` or `OPTIONS` | public read |
| `/api/updates/status`, `/version` | `GET` or `OPTIONS` | public read |
| **any** route | `OPTIONS` | CORS preflight |

Everything else falls through to bearer authentication (fail-closed by
default). Write endpoints (`POST /api/ingest`, `POST /api/ingest/sync`,
`POST /api/updates/check`, all rule/project/failure mutations) require a key.

**Known sharp edges in the current set:**

- `/metrics` is public. It can expose route names, handler latency histograms,
  and volume — an information-disclosure surface on an internet-reachable
  listener.
- The `OPTIONS`-skips-everything rule means any path answers `OPTIONS` through
  the middleware chain; whether it returns 204 or 405 depends on Echo CORS
  matching, not on the exposure contract.
- The static-asset suffix exemption was already narrowed (method check +
  `/api/` exclusion) after a bug where `POST /api/ingest/x.js` skipped auth.
  It remains a suffix heuristic, not a route-allowlist.
- `/ide/*` restriction is ineffective: `middleware.go` checks
  `keyType != "ide" && keyType != "mcp"`, but `keyType` is only ever `"ide"` or
  `"mcp"`, so the branch never fires. Both key types reach `/ide/*`.

**CORS.** `config.go` defaults `CORS_ALLOWED_ORIGINS` to `*`, and validation
only rejects an empty list (`config.go:239-242`). `server.go:97-111` rewrites a
`*`/empty list to `http://localhost:8081`,`https://localhost:8081` when
`PRODUCTION_MODE` is true, and to `http://localhost:*`,`https://localhost:*`
otherwise. Any explicit non-`*` origin list is passed through unchanged.
`AllowMethods` defaults to `GET,POST,PUT,DELETE,OPTIONS`; `AllowHeaders` to
`Authorization,Content-Type,X-Request-ID`.

**Trusted proxy.** None. `c.RealIP()` (Echo) trusts `X-Forwarded-For`,
`X-Real-IP`, and `Forwarded` from any peer. It is used as the rate-limit key
fallback (`middleware.go:225`) and for log attribution
(`middleware.go:117`, `internal/middleware/logging.go:65`).

**Security headers.** `securityHeadersMiddleware` sets CSP, `nosniff`, frame
denials, and a `Permissions-Policy`, but also emits
`X-XSS-Protection: 1; mode=block`, which is obsolete and, in some legacy
browsers, itself a vector. CSP `connect-src 'self'` will also break a web UI
served from a different origin than the API.

## 3. Proposed contract

### 3.1 Declared exposure classes

Every route is assigned exactly one exposure class in a single registry that is
the source of truth for both authentication and rate limiting:

- **`public`** — reachable with no credential, on the declared methods only.
- **`authenticated`** — requires a valid bearer key; no anonymous access.
- **`privileged`** — requires an authenticated principal *and* an authorization
  decision per [`11-authorization-scopes-and-roles.md`](11-authorization-scopes-and-roles.md).

Rules:

1. **Deny by default.** A route not present in the registry is
   `authenticated`. Adding a route without declaring it cannot accidentally make
   it public.
2. **One registry, both middlewares.** `APIKeyAuth` and `RateLimitMiddleware`
   consult the same table, eliminating the drift between the two hand-copied
   lists.
3. **`public` is a written decision, not a suffix.** The static-asset
   exemption is replaced by explicit static route registration; the
   `.js`-suffix heuristic is removed once the web root is registered as a
   `public` prefix with its own method and content-type constraints.
4. **Minimized public read set.** Each currently-public read route is
   re-justified. Default is to move a route to `authenticated` unless its
   public exposure is required by a stated consumer.

### 3.2 Public route re-decision (target)

| Route | Proposed class | Rationale |
|---|---|---|
| `/health/live` | `public` | Container/orchestrator probe. |
| `/health/ready` | `public` | Probe; already omits failing component. |
| `/version` | `public` | No secret material; used by clients. |
| `/metrics` | **`authenticated`** (default) | Information disclosure; scrape should carry a key or be bound to an internal listener. |
| `/docs`, `/openapi.yaml` | `public` in dev, **`authenticated` in `production`** | Schema disclosure is low-risk in dev, avoidable in prod. |
| Web UI shell + static assets | `public` | Browser must load shell before auth; data routes stay authenticated. |
| `/api/documents*`, `/api/rules*`, `/api/stats*`, `/api/projects*`, `/api/failures*` (GET) | **`authenticated`** unless a named anonymous consumer exists | Guardrail content can embed repository paths and rule text. |
| `/api/ingest/status`, `/api/ingest/orphans`, `/api/updates/status` (GET) | `authenticated` | Operational state; not needed anonymously. |

The exact final set is an owner decision (§7); the contract is that the set is
*declared and reviewed*, not inherited.

### 3.3 CORS policy

1. `CORS_ALLOWED_ORIGINS` **must** enumerate explicit scheme+host+port origins.
   Wildcards and `*` are rejected in `production` (validation error, not a
   silent rewrite).
2. When `PRODUCTION_MODE` is true, `*` (or empty) is a **startup failure**, not
   a substitution. Non-production retains a convenient default but logs a
   warning naming the effective origin list.
3. `AllowCredentials` stays `false` while auth is bearer-header based. If a
   cookie flow is added later, `*` becomes unconditionally illegal.
4. Preflight (`OPTIONS`) is answered by the CORS middleware for **declared**
   routes only; `OPTIONS` no longer bypasses authentication for undeclared
   paths.

### 3.4 Trusted-proxy and client-IP stance

1. A new `TRUSTED_PROXIES` setting lists CIDRs permitted to assert
   `X-Forwarded-For`/`X-Real-IP`. Default is empty (trust no proxy).
2. When the immediate peer is not trusted, client IP is the socket peer
   address and forwarded headers are ignored for rate-limit keys, audit, and
   logging.
3. `RealIP`-derived values are **never** used as an authorization input. The
   rate-limit fallback (unkeyed requests) uses the socket peer, so forwarded
   headers cannot be used to evade or poison rate-limit buckets.
4. Requests from an untrusted peer that carry forwarded headers are logged as a
   protocol anomaly (not rejected), so misconfiguration is visible.

### 3.5 Deployment exposure profiles

The listener bind address becomes configurable and is tied to named profiles:

| Profile | Bind | Intended reach | Required posture |
|---|---|---|---|
| `local` | `127.0.0.1` | single workstation | loopback only; TLS optional |
| `tailnet` | tailnet interface | private overlay | trusted-proxy CIDRs set to overlay; TLS recommended |
| `public` | `0.0.0.0` | internet | `PRODUCTION_MODE=true`, explicit CORS origins, TLS enabled, `/metrics` and `/docs` authenticated, trusted-proxy set or empty |

`0.0.0.0` remains available but is no longer the unconditional default: it is
selected by the `public` profile, which also requires the hardened settings.

### 3.6 Security headers

- Remove `X-XSS-Protection` (obsolete; can introduce vulnerabilities in legacy
  engines).
- Keep CSP `default-src 'self'`; document that a cross-origin web UI requires a
  reviewed `connect-src` addition rather than an implicit widening.
- Keep `nosniff`, frame denial, `Referrer-Policy`, and the empty
  `Permissions-Policy`.
- Add HSTS **only** when TLS is enabled and the profile is `public`.

## 4. Acceptance criteria

Each criterion names an executable check, the expected result, and the failure
condition. All are to run in CI under the Phase 0 security gate.

1. **Registry completeness.** A test enumerates every registered Echo route and
   asserts each maps to exactly one exposure class. *Expected:* zero unmapped
   routes. *Failure:* any route with no declared class, or two classes.
2. **Middleware agreement.** A test asserts `APIKeyAuth` and
   `RateLimitMiddleware` derive their skip set from the same registry. *Failure:*
   a route classified `public` in one path but not the other.
3. **Deny-by-default.** A test registers a new undocumented route and asserts it
   requires authentication without any middleware change. *Failure:* the new
   route is reachable anonymously.
4. **Static-asset exemption is gone.** A test posts to `/api/ingest/evil.js`
   (and sibling suffix variants) and asserts `401` without a key. *Failure:*
   any non-`2xx`-less bypass; i.e., the request is not rejected with `401`.
5. **CORS production rejection.** With `PRODUCTION_MODE=true` and
   `CORS_ALLOWED_ORIGINS=*` or empty, config validation returns an error.
   *Failure:* the process starts and serves with a wildcard-derived origin list.
6. **CORS explicit origins.** With an explicit origin, a preflight from that
   origin succeeds and a preflight from another origin does not receive
   `Access-Control-Allow-Origin`. *Failure:* origin reflected for an unlisted
   origin.
7. **Trusted-proxy.** With `TRUSTED_PROXIES` unset, a request carrying
   `X-Forwarded-For: 1.2.3.4` from an untrusted peer uses the socket peer for
   the rate-limit key; two requests with different forged headers and one real
   peer share a bucket. *Failure:* forged headers create separate buckets.
8. **Metrics exposure.** In the `public` profile, an unauthenticated
   `GET /metrics` returns `401`. *Failure:* metrics body served.
9. **Profile binding.** The `local` profile binds loopback only; a connection to
   the host's external address is refused. *Failure:* `0.0.0.0` bind under
   `local`.
10. **X-XSS-Protection absent.** A response does not set
    `X-XSS-Protection`. *Failure:* header present.

## 5. Migration and compatibility

- **Registry first.** Introduce the exposure registry and route both
  middlewares through it with behavior identical to today, then tighten the
  public set in a separate change. This keeps each step reviewable and avoids a
  single change that both refactors and widens/narrows access.
- **Default-origin change.** Moving `PRODUCTION_MODE` from "silently rewrite
  `*`" to "fail on `*`" may break existing deployments that relied on the
  rewrite. Such deployments must set an explicit origin list; the migration note
  names the previous effective value.
- **Bind change.** The hardcoded `0.0.0.0` becomes the `public` profile. Existing
  container deployments stay reachable only if they select `public` (or another
  non-loopback profile); the compatibility note and default profile must be
  chosen so that a containerized deployment does not silently lose reachability.
- **Docs.** `docs/platform-current-state.md` is updated to state the exposure
  contract and the selected profile; the route table here is the review record.

## 6. Open decisions and first-implementation defaults

[Spec 16 §4](16-authentication-authorization-and-evidence-remediation.md)
selects the conservative initial profile for implementation: only liveness,
readiness and version are anonymous operational endpoints; metrics, docs and
API data require authentication in production; static assets are explicit
paths, not suffix exemptions; local bind is loopback; production CORS uses
named origins and forwarded client IP is trusted only from configured proxy
CIDRs. This is a proposed rollout target, not current behavior. Any required
anonymous consumer or external bind needs a reviewed exception and negative
route tests before enabling it.

1. **Public reads:** existing document/rule/stats/project/failure GETs become
   authenticated by default; only live/ready/version remain anonymous. Any
   additional anonymous consumer needs a named, reviewed route exception.
2. **Metrics/docs:** `/metrics`, `/docs` and `/openapi.yaml` require an
   authenticated principal in production, or a separate restricted internal
   listener with an authenticated scraper. They are not public by default.
3. **Default profile:** `local` binds host loopback. Existing container users
   must opt into a verified isolated bridge or `tailnet`/`public` profile; a
   failed exposure preflight blocks startup rather than silently widening bind.
4. **Trusted proxies:** empty by default; forwarded headers are ignored unless
   the operator explicitly configures a verified CIDR. The `public` proxy
   profile requires both edge sanitization and backend restriction.

These are proposed implementation defaults. If a real client cannot migrate,
stop and revise the profile with tests instead of enabling wildcard access.

## 8. Implementation note — §3.4 trusted-proxy and client-IP stance (2026-10-15)

§3.4 is implemented. The effective client IP is derived from a single explicit
stance, not from `c.RealIP()`'s unconditional header trust:

- New `internal/clientip` package: `NewStance(entries)` parses `TRUSTED_PROXIES`
  CIDRs (a bare IP is a single-host entry). An **empty list trusts no proxy**
  (the safe default); a malformed entry is a **hard error**, not a silent
  narrowing to "trust nothing" (a silent narrowing can flip the exposure
  posture without telling the operator).
- `Stance.ClientIP(r)` returns the socket peer unless the immediate peer is
  inside a configured CIDR; only then is a forwarded address used
  (`X-Forwarded-For` left-most, else `X-Real-IP`, else RFC 7239 `Forwarded`
  `for=`). IPv4-mapped/bracketed/zoned forms are canonicalised.
- `Server.setupMiddleware` installs `stance.Extractor()` as `echo.IPExtractor`,
  so `c.RealIP()` — used for the rate-limit key fallback (`middleware.go`) and
  log attribution (`middleware/logging.go`) — is the stance-derived value.
  Forwarded headers can no longer be used to evade or poison rate-limit buckets
  (§3.4.3); `RealIP` remains a non-authoritative value (never an authz input).
- `forwardedHeaderAnomalyMiddleware` logs a protocol anomaly (§3.4.4) when an
  untrusted peer asserts a forwarded header — logged, **not rejected**.
- `Config.Validate` parses the stance so a malformed `TRUSTED_PROXIES` fails
  startup (fail-closed).

**Evidence:** `internal/clientip/clientip_test.go` (default trust, trusted peer,
bare-IP entry, malformed-entry fail-closed, IPv6/zone, extractor, anomaly
probe); `internal/web/middleware_clientip_test.go` (acceptance criterion 7 —
forged headers share the socket-peer bucket; trusted peer uses the forwarded IP;
anomaly middleware passes through); `internal/config/trusted_proxies_test.go`
(empty/valid/malformed). CI row `security-matrix-clientip-trust` runs these
verbatim. Mutation-kill case `clientip-stance-trusts-every-peer` proves the
suite flips PASS→FAIL when the stance is neutered.

**Remaining in this spec (not addressed here):** the shared exposure registry
(§3.1/§3.2) and the `public`-profile `/metrics` auth decision (§3.2) — the
latter is an operator contract (scrape client) and stays an owner decision.

## 9. Implementation note — §3.3.2 CORS production rejection + §3.3.4 declared-route preflight (2026-10-16)

§3.3.2 and §3.3.4 are implemented at the wiring layer.

- **§3.3.2 — production wildcard is a startup failure, never a substitution.**
  `config.Validate` already rejected `*`/empty in production, but
  `Server.setupMiddleware` re-implemented the default substitution inline: it
  replaced `*` with localhost defaults *before* the production branch was
  consulted in the wiring code path, and it silently substituted rather than
  failing. The substitution is now a single method, `Config.EffectiveCORSOrigins`
  (new `internal/config/cors.go`): explicit origins pass through; production +
  empty/`*` returns an error and a nil list; non-production + empty/`*` returns
  the restrictive localhost defaults with `substituted=true` so the caller logs
  a warning naming the effective list. `setupMiddleware` fails closed on the
  error (a sentinel origin that matches nothing — echo treats an empty
  `AllowOrigins` as the wildcard default) and logs the substitution when it
  occurs. A directly-constructed `Server` can no longer widen exposure.
- **§3.3.4 — preflight is answered for declared routes only.** The Web UI mounts
  a catch-all (`echo.Static("/", …)` plus the SPA fallback `GET /web/*`).
  Echo's router therefore matches every path, and its default OPTIONS handling
  answers any such path with `204` + `Allow`, so an OPTIONS request to an
  *undeclared* path was served as if it were a preflight for a real route —
  OPTIONS "declared" arbitrary paths and bypassed the normal 404/405 outcome.
  New `declaredPreflightOnly` middleware (new `internal/web/preflight.go`) runs
  before CORS and acts only on OPTIONS: when the matched route pattern is a
  catch-all (contains `*`) the request matched only the file/SPA fallback and is
  refused with `404`, never reaching CORS. Every other OPTIONS request is left
  for the CORS middleware. Live probe (before the fix) confirmed the defect:
  with the catch-all registered, `OPTIONS /api/nope` returned `204` with
  `Allow: OPTIONS, GET`; after the fix it returns `404`.

**Evidence:** `internal/config/cors_test.go` (explicit pass-through;
production empty/wildcard fails closed with a nil list; non-production
substitution reported; no caller-slice aliasing) and
`internal/web/preflight_test.go` (declared route preflight 204; undeclared,
undeclared-nested and unrelated paths 404; non-OPTIONS untouched) — all driving
the real router + middleware chain. CI row `security-matrix-cors-preflight` runs
these verbatim. Mutation-kill cases
`preflight-declared-route-guard-removed` and
`cors-production-wildcard-substituted-not-refused` prove each suite flips
PASS→FAIL when its control is neutered.

## 7. Non-goals

- Principal identity and role enforcement — Spec 11.
- Webhook destination SSRF policy — Spec 13 (`T37.2.3`).
- TLS certificate provisioning and secret handling — Spec 14 (`T37.2.4`).
- Migration startup/readiness semantics — Spec 15 (`T37.2.5`).
