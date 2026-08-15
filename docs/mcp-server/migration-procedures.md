# Migration: Procedures

> Step-by-step upgrade paths for each version, including pre-migration checklist

See the [Migration Overview](version-migration.md) for the compatibility matrix.
See [Breaking Changes](migration-breaking-changes.md) for what changed in each version.

---

## Pre-Migration Checklist

Before starting any migration:

```bash
# 1. Backup current state
./scripts/backup.sh --full

# 2. Verify backup integrity
./scripts/verify_backup.sh /backups/guardrails-$(date +%Y%m%d).tar.gz

# 3. Check current version
git describe --tags

# 4. Review breaking changes
cat docs/MIGRATION.md | grep -A 20 "v$(TARGET_VERSION)"

# 5. Test in staging environment
./scripts/test_migration.sh --version $TARGET_VERSION
```

---

## Migrating to v2.6.0 (Go Implementation)

**Go Migration:** The MCP server and team management have been migrated from Python to Go.
- **Benefits:** Smaller container size, distroless compatibility, improved security
- **API Compatibility:** Unchanged from MCP perspective
- **Location:** Go code is in `mcp-server/internal/`

**Estimated Time:** 30-45 minutes
**Downtime Required:** Yes (5-10 minutes)

### Step 1: Pre-Migration (5 min)

```bash
# Stop the MCP server
pkill -f mcp_server || true

# Create full backup
mkdir -p backups/$(date +%Y%m%d)
cp -r .teams/ .guardrails/ backups/$(date +%Y%m%d)/
cp .env backups/$(date +%Y%m%d)/

# Export team configurations (Go binary)
cd mcp-server && go run ./cmd/tools/export_teams.go --format json > ../backups/$(date +%Y%m%d)/teams_export.json && cd ..
```

### Step 2: Update Configuration (10 min)

```bash
# Update environment variables
# OLD:
# MCP_PORT=8094
# WEB_PORT=8093

# NEW:
cat >> .env << 'EOF'
# v2.6.0 Configuration (Go Implementation)
MCP_SERVER_PORT=8094
WEB_UI_PORT=8093
TEAM_CONFIG_VERSION=2
EOF

# Update team configuration schema (Go binary)
cd mcp-server && go run ./cmd/tools/migrate_config.go --from-version 1 --to-version 2 && cd ..
```

### Step 3: Database Migration (10 min)

```bash
# Run database migrations (using golang-migrate)
cd mcp-server
export DATABASE_URL="postgresql://guardrails:password@localhost:5432/guardrails?sslmode=disable"
make migrate-up

# Verify migration
psql -U guardrails -d guardrails -c "\dt"
psql -U guardrails -d guardrails -c "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1;"
```

### Step 4: Deploy New Version (5 min)

```bash
# Pull new version
git fetch origin
git checkout v2.6.0

# Build Go binary
cd mcp-server
go build -o bin/server ./cmd/server
cd ..
```

### Step 5: Post-Migration (5 min)

```bash
# Start server
./start-mcp-server.sh

# Verify health
curl -s http://localhost:8094/mcp/v1/health | jq .

# Test team operations
curl -X POST http://localhost:8094/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"guardrail_team_list","arguments":{"project_name":"test-project"}}}'

# Verify team size validation
curl -X POST http://localhost:8094/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"guardrail_team_size_validate","arguments":{"project_name":"test-project"}}}'
```

### Step 6: Update Client Configurations

**Claude Code:**
```json
// .claude/settings.json
{
  "mcpServers": {
    "guardrails": {
      "url": "http://localhost:8094/mcp",
      "headers": {
        "Authorization": "Bearer YOUR_API_KEY"
      }
    }
  }
}
```

**OpenCode:**
```jsonc
// .opencode/oh-my-opencode.jsonc
{
  "mcp": {
    "servers": [
      {
        "name": "guardrails",
        "url": "http://localhost:8094/mcp",
        "apiKey": "YOUR_API_KEY"
      }
    ]
  }
}
```

---

## Migrating to v1.10.0

**Estimated Time:** 15-20 minutes
**Downtime Required:** Minimal (rolling update possible)

### Step 1: Backup

```bash
cp -r .teams/ .teams-backup-$(date +%Y%m%d)/
cp .env .env.backup-$(date +%Y%m%d)
```

### Step 2: Update Code

```bash
git fetch origin
git checkout v1.10.0
pip install -r requirements.txt
cd mcp-server && go build ./cmd/server && cd ..
```

### Step 3: Restart Server

```bash
# Rolling restart (no downtime)
pkill -HUP -f mcp_server

# Or full restart
pkill -f mcp_server
./mcp-server/cmd/server/server
```

### Step 4: Verify New Features

```bash
# Access Web UI
open http://localhost:8080/web

# Test new tools
curl -X POST http://localhost:8092/mcp/v1/message \
  -d '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"guardrail_validate_scope","arguments":{"path":"src/main.py","allowed_paths":["src/","tests/"]}}}'
```

---

## Migrating to v1.9.0

**Estimated Time:** 45-60 minutes
**Downtime Required:** Yes (protocol change)

### Step 1: Full Backup

```bash
./scripts/full_backup.sh --output backups/pre-mcp-migration/
```

### Step 2: Prepare New Infrastructure

```bash
# Setup PostgreSQL and Redis
# See deployment guide in mcp-server/DEPLOYMENT_GUIDE.md

# Create new environment file
cat > .env.v1.9.0 << 'EOF'
MCP_API_KEY=<generate with: openssl rand -hex 32>
IDE_API_KEY=<generate with: openssl rand -hex 32>
JWT_SECRET=<generate with: openssl rand -hex 48>

DB_HOST=postgres
DB_PORT=5432
DB_NAME=guardrails
DB_USER=guardrails
DB_PASSWORD=<secure password>
DB_SSLMODE=disable

REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=<secure password>
REDIS_USE_TLS=false

MCP_PORT=8080
WEB_PORT=8081
WEB_ENABLED=true
LOG_LEVEL=info
REQUEST_TIMEOUT=30s
EOF
```

### Step 3: Deploy New Server

```bash
# Build container
cd mcp-server
docker build -t guardrail-mcp:v1.9.0 -f deploy/Dockerfile .

# Deploy
docker-compose -f deploy/docker-compose.yml up -d
```

### Step 4: Migrate Data

```bash
# Export from old format
python scripts/export_v1.8.py > migration_data.json

# Import to new format
python scripts/import_v1.9.py --input migration_data.json
```

### Step 5: Update Clients

Update all client configurations to use new MCP endpoints:
- Old: `http://localhost:8094`
- New: `http://localhost:8092/mcp`
