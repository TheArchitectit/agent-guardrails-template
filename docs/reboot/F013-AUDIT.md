# F013 audit: scope validator (Oct 2, 2026)
Target: handleValidateScope in mcp-server/internal/mcp/tools_extended_validation.go.

## Findings (before)
1. String-prefix check: scope `/repo/src` accepted `/repo/src-evil/x`.
2. Empty `authorized_scope` returned Valid=true ("file allowed"). With no scope there is nothing to validate against; that is UNKNOWN, not a pass.
3. Symlinks and relative paths were not resolved before comparing.

## Changes
- `pathWithinScope`: resolves relative paths against the working directory, resolves symlinks (and the deepest existing parent for new files), then uses a boundary-aware relative-path check.
- Empty scope now returns Valid=false with an UNKNOWN message.
- Test: internal/mcp/scope_test.go (inside, scope itself, new nested file, shared-prefix sibling, `..` escape, symlink escape, outside).

## Limits
Advisory: it only runs when the agent calls the tool. It validates one path per call; it does not stop a write by itself.
Behavior change (owner verdict needed): callers that relied on empty scope meaning "allow" will now see a failure.
