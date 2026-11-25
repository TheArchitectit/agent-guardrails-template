# Spec Delta

## ADDED Requirements

### Requirement: Runtime evidence binds policy-pack trust and activation
<!-- id: tpl-45-policy-pack-runtime -->
A policy-pack acceptance claim SHALL distinguish manager candidate atomicity from production startup activation. Startup evidence SHALL bind the source, lock record, pack digest, out-of-band signer identity and result, active effective digest, and an enforcement rule ID. A valid signature SHALL NOT imply semantic approval. Missing or mismatched evidence SHALL leave the claim unverified. Hot reload is not asserted.

#### Scenario: trusted pack activates at production startup
- **GIVEN** an isolated server configured with a locked pack and out-of-band test trust key
- **WHEN** production startup activates the pack and a fixture operation reaches the policy engine
- **THEN** evidence identifies signer ID, lock/content digest, active effective digest, and the rule governing the decision without exposing key material

#### Scenario: production startup reports invalid trust or pack
- **GIVEN** the configured trust set is unusable, locked bytes changed, or the signer/signature is invalid
- **WHEN** the production startup/activation path reads the configuration
- **THEN** it reports the implemented refusal/failed/not-configured outcome, shows no untrusted digest as active, and makes no successful enforcement claim

#### Scenario: manager refuses a failed candidate without replacing active state
- **GIVEN** the `PolicyPackManager` has an active verified set and a candidate fails resolution or verification
- **WHEN** manager-level candidate activation is invoked
- **THEN** the prior active set/digest remains unchanged; this is manager-level evidence and does not claim production hot reload
