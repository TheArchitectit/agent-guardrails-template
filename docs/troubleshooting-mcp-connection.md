# Troubleshooting: MCP Server Connection

> Diagnosing and fixing MCP server connectivity issues

See the [Troubleshooting Index](troubleshooting.md) for other topics.

---

## MCP Server Connection Failed

**Symptoms:**
```
Error: Connection refused (localhost:8094)
Error: Could not connect to MCP server
```

**Causes:**
- MCP server not running
- Wrong port configuration
- Firewall blocking connection

**Solutions:**

1. **Start the MCP server:**
   ```bash
   python mcp_server.py
   # or
   ./start-mcp-server.sh
   ```

2. **Verify port configuration:**
   ```bash
   # Check if server is listening
   netstat -tlnp | grep 8094
   # or
   lsof -i :8094
   ```

3. **Check firewall rules:**
   ```bash
   # For Linux
   sudo ufw allow 8094/tcp
   # For macOS
   sudo pfctl -e
   ```
