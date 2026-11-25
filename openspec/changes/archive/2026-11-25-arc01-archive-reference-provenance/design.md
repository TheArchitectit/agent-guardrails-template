# Design

## Context

There are four archived `aiggp-03` through `aiggp-06` package folders. The dated audit identifies seven unique short-spec outputs overall and a retirement reference that is not present in the reviewed checkout.

## Goals / Non-Goals

**Goals:** Make the archive manifest internally accurate and make historical links resolvable or clearly external/historical.

**Non-Goals:** Change archived requirement text, reinterpret checked historical tasks as current implementation verification, or update the present-day spec index (`INV-01`).

## Decisions

- Recount outputs from actual archived delta paths rather than package headers alone.
- Follow each retirement reference before editing it; preserve provenance where a replacement source can be verified.
- Validate local Markdown links and OpenSpec archived-change inventory after the correction.

## Risks / Trade-offs

- Historical retirement records may not be present in this snapshot; if provenance cannot be verified, retain the dated reference context and report the unresolved source rather than fabricate a replacement.
