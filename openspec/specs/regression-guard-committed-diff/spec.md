# regression-guard-committed-diff Specification

## Purpose

Defines the regression guard's committed-CI range and local working-tree scopes.

## Requirements

### Requirement: CI evaluates the committed change range
The regression guard SHALL use the event's exact revisions. For pull requests it SHALL scan the unique merge-base-to-head patch. For pushes it SHALL scan before-to-head, with before an ancestor of head. Missing or unresolvable revisions, shallow or ambiguous history, and empty file ranges SHALL fail the check.

#### Scenario: committed match in a pull request
- **GIVEN** a temporary repository has a committed base and a head commit that adds a benign string matching a registered regression rule
- **WHEN** CI invokes the guard with that base and head
- **THEN** the guard reports the changed file and matching rule

#### Scenario: clean committed range
- **GIVEN** the base-to-head range contains no changed file matching a registry or pattern rule
- **WHEN** CI invokes the guard with the resolved range
- **THEN** it reports no potential regression after evaluating the non-empty range

#### Scenario: divergent PR uses the unique merge base
- **GIVEN** the event base advanced after the PR head forked, and a benign matching sentinel exists only in a base-side commit
- **WHEN** CI invokes the guard in PR merge-base mode with the event's exact base and head revisions
- **THEN** it evaluates the unique common ancestor to PR-head patch and does not report the base-only sentinel

#### Scenario: ambiguous or shallow PR history
- **GIVEN** the PR revisions have no unique merge base or the checkout is shallow
- **WHEN** CI invokes the guard in PR merge-base mode
- **THEN** it exits nonzero with a history error

#### Scenario: push base is not an ancestor
- **GIVEN** the push before revision does not precede the pushed head
- **WHEN** CI invokes the guard in push range mode
- **THEN** it exits nonzero instead of scanning a misleading range

#### Scenario: missing base revision
- **GIVEN** the CI event base does not resolve or checkout history is incomplete
- **WHEN** the guard is invoked
- **THEN** it exits nonzero with the unavailable-range cause instead of reporting a green empty scan

### Requirement: Local mode evaluates each selected patch
Local staged and unstaged modes SHALL inspect their corresponding patch content. The `--all` mode SHALL evaluate each patch independently before combining findings.

#### Scenario: local staged and unstaged review
- **GIVEN** a developer has staged or unstaged edits in a working tree
- **WHEN** the developer invokes the existing local mode
- **THEN** the guard continues to evaluate those edits without requiring event SHAs

#### Scenario: combined local mode sees unstaged pattern content
- **GIVEN** a benign registered pattern is added only in an unstaged file
- **WHEN** a developer invokes `--all`
- **THEN** the guard evaluates the unstaged patch and reports the matching rule

#### Scenario: forbidden context is patch-local
- **GIVEN** the staged patch adds a matching sentinel and the unstaged patch adds only its forbidden-context marker
- **WHEN** a developer invokes `--all`
- **THEN** the staged match is still reported because each patch is evaluated independently
