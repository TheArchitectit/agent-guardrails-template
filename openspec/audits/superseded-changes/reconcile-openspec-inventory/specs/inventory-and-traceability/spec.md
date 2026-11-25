# Inventory and Traceability Specification

## Purpose

Make repository specifications and their lifecycle evidence discoverable and
unambiguous.

## ADDED Requirements

### Requirement: complete specification inventory

The repository SHALL provide an inventory that identifies every current
specification by its full path and reports status from that specification's
own explicit status or requirement closure evidence.

#### Scenario: overlapping short identifiers

- **WHEN** two specifications share a short identifier but have distinct
  titles or requirements
- **THEN** the inventory distinguishes them by full path and preserves both
  entries

### Requirement: change-to-spec traceability

The repository SHALL provide traceability from each archived change package
to the specifications that package produced, where that relationship is
supported by repository evidence.

#### Scenario: completed package is archived

- **WHEN** a completed change is moved to the archive
- **THEN** its archive record identifies the resulting specifications and
  does not imply additional implementation evidence

### Requirement: archive manifest accuracy

An archive manifest SHALL enumerate the distinct specification outputs of
archived changes accurately and SHALL not point to absent local records as if
they were available.

#### Scenario: multiple archived changes share one output

- **WHEN** archived changes publish overlapping spec deltas
- **THEN** the manifest identifies each change's outputs and states the
  correct count of unique outputs
