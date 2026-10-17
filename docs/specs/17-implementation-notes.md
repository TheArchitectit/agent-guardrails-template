# Spec 17 implementation notes (part 1 of 2)

Companion to [`17-deployment-tls-and-secret-boundary.md`](17-deployment-tls-and-secret-boundary.md).

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
either serving path — see Sec. 1). R17-02's fail-closed startup gate for an
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

