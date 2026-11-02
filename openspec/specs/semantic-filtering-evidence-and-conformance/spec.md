# Semantic Filtering Evidence and Conformance

Status: CLOSED (EXERCISED 3)

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
| `per-category conformance corpora` | **EXERCISED** | `mcp-server/internal/guardrails/content_filter_conformance.go` (`RunSemanticConformance`, `DefaultSemanticConformanceCorpus`, `ReferenceSemanticClassifier`), `mcp-server/internal/guardrails/content_filter_policy.go` | `mcp-server/internal/guardrails/content_filter_conformance_test.go`, `mcp-server/internal/guardrails/content_filter_taxonomy_test.go` |
| `version-bump invalidation` | **EXERCISED** | `mcp-server/internal/guardrails/content_filter_conformance.go` (`SemanticDriftMonitor.VerifyClaim`, `SemanticDriftMonitor.CheckDrift`), `mcp-server/internal/guardrails/policy_pack.go` | `mcp-server/internal/guardrails/content_filter_conformance_test.go`, `mcp-server/internal/guardrails/policy_pack_test.go` |
