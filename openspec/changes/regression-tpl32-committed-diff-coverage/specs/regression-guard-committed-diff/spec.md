# Spec Delta

## Purpose

Defines the regression guard's committed-CI range and local working-tree scopes.

## ADDED Requirements

### Requirement: CI evaluates the committed change range
For pull requests and pushes, the regression guard SHALL evaluate the event's full base-to-head patch; an unavailable base, incomplete history, or empty CI range SHALL fail rather than report a clean result. Local staged and unstaged modes SHALL inspect their corresponding patch content; `--all` SHALL inspect both staged and unstaged content.

#### Scenario: committed match in a pull request
- **GIVEN** a temporary repository has a committed base and a head commit that adds a benign string matching a registered regression rule
- **WHEN** CI invokes the guard with that base and head
- **THEN** the guard reports the changed file and matching rule

#### Scenario: clean committed range
- **GIVEN** the base-to-head range contains no changed file matching a registry or pattern rule
- **WHEN** CI invokes the guard with the resolved range
- **THEN** it reports no potential regression after evaluating the non-empty range

#### Scenario: missing base revision
- **GIVEN** the CI event base does not resolve or checkout history is incomplete
- **WHEN** the guard is invoked
- **THEN** it exits nonzero with the unavailable-range cause instead of reporting a green empty scan

#### Scenario: local staged and unstaged review
- **GIVEN** a developer has staged or unstaged edits in a working tree
- **WHEN** the developer invokes the existing local mode
- **THEN** the guard continues to evaluate those edits without requiring event SHAs

#### Scenario: combined local mode sees unstaged pattern content
- **GIVEN** a benign registered pattern is added only in an unstaged file
- **WHEN** a developer invokes `--all`
- **THEN** the guard evaluates the unstaged patch and reports the matching rule
