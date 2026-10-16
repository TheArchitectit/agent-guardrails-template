# OpenSpec 17: Deployment TLS and secret boundary

**Status:** Proposed; NOT implemented. Source observations below describe the current files, not deployed guarantees. Re-check at implementation time.
**Owner:** Solo maintainer (deployment/security owner); no separate operations team assumed.
**Dependencies:** [Spec 11](11-authorization-scopes-and-roles.md) owns scoped credentials and role intersection; [Spec 12](12-web-exposure-boundary.md) owns route classes, CORS, proxy trust and bind profiles; [Spec 16](16-authentication-authorization-and-evidence-remediation.md) owns fail-closed registry loading, config-resource redaction, audit, lifecycle and release gates. This spec adds transport, deployment and secret-source gates; it does not mark those dependencies shipped.

## 1. Source evidence versus proposed target

| Current observation (source) | Risk / distinction from proposal |
|---|---|
| `docker-compose.yml:3,14,19,27,32,59,67-69,84-90` uses floating Redis/Postgres image tags and a floating app image tag; Redis and database password variables fall back to a fixed, publicly known placeholder; `DB_SSLMODE` falls back to `disable` and `REDIS_USE_TLS=false`. Both published app ports are host-loopback only. | Loopback publishing is useful for a local Compose run, not a production security profile or a guarantee about other orchestrators/containers on the bridge. Do not use the placeholder credentials in any real deployment. |
| The primary `docker-compose.yml:6-11,19-20` and `mcp-server/deploy/docker-compose.example.yml:29-30` use a credential-bearing Redis healthcheck argument; the primary Compose also builds Redis config using a shell command. `docker-compose.yml:91-96` injects MCP/IDE API keys and JWT secret as environment values. | Required auth values do not make DB/Redis defaults safe; both healthchecks and command arguments must be replaced, and container environment/inspection remains a sensitive surface. |
| `mcp-server/cmd/server/main.go:162-190` starts both web and MCP listeners on `0.0.0.0` using HTTP; `internal/mcp/server.go:356-375` calls `ListenAndServe`. `main.go:239-261` probes local HTTP; `:264-276` binds pprof to `localhost` when enabled. | **Native listener TLS is absent:** `TLS_ENABLED` and certificate settings are validated (`config.go:223-234`) but do not change either HTTP serving path. Host loopback publishing does not change the in-container binds. |
| `mcp-server/internal/config/config.go:45-75,126-127,223-245` defaults DB SSL to `require` **outside Compose**, Redis TLS and `TLS_ENABLED` to false, and TLS minimum to `1.3`; validation accepts TLS minimum `1.2` if enabled and permits DB `disable`, `prefer` and `require`. | A TLS field or minimum-version validation is not evidence that the listener uses TLS. `require` without certificate/hostname validation is insufficient for off-host database trust. |
| `config.go:77-94,160-166,362-367,417-428` carries API keys, verifier key, inline/file registry, JWT secret and a password-bearing database URL; registry validation checks verifier-key length, not file validity. `Masked()` masks several fields but not `CredentialRegistryFile` (a path). | Never serialize the full config or its DSN to a resource, error, metric, trace or log. Validate secret sources and registry integrity before opening listeners; see Spec 16 R16-03–05. |
| `mcp-server/internal/database/webhooks.go:25-36,43-52` writes and reads a `secret_hmac` field for webhook configuration. Spec 12 §2 identifies `/metrics` and `/docs` as currently public; Spec 16 §1 identifies the current full-config MCP resource and raw tool-argument logging. | Webhook signing material and operational routes need their own access boundaries. Merely storing a webhook secret in a database is not proof of encryption, redaction or safe rotation. |

**Scope:** Production-ready deployment contracts for web/MCP ingress, database/Redis connections, secrets, observability and CI. This is a documentation proposal only: no existing config field, TLS server, ingress policy, secret mount loader or tests are claimed to exist. Spec 12's profile vocabulary is retained; the `public` profile below means *public HTTPS at a terminating proxy*, not direct public HTTP. If a future native-TLS listener is chosen, it must satisfy the same gates before external use.

## 2. Deployment profiles and mandatory requirements

| Profile | Listener / reachable path | Transport and admission |
|---|---|---|
| `local` (safe default) | Native workstation: web and MCP bind loopback only. Compose: container may listen on its isolated bridge for host-loopback publishing; prohibit any non-loopback host publish or untrusted bridge peer. | HTTP allowed solely across isolated local loopback/bridge; use distinct generated local credentials. DB/Redis plaintext only on that verified isolated local network. No implied external reachability. |
| `tailnet` (private overlay) | Bind only the selected private overlay interface, or proxy from it to an isolated backend; firewall rejects other ingress. Do not treat a wildcard bind as private by name alone. | TLS with peer authentication for overlay clients unless a separately reviewed encrypted authenticated tunnel terminates at the same trust boundary; document tunnel endpoints and direct-path denial. Named private client identities/scopes still required. |
| `public` (reverse-proxy TLS termination) | Public HTTPS at a controlled ingress; backend web/MCP HTTP reachable only from the dedicated proxy over an isolated private network or local socket. Firewall/network policy blocks direct public and lateral backend reach. Both paths must be covered. | External TLS >=1.3 with valid certificate and hostname checks; reject plaintext public connections (redirect is not authorization). Proxy forwards only approved headers and replaces untrusted forwarded headers. No public direct HTTP fallback. Native TLS, if implemented later, must use TLS >=1.3 for both listeners. |

**R17-01 — Explicit effective profile and bind.** Select a named profile at startup, default `local`. Validate the actual app bind, host-published mappings, proxy upstream and firewall/network-policy reachability as a deployment unit. A container-wide `0.0.0.0` is allowed only on a verified isolated backend network in `local` Compose or behind an isolated proxy in `tailnet`/`public`; it is never by itself evidence of safe exposure. If the process cannot inspect an external host mapping or firewall, a deployment preflight must provide a verifiable attestation; missing/contradictory evidence blocks nonlocal readiness.

**R17-02 — Fail closed on insecure exposure.** Before starting either listener, fail startup (nonzero exit and no app socket) on an external bind or publish without an approved profile, on `public` without proven TLS termination and direct-backend denial, or on `tailnet` without verified private ingress plus encrypted authenticated transport. A true `PRODUCTION_MODE` flag, `TLS_ENABLED=true` or presence of certificate paths alone never satisfies this gate. Do not silently downgrade to `local` if the configured interface remains externally reachable. Block readiness if post-start deployment-policy verification detects drift; halt protected traffic until repaired.

**R17-03 — Public TLS and proxy boundary.** Terminate TLS 1.3+ at the public ingress; rotate and validate its certificate, key and trust chain. Disable TLS 1.2 and older for public clients; reject an invalid certificate, wrong hostname or an untrusted backend route. Only configured proxy peers may supply forwarded identity/scheme; clear inbound forwarded headers at the edge and use the trusted-proxy stance from Spec 12. HSTS may be emitted only for a verified public HTTPS origin, never as a substitute for TLS. If proxy-to-backend traffic crosses a non-isolated network, encrypt and authenticate that hop too.

**R17-04 — DB and Redis transit.** Within one verified isolated local bridge/loopback, unencrypted DB/Redis is permitted for `local` only; no other workload may attach to that network. Whenever either connection leaves that boundary (including private overlay, cross-host, shared container network or public deployment with remote storage), require authenticated encryption and fail startup if missing: PostgreSQL `verify-full` with trusted CA and hostname verification, Redis TLS with verified server certificate/hostname, no permissive skip-verify or opportunistic fallback. Restrict DB/Redis reachability to the app and approved operators; never publish their ports publicly. The Compose `DB_SSLMODE=disable`/Redis TLS=false settings cannot be reused as production defaults.

**R17-05 — No production default credentials.** Nonlocal or production profiles require unique high-entropy operator-generated DB, Redis, MCP/IDE (during migration), scoped registry and JWT credentials; reject unset, empty, known placeholder/default, reused or test credentials before startup. No checked-in sample is a production credential. DB and Redis account privilege is service-specific, with separate accounts per environment. Local also generates distinct non-placeholder credentials at setup; do not advertise the current fixed Compose fallback as safe even on loopback.

**R17-06 — Secret source and least privilege.** Use a managed secret store or runtime read-only mounted secret files as first-choice production source for DB/Redis, MCP/IDE legacy keys, `CREDENTIAL_REGISTRY_FILE`, `CREDENTIAL_VERIFIER_KEY`, JWT signing material, webhook HMAC secrets, `FALLBACK_API_KEY` for vision and `OPENAI_API_KEY` for content filtering, and TLS private keys. Webhook HMAC material currently persists as `secret_hmac` in `internal/database/webhooks.go:32-52`; implementation MUST define encrypted-at-rest storage or a managed secret reference, restrict read/list APIs, and rotate receivers without exposing the secret. Add a file/secret-provider configuration path as part of implementation; do not pretend existing env-only fields already support it. Restrict file read to the service identity and rotation operator (POSIX owner-only mode such as `0600`, private parent directory; equivalent restrictive Windows ACL), prevent group/world access, never mount into unrelated services, and exclude secret volumes from backups/log exports unless encrypted and access-controlled. Environment injection, if unavoidable during migration, is restricted to trusted runtime, with no diagnostic dumps or inspect access for untrusted operators; no plaintext secrets in committed `.env`, Compose files or container images.

**R17-07 — Registry and identity secrets.** Treat the registry's verifiers and verifier key as separate sensitive inputs; load and validate the entire configured registry before listeners, reject unreadable/malformed/empty configured sources, and never fall back to unrestricted legacy keys (Spec 16 R16-03). Generate the verifier key out of band with >=256 random bits; a length check is not an entropy proof. Never put raw credentials in the registry or return verifier material. Enforce Spec 11 scope/role/resource intersection on web and MCP, including resources; replace or restrict `guardrail://config` with a reviewed non-secret projection before expanding exposure.

**R17-08 — Secret-free process and observability.** Do not pass credentials via argv, shell-expanded command strings, healthcheck flags or URLs; replace the Redis `redis-cli -a` probe with a health method that does not expose a secret in process arguments or logs. Redact nested tool arguments, DSNs, authorization headers, registry JSON, webhook payload/signature keys, vision `FALLBACK_API_KEY`, content-filter `OPENAI_API_KEY`, and secret file contents from errors, logs, audit, traces, metrics, crash reports, health output and MCP/REST resources. Use opaque credential IDs for attribution, not raw keys or their reversible forms; test error paths and startup failure output as well as success paths. Paths to secret files should not be disclosed to ordinary callers.

**R17-09 — Metrics, docs and debug binding.** Follow Spec 12/16: `/metrics`, `/docs` and `/openapi.yaml` are authenticated by default in production, or served on a separately restricted internal listener with authenticated scrape clients. Only reviewed health/version endpoints may be anonymous on public ingress; health output contains no config values. Keep pprof disabled by default; when enabled for diagnosis, bind to a local-only administrative endpoint with no public/tailnet proxy route and time-box access. Explicitly test both web and MCP paths and all alternate ingress routes.

**R17-10 — Rotation and emergency revocation.** Maintain a secret inventory (owner, consumer, source, expiry, last rotation, revocation procedure) without values. Rotate DB/Redis, JWT, webhook and TLS material with a staged replacement and explicit cutover; invalidate old material after a bounded overlap, preserve service continuity only while both are valid, and log opaque rotation events. JWT signing-key rollover must identify the verifying key, bound token expiry and reject retired keys; webhook rotation must identify the active signing key to receivers without leaking it. For registered credentials adopt Spec 16's proposed <=90-day validity, <=24-hour planned overlap and <=60-second tested denial of revoked keys on both transports; if reload cannot meet the bound, replace/restart instances before claiming revocation. On suspected disclosure revoke/rotate affected credentials and certificates, invalidate exposed sessions, review audit and block unsafe effects rather than restoring old secrets.

**R17-11 — CI and image provenance.** Pin build/test actions by immutable full commit SHA and production base/service images by immutable digest; record review/refresh ownership and automated update cadence, not a `latest`/floating tag as release proof. Supply CI secrets only to approved trusted jobs from the CI secret store with minimal scope; untrusted fork/PR code receives no deployment secrets, no privileged runner or mutable production credentials. Never print tokens or interpolate them into command arguments/artifacts; redact job logs and run secret scanning as a blocking release gate after classifying findings per Spec 16 R16-02. Build once from the reviewed commit, verify the pinned image digest at promotion, and prevent unreviewed CI workflow changes from gaining secret access.

## 3. Executable acceptance scenarios (future tests; none run here)

Use synthetic credentials, disposable local services and documentation-only topology descriptors; never test against production accounts or a public endpoint. A profile/preflight integration harness must start the real server with a stubbed deployment-attestation provider and capture listener addresses, readiness and exit codes; a network harness must verify proxy and storage reachability, not merely compare env values.

1. **Local positive:** select `local` with unique synthetic secrets, isolated Compose bridge and host-loopback-only app publishes; start server, reach both app endpoints from allowed host loopback, refuse an external-interface connection, accept HTTP only within that isolated boundary. **Negative:** advertise an externally published app port or attach an untrusted bridge peer; preflight/startup fails rather than silently serving.
2. **Tailnet positive:** present a private-interface bind (or restricted proxy), approved encrypted peer-verified tunnel/TLS and denied alternate ingress; connect as a permitted scoped client. **Negative:** wildcard/public route, unverified peer or plaintext private off-host hop fails startup; no app socket remains.
3. **Public positive:** terminate TLS 1.3+ for web and MCP at the proxy, verify certificate and hostname from the client, restrict backend to proxy-only, then confirm approved requests work. **Negative:** TLS 1.2 handshake, cleartext request, invalid certificate, direct-backend reach, absent termination attestation or false-positive `TLS_ENABLED=true` with HTTP-only start blocks exposure/startup. Assert both listeners are covered.
4. **Storage positive:** with DB/Redis outside the isolated local network, use verified PostgreSQL `verify-full` and Redis TLS with hostname validation. **Negative:** DB `disable`/`prefer`/unverified `require`, Redis TLS=false or skip-verify, bad hostname, or plaintext remote storage connection prevents startup/readiness.
5. **Secrets positive:** mount synthetic registry/verifier/JWT/webhook/DB/provider secrets in service-only readable files, start with a valid registry and exercise a scoped authorized call. Read/list webhook configurations and assert HMAC values are absent; rotate an HMAC with overlapping receiver verification under a tested bound. **Negative:** missing file, permissive file ACL, configured empty/malformed registry, placeholder credential, plaintext webhook secret in an export, or revoked key denies at the prescribed boundary; legacy key never gains administrator authority.
6. **Leak positive:** seed distinctive fake secret markers in each source, including nested MCP args and a forced DB-connect error, then search complete stdout/stderr, log/audit/trace/metric snapshots, health/docs, REST and MCP resource responses, process argv and CI artifacts; none contains a marker. **Negative control:** current-style credential-bearing Redis healthcheck arguments or full-config resource serialization must be detected and fail the harness.
7. **Operational routes positive:** health is minimal and permitted; authorized internal scraper/docs user succeeds, pprof stays local/disabled. **Negative:** unauthenticated public metrics/docs, forwarded-header spoofing from an untrusted peer, and proxy routing to pprof fail.
8. **Lifecycle/CI positive:** rotate each secret with a short test overlap; new credential works, old stops on schedule, registry revocation denies new web/MCP calls within 60 seconds. CI policy checker accepts only digest-pinned production images, full-SHA actions and secret-free untrusted jobs. **Negative:** stale revoked key, perpetual JWT/webhook overlap, floating image/action or PR access to deployment secrets blocks release.

**Acceptance gate:** Owner records the exact commit/image digest, resolved profile and preflight evidence, named positive/negative results, secure rollback snapshot and dependency gates. Any secret marker leak, unauthorized reachability, plaintext nonlocal hop, untrusted certificate, default credential, incomplete Spec 11/12/16 gate or unresolved real secret-scan finding blocks production release. No passing status is asserted by this document.

## 4. Migration, rollback and solo-owner defaults

1. Inventory running workloads, ingress paths, published ports, backend peers, credential consumers and secret owners; map each to one profile. Default new installs to `local` and deny nonlocal startup until explicit attestation and ingress tests exist. Do not infer safety from `PRODUCTION_MODE` or Compose loopback mapping alone.
2. First introduce secret-file/provider loading and negative leak tests, remove command-line healthcheck secrets and full-config disclosure, validate registered credentials and apply Spec 11/12/16 gates. Provision distinct synthetic staging and then real production secrets out of band; migrate one consumer at a time, including DB/Redis and webhook receivers. Never reuse local/CI values in production.
3. Deploy ingress with TLS 1.3+, verified private backend restriction, DB/Redis transport policy and pinned images/actions. Exercise both ports and denial paths from each network zone before changing traffic. Record certificate expiry, owner and renewal alert; only then cut over `tailnet` or `public`.
4. Roll back config/policy to the last known-good **restrictive** profile, still-pinned image and valid rotated secret set; a previous HTTP external bind, placeholder DB/Redis credential, unredacted resource, legacy unrestricted auth or disabled TLS is not a valid rollback. If no safe snapshot exists, stop external/protected traffic and preserve audit records while repairing. Revoke exposed material rather than reusing it.

**Solo-maintainer first decisions:** the maintainer owns the release profile and evidence. Default `local` binds loopback; `tailnet` and `public` use an operator-managed TLS 1.3+ reverse proxy with a private backend, not the currently unwired native TLS flag. First implementation reads secrets from read-only mounted files with restrictive permissions; external KMS is optional later. No production fallback credential; DB `verify-full` and Redis certificate-verified TLS are mandatory outside a proven isolated local bridge. Metrics/docs require auth, pprof stays disabled. The deployment preflight emits a commit-bound record of selected profile, listener/host mappings, proxy upstream, certificate identity/expiry, allowed network paths, DB/Redis TLS modes and synthetic negative probes; an operator-supplied JSON assertion without an actual external reachability/certificate test is NOT evidence. Security exceptions need a named scope, expiry and negative tests before activation. Specific proxy product, real host topology, JWT/webhook key-ID mechanism and CI pin inventory remain implementation details, **not shipped capabilities**.

## 5. Implementation note (2026-10-15) — R17-11 action pinning half

R17-11 requires build/test actions pinned by immutable full commit SHA (and
production images by digest). The **action half is now implemented and
enforced**: every third-party action in all six workflows under
`.github/workflows/` is pinned by a 40-hex commit SHA (resolved and re-verified
live via the GitHub API; human tag kept in a trailing comment), and
`scripts/check_action_pins.py` enforces it on every push — fail-closed and
non-vacuous, with `scripts/test_check_action_pins.py` as the mutation fixture
(a mutable tag / missing ref / unpinned container must change PASS → FAIL). The
gate runs in `.github/workflows/regression-guard.yml`.

The **image-digest half is now implemented and enforced too** (2026-10-15).
Every external image ref in the container manifests is pinned by an immutable
`@sha256` digest, and a reviewed registry records the digest per `name:tag`:

- `docker-compose.yml`, `mcp-server/deploy/podman-compose.yml`,
  `mcp-server/deploy/docker-compose.example.yml`: `redis:7-alpine` and
  `postgres:16-alpine` are pinned by digest.
- `mcp-server/deploy/Dockerfile`: both base images (`golang:1.25-alpine` builder,
  `gcr.io/distroless/static:nonroot` runtime) are pinned by digest.
- `image-pins.json` is the reviewed record (canonical `name:tag` → digest, plus a
  role and a review date). Each digest was resolved live against the registry and
  re-verified to be that tag's current index digest.
- `scripts/check_image_pins.py` enforces the rule on every push: fail-closed (an
  unreadable or missing manifest is a violation, not a skip) and non-vacuous (a
  scan that finds zero external image refs FAILS). A tag-only ref, a malformed
  digest, a pin absent from the registry, or a digest that disagrees with the
  reviewed record is a violation. The first-party image built from this repo's
  own Dockerfile is allowlisted by name (its digest is recorded at promotion per
  R17-11); the allowlist carries a reason per entry.
- `scripts/test_check_image_pins.py` is the mutation fixture: it asserts the gate
  changes PASS → FAIL on a tag-only ref, a mismatched digest, and an unreviewed
  pin, and PASS on a digest-pinned, registry-matching ref.
- The gate + self-test run in `.github/workflows/regression-guard.yml` on every
  push and PR (the step "Image-digest pinning (Spec 19 R19-15 / Spec 17 R17-11)").

Known limit (documented, not a gate weakening): `ci/gitlab-ci.yml` is a copy-me
**template** (its own header says "Copy this file to .gitlab-ci.yml"), not an
active pipeline, so its variable-tagged `python`/`golang` images are out of the
gate's manifest scope; a repository that activates it must pin those images too.
The `.github/workflows/` chain — the one that actually gates releases — is fully
covered. No gate was weakened.

## 6. Implementation note (2026-10-15) — R17-08 secret-free process (argv half)

R17-08 names one concrete defect: *"replace the Redis `redis-cli -a` probe with
a health method that does not expose a secret in process arguments or logs."*
That defect was present in three manifests and is now fixed, with a gate that
keeps it fixed.

**Defect (live-verified before the fix).** A credential passed on a command line
is exposed to every process on the host (`/proc/<pid>/cmdline`) and to
`docker inspect` / `podman inspect`, and it is written to logs on a crash or a
verbose error path. The tree carried four instances:

- `docker-compose.yml:20` — healthcheck `["CMD", "redis-cli", "-a", "${REDIS_PASSWORD:-changeme}", "ping"]`.
- `mcp-server/deploy/podman-compose.yml:9,35` — server `--requirepass ${REDIS_PASSWORD}` in argv **and** the same `-a` healthcheck.
- `mcp-server/deploy/docker-compose.example.yml:8,30` — the same two forms.

**Second defect found and fixed in the same pass (live-verified).**
`docker-compose.yml`'s redis `command:` wrote its config to
`/usr/local/etc/redis/redis.conf` without creating the directory. `redis:7-alpine`
ships no such directory, so the shell aborted with `can't create
/usr/local/etc/redis/redis.conf: nonexistent directory` and **redis never
served** (reproduced live). The fix adds `mkdir -p`.

**Fix (argv-free path).** The password reaches redis only through a config file
written from the environment at startup, and the probe authenticates with
`REDISCLI_AUTH` from the environment:

- server: `sh -c 'mkdir -p /tmp/redis && printf \'requirepass %s\\n…\' "$REDIS_PASSWORD" > /tmp/redis/redis.conf && exec redis-server /tmp/redis/redis.conf'`;
- healthcheck: `["CMD", "redis-cli", "ping"]` with `REDISCLI_AUTH` in the
  service `environment:` block.

Live proof on the pinned `redis:7-alpine` digest: `redis-cli ping` under
`REDISCLI_AUTH` returns `PONG`; without it the same probe returns
`NOAUTH Authentication required.`; the `-a` form prints redis's own warning
`Using a password with '-a' or '-u' option on the command line interface may not
be safe.`; and `docker inspect` renders `[redis-server --requirepass <pw>]` for
the argv form. The argv-free form was re-verified under the hardened runtime
(`--user 999:1000 --cap-drop ALL --security-opt no-new-privileges`).

**Gate.** `scripts/check_secret_argv.py` scans the four container manifests on
every push and PR (step "Secret-free process (Spec 17 R17-08)" in
`regression-guard.yml`). Two rule families per command sequence:

1. **credential flag with a value** — a token in `{-a, --requirepass,
   --password, --pass, --auth}` followed by a value (literal or `${VAR}`), or
   the `--flag=value` form. This is the named `redis-cli -a` /
   `redis-server --requirepass` defect.
2. **secret variable expanded in argv** — a probe/healthcheck token expanding
   `*_PASSWORD` / `*_SECRET` / `*_TOKEN` / `*_KEY` / `REDISCLI_AUTH` / `DB_DSN`
   / `DATABASE_URL`. Orchestrators inspect and log probe argv, so a secret
   reference there is the named defect.

Fail-closed (an unreadable or missing manifest is a violation, not a skip) and
non-vacuous (zero command sequences FAILS). `scripts/test_check_secret_argv.py`
is the mutation fixture (10 cases): the gate must change PASS → FAIL on the
inline `-a` form, the `--requirepass` block-scalar form, the `--requirepass=`
form, a shell-string `-a` probe, and a k8s probe secret; and stay PASS on the
argv-free config-file form.

**Named limit (documented, not hidden).** Rule 2 is deliberately not applied to
a container `command:`. A command that *writes a requirepass config file from
the environment* legitimately contains the variable reference in its shell
script text while the secret value never enters any program's argv — this is the
accepted argv-free pattern. Rule 1 still applies to `command:`, so a
`--requirepass <value>` server flag is caught. The remaining R17-08 scope
(URL-embedded credentials, shell-expanded command strings outside these
manifests, and the redaction half) is tracked in `SPRINT-A2-credential-lifecycle`
and is **not** claimed by this note.

## 7. Implementation note (2026-10-16) — R17-03 HSTS emission half

The **HSTS emission clause** of R17-03 is enforced. R17-03 states "HSTS may be
emitted only for a verified public HTTPS origin, never as a substitute for
TLS"; the header is now emitted under exactly that condition and under no other.

- `Config.EmitsHSTS` (`internal/config/headers.go`) is the single decision:
  `TLS_ENABLED=true` **and** the effective `DEPLOYMENT_PROFILE` is `public`.
  An unparsable profile is treated as non-public (no header) — the failure mode
  of an unread posture must not be an emitted security claim. `Config.Validate`
  rejects such a profile at load, so this is belt-and-braces for a
  directly-constructed `Config`.
- `securityHeadersMiddleware` (`internal/web/server.go`) takes the
  `*config.Config` and evaluates the condition per request; a nil config emits
  no HSTS.
- Emitted value: `max-age=31536000; includeSubDomains` — **no** `preload`
  assertion, which is a commitment the operator has not made. Pinned by
  `TestHSTSHeaderValueIsBounded`.

**Evidence.** `internal/config/headers_test.go` (`TestEmitsHSTS` covers all
seven profile×TLS combinations) and `internal/web/middleware_test.go`
(`TestSecurityHeaders_HSTSOnlyOnVerifiedPublicTLS` drives the real middleware
for public+TLS, public-without-TLS, local+TLS, tailnet+TLS and nil config).
CI row `security-matrix-security-headers`; mutation-kill case
`hsts-emitted-for-non-public-origin` proves the negative suite flips PASS→FAIL
when the posture check is neutered.

**Named limit (documented, not hidden).** The rest of R17-03 — terminating
TLS 1.3+ at the public ingress, certificate/trust-chain rotation and validation,
rejecting TLS 1.2 and plaintext public connections, and the direct-backend
denial — is an **operator/proxy contract**, not a repo artifact: this binary
does not terminate public TLS (`TLS_ENABLED` is validated but does not change
either serving path — see §1). R17-02's fail-closed startup gate for an
external bind without an approved profile is likewise only partially enforced
today: `ValidateDeploymentBinding` covers the bind/profile match and the
`public`-profile `TRUSTED_PROXIES` requirement, but not "proven TLS termination
and direct-backend denial". Those remain open and are **not** claimed here.

## 8. Implementation note (2026-10-16) — R17-04 storage transit fail-closed

R17-04 requires authenticated encryption for DB/Redis **whenever either
connection leaves one verified isolated local boundary**, and requires startup
to fail if it is missing. Before this change `DB_SSLMode` and `RedisUseTLS`
were validated only as well-formed values (`config.go`); nothing coupled them to
the effective deployment profile, so a `tailnet`/`public` deployment could start
with plaintext DB and Redis and connect in the clear.

**New `Config.ValidateStorageTransit` (`internal/config/storage.go`)** is the
single decision, called from `Config.Validate` so it gates startup:

- `local` (the default) keeps the isolated-bridge/loopback exception named in
  R17-04 — the existing Compose posture is unchanged.
- Any nonlocal profile (`tailnet`, `public`) requires **`DB_SSLMODE=verify-full`**
  (certificate **and** hostname verification against a trusted CA — `disable`,
  `prefer`, unverified `require` and `verify-ca` are each refused) **and**
  `REDIS_USE_TLS=true` (verified server certificate and hostname; no
  skip-verify path exists in the tree).
- The check is a hard error naming the requirement and the offending variable,
  never a silent upgrade, so a misconfiguration stops the process instead of
  quietly connecting in the clear.

**Evidence.** `internal/config/storage_test.go`: `TestValidateStorageTransit`
covers the local exception (incl. the empty-profile default), the tailnet and
public positive cases, and the named negatives (`DB_SSLMODE` ∈ {disable, prefer,
require, verify-ca} each refused; Redis without TLS refused) asserting the error
names R17-04 and the variable; `TestValidateCouplesStorageTransitToProfile`
proves the coupling runs inside `Validate` (a tailnet config with plaintext
storage fails startup while every other rule is satisfied, and the same config
with `verify-full` + Redis TLS passes). CI row
`security-matrix-storage-transit`; mutation-kill case
`storage-transit-gate-always-permits` proves the negative suite flips PASS→FAIL
when the gate is neutered (18th case).

**Named limit (documented, not hidden).** This enforces the **posture** — the
configured transport mode for each connection — which is what a repo-side gate
can decide. It does not itself prove that a live `verify-full` handshake
succeeds against a particular CA or that Redis presents a valid certificate;
that remains a deployment/acceptance-harness assertion under R17-04's scenario 4
(disposable services), not a claim made here. The rest of spec 17 (profiles
beyond the bind check, the secret-source/rotation/redaction halves, and the
metrics/docs binding) remains open.

## 9. Implementation note (2026-10-16) — R17-05 no production default credentials

R17-05 requires nonlocal/production profiles to reject unset, empty, known
placeholder/default, reused or test credentials before startup, and states
plainly that "no checked-in sample is a production credential". The tree ships a
fixed, publicly-known Compose fallback (`changeme`) for the DB and Redis
passwords (`docker-compose.yml`: `${DB_PASSWORD:-changeme}` /
`${REDIS_PASSWORD:-changeme}`). That fallback is fine for a loopback-only local
run but was never rejected anywhere: `Validate` checked only length/character
class for the API keys and the JWT secret and checked the DB/Redis passwords not
at all, so a `public`/`tailnet` deployment could start with `changeme` and
connect using a credential any reader of this repository already knows.

**New `Config.ValidateNoProductionDefaultCredentials` (`internal/config/credentials.go`)**
is the single decision, called from `Config.Validate` so it gates startup:

- `local` (the safe default) is exempt — R17-05 explicitly keeps the documented
  Compose development posture working.
- Any nonlocal profile (`tailnet`, `public`) or an explicit `PRODUCTION_MODE`
  rejects a DB/Redis/MCP/IDE/JWT value that is unset, empty, a known
  placeholder/default, or an obvious sample form (`your-…`/`replace-…`, a
  template `<…>`, or a single repeated character). `IsPlaceholderCredential`
  is the reviewed literal set plus those sample forms.
- The check is a hard error naming the requirement (R17-05) and the offending
  variable, never a silent replacement, so a misconfiguration stops the process
  rather than starting with a default secret.

**Evidence.** `internal/config/credentials_test.go`:
`TestIsPlaceholderCredential` (the reviewed set + sample forms vs strong
values), `TestValidateNoProductionDefaultCredentials` (the local exemption
incl. the empty-profile default; named negatives for a placeholder DB password,
an empty Redis password and a placeholder JWT secret, each asserting the error
names R17-05 and the variable; a strong-credential positive), and
`TestValidateCouplesDefaultCredentialGateToProfile` proves the coupling runs
inside `Validate` (a public config with the Compose fallback fails startup while
every other rule is satisfied, and the same config with strong credentials
passes). CI row `security-matrix-default-credentials`; mutation-kill case
`default-credential-gate-always-permits` proves the negative suite flips
PASS→FAIL when the gate is neutered (19th case).

**Named limit (documented, not hidden).** This rejects a *known* placeholder /
default / obvious-sample credential; it is a reviewed floor, not an entropy
proof, and it does not verify uniqueness across environments or privilege
separation (separate DB/Redis accounts per environment) — those stay operator
contract items under R17-05/R17-06. The rest of spec 17 (secret-source/rotation/
redaction halves, metrics/docs binding) remains open.

## 10. Implementation note (2026-10-16) — R17-06 secret-source file path

R17-06 makes a managed secret store or a **runtime read-only mounted secret
file** the first-choice production source for DB/Redis credentials, the legacy
MCP/IDE keys and JWT signing material, and requires that file read be restricted
to the service identity (owner-only mode such as `0600`, private parent
directory) with group/world access prevented. It also says plainly: "do not
pretend existing env-only fields already support it."

**The gap was real, not a doc lag.** The tree had exactly two file-based secret
sources — `CREDENTIAL_REGISTRY_FILE` and `CREDENTIAL_VERIFIER_KEY_FILE` — and
**no way at all** to supply the five primary secrets from a file. `MCP_API_KEY`,
`IDE_API_KEY`, `JWT_SECRET` and `DB_PASSWORD` carried `required` env tags, so a
deployment that mounted its secrets as files could not even start; and
`REDIS_PASSWORD` had only an environment source. Environment injection was
therefore the *only* available path for every one of them — exactly the posture
R17-06 forbids as a production default.

**The rule is now one implementation, not two.** `internal/secretfile` holds the
single read-only-secret-file decision: the source must be a regular file, must
carry no group/world bits on POSIX hosts, and any error is fail-closed. The
credential-registry loader (`internal/auth/secrets.go`) now delegates to it
instead of carrying its own copy, so the registry path and the new config path
cannot drift apart.

**Each primary secret accepts an explicit `<NAME>_FILE` alternative:**
`MCP_API_KEY_FILE`, `IDE_API_KEY_FILE`, `JWT_SECRET_FILE`, `DB_PASSWORD_FILE`,
`REDIS_PASSWORD_FILE`.

- `Config.ResolveSecretSources` reads each configured file once at load; inline
  environment material wins when both are supplied (the same precedence the
  credential-registry loader already used), so an existing environment-only
  deployment is unaffected.
- A configured-but-unreadable, non-regular, group/world-accessible or empty file
  is a **hard error naming R17-06 and the variable**, never a silent fallback.
  A mounted secret's trailing newline is trimmed.
- The `required` env tags on those five fields are gone; `ValidateSecretSources`
  requires *a* source (inline or file) so "unset" still fails startup. This is
  what makes a file-only deployment possible to start without weakening the
  requirement that the secret be present.
- `Validate` resolves the file sources itself, so a directly-constructed
  `Config` — not only one that went through `Load()` — is on the same path, and
  the resolved value flows through **every existing gate unchanged**: the R17-04
  storage-transit coupling and the R17-05 placeholder/default-credential
  rejection both still fire on a value supplied from a file.
- `Config.Masked()` masks the six new file paths along with the values, so a
  config dump cannot disclose where a secret is mounted.

**Evidence.** `internal/secretfile/secretfile_test.go`
(`TestPermitsRestrictedOnly`, `TestReadAcceptsRestrictedFile`,
`TestReadRejectsNonRegularAndPermissive`) and
`internal/config/secret_sources_test.go` (`TestResolveSecretSourcesFromFiles` —
a file-only deployment resolves all five and then passes `Validate`;
`TestResolveSecretSourcesInlineWins`; `TestResolveSecretSourcesFailsClosed` —
absent / empty / non-regular / group-world-readable each stop startup;
`TestValidateRequiresASecretSource`; and
`TestSecretFileSourceFeedsTheExistingGates`, which proves a placeholder supplied
from a *file* is still rejected by R17-05). New CI row
`security-matrix-secret-sources` in `team-validation.yml` (20 jobs). New
mutation-kill case `secret-source-requirement-always-satisfied` proves the
negative suite flips PASS→FAIL when the source requirement is neutered.

**Named limit (documented, not hidden).** This adds the file *source* and its
access rule; it does not add a managed-store (KMS/Vault) client, an encrypted-
at-rest webhook-HMAC store, or rotation — the secret-source half of R17-06 for
`FALLBACK_API_KEY`/`OPENAI_API_KEY` (read directly from the environment by
`internal/mcp/vision_tools.go` and
`internal/guardrails/content_filter_backends.go`) is also still environment-only.
Rotation and the redaction half of R17-08 remain open in this spec.

## 11. Implementation note (2026-10-16) — R17-08 redaction half (request log)

R17-08 requires that credential material never reach logs, naming "authorization
headers" and secret values among the surfaces to redact. The request logger
(`internal/middleware/logging.go`, mounted on the real echo chain at
`internal/web/server.go:204`) wrote the **raw** `req.RequestURI` as the `uri`
attribute. A query string routinely carries exactly the material R17-08 forbids
— `?api_key=…`, `?token=…`, `?access_token=…`, a signed URL — so every such
value landed in the log in cleartext. The defect was live: driving the real
logger with `/api/search?token=<canary>&q=hi` wrote the canary verbatim
(recorded below).

**Fix.** New `middleware.RedactRequestURI` preserves the path (needed for
correlation and rate analysis) and replaces the query string with a visible
`?<redacted>` marker. The whole query is withheld rather than filtered by
parameter name: a name-based allowlist fails open on the first unlisted
parameter, and a secret-bearing query cannot be told from an ordinary one by
inspection. The logger now logs `RedactRequestURI(req.RequestURI)`.

**Evidence.** `internal/middleware/redact_test.go`:
`TestRedactRequestURI` pins the rule (no query unchanged, empty unchanged, query
withheld, `api_key=` withheld, bare `?` withheld, path verbatim) and
`TestRequestLoggerWithholdsQueryString` drives the **real** logger through the
real echo chain and asserts the log names `/api/search` and contains the marker
but never the canary — with a positive control that the record was actually
emitted (non-vacuous). Red-on-drift proven: neutering `RedactRequestURI` to
return the raw URI flips both tests PASS→FAIL (the log then shows
`uri="/api/search?token=<canary>&q=hi"`). New CI step "Request-log query
redaction (Spec 17 R17-08)" in the `security-matrix-secret-redaction` job
(`team-validation.yml`); new mutation-kill case
`request-uri-query-redaction-removed` (harness now 21 cases, all killed).

**Named limit (documented, not hidden).** This redacts the query string in the
request log. A credential embedded in the *path* is not redacted (paths are the
correlation key, and R17-08 keeps "paths to secret files" out of ordinary-caller
*responses* rather than log correlation fields). The DB-connect error path was
probed and is **not** a leak: `pgx v5` masks the password in its own error text
(`cannot parse …@x/d` renders the credential as `xxxxxx`; connect/dial errors
name only host and database). The remaining R17-08 scope — argv in manifests
(done, §6), shell-expanded command strings outside the manifests, and rotation —
stays tracked under `SPRINT-A2-credential-lifecycle`; this note claims only the
request-log query half.

## 12. Implementation note (2026-10-16) — R17-09 debug (pprof) binding

R17-09 requires that pprof stay **disabled by default** and, when enabled for
diagnosis, "bind to a local-only administrative endpoint with no public/tailnet
proxy route and time-box access". Only the "disabled by default" half was
honoured: `PPROF_ENABLED` gated a `startPProfServer(port)` that bound
`localhost:<port>` **unconditionally**, served the process-wide
`http.DefaultServeMux` (`http.ListenAndServe(addr, nil)`), and had **no
expiry** — once enabled it served for the life of the process, and any handler
registered on the default mux anywhere in the process was reachable through it.

**Fix.** New `internal/config/debug.go` is the single decision, enforced from
`Config.Validate` at startup and consulted again when the listener starts:

- `ValidateDebugEndpoint` refuses a non-loopback `PPROF_BIND_HOST`, an
  out-of-range port, an **enabled endpoint with no `PPROF_UNTIL`**, an already
  elapsed window, a malformed window (never a silent unbounded window), and —
  outside the `local` profile — an enabled endpoint without
  `PPROF_NO_PROXY_ROUTE_ATTESTATION`. A disabled endpoint validates to nil, so
  the default posture is unchanged.
- `DebugEndpointActive(now)` is the runtime gate; `PProfDeadline` parses the
  RFC3339 window.
- `cmd/server/main.go` now binds `cfg.PProfBindAddr()`, refuses to start
  without a bounded window, serves an **explicit mux holding only the pprof
  endpoints** (never `DefaultServeMux`), and shuts the listener down the moment
  the window closes rather than waiting for the next request.

**Evidence.** `internal/config/debug_test.go`: `TestPProfDeadline` (unset /
valid / malformed), `TestValidateDebugEndpoint` (disabled-nil, local
loopback+window positive, non-loopback refusal, missing-window refusal, expired
refusal, malformed refusal, nonlocal-attestation requirement),
`TestDebugEndpointActive` (within window / elapsed / unconfigured / disabled),
and `TestValidateCouplesDebugPostureToStartup` proving the gate runs inside
`Validate` (a config that satisfies every other rule still fails startup on a
non-loopback debug endpoint). New CI row `security-matrix-debug-binding` in
`team-validation.yml`; new mutation-kill case
`debug-endpoint-posture-gate-always-permits` (harness now 22 cases) proves the
negative suite flips PASS→FAIL when the posture gate is neutered.

**Named limit (documented, not hidden).** This enforces the debug endpoint's
*bind* posture and access window. The `public`-profile `/metrics`, `/docs` and
`/openapi.yaml` auth re-decision (R17-09's first sentence, and Spec 12 §3.2)
remains the owner's scrape-contract call recorded in Spec 12 §7; the current
posture (authenticated-by-default in production via the exposure registry, with
`/metrics` declared public per the recorded §3.2 deferral) is unchanged here.

## 13. Implementation note (2026-10-16) — R17-07 verifier-key entropy floor

R17-07 requires the credential-registry verifier key to be generated out of
band with **>=256 random bits**, and states plainly that "a length check is not
an entropy proof". The tree enforced only a **16-character length floor**
(`config.go` `validateCredentialRegistry` and `auth` `buildSnapshot`), which
accepts `strings.Repeat("x", 64)` — a key with a single distinct byte and
therefore **zero** real entropy — and any 16-character key drawn from a tiny
alphabet. The verifier key is the HMAC key behind every stored credential
verifier (`auth.Digest`), so a guessable key makes the whole credential registry
forgeable: anyone who can predict the key can mint a verifier digest for a
secret of their choosing and resolve as any principal.

**Fix.** New `internal/keymaterial` holds the single decision, shared by the two
loaders that enforce it — the credential-registry loader (`internal/auth`) and
the configuration gate (`internal/config`) — so they cannot drift apart.
`VerifierKeyEntropyBits` is a deliberate **upper** bound (alphabet size ×
effective length, where the effective length is the shortest repeating block so
a pattern like `abcabcabc` is credited one block, not its full length), so it can
only reject a key that is provably weak; it never blesses a key as strong.
`ValidateVerifierKeyEntropy` refuses a key below the floor with an error naming
R17-07 and the observed bound.

- **`internal/auth/registry.go`:** `buildSnapshot` now refuses a key below
  `MinVerifierKeyBits` (128) at the load boundary, for every profile — an
  unusable key can never silently produce a registry that fails open. This is
  the path a file-sourced `CREDENTIAL_VERIFIER_KEY_FILE` flows through, so a
  weak file key is caught even though the config gate sees only the inline
  value.
- **`internal/config/config.go`:** `validateCredentialRegistry` keeps the
  16-character length floor as the cheap first gate and adds the strength
  floor. `local` (the documented development posture) keeps the 128-bit floor;
  a nonlocal or production deployment must clear the **256-bit** bound R17-07
  names, so a weak inline key fails startup rather than becoming the HMAC key
  behind every stored verifier. The gate runs inside `Config.Validate`.

**Evidence.** `internal/keymaterial/keymaterial_test.go`
(`TestVerifierKeyEntropyBits` — empty/whitespace/single-byte/periodic/strong;
`TestValidateVerifierKeyEntropyRejectsWeak` — the 64-byte single-char key is
refused, a 147-bit key clears the local floor but not the production bound, the
error names R17-07); `internal/auth/registry_failclosed_test.go`
(`TestNewRejectsZeroEntropyVerifierKey` — the load boundary refuses
`strings.Repeat("x", 64)` and accepts a 256-bit key);
`internal/config/verifier_key_test.go`
(`TestValidateCredentialVerifierKeyStrength` local/production split;
`TestValidateCouplesVerifierKeyStrengthToStartup` proving the gate runs inside
`Validate`). New CI row `security-matrix-verifier-key-strength` in
`team-validation.yml` (22 jobs); new mutation-kill case
`verifier-key-entropy-floor-removed` (harness now 23 cases) proves the negative
suite flips PASS→FAIL when the entropy floor is neutered.

**Fix-forward (cause, not gate).** `TestRegistryRotationOverlapBounds` used the
fixture `new-verifier-key-1234567890` (117 bits), which the new floor correctly
refused; the **fixture** was strengthened, not the gate relaxed.

**Named limit (documented, not hidden).** This is a floor on the *configured*
key material, not an oracle: the estimator is an upper bound, so a key that
passes could still be weaker than its alphabet suggests. It does not generate or
rotate the key, and it does not cover the JWT signing secret (already gated by
`ValidateJWTSecret`) or the webhook HMAC secrets (their storage-at-rest and
rotation half remain open in R17-06/R17-10).

## 13. Implementation note (2026-10-16) — R17-10 secret inventory

R17-10 requires a maintained **secret inventory** — "owner, consumer, source,
expiry, last rotation, revocation procedure — **without values**". The tree had
none: the phrase appeared only in this spec and in the SPRINT-A2 entry-gate
checklist as an unchecked box, so the requirement was genuinely ABSENT, not a
doc lag. An inventory that can silently fall out of date is not an inventory.

**`docs/security/SECRET-INVENTORY.md`** is the inventory. It carries one row per
secret with the six R17-10 fields and **no values**, and it records the honest
gaps rather than implying coverage that does not exist:

- `TEAM_ENCRYPTION_KEY`, `FALLBACK_API_KEY` and `OPENAI_API_KEY` are
  **environment-only** today — R17-06's file-source rule is not yet applied to
  them (the R17-06 residual), so their Source cell says so.
- `WEBHOOK_HMAC_SECRET` persists as the `secret_hmac` column of
  `webhook_configs`; encrypted-at-rest storage remains an open R17-06 item and
  the row does not claim otherwise.
- `MIGRATION_DSN` / `MIGRATION_DATABASE_URL` (the one-shot migration job's DSN)
  and the `TLS_CERT_PATH`/`TLS_KEY_PATH` deployment material are inventoried
  alongside the config-struct secrets.

**The inventory is enforced, not just written.**
`scripts/gates` is not a directory in this repo; the gate lives at
`scripts/check_secret_inventory.py` beside its siblings. It fails closed when:

- a secret-bearing environment name discovered in the Go tree — an
  `env:"NAME"` tag or an `os.Getenv`/`getEnv` read matching the reviewed secret
  pattern — has **no** inventory row (a newly added secret must be documented);
- a row is missing any of the six required cells;
- a row contains a pasted **secret** value assignment (`NAME=<value>` for a
  secret-shaped name) — the inventory holds policy, not credential material;
- the scan discovers **zero** secret names (non-vacuity).

A `<NAME>_FILE` source is documented by its base `<NAME>` row — the file variant
is an alternative source for the same secret, not a second secret.

**Evidence.** `scripts/test_check_secret_inventory.py` (8 cases) proves the gate
fails on a missing row, an empty required cell, a pasted secret value and an
empty scan scope, passes on a complete inventory, treats a documented non-secret
flag (`DB_SSLMODE=verify-full`) as policy rather than a value, maps
`JWT_SECRET_FILE` to the `JWT_SECRET` row, and fails closed on a missing
inventory file — so a green run cannot come from a gate that never exercises its
assertion. The gate runs live: **16 secret names covered, 15 rows**. Wired into
`regression-guard.yml` as the `Secret inventory coverage (Spec 17 R17-10)` step.

**Named limit.** The coverage gate proves *no secret-bearing environment name in
the tree is absent from the inventory*. It cannot discover a non-environment
secret such as `WEBHOOK_HMAC_SECRET` (documented, but not tree-discoverable), nor
a secret hard-coded in a Go literal — that class is covered by the secret
scanner and the R17-11/R17-08 gates, not here. The inventory records what must be
rotated and revoked; it does not itself rotate or revoke.
