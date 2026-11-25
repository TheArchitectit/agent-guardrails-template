# Spec Delta

## Purpose

The OpenSpec catalog gives reviewers a complete, unambiguous view of specification files and their own status evidence.

## ADDED Requirements

### Requirement: Catalog lists spec files by unique path
The repository catalog SHALL enumerate every OpenSpec specification by full path and preserve distinct requirement sets even when filenames or short slugs overlap.

#### Scenario: Catalog covers all specification paths
- **WHEN** the repository specification inventory is published
- **THEN** each current specification file appears once by full path and overlapping short names remain distinguishable.
