# Spec Delta

## ADDED Requirements

### Requirement: Pi consumer decisions are evidenced at the event boundary
<!-- id: tpl-44-pi-mediation -->
A Pi consumer conformance claim SHALL be based on the registered production handler path and SHALL bind each tested Four Laws, classifier, and permission decision to the resulting operation effect or denial. Helper-only results SHALL NOT establish consumer enforcement.

#### Scenario: denied Pi operation is stopped before effect
- **GIVEN** a synthetic Pi operation that violates scope, receives a blocking classifier result, or is denied by session permission
- **WHEN** the registered production event handler processes it
- **THEN** the evidence records the handler, applicable decisions, and denied result, and the fixture confirms no edit or process effect occurred

#### Scenario: unavailable classifier or ambiguous permission is encountered
- **GIVEN** a classifier is unconfigured/unavailable or the permission state is `ask`
- **WHEN** the event handler evaluates an operation without an explicit user approval in the fixture
- **THEN** it does not record an allowed effect, and evidence labels the result incomplete, denied, or awaiting approval according to the actual contract

### Requirement: Pi output validation reports its actual redaction effect
<!-- id: tpl-44-pi-output -->
Pi output-validation evidence SHALL distinguish content removal from warning or side-effect logging. A redaction claim SHALL require the returned content to omit the tested span and the evidence to avoid storing the raw span; warning-only behavior SHALL NOT be described as redaction enforcement.

#### Scenario: output validator removes a synthetic sensitive span
- **GIVEN** a synthetic tool result containing a known fixture secret and an active redaction policy
- **WHEN** the real Pi output lifecycle runs
- **THEN** the returned content excludes the span, disclosure is observable, and evidence contains only a permitted digest/category/reference

#### Scenario: validator only warns or records a side effect
- **GIVEN** the real output hook leaves tool-result content unchanged
- **WHEN** a sensitive fixture reaches the hook
- **THEN** the record identifies warning/telemetry behavior and explicitly leaves redaction enforcement unverified
