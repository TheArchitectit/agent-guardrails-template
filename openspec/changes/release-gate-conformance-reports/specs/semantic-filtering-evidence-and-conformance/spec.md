# Semantic Filtering Evidence and Conformance

## MODIFIED Requirements

### Requirement: version-bump invalidation

Incrementing a classifier or model version SHALL invalidate prior claims for affected categories until conformance runs against the new version. Release promotion SHALL require a current passing report bound to the classifier/model, bundle policy, corpus versions, and candidate. Missing, stale, or failing reports SHALL block verified promotion.

#### Scenario: model upgraded without corpus run

- **GIVEN** a bundle updates a classifier version
- **WHEN** the bundle is evaluated for release without a matching conformance report for the new version
- **THEN** the gate flags category claims as stale and blocks verified promotion
