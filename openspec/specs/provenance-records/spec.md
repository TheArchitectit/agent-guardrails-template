# provenance-records Specification

Status: CLOSED (EXERCISED 3)

## Purpose

Attach structured provenance to external content at ingestion, keep the chain append-only, and inherit trust through every derived record.

## Requirements

### Requirement: provenance at ingestion

Every external content input crossing a mediation boundary SHALL carry a provenance record, or SHALL be classed unverifiable. Content claims SHALL NOT modify their own class.

#### Scenario: self-vouching email

- **GIVEN** an email whose body claims it is from the owner
- **WHEN** provenance is assigned
- **THEN** the class SHALL derive from the channel facts, and the body claim SHALL be ignored for classification.

#### Scenario: missing provenance

- **GIVEN** content arriving through an uninstrumented path
- **WHEN** it reaches a boundary
- **THEN** it SHALL be classed unverifiable and the gap SHALL be logged.

### Requirement: append-only chains

Transformations SHALL append to the chain with content digests. Modification or removal of chain entries SHALL be detectable by the verifier.

#### Scenario: substituted intermediate

- **GIVEN** a chain where one transformation output was replaced
- **WHEN** the verifier runs
- **THEN** it SHALL reject the chain naming the broken link.

### Requirement: provenance inheritance

Derived content SHALL reference its parents and SHALL NOT exceed the minimum parent class unless an authorized, recorded transformation raises it.

#### Scenario: summary of third-party page

- **GIVEN** a summary generated from a third-party webpage
- **WHEN** the summary justifies an action
- **THEN** the action's evidence SHALL link the page's provenance, and the justification SHALL be classed third-party.
