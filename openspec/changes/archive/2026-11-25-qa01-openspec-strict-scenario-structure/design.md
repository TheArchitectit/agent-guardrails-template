# Design

## Context

OpenSpec CLI 2.2.0 strict validation reports seven spec failures caused by scenario formatting. All four active changes validate. The failure is a document conformance gap and does not negate implementation evidence or establish behavior defects.

## Goals / Non-Goals

**Goals:** Ensure every repository spec is included in the strict check and each scenario expresses a trigger and outcome.

**Non-Goals:** Change requirement intent, weaken strict mode, or use validation status as proof that implementation is complete.

## Decisions

- Preserve existing scenario meaning and rewrite only structure needed for WHEN/GIVEN and THEN parsing.
- Validate the complete tree with the pinned/official OpenSpec CLI and retain before/after counts.
- Treat CI enforcement as a separate implementation choice; do not mark it complete until an actual required workflow invokes the strict command.

## Risks / Trade-offs

- Mechanical formatting may obscure distinctions among distinct short/verbose specs; coordinate changes with `INV-01` without folding their requirements together.
