# Spec Delta

## Purpose

Defines the canonical and migration-compatible request shape for IDE selection validation.

## ADDED Requirements

### Requirement: IDE selection payloads use a canonical field
New IDE clients SHALL send selected text as `selection`. The handler SHALL accept legacy `code` only when `selection` is absent and SHALL reject conflicting values when both fields are present.

#### Scenario: canonical selection request
- **GIVEN** a request contains a non-empty `selection` field
- **WHEN** the selection-validation handler processes it
- **THEN** it validates that selected text

#### Scenario: legacy client migration
- **GIVEN** a request contains non-empty legacy `code` and no `selection` field
- **WHEN** the handler processes it
- **THEN** it validates the legacy selected text during migration

#### Scenario: conflicting fields
- **GIVEN** a request contains different non-empty values for `selection` and `code`
- **WHEN** the handler processes it
- **THEN** it returns HTTP 400 without selecting either value

#### Scenario: empty selection
- **GIVEN** neither field contains non-empty selected text
- **WHEN** the handler processes the request
- **THEN** it returns HTTP 400
