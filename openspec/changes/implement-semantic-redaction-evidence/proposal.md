# Proposal: Implement semantic redaction evidence

## Why

Canonical semantic specs claim that redaction removes sensitive spans and records a digest, but the Go content-filter action set does not currently support redaction. A separate optional Pi output scrub does not provide the claimed decision evidence.

## What Changes

- Require the configured semantic runtime to remove declared spans and emit category and digest evidence without retaining raw secret content.
- Fail closed for unsupported redaction configuration and do not report it as exercised.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `semantic-filtering-evidence-and-conformance`: require auditable span removal and evidence.
- `semantic-filtering-categories-and-actions`: require redaction support when selected by the signed bundle.

## Impact

Semantic action configuration, filter runtime, and evidence ledger. It does not change corpus conformance or report-gate requirements.
