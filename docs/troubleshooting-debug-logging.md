# Troubleshooting: Debug Mode and Log Analysis

> Enabling debug logging and analyzing log output to diagnose issues

See the [Troubleshooting Index](troubleshooting.md) for other topics.

---

## Debug Mode

Enable debug mode to get detailed logging for troubleshooting.

### Enable Debug Logging

**Option 1: Environment Variable**
```bash
export MCP_DEBUG=1
export MCP_LOG_LEVEL=debug
python mcp_server.py
```

**Option 2: Configuration File**
```json
// .mcp/config.json
{
  "logging": {
    "level": "debug",
    "file": ".mcp/mcp.log",
    "console": true
  }
}
```

**Option 3: Command Line Flag**
```bash
python mcp_server.py --debug --log-file .mcp/debug.log
```

### Debug Output Examples

**Normal Operation:**
```
[DEBUG] Received request: tools/call
[DEBUG] Tool: guardrail_team_list
[DEBUG] Parameters: {"project_name": "my-project"}
[DEBUG] Execution time: 45ms
[INFO] Response sent successfully
```

**Error Condition:**
```
[DEBUG] Received request: tools/call
[DEBUG] Tool: guardrail_team_assign
[DEBUG] Parameters: {"team_id": 99, ...}
[ERROR] Validation failed: Invalid team_id
[ERROR] Error code: TEAM-002
[DEBUG] Stack trace:
  File "scripts/team_manager.py", line 45, in validate_team
    raise InvalidTeamError(f"Team {team_id} not found")
[INFO] Error response sent
```

---

## Log Analysis

### Log File Locations

| Component | Log File | Description |
|-----------|----------|-------------|
| MCP Server | `.mcp/mcp.log` | Main server logs |
| Team Manager | `.mcp/team_manager.log` | Team operations |
| Validation | `.mcp/validation.log` | Guardrail checks |
| Audit | `.mcp/audit.log` | Security events |

### Log Format

```
[TIMESTAMP] [LEVEL] [COMPONENT] Message
```

**Example:**
```
2026-02-15 14:32:15 [INFO] [MCP] Server started on port 8080
2026-02-15 14:32:18 [DEBUG] [TEAM] Validating team assignment
2026-02-15 14:32:18 [ERROR] [VALID] TEAM-002: Invalid team ID
```

### Common Log Patterns

**Startup Issues:**
```bash
# Check for port binding errors
grep "Address already in use" .mcp/mcp.log

# Check for permission denied
grep "Permission denied" .mcp/mcp.log

# Check for missing files
grep "No such file" .mcp/mcp.log
```

**Authentication Issues:**
```bash
# Find failed authentication attempts
grep "AUTH-" .mcp/audit.log

# List unauthorized access attempts
grep "403\|Unauthorized" .mcp/audit.log
```

**Performance Issues:**
```bash
# Find slow requests
grep "slow\|timeout" .mcp/mcp.log

# High response times
awk '/Execution time/ && $NF > 1000 {print}' .mcp/mcp.log
```

### Log Rotation

Configure automatic log rotation to prevent disk space issues:

```bash
# Add to crontab
crontab -e

# Rotate logs daily at midnight
0 0 * * * /usr/sbin/logrotate /etc/logrotate.d/mcp
```

**logrotate configuration:**
```
# /etc/logrotate.d/mcp
/mnt/ollama/git/agent-guardrails-template/.mcp/*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    create 644 user user
}
```
