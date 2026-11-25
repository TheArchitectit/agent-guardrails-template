# Spec Delta

## Purpose

Defines release-candidate metadata validation while preserving the existing tag-at-HEAD policy.

## ADDED Requirements

### Requirement: Release metadata identifies the tagged candidate
A release candidate SHALL have a VERSION value and changelog entry matching its reviewed version, and Release Coverage SHALL accept only an annotated tag with that version at the checked-out HEAD. A mismatch SHALL fail validation without changing Git refs.

#### Scenario: matching release candidate in an isolated test repository
- **GIVEN** a temporary repository has a matching changelog entry and annotated version tag at HEAD
- **WHEN** Release Coverage evaluates the candidate
- **THEN** validation passes and no remote or real repository ref is changed

#### Scenario: main has newer changes than the release tag
- **GIVEN** the VERSION tag points at the parent commit while HEAD contains a newer reviewed change
- **WHEN** Release Coverage evaluates the candidate
- **THEN** validation fails with a tag-at-HEAD mismatch and does not move or create a tag

#### Scenario: matching tag lacks release notes
- **GIVEN** an annotated version tag points at HEAD but CHANGELOG has no entry for that version
- **WHEN** Release Coverage evaluates the candidate
- **THEN** validation fails and does not publish the candidate
