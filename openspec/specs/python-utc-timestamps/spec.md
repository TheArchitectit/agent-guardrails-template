# python-utc-timestamps Specification

## Purpose

Defines UTC timestamp handling for repository Python utilities, including
consistent time comparisons and compatible serialized UTC values.

## Requirements

### Requirement: UTC time comparisons use aware values

Python utilities SHALL use timezone-aware UTC values when creating or comparing
instants and time-window boundaries.

#### Scenario: filter timestamped records by a UTC time window

- **WHEN** a time-window filter compares a generated UTC boundary with a
  serialized timestamp ending in `Z`
- **THEN** both values are interpreted as UTC-aware instants and the comparison
  completes without naive/aware type errors

### Requirement: serialized UTC timestamp shape stays compatible

Python utilities SHALL preserve the established JSON timestamp field names
and ISO timestamp strings ending in `Z` when emitting UTC instants.

#### Scenario: emit a structured log or metric timestamp

- **WHEN** a script emits a timestamp for a structured log, metric, audit
  record, or generated report
- **THEN** it retains the existing field name and serializes the UTC value as
  an ISO timestamp with a terminal `Z`

#### Scenario: parse an emitted UTC timestamp

- **WHEN** an existing consumer parses an emitted timestamp with the current
  ISO-and-`Z` format
- **THEN** it continues to interpret the value as UTC without a schema or key
  migration
