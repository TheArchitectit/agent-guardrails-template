# Semantic Filtering Categories and Actions

Status: CLOSED (EXERCISED 3)

## Purpose

Enforce policy-declared semantic content categories and mediation actions (`allow`, `warn`, `redact`, `block`) across Guardrails classification boundaries, naming the classifier backend per decision and failing closed when classification is unavailable.

## Requirements

### Requirement: bundle-declared categories

Active content categories, their actions (allow, redact, block, escalate), and their failure-mode declarations SHALL come from the signed policy bundle.

#### Scenario: undeclared category output by classifier

- **GIVEN** a classifier adapter emits a label not enabled in the active bundle
- **WHEN** the mediation engine processes the classification
- **THEN** the label is not silently enforced as a policy rule and the mismatch is recorded as a configuration warning in evidence.

### Requirement: named classifier per decision

Every semantic filtering decision SHALL record the classifier identity, version, and configuration digest in the evidence envelope.

#### Scenario: audit query on filtered action

- **GIVEN** a mediated action in the ledger
- **WHEN** an auditor inspects the envelope
- **THEN** the exact classifier, version, category, and action taken are present and verifiable.

### Requirement: conservative boundary defaults

When a classifier fails or returns an indeterminate result at an external-egress or commit boundary, the runtime SHALL apply the category's declared failure action, defaulting to block or escalate for high-consequence categories.

#### Scenario: secret/PII classifier timeout on outbound action

- **GIVEN** an outbound action carrying content and a timeout from the secret/PII classifier
- **WHEN** mediation evaluates the failure
- **THEN** the action is blocked or escalated per the category's failure mode, never allowed silently.

## Requirement Closure

| Requirement | Status | Implementation | Verification |
|---|---|---|---|
| `bundle-declared categories` | **EXERCISED** | `mcp-server/internal/guardrails/content_filter_config.go` (`LoadContentConfig`, `Taxonomy.EnabledCategories`, `CustomCategories`), `mcp-server/internal/guardrails/content_filter_policy.go` (`NewPolicyEngineWithConfig`) | `mcp-server/internal/guardrails/content_filter_taxonomy_test.go`, `mcp-server/internal/guardrails/config_file_wiring_test.go` |
| `named classifier per decision` | **EXERCISED** | `mcp-server/internal/guardrails/content_filter.go` (`ClassificationResult.Backend`, `CategoryResult`), `mcp-server/internal/guardrails/content_filter_backends.go` (`OllamaBackend`, `OpenAIModerationBackend`) | `mcp-server/internal/guardrails/content_filter_test.go`, `mcp-server/internal/mcp/tools_guardrails_test.go` |
| `conservative boundary defaults` | **EXERCISED** | `mcp-server/internal/guardrails/content_filter.go`, `mcp-server/internal/guardrails/content_filter_config.go` (refuses `fail_policy: allow`), `mcp-server/internal/mcp/tools_guardrails.go` (`handleClassifyContent`, `handleCheckPolicy` fail closed) | `mcp-server/internal/mcp/tools_guardrails_test.go` (`TestHandleClassifyContent_NoEngine_FailClosed`, `TestHandleCheckPolicy_NoEngine_FailClosed`), `mcp-server/internal/guardrails/content_filter_policy_test.go` |
