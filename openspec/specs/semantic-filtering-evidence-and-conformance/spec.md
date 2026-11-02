# Semantic Filtering Evidence and Conformance

Status: PARTIAL (EXERCISED 1, WIRED 2)

## Purpose

Ensure semantic content redaction and blocking produce auditable hashed evidence without leaking raw sensitive payloads, validate categories against conformance corpora, and invalidate stale claims when classifier or policy versions change.

## Requirements

### Requirement: auditable redaction

When policy action is redact, the runtime SHALL remove the offending span, record a cryptographic digest of the removed span and the rule triggered in the evidence envelope, and never store raw sensitive content in the ledger unless explicitly policy-authorized for a restricted vault.

#### Scenario: credential redacted from tool output

- **GIVEN** tool output containing a credential pattern or classified secret
- **WHEN** redaction runs
- **THEN** the agent receives the redacted output with disclosure, and the envelope contains the span digest and category without leaking the secret into the ledger.

### Requirement: per-category conformance corpora

Each shipped category SHALL have a versioned positive and negative corpus and a declared false-positive budget; conformance SHALL gate verified status per category.

#### Scenario: category misses detection bar

- **GIVEN** a category whose corpus detection rate falls below its declared bar or whose false-positive rate exceeds its budget
- **WHEN** release conformance runs
- **THEN** that category's verified claim is blocked and the report names the failing fixtures.

### Requirement: version-bump invalidation

Incrementing a classifier or model version in the bundle SHALL invalidate prior conformance claims for affected categories until conformance is re-run against the new version.

#### Scenario: model upgraded without corpus run

- **GIVEN** a bundle updates a classifier version
- **WHEN** the bundle is evaluated for release without a matching conformance report for the new version
- **THEN** the gate flags the category claims as stale and blocks verified promotion.

## Requirement Closure

| Requirement | Status | Implementation | Verification |
|---|---|---|---|
| `auditable redaction` | **EXERCISED** | `mcp-server/internal/guardrails/content_filter.go`, `mcp-server/internal/guardrails/content_filter_policy.go` (`ActionRedact`, `ActionBlock`), `mcp-server/internal/mcp/tools_guardrails.go` | `mcp-server/internal/mcp/secret_leak_redaction_test.go`, `mcp-server/internal/mcp/argument_privacy_test.go`, `mcp-server/internal/guardrails/content_filter_test.go` |
| `per-category conformance corpora` | **WIRED** | `mcp-server/internal/guardrails/content_filter_taxonomy_test.go`, `mcp-server/internal/guardrails/content_filter_policy_test.go` (S1–S15 category rules and threshold unit tests) | `mcp-server/internal/guardrails/content_filter_taxonomy_test.go` (labelled FP <5% / FN <1% evaluation harness open in `docs/specs/guardrail-gaps-2026/STATUS.md` §02) |
| `version-bump invalidation` | **WIRED** | `mcp-server/internal/guardrails/policy_pack.go` (`LoadPolicySet`, `effectiveDigest`), `scripts/check_policy_pack_lock.py` (refuses pack version or digest drift without re-lock) | `mcp-server/internal/guardrails/policy_pack_test.go`, `scripts/test_check_policy_pack_lock.py` |
