# OpenSpec 19: Phase 0 Same-Commit CI Security Gates

**Status:** Proposed; not implemented. This document defines the release contract for
Phase 0 and does not claim that current workflows enforce it.
**Owner:** Project maintainer (solo-maintainer defaults below).
**Scope:** Same-commit blocking validation for web, MCP, configuration, deployment,
secret, documentation, build, and regression controls.
**Related:** [11 — authorization](11-authorization-scopes-and-roles.md),
[12 — web exposure](12-web-exposure-boundary.md), [13 — webhook SSRF](13-webhook-ssrf-hardening.md),
[16 — auth remediation](16-authentication-authorization-and-evidence-remediation.md).

## 1. Problem and source-grounded baseline

Phase 0 needs one decision surface: a commit is releasable only when every required
security and correctness control has produced an attributable result for that exact
commit. The current repository does not yet provide that contract:

- `.github/workflows/regression-guard.yml:25-31` runs both regression invocations
  with `|| true`; `:214-222` warns rather than failing when a bug-fix commit has no
  regression test.
- `.github/workflows/secret-validation.yml:18-27` scans full checkout history with
  Gitleaks and `:37-49` fails on non-example `.env` files, but
  `:54-83` makes credential-file findings warning-only and `:93-126` makes
  hardcoded-secret findings warning-only. Its triggers cover pushes/PRs for
  `main`/`develop`, not release tags or a release candidate event. Existing
  workflow actions use mutable version tags (for example `actions/checkout@v4`),
  not the immutable full-SHA pins required by Spec 17 R17-11.
- `.github/workflows/documentation-check.yml:3-8` is PR-only and path-filtered;
  its line-count and link jobs are therefore not an unconditional same-commit
  release gate. Required documentation results must also cover push/release refs.
- `.github/workflows/team-validation-fleet.yml:15-20` is a trusted `main`-only
  self-hosted UCS03 job. `.github/workflows/team-validation.yml:3-15` keeps PRs,
  including forks, on GitHub-hosted runners; this separation is required for
  isolation and must not be weakened.
- Go is split across `mcp-server/go.mod`, `cmd/team-cli/go.mod`, and
  `examples/go/go.mod`; no `go.work` was found. Current team validation tests
  `mcp-server`, builds `cmd/team-cli`, and omits `examples/go`; neither build
  is evidence that each module's tests ran. A gate that tests only one module
  is incomplete.
- Existing Windows-only failures are known evidence limits, not passes: the focused
  MCP suite previously hit Python `fcntl` and Windows symlink privilege failures
  ([16](16-authentication-authorization-and-evidence-remediation.md) §3).

This proposal closes those gaps without changing any workflow in this document.
Required status checks MUST be configured outside repository-controlled workflow
files (for example, protected-branch/ruleset required checks or an external gate
service) so a pull request cannot make its own gate optional by editing YAML.

## 2. Gate model and result semantics

### 2.1 Same-commit rule

For commit `C`, every required check MUST check out `C` (or a trusted immutable
merge ref whose tree is exactly `C`), record the commit SHA, tool versions, runner
class, command, and artifact locations, and report one terminal result. A result
from another SHA, a stale rerun, a local-only command, or a job cancelled before
its assertions execute is not evidence for `C`.

The external required-check policy MUST require all matrix rows below for the
applicable event. A green aggregate is invalid if a required row is absent,
renamed, skipped without an allowed reason, or reports `NOT_RUN`.

### 2.2 Result vocabulary

Every row and individual assertion uses exactly one of:

- **PASS:** all required assertions executed and passed.
- **FAIL:** an assertion executed and failed, a forbidden effect occurred, or a
  command returned nonzero.
- **SKIP:** the control is intentionally inapplicable under a declared, reviewed
  condition (for example no webhook feature is present); the job MUST emit the
  condition and justification. Security controls default to not skippable.
- **NOT_RUN:** setup, checkout, dependency installation, runner allocation, path
  filtering, timeout, cancellation, missing fixture, or other interruption
  prevented execution. `NOT_RUN` is non-releaseable and MUST NOT be converted to
  `SKIP` by a wrapper.

No command in a required gate may use `|| true`, `continue-on-error`, warning-only
success, broad exception swallowing, or an unconditional `exit 0`. Expected
negative tests must invert assertions explicitly: the protected operation is
expected to deny, while any unauthorized effect is a failure.

## 3. Proposed Phase 0 blocking matrix

The following check names are stable identifiers for external branch-protection
configuration. Exact implementation may use one or more jobs, but the identifiers,
coverage, status semantics, and same-commit evidence are normative.

| Check ID | Required coverage | Minimum blocking assertion |
|---|---|---|
| `R19-web-negative` | Web route, method, exposure, CORS, proxy, static-asset controls | Every unlisted route/method, suffix trick, path normalization variant, forged forwarded IP, and unauthorized mutation is denied; preflight never authorizes an effect. |
| `R19-mcp-negative` | MCP bearer, tool/resource dispatch, authorization and argument privacy | Missing, malformed, insufficient-scope, insufficient-role, wrong-project, legacy, and unauthorized resource/tool calls deny before effects; config secrets and fake argument secrets are absent from output/log evidence. |
| `R19-registry-fail-closed` | Configured registry load and verifier/catalog validation | Malformed, unreadable, empty, duplicate, unknown-scope/role, or invalid verifier registry makes readiness fail or protected web and MCP traffic deny; no legacy fallback widens access. |
| `R19-secret-boundary` | Full-history secret scan plus env/config resource scans | Gitleaks, `.env`/credential-file detection, and hardcoded-secret checks fail on a real finding; synthetic fixtures prove allowlisted examples are bounded and cannot hide a real finding. |
| `R19-webhook-ssrf` | Webhook create/update and both delivery paths | Private/special/mixed DNS answers, rebinding, prohibited literals, redirects, unsafe ports/schemes, and resolver uncertainty never reach a socket; event and test delivery enforce the same policy. |
| `R19-deploy-readiness` | Deployment profiles, listener exposure, readiness and migration | Invalid production CORS/proxy/bind settings fail closed; readiness reflects dependency/registry/migration state; migrations are deterministic, bounded, and rollback-safe. |
| `R19-assets-cors-proxy` | Static route allowlist, CORS origins, trusted proxy | Only declared assets are public; `/api/*.js`-style suffixes and unsafe methods require auth; wildcard production CORS and forged client IPs are rejected. |
| `R19-package-build-test` | Python tests and every Go module | Python tests, `go test` (including race where supported), and builds run for `mcp-server`, `cmd/team-cli`, and `examples/go`; no module may be silently omitted. |
| `R19-provenance` | Actions and image identity | Every action is pinned to a reviewed full commit SHA, production image to a digest, and the executed identity is recorded; mutable tag-only references fail. |
| `R19-secret-scan` | Dedicated immutable full-history scan | The scan covers the complete commit history for `C`, publishes detector/location/severity evidence, and fails unresolved real findings. This is separate from source regex checks. |
| `R19-docs` | Markdown links and line limit | Internal links resolve, changed and release-tree Markdown is checked on push and PR, and every Markdown file is at most 500 lines. Broken links or over-limit docs fail. |
| `R19-test-floor` | Nonzero executed-test floor and regression signal | A configured positive test count is required per applicable package/module; zero tests, missing discovery, swallowed failures, and regression-check nonzero exit are failures. |

`R19-secret-boundary` and `R19-secret-scan` may share downloaded artifacts but MUST
remain separately visible results: a regex/config finding cannot substitute for
full-history Gitleaks, and a Gitleaks pass cannot substitute for environment and
resource-boundary tests.

## 4. Normative R19 requirements

### R19-01 — Required same-commit matrix

The external required-check policy MUST require every applicable matrix row in §3
on PRs (including forks on hosted runners), pushes to protected `main`, and
release-candidate tags or manual release dispatches bound to an immutable SHA.
The solo maintainer configures and records the required-check list in GitHub
branch/ruleset settings **outside** the editable workflow YAML; if those
settings cannot be verified, the release gate is NOT_RUN. A check name MUST be
stable and owned outside the repository's editable workflow definitions. Any
missing, renamed, stale-SHA, `NOT_RUN`, or unexplained `SKIP` result blocks merge
and release.

### R19-02 — No swallowed errors

Required commands MUST preserve nonzero exit status and fail the row. Reports may
collect diagnostics, but collection MUST not mask the original failure. A warning
is informational only when the underlying assertion passed; it cannot represent a
security failure, missing test, or unresolved scan.

### R19-03 — Web and MCP negative controls

The matrix MUST execute real web middleware/handler and StreamableHTTP MCP calls,
not only helper-unit tests. It MUST include positive authorized controls and
negative controls for missing credentials, wrong credentials, scope/role/resource
mismatch, legacy fallback, path/method bypasses, public-route drift, and
`guardrail://config` or equivalent sensitive resources. Each denied case MUST
assert both the stable denial and zero protected side effect.

### R19-04 — Prove negative controls kill the gate

For each security row, the acceptance harness MUST have a mutation test or
controlled fixture that intentionally removes/bypasses the protection. The
harness MUST observe the check change from PASS to FAIL and restore the protected
version before publishing PASS. At minimum, kill tests cover: an auth bypass,
an MCP authorization bypass, a configured-registry fallback, an SSRF private
address/redirect bypass, a wildcard production CORS acceptance, a secret fixture,
and a broken documentation link. A test suite that passes only because its
negative assertion is never exercised is not acceptance evidence.

### R19-05 — Configured registry fails closed

When registry configuration is present, malformed JSON, unreadable files, empty
records, duplicate IDs/verifiers, unknown catalog values, invalid expiry/revocation
state, or unusable verifier material MUST produce `NOT_READY`/startup failure or
protected-request denial on both transports. The implementation MUST NOT set a
nil registry and restore unrestricted legacy behavior. An absent registry may be
allowed only under an explicitly bounded migration mode that cannot authorize
privileged or mutating operations.

### R19-06 — Secret leakage and resource boundaries

Full-history Gitleaks, environment-file/config filename checks, hardcoded-secret
checks, and sensitive-resource tests MUST fail on real findings. Test fixtures
MUST use unmistakably synthetic values and narrow documented exclusions only.
Logs, metrics, MCP responses, errors, and evidence MUST omit DB passwords, API
keys, verifier/JWT secrets, raw credentials, webhook secrets, and nested fake
secrets. A scanner failure or unresolved finding blocks release regardless of
whether another scanner is green.

### R19-07 — Webhook SSRF control

Acceptance tests MUST use deterministic resolver and dialer fixtures with no
external network. They MUST prove no connection for private, loopback, link-local,
metadata, special-purpose, mixed, rebinding, malformed, or policy-uncertain
answers; no redirects by default; bounded retries; and equivalent enforcement
for normal delivery and test events. A policy-denied destination is a failed
outcome, never success or a retryable transport failure.

### R19-08 — Deployment, readiness, and migration

Production profiles MUST reject wildcard/empty CORS, undeclared trusted proxies,
and unsafe bind/exposure combinations instead of silently rewriting them.
Readiness MUST be false or protected traffic denied when required registry,
migration, or dependency state is invalid. Migration tests MUST cover clean apply,
repeat/idempotent apply, partial-failure rollback, and refusal to proceed with an
incompatible schema. No test may use production credentials or a production DB.

### R19-09 — Static assets, CORS, and proxy trust

Static access MUST come from explicit route/method declarations, never filename
suffixes or arbitrary `OPTIONS`. Tests MUST cover `/api/ingest/evil.js`, method
variants, encoded/path-normalized forms, and undeclared routes. Production CORS
must enumerate origins; untrusted forwarded headers must not change audit,
rate-limit, or authorization identity. These controls align with Spec 12 and
remain blocking even when the UI is otherwise healthy.

### R19-10 — Complete package and test coverage

The package gate MUST enumerate modules from the repository and execute each Go
module, rather than assuming a `go.work` workspace. It MUST run Python tests and
all applicable Go tests/builds, record discovered and executed test counts, and
fail on compilation, race, test, or package-discovery errors. Known Windows-only
failures (`fcntl`, symlink privilege) are `FAIL` or `NOT_RUN` with a required
supported-runner rerun, never PASS and never an undocumented SKIP.

### R19-11 — Documentation and 500-line invariant

The documentation gate MUST run on PR and protected push/release events. It MUST
check internal Markdown links, including links in the new spec, and enforce a
hard maximum of 500 lines per Markdown file. Path filtering may optimize work but
must not omit release-wide checks or permit a changed documentation dependency to
escape validation. This spec itself MUST remain at or below 500 lines.

### R19-12 — Nonzero test floor and classification

A checked-in module/package inventory MUST declare each required package,
test command, positive minimum count and applicability condition before branch
protection is enabled. `mcp-server` and `cmd/team-cli` run tests and builds;
`examples/go` runs its tests when present, and a module with no tests is
explicitly `SKIP` only if the maintainer reviews a documented illustrative-only
status and another contract test covers its build. A missing toolchain,
discovery error, missing report or unclassified module is `NOT_RUN`, never
`SKIP`. A command that discovers zero tests where a positive floor is declared
is non-releaseable. The final report MUST
show PASS/FAIL/SKIP/NOT_RUN per row, the reason for every SKIP, and test counts;
there is no implicit green default.

### R19-13 — Runner trust and repository isolation

PRs and forks MUST execute on GitHub-hosted runners as in `team-validation.yml`.
Only trusted `main` validation MAY use the self-hosted UCS03 label from
`team-validation-fleet.yml`. The UCS03 job MUST be repo-scoped and isolated:
clean checkout, no cross-repository workspace/artifact/cache/credential reuse,
least-privilege token, no production credentials, bounded concurrency, and
post-job cleanup. A runner label alone is not isolation evidence. Release
acceptance MUST include an isolation probe or recorded host attestation.

### R19-14 — Path-filter safeguards

A path filter MAY select a narrower fast suite only when a dependency manifest
maps changed paths to every affected gate. Security, secret, registry,
deployment/readiness, package inventory, and test-floor checks MUST run on every
protected push and release candidate. A workflow edit, script/tool change,
configuration/schema/migration change, shared handler, auth/route/transport
change, dependency-file change, or uncertain path MUST select the full matrix.
A filter error, missing diff, shallow checkout, or unclassified path is
`NOT_RUN`/FAIL, not an automatic skip.

### R19-15 — Immutable workflow and image provenance

Required workflow actions MUST use reviewed full commit SHA references, and
release/base images MUST use immutable digests. A test validates workflow
references and the built image's recorded/executed digest at the exact commit;
a mutable tag, unknown action SHA, omitted identity, or unreviewed update is
FAIL. Fork PR jobs receive no deployment credentials or private-runner access.

### R19-16 — Release and rollout contract

A release MUST fail on any FAIL, NOT_RUN, unresolved real secret finding,
unauthorized side effect, missing required row, stale-SHA evidence, broken link,
Markdown over 500 lines, zero-test floor, package omission, readiness failure,
registry fallback, Windows-only failure without supported rerun, or unexplained
SKIP. Green unrelated checks cannot override a failed security row.

Rollout is staged: (1) implement external required-check names and report-only
classification without changing authorization behavior; (2) add deterministic
positive/negative fixtures and mutation kill tests; (3) run the full matrix on a
non-production environment and hosted PR path; (4) prove UCS03 repo isolation on
trusted `main`; (5) enable branch/release blocking for the exact commit; and
(6) observe one complete release cycle before tightening defaults. Rollback is to
the last known-good restrictive gate configuration; if gate integrity is unknown,
stop protected traffic rather than restoring permissive fallback.

## 5. Acceptance test record

Before declaring Phase 0 complete, the evidence bundle MUST include:

1. The exact commit SHA and matrix manifest showing every row and result class.
2. Positive/negative web and MCP results with side-effect assertions.
3. Mutation-kill results proving each named bypass changes PASS to FAIL.
4. Registry-invalid, secret-leakage, SSRF, CORS/proxy, readiness/migration, and
   static-asset reports with deterministic fixture identifiers.
5. Full-history scan result, detector/location/severity disposition, and proof that
   no real finding is suppressed.
6. Per-module Go and Python discovery/execution counts, build results, runner and
   tool versions, and explicit handling of Windows-only failures.
7. Documentation link/line report, including this file's line count.
8. Workflow action full-SHA and image-digest provenance report at the audited
   commit, with a mutation fixture that changes one pin to a tag and fails.
9. Hosted PR/fork evidence, trusted-main UCS03 isolation evidence, and cleanup
   confirmation; the maintainer records external required-check settings.

These requirements are proposed review criteria, not evidence that current CI has
implemented them. No workflow file is changed by this specification.
