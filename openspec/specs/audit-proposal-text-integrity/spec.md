# audit-proposal-text-integrity Specification

## Purpose

Keep audit proposals readable and compatible with the pinned charset policy
while preserving the section references that support each finding.

## Requirements

### Requirement: Audit proposal section citations pass the existing charset policy

Audit proposal Markdown SHALL use section citations accepted by the
repository's pinned charset gate and preserve the referenced section
numbers and ranges.

#### Scenario: A proposal cites a section range

- **WHEN** the candidate audit proposals are checked with the pinned charset gate
- **THEN** the scan finds no disallowed citation characters
- **AND** the citations retain the original section numbers and range boundaries
