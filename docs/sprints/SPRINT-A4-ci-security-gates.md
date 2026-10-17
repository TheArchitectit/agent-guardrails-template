# Sprint: Phase 0 CI Security Gates

**Sprint ID:** SPRINT-A4
**Source Spec:** 19 — Phase 0 Same-Commit CI Security Gates (R19-01 through R19-16)
**Priority:** P1 (Blocking)
**Estimated Effort:** ~2–3 days
**Status:** PARTIALLY IMPLEMENTED (no item ACCEPTED) — see Reconciliation 2026-10-04 and 2026-10-05**Depends On:** SPRINT-A0, SPRINT-A1, SPRINT-A2, SPRINT-A3
**Blocks:** Phase 0 release / branch-protection enablement

---

> Historical reconciliations (2026-10-04, 2026-10-05 parts 1–3, including the
> reviewed Gitleaks baseline) moved to
> [SPRINT-A4-reconciliation-log.md](SPRINT-A4-reconciliation-log.md) on
> 2026-10-17 to keep this file within the 500-line limit (Spec 19 R19-11).

---

## Reconciliation 2026-10-15 — client-IP trust boundary row (R19-09 proxy half)

Spec 12 §3.4 (trusted-proxy and client-IP stance) is implemented, and the
matrix now carries a dedicated row:

- `security-matrix-clientip-trust` — client-IP trust boundary. `internal/clientip`
  derives the effective client IP from an explicit `TRUSTED_PROXIES` stance
  (empty = trust no proxy, the default; a bare IP is a single-host entry; a
  malformed entry fails config validation closed). The stance is installed as
  `echo.IPExtractor`, so the rate-limit key fallback and log attribution use the
  socket peer unless the immediate peer is a configured proxy. Assertions: with
  `TRUSTED_PROXIES` unset a forged `X-Forwarded-For`/`X-Real-IP`/`Forwarded` does
  not change the client IP and two forged headers share the socket-peer bucket
  (acceptance criterion 7); a peer inside a configured CIDR may supply the
  client IP; an untrusted peer's forwarded header is logged as a protocol
  anomaly, not rejected; a malformed `TRUSTED_PROXIES` fails `Config.Validate`.
  **WIRED** + **EXERCISED**, command:
  `cd mcp-server && go test ./internal/clientip ./internal/web ./internal/config -run 'TestDefaultTrustsNoProxy|TestTrustedPeerHonoursForwarded|TestBareIPEntryTrusted|TestMalformedEntryFailsClosed|TestRateLimitKeyIgnoresForgedForwardedHeaders|TestTrustedProxyPeerUsesForwardedHeader|TestForwardedHeaderAnomalyMiddlewarePassesThrough|TestValidateTrustedProxiesStance' -count=1` → ok.
- Mutation-kill case `clientip-stance-trusts-every-peer` (seventh named bypass):
  `Stance.Trusts` returns true unconditionally; kills
  `TestRateLimitKeyIgnoresForgedForwardedHeaders`. **EXERCISED**.

## Reconciliation 2026-10-16 — readiness/lifecycle row (R19-08 readiness half)

Spec 18 R18-07..R18-13 (ordered readiness and lifecycle) is implemented, and the
matrix now carries a dedicated row:

- `security-matrix-readiness-lifecycle` — ordered readiness/lifecycle contract.
  `internal/web/readiness.go` runs the ordered checks (`lifecycle`, `config`,
  `database`, `cache`, `schema`, `credential_registry`, `policy`); an absent
  probe is `NOT_RUN` (never `PASS`); a configured-but-invalid registry is
  `NOT_READY` while an absent registry is `SKIPPED`; a disabled or unexercised
  policy engine is never an enforcement `PASS`; the explicit
  `STARTING`/`READY`/`DRAINING`/`STOPPED` lifecycle makes a non-`READY` instance
  non-ready; and evidence carries only bounded reason codes. `/health/ready`
  returns `503` for any required non-PASS and `200` only when ready;
  `--health-check` now calls `/health/ready`. **WIRED** + **EXERCISED**:
  `go test ./internal/web -run 'TestReadiness|TestLifecycle|TestHealthReady|TestPolicyHonesty|TestMissingProbe|TestRegistryFailClosed' -count=1` → ok.
- Mutation-kill cases `readiness-lifecycle-gate-removed` (eighth bypass: the
  aggregate decision ignores the lifecycle) and
  `readiness-required-probe-failure-swallowed` (ninth: a failing required probe
  is recorded `PASS`); each kills the corresponding negative suite. **EXERCISED**.

A live-process shutdown-drain test and the Compose readiness probe swap stay
future integration work — the readiness *contract* is implemented and exercised.

The web-exposure spec's shared-registry, CORS-wiring and `public`-profile
`/metrics` rows remain **NOT_RUN** (the last is an operator scrape-contract
decision, §3.2/§7).

The `check-broken-links` job in `.github/workflows/documentation-check.yml` was
rewired to call `scripts/check-doc-links.sh` so the CI control and the harness
control are the same artifact (no drift). The 500-line job remains inline.

**Full-history Gitleaks (R19-06).** `scripts/gitleaks-history.sh` runs Gitleaks
over every commit (`gitleaks git --log-opts=--all`, exit 127 with a NOT_RUN
message when Gitleaks is absent — never a silent pass). Run locally on this
branch with Gitleaks 8.30.1: **EXERCISED, NOT GREEN — 1149 commits scanned, 16
leaks reported (exit 1)**. Findings are dominated by intentional test-canary and
placeholder values (e.g. `secret_leak_redaction_test.go`, `auth_test.go`,
`resource_config_secrecy_test.go`), plus historical docs/examples entries
(`docs/MCP_TOOLS_REFERENCE.md`, `STATUS.md`, `cpofopencode`,
`docs/standards/PROJECT_CONTEXT_TEMPLATE.md`). Because the scan is **not green
locally**, it is deliberately **not wired as a CI required-check row**: doing so
would make the gate fail on day one. Wiring requires a reviewed baseline or a
`.gitleaksignore` classifying each historical finding first. This is the real
captured result, not fabricated.

**Reconciliation 2026-10-03 (part 4) — reviewed Gitleaks baseline.** Each of the
16 findings was reviewed individually and a reviewed `.gitleaksignore` added at
the repo root (one `<commit>:<file>:<rule>:<line>` fingerprint per entry, each
with a one-line rationale). **11 findings are intentional false positives** and
are now allowlisted: the test canaries in
`mcp-server/internal/mcp/secret_leak_redaction_test.go` (`CANARY_SECRET_LEAK_…`),
`resource_config_secrecy_test.go` (`SECRET_MARKER_…`),
`streamable_http_revocation_test.go` (`revocation-propagation-verifier-key-…`),
`registry_lifecycle_test.go` (`rotated-verifier-key-…`),
`auth_test.go` (`mcp-test-verifier-key-…`), plus documentation/example
placeholders in `docs/MCP_TOOLS_REFERENCE.md` (a Stripe-style example key),
`docs/standards/PROJECT_CONTEXT_TEMPLATE.md` (an illustrative `sk-…` key), and
`examples/regression-prevention/prevention-rules-examples.json` (illustrative
`bad`-example credentials).

**5 findings were classified as REAL credentials and left un-ignored**: the four
`cpofopencode` hits at commit `0c962de` (an exported MCP/LLM gateway config with
a live-looking `mcp_…` API key and a 64-hex `ah-…` API key aimed at internal
Tailscale endpoints) and the `STATUS.md` hit at commit `b651730` (a 60-char
deployment API key). These are not canary/placeholder-shaped and were not
allowlisted, so the scan correctly **still exits 1** (5 leaks) — the gate stays
**NOT_RUN** and is not wired as a required check until the credentials are
rotated and purged from history.

---

## Reconciliation 2026-10-17 — module enumeration and per-module test floor (R19-10)

Spec 19 R19-10 requires the package gate to **enumerate modules from the
repository** and run each Go module, rather than assuming a `go.work` workspace.
The gap was real and narrow: this repo has **no `go.work`**, and the module list
was hardcoded in three independent places — the `team-validation.yml` test/build
steps, the `team-validation-fleet.yml` steps, and `.guardrails/test-floor.json` —
each naming the same three modules (`mcp-server`, `cmd/team-cli`,
`examples/go`). Nothing tied that list to the tree, so a newly added `go.mod`
would silently escape every Go test, build and floor check.

- `scripts/check_go_modules.py` — fail-closed discovery gate. It discovers every
  `go.mod` in the tree (excluding `.git`, `node_modules`, `vendor`, `testdata`)
  and requires **an inventory row per discovered module and an existing module
  per row**. A discovered module with no row, a stale row, a duplicate row, a
  missing `test_command`/`build_command`/`min_tests`/`applicability` field, a
  non-positive `min_tests`, an empty scan scope, or a missing/unreadable
  inventory each **FAIL** (non-vacuity: zero discovered modules FAILS).
  **WIRED** + **EXERCISED**: `python3 scripts/check_go_modules.py` → ok;
  red-on-drift proven both ways — `scripts/test_check_go_modules.py` (10 cases,
  `python3 scripts/test_check_go_modules.py` → 10 passed) and a live fourth
  `go.mod` flipping the gate FAIL.
- `.guardrails/test-floor.json` — inventory rows now declare the R19-12 fields
  per module: `mcp-server` (`go test ./... -race -count=1`, floor 250),
  `cmd/team-cli` (`make build`, floor 8), `examples/go`
  (`go test ./... -count=1`, floor 10).
- CI steps no longer name modules: `team-validation.yml` and the trusted-main
  fleet lane run `Enforce Go module discovery (Spec 19 R19-10)` then
  `Test and build every enumerated Go module`, iterating the inventory and
  executing each row's declared `test_command`/`build_command` before checking
  its positive floor. `regression-guard.yml` gains a
  `Go module discovery (Spec 19 R19-10)` step. Execution now follows the tree
  rather than a hardcoded list. **EXERCISED** end-to-end locally: all three
  modules TEST_OK/BUILD_OK, floors clear (`mcp-server` 436 ≥ 250,
  `cmd/team-cli` 9 ≥ 8, `examples/go` 11 ≥ 10), `LOOP_DONE`.

Known-limitation (honest scope): the enumeration covers Go modules only. Python
package/test discovery and the Windows-only `fcntl`/symlink rerun requirement
(R19-10) are not yet separate rows; the floor is per-module, not per-package.

---

## Problem Statement

CI does not enforce a same-commit security matrix with stable required checks.
Negative controls can be unexercised, errors swallowed, registry fail-closed
untested, secret scans non-blocking, webhook SSRF unproven, deploy/readiness and
migration rows missing, static/CORS/proxy checks incomplete, modules omitted
from test coverage, docs/500-line invariant unchecked, zero-test packages
implicitly green, runner trust/isolation unproven, path filters able to skip
security rows, and actions/images mutable. Without these gates, earlier
remediation sprints cannot be proven at release time.

**Why:** Phase 0 required-check policy, mutation-kill acceptance, and result
vocabulary (PASS/FAIL/SKIP/NOT_RUN) are not fully implemented.
**Where:** `.github/workflows/` (validation, fleet), test harnesses, module
inventory, secret scan jobs (Spec 19 R19-01–16).

---

## Entry Gate

Before starting this sprint:

- [ ] SPRINT-A0–A3 exit gates complete (or their blocking tests exist and are green on the target commit).
- [ ] Exposure inventory, permission matrix, and secret-scan disposition (Spec 16 Gate 0) available.
- [ ] Maintainer can set GitHub branch/ruleset required-check names **outside** editable workflow YAML.
- [ ] Baseline workflow runs recorded for `team-validation.yml` / `team-validation-fleet.yml` (or current equivalents).

---

## Exit Gate

This sprint is complete only when all of the following hold:

- [ ] External required-check policy requires every applicable matrix row on PRs (including forks on hosted runners), pushes to protected `main`, and release-candidate tags/dispatches bound to an immutable SHA (R19-01). Missing/renamed/stale-SHA/`NOT_RUN`/unexplained `SKIP` blocks merge and release. — **NOT_RUN**.
- [ ] Required commands preserve nonzero exit; collection never masks failures (R19-02). — **NOT_RUN**.
- [ ] Web + MCP negative controls run real middleware/StreamableHTTP paths with positive controls and zero-side-effect assertions (R19-03). — **WIRED** + **EXERCISED** (rows `security-matrix-web-failclosed`, `security-matrix-mcp-fullpath-authz`); mutation-kill half of R19-04 now PARTIAL (see row `security-matrix-mutation-kill`).
- [ ] Mutation-kill fixtures prove each named bypass changes PASS→FAIL and are restored before publishing PASS (R19-04): auth bypass, MCP authz bypass, registry fallback, SSRF bypass, wildcard prod CORS, secret fixture, broken doc link. — **PARTIAL**: eight bypasses EXERCISED (registry fallback, SSRF, auth, MCP authz, secret fixture, broken doc link, client-IP stance, plus the readiness lifecycle gate + required-probe swallow added 2026-10-16 — row `security-matrix-mutation-kill`); wildcard prod CORS **NOT_RUN** (no negative suite drives CORS origin acceptance).
- [ ] Configured registry fail-closed matrix row (R19-05); no nil-registry legacy restoration. — **WIRED** + **EXERCISED** (row `security-matrix-registry-failclosed`).
- [ ] Secret leakage / resource boundary rows fail on real findings (R19-06); full-history Gitleaks separate from source regex checks. — **NOT_RUN** (full-history Gitleaks): redaction/no-leak half WIRED + EXERCISED (row `security-matrix-secret-redaction`); full-history Gitleaks WIRED as `scripts/gitleaks-history.sh` + EXERCISED locally over 1149 commits and a reviewed `.gitleaksignore` baseline added (11/16 findings classified as intentional test canaries / documentation placeholders and allowlisted). The remaining **5 findings look like REAL credentials** (`cpofopencode` ×4, `STATUS.md` ×1) and were **deliberately left un-ignored**, so the scan still exits 1 and is **not wired as a blocking CI row**. Blocking wiring is blocked on rotating/purging those credentials; source-regex scans **NOT_RUN**.
- [ ] Webhook SSRF deterministic resolver/dialer fixtures; policy-denied destination is failed outcome (R19-07). — **WIRED** + **EXERCISED** (row `security-matrix-webhook-ssrf`); configuration-time guard only, no dialer-level/DNS-rebinding defense (documented limitation).
- [ ] Deployment/readiness/migration rows (R19-08); static assets/CORS/proxy trust (R19-09). — **PARTIAL**: migration half WIRED + EXERCISED as spec 18 R18-01..R18-06 (`mcp-server/scripts/run_migrations.go` + `internal/database/migration_manifest_test.go`, landed `main@e4bb65d9`); **readiness/lifecycle half WIRED + EXERCISED 2026-10-16** (row `security-matrix-readiness-lifecycle`, Spec 18 R18-07..R18-13 — ordered checks, lifecycle non-ready states, `NOT_RUN` for an absent probe, registry fail-closed, policy honesty, bounded evidence; `internal/web/readiness_test.go`); the client-IP/proxy-trust half of R19-09 is WIRED + EXERCISED (row `security-matrix-clientip-trust`, Spec 12 §3.4, landed 2026-10-15); static-asset/CORS registry rows remain **NOT_RUN**.
- [ ] Complete package and test coverage: every Go module (`mcp-server`, `cmd/team-cli`, `examples/go`) + Python tests; counts recorded (R19-10). — **WIRED + EXERCISED (Go modules)**; module list is now discovered from the tree, not hardcoded — `scripts/check_go_modules.py` (fail-closed, red-on-drift, 10-case self-test) + inventory-driven CI loop; end-to-end run green (436/9/11 tests vs floors 250/8/10). Python discovery and per-package floors remain **NOT_RUN**.
- [ ] Docs gate: internal Markdown links + hard 500-line max on PR and protected push/release (R19-11). — **WIRED** (link check now `scripts/check-doc-links.sh`, shared by CI and the mutation harness; 500-line job still inline; trigger incomplete — does not fire on every protected push/release).
- [ ] Nonzero test floor with checked-in inventory and explicit SKIP/NOT_RUN classification (R19-12). — **WIRED**.
- [ ] Runner trust and repo isolation: forks/hosted; UCS03 self-hosted only for trusted `main`, isolated, cleaned (R19-13). — **NOT_RUN**.
- [ ] Path-filter safeguards: security/secret/registry/deploy/package/test-floor rows on every protected push and release candidate (R19-14). — **NOT_RUN**.
- [ ] Immutable workflow/image provenance: full commit SHA actions, image digests, identity recorded (R19-15). — **NOT_RUN**.
- [ ] Release contract fails on any FAIL/NOT_RUN/real secret/side effect/missing row/etc. (R19-16); staged rollout executed per spec. — **NOT_RUN**.
- [ ] Acceptance evidence bundle recorded (Spec 19 §5) at the exact commit SHA. — **NOT_RUN** (release evidence bundle).

---

## Scope Boundary

```
IN SCOPE (may modify):
  - .github/workflows/ required-check / matrix jobs (names stable, owned outside editable YAML where required)
  - CI test harnesses: negative controls, mutation-kill fixtures, side-effect asserts
  - Module/package inventory and test-floor configuration
  - Secret-scan jobs (full-history + env/config) with classification evidence
  - Docs link + 500-line checks; provenance pin validators
  - Path-filter dependency manifest; runner isolation probes/attestation records
  - Acceptance evidence bundle layout (no real secrets)

OUT OF SCOPE (DO NOT TOUCH):
  - Runtime authorization/registry/deployment behavior (SPRINT-A0–A3; tests may use their fixtures)
  - Branch protection / ruleset UI settings (maintainer action outside repo YAML)
  - Production credentials, production images as test targets
  - Spec 15/14 external evidence (not Phase 0 blocking matrix)
  - Unrelated workflow features, UI, docs content beyond gate reports
```

---

## Numbered Tasks

### Task 1 — Same-commit required matrix and result semantics (R19-01, R19-02)

**Action:** Define the full Phase 0 blocking matrix rows (web-negative,
mcp-negative, registry-fail-closed, secret-boundary, webhook-ssrf,
deploy-readiness, assets-cors-proxy, package-build-test, provenance,
secret-scan, docs, test-floor). Wire stable check names required outside
editable workflow YAML. Enforce result vocabulary: missing/renamed/stale-SHA/
`NOT_RUN`/unexplained `SKIP` blocks; nonzero exits preserved; warnings cannot
represent security failures.

**Acceptance criteria:**
- Matrix manifest enumerates every applicable row and result class at the audited commit.
- A deliberately missing/renamed check or swallowed nonzero exit fails the release gate.
- Fork PRs run on GitHub-hosted runners; required list recorded by maintainer.

**Expected test commands:**
```sh
# Validate workflow references and matrix presence (from repo root)
go test ./... -count=1   # or python -m pytest for harness-side checks
# Manual: gh api branch protection / ruleset required checks (maintainer-recorded)
```

### Task 2 — Web/MCP negative controls and mutation kill tests (R19-03, R19-04)

**Action:** Execute real web middleware/handler and StreamableHTTP MCP calls
with positive authorized controls and negatives: missing credentials, wrong
credentials, scope/role/resource mismatch, legacy fallback, path/method
bypasses, public-route drift, `guardrail://config` or sensitive resources. Assert
stable denial + zero protected side effect. Add mutation/fixture kill tests that
bypass protections and observe PASS→FAIL, then restore.

**Acceptance criteria:**
- Kill tests cover at minimum: auth bypass, MCP authorization bypass, configured-registry fallback, SSRF private address/redirect bypass, wildcard production CORS acceptance, secret fixture, broken documentation link.
- Suite cannot pass because a negative assertion is never exercised.

**Expected test commands:**
```sh
go test ./internal/auth ./internal/web ./internal/mcp -count=1
go test ./internal/... -run 'TestNegative|TestMutation|TestKill|TestSideEffect' -count=1
```

### Task 3 — Registry, secrets, SSRF, deploy, assets rows (R19-05 through R19-09)

**Action:**
- **Registry:** malformed/unreadable/empty/duplicate/unknown catalog/invalid expiry/revocation/unusable verifier → `NOT_READY`/startup fail or protected deny; no nil-registry legacy restore.
- **Secrets:** full-history Gitleaks, env/credential-file detection, hardcoded-secret checks fail on real findings; synthetic fixtures cannot hide real findings; logs/MCP/errors omit secrets.
- **SSRF:** deterministic resolver/dialer, no external network; private/loopback/link-local/metadata/rebinding/redirect/unsafe port never dial; test and normal delivery equal; denied destination = failed outcome.
- **Deploy:** production profiles reject wildcard/empty CORS, undeclared proxies, unsafe bind; readiness false when registry/migration/deps invalid; migration clean/repeat/rollback/incompatible tests.
- **Assets:** explicit route/method allowlist (no suffix tricks); `/api/ingest/evil.js`-style and encoded paths denied; forged forwarded IP cannot change authz/audit/rate-limit.

**Acceptance criteria:**
- Each row has deterministic fixtures and PASS/FAIL evidence; policy-denied SSRF is never success/retryable transport.
- No production credentials or production DB in any test.

**Expected test commands:**
```sh
go test ./internal/auth ./internal/web ./internal/mcp ./internal/database ./internal/config -count=1
go test ./internal/... -run 'TestRegistry|TestSecret|TestSSRF|TestWebhook|TestReadiness|TestMigration|TestCORS|TestStatic' -count=1
```

### Task 4 — Package coverage, docs, test floor (R19-10 through R19-12)

**Action:** Enumerate Go modules from the repository (not assume `go.work`);
run Python tests and all applicable Go tests/builds for `mcp-server`,
`cmd/team-cli`, `examples/go`; record discovery/execution counts; fail on
compile/race/discovery errors. Windows-only failures (`fcntl`, symlink) are
FAIL/`NOT_RUN` with required supported-runner rerun — never silent PASS. Docs
gate checks internal Markdown links and ≤500 lines per Markdown file on PR and
protected push/release. Checked-in inventory declares required packages, test
commands, positive minimum counts, applicability; zero tests with positive
floor is non-releaseable; missing toolchain/discovery/report = `NOT_RUN`.

**Acceptance criteria:**
- No module silently omitted; test counts > 0 where floor declared.
- This spec and all sprint docs remain ≤500 lines; broken links fail.
- Report shows PASS/FAIL/SKIP/NOT_RUN per row with SKIP reasons.

**Expected test commands:**
```sh
# From each Go module directory (mcp-server, cmd/team-cli, examples/go):
go test ./... -count=1
go test ./... -race -count=1   # where supported
go build ./...
# Python
python -m pytest
# Docs (project checker)
# e.g. markdown link check + line-count gate script (name per repo inventory)
```

### Task 5 — Runner trust, path filters, provenance, release contract (R19-13 through R19-16)

**Action:** PRs/forks on GitHub-hosted runners; UCS03 self-hosted only for
trusted `main` with clean checkout, no cross-repo reuse, least-privilege token,
no production credentials, bounded concurrency, post-job cleanup, isolation
probe/attestation. Path filters only narrow via dependency manifest; security
rows always run on protected push/release; unclassified/missing diff = FAIL/
`NOT_RUN`. Actions pinned to reviewed full commit SHA; images to digest;
mutation fixture (tag pin) must fail. Release fails on any blocking defect;
rollout staged (1 report-only names → 2 fixtures/mutation → 3 non-prod + hosted
PR → 4 UCS03 isolation → 5 enable blocking → 6 observe one release cycle).
Rollback = last known-good restrictive gate config; if integrity unknown, stop
protected traffic.

**Acceptance criteria:**
- Isolation probe or host attestation recorded for UCS03; fork jobs have no deployment secrets/private runners.
- Provenance test accepts only full-SHA actions and digest images; tag-only fails.
- Release bundle includes commit SHA, matrix manifest, negative/side-effect results, mutation-kill results, scan dispositions, module counts, docs report, provenance report, runner evidence (Spec 19 §5).

**Expected test commands:**
```sh
# Provenance validator (workflow action SHA / image digest checks — implement in harness)
go test ./... -run 'TestProvenance|TestPinned|TestWorkflow' -count=1
# Full matrix on PR and protected push (CI)
# Manual: release dispatch bound to immutable SHA; required-check list verified
```

---

## Dependencies

| Direction | Sprint / Requirement | Relationship |
|-----------|----------------------|--------------|
| Requires | SPRINT-A0 (R16-03–05) | Registry-fail-closed and secret-leak rows need implementation + fixtures |
| Requires | SPRINT-A1 (R16-06–08) | Web/MCP negative and audit rows need authorization behavior |
| Requires | SPRINT-A2 (R17-06–08) | Secret-boundary and argv-free rows need secret sources/redaction |
| Requires | SPRINT-A3 (R17-01–05, R18) | Deploy-readiness and migration rows need profile/migration work |
| Aligns | Spec 16 R16-02, Spec 17 R17-11 | Secret-scan reconciliation and CI/image provenance |
| Blocks | Phase 0 release claim | Cannot advertise Phase 0 complete without §5 evidence bundle |

---

## Rollback Notes

- Roll back to the last known-good **restrictive** gate configuration (previous matrix names + required-check set that still deny unsafe merges).
- If gate integrity is unknown (stale SHA, missing row, swallowed failure), stop protected traffic / halt release rather than restoring permissive fallback.
- Do not disable required checks, convert security rows to warning-only, or add blanket allowlists to make CI green.
- Mutation-kill fixtures must be restored to protected versions before any PASS is published.
- Workflow edits that grant secret access require review; unreviewed CI workflow changes must not gain deployment secrets.

---

## Do Not Proceed If

**STOP this sprint (and do not claim Phase 0 complete) if any of the following is true:**

- Any security row can be missing, renamed, `NOT_RUN`, or unexplained `SKIP` while merge/release still succeeds.
- Negative controls are helper-unit-only, or mutation-kill tests never observe PASS→FAIL.
- Registry invalidity can restore unrestricted legacy behavior in CI or runtime evidence.
- A real secret finding is suppressed, or scanner failure does not block release.
- SSRF policy-denied destinations report success or retryable transport.
- A Go module or Python tests are omitted, or zero-test floors are green by default.
- Markdown exceeds 500 lines or internal links break while docs gate is green.
- Fork PRs receive deployment secrets or private runners; UCS03 isolation is unproven.
- Actions/images use mutable tags as release proof.
- Windows-only failures are silent PASS without supported-runner rerun.
- Unrelated green checks are used to override a failed security row.

---

## Acceptance Criteria Summary

| # | Criterion | Test | Pass Condition |
|---|-----------|------|----------------|
| 1 | Same-commit matrix (R19-01–02) | Matrix manifest + exit-code checks | All rows required; failures not swallowed |
| 2 | Negative + kill (R19-03–04) | Real web/MCP + mutation fixtures | Deny + zero side effect; kill flips PASS→FAIL |
| 3 | Registry/secrets/SSRF (R19-05–07) | Deterministic fixtures | Fail closed; real findings block |
| 4 | Deploy/assets (R19-08–09) | Profile/CORS/static negatives | Reject unsafe; readiness honest |
| 5 | Coverage/docs/floor (R19-10–12) | Per-module counts + docs gate | No omission; ≤500 lines; positive floors |
| 6 | Trust/provenance/release (R19-13–16) | Isolation + pin validators + release dry-run | Restrictive gates; §5 evidence complete |

---

## Reference

- [Spec 19](../specs/19-phase0-ci-security-gates.md) — R19-01–16, §3 matrix, §5 evidence bundle
- [Spec 16](../specs/16-authentication-authorization-and-evidence-remediation.md) — R16-02 secret-scan reconciliation
- [Spec 17](../specs/17-deployment-tls-and-secret-boundary.md) — R17-11 CI and image provenance
- [Spec 18](../specs/18-migration-startup-and-readiness.md) — R18 migration/readiness (deploy-readiness row)
- Template basis: [SPRINT_TEMPLATE.md](../archive/sprints/SPRINT_TEMPLATE.md) (adapted for security remediation)

---

**Created:** 2026-10-04
**Version:** 1.4
**Status:** PARTIALLY IMPLEMENTED (reconciled 2026-10-04 against commit 1c838f4; 2026-10-05 CI rows added for web-failclosed/mcp-fullpath/registry; 2026-10-05 part 2 added secret-redaction, webhook-SSRF, and mutation-kill rows; 2026-10-05 part 3 extended mutation-kill to six bypasses, scripted the docs-gate link check, and scripted full-history Gitleaks — real result 16 findings, not yet green, so not a CI row)
