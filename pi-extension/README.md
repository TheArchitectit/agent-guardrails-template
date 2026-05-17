# @thearchitectit/pi-guardrails

Four Laws guardrails enforcement for the pi coding agent. Works standalone (no MCP server required) and can bridge to the existing Go MCP server when available.

## Installation

```bash
pi install npm:@thearchitectit/pi-guardrails
```

Or manually:

```bash
npx @thearchitectit/pi-guardrails
```

## Architecture

**Hybrid design:** The extension operates in two modes:

- **Standalone** (default): All enforcement runs locally within the pi extension. No external server needed.
- **MCP Bridge**: When the Go MCP server is available, tools proxy to it for enhanced enforcement.

The standalone mode is the primary value proposition — teams don't need to run the MCP server to get guardrails protection.

## The Four Laws of Agent Safety

1. **Read Before Editing** — The agent must read a file before editing it
2. **Stay in Scope** — The agent only operates on authorized file paths
3. **Verify Before Committing** — Changes must be verified before committing
4. **Halt When Uncertain** — The Three Strikes rule: 3 consecutive failures triggers a halt

## Tools

| Tool | Purpose |
|------|---------|
| `guardrail_init` | Initialize a guardrails session |
| `guardrail_record_read` | Mark a file as read (Law 1) |
| `guardrail_verify_read` | Check if a file was read before editing |
| `guardrail_set_scope` | Define authorized file paths (Law 2) |
| `guardrail_check_scope` | Check if a path is in scope |
| `guardrail_record_attempt` | Record a task attempt result (Law 4) |
| `guardrail_check_strikes` | Check strike count for a task |
| `guardrail_reset_strikes` | Reset strikes after resolution |
| `guardrail_check_halt` | Evaluate halt conditions |
| `guardrail_log_violation` | Log a guardrail violation |
| `guardrail_status` | Get current session status |
| `guardrail_mcp` | Proxy to MCP server (Sprint 1) |

## Automatic Enforcement

The extension registers event handlers that enforce the Four Laws automatically:

- **Read tracking**: File reads are tracked via `tool_result` events
- **Pre-edit enforcement**: Edits to unread files are blocked (Law 1)
- **Scope enforcement**: Edits outside the authorized scope are blocked (Law 2)
- **Bash safety**: Dangerous commands (`rm -rf /`, `git push --force`, `sudo`, etc.) are blocked

## Configuration

Config file: `~/.pi/agent/extensions/pi-guardrails/config.json`

```json
{
  "mcpBinaryPath": "",
  "enabledRules": ["four-laws", "three-strikes", "scope-validator"],
  "autoRegister": true,
  "defaultScope": [],
  "maxStrikes": 3,
  "statusBarEnabled": true,
  "panelAutoOpen": false
}
```

Environment variables:
- `PI_GUARDRAILS_MCP_API_KEY` — API key for the MCP server

## Status Bar

When enabled, shows a compact status in the pi status bar:

- `g: ok` — no issues
- `g: !!2/3 src/ mcp:*` — 2/3 strikes, scope set to `src/`, MCP connected
- `g: src/ !3v mcp:.` — scope set, 3 violations, MCP disconnected

## Storage

All state is stored under `~/.pi/agent/extensions/pi-guardrails/`:

- `sessions/` — session state JSON files
- `violations.jsonl` — append-only violation log
- `config.json` — user configuration
