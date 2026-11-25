#!/usr/bin/env sh
# Regression test for FAILURE-ID: FAIL-4490ed64
# Bug: fresh IDE REST clients defaulted to localhost:8095, where the server
# does not listen; the Go web/REST listener is 8081 and MCP remains 8080.
# Fix: align the four IDE REST defaults/examples with 8081 and preserve
# explicit per-client overrides. The canonical Python and Go tests below
# exercise the checked-in source and handler without duplicating assertions.

set -eu
repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$repo_root"

python3 -m pytest tests/test_ide_rest_endpoint_contract.py -q
go_log=$(mktemp)
trap 'rm -f "$go_log"' EXIT HUP INT TERM
if ! (cd mcp-server && go test ./internal/web -run '^TestIDERestEndpointContract$' -count=1 -v) >"$go_log" 2>&1; then
    cat "$go_log"
    exit 1
fi
cat "$go_log"
if ! grep -F -- '--- PASS: TestIDERestEndpointContract' "$go_log" >/dev/null; then
    echo "expected Go test TestIDERestEndpointContract did not report PASS" >&2
    exit 1
fi
