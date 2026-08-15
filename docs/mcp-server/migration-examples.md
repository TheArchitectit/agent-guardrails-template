# Migration: Examples

> Worked examples for single-project, batch, and zero-downtime migrations

See the [Migration Overview](version-migration.md) for the compatibility matrix.
See [Migration Procedures](migration-procedures.md) for step-by-step upgrade paths.

---

## Example 1: Single Project Migration

```bash
# Migrate single project from v1.9.0 to v2.0.0
PROJECT_NAME="my-web-app"

# Step 1: Export project config
python scripts/export_project.py "$PROJECT_NAME" > "$PROJECT_NAME-v1.9.json"

# Step 2: Transform to v2.0.0 format
python scripts/transform_team_config.py \
  --input "$PROJECT_NAME-v1.9.json" \
  --from-version 1.9 \
  --to-version 2.0 \
  --output "$PROJECT_NAME-v2.0.json"

# Step 3: Validate new format
python scripts/validate_team_config.py "$PROJECT_NAME-v2.0.json"

# Step 4: Import to v2.0.0
python scripts/import_project.py "$PROJECT_NAME-v2.0.json"

# Step 5: Verify
./scripts/verify_project.sh "$PROJECT_NAME"
```

---

## Example 2: Batch Migration Script

```bash
#!/bin/bash
# migrate_all_projects.sh

set -e

FROM_VERSION="1.9.0"
TO_VERSION="2.0.0"
FAILED_LOG="migration_failed_$(date +%Y%m%d).log"
SUCCESS_COUNT=0
FAIL_COUNT=0

echo "Starting batch migration from $FROM_VERSION to $TO_VERSION"

# Get all projects
PROJECTS=$(ls .teams/*.json | xargs -n1 basename | sed 's/.json$//')

for PROJECT in $PROJECTS; do
    echo "Migrating $PROJECT..."

    if python scripts/migrate_project.py \
        --project "$PROJECT" \
        --from-version "$FROM_VERSION" \
        --to-version "$TO_VERSION" \
        --backup; then

        echo "  SUCCESS: $PROJECT"
        ((SUCCESS_COUNT++))
    else
        echo "  FAILED: $PROJECT"
        echo "$PROJECT" >> "$FAILED_LOG"
        ((FAIL_COUNT++))
    fi
done

echo ""
echo "Migration Summary:"
echo "  Successful: $SUCCESS_COUNT"
echo "  Failed: $FAIL_COUNT"

if [ $FAIL_COUNT -gt 0 ]; then
    echo "Failed projects logged to: $FAILED_LOG"
    exit 1
fi
```

---

## Example 3: Zero-Downtime Migration

For v1.10.0 (no breaking changes):

```bash
#!/bin/bash
# zero_downtime_migration.sh

# Start new version on different port
MCP_PORT=8096 WEB_PORT=8097 ./mcp-server/cmd/server/server &
NEW_PID=$!

# Wait for health check
for i in {1..30}; do
    if curl -s http://localhost:8096/mcp/v1/health; then
        echo "New server ready"
        break
    fi
    sleep 1
done

# Switch load balancer to new port
sudo sed -i 's/8094/8096/g' /etc/nginx/conf.d/mcp.conf
sudo nginx -s reload

# Stop old server
pkill -f "mcp_server.*8094"

# Update to use standard port
kill $NEW_PID
MCP_PORT=8094 ./mcp-server/cmd/server/server &
sudo sed -i 's/8096/8094/g' /etc/nginx/conf.d/mcp.conf
sudo nginx -s reload

echo "Zero-downtime migration complete"
```
