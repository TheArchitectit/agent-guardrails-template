# Proposal: Repair validation regex cache concurrency

## Why

Concurrent validation requests can read and write `cachedRegex.lastAccess` without synchronization. `ClearRegexCache` also replaces the global `sync.Map` while callers may access it. Go's race detector has observed conflicting accesses in `CompilePattern` during `MatchPattern` corpus validation. This can make the shipped-pattern corpus fail and makes cache clearing unsafe under concurrent use.

## What Changes

- Store immutable compiled `*regexp.Regexp` values directly in the concurrent cache.
- Publish compiled patterns safely and clear the existing map without replacing it.
- Add a deterministic same-pattern and concurrent-clear regression in the validation package, plus a runnable repository regression entry that invokes it.

## Capabilities

### New Capabilities
- `validation-regex-cache`: concurrent compile, lookup, matching, and clearing are race-free while preserving current errors, matches, and timeout behavior.

## Impact

`mcp-server/internal/validation/safe_regex.go`, its Go tests, `tests/regression/`, and the template gap register. No cache eviction policy is introduced; the existing cache retains its current lifetime semantics.
