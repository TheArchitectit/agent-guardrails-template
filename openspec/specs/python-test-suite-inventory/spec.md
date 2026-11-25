# python-test-suite-inventory Specification

## Purpose

Defines how discovered Python test suites are represented in the repository inventory so CI can run every suite and detect collection drift.

## Requirements

### Requirement: Every discovered suite has an executable inventory row
Every directory discovered as a Python test suite SHALL have one inventory row declaring its path, test command, setup when needed, applicability, and positive minimum collection floor. The floor SHALL be no greater than the suite's current collected test count.

#### Scenario: Root tests suite is covered by the inventory
- **GIVEN** the repository contains collectable tests directly under `tests/`
- **WHEN** the Python suite discovery gate checks the inventory
- **THEN** it finds a row for `tests/` with a runnable pytest command and positive floor no higher than the current collection

#### Scenario: Unregistered discovered suite fails coverage
- **GIVEN** a collectable Python test directory has no inventory row
- **WHEN** the Python suite discovery gate checks coverage
- **THEN** it exits nonzero and identifies the unregistered suite

#### Scenario: Collection floor tracks a nonempty suite
- **GIVEN** the root `tests/` suite has a current collection of 11 cases, including any conditionally skipped Docker integration test
- **WHEN** CI collects and runs that suite
- **THEN** the inventory floor is 11 and the ordinary pytest command runs the collected suite, allowing only the Docker case's declared skip condition
