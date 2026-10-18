#!/usr/bin/env bash
# F001: fail if documented tool counts differ from tools declared in source.
#
# This used to be the whole gate. It is now a thin wrapper, and the reason is
# worth keeping in the file:
#
#   for d in $(grep -oE '(Go server with|registers) [0-9]+ tools' "$f" ...); do
#
# The loop body ran once per match. When the R17-10 landing rewrote
# mcp-server/README.md to "The server registers 39 `guardrail_*` tools.", the
# backticked token stopped matching that pattern, the loop ran zero times,
# rc stayed 0, and the gate exited 0 while the documented count was wrong.
# A gate whose scope is a regex over prose fails open when the prose is
# rewritten; it cannot tell "everything agrees" from "I found nothing".
#
# The real gate is scripts/check_tool_count.py: it counts from the Go
# registries, requires every canonical claim site to match, and treats a
# non-matching site as a violation rather than a skip. This wrapper stays so
# the documented command (docs/reboot/changes/F001.md) keeps working.
set -euo pipefail
cd "$(dirname "$0")/.."
exec python3 scripts/check_tool_count.py --repo .
