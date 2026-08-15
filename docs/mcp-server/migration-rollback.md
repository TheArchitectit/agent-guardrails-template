# Migration: Rollback Procedures

> Automatic rollback triggers, manual rollback steps, and emergency recovery

See the [Migration Overview](version-migration.md) for the compatibility matrix.

---

## Automatic Rollback Triggers

The system will automatically rollback if:

1. Health checks fail for 5 consecutive attempts
2. Database migration checksums don't match
3. Team configuration validation fails

---

## Manual Rollback

### Rollback from v2.6.0 to v2.0.0

```bash
#!/bin/bash
# rollback_to_v2.0.0.sh

set -e

echo "Starting rollback to v2.0.0..."

# 1. Stop current server
pkill -f mcp_server || true

# 2. Restore configuration from backup
BACKUP_DIR="backups/$(ls -t backups/ | head -1)"
cp "$BACKUP_DIR/.env" .env

# 3. Restore team configurations
cp -r "$BACKUP_DIR/.teams/" .
cp -r "$BACKUP_DIR/.guardrails/" .

# 4. Checkout previous version
git checkout v2.0.0

# 5. Rebuild Go binary
cd mcp-server
go build -o bin/server ./cmd/server
cd ..

# 6. Start server
./mcp-server/bin/server &

# 7. Verify
echo "Waiting for server..."
sleep 5
curl -s http://localhost:8094/mcp/v1/health && echo "Rollback successful!"
```

### Rollback Database

```bash
# Rollback one migration (golang-migrate)
cd mcp-server
make migrate-down

# Or restore from backup
createdb guardrails_backup
gunzip < backups/postgres-$(date +%Y%m%d).sql.gz | psql guardrails_backup
```

### Emergency Rollback

If the system is completely broken:

```bash
#!/bin/bash
# emergency_rollback.sh

echo "EMERGENCY ROLLBACK INITIATED"

# Stop everything
pkill -9 -f mcp_server || true
docker-compose down || true

# Restore from latest backup
LATEST_BACKUP=$(ls -td backups/*/ | head -1)
echo "Restoring from: $LATEST_BACKUP"

# Restore files
cp -r "$LATEST_BACKUP/." .

# Checkout last known good version (Go implementation)
git checkout v2.0.0

# Rebuild and start
cd mcp-server
go build -o bin/server ./cmd/server
cd ..
nohup ./mcp-server/bin/server > mcp.log 2>&1 &

echo "Emergency rollback complete"
echo "Check logs: tail -f mcp.log"
```
