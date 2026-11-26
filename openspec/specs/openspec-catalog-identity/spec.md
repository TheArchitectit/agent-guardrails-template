# openspec-catalog-identity Specification

## Purpose

The OpenSpec catalog gives reviewers a complete, unambiguous view of specification files and their own status evidence.

## Requirements

### Requirement: Catalog lists spec files by unique path
The repository catalog SHALL enumerate every OpenSpec specification by full path and preserve distinct requirement sets even when filenames or short slugs overlap.

#### Scenario: Catalog covers all specification paths
- **WHEN** the repository specification inventory is published
- **THEN** each current specification file appears once by full path and overlapping short names remain distinguishable.

### Requirement: Template audit work item identifiers are unique
The template audit gap register SHALL assign a unique identifier to each open and archived work item. The integration audit, verification queue, and numbered change path SHALL refer to the same work item when they use that identifier.

#### Scenario: New audit proposals coexist with archived audit work
- **WHEN** an architecture proposal is added to the template audit register
- **THEN** its identifier is distinct from every existing register row, resolves to the same proposal across the integration audit and verification queue, and leaves prior archive identifiers attached to their original evidence.
