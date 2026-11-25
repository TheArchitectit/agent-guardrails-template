# Spec Delta

## ADDED Requirements

### Requirement: Compliance report coverage is bound to live probe observations
<!-- id: tpl-47-compliance-live-probes -->
A measured compliance report SHALL derive each covered control only from a named live probe result tied to the tested runtime, active configuration, and collection interval. Declared status, simulated collection, and helper-only results SHALL NOT count as live observations. Unavailable or unmeasured controls SHALL remain distinguishable from observed controls, and empty probe input SHALL NOT produce a measured report.

#### Scenario: report includes an observed control and an unknown control
- **GIVEN** an authorized isolated runtime with one live probe reporting an observed control and another control with no usable probe
- **WHEN** the measured report is generated and exported
- **THEN** the observed row links to its named probe/source and runtime interval, while the unprobed control remains unknown or unmeasured and does not increase measured coverage

#### Scenario: declarations or simulated collection are the only inputs
- **GIVEN** a compliance declaration or legacy simulated query with no live probe observation
- **WHEN** measured reporting is requested
- **THEN** the report does not count that input as observed coverage and labels it as declared, simulated, or unsupported according to the actual path

#### Scenario: no probes are available
- **GIVEN** an empty or unusable probe set
- **WHEN** a measured report is requested
- **THEN** the operation refuses or reports a non-measured result and emits no successful coverage claim

#### Scenario: exported evidence contains sensitive fields
- **GIVEN** a probe result or configuration includes credentials, personal data, raw query output, or private endpoints
- **WHEN** the evidence artifact is retained
- **THEN** sensitive values are removed or appropriately redacted while probe identity, observation status, and verification references remain reviewable
