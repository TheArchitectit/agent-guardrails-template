# Provenance Records

Status: PARTIAL (EXERCISED 2, WIRED 1)

## Purpose

Attach structured provenance records (`source`, `source_path`, `trust_level`, `read_by`, `timestamp`, `hash`) to external content at ingestion, default uninstrumented or unmatched sources to lowest trust, and propagate provenance across downstream tool outputs.

## Requirements

### Requirement: provenance at ingestion

Every external content input crossing an ingestion boundary SHALL receive a provenance record with origin identifier, retrieval path, timestamp, and provenance class, or SHALL be assigned the `anonymous` class by default.

#### Scenario: uninstrumented content source

- **GIVEN** content arriving through a path that does not attach a provenance record
- **WHEN** the content reaches a mediation boundary
- **THEN** it is assigned the `anonymous` provenance class and treated as lowest trust, never assumed trusted.

### Requirement: append-only chains

Transformations of content (summarization, extraction, quoting, tool post-processing) SHALL append to the provenance chain rather than replacing prior links, and the chain SHALL be integrity-protected in the evidence envelope.

#### Scenario: chain link stripped or altered

- **GIVEN** content whose provenance chain has been truncated or modified outside an attested transformation
- **WHEN** the kernel verifier checks the envelope
- **THEN** verification fails and names the broken link.

### Requirement: provenance inheritance

When an agent action or derived artifact depends on content inputs, the resulting decision's evidence envelope SHALL reference the provenance chains of those inputs, and the effective trust class SHALL be no higher than the lowest-trust input that materially justified the action.

#### Scenario: action derived from mixed-trust inputs

- **GIVEN** an action informed by both owner-channel instructions and third-party web content
- **WHEN** policy evaluates the action and emits evidence
- **THEN** both provenance chains are linked in the envelope and third-party policy constraints apply to claims sourced from the web content.

## Requirement Closure

| Requirement | Status | Implementation | Verification |
|---|---|---|---|
| `provenance at ingestion` | **EXERCISED** | `mcp-server/internal/guardrails/provenance.go` (`ProvenanceTracker.TagContent`, `resolveTrust` defaulting to `TrustLevelUnknown`), `mcp-server/internal/guardrails/provenance_scan.go` (`ScanExternalContent`) | `mcp-server/internal/guardrails/provenance_test.go`, `mcp-server/internal/mcp/tools_indirect_injection_test.go` |
| `append-only chains` | **WIRED** | `mcp-server/internal/guardrails/provenance.go` (`MarkWithProvenance`, `LookupProvenance`), `mcp-server/internal/guardrails/provenance_cache.go`, `mcp-server/internal/mcp/evidence_wiring.go` | `mcp-server/internal/guardrails/provenance_cache_test.go`, `mcp-server/internal/mcp/evidence_test.go` (multi-hop cryptographic append-only chain verifier open in `docs/specs/guardrail-gaps-2026/STATUS.md` §05) |
| `provenance inheritance` | **EXERCISED** | `mcp-server/internal/mcp/tools_indirect_injection.go` (`handleScanExternalContent`, `handleMarkProvenance`, `handleCheckProvenance`), `mcp-server/internal/guardrails/provenance_scan.go` | `mcp-server/internal/guardrails/provenance_integration_test.go`, `mcp-server/internal/mcp/tools_indirect_injection_test.go` |
