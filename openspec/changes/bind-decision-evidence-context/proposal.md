# Proposal: Bind decision evidence to active context

## Why

The injection evidence path can emit an unsigned envelope, and its policy digest is a fixed identity rather than a digest of the active policy. The envelope also lacks classifier and category fields required by the canonical claim.

## What Changes

- Require injection decision evidence to bind active policy and classifier context, category, outcome, subject, and signature-verification state.
- Scope unsigned or incomplete results to the fields they actually prove.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `prompt-injection-evidence-and-conformance`: require active decision context and verifiable signing for sealed evidence claims.

## Impact

Injection evidence construction, serialization, offline verification, and downstream reporting. It does not add sandbox runtime attestations or semantic-redaction evidence.
