# Verify queued draft workflows on the laptop

## Why

All jobs for the template draft require the `guardrails-ucs03` Linux runner.
GitHub queued five workflows and assigned no runner. The maintainer requested
local execution when a job cannot be assigned.

## What Changes

- Replay the available verification commands from each queued workflow against
  a clean copy of the exact draft base and head.
- Record the tested command, result, and any GitHub-only step for each job.
- Keep the verification isolated from the active Windows pilot.

## Capabilities

### New Capabilities

- `local-draft-ci-evidence` records results for locally assigned draft checks.

### Modified Capabilities

None.

## Impact

Template draft #13 verification and OpenSpec evidence for its submitted checks.
