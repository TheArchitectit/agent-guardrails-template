# Spec Delta

## Purpose

Defines target resolution for platform skill installers whose destination is a single rules file.

## ADDED Requirements

### Requirement: Single-file platform skills install at their declared file path

The platform skill installer SHALL write single-file rules directly to their declared destination rather than treating that destination as a directory.

#### Scenario: install Windsurf rules individually

- **WHEN** the user installs the `windsurf-rules` skill into a target root
- **THEN** the rules are written as a regular `.windsurfrules` file directly beneath that root

### Requirement: Directory-based destinations remain directories

The platform skill installer SHALL preserve directory-based destinations for skills declared with a destination directory.

#### Scenario: install a directory-based skill

- **WHEN** a skill declares a platform destination directory
- **THEN** its source file is placed within that directory using the platform's expected relative path
