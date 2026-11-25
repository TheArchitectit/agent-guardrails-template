# Design

## Context

`regexCache` is a package-level `sync.Map`. Its values currently wrap a compiled regular expression and a mutable `lastAccess` timestamp that is written on every cache hit. The map synchronizes its own operations, but does not synchronize access to fields of a stored pointer. `ClearRegexCache` assigns a new map value, which can race with concurrent map operations.

## Decisions

- Store `*regexp.Regexp` directly. Compiled Go regular expressions are safe for concurrent matching and are immutable for this package's use.
- Compile a miss, then use `LoadOrStore`; return the published value so concurrent misses converge on the cache's stable entry.
- Implement clearing with `regexCache.Clear()`; never replace the package-level map.
- Preserve compile error wrapping, matching results, timeout handling, and cache retention semantics.
- Add a deterministic regression with a barrier-started group of goroutines repeatedly compiling/matching one benign pattern while another goroutine clears the cache. The native Go test is the source of test logic; a Python regression entry invokes that exact test through `go test -race` from the module root.

## Verification

Run the regression against the unmodified source and require the race detector to identify the existing conflict. Then run it against the fix, repeat focused validation and corpus tests with `-race -count=1`, and run the package suite. Preserve the existing 100ms validation controls and report any timeout observations independently from race findings.

## Compatibility

The module pins Go 1.26.11, which provides `sync.Map.Clear`. No new dependencies or public API changes are introduced.
