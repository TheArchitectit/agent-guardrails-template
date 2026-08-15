# Troubleshooting Guide

> Common issues, solutions, and recovery procedures for Agent Guardrails Template

---

## Quick Diagnostics

Run this checklist to quickly identify common issues:

```bash
# 1. Verify the server is up
curl -s http://localhost:8081/health/ready | jq .

# 2. Check configuration
ls -la .teams/
ls -la .guardrails/

# 3. Test the MCP endpoint
curl -s -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | jq .
```

**Expected results:**
- Health endpoint returns `{"status":"ready",...}`
- `.teams/` and `.guardrails/` directories exist
- Tools list returns the 35 guardrail tools

---

## Topic Guides

| Topic | File | Covers |
|-------|------|--------|
| MCP connection | [troubleshooting-mcp-connection.md](troubleshooting-mcp-connection.md) | Connection refused, port issues, firewall rules |
| API authentication | [troubleshooting-api-auth.md](troubleshooting-api-auth.md) | AUTH-001, AUTH-002, API key format and rotation |
| Team management | [troubleshooting-team-management.md](troubleshooting-team-management.md) | Team init, size violations, phase gate checks, role conflicts |
| Debug logging | [troubleshooting-debug-logging.md](troubleshooting-debug-logging.md) | Enabling debug mode, log file locations, log rotation |
| Performance | [troubleshooting-performance.md](troubleshooting-performance.md) | Slow operations, high memory, rate limiting |
| Recovery | [troubleshooting-recovery.md](troubleshooting-recovery.md) | Restore from backup, reset state, repair config, emergency rollback |
| Getting help | [troubleshooting-getting-help.md](troubleshooting-getting-help.md) | Support channels, error reporting, diagnostic script |

---

## Error Code Quick Reference

| Code | Topic | Guide |
|------|-------|-------|
| TEAM-001 | Team not found | [Team management](troubleshooting-team-management.md) |
| TEAM-002 | Invalid team ID | [Team management](troubleshooting-team-management.md) |
| TEAM-004 | Person already assigned | [Team management](troubleshooting-team-management.md) |
| TEAM-005 | Team size violation | [Team management](troubleshooting-team-management.md) |
| AUTH-001 | Authentication required | [API authentication](troubleshooting-api-auth.md) |
| AUTH-002 | Invalid API key | [API authentication](troubleshooting-api-auth.md) |
| PROJ-002 | Project configuration missing | [Recovery](troubleshooting-recovery.md) |
| RATE-001 | Rate limit exceeded | [Performance](troubleshooting-performance.md) |

---

## Getting Help

For self-service resources, support channels, and a diagnostic script you can run to gather information for bug reports, see [Getting Help](troubleshooting-getting-help.md).
