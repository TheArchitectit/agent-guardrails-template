#!/usr/bin/env bash
# Regression test for failure FAIL-01925f50.
#
# Bug: Concurrent CompilePattern cache hits wrote shared lastAccess metadata,
# and concurrent ClearRegexCache replaced the shared sync.Map. The Go race
# detector identified conflicting writes during MatchPattern corpus validation.
# Fix: Cache immutable compiled regexp pointers and clear the existing sync.Map.
# This entry invokes the native Go regression; it does not duplicate its logic.

set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)
cd "$REPO_ROOT/mcp-server"
if output=$(go test -v -race -count=1 -run '^TestCompilePatternConcurrentClear$' ./internal/validation 2>&1); then
	printf '%s\n' "$output"
else
	status=$?
	printf '%s\n' "$output" >&2
	exit "$status"
fi

if ! printf '%s\n' "$output" | grep -F -- '--- PASS: TestCompilePatternConcurrentClear (' >/dev/null; then
	echo "expected native race regression did not run or pass" >&2
	exit 1
fi
