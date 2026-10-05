# Sprint: Deployment Readiness, TLS, and Migration

**Sprint ID:** SPRINT-A3
**Source Spec:** 17 — profiles/TLS (R17-01 through R17-05) + Spec 18 — migration/startup/readiness (R18-01 through R18-13)
**Priority:** P1 (Blocking)
**Estimated Effort:** ~3–4 days
**Status:** PARTIALLY IMPLEMENTED (no item ACCEPTED) — see Reconciliation 2026-10-04
**Depends On:** SPRINT-A0, SPRINT-A1, SPRINT-A2
**Blocks:** SPRINT-A4 (CI deploy-readiness and migration gates)

---

## Reconciliation 2026-10-04 — verified against source at commit 1c838f4

Status vocabulary: **WIRED** (code exists, no exact live test), **EXERCISED**
(exact test ran, cited), **NOT_EXERCISED** (live dependency skipped), **NOT_RUN**
(tool/runner unavailable). No item below is ACCEPTED.

- Profile / exposure validation — **EXERCISED (unit)**: `internal/config/profile_test.go`.
- Public TLS 1.3+ termination — **NOT_EXERCISED**.
- DB / Redis transit security — **WIRED**.
- Nonlocal profile rejects default credentials — **WIRED**.
- Migration job — **NOT_EXERCISED** (fake ledger only; no real PostgreSQL).
- Readiness endpoints — **WIRED**.
- `docker compose config` — **NOT_RUN**.

---

## Problem Statement

Deployment does not select and validate an explicit effective profile (`local`
/ `tailnet` / `public`) as a unit. Insecure exposure can start listeners
without proven TLS termination, private ingress, or DB/Redis transit security.
Default/placeholder credentials can reach nonlocal profiles. Schema changes
lack a one-shot versioned migration job with locking, checksum truth, and
deliberate rollback. Startup order, readiness vs liveness separation, dependency
classification, policy honesty, and shutdown/outage behavior are not fully
contracted.

**Why:** Profile/exposure validation, storage transit policy, migration
ownership, and readiness semantics are incomplete.
**Where:** Config/profile selection, listeners, Compose/TLS, database
migrations, `/health/ready` and `/health` (Spec 17 R17-01–05; Spec 18 R18-01–13).

---

## Entry Gate

Before starting this sprint:

- [ ] SPRINT-A0–A2 exit gates complete (fail-closed registry, secrecy, authorization, credential sources).
- [ ] Workloads/ingress/credentials inventory mapped to one profile each (R17 migration §4.1); default remains `local`.
- [ ] Disposable PostgreSQL/Redis and synthetic credentials only for tests.
- [ ] Baseline: `go test ./internal/database ./internal/web ./internal/config ./internal/auth -count=1`.

---

## Exit Gate

This sprint is complete only when all of the following hold:

- [ ] Named profile selected at startup (default `local`); app bind, host publishes, proxy upstream, and reachability validated as a deployment unit (R17-01). — **EXERCISED (unit)**: `internal/config/profile_test.go`.
- [ ] Startup fails closed (nonzero exit, no app socket) on insecure external exposure, `public` without TLS termination + direct-backend denial, `tailnet` without verified private ingress + encrypted authenticated transport (R17-02). — **WIRED** (no real-listener negative exercised).
- [ ] `PRODUCTION_MODE`, `TLS_ENABLED=true`, or cert paths alone never satisfy the exposure gate; no silent downgrade to `local` (R17-02). — **WIRED**.
- [ ] Public TLS 1.3+ termination, certificate/hostname/trust validation, trusted-proxy header stance (R17-03). — **NOT_EXERCISED**.
- [ ] DB/Redis: plaintext only on verified isolated local bridge/loopback for `local`; outside that boundary require `verify-full` / verified Redis TLS, no skip-verify (R17-04). — **WIRED**.
- [ ] Nonlocal profiles reject unset/empty/placeholder/default/reused/test credentials (R17-05); local generates distinct non-placeholder credentials. — **WIRED**.
- [ ] Migration job owns schema (R18-01): ordered manifest (R18-02), atomic idempotency + checksum (R18-03), advisory lock (R18-04), version truth (R18-05), deliberate down migrations only (R18-06). — **NOT_EXERCISED** (fake ledger only; no real PostgreSQL).
- [ ] Ordered startup (R18-07); liveness local/process-only vs readiness gating traffic (R18-08); required/optional dependency classification (R18-09). — **WIRED** (readiness endpoints).
- [ ] Invalid registry → readiness fail or protected deny without legacy broadening (R18-10); policy honesty (R18-11); shutdown/outage fail-closed (R18-12); bounded evidence without secrets (R18-13). — **WIRED**.
- [ ] Current `run_migrations.go` DSN-argument interface is replaced or not used as a production credential entrypoint. — **NOT_EXERCISED** (no audited verdict).

---

## Scope Boundary

```
IN SCOPE (may modify):
  - Deployment profile selection and preflight/attestation checks
  - Listener exposure validation (web + MCP), TLS/proxy boundary wiring
  - DB/Redis transport policy enforcement (local vs nonlocal)
  - Production credential rejection (no default/placeholder secrets)
  - Versioned migration job, ledger, locking, checksums, rollback operator path
  - Startup order, liveness/readiness endpoints, dependency classification
  - Shutdown/drain and required-dependency outage behavior
  - Tests: profile negatives, migration idempotency/rollback, readiness transitions

OUT OF SCOPE (DO NOT TOUCH):
  - Credential lifecycle APIs (SPRINT-A2; use existing sources)
  - Principal intersection matrix (SPRINT-A1; verify only)
  - CI workflow YAML gates and required-check config (SPRINT-A4 / R19)
  - External KMS product integration (optional later)
  - Production databases, production secrets, public live endpoints in tests
  - Advertising native TLS or proxy products as shipped capabilities beyond reviewed defaults
```

---

## Numbered Tasks

### Task 1 — Effective profile and fail-closed exposure (R17-01, R17-02)

**Action:** Select a named profile at startup (default `local`). Validate app
bind, host-published mappings, proxy upstream, and firewall/network-policy
reachability as one deployment unit. Allow container-wide `0.0.0.0` only on a
verified isolated backend network (`local` Compose) or behind an isolated proxy
(`tailnet`/`public`). Require preflight attestation when the process cannot
inspect external mapping/firewall. Before starting either listener, fail
startup (nonzero exit, no app socket) on insecure exposure; do not silently
downgrade profiles; block readiness on post-start drift.

**Acceptance criteria:**
- External publish or untrusted bridge peer → preflight/startup fails (not silent serve).
- `public` without proven TLS termination and direct-backend denial → fail.
- `tailnet` without verified private ingress + encrypted authenticated transport → fail.
- Flag-only `PRODUCTION_MODE` / `TLS_ENABLED` / cert path presence → still fail if exposure unproven.
- Both web and MCP listeners covered.

**Expected test commands** (from `mcp-server/`):
```sh
go test ./internal/config ./internal/web ./internal/mcp -run 'TestProfile|TestBind|TestExposure|TestPreflight|TestStartup' -count=1
go test ./internal/config ./internal/web ./internal/mcp -count=1
```

### Task 2 — Public TLS, proxy, and storage transit (R17-03, R17-04)

**Action:** Terminate TLS 1.3+ at public ingress; validate certificate, key,
trust chain, hostname; reject invalid cert/wrong host/untrusted backend route.
Only configured proxy peers supply forwarded identity/scheme; clear untrusted
forwarded headers at the edge (Spec 12 stance). DB/Redis plaintext only on
verified isolated local boundary for `local`; otherwise PostgreSQL
`verify-full` with trusted CA + hostname verification and Redis TLS with
verified server cert/hostname; no skip-verify or opportunistic fallback.

**Acceptance criteria:**
- TLS 1.2 handshake / cleartext public request / invalid certificate / direct backend reach → blocked.
- DB `disable`/`prefer`/unverified `require`, Redis TLS=false or skip-verify, bad hostname → startup/readiness fail when outside isolated local network.
- Compose `DB_SSLMODE=disable` / Redis TLS=false cannot be reused as production defaults.

**Expected test commands:**
```sh
go test ./internal/config ./internal/database -run 'TestTLS|TestDBSSL|TestRedis|TestTransit' -count=1
# Network/attestation harness (synthetic topology descriptors; no public endpoints)
go test ./internal/config -run 'TestProxy|TestForwarded' -count=1
```

### Task 3 — No production default credentials (R17-05)

**Action:** Nonlocal/production profiles require unique high-entropy
operator-generated DB, Redis, MCP/IDE (migration), scoped registry, and JWT
credentials. Reject unset, empty, known placeholder/default, reused, or test
credentials before startup. Local generates distinct non-placeholder
credentials at setup. Separate DB/Redis accounts per environment.

**Acceptance criteria:**
- Placeholder/default/test credential in nonlocal profile → startup rejected.
- Checked-in sample values never accepted as production credentials.
- Local setup does not advertise fixed Compose fallback as safe.

**Expected test commands:**
```sh
go test ./internal/config ./internal/database -run 'TestCredentialPolicy|TestDefaultSecret|TestPlaceholder' -count=1
```

### Task 4 — Versioned migration job (R18-01 through R18-06)

**Action:** Own numbered schema changes in a one-shot versioned migration job
(not normal app startup). Validate and apply immutable numeric migrations in
ascending order under a stable manifest. Each migration + ledger record commits
transactionally; repeat runs are no-op + checksum verification. Take a bounded
PostgreSQL advisory lock; fail on contention/timeout. Version, checksum, dirty
state, and compatibility bounds are authoritative; incompatible state is
non-ready. Down migrations never automatic; approved rollback is locked,
backed up, tested, explicit.

**Acceptance criteria:**
- Clean DB: first run exits 0 and records versions/checksums in order; second run exits 0, no schema/data change, identical checksums.
- Concurrent jobs: at most one mutates; other times out with FAIL (never PASS).
- Mid-failure: version not recorded, transaction rolled back, `/health/ready` non-2xx until operator resolves.
- Edited applied file without new version → checksum mismatch fails before mutation; no auto-repair.
- `run_migrations.go` must not accept raw DSN argv as a production-safe entrypoint.

**Expected test commands:**
```sh
go test ./internal/database ./internal/web ./internal/config ./internal/auth -count=1
go test ./internal/database -run 'TestMigrationJob|TestMigration' -count=1
go vet ./...
```

### Task 5 — Startup, readiness, policy honesty, lifecycle (R18-07 through R18-13)

**Action:** Enforce ordered startup (config → dependencies → schema → registry
→ dependency graph → policy). Liveness is local/process-only; readiness gates
normal traffic and includes all required state. Classify required vs optional
dependencies per profile; unknown/error/timeout is never PASS. Invalid registry
fails readiness or denies protected requests without legacy broadening. Disabled
or unexercised policy never reports enforcement PASS. Shutdown and
required-dependency outage remove readiness before drain/recovery and preserve
fail-closed behavior. Each check exposes bounded reason, timestamp,
version/digest where applicable, terminal result — no secrets.

**Acceptance criteria:**
- Registry malformed/empty/duplicate/unknown scope → readiness non-2xx or protected deny; no legacy fallback.
- Policy disabled in enforcement profile → readiness non-2xx; no policy PASS / no protected effect.
- PostgreSQL/Redis unreachable after ready → readiness non-2xx within published detection bound; liveness may stay 2xx; recovery needs fresh successful check.
- SIGTERM: `DRAINING` and non-2xx readiness before first listener drains; in-flight work gets configured deadline; no migration during shutdown.
- `docker compose config` valid; healthcheck uses readiness (not only liveness `--health-check`).

**Expected test commands:**
```sh
go test ./internal/database -run 'TestMigrationJob|TestReadiness' -count=1
go test ./internal/web ./internal/config ./internal/auth -run 'TestReadiness|TestLiveness|TestStartupOrder|TestShutdown|TestPolicy' -count=1
go test ./internal/... -count=1
docker compose config
```

---

## Dependencies

| Direction | Sprint / Requirement | Relationship |
|-----------|----------------------|--------------|
| Requires | SPRINT-A0 (R16-03–05) | Registry fail-closed feeds R18-10 readiness |
| Requires | SPRINT-A1 (R16-06–08) | Authorization/audit before protected traffic readiness |
| Requires | SPRINT-A2 (R17-06–08) | Secret sources and argv-free probes before deployment tests |
| Blocks | SPRINT-A4 (R19-07–08, R19-12) | CI deploy-readiness/migration/test-floor rows |
| Aligns | Spec 12 exposure classes, Spec 16 Gate 1 | Same fail-closed exposure and registry contract |

---

## Rollback Notes

- Roll back config/policy to the last known-good **restrictive** profile, still-pinned image, and valid rotated secret set.
- A previous HTTP external bind, placeholder DB/Redis credential, unredacted resource, legacy unrestricted auth, or disabled TLS is **not** a valid rollback.
- If no safe snapshot exists: stop external/protected traffic, preserve audit records, repair; revoke exposed material rather than reusing it.
- Migration rollback: locked, backed up, tested, explicit down steps only — never automatic.
- Incompatible/dirty schema after partial failure stays non-ready until operator resolution; do not force-forward.

---

## Do Not Proceed If

**STOP this sprint (and do not continue to SPRINT-A4) if any of the following is true:**

- Insecure external exposure can start listeners or downgrade silently to `local`.
- Nonlocal profile accepts placeholder/default/test/empty credentials.
- DB/Redis outside the isolated local boundary can connect without verified encryption (`verify-full` / verified Redis TLS).
- Migration job can run unserialized, auto-repair checksum drift, or auto-run down migrations.
- Readiness reports ready with invalid registry, incompatible schema, required dependency down, or policy disabled/unexercised in an enforcement profile.
- Liveness depends on external secrets/process argv or readiness is the only probe used for liveness.
- Tests require production credentials, a production database, or live public endpoints.
- Windows-only failures are marked PASS without a documented supported-runner rerun.

---

## Acceptance Criteria Summary

| # | Criterion | Test | Pass Condition |
|---|-----------|------|----------------|
| 1 | Profile + bind (R17-01–02) | Preflight/startup negatives | Fail closed; no app socket on insecure exposure |
| 2 | TLS/proxy (R17-03) | Handshake/cert/hostname/forwarded-header cases | 1.3+ only; untrusted paths deny |
| 3 | Storage transit (R17-04) | DB/Redis mode matrix | Verified encryption outside local isolated net |
| 4 | No default creds (R17-05) | Placeholder/reuse negatives | Startup rejects before serve |
| 5 | Migration job (R18-01–06) | Clean/repeat/concurrent/checksum-drift | Idempotent, locked, rollback deliberate |
| 6 | Readiness/lifecycle (R18-07–13) | Registry/policy/outage/SIGTERM scenarios | Fail-closed evidence; no secrets in output |

---

## Reference

- [Spec 17](../specs/17-deployment-tls-and-secret-boundary.md) — R17-01–05 (profiles, TLS, transit, credentials)
- [Spec 18](../specs/18-migration-startup-and-readiness.md) — R18-01–13
- [Spec 12](../specs/12-web-exposure-boundary.md) — exposure classes and trusted proxy
- [Spec 19](../specs/19-phase0-ci-security-gates.md) — R19-08 (deploy readiness/migration gates)
- Template basis: [SPRINT_TEMPLATE.md](../archive/sprints/SPRINT_TEMPLATE.md) (adapted for security remediation)

---

**Created:** 2026-10-04
**Version:** 1.1
**Status:** PARTIALLY IMPLEMENTED (reconciled 2026-10-04 against commit 1c838f4)
