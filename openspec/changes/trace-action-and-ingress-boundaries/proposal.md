# Proposal: Trace action and ingress boundaries

## Why

Injection scanning and provenance tagging are available through explicit MCP calls, but the repository does not show automatic enforcement at every action or content-ingestion boundary claimed by the canonical specs.

## What Changes

- Require call-site evidence for each claimed destructive action and external-content ingress path.
- Test action and ingress surfaces separately.
- Scope claims to explicit analysis tools when no consumer integration is present.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `classification-and-mediation`: require evidence that denial occurs before a claimed destructive effect.
- `provenance-records`: require evidence of channel-derived provenance before content acceptance.

## Impact

Action consumers, external-content ingress integrations, boundary tests, and closure reporting. It does not change classification, provenance-record format, or evidence-envelope fields.
