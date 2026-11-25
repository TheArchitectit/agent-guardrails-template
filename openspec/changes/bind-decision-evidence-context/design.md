# Design

## Context

The evidence builder attaches decision metadata and signs only when a signer is configured. Its current fixed digest cannot show which active policy produced a decision.

## Decisions

- Compute the policy digest from the active policy or bundle bytes and identify the classifier and version used for the decision.
- Include subject, category, outcome, and explicit signature state in the serialized envelope.
- Require offline signature verification for claims that say evidence is sealed; label unsigned records as unsigned and do not make the sealed claim.
- Preserve reader compatibility for historical envelopes while distinguishing absent fields from verified values.

## Rollout

Emit the full context first and test signed and unsigned paths. Promotion claims remain scoped until verifiers enforce signatures for evidence advertised as sealed.
