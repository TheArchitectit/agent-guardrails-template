# Design

## Context

`clone_skill` currently strips any leading run of dots and slashes from a configured path before joining it to the target root. This changes hidden platform paths such as `.claude/skills/guardrails-enforcer.json`.

## Decisions

- Strip only the exact relative prefix `./` when present; retain all other leading path components, including a single hidden-directory dot.
- Resolve the final destination and require it to remain under the configured install root before writing.
- Test both intended hidden destinations and rejected traversal paths using temporary directories.

## Compatibility and Rollback

Correctly configured hidden paths now land at their documented locations. If a target-root containment test fails, abort before writing. Roll back normalization and containment changes together; no installed files are automatically relocated.

## Verification

Run focused installer tests with a temporary target root; assert `.claude` remains present in the output path and parent traversal cannot escape the root.
