# Provenance Records

## MODIFIED Requirements

### Requirement: provenance at ingestion

Every external content input crossing a claimed mediation boundary SHALL receive provenance derived from authenticated channel facts before acceptance, or SHALL be classed unverifiable with the instrumentation gap logged. Content claims SHALL NOT modify their own class. Conformance evidence SHALL enumerate the ingress surfaces exercised; a caller-invoked scan tool SHALL NOT establish automatic coverage of other paths.

#### Scenario: self-vouching email

- **GIVEN** an email whose body claims it is from the owner
- **WHEN** provenance is assigned from channel facts
- **THEN** the body claim SHALL be ignored for classification

#### Scenario: missing provenance

- **GIVEN** content arriving through an uninstrumented path
- **WHEN** it reaches a claimed boundary
- **THEN** it SHALL be classed unverifiable and the gap SHALL be logged before the content is accepted

#### Scenario: explicit analysis tool only

- **GIVEN** a caller-invoked scan tool that tags supplied content
- **WHEN** boundary coverage is reported
- **THEN** evidence SHALL identify that tool and SHALL NOT imply automatic instrumentation of other ingress paths
