# commit-attribution-records Specification

## Purpose

Ensure the compliance workflow inspects the complete commit messages selected for the event.

## Requirements

### Requirement: Attribution checks use complete event-scoped commit messages

The workflow SHALL compute the event commit range via its shared helper and SHALL inspect each complete message. It SHALL report missing attribution as advisory information and fail when it cannot inspect the computed range.

#### Scenario: a commit body spans several lines
- **WHEN** a selected commit contains a keyword and the expected attribution on different lines
- **THEN** both lines belong to the same inspected commit and the attribution is recognized

#### Scenario: a commit body resembles a delimiter
- **WHEN** a selected body contains text that resembles a record separator
- **THEN** the workflow preserves its whole body and separately checks the next commit

#### Scenario: the range is unavailable
- **WHEN** the shared helper cannot resolve the event range or Git cannot read a selected commit
- **THEN** the job fails and reports that the inspection could not complete

#### Scenario: the range is empty
- **WHEN** the shared helper successfully returns an empty range
- **THEN** the workflow explicitly reports zero commits available for inspection
