# Semantic Filtering Categories and Actions

## MODIFIED Requirements

### Requirement: bundle-declared categories

Active content categories, their actions (allow, redact, block, escalate), and their failure-mode declarations SHALL come from the signed policy bundle. A bundle selecting redact SHALL be accepted only when the configured runtime supports span removal and the required redaction evidence; otherwise the policy load or decision SHALL fail closed.

#### Scenario: undeclared category output by classifier

- **GIVEN** a classifier adapter emits a label not enabled in the active bundle
- **WHEN** the mediation engine processes the classification
- **THEN** the label is not silently enforced as a policy rule and the mismatch is recorded as a configuration warning in evidence

#### Scenario: redaction unsupported by runtime

- **GIVEN** the active bundle selects redact for a category and the runtime lacks supported span removal
- **WHEN** the bundle is loaded or the decision is evaluated
- **THEN** the selection fails closed and is not reported as an exercised redaction action
