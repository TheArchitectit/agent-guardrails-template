# agent-install-path-preservation Specification

## Purpose

Defines safe installer destination handling while preserving hidden platform-specific path components.

## Requirements

### Requirement: Installer preserves hidden destination components

The agent installer SHALL preserve hidden path components in configured destinations while normalizing an optional literal `./` prefix.

#### Scenario: install a hidden Claude skill

- **WHEN** an installation request targets `.claude/skills/guardrails-enforcer.json` under a configured root
- **THEN** the resulting file is placed beneath the root's `.claude/skills` directory with the hidden component preserved

### Requirement: Installer confines writes to configured root

The agent installer SHALL reject a destination whose resolved path escapes the configured installation root.

#### Scenario: destination contains parent traversal

- **WHEN** a requested destination resolves outside the configured root
- **THEN** the installer reports failure and writes no file outside that root
