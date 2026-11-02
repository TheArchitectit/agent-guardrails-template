# Prompt Injection Evidence and Conformance

Status: CLOSED (EXERCISED 3)

## Purpose

Bind every prompt-injection mediation decision to a verifiable evidence envelope, test detection across the six adversarial categories, and bound false positives on benign structured inputs.

## Requirements

### Requirement: per-decision evidence

Every classification and mediation decision SHALL emit a sealed evidence envelope binding the action subject, policy digest, classifier identity and version, category, and mediation outcome.

#### Scenario: blocked injection attempt

- **GIVEN** an input classified as suspected prompt injection and blocked by policy
- **WHEN** the mediation decision completes
- **THEN** an evidence envelope is emitted and verifiable offline against the active bundle.

### Requirement: adversarial conformance corpus

No prompt-injection defense capability SHALL be labeled verified without passing the versioned adversarial conformance corpus at or above the declared per-category bar, with all misses recorded as findings.

#### Scenario: release gate evaluation

- **GIVEN** a candidate Guardrails module release and the current adversarial corpus covering direct injection, instruction-in-content, encoded payloads, multi-turn setup, authority impersonation, and exfiltration lures
- **WHEN** conformance runs
- **THEN** per-category detection rates and every missed fixture are published in the conformance report, and a rate below the bar blocks the verified claim.

### Requirement: benign-corpus false-positive budget

Conformance SHALL evaluate a benign corpus alongside the adversarial corpus and enforce the declared false-positive budget.

#### Scenario: tuning tightens classifier past budget

- **GIVEN** a bundle change that increases sensitivity
- **WHEN** conformance runs against the benign corpus and the false-positive rate exceeds the declared budget
- **THEN** conformance fails and the bundle change cannot be promoted as verified.

## Requirement Closure

| Requirement | Status | Implementation | Verification |
|---|---|---|---|
| `per-decision evidence` | **EXERCISED** | `mcp-server/internal/mcp/evidence_wiring.go` (`buildEvidence`, `withEvidence`, `evidencePolicyDigest`), `mcp-server/internal/evidence/` | `mcp-server/internal/mcp/evidence_test.go`, `mcp-server/internal/mcp/tools_injection_test.go` |
| `adversarial conformance corpus` | **EXERCISED** | `mcp-server/internal/guardrails/injection_conformance.go` (`RunInjectionConformance`, `DefaultInjectionConformanceCorpus`), `config/blocklists/injection-patterns.txt`, `mcp-server/internal/guardrails/injection_detection_patterns.go` | `mcp-server/internal/guardrails/injection_conformance_test.go`, `mcp-server/internal/guardrails/injection_spec01_defaults_test.go` |
| `benign-corpus false-positive budget` | **EXERCISED** | `mcp-server/internal/guardrails/injection_detection.go` (bounded L2 perplexity score in `[0,1]`, 32-rune abstention floor) | `mcp-server/internal/guardrails/injection_l2_bounded_score_test.go` |
