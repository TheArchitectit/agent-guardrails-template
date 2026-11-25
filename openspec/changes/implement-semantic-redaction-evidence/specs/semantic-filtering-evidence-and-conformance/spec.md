# Semantic Filtering Evidence and Conformance

## MODIFIED Requirements

### Requirement: auditable redaction

When policy action is redact, the runtime SHALL remove the offending span before returning content, record a cryptographic digest of the removed span, category, and triggered rule in the evidence envelope, and never store raw sensitive content in the ledger unless explicitly policy-authorized for a restricted vault. If the configured runtime cannot perform span removal or produce the required evidence, policy loading or the decision SHALL fail closed and SHALL NOT report redaction as exercised.

#### Scenario: credential redacted from tool output

- **GIVEN** tool output containing a credential pattern or classified secret
- **WHEN** a supported redaction rule runs
- **THEN** the agent receives the output with the declared span removed and disclosure, and the envelope contains its digest, category, and rule without the secret in the ledger

#### Scenario: redaction action unsupported

- **GIVEN** a policy selects redaction but the configured runtime lacks span-removal support
- **WHEN** the policy loads or the decision is evaluated
- **THEN** evaluation fails closed and does not claim redaction was exercised
