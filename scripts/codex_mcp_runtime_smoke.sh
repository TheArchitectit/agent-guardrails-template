#!/usr/bin/env bash
# Exercise a running Guardrails MCP endpoint through the real Codex CLI.
# The endpoint must be a loopback-only SSH forward to a disposable test stack.
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: GUARDRAILS_SMOKE_MCP_KEY=<synthetic-test-key> \
       [GUARDRAILS_SMOKE_MCP_URL=http://127.0.0.1:48391/mcp] \
       scripts/codex_mcp_runtime_smoke.sh

Uses the installed Codex CLI with its existing local model authorization and
temporary MCP configuration. It calls only guardrail_advisor_list, which is a
read-only tool, and asserts the completed tool result reports nine advisors.
The synthetic MCP key must belong to the isolated test server. It is never
printed, written to the user Codex config, or passed as a command-line value.
USAGE
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi
if (($#)); then
  usage >&2
  exit 2
fi

if [[ -z "${GUARDRAILS_SMOKE_MCP_KEY:-}" ]]; then
  echo "error: GUARDRAILS_SMOKE_MCP_KEY is required (use the isolated synthetic test key)" >&2
  exit 2
fi
if ! command -v codex >/dev/null 2>&1; then
  echo "error: Codex CLI is missing; install/use the real Codex client on this workstation" >&2
  exit 2
fi

endpoint="${GUARDRAILS_SMOKE_MCP_URL:-http://127.0.0.1:48391/mcp}"
python3 - "$endpoint" <<'PY'
import sys
from urllib.parse import urlsplit

value = urlsplit(sys.argv[1])
if (value.scheme != "http" or value.hostname not in {"127.0.0.1", "localhost"}
        or value.path != "/mcp" or value.username or value.password
        or value.query or value.fragment or value.port is None):
    raise SystemExit(
        "error: endpoint must be a loopback HTTP URL with path /mcp, "
        "for example http://127.0.0.1:48391/mcp"
    )
PY

scratch="$(mktemp -d "${TMPDIR:-/tmp}/codex-guardrails-mcp.XXXXXXXX")"
chmod 700 "$scratch"
trap 'rm -rf -- "$scratch"' EXIT HUP INT TERM

codex --version
echo "Codex MCP smoke target: $endpoint"
echo "Calling read-only guardrail_advisor_list through Codex; MCP key is not displayed."

# --ignore-user-config plus command-line config overrides keep the user's MCP
# registry untouched. --ephemeral avoids persisting a Codex rollout; read-only
# sandbox and a private empty working directory prevent workspace changes.
if ! codex exec \
  --json \
  --ephemeral \
  --ignore-user-config \
  --sandbox read-only \
  --skip-git-repo-check \
  --cd "$scratch" \
  -c "mcp_servers.guardrails_smoke.url=\"$endpoint\"" \
  -c 'mcp_servers.guardrails_smoke.bearer_token_env_var="GUARDRAILS_SMOKE_MCP_KEY"' \
  -c 'mcp_servers.guardrails_smoke.enabled_tools=["guardrail_advisor_list"]' \
  -c 'mcp_servers.guardrails_smoke.tools.guardrail_advisor_list.approval_mode="approve"' \
  'Use the guardrails_smoke MCP server. Call its read-only guardrail_advisor_list tool with no arguments. Report only whether the tool call succeeded and how many advisors the returned payload contains. Do not call any other tools.' \
  >"$scratch/codex.jsonl" 2>"$scratch/codex.stderr"; then
  echo "error: Codex exec failed; its captured output was kept private and removed on exit" >&2
  exit 1
fi

python3 - "$scratch/codex.jsonl" <<'PY'
import json
import sys
from pathlib import Path

path = Path(sys.argv[1])
events = []
for number, line in enumerate(path.read_text(encoding="utf-8").splitlines(), 1):
    try:
        events.append(json.loads(line))
    except json.JSONDecodeError as exc:
        raise SystemExit(f"error: Codex JSONL line {number} is invalid: {exc}")

completed = []
for event in events:
    item = event.get("item") if isinstance(event, dict) else None
    if not isinstance(item, dict):
        continue
    if (item.get("type") != "mcp_tool_call"
            or item.get("server") != "guardrails_smoke"
            or item.get("tool") != "guardrail_advisor_list"):
        continue
    # The Codex CLI reports the actual completed MCP call as an item event.
    # This one explicitly authorized read-only tool is enabled and approved;
    # no other server tool is exposed to the model for this smoke run.
    status = str(item.get("status", "")).lower()
    if (event.get("type") != "item.completed"
            or status != "completed"
            or item.get("error") is not None):
        continue
    result = item.get("result")
    content = result.get("content", []) if isinstance(result, dict) else []
    result_text = "\n".join(
        block.get("text", "") for block in content if isinstance(block, dict)
    )
    try:
        result_payload = json.loads(result_text)
    except json.JSONDecodeError:
        result_payload = {}
    if isinstance(result_payload, dict) and result_payload.get("count") == 9:
        completed.append(item)

if not completed:
    saw_call = any(
        isinstance(event, dict)
        and isinstance(event.get("item"), dict)
        and event["item"].get("type") == "mcp_tool_call"
        and event["item"].get("server") == "guardrails_smoke"
        and event["item"].get("tool") == "guardrail_advisor_list"
        for event in events
    )
    reason = "call was absent" if not saw_call else "no successful result with count 9"
    raise SystemExit(f"error: Codex MCP verification failed: {reason}")

print("PASS: Codex completed guardrail_advisor_list and received the expected count 9.")
PY
