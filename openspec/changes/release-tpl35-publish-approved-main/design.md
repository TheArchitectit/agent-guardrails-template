# Design

## Context

Release Coverage compares `VERSION` to an annotated tag at HEAD and checks for a changelog entry. Current main differs from the existing `v3.8.11` tag. These checks can be exercised in a temporary Git repository without creating a tag or release in the actual repository.

## Goals / Non-Goals

**Goals:**

- Prepare reviewed release metadata and test the configured tag-at-HEAD contract.
- Keep local acceptance tests isolated from real repository refs and remote publication.

**Non-Goals:**

- Weaken or bypass Release Coverage.
- Create or publish an actual tag/release as part of package implementation or tests.

## Decisions

- Preserve the existing requirement that VERSION identifies an annotated tag at the checked-out HEAD and CHANGELOG contains a matching entry.
- Use temporary Git repositories with annotated matching tags and mismatch controls; verify both successful validation and fail-closed rejection without touching real refs.
- Treat actual tag creation and publication as a separate authorized release-owner operation after metadata and required checks are reviewed.

## Risks / Trade-offs

- [A fixture test accidentally touches real refs] → create all test commits/tags under a temporary repository and assert the real repository ref list is unchanged.
- [Metadata is prepared before approval] → keep the version/notes task blocked until the release owner approves the candidate.

## Migration Plan

Prepare VERSION and CHANGELOG only after approval, run the isolated release-gate tests, and leave actual publication to the authorized release follow-up.
