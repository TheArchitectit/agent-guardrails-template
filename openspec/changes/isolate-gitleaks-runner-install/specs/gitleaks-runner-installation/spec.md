# Isolated Gitleaks runner installation

## Purpose

Defines isolated, integrity-verified scanner setup and fail-closed execution for CI secret scanning.

## ADDED Requirements

### Requirement: CI installs the pinned scanner in job-owned scratch

The Secret Validation job SHALL install and verify its pinned Gitleaks binary in a unique directory owned by that job, using the checksum metadata published for the pinned release, without deleting or replacing shared or pre-existing scratch paths.

#### Scenario: concurrent installers share a temporary parent

- **WHEN** two installers run concurrently with the same `RUNNER_TEMP` parent
- **THEN** each atomically creates a distinct child directory, verifies v8.24.3 against its published checksum metadata, and neither installer removes or overwrites the other's files

#### Scenario: the former shared path already exists

- **WHEN** `/tmp/gitleaks.tmp` exists before a job starts
- **THEN** the job leaves that path and its contents unchanged and performs installation in its own unique scratch directory

### Requirement: Scanner setup and scan failures remain fail-closed

The Secret Validation job SHALL report success only after scanner installation, integrity/version checks, and the configured scan have all completed successfully.

#### Scenario: scanner install or verification fails

- **WHEN** download, integrity verification, or version verification fails
- **THEN** the job exits unsuccessfully and does not report a clean scan

#### Scenario: installed scanner preserves current workflow contract

- **WHEN** the isolated scanner runs in the existing Secret Validation workflow
- **THEN** it scans the same event/range with the existing credentials and license, keeps the `--exit-code=2` failure behavior, read-only `contents` permission and redaction settings, and does not enable SARIF artifact upload

### Requirement: Scanner replacement preserves workflow scope

The replacement scanner SHALL retain the workflow's existing event triggers, full-history scan range, scan arguments, license/token inputs, permission scope, redaction, failure behavior, and artifact-upload setting.

#### Scenario: inspect workflow compatibility

- **GIVEN** the scanner installation is replaced
- **WHEN** the workflow runs on a push or same-repository pull request
- **THEN** it retains full-history checkout, existing scan arguments and inputs, read-only `contents` permission, `--exit-code=2`, and disabled SARIF upload

### Requirement: Scanner path is explicit across workflow steps

When installation and scanning use separate workflow steps, the job SHALL make its verified binary available to the scan step through an explicit step-scoped path.

#### Scenario: scanner path crosses workflow steps

- **GIVEN** installation and scanning run in separate steps of one job
- **WHEN** the scan step starts
- **THEN** it resolves the verified job-owned binary through the workflow-provided step path, without relying on shell-local or runner-global `PATH` state

#### Scenario: clean and detected fixtures are evaluated

- **WHEN** the configured scanner runs against a known-clean fixture and a separate approved synthetic-detection fixture
- **THEN** the clean fixture completes successfully, the detection fixture is identified without printing its value, and production scan failure semantics remain enforced
