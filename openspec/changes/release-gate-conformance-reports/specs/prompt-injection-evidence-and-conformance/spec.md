# Prompt Injection Evidence and Conformance

## MODIFIED Requirements

### Requirement: adversarial conformance corpus

No prompt-injection defense capability SHALL be labeled verified without passing the versioned adversarial conformance corpus at or above the declared per-category bar, with all misses recorded as findings. A required release gate SHALL block verified promotion when the report is missing, stale, incomplete, or failing.

#### Scenario: release gate evaluation

- **GIVEN** a candidate release and the current adversarial corpus covering the declared categories
- **WHEN** the required release gate evaluates the version-bound report
- **THEN** it publishes per-category results and missed fixtures, and blocks verified promotion if the report is absent, stale, incomplete, or below a bar

### Requirement: benign-corpus false-positive budget

Conformance SHALL evaluate a benign corpus alongside the adversarial corpus and enforce the declared false-positive budget. A missing, stale, or over-budget result SHALL block verified promotion.

#### Scenario: tuning tightens classifier past budget

- **GIVEN** a bundle change that increases sensitivity
- **WHEN** the required release gate evaluates conformance against the benign corpus
- **THEN** an exceeded false-positive budget blocks verified promotion and the report records the measured rate

## ADDED Requirements

### Requirement: immutable conformance report retention

Reports used for verified promotion SHALL be retained as immutable, content-addressed artifacts bound to candidate, source revision, corpus, classifier/configuration, policy, and applicable runtime. Their digests SHALL be recorded and verified on retrieval; existing reports SHALL NOT be overwritten for the same identity.

#### Scenario: retained report integrity

- **GIVEN** a release candidate with a retained report and recorded content digest
- **WHEN** the release gate retrieves the report for promotion
- **THEN** it verifies the digest and blocks promotion if the report is missing, altered, or unavailable
