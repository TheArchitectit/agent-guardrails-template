# Proposal

## Why

The agent skill installer derives a destination with `path.lstrip("./")`, which strips leading dots as well as relative-path prefixes. For a documented `.claude/skills/...` destination, the hidden `.claude` component is lost and installation no longer targets the platform's expected directory.

## What Changes

- Preserve hidden path components while normalizing only a literal leading `./`.
- Keep installation targets constrained to the configured destination root.
- Add temporary-directory coverage for hidden paths and traversal attempts.

## Capabilities

### New Capabilities
- `agent-install-path-preservation`: safe preservation of platform path components during installation.

### Modified Capabilities
- None.

## Impact

`scripts/setup_agents.py` target normalization and installer tests. Existing platform layouts and explicit target-root behavior remain supported.
