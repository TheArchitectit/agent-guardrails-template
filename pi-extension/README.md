# @architectit/pi-guardrails

Four Laws guardrails enforcement for the pi coding agent. Works standalone (no MCP server required) and can bridge to the existing Go MCP server when available.

## Installation

```bash
pi install npm:@architectit/pi-guardrails
```

Or manually:

```bash
npx @architectit/pi-guardrails
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

## Tools (28 registered)

### Core Enforcement

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
| `guardrail_check_halt` | Evaluate halt conditions (includes uncertainty score) |
| `guardrail_log_violation` | Log a guardrail violation |
| `guardrail_status` | Get current session status |
| `guardrail_acknowledge_halt` | Acknowledge a halt condition to resume |

### Language & Pattern Rules

| Tool | Purpose |
|------|---------|
| `guardrail_detect_language` | Auto-detect project languages |
| `guardrail_get_language_profile` | Get language profile with available rules |
| `guardrail_check_pattern` | Check code against prevention pattern rules |
| `guardrail_list_languages` | List available language rule sets |
| `guardrail_list_skills` | List all guardrails skills |
| `guardrail_read_skill` | Read a skill's documentation |

### Regression & Validation

| Tool | Purpose |
|------|---------|
| `guardrail_check_regression` | Check if file edits risk regressing past failures |
| `guardrail_verify_fixes` | Verify that past fixes are still intact |
| `guardrail_register_failure` | Register a failure in the cross-session registry |
| `guardrail_validate_replacement` | Validate edit old_content matches actual file |
| `guardrail_validate_git` | Validate git operations (branch protection, force-push) |

### Planning & Scope

| Tool | Purpose |
|------|---------|
| `guardrail_pre_work_check` | Generate pre-work risk checklist |
| `guardrail_detect_creep` | Detect feature creep against authorized scope |
| `guardrail_mcp` | Proxy to MCP server (when connected) |

## Automatic Enforcement

The extension registers event handlers that enforce the Four Laws automatically:

- **Read tracking**: File reads are tracked via `tool_result` events
- **Pre-edit enforcement**: Edits to unread files are blocked (Law 1)
- **Scope enforcement**: Edits outside the authorized scope are blocked (Law 2)
- **Bash safety**: Dangerous commands (`rm -rf /`, `git push --force`, `sudo`, etc.) are blocked
- **Injection defense**: Scans tool inputs for prompt injection patterns
- **Output validation**: Detects secrets and PII in tool output
- **Content filtering**: Detects denied topics in output (warn-only)
- **Canary tokens**: Detects data exfiltration via embedded tokens (warn-only)
- **Permission system**: Per-tool permission levels (auto/ask/blocked)
- **Halt lifecycle**: Blocked operations record halt state; requires acknowledgment to resume

## Language-Specific Rules

Auto-detects project languages and loads prevention rules from `.guardrails/prevention-rules/languages/`:

| Language | Rules | Examples |
|----------|-------|---------|
| Python | 8 | eval/exec, subprocess shell=True, bare except, pickle, SQL injection |
| TypeScript | 7 | any type, non-null assertion, eval, innerHTML, hardcoded secrets |
| Go | 6 | ignored errors, panic, SQL concat, goroutine without context |
| Rust | 6 | unsafe blocks, unwrap, panic!, todo!, raw pointer deref |

Add new languages by creating a JSON file in `.guardrails/prevention-rules/languages/`.

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
  "panelAutoOpen": false,
  "toolPermissions": {
    "defaultLevel": "auto",
    "tools": {
      "bash": "ask",
      "write": "auto",
      "edit": "auto",
      "read": "auto"
    }
  },
  "injectionDefense": {
    "blockThreshold": 0.8,
    "warnThreshold": 0.5,
    "heuristicEnabled": true
  },
  "outputValidation": {
    "enablePII": false,
    "autoRedact": false,
    "redactionText": "[REDACTED]",
    "contentFilter": {
      "deniedTopics": ["malicious code"],
      "allowedTopics": [],
      "strictMode": false
    }
  },
  "canary": {
    "prefix": "CATALOG:",
    "tokenLength": 32
  },
  "gitPolicy": {
    "protectedBranches": ["main", "master"],
    "commitFormat": "conventional",
    "requireAIAttribution": true
  }
}
```

Environment variables:
- `PI_GUARDRAILS_MCP_API_KEY` — API key for the MCP server

## Halt Lifecycle

When a handler blocks an operation, a halt is recorded:

1. **active** → operation attempted
2. **halted** → handler blocked, reason recorded
3. **acknowledged** → `guardrail_acknowledge_halt` called after review

## Uncertainty Scoring

`guardrail_check_halt` returns an `uncertaintyScore` (0-1):

| Score | Level | Meaning |
|-------|-------|---------|
| 0-0.2 | Certain | No concerns |
| 0.2-0.5 | Probably | Mild uncertainty (e.g. edit without details) |
| 0.5-0.8 | Uncertain | Significant concern (e.g. delete without details) |
| 0.8-1.0 | Guessing | High risk (e.g. production-affected operations) |

## Status Bar

When enabled, shows a compact status in the pi status bar:

- `g: ok` — no issues
- `g: !!2/3 src/ mcp:*` — 2/3 strikes, scope set to `src/`, MCP connected
- `g: src/ !3v mcp:.` — scope set, 3 violations, MCP disconnected

## TUI Dashboard

Open the guardrails overlay with:

```
/guardrails
```

The panel shows safety score, Four Laws status, strike tracker, scope, and MCP connection status.

Close with `Esc` or `q`. Scroll with `j`/`k`.

## MCP Bridge

When the Go MCP server is available, the extension can proxy calls to it for enhanced enforcement:

1. Configure the server endpoint in `config.json` under `mcpBinaryPath` (URL for SSE, command for stdio)
2. Initialize a session with `guardrail_init` — the extension auto-connects
3. Use `guardrail_mcp` with an `action` parameter to call any MCP server tool
4. Reconnection uses exponential backoff (1s base, 30s max, 5 attempts)

## Storage

All state is stored under `~/.pi/agent/extensions/pi-guardrails/`:

- `sessions/` — session state JSON files
- `violations.jsonl` — append-only violation log
- `.guardrails/regression/failure-registry.jsonl` — cross-session failure registry
- `config.json` — user configuration

## 22 Code Modules

FileReadStore, ScopeValidator, StrikeCounter, HaltChecker, ViolationLog, SessionStore, InjectionDetector, OutputValidator, ContentFilter, CanaryTokenManager, PermissionManager, PolicyLoader, MCPClient, PreWorkChecker, FeatureCreepDetector, PatternRuleEngine, GitValidator, LanguageDetector, RegressionGuard, ExactReplacementValidator, SandboxRunner, GuardrailsPanel

## CI/CD Integration

### Pre-commit Hook

```bash
cp guardrails/pre-commit.sh .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
```

### GitHub Actions

The `.github/workflows/pi-guardrails-ci.yml` workflow runs on PRs:
- Unit tests for pi-extension and guardrails modules
- Secret scanning on changed files
- Scope compliance validation
