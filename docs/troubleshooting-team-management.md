# Troubleshooting: Team Management

> Diagnosing and fixing team initialization, size violations, phase gates, and role assignment errors

See the [Troubleshooting Index](troubleshooting.md) for other topics.

---

## Team Initialization Failed

**Symptoms:**
```
Error: TEAM-001: Team not found
Error: Failed to initialize project
```

**Causes:**
- Project name contains invalid characters
- `.teams/` directory does not exist
- Permission issues

**Solutions:**

1. **Verify project name:**
   ```bash
   # Valid: my-project, project_123, team-alpha
   # Invalid: my project, project;rm -rf /
   ```

2. **Create required directories:**
   ```bash
   mkdir -p .teams/
   chmod 755 .teams/
   ```

3. **Check permissions:**
   ```bash
   ls -la .teams/
   # Should be writable by current user
   ```

---

## Team Size Violation (TEAM-007)

**Symptoms:**
```
Error: TEAM-005: Team size violation
Team 7 has 8 members (maximum is 6)
```

**Causes:**
- Too many members assigned to a team
- Batch assignment exceeded limits

**Solutions:**

1. **Check current team sizes:**
   ```bash
   curl -X POST http://localhost:8080/mcp \
     -d '{
       "jsonrpc":"2.0",
       "method":"tools/call",
       "params":{
         "name":"guardrail_team_size_validate",
         "arguments":{"project_name":"my-project"}
       }
     }'
   ```

2. **Remove excess members:**
   ```bash
   curl -X POST http://localhost:8080/mcp \
     -d '{
       "jsonrpc":"2.0",
       "method":"tools/call",
       "params":{
         "name":"guardrail_team_unassign",
         "arguments":{
           "project_name":"my-project",
           "team_id":7,
           "role_name":"Extra Role"
         }
       }
     }'
   ```

3. **Rebalance across teams:**
   - Move members to teams with fewer than 4 members
   - Split large teams into multiple smaller teams

---

## Phase Gate Check Failed

**Symptoms:**
```
Error: Phase gate requirements not met
Missing deliverables: Architecture Decision Records
```

**Causes:**
- Required deliverables not complete
- Required teams not assigned
- Approval not obtained

**Solutions:**

1. **List gate requirements:**
   ```bash
   curl -X POST http://localhost:8080/mcp \
     -d '{
       "jsonrpc":"2.0",
       "method":"tools/call",
       "params":{
         "name":"guardrail_phase_gate_check",
         "arguments":{
           "project_name":"my-project",
           "from_phase":1,
           "to_phase":2
         }
       }
     }'
   ```

2. **Complete missing deliverables:**
   - Review the phase gate requirements returned by the check above
   - Submit required documents
   - Obtain approvals

3. **Verify team assignments:**
   ```bash
   curl -X POST http://localhost:8080/mcp \
     -d '{
       "jsonrpc":"2.0",
       "method":"tools/call",
       "params":{
         "name":"guardrail_team_status",
         "arguments":{
           "project_name":"my-project",
           "phase":"Phase 1"
         }
       }
     }'
   ```

---

## Role Already Assigned

**Symptoms:**
```
Error: TEAM-004: Person already assigned
Role 'Technical Lead' already has 'Alice Johnson' assigned
```

**Causes:**
- Attempted to assign to an occupied role
- Duplicate assignment in batch script

**Solutions:**

1. **Unassign current person:**
   ```bash
   curl -X POST http://localhost:8080/mcp \
     -d '{
       "jsonrpc":"2.0",
       "method":"tools/call",
       "params":{
         "name":"guardrail_team_unassign",
         "arguments":{
           "project_name":"my-project",
           "team_id":7,
           "role_name":"Technical Lead"
         }
       }
     }'
   ```

2. **Assign new person:**
   ```bash
   curl -X POST http://localhost:8080/mcp \
     -d '{
       "jsonrpc":"2.0",
       "method":"tools/call",
       "params":{
         "name":"guardrail_team_assign",
         "arguments":{
           "project_name":"my-project",
           "team_id":7,
           "role_name":"Technical Lead",
           "person":"New Lead Name"
         }
       }
     }'
   ```
