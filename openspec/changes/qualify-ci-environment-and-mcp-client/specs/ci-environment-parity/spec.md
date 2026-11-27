# Spec Delta

## Purpose

CI uses both GitHub-hosted and administrator-selected runners, while local validation uses separate commands. A source-level selector check does not prove runner assignment, and a passing subset must not be reported as full environment parity.

## ADDED Requirements

### Requirement: Deterministic CI gates have an explicit environment parity contract

The repository SHALL map each deterministic workflow gate to a local command and to its hosted and self-hosted execution. The mapping SHALL state prerequisites, versions, event-specific behavior, and output differences. Missing or skipped checks SHALL NOT be represented as successful parity.

#### Scenario: Full deterministic parity report

- **WHEN** a supported CI environment runs the required validation suite
- **THEN** the receipt identifies every required gate, its command, environment, tool versions, result, and any explicitly unavailable prerequisite

#### Scenario: GitHub-only evidence

- **WHEN** a check depends on GitHub events, PR comments, Actions artifacts, repository variables, branch settings, release publication, or remote-runner attestations
- **THEN** local validation labels it GitHub-only and does not create a substitute green result

### Requirement: Runner selection is verified with missing and configured variables

Jobs SHALL use the approved `CI_RUNNER_LABELS` selector with a GitHub-hosted default when the variable is unset. Verification SHALL exercise the exact expression on GitHub Actions with the variable absent and with the trusted configured runner labels, and SHALL retain fork guards before private-runner assignment.

#### Scenario: Variable absent

- **WHEN** a workflow job uses the approved selector and `CI_RUNNER_LABELS` is unset
- **THEN** GitHub assigns a GitHub-hosted `ubuntu-latest` runner and the receipt identifies the hosted environment

#### Scenario: Trusted runner configured

- **WHEN** an administrator configures the approved runner-label JSON for trusted repository workflows
- **THEN** the job uses the selected runner and its receipt identifies the actual registered runner

#### Scenario: Fork pull request

- **WHEN** a fork-origin pull request could otherwise select a private self-hosted runner
- **THEN** the job-level guard prevents assignment to that private runner, while trusted same-repository events retain their existing checks

### Requirement: Synthetic isolation tests do not depend on host Docker access

Synthetic context and filtering checks SHALL run in unique disposable paths on AI03, AI04, and AI05 without mounting host engine sockets, using privileged execution, changing runner services, or sharing mutable fixture paths. Receipts SHALL distinguish context-transport tests from engine builds.

#### Scenario: Synthetic context boundary check

- **WHEN** the synthetic isolation suite runs on one of AI03, AI04, or AI05
- **THEN** it verifies excluded and required paths using benign fixture files, reports the host and test identifier, and leaves unrelated runner state unchanged

#### Scenario: No host daemon

- **WHEN** the test runs in an environment without a Docker daemon
- **THEN** the synthetic transport check can complete without a daemon, but no image build or runtime acceptance is claimed
