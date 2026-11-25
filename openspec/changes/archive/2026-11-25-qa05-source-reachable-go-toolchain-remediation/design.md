# Design

## Context

The merged #12 change at `943bcf6` updates `mcp-server/go.mod`, Go workflow selection, image pins and the Docker builder. The paired before/after scan and build record is in `evidence.md`, copied from `strengthen-repository-qa/qa-05-evidence.md`. This change adds only the bounded `go-advisory-evidence` contract for preserving that record.

## Goals / Non-Goals

**Goals:** Preserve the tested #12 result, its exact Go 1.26.11 build alignment, dated before/after reachability counts, four residual module-only findings, and evidence limitations in a standalone contract and evidence artifact.

**Non-Goals:** Add a continuous Go advisory scan or CI gate (tracked separately as TPL-34), resolve the four module-only residuals, or extend this record to unrelated QA issues.

## Decisions

- Keep the existing QA-05 evidence text intact as a copy, with the merged commit and source artifact cited for provenance.
- Add the `go-advisory-evidence` delta as the sole capability in this change; it requires the evidence record to identify exact toolchain/build pins, paired reachability results, residuals, and build scope.
- Publish the evidence contract when archiving the verified change. Use the merged #12 result and preserved build and scan record as its acceptance evidence.
- Do not imply that govulncheck reports standardized severity or that the dependency graph is clear: four module-only advisories remain and are documented.

## Risks / Trade-offs

- The finding database is dated and govulncheck reports reachability tiers, not standardized severity; retain its timestamp and counts in the evidence.
- Four module-only advisories remain in the dependency graph; preserve them as residuals rather than presenting the scan as zero findings.
