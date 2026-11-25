# validation-regex-cache Specification

## Purpose

Define safe concurrent behavior for the compiled regular expression cache used by validation.

## ADDED Requirements

### Requirement: Concurrent cache operations are race-free

The validation regex cache SHALL permit concurrent compilation, lookup, matching, and clearing without unsynchronized access to mutable cache values or replacement of the shared map.

#### Scenario: Concurrent callers compile the same pattern and clear the cache

- **WHEN** multiple callers compile and match the same valid pattern while another caller clears the regex cache
- **THEN** all successful matches SHALL preserve the standard matching result and the race detector SHALL report no data race

### Requirement: Regex behavior remains compatible

The cache repair SHALL preserve existing compile-error wrapping, matching behavior, and caller-provided timeout handling.

#### Scenario: Invalid patterns and timed matching

- **WHEN** a caller compiles an invalid pattern or performs a match with a configured timeout
- **THEN** the existing error and timeout behavior SHALL remain unchanged
