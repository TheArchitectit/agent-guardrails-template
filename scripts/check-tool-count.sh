#!/usr/bin/env bash
# F001: fail if documented tool counts differ from tools declared in source.
set -euo pipefail
cd "$(dirname "$0")/.."
n=$(grep -rhoE 'Name:\s+"guardrail_[a-z_]+"' mcp-server/internal/mcp --include=*.go --exclude=*_test.go | sort -u | wc -l)
rc=0
for f in README.md mcp-server/README.md; do
  for d in $(grep -oE '(Go server with|registers) [0-9]+ tools' "$f" | grep -oE '[0-9]+'); do
    if [ "$d" != "$n" ]; then echo "FAIL $f says $d, source has $n"; rc=1; else echo "PASS $f: $d"; fi
  done
done
exit $rc
