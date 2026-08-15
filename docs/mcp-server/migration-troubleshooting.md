# Migration: Troubleshooting

> Common migration issues, their causes, and solutions — plus a verification script

See the [Migration Overview](version-migration.md) for the compatibility matrix.

---

## Common Migration Issues

### Issue: "Team configuration version mismatch"

**Cause:** Migration script didn't update all config files

**Solution:**
```bash
# Force version update (Go binary)
cd mcp-server
go run ./cmd/tools/update_config.go --version 2.0

# Re-run migration
go run ./cmd/tools/migrate_config.go --from-version 1 --to-version 2 --force
```

### Issue: "Database migration failed"

**Cause:** Partial migration or checksum mismatch

**Solution:**
```bash
# Check migration status
cd mcp-server
make migrate-status

# Fix by marking as applied
migrate -path internal/database/migrations -database "$DATABASE_URL" force 20260215000001

# Or rollback and retry
make migrate-down
make migrate-up
```

### Issue: "Port already in use"

**Cause:** Old server still running

**Solution:**
```bash
# Find and kill old process
lsof -ti:8094 | xargs kill -9

# Or use different port temporarily
MCP_PORT=8096 ./mcp-server/cmd/server/server
```

### Issue: "Client connection refused"

**Cause:** Client configured for old endpoint

**Solution:**
```bash
# Update client configuration
./scripts/update_client_configs.sh --new-port 8094 --new-path /mcp

# Verify connectivity
curl -H "Authorization: Bearer $API_KEY" \
  http://localhost:8094/mcp/v1/health
```

---

## Migration Verification

```bash
#!/bin/bash
# verify_migration.sh

echo "Verifying migration..."

# Check server health
echo -n "Health check: "
curl -sf http://localhost:8094/mcp/v1/health && echo "PASS" || echo "FAIL"

# Check version
echo -n "Version check: "
git describe --tags | grep -q "v2.0" && echo "PASS" || echo "FAIL"

# Check database
echo -n "Database check: "
psql -U guardrails -c "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1;" | grep -q "20260215" && echo "PASS" || echo "FAIL"

# Check team configs
echo -n "Team config check: "
python scripts/validate_all_configs.py && echo "PASS" || echo "FAIL"

# Test basic operation
echo -n "Operation check: "
curl -s -X POST http://localhost:8094/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/list"}' | grep -q "guardrail_team" && echo "PASS" || echo "FAIL"

echo "Verification complete"
```
