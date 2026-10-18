# Guardrail MCP Server Deployment — Verification and Operations

> Verification checklist, maintenance, production hardening and reference material.

**Part of:** [Guardrail MCP Server Deployment Guide](./deployment-guide.md)

---

## Verification Checklist

After deployment, verify:

- [ ] All three containers running: `podman ps | grep guardrail`
- [ ] Postgres healthy: `podman logs guardrail-postgres | tail -5`
- [ ] Redis healthy: `podman logs guardrail-redis | tail -5`
- [ ] MCP server started: `podman logs guardrail-mcp-server | grep "Starting Guardrail MCP Server"`
- [ ] Database connected: `podman logs guardrail-mcp-server | grep "Database connected"`
- [ ] Redis connected: `podman logs guardrail-mcp-server | grep "Redis connected"`
- [ ] MCP endpoint responding: `curl -s -X POST http://localhost:8080/mcp`
- [ ] Web UI responding: `curl -s http://localhost:8081/health/ready`
- [ ] API key authentication working: `curl -H 'Authorization: Bearer YOUR_KEY' http://localhost:8080/version`
- [ ] Ports accessible from network: `curl http://your-server:8080/mcp`

## Maintenance

### Viewing Logs

```bash
# View all logs
podman logs guardrail-mcp-server

# Follow logs in real-time
podman logs -f guardrail-mcp-server

# View specific number of lines
podman logs --tail 50 guardrail-mcp-server

# View logs for all containers
podman logs guardrail-postgres
podman logs guardrail-redis
```

### Restarting Services

```bash
# Restart MCP server only
podman restart guardrail-mcp-server

# Restart all containers
podman restart guardrail-postgres guardrail-redis guardrail-mcp-server

# Recreate entire pod
podman pod stop guardrail-pod
podman pod rm guardrail-pod
# Then run deployment steps again
```

### Updating Configuration

```bash
# Update .env file
vim .env

# Restart containers to pick up changes
podman restart guardrail-mcp-server

# For database changes, also restart postgres
podman restart guardrail-postgres
```

### Backup and Restore

```bash
# Backup postgres data
podman run --rm \
  -v guardrail_pg_data:/data \
  -v $(pwd):/backup \
  alpine \
  tar czf /backup/postgres-backup.tar.gz /data

# Restore postgres data
podman run --rm \
  -v guardrail_pg_data:/data \
  -v $(pwd):/backup \
  alpine \
  tar xzf /backup/postgres-backup.tar.gz -C /
```

## Production Hardening

### Security Recommendations

1. **Change all default secrets** before production use
2. **Use strong random passwords**:
   ```bash
   openssl rand -hex 32  # For API keys
   openssl rand -base64 32  # For passwords
   ```
3. **Enable TLS** for database connections
4. **Set specific CORS origins** instead of "*"
5. **Enable production mode** for stricter security checks
6. **Use network policies** to restrict container communication
7. **Enable audit logging** and ship logs to central system
8. **Set resource limits** to prevent resource exhaustion

### Performance Tuning

```bash
# Database connection pooling
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=30m

# Redis connection pooling
REDIS_POOL_SIZE=10
REDIS_MIN_IDLE_CONNS=2

# Rate limiting (adjust based on load)
RATE_LIMIT_MCP=1000
RATE_LIMIT_IDE=500

# Cache TTL (adjust based on data volatility)
CACHE_TTL_RULES=5m
CACHE_TTL_DOCS=10m
```

## OpenCode Configuration

### MCP Server Configuration

Add to `.opencode/oh-my-opencode.jsonc`:

```jsonc
{
  "mcpServers": {
    "guardrails": {
      "type": "remote",
      "url": "http://your-server:8080/mcp",
      "headers": {
        "Authorization": "Bearer YOUR_MCP_API_KEY"
      }
    }
  }
}
```

### Environment Variables

Create `.env.opencode`:

```bash
# MCP Server Connection
export MCP_SERVER_URL=http://your-server:8080
export MCP_API_KEY=YOUR_MCP_API_KEY
export IDE_API_KEY=YOUR_IDE_API_KEY

# Local Configuration (for OpenCode)
export GUARDRAILS_PROJECT_SLUG=your-project
export GUARDRAILS_AGENT_TYPE=opencode
```

## References

- [MCP Server README](./README.md)
- [API Documentation](./api.md)
- [Security Review](./observability-review.md)
- [Dockerfile](./deploy/Dockerfile)
- [Podman Compose](./deploy/podman-compose.yml)
- [Kubernetes Deployment](./deploy/k8s-deployment.yaml)

## Support

For issues or questions:
1. Check troubleshooting section above
2. Review container logs: `podman logs guardrail-mcp-server`
3. Verify configuration against this guide
4. Check [GitHub Issues](https://github.com/TheArchitectit/agent-guardrails-template/issues)

## Changelog

### 2026-02-13 - Initial Deployment Guide
- Documented production deployment process
- Added schema validation error fix
- Added postgres permission fix
- Added configuration requirements
- Added troubleshooting guide
- Added verification checklist