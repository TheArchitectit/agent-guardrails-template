# Troubleshooting: Recovery Procedures

> Restoring from backup, resetting project state, repairing corrupted config, and emergency rollback

See the [Troubleshooting Index](troubleshooting.md) for other topics.

---

## Restore from Backup

**Prerequisites:**
- Backup files in `.teams/backups/`
- Valid project configuration

**Steps:**
```bash
# 1. Stop MCP server
pkill -f mcp_server

# 2. Backup current state
cp -r .teams/ .teams/emergency-backup-$(date +%Y%m%d)

# 3. Restore from backup
cp .teams/backups/project-backup-20260214.json .teams/my-project.json

# 4. Restart server
python mcp_server.py

# 5. Verify restoration
curl -X POST http://localhost:8094/mcp \
  -d '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"guardrail_team_list","arguments":{"project_name":"my-project"}}}'
```

---

## Reset Project State

**Warning:** This will remove all team assignments and reset to initial state.

```bash
# 1. Archive current state
mv .teams/my-project.json .teams/my-project-$(date +%Y%m%d).json.bak

# 2. Re-initialize project
curl -X POST http://localhost:8094/mcp \
  -d '{
    "jsonrpc":"2.0",
    "method":"tools/call",
    "params":{
      "name":"guardrail_team_init",
      "arguments":{"project_name":"my-project"}
    }
  }'

# 3. Re-assign team members from backup reference
```

---

## Repair Corrupted Configuration

**Symptoms:**
```
Error: Invalid JSON in team configuration
Error: PROJ-002: Project configuration missing
```

**Steps:**
```bash
# 1. Validate JSON syntax
python -m json.tool .teams/my-project.json > /dev/null

# 2. If invalid, try to recover
python scripts/repair_config.py .teams/my-project.json

# 3. If recovery fails, restore from backup
cp .teams/backups/my-project.json .teams/my-project.json
```

---

## Emergency Rollback

Use when critical errors occur during batch operations:

```bash
#!/bin/bash
# emergency_rollback.sh

PROJECT_NAME="$1"
BACKUP_FILE=".teams/backups/${PROJECT_NAME}-pre-batch.json"

if [ ! -f "$BACKUP_FILE" ]; then
    echo "No backup found for $PROJECT_NAME"
    exit 1
fi

echo "Rolling back $PROJECT_NAME..."
cp "$BACKUP_FILE" ".teams/${PROJECT_NAME}.json"
echo "Rollback complete."

echo "Verifying..."
curl -s -X POST http://localhost:8094/mcp \
  -d "{\"jsonrpc\":\"2.0\",\"method\":\"tools/call\",\"params\":{\"name\":\"guardrail_team_list\",\"arguments\":{\"project_name\":\"$PROJECT_NAME\"}}}"
```
