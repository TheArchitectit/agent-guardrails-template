# Design

## Context

The MCP scan tools accept caller-provided content and return findings. The reviewed tree does not establish that command, edit, git, send, file, web, email, or other claimed consumers invoke those tools before effects or acceptance.

## Decisions

- Maintain separate inventories and acceptance results for action boundaries and ingestion boundaries.
- For each action surface, show the supported caller path invokes classification and policy before effect, and that a benign denied decision prevents the effect.
- For each ingestion surface, show trusted channel facts create provenance before content is accepted; missing instrumentation is recorded as unverifiable.
- Test caller-invoked MCP tools as explicit analysis surfaces only; do not infer automatic integration from their existence.
- Narrow any unintegrated claims to the call sites actually demonstrated.

## Rollout

Add integrations incrementally and publish a boundary-to-call-site matrix. Each surface remains unverified until its own safe acceptance cases pass.
