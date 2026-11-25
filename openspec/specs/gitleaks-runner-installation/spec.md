# gitleaks-runner-installation Specification

## Purpose

Establish a reproducible scanner install for secret validation and keep scanner acceptance available on the laptop and the project's self-hosted CI.

## Requirements

### Requirement: Scanner scratch belongs to one invocation

The scanner command SHALL create a unique temporary directory and clean up only files belonging to its invocation.

#### Scenario: concurrent installers use one parent
- **WHEN** two invocations share the same temporary parent or the old fixed path already exists
- **THEN** the installers use different directories and leave the pre-existing path intact

### Requirement: Scanner integrity is verified before execution

The scanner command SHALL compare the pinned release archive against a reviewed SHA-256 value before extraction and SHALL check the installed executable's exact version before scanning. The scanner SHALL exit unsuccessfully on any install, verification, or scan failure.

#### Scenario: incorrect scanner bytes or version
- **WHEN** the archive digest or reported scanner version disagrees with the repository pin
- **THEN** the command fails before scanning and does not report a clean result

### Requirement: The scan preserves event and commit coverage

Secret Validation SHALL preserve the existing push and same-repository pull-request triggers, full-history checkout, and read-only permission. The scan SHALL include the history newly introduced by the event and SHALL validate the commit boundaries before scanning.

#### Scenario: a pull request advances its head
- **WHEN** the target and head are available as pinned commits
- **THEN** the scanner checks the history between their merge base and the event's head, including introduced merge history

#### Scenario: a ref is newly created or rewritten
- **WHEN** the event has a valid head but an all-zero or unavailable ancestor boundary
- **THEN** the scanner checks available full history at that head and a shallow or unresolved checkout fails closed

### Requirement: Scanner use is available locally and reports safe outcomes

The repository SHALL provide the same verified CLI path for local and self-hosted execution without licensed Action inputs. The command SHALL use full redaction, report detections as failure, and summarize status without publishing scan contents or artifacts.

#### Scenario: local clean and detected fixtures
- **WHEN** the verified command runs against a clean repository and a separate approved synthetic-detection repository
- **THEN** the clean run succeeds, the detection run fails with the defined finding status, and the fixture value is absent from emitted output
