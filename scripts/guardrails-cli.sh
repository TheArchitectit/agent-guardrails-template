#!/usr/bin/env bash
# guardrails init | doctor | uninstall  (F061, F062, F067)
# Installs ADVISORY Copilot wiring into a target project. Writes only files it lists in .guardrails-install.manifest.
set -euo pipefail
here="$(cd "$(dirname "$0")/.." && pwd)"
cmd="${1:-}"; target="${2:-.}"
manifest="$target/.guardrails-install.manifest"
case "$cmd" in
  init)
    mkdir -p "$target/.vscode" "$target/.github"; : > "$manifest.tmp"
    put(){ # src dest ; never overwrite an existing file
      if [ -e "$2" ]; then echo "SKIP  $2 (exists)"; else cp "$1" "$2"; echo "$2" >> "$manifest.tmp"; echo "ADD   $2"; fi; }
    put "$here/integrations/copilot/.vscode/mcp.json" "$target/.vscode/mcp.json"
    if [ -e "$target/.github/copilot-instructions.md" ]; then
      if ! grep -q '^## Guardrails MCP (advisory)' "$target/.github/copilot-instructions.md"; then
        cp "$target/.github/copilot-instructions.md" "$target/.github/copilot-instructions.md.guardrails-bak"
        cat "$here/integrations/copilot/copilot-instructions.addendum.md" >> "$target/.github/copilot-instructions.md"
        echo "$target/.github/copilot-instructions.md.guardrails-bak" >> "$manifest.tmp"; echo "APPEND $target/.github/copilot-instructions.md (backup kept)"
      fi
    else put "$here/integrations/copilot/copilot-instructions.addendum.md" "$target/.github/copilot-instructions.md"; fi
    cat "$manifest.tmp" >> "$manifest" 2>/dev/null || true; rm -f "$manifest.tmp"
    echo "Installed advisory wiring. Nothing here blocks an agent." ;;
  doctor)
    rc=0; ok(){ echo "PASS  $1"; }; no(){ echo "FAIL  $1"; rc=1; }
    [ -f "$target/.vscode/mcp.json" ] && ok "mcp.json present" || no "mcp.json missing"
    grep -q '"/mcp"\|localhost:8080/mcp' "$target/.vscode/mcp.json" 2>/dev/null && ok "mcp.json points at /mcp" || no "mcp.json does not point at /mcp"
    grep -q 'Guardrails MCP (advisory)' "$target/.github/copilot-instructions.md" 2>/dev/null && ok "instructions addendum present" || no "instructions addendum missing"
    if curl -fsS -m 3 -o /dev/null -X POST -H 'Content-Type: application/json' http://localhost:8080/mcp -d '{}' 2>/dev/null; then no "server accepted an unauthenticated request"
    else code=$(curl -s -m 3 -o /dev/null -w '%{http_code}' -X POST http://localhost:8080/mcp -d '{}' 2>/dev/null || true)
      case "$code" in 401) ok "server reachable and rejects unauthenticated requests";; 000|"") echo "UNKNOWN server not reachable on localhost:8080 (not a pass)";; *) echo "UNKNOWN server returned $code";; esac; fi
    echo "Enforcement level of this install: advisory only."; exit $rc ;;
  uninstall)
    [ -f "$manifest" ] || { echo "No manifest; nothing to remove."; exit 0; }
    while IFS= read -r f; do
      case "$f" in *.guardrails-bak) orig="${f%.guardrails-bak}"; mv "$f" "$orig"; echo "RESTORE $orig";;
        *) rm -f "$f"; echo "REMOVE $f";; esac
    done < "$manifest"
    rm -f "$manifest"; echo "Uninstalled." ;;
  *) echo "usage: $0 init|doctor|uninstall [target-dir]"; exit 2 ;;
esac
