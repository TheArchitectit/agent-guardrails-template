# openspec-strict-validation Specification

## Purpose

Strict validation reports specification structure across the complete repository inventory.

## Requirements

### Requirement: Strict validation covers every specification
The repository strict-validation procedure SHALL include every current OpenSpec specification and report scenario-format failures without treating validation as implementation evidence.

#### Scenario: Strict validation covers all specs
- **WHEN** the repository strict-validation procedure runs
- **THEN** every current specification is checked and each scenario-format failure is reported separately from implementation status.
