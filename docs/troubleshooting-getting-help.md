# Troubleshooting: Getting Help

> Self-service resources, support channels, and diagnostic scripts

See the [Troubleshooting Index](troubleshooting.md) for other topics.

---

## Self-Service Resources

1. **Documentation:**
   - [TEAM_TOOLS.md](teams/team-tools.md) - Tool reference
   - [AGENT_GUARDRAILS.md](getting-started/agent-guardrails.md) - Safety protocols
   - [TEAM_STRUCTURE.md](teams/team-structure.md) - Team definitions

2. **Error Code Lookup:**
   - See TEAM_TOOLS.md Error Handling section
   - Search logs for error codes

3. **Community:**
   - GitHub Issues: Report bugs and feature requests
   - Discussions: Ask questions, share solutions

## Support Channels

| Issue Type | Channel | Response Time |
|------------|---------|---------------|
| Critical outage | Email: oncall@example.com | 15 minutes |
| Security issue | security@example.com | 4 hours |
| Feature request | GitHub Issues | 2-3 days |
| General question | GitHub Discussions | 1-2 days |

## Required Information

When reporting issues, include:

1. **Error message:** Full text or screenshot
2. **Log snippets:** Relevant sections from `.mcp/mcp.log`
3. **Steps to reproduce:** Minimal example
4. **Environment:**
   ```bash
   python --version
   uname -a
   git log --oneline -1
   ```
5. **Configuration:** (sanitized)
   ```bash
   cat .mcp/config.json | grep -v password
   ```

## Diagnostic Script

Run this script to gather diagnostic information:

```bash
#!/bin/bash
# diagnose.sh - Collect diagnostic information

echo "=== Agent Guardrails Diagnostics ==="
echo "Date: $(date)"
echo ""

echo "=== System Information ==="
uname -a
echo ""

echo "=== Python Version ==="
python --version
echo ""

echo "=== MCP Server Status ==="
pgrep -f mcp_server || echo "MCP server not running"
echo ""

echo "=== Recent Log Entries ==="
tail -50 .mcp/mcp.log 2>/dev/null || echo "No log file found"
echo ""

echo "=== Configuration ==="
ls -la .teams/ 2>/dev/null || echo "No .teams directory"
ls -la .guardrails/ 2>/dev/null || echo "No .guardrails directory"
echo ""

echo "=== Health Check ==="
curl -s http://localhost:8094/mcp/v1/health 2>/dev/null || echo "Health check failed"
echo ""

echo "=== Diagnostics Complete ==="
```
