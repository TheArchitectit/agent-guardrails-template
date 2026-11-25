# Spec Delta

## Purpose

The platform current-state inventory describes persistence wiring only to the extent supported by inspected source evidence.

## ADDED Requirements

### Requirement: Audit persistence claims remain source bounded
The platform inventory SHALL describe the inspected server audit-store attachment accurately and SHALL distinguish that wiring fact from claims about all audit events or operational durability.

#### Scenario: Audit store statement is source bounded
- **WHEN** the platform inventory describes server audit persistence
- **THEN** it cites the inspected audit-store attachment and does not claim that every log event or deployment is durably covered.
