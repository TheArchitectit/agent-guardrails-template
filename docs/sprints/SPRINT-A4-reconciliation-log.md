# SPRINT-A4 — Reconciliation Log (2026-10-04 .. 2026-10-05)

> Historical reconciliation records for [SPRINT-A4](SPRINT-A4-ci-security-gates.md).
> Moved here 2026-10-17 so the sprint doc stays within the 500-line limit
> (Spec 19 R19-11); content is unchanged.

**Related:** [SPRINT-A4-ci-security-gates.md](SPRINT-A4-ci-security-gates.md) | [Spec 19](../specs/19-phase0-ci-security-gates.md)

---

## Reconciliation 2026-10-04 — verified against source at commit 1c838f4

Status vocabulary: **WIRED** (code exists, no exact live test), **EXERCISED**
(exact test ran, cited), **NOT_EXERCISED** (live dependency skipped), **NOT_RUN**
(tool/runner unavailable). No item below is ACCEPTED. `NOT_RUN` rows here mean
the CI runner / required-check run was not available to verify against.

- Same-commit required matrix — **NOT_RUN**.
- Negative-control + mutation-kill rows — **NOT_RUN**.
- Registry / secrets / SSRF / deploy / assets matrix rows — **NOT_RUN**.
- Package coverage + nonzero test floor — **WIRED**.
- Docs gate (500-line max / internal links) — **WIRED** (trigger incomplete: does not cover every protected push/release path).
- Runner trust / isolation — **NOT_RUN**.
- Provenance (full-SHA actions / digest images) — **NOT_RUN**.
- Release evidence bundle — **NOT_RUN**.

---

## Reconciliation 2026-10-05 — CI security-matrix rows WIRED/EXERCISED

Three matrix rows are now WIRED as stable required-check jobs in
`.github/workflows/team-validation.yml` and EXERCISED locally (Windows subset;
CI runs on the supported `ubuntu-latest` runner):

- `security-matrix-web-failclosed` — web fail-closed middleware (configured-invalid
  registry denies all protected traffic) + registered/legacy REST authorization
  (R19-03 / R19-05). **WIRED** + **EXERCISED**, command:
  `cd mcp-server && go test ./internal/web -run 'TestRegistryConfiguredInvalidDeniesAllProtectedTraffic|TestAPIKeyAuth' -count=1` → ok.
- `security-matrix-mcp-fullpath-authz` — MCP full-path authorization over the real
  StreamableHTTP endpoint (allow AND deny + zero-side-effect), including the live
  revocation-propagation bound test `TestLiveRevocationPropagationBound`, which
  authorizes a credential on both the real web and MCP httptest paths, revokes it,
  and asserts both paths deny within `auth.RevocationPropagationBound` (R19-03).
  **WIRED** + **EXERCISED**, command:
  `cd mcp-server && go test ./internal/mcp -run 'TestStreamableHTTP|TestLiveRevocationPropagationBound' -count=1` → ok.
- `security-matrix-registry-failclosed` — registry fail-closed and lifecycle rows
  (R19-05). **WIRED** + **EXERCISED**, command:
  `cd mcp-server && go test ./internal/auth -run 'TestRegistry|TestRevocation|TestPermitsRestrictedOnly|TestSecretFile' -count=1` → ok.

The workflow now also triggers on `push` to `main`/`master` (previously only
`pull_request` + `workflow_dispatch`). Rows with no implemented suite (deploy/assets,
runner trust, provenance, release bundle) remain **NOT_RUN** and are deliberately
not faked as CI jobs.

---

## Reconciliation 2026-10-05 (part 2) — secrets, SSRF, and mutation-kill rows

Three further rows are now WIRED as stable required-check jobs in
`.github/workflows/team-validation.yml` and EXERCISED locally (Windows subset;
CI runs on the supported `ubuntu-latest` runner):

- `security-matrix-secret-redaction` — secret-leak / redaction (R19-06). Drives
  real paths with a single canary planted in every secret-bearing config field
  and in nested tool arguments: a `guardrail://config` resource read, an
  authorized `guardrail_init_session` call that persists a real audit decision
  record, and a denied error response. Asserts the canary is absent from logs,
  audit records, error responses, and serialized output; the audit capture is a
  real `AuditStoreInterface` insert, so the check is on the operator-visible
  record, not a helper unit. **WIRED** + **EXERCISED**, command:
  `cd mcp-server && go test ./internal/mcp -run 'TestSecretLeakRedactionAcrossRealPaths|TestArgumentPrivacy|TestGuardrailConfigOmitsSecrets|TestConfigPublicViewOmitsSecrets|TestSafeResourceIDRejectsSecrets' -count=1` → ok.
- `security-matrix-webhook-ssrf` — webhook URL SSRF guard (R19-07). The
  dispatcher POSTs to the stored URL, so the configuration-time guard is
  exercised against loopback (`localhost`, `127.0.0.1`, `127.0.0.2`, `[::1]`,
  `[::ffff:127.0.0.1]`), link-local and cloud metadata (`169.254.169.254` and
  `[::ffff:169.254.169.254]`, `[fe80::1]`), private (`10/8`, `172.16/12`,
  `192.168/16`), unique-local (`[fd00::1]`), unspecified, and non-http(s)
  schemes; a public `https` literal IP is the positive control. A policy-denied
  destination is the failed outcome. **WIRED** + **EXERCISED**, command:
  `cd mcp-server && go test ./internal/mcp -run 'TestValidateWebhookURL' -count=1` → ok.
  Limitation (documented in code): resolution happens at configuration time, so
  this removes the trivial and stored-internal-address classes but does **not**
  defend against DNS rebinding between configuration and delivery; there is no
  dialer-level re-check.

  Addendum 2026-10-06: commit bc7f523 added delivery-time enforcement —
  `internal/notifications/webhook_dispatcher.go` now re-validates the resolved
  destination at dispatch (public-IP check, redirect refusal, controlled
  dialer), with `webhook_delivery_ssrf_test.go` covering loopback, link-local,
  private, metadata, and redirect rejection at delivery time. The limitation
  above predates that commit; `go test ./internal/notifications -count=1`
  passes locally on Windows.
- `security-matrix-mutation-kill` — mutation-kill control (R19-04). A hermetic
  harness (`internal/mutationkill`) copies the module to a scratch directory,
  confirms the real negative suite PASSES against protected source, injects a
  deterministic bypass, and requires the SAME suite to FAIL. Two named bypasses
  are covered: webhook SSRF guard always permits (kills
  `TestValidateWebhookURL`) and registry fail-closed swallows the load error
  (kills `TestLoadFromSourcesConfiguredInvalidFailsClosed`). Both flip
  PASS→FAIL. The scratch copy means the protected tree is never left mutated, so
  the R19-04 "restore before publishing PASS" requirement is satisfied
  structurally. **WIRED** + **EXERCISED**, command:
  `cd mcp-server && go test ./internal/mutationkill -run 'TestNegativeSuitesKillInjectedBypass' -count=1` → ok.
  Limitation (honest scope): this is a textual-mutation harness over copies, not
  a Go-mutation framework (e.g. go-mutesting); it proves the named suites detect
  the injected bypasses, not a mutation-score across the codebase. At the time of
  this reconciliation two bypasses were covered; the remaining R19-04 named
  bypasses were still **NOT_RUN** (see part 3).

Rows that still have no implemented suite (deploy/readiness/migration,
static-assets/CORS/proxy, runner trust, provenance, path filters, release
contract, release evidence bundle) remain **NOT_RUN** and are not faked as CI
jobs. The full-history Gitleaks secret scan now has a script (see part 3).

---

## Reconciliation 2026-10-05 (part 3) — remaining mutation kills and full-history Gitleaks

The `security-matrix-mutation-kill` row (R19-04) now exercises **six** injected
bypasses, each proven to flip a real negative suite PASS→FAIL (scratch copy, so
the protected tree is never left mutated). Command:
`cd mcp-server && go test ./internal/mutationkill -count=1` → ok.

- `webhook-ssrf-guard-always-permits` → kills `TestValidateWebhookURL`. **EXERCISED**.
- `registry-failclosed-swallows-load-error` → kills `TestLoadFromSourcesConfiguredInvalidFailsClosed`. **EXERCISED**.
- `auth-bypass-always-permits` → `auth.Decide` returns allow unconditionally;
  kills `TestDecideScopeRoleResourceIntersection`. **EXERCISED**.
- `mcp-authz-tools-call-always-authorizes` → `authorizeToolCall` returns nil
  unconditionally; kills `TestStreamableHTTPToolsCallAuthorizesAndDenies` over the
  real StreamableHTTP path. **EXERCISED**.
- `secret-fixture-leaks-into-output` → `readConfigResourceContents` marshals the
  raw config instead of `PublicView()`; kills `TestSecretLeakRedactionAcrossRealPaths`.
  **EXERCISED**.
- `broken-internal-doc-link` → the docs-gate link checker
  (`scripts/check-doc-links.sh`, the same script CI now runs) flips PASS→FAIL when
  a broken relative `.md` link is injected into a scratch tree. **EXERCISED**.
- `wildcard-production-CORS-acceptance` — **NOT_RUN**: no negative suite exists to
  catch it. `internal/web/server.go` substitutes localhost origins when the CORS
  allow-list is `*` and `ProductionMode` is set, but no test drives the CORS
  middleware or asserts a rejected foreign `Origin`; there is no `internal/web`
  CORS/origin test at all, so a mutation here would be caught by nothing.

---
