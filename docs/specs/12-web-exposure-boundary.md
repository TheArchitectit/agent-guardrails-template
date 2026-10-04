# OpenSpec: Web Exposure Boundary — Public Routes, CORS, Trusted Proxies, and Exposure Profiles

**Status:** Proposed. Describes intended behavior; the current code does not yet
enforce everything below.
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

## 6. Open decisions (owner)

1. **Final public read set.** Which of the currently-public GET routes must
   remain anonymous, and for which named consumer? Default proposal: all become
   `authenticated` (§3.2).
2. **`/metrics` exposure.** Authenticate, move to a separate internal listener,
   or keep public only under `local`/`tailnet`?
3. **`/docs` and `/openapi.yaml`.** Remain public in production, or gated?
4. **Default deployment profile.** Should the shipped default be `localhost`
   (safe, may break container reachability) or `public` (reachable, requires
   hardening)? This determines whether existing deployments must opt in.
5. **Trusted-proxy default.** Empty (trust none) is assumed; confirm whether any
   current deployment sits behind a reverse proxy that must be declared.

## 7. Non-goals

- Principal identity and role enforcement — Spec 11.
- Webhook destination SSRF policy — Spec 13 (`T37.2.3`).
- TLS certificate provisioning and secret handling — Spec 14 (`T37.2.4`).
- Migration startup/readiness semantics — Spec 15 (`T37.2.5`).
