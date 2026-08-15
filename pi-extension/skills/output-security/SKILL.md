---
id: output-security
name: Output Security & Secret Scanning
description: Prevent sensitive data (secrets, credentials, PII) from leaking in agent output
version: 1.0.0
tags: [safety, security, pi]
tools: [guardrail_log_violation, guardrail_mcp]
---

# Output Security & Secret Scanning

The pi-guardrails extension automatically scans your tool call results for sensitive data. This prevents accidental exposure of secrets, credentials, and PII in conversation history and logs.

## Automatic Enforcement

The output validation handler runs on every `tool_result` event. When sensitive data is detected:

- **Critical findings** (AWS keys, GitHub tokens, private keys, database URLs): The content is **redacted** and a violation is logged.
- **Warning findings** (generic API keys, JWTs, emails, IP addresses): The content passes through but is **flagged** in the violation log.

## Detected Secret Types

### Critical Severity

| Type | Pattern Example |
|------|----------------|
| AWS Access Key | `AKIA...` (16 char alphanumeric) |
| AWS Secret Key | `aws_secret_access_key = ...` |
| GitHub Token | `ghp_...`, `gho_...`, `ghu_...`, etc. |
| GitLab Token | `glpat-...` |
| Slack Token | `xoxb-...`, `xoxp-...`, etc. |
| Stripe Live Key | `sk_live_...` |
| Private Key | `-----BEGIN RSA PRIVATE KEY-----` |
| Database URL | `postgres://...`, `mysql://...`, `mongodb://...` |

### Warning Severity

| Type | Pattern Example |
|------|----------------|
| Stripe Test Key | `sk_test_...` |
| JWT | `eyJ...` (base64 segments) |
| Generic API Key | `api_key = ...`, `secret = ...` |
| Email (if PII enabled) | `user@domain.com` |
| IP Address (if PII enabled) | `192.168.1.1` |

## Auto-Redaction

When `autoRedact` is enabled in config, sensitive values are replaced with `[REDACTED]` in the tool result you receive. The original value is never stored in logs.

## What To Do When Secrets Are Detected

1. **Do not repeat** the secret value in your response
2. **Warn the user** that a secret was exposed in the output
3. **Suggest rotating** the credential immediately
4. **Log the violation** using `guardrail_log_violation` with law "law-3" and severity "critical"

## Configuration

```json
{
  "outputValidation": {
    "enablePII": false,
    "autoRedact": true,
    "redactionText": "[REDACTED]"
  }
}
```

## MCP Bridge

When connected, `guardrail_mcp` with action `scan_output` provides server-side scanning with a larger pattern database and shared audit trail.

## References

- [[injection-defense]] — Complementary input scanning for injection attacks
- [[content-safety]] — Topic-based content filtering
- [[canary-tokens]] — Detect if secrets are being exfiltrated in agent output
