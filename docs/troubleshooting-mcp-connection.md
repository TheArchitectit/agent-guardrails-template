# Troubleshooting: MCP Server Connection

> Diagnosing and fixing MCP server connectivity issues

See the [Troubleshooting Index](troubleshooting.md) for other topics.

---

## MCP Server Connection Failed

**Symptoms:**
```
Error: Connection refused (localhost:8080)
Error: Could not connect to MCP server
```

**Causes:**
- MCP server not running
- Wrong port configuration
- Firewall blocking connection

**Solutions:**

1. **Start the MCP server:**
   ```bash
   cd mcp-server && make dev
   # or, if running the container:
   docker compose -f deploy/podman-compose.yml up -d
   ```

2. **Verify port configuration:**
   ```bash
   # Check if server is listening
   netstat -tlnp | grep 8080
   # or
   lsof -i :8080
   ```

3. **Check firewall rules:**
   ```bash
   # For Linux
   sudo ufw allow 8080/tcp
   # For macOS
   sudo pfctl -e
   ```
