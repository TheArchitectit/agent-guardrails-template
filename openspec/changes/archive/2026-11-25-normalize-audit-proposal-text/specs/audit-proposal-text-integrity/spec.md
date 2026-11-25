# Audit proposal text integrity

## ADDED Requirements

### Requirement: Audit proposal section citations pass the existing charset policy

Audit proposal Markdown SHALL use section citations accepted by the
repository's pinned charset gate and preserve the referenced section
numbers and ranges.

#### Scenario: A proposal cites a section range

- **WHEN** the candidate audit proposals are checked with the pinned charset gate
- **THEN** the scan finds no disallowed citation characters
- **AND** the citations retain the original section numbers and range boundaries
