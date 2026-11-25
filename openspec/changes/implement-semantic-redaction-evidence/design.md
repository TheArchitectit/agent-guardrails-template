# Design

## Context

The category/action spec declares redaction as a mediation action, while the Go filter supports block, warn, and allow variants. The evidence spec requires a removed-span digest and category without raw secret storage.

## Decisions

- Treat redaction as a first-class configured action with explicit span boundaries and deterministic replacement behavior.
- Fail policy loading or the decision closed when a selected redaction action cannot be executed.
- Return only redacted content and record a cryptographic digest, category, and rule identity; never copy the removed secret into the evidence ledger.
- Verify this path at the Go runtime boundary. A separate output validator is not evidence for this contract unless it produces the same result and evidence fields.

## Rollout

Enable redaction only for categories with supported rules and evidence. Keep unsupported categories blocked or escalated according to their declared failure policy until their redaction path is verified.
