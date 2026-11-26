# Spec Delta

## Purpose

The generic document store supports ID, slug, category and full-text paths. Those paths do not establish the current source permission and trusted organization isolation required for a shared document service.

## ADDED Requirements

### Requirement: Shared document access follows trusted organization and source authority
Shared document surfaces SHALL enforce trusted organization isolation and current source permissions across every supported lookup, listing, search, write and cache path. A profile that delegates retrieval SHALL identify its external enforcement owner.

#### Scenario: Permitted shared retrieval
- **WHEN** a validated caller has source permission within the trusted organization
- **THEN** only permitted content and bounded metadata are returned

#### Scenario: Cross-organization or withdrawn authority
- **WHEN** another organization reuses a logical identifier or a source grant is withdrawn
- **THEN** lookups, search results and cached reads disclose no record outside the caller's current permission

#### Scenario: External retrieval profile
- **WHEN** an adopting profile delegates permission-sensitive retrieval
- **THEN** the profile exposes only the approved repository-owned surfaces and records the external enforcement owner
