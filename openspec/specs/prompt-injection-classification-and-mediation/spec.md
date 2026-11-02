# Prompt Injection Classification and Mediation

Status: CLOSED (EXERCISED 3)

## Purpose

Separate instruction-bearing inputs from untrusted content data across Guardrails mediation boundaries, enforce fail-closed blocking on destructive boundaries when suspected injection is detected, and disclose sanitization or mediation outcomes explicitly to callers.

## Requirements

### Requirement: instruction-channel separation

The Guardrails runtime SHALL treat user-authorized instructions and content data as distinct channels at every mediation boundary. Content data SHALL NOT be promoted to instruction authority without explicit policy mediation.

#### Scenario: content carries imperative phrasing

- **GIVEN** content ingested from a file, web page, or tool output containing imperative instructions aimed at the agent
- **WHEN** the content crosses a command, edit, git, or action mediation boundary
- **THEN** the runtime classifies the input as suspected injection rather than user instruction and mediates per bundle policy.

### Requirement: fail-closed destructive boundaries

Mediation at destructive or state-changing boundaries SHALL fail closed when classification fails, times out, or returns an indeterminate result.

#### Scenario: classifier unavailable on destructive git or command action

- **GIVEN** an agent requests a destructive git or shell action and the classifier fails or times out
- **WHEN** mediation evaluates the action
- **THEN** the action is blocked or escalated, never allowed by default, and an ERROR evidence envelope is emitted.

### Requirement: disclosure on mediation

Sanitize-and-continue mediation SHALL disclose to the caller and record in evidence that content was modified and why.

#### Scenario: suspected segment sanitized

- **GIVEN** policy permits sanitize-and-continue for a content source and a segment is neutralized
- **WHEN** the mediated content is returned to the agent
- **THEN** the response includes a disclosure marker and the envelope records the original digest, sanitized digest, and rule triggered.

## Requirement Closure

| Requirement | Status | Implementation | Verification |
|---|---|---|---|
| `instruction-channel separation` | **EXERCISED** | `mcp-server/internal/guardrails/injection_detection.go` (`SourceUser`, `SourceToolOutput`, `SourceFileContent`, `SourceAPIResponse`), `mcp-server/internal/mcp/tools_guardrails.go` (`handleDetectInjection`, `handleScanTextBatch`) | `mcp-server/internal/mcp/tools_injection_test.go`, `mcp-server/internal/guardrails/injection_detection_pipeline_test.go` |
| `fail-closed destructive boundaries` | **EXERCISED** | `mcp-server/internal/guardrails/injection_detection.go`, `mcp-server/internal/guardrails/injection_config.go`, `mcp-server/internal/mcp/tools_guardrails.go` | `mcp-server/internal/guardrails/injection_l1_failclosed_test.go`, `mcp-server/internal/guardrails/injection_l3_wiring_test.go`, `mcp-server/internal/guardrails/injection_startup_test.go` |
| `disclosure on mediation` | **EXERCISED** | `mcp-server/internal/guardrails/provenance_scan.go` (`ScanExternalContent`), `mcp-server/internal/mcp/evidence_wiring.go` (`withEvidence`, `buildEvidence`) | `mcp-server/internal/mcp/tools_indirect_injection_test.go`, `mcp-server/internal/mcp/evidence_test.go` |
