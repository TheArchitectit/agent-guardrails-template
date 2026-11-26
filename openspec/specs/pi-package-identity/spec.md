# pi-package-identity Specification

## Purpose

Defines one package identity for the Pi extension manifest, installation examples, and installer help.

## Requirements

### Requirement: Pi install instructions identify the published package
The Pi package manifest, README installation commands, and installer help SHALL use the verified identity `@architectit/pi-guardrails`; the internal extension install directory SHALL remain `pi-guardrails`.

#### Scenario: user follows a clean-install command
- **GIVEN** a user reads the README or installer help
- **WHEN** the user copies the published-package install command
- **THEN** it names `@architectit/pi-guardrails` and resolves to the package whose manifest has that name

#### Scenario: local installer help is checked
- **GIVEN** `install.mjs --help` runs with a temporary home directory
- **WHEN** the help text is compared with package metadata
- **THEN** the package identity matches and no files are installed or removed
