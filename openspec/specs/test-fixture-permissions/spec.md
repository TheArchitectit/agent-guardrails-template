# test-fixture-permissions Specification

## Purpose

Defines deterministic permission-sensitive test fixtures while preserving
production checks for restrictive credential and secret-file modes.

## Requirements

### Requirement: permission test fixtures have deterministic modes

Permission-sensitive test fixtures SHALL establish the intended file mode
independently of the test process umask.

#### Scenario: permissive fixture is created under restrictive umask

- **WHEN** a permission test needs a group-readable fixture while running
  under umask `0077`
- **THEN** the fixture has the intended permissive mode and the test evaluates
  the rejection path

#### Scenario: fixture is created under ordinary umask

- **WHEN** the same permission test runs under umask `0022`
- **THEN** it evaluates the same mode and produces the same pass or fail result

### Requirement: production secret mode checks remain restrictive

Tests SHALL continue to verify that production credential and secret-file
creation uses the documented owner-only mode and rejects group/world-readable
files.

#### Scenario: private secret is created

- **WHEN** production code writes a credential or secret file on a POSIX host
- **THEN** its mode remains `0600` and a group/world-readable file is rejected
