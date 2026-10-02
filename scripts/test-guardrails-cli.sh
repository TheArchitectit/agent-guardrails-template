#!/usr/bin/env bash
set -euo pipefail
cli="$(cd "$(dirname "$0")" && pwd)/guardrails-cli.sh"; t=$(mktemp -d); trap 'rm -rf "$t"' EXIT
fail(){ echo "FAIL $1"; exit 1; }
# clean init, then uninstall removes everything
"$cli" init "$t/a" >/dev/null; [ -f "$t/a/.vscode/mcp.json" ] || fail "init mcp.json"; "$cli" uninstall "$t/a" >/dev/null
[ ! -e "$t/a/.vscode/mcp.json" ] && [ ! -e "$t/a/.github/copilot-instructions.md" ] || fail "uninstall left files"
# existing instructions are preserved and restored byte-for-byte
mkdir -p "$t/b/.github" "$t/b/.vscode"; echo "mine" > "$t/b/.github/copilot-instructions.md"; echo '{"keep":1}' > "$t/b/.vscode/mcp.json"
"$cli" init "$t/b" >/dev/null; grep -q mine "$t/b/.github/copilot-instructions.md" || fail "lost user text"; grep -q '"keep":1' "$t/b/.vscode/mcp.json" || fail "overwrote mcp.json"
"$cli" uninstall "$t/b" >/dev/null; [ "$(cat "$t/b/.github/copilot-instructions.md")" = "mine" ] || fail "not restored"; [ -f "$t/b/.vscode/mcp.json" ] || fail "removed user mcp.json"
# doctor fails on empty project and never reports an unreachable server as PASS
"$cli" doctor "$t/c" >/dev/null && fail "doctor passed empty project"
"$cli" init "$t/d" >/dev/null; out=$("$cli" doctor "$t/d" || true); echo "$out" | grep -q "advisory only" || fail "doctor missing level"
echo "$out" | grep -E "PASS.*server" >/dev/null && fail "doctor passed unreachable server" || true
echo "PASS all"
