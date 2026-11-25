# openspec-archive-provenance Specification

## Purpose

The archived-change manifest preserves verifiable links from historical packages to their produced specification files.

## Requirements

### Requirement: Archive manifest matches package outputs
Each archived OpenSpec package manifest SHALL enumerate its actual specification outputs and SHALL resolve or explicitly identify unavailable historical references without inventing provenance.

#### Scenario: Archive manifest matches package outputs
- **WHEN** an archived package is listed in the manifest
- **THEN** its enumerated output paths match the package files and every historical reference resolves or is marked unavailable.
