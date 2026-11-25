# Repository QA and Distribution Specification

## Purpose

Make repository QA results cover the declared specification set and make
continuous integration and installed command-line behavior reproducible.

## ADDED Requirements

### Requirement: complete strict specification validation

Repository QA SHALL discover and strictly validate every current OpenSpec
specification and proposed change, and SHALL report failures separately from
successful validation of a subset.

#### Scenario: indexed subset passes but other specs fail

- **WHEN** strict validation passes indexed specs but fails discovered specs
  outside the index
- **THEN** the QA result reports the full discovered count, pass and fail
  counts, and the affected paths

### Requirement: permission tests are host-umask independent

Tests that verify private credential or configuration file permissions SHALL
set and inspect fixture modes explicitly, independent of the host process
umask, while retaining the production permission expectations.

#### Scenario: restrictive umask changes fixture mode

- **WHEN** the test process uses an umask that removes permissions from a
  requested fixture mode
- **THEN** the test fixture establishes the intended mode explicitly and
  verifies the resulting mode before making its assertion

### Requirement: Python timestamps are timezone-aware

Python QA code SHALL serialize UTC timestamps with timezone-aware APIs and
preserve the established serialized timestamp format.

#### Scenario: supported Python warns on naive UTC API

- **WHEN** the supported Python runtime emits deprecation warnings for naive
  UTC timestamp construction
- **THEN** the suite uses timezone-aware UTC timestamps and preserves the
  expected serialized values

### Requirement: dependency advisories are reviewed in CI

CI SHALL run a dated dependency advisory check for locked dependencies and
report development and runtime dependency results separately.

#### Scenario: advisory source reports development dependency findings

- **WHEN** the advisory source reports affected locked development packages
- **THEN** the report records the advisory date and affected dependency scope,
  and CI exposes the result for review

### Requirement: CI inputs match repository contents

CI jobs SHALL install dependencies from a manifest that exists at the declared
path in the checked-out repository.

#### Scenario: configured dependency manifest is absent

- **WHEN** a CI job invokes dependency installation using a missing manifest
- **THEN** the configuration is considered invalid until the manifest path or
  installation command is reconciled

### Requirement: CI toolchain matches module requirements

CI jobs building a Go module SHALL use a documented toolchain compatible with
the module's declared Go version and SHALL make toolchain acquisition behavior
explicit.

#### Scenario: CI declares an older Go version

- **WHEN** the configured Go version is older than the module declaration
- **THEN** the QA record identifies whether the job obtains the required
  toolchain explicitly and does not assume an implicit download will occur

### Requirement: installed CLI backend resolution

An installed team CLI SHALL resolve its required Python backend from a
documented installed location or an explicit configuration value, independent
of the caller's working directory.

#### Scenario: command is run outside the source checkout

- **WHEN** an installed CLI is invoked from a directory outside the checkout
- **THEN** it locates its backend or returns an actionable configuration
  message describing the supported backend path

### Requirement: IDE client and server contract alignment

IDE integrations SHALL initialize their connection configuration before
their first request and SHALL use the REST service's configured address and
request schema.

#### Scenario: initial connection check runs

- **WHEN** an IDE extension activates and checks its configured server
- **THEN** it uses initialized client settings and reports the actual
  connection result

#### Scenario: IDE validation request is sent

- **WHEN** an IDE client requests selection validation
- **THEN** its request fields match the server handler contract and the
  configured REST endpoint

### Requirement: install targets preserve platform paths

Installation tooling SHALL preserve platform-specific filenames and hidden
directory names when resolving destinations.

#### Scenario: hidden configuration file is selected

- **WHEN** an installation target begins with a dot or is itself a hidden
  rules filename
- **THEN** installation writes to the exact documented target path

### Requirement: package metadata matches install instructions

Published package metadata SHALL use the package identifier named by its
installation instructions, and required test dependencies SHALL be available
from a clean declared dependency install.

#### Scenario: clean package installation is tested

- **WHEN** a consumer installs the documented package and runs its test suite
- **THEN** package resolution and declared dependencies permit the documented
  install and test commands to complete

### Requirement: status claims preserve evidence dates and residuals

Repository status documents SHALL distinguish dated historical baselines from
current code/test evidence and SHALL identify residual requirements when a
document's overall status is partial or proposed.

#### Scenario: current evidence supersedes a dated baseline

- **WHEN** current source and named tests contradict an earlier dated status
  statement
- **THEN** the document labels the earlier statement as historical and
  identifies the current evidence and remaining open requirements
