# Design

## Context

`mcp-server/internal/guardrails/policy_pack.go` resolves and activates packs; `policy_signature.go` checks signatures against an out-of-band trusted-key set; MCP startup configures the manager. The audit notes tests for digest drift, failed-candidate state preservation, trust status, and signature errors. The production server constructs a fresh manager at startup; no runtime hot-reload caller was found. TPL-45 therefore requires two distinct evidence levels: the manager's atomic candidate-replacement behavior and the production startup activation/diagnostic path. It does not specify or imply hot reload.

## Goals / Non-Goals

**Goals:** in a disposable fixture, prove successful manager activation and production startup diagnostics, trusted signer and digest state, rule attribution, and fail-closed/refused behavior for tampering/trust failures.

**Non-Goals:** claim manager-level replacement tests prove server hot reload; change policy-pack implementation or trust policy; use production signing keys or packs; claim semantic approval from signature validity; duplicate TPL-36 assembly identity or RUN-04 generic envelope work.

## Decisions

- Use temporary pack and trust-key files and disposable MCP startup state. Never use a production/private signing key.
- At manager level, activate a verified candidate, then submit a bad candidate and assert the prior active pointer/digest remains unchanged. Label this atomic API evidence, not a production reload capability.
- Separately start the production server through its actual configuration path. Verify startup diagnostics/status for an active trusted pack and for trust-set or candidate activation errors; distinguish hard trust-set configuration failure, nonfatal pack resolution failure, and `not_configured` state as implemented.
- Record the signed-pack digest, lock entry, public key ID/fingerprint (not key material), signature-verification state, effective digest, startup/status outcome and a rule attribution. Do not include secrets.
- A valid signature establishes only the configured signer relationship. Semantic policy approval remains a separate owner decision.

## Risks / Trade-offs

- [A unit manager test is mistaken for server activation] -> keep manager atomicity and production startup tests as separate evidence sections.
- [A test describes nonexistent hot reload] -> scope prior-state retention to `PolicyPackManager` candidate activation only and state that production server constructs the manager at startup.
- [Test keys leak] -> generate ephemeral keys in a temporary directory, redact private material, and destroy the fixture after review.
- [A valid signature is overclaimed] -> label trust verification separately from semantic approval.

## Migration Plan

Run manager-level and production-startup fixtures independently, capture redacted outputs, then have an independent reviewer compare the actual active status/diagnostic to the selected manifest and trust set. Leave any unobserved claim open.
