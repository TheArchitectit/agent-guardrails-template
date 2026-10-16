# Spec 17 implementation notes (part 2 of 2)

Companion to [`17-deployment-tls-and-secret-boundary.md`](17-deployment-tls-and-secret-boundary.md).

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

## 14. Implementation note (2026-10-16) — R17-02 fail-closed nonlocal ingress posture

R17-02 requires that startup fail — nonzero exit, no app socket — "on an
external bind or publish without an approved profile, on `public` without
proven TLS termination and direct-backend denial, or on `tailnet` without
verified private ingress plus encrypted authenticated transport", and adds the
clause that decides the whole shape of this gate: "A true `PRODUCTION_MODE`
flag, `TLS_ENABLED=true` or presence of certificate paths alone never satisfies
this gate."

Only the *bind* half was enforced. `ValidateDeploymentBinding` (R17-01) admitted
a `public` deployment because `TRUSTED_PROXIES` was non-empty, and a `tailnet`
deployment because the bind host was not a wildcard. Neither check says anything
about TLS termination, backend isolation or transport encryption, so a `public`
profile could start on a plaintext, directly reachable backend with every other
gate green — exactly the "TLS_ENABLED is not evidence" failure R17-02 names.
The audit row recorded the gap honestly: `17-R17-02 | W | ValidateDeploymentBinding;
no real-listener negative exercised`.

**The evidence form is the requirement.** §4 states plainly that "an
operator-supplied JSON assertion without an actual external reachability/
certificate test is NOT evidence". So the gate does not accept a free-text env
attestation (the shape R17-09 uses for the debug endpoint). A nonlocal profile
must present a **commit-bound preflight record** naming the *proven* results of
its probes:

- `public` requires `tls_termination.proven` **and** `direct_backend_denied.proven`;
- `tailnet` requires `private_ingress.proven` **and** `encrypted_transport.proven`;
- every proven result must carry a non-empty `detail` — a proven flag with no
  observation is an assertion, not evidence;
- `negative_probes` must be non-empty and every entry non-empty — a record with
  no negative probe proves only the happy path;
- `contract_version` must match this build (`guardrail-ingress-preflight/v1`),
  `profile` must equal the effective profile, and `commit` must be a full
  40-hex revision; when the binary carries a VCS stamp the two must agree, so a
  stale preflight is refused rather than reused.

`TLS_ENABLED` is deliberately **not** consulted: for the `public` profile TLS is
terminated at the proxy, so this process's own flag would be the false-positive
the requirement forbids. The proxy's termination is proven by the record instead.

**Custody is checked.** A record whose file is missing, not a regular file,
empty, or group/world-writable is refused — a record another identity can
rewrite is not evidence.

**Evidence.** `internal/config/ingress_test.go`:
`TestValidateIngressPosture` (local/empty pass with no record; public and tailnet
fail without one; both pass with a fully proving record; and the named negatives
— missing `tls_termination`, unproven `tls_termination`, proven-without-detail,
missing `direct_backend_denied`, missing `private_ingress`, missing
`encrypted_transport`, record for the wrong profile, short commit, no negative
probes, empty negative probe, wrong contract version, unknown field refused, and
malformed/empty JSON), `TestValidateIngressPostureRejectsWeakRecordFile`
(group/world-writable, missing, directory), `TestValidateCouplesIngressPostureToStartup`
(the gate runs inside `Validate`; and pins that `TLS_ENABLED` stays false for the
public profile), and `TestIngressPreflightPath`. New CI row
`security-matrix-ingress-posture` (26 jobs). New mutation-kill case
`ingress-posture-gate-always-permits`; the harness is now **24 cases, all
killed**.

**Named limit.** This gate enforces the *presence and shape of proven evidence*;
it does not itself perform the external reachability or certificate test — that
is the operator-run preflight the record reports, exactly as §4 specifies. A
`local` deployment is not constrained here: its exposure is bounded by the
R17-01 loopback/bridge bind rule. Post-start drift re-verification ("halt
protected traffic until repaired") remains future integration work alongside the
readiness contract.

## 15. Implementation note (2026-10-16) — R17-08 redaction half (webhook delivery path)

The request-log half (§11) closed one log surface. A second, independent surface
carried the same class of secret and was still live: the **webhook delivery
path**. R17-08 requires credential material never reach "errors, logs, audit,
traces, metrics, crash reports, health output and MCP/REST resources", and a
webhook target is operator-supplied and routinely carries credential material in
its query string — `?token=***`, `?api_key=***`, a signed URL, a shared-secret
query parameter.

Two sites wrote it:

- `internal/notifications/webhook_dispatcher.go` `deliver` recorded the raw
  `wh.URL` as the `url` attribute of the delivery-failure log line.
- the error from `http.Client.Do` is a `*url.Error` whose `Error()` text embeds
  the **FULL** target URL, query string included. That text was written to the
  same log line's `error` attribute, stored on the delivery row
  (`recordDelivery`), and returned by the `get_webhook_deliveries` tool.
  `SendTestEvent` stored and returned the same shape through `test_webhook`.

**Verified live before the fix.** Driving the real `deliver` path with a
loopback target carrying a canary in its query (`…/hook?token=***<canary>`)
wrote the canary verbatim, twice over:

```
level=WARN msg="webhook delivery failed after retries" webhook_id=wh1
  url="http://127.0.0.1:38033/hook?token=***<canary>"
  error="request failed: Post \"http://127.0.0.1:38033/hook?token=***<canary>\":
         refusing webhook delivery to 127.0.0.1 (ipv4 loopback)"
```

**Fix.** New `internal/notifications/redact.go` is the single decision for URL
redaction on log and error surfaces: `RedactURLsInText` withholds the query
string of every absolute http(s) URL in a piece of text (including one embedded
in a wrapped error) and leaves the rest of the text untouched, so an ordinary
error message is unaltered. `deliver` now logs `RedactURLForLog(wh.URL)` and
`RedactURLsInText(lastErr.Error())`; `SendTestEvent` stores
`RedactURLsInText(err.Error())`. Scheme, host and path survive for correlation —
the same rule the request logger applies to `RequestURI` (§11). The whole query
is withheld rather than filtered by parameter name, because a name-based
allowlist fails open on the first unlisted parameter.

**Evidence.** `internal/notifications/redact_test.go`:
`TestRedactURLsInText` pins the rule (no URL unchanged, empty unchanged, bare URL
unchanged, query withheld, URL inside wrapped error text redacted, http scheme
covered, two URLs both redacted, a non-http scheme untouched);
`TestWebhookDeliveryFailureLogWithholdsTargetQuery` drives the **real** `deliver`
path through the real egress dial guard and asserts the log names the path, the
marker is present, the canary is absent from both the log and the stored error,
with a positive control that the failure record was emitted (non-vacuous); and
`TestWebhookTestEventErrorWithholdsTargetQuery` does the same for `SendTestEvent`
— the path the `test_webhook` / `get_webhook_deliveries` tools expose.
Red-on-drift proven: neutering `RedactURLsInText` to return its input flips both
path tests PASS→FAIL (the canary reappears in the log and the returned error).
New CI step "Webhook target-URL query redaction (Spec 17 R17-08)" in the
`security-matrix-secret-redaction` job (`team-validation.yml`); new mutation-kill
case `webhook-target-url-query-redaction-removed` (harness now **25 cases, all
killed**).

**Named limit (documented, not hidden).** A credential carried in the URL
*path* is not redacted (paths are the correlation key, mirroring §11). The
`WEBHOOK_HMAC_SECRET` column, encrypted-at-rest storage and rotation remain the
open R17-06/R17-10 items already recorded in `docs/security/SECRET-INVENTORY.md`
— this note claims only the delivery-path query half.
