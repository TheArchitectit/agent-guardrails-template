# OpenSpec 18: Migration, startup, and readiness

**Status:** Proposed; **not implemented**. This document defines a future
startup/deployment contract. It does not claim that the current server applies
migrations, validates all readiness state, or enforces these gates.

**Owner default:** The solo maintainer owns the migration manifest, release
decision, rollback decision, and readiness policy. A second reviewer is
required for destructive/down migrations when one is available; lack of a
reviewer does not authorize an automatic rollback.

**Related:** [11 — authorization scopes and roles](11-authorization-scopes-and-roles.md),
[16 — authentication, authorization, and evidence remediation](16-authentication-authorization-and-evidence-remediation.md),
[17 — deployment TLS and secret boundary](17-deployment-tls-and-secret-boundary.md)
for deployment profiles and secret sources, and
[19 — Phase 0 CI/security gates](19-phase0-ci-security-gates.md)
contains the release-gate dependency for this proposal.

## 1. Source-grounded baseline

The inspected `mcp-server/cmd/server/main.go` currently loads and validates
configuration, connects to PostgreSQL, starts the database metrics collector,
connects to Redis, constructs web/MCP services, and then starts both listeners.
The collector starts before the web and MCP servers (`main.go:64-125`). The
startup path contains no automatic numbered migration application; migration
execution is therefore an external deployment step until a reviewed contract is
implemented.

The current external runner is `mcp-server/scripts/run_migrations.go`. It takes
a database URL as a command argument (which can expose credentials in process
listings and conflicts with Spec 17 R17-08), creates `schema_migrations` if
needed, reads `.up.sql` files,
skips versions already recorded, executes each migration in a transaction, and
records the version in that transaction. It walks `os.ReadDir` results without
an explicit numeric-version sort, has only a string version ledger, and lacks
checksum, dirty-state and advisory-lock semantics; it also verifies a fixed
table list. This is useful current evidence, not proof of the requirements
below. The database URL is currently supplied through argv (Spec 17's secret
boundary forbids that for real credentials).

The web server exposes anonymous `/health/live` and `/health/ready`
(`internal/web/server.go:135-140`). Liveness returns `200` without dependency
checks. Readiness currently checks only database and cache connectivity and
returns `503` on those failures (`internal/web/server.go:368-405`); it does not
check schema version, migration state, credential-registry validity,
policy-engine state, or the broader dependency graph. The binary's
`--health-check` mode currently calls `/health/live`, not `/health/ready`
(`cmd/server/main.go:54-62,239-261`). Compose uses that mode as its server
healthcheck (`docker-compose.yml:111-116` and
`mcp-server/deploy/docker-compose.example.yml:210-217`). These facts must not
be read as an implementation of this proposal.

## 2. Decision: explicit versioned migration job

Migration ownership is deliberately **not** assigned to normal application
startup. A release MUST run one explicit, versioned migration job before the
application is eligible for traffic. The application MUST refuse readiness
when its database is below the image's declared compatible schema target.
Application startup MUST NOT silently create, renumber, repair, or apply
numbered migrations.

The deployment owner is responsible for ordering the job before the server
rollout. In Compose, the future shape is a dedicated one-shot migration
service/job that uses the same image and database credentials, completes
successfully, and is a prerequisite for the server. `depends_on` healthchecks
for Postgres/Redis only establish service reachability; they are not migration
success evidence. No current Compose file is changed by this specification.

## 3. Ordered startup and readiness checks

The future startup sequence MUST be deterministic and report a named state for
each check. The order is:

1. Parse and validate configuration, including the declared schema target,
   migration mode, dependency profile, credential-registry source, and policy
   profile. Invalid configuration stops startup and is never `PASS`.
2. Establish required database and cache connections with bounded timeouts.
   A successful TCP connection alone is insufficient; execute each dependency's
   authenticated, application-level ping.
3. Confirm the database schema is compatible with the image: the recorded
   version exists, is not dirty, has a valid checksum, and is at or above the
   minimum required version while not exceeding an incompatible future version.
4. Validate the credential registry before serving protected traffic. A
   configured source that is malformed, unreadable, empty, duplicated, or
   outside the approved catalog fails readiness; it MUST NOT fall back to a
   wider legacy surface. This follows Spec 16/R16-03 and Spec 19/R19-05.
5. Resolve the dependency graph and classify every dependency as required or
   optional for the selected profile. Detect missing, cyclic, unknown, timed
   out, or errored dependencies before declaring readiness.
6. Resolve policy state. Confirm the selected policy version/digest and policy
   engine mode. A disabled policy engine is an explicit non-enforcing state,
   not a successful enforcement check.
7. Publish the startup/readiness state and only then admit normal traffic.
   Metrics may begin earlier (as they currently do after DB connection and
   before web/MCP construction), but metrics availability MUST NOT substitute
   for any readiness check.

Every check has one of `PASS`, `FAIL`, `NOT_READY`, `DEGRADED`, `SKIPPED`, or
`NOT_RUN`, with a reason code and bounded diagnostic. `FAIL`, `NOT_READY`,
`SKIPPED` for a required check, `NOT_RUN`, timeout, unknown, or error MUST
never be converted to `PASS`. An optional dependency may yield `DEGRADED` only
when the selected profile explicitly permits operation without it.

## 4. Migration job contract

### 4.1 Ownership and locking

The migration job is the sole owner of schema mutation. It MUST read database
credentials from a protected runtime secret source (file descriptor, mounted
secret, or secret provider), never a URL/secret in argv or logs. It MUST acquire
a PostgreSQL advisory lock derived from a documented, stable application/schema
namespace before reading or changing migration state. Lock acquisition MUST
have a bounded timeout; a timeout is a failed job, not a successful no-op. The
lock MUST be released on commit, rollback, cancellation, or connection loss.
The job MUST re-read schema state after acquiring the lock so two jobs cannot
make decisions from stale state.

### 4.2 Ordered, transactional, idempotent application

Migration files MUST have an immutable, sortable numeric version and a unique
name. The job applies pending versions in ascending numeric order, never
filesystem enumeration order alone. Before execution it MUST validate the
complete manifest: no duplicate versions, missing paired metadata, invalid
names, gaps where the release policy forbids gaps, or unsupported future
versions.

For each version, the job MUST:

1. Read the immutable migration content and expected checksum.
2. Compare it with the recorded checksum if the version is already applied;
   checksum mismatch is a hard failure and MUST NOT be repaired automatically.
3. Execute the version's up operation in one database transaction.
4. Record version, name, checksum, application time, and tool format in that
   same transaction.
5. Commit only after all statements and the record succeed; otherwise roll
   back the entire version and exit nonzero.

Re-running a completed job MUST make no schema or data change and MUST verify
checksums. `CREATE IF NOT EXISTS` inside SQL is not sufficient evidence of
idempotency: the version ledger and transaction boundary are authoritative.
A partial migration MUST leave the failed version unapplied/dirty and the
application not ready until an operator resolves it.

### 4.3 Version and rollback semantics

The database MUST expose one authoritative schema state: current version,
expected checksum, and a dirty/blocked marker. The reviewed image's immutable
migration manifest selects the highest target version; its build metadata
records the minimum compatible and maximum tested versions. A missing or
ambiguous manifest is NOT_READY, never an operator-guessable version. The image
MUST declare its minimum compatible and maximum tested schema versions. A database newer than
the image's supported maximum, a dirty version, or checksum drift is
`NOT_READY` and blocks protected traffic.

Rollback defaults to **application rollback first, schema rollback only when
explicitly approved**. The operator first deploys an image compatible with the
already-applied schema, or rolls forward with a corrective migration. Down
migrations are never run automatically on application shutdown, health failure,
or deployment retry. A down migration requires a named target version,
backup/restore point, maintenance window, tested down script, advisory lock,
and an explicit operator command. Destructive down migrations MUST be
backward-reviewed and must state data-loss consequences.

Migration authors SHOULD use expand/contract changes: add compatible schema,
deploy code that can read both forms, backfill separately, then remove old
forms only in a later release. If a migration or rollback cannot preserve the
old application contract, the deployment MUST stop before traffic shifts.

## 5. Readiness and liveness contract

### 5.1 Liveness

`/health/live` answers whether the process can accept a probe and is not in
terminal shutdown. It MUST be fast, local, and independent of PostgreSQL,
Redis, schema version, registry, policy engine, or optional providers. A live
process may still be unready. During shutdown it MUST stop reporting live
before the process exits (or close the listener so probes fail).

### 5.2 Readiness

`/health/ready` answers whether this instance may receive normal protected
traffic under its selected profile. It MUST return non-2xx (normally `503`)
until the ordered checks pass, and whenever a required dependency, schema,
registry, policy, or startup invariant becomes invalid. The response MUST NOT
include credentials or raw dependency errors; logs and metrics may carry a
bounded reason code and correlation ID.

An explicit process lifecycle state (`STARTING`, `READY`, `DRAINING`,
`STOPPED`) SHALL be observed by `/health/ready`. It moves to `DRAINING` and
returns non-2xx before graceful shutdown begins, remains false while requests
drain, and stays false after a required outage until the ordered recovery
checks pass again. Current `main.go:205-234` has no such transition; adding
it is implementation work. Liveness MUST NOT be used to claim readiness.
The container probe contract MUST eventually call readiness (not only
liveness) once the implementation changes; updating that deployment wiring is
future work, not current behavior.

### 5.3 Required versus optional dependencies

The profile manifest MUST declare this classification rather than infer it
from whether a package is installed:

| Dependency/state | Default contract | If unavailable or invalid |
|---|---|---|
| PostgreSQL | Required | `NOT_READY`; protected operations stop |
| Redis/cache | Required for the current server profile | `NOT_READY`; no stale `PASS` |
| Applied compatible schema | Required | `NOT_READY`; migration job/operator action required |
| Configured credential registry | Required when configured | `NOT_READY` or protected denial; never broaden legacy access |
| Policy engine | Required for an enforcement profile | `NOT_READY`; disabled is non-enforcing, never `PASS` |
| Optional provider (for example Ollama/vision) | Optional only if profile says so | `DEGRADED`; dependent features remain unavailable/denied |
| Metrics exporter | Operationally recommended, not a readiness authority | Record exporter failure; do not turn other failures into `PASS` |

An optional dependency that is selected as required by a feature/profile changes
that profile's result to `NOT_READY` when unavailable. “Installed,”
“reachable,” and “exercised” are distinct states. An unexercised library or
configured URL is not evidence that a dependency is healthy.

## 6. Credential registry and policy state

A configured credential registry MUST be parsed and validated atomically before
readiness. Validation includes source readability, JSON/schema validity,
non-empty records, unique credential and verifier identities, valid principal
IDs, approved scope/role vocabulary, verifier-key usability, and valid
expiry/revocation fields. A replacement is loaded off to the side and swapped
only after complete validation; a bad replacement MUST preserve neither an
expanded access mode nor an accidental nil-registry fallback. The absence of a
registry may be allowed only under an explicit, time-bounded legacy migration
profile that cannot authorize privileged or mutating operations.

The policy profile MUST report `enabled`, `disabled`, `configured`, and
`exercised` distinctly. If policy enforcement is required, `disabled`, missing,
unknown, errored, timed out, or merely installed policy state is non-ready.
If policy is deliberately optional, readiness may be `DEGRADED`, but affected
operations MUST return an explicit not-configured/non-enforcing result and
must not claim a policy `PASS`. This aligns with Spec 11's authorization model,
Spec 16's fail-closed registry remediation, and Spec 17's dependency state
contract.

## 7. Shutdown and outage behavior

On SIGTERM/SIGINT or an internal fatal startup error, the service MUST mark
readiness false, stop new protected work, drain web and MCP requests within the
configured shutdown deadline, stop background collectors/dispatchers, and
close database/cache clients. A shutdown timeout is logged as failure and the
process exits; it is not a readiness success. The migration job is never
started as part of shutdown.

For a transient required-dependency outage, probes and request admission must
fail closed without repeatedly applying migrations or widening authorization.
Recovery requires fresh authenticated checks, schema/registry/policy
revalidation where relevant, and a successful readiness transition. Repeated
outage/recovery events MUST be measurable with bounded reason labels and MUST
not leak secrets.

## 8. Numbered requirements (R18)

1. **R18-01 — Explicit ownership.** Numbered schema changes MUST be owned by a
   one-shot versioned migration job, not normal application startup.
2. **R18-02 — Ordered manifest.** The job MUST validate and apply immutable,
   numeric migrations in ascending order under a stable manifest.
3. **R18-03 — Atomic idempotency.** Each migration and its ledger record MUST
   commit transactionally; repeat runs MUST be no-op plus checksum verification.
4. **R18-04 — Serialized execution.** The job MUST take a bounded PostgreSQL
   advisory lock and fail on contention/timeout.
5. **R18-05 — Version truth.** Version, checksum, dirty state, and compatibility
   bounds MUST be authoritative and incompatible state MUST be non-ready.
6. **R18-06 — Deliberate rollback.** Down migrations MUST never be automatic;
   approved rollback MUST be locked, backed up, tested, and explicit.
7. **R18-07 — Ordered startup.** Config, dependencies, schema, registry,
   dependency graph, and policy checks MUST run in the specified order.
8. **R18-08 — Probe separation.** Liveness MUST be local/process-only;
   readiness MUST gate normal traffic and include all required state.
9. **R18-09 — Dependency classification.** Required and optional dependencies
   MUST be profile-declared; unknown/error/timeout is never `PASS`.
10. **R18-10 — Registry fail-closed.** Invalid configured registry state MUST
    fail readiness or deny protected requests without legacy broadening.
11. **R18-11 — Policy honesty.** Disabled or unexercised policy MUST never be
    represented as an enforcement `PASS`.
12. **R18-12 — Lifecycle safety.** Shutdown and required-dependency outage MUST
    remove readiness before drain/recovery and preserve fail-closed behavior.
13. **R18-13 — Evidence.** Each check MUST expose a bounded reason, timestamp,
    version/digest where applicable, and terminal result for the deployment
    record without exposing secrets.

## 9. Measurable G/W/T scenarios

These are future acceptance scenarios, not current passing evidence.

- **G:** A clean database is below target version. **W:** The explicit job runs
  once, then runs again. **T:** The first run exits zero and records every
  version/checksum in order; the second exits zero, changes no schema/data, and
  verifies identical checksums.
- **G:** Two migration jobs start concurrently. **W:** Both request the same
  advisory lock. **T:** At most one mutates the schema; the other waits up to
  the configured bound and reports `FAIL` on timeout, never `PASS`.
- **G:** A migration statement fails midway. **W:** The job exits nonzero.
  **T:** The version is not recorded as applied, the transaction is rolled
  back, and `/health/ready` is non-2xx until an operator resolves the state.
- **G:** An applied file is changed without a new version. **W:** The job runs.
  **T:** Checksum mismatch fails before schema mutation and does not auto-repair.
- **G:** The registry file is malformed, empty, duplicated, or uses an unknown
  scope. **W:** The server starts. **T:** Readiness is non-2xx (or protected
  requests deny) and no legacy fallback grants mutation.
- **G:** The policy engine is disabled in an enforcement profile. **W:** A
  readiness probe and protected operation run. **T:** Readiness is non-2xx and
  the operation cannot report policy `PASS` or produce its protected effect.
- **G:** PostgreSQL or Redis becomes unreachable after readiness. **W:** The
  next bounded readiness check runs. **T:** Readiness becomes non-2xx within
  the published detection bound; liveness may remain 2xx; recovery requires a
  fresh successful dependency check.
- **G:** SIGTERM arrives while traffic is active. **W:** Shutdown begins.
  **T:** A bounded real-process test observes `DRAINING` and non-2xx readiness
  before the first listener drains; in-flight work receives the configured
  deadline, clients close, and no migration is attempted.

## 10. Verification commands and rollout defaults

Commands below are future acceptance commands; their output is not evidence
that this proposal is implemented. Run from `mcp-server/` with a disposable
PostgreSQL/Redis environment and synthetic credentials only:

```sh
go test ./internal/database ./internal/web ./internal/config ./internal/auth -count=1
go test ./internal/... -count=1
go vet ./...
# Future integration tests (added with the migration job implementation):
go test ./internal/database -run 'TestMigrationJob|TestReadiness' -count=1
# The current run_migrations.go accepts a DSN argument and is NOT a safe
# acceptance entrypoint for any real credential. Replace its interface with a
# protected secret source before testing the job in deployment.
```

The future integration suite MUST add deterministic tests for lock contention,
checksum drift, partial transaction rollback, incompatible schema, registry
invalidity, policy-disabled readiness, dependency outage/recovery, readiness
before shutdown, and zero unauthorized side effects. Compose validation should
also include `docker compose config` and an isolated probe that verifies the
server healthcheck uses readiness rather than only `--health-check` liveness.
Never run these checks against a production database or with production
credentials.

Deployment rollback is staged: (1) run and record the job against a backup or
restore point; (2) verify schema compatibility and readiness; (3) shift
traffic; (4) if the image fails, return to the last compatible application
image without undoing schema; (5) use a reviewed down migration only when
forward compatibility and restoration are impossible. If migration state,
registry validity, policy state, or rollback safety is unknown, stop traffic
and keep readiness false rather than restore a permissive fallback. No code,
Compose file, workflow, or migration is changed by this document.
