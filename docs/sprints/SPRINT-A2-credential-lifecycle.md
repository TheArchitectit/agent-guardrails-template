# Sprint: Credential Lifecycle and Secret Sources

**Sprint ID:** SPRINT-A2
**Source Spec:** 16 — Gate 3 (R16-09) + Spec 17 secret source / least privilege (R17-06 through R17-08)
**Priority:** P1 (Blocking)
**Estimated Effort:** ~2–3 days
**Status:** PARTIALLY IMPLEMENTED (no item ACCEPTED) — see Reconciliation 2026-10-04
**Depends On:** SPRINT-A0 (registry fail-closed, config secrecy), SPRINT-A1 (principal authorization)
**Blocks:** SPRINT-A3 (deployment readiness assumes secret sources), SPRINT-A4 (CI secret gates)

---

## Reconciliation 2026-10-04 — verified against source at commit 1c838f4

Status vocabulary: **WIRED** (code exists, no exact live test), **EXERCISED**
(exact test ran, cited), **NOT_EXERCISED** (live dependency skipped), **NOT_RUN**
(tool/runner unavailable). No item below is ACCEPTED.

- Revocation propagation bound — **WIRED** (in-process only): `internal/auth/registry_lifecycle_test.go:26`; no cross-process/multi-instance denial proven.
- Atomic reload — **EXERCISED (unit)**: `internal/auth/registry_lifecycle_test.go:64`.
- Secret-file source + ACL — **EXERCISED (unit)**: `registry_lifecycle_test.go:215`, `registry_lifecycle_test.go:266`.
- Webhook HMAC at rest — **NOT_EXERCISED**.
- No credentials via argv / healthcheck flags — **NOT_EXERCISED**.
- Verifier key ≥256-bit generation — **WIRED**.

---

## Problem Statement

Credential operations lack defined generation, issuance, distinct principal
assignment, expiry, overlap/rotation, revocation, and atomic reload or immutable
replacement. Verifier-key rotation and operator audit are undefined. Secrets
are sourced from environment fields without a file/secret-provider path, webhook
HMAC material persists in plaintext `secret_hmac`, credentials may appear in
process argv (e.g. Redis `redis-cli -a` healthcheck), and observability paths
can leak nested secrets. Lifecycle claims must not exceed implemented and
protected operations.

**Why:** No approved lifecycle bounds or first-choice secret source; process
and observability paths are not secret-free.
**Where:** Credential/registry operations, `internal/database/webhooks.go`
HMAC storage, healthcheck probes, log/audit/error paths (Spec 16 R16-09;
Spec 17 R17-06–08).

---

## Entry Gate

Before starting this sprint:

- [ ] SPRINT-A0 and SPRINT-A1 exit gates are complete (fail-closed registry, secrecy, principal/audit contract).
- [ ] Secret inventory skeleton exists (owner, consumer, source, expiry, last rotation, revocation procedure) **without values**.
- [ ] Baseline: `go test ./internal/auth ./internal/web ./internal/mcp ./internal/database -count=1`.
- [ ] Synthetic credentials only; no production secrets or production DB.

---

## Exit Gate

This sprint is complete only when all of the following hold:

- [ ] Approved key generation, issuance, distinct principal assignment, expiry, overlap/rotation, revocation, atomic reload or immutable deployment replacement, verifier-key rotation, and operator audit are defined **and** implemented/protected (no lifecycle API claimed until it exists). — partial: reload/rotation/revocation helpers exist and are unit-tested; **NOT_EXERCISED** end-to-end.
- [ ] Measured maximum revocation propagation delay is set and published; revoked keys deny new web and MCP calls within the approved bound (target from Spec 16/17: ≤60s tested denial). — **WIRED**, in-process only (`registry_lifecycle_test.go:26`); cross-process denial **NOT_EXERCISED**.
- [ ] A failed reload or bad replacement never enlarges access or restores privileged legacy behavior. — **EXERCISED (unit)** (`registry_lifecycle_test.go:64`).
- [ ] File/secret-provider configuration path exists (do not pretend env-only fields already support it). — **EXERCISED (unit)** (`registry_lifecycle_test.go:215`).
- [ ] Production secret source is managed store or runtime read-only mounted secret files (POSIX `0600` / private parent; restrictive Windows ACL equivalent). — **EXERCISED (unit)** for the POSIX ACL path (`registry_lifecycle_test.go:266`); Windows ACL equivalent **NOT_EXERCISED**.
- [ ] Webhook HMAC: encrypted-at-rest storage or managed secret reference; read/list APIs restricted; rotation without exposing the secret. — **NOT_EXERCISED**.
- [ ] No credentials via argv, shell-expanded commands, healthcheck flags, or URLs (Redis `redis-cli -a` probe replaced). — **NOT_EXERCISED**.
- [ ] Nested tool arguments, DSNs, auth headers, registry JSON, webhook signature keys, `FALLBACK_API_KEY`, `OPENAI_API_KEY`, and secret file contents redacted from errors, logs, audit, traces, metrics, crash reports, health output, and MCP/REST resources. — **WIRED** (argument/resource helpers, S-A0).
- [ ] Opaque credential IDs used for attribution; secret file paths not disclosed to ordinary callers. — **WIRED**.

---

## Scope Boundary

```
IN SCOPE (may modify):
  - Credential lifecycle: generate/issue/expire/rotate/revoke/reload or replace
  - Verifier-key rotation and operator audit of lifecycle events
  - Secret source/file-provider loading path and permission checks
  - Webhook HMAC encrypted-at-rest or managed reference + restricted APIs
  - Health/probe and observability redaction (argv, logs, errors, traces, metrics)
  - Tests: lifecycle bounds, revocation propagation, secret-free process/observability

OUT OF SCOPE (DO NOT TOUCH):
  - Registry fail-closed load semantics (SPRINT-A0; verify only)
  - Principal intersection matrix (SPRINT-A1; verify only)
  - Deployment profiles, TLS termination, Compose production defaults (SPRINT-A3 / R17-01–05)
  - CI image/action provenance and secret-scan gates (SPRINT-A4 / R17-11, R19)
  - External KMS product selection (optional later; not this sprint)
  - Claiming lifecycle APIs that are not implemented and protected
```

---

## Numbered Tasks

### Task 1 — Credential operations and bounds (R16-09)

**Action:** Define and implement approved key generation, issuance, distinct
principal assignment, expiry, overlap/rotation, revocation, atomic reload or
immutable deployment replacement, verifier-key rotation, and operator audit.
Publish a measured maximum revocation propagation delay. Reject a bad
replacement without restoring privileged access. Do not claim a lifecycle API
exists until implemented and protected.

**Acceptance criteria:**
- Revoked key → new web and MCP calls deny within the approved bound (≤60s target; measured and recorded).
- Overlapping replacement works only for its intended scopes and role.
- Failed reload / bad replacement never enlarges access.
- Verifier-key rotation is covered with operator audit (opaque events).
- Lifecycle operations are authenticated and authorized (not open endpoints).

**Expected test commands** (from `mcp-server/`):
```sh
go test ./internal/auth ./internal/web ./internal/mcp -run 'TestCredential|TestRotate|TestRevoke|TestLifecycle|TestVerifier' -count=1
go test ./internal/auth ./internal/web ./internal/mcp -count=1
```

### Task 2 — Secret source and least privilege (R17-06)

**Action:** Add a file/secret-provider configuration path as a first-choice
production source for DB/Redis, MCP/IDE legacy keys, `CREDENTIAL_REGISTRY_FILE`,
`CREDENTIAL_VERIFIER_KEY`, JWT signing material, webhook HMAC secrets,
`FALLBACK_API_KEY`, `OPENAI_API_KEY`, and TLS private keys. Restrict file read
to the service identity and rotation operator (`0600` / private parent;
restrictive Windows ACL). Never mount into unrelated services; exclude secret
volumes from backups/log exports unless encrypted and access-controlled.
Environment injection, if unavoidable during migration, is trusted-runtime only
with no diagnostic dumps. No plaintext secrets in committed `.env`, Compose
files, or images. Webhook HMAC (`internal/database/webhooks.go`) must use
encrypted-at-rest storage or a managed secret reference with restricted
read/list APIs and rotation without exposure.

**Acceptance criteria:**
- Missing secret file / permissive ACL / empty configured registry fails at the prescribed boundary.
- Webhook read/list never returns HMAC values; rotation works with overlapping receiver verification under a tested bound.
- No plaintext secrets in committed `.env`, Compose, or images (fixture scan).
- File paths of secrets not disclosed to ordinary callers.

**Expected test commands:**
```sh
go test ./internal/config ./internal/database ./internal/auth -run 'TestSecretSource|TestSecretFile|TestWebhookHmac|TestHmac' -count=1
go test ./internal/... -count=1
```

### Task 3 — Registry and identity secrets (R17-07, related)

**Action:** Treat registry verifiers and verifier key as separate sensitive
inputs; load and validate the entire configured registry before listeners
(SPRINT-A0). Generate verifier key out of band with ≥256 random bits (length
check is not an entropy proof). Never put raw credentials in the registry or
return verifier material. Keep Spec 11 intersection on web and MCP; keep
`guardrail://config` as reviewed non-secret projection.

**Acceptance criteria:**
- Verifier key generation records entropy source ≥256 bits; invalid/short key rejected at load.
- Registry never contains raw credentials; verifier material never returned.
- No legacy unrestricted fallback on registry failure.

**Expected test commands:**
```sh
go test ./internal/auth ./internal/mcp -run 'TestVerifier|TestRegistry|TestLoadFromSources' -count=1
```

### Task 4 — Secret-free process and observability (R17-08)

**Action:** Do not pass credentials via argv, shell-expanded command strings,
healthcheck flags, or URLs. Replace the Redis `redis-cli -a` probe with a
health method that does not expose a secret in process arguments or logs.
Redact nested tool arguments, DSNs, authorization headers, registry JSON,
webhook payload/signature keys, `FALLBACK_API_KEY`, `OPENAI_API_KEY`, and secret
file contents from errors, logs, audit, traces, metrics, crash reports, health
output, and MCP/REST resources. Use opaque credential IDs. Test error paths and
startup failure output as well as success paths.

**Acceptance criteria:**
- Distinctive fake secret markers (including nested MCP args and forced DB-connect errors) absent from stdout/stderr, logs, audit, traces, metrics, health/docs, REST/MCP responses, process argv, and CI artifacts.
- Negative control: credential-bearing Redis healthcheck arguments or full-config serialization fail the harness if reintroduced.
- Startup failure output does not print secrets.

**Expected test commands:**
```sh
go test ./internal/web ./internal/mcp ./internal/config ./internal/database -run 'TestRedact|TestLeak|TestHealth|TestArgv|TestSecretFree' -count=1
go test ./internal/... -count=1
go vet ./...
```

---

## Dependencies

| Direction | Sprint / Requirement | Relationship |
|-----------|----------------------|--------------|
| Requires | SPRINT-A0 (R16-03–05) | Fail-closed registry and secrecy before lifecycle ops |
| Requires | SPRINT-A1 (R16-06–08) | Attributable principals + audit for lifecycle events |
| Blocks | SPRINT-A3 (R17-01–05, R18) | Deployment readiness assumes secret sources and rotation story |
| Blocks | SPRINT-A4 (R19-06, R17-11) | CI secret-boundary gates assume redaction and secret sources |
| Aligns | Spec 17 R17-10 rotation/revocation (ops runbook; full inventory) | Shared bounds (≤90d validity, ≤24h overlap, ≤60s revocation) |

---

## Rollback Notes

- Roll back to last known-good **restrictive** credential behavior — never to shared/default/placeholder credentials or unrestricted legacy keys.
- If a rotation or reload fails, keep old material only within the approved overlap while both are valid; never resurrect revoked privileged access.
- On suspected disclosure: revoke/rotate affected credentials and certificates, invalidate exposed sessions, review audit, block unsafe effects — do not restore old secrets.
- Revert code with `git checkout HEAD -- <files>` if argv secrets or full-config leaks reappear; fail closed until fixed.
- `run_migrations.go` accepting a DSN argument is **not** a safe acceptance entrypoint for real credentials; do not roll back to that interface for production tests.

---

## Do Not Proceed If

**STOP this sprint (and do not continue to SPRINT-A3) if any of the following is true:**

- A revoked key can still authorize web or MCP effects beyond the published bound.
- A failed reload or bad replacement enlarges access or restores legacy unrestricted authority.
- Credentials appear in process argv, healthcheck flags/URLs, logs, errors, traces, metrics, or MCP/REST resources (including nested arguments).
- Webhook HMAC values are returned by read/list APIs or stored/exported in plaintext without encryption or managed reference.
- A lifecycle API is documented/claimed but not implemented and protected.
- Secret files are group/world readable, or secrets are present in committed `.env`/Compose/images.
- Revocation propagation is unmeasured or the test uses production credentials/DB.

---

## Acceptance Criteria Summary

| # | Criterion | Test | Pass Condition |
|---|-----------|------|----------------|
| 1 | Lifecycle bounds (R16-09) | Rotate/revoke/reload tests | Deny within bound; no access enlargement |
| 2 | Distinct principals | Issue/assign/expire cases | One principal per credential; audited |
| 3 | Secret source (R17-06) | File provider, ACL, missing-file negatives | Fail closed; restricted read |
| 4 | Webhook HMAC (R17-06) | Read/list/rotate | No HMAC in output; rotation without exposure |
| 5 | Verifier key (R17-07) | ≥256-bit generation, load validation | Rejected if unusable; never returned |
| 6 | Secret-free process (R17-08) | Marker leak sweep + argv probe | Zero markers; healthcheck secret-free |

---

## Reference

- [Spec 16](../specs/16-authentication-authorization-and-evidence-remediation.md) — Gate 3, R16-09
- [Spec 17](../specs/17-deployment-tls-and-secret-boundary.md) — R17-06–08 (and R17-10 bounds)
- [Spec 18](../specs/18-migration-startup-and-readiness.md) — migration job must use protected secret source
- Template basis: [SPRINT_TEMPLATE.md](../archive/sprints/SPRINT_TEMPLATE.md) (adapted for security remediation)

---

**Created:** 2026-10-04
**Version:** 1.1
**Status:** PARTIALLY IMPLEMENTED (reconciled 2026-10-04 against commit 1c838f4)
