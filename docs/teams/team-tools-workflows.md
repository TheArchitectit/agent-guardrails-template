# Workflows and Batch Operations

> Common workflow patterns and batch operation scripts for team management

Practical patterns for setting up projects, transitioning phases, and performing batch team assignments. Includes shell scripts for common multi-step operations.

---

## Typical Project Setup Workflow

```
1. Initialize team structure
   └─ guardrail_team_init → Creates all 12 teams

2. Assign team members to roles
   └─ guardrail_team_assign → Assign people to specific roles

3. Check phase status
   └─ guardrail_team_status → Verify team readiness

4. Progress through phase gates
   └─ guardrail_phase_gate_check → Validate gate requirements
```

For the agent-specific assignment workflow, see [team-tools-agent-mapping.md](./team-tools-agent-mapping.md).

---

## Example: Complete Project Initialization

```bash
# Initialize project
curl -X POST "http://localhost:8094/mcp" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"guardrail_team_init","arguments":{"project_name":"web-platform"}}}'

# Assign backend lead
curl -X POST "http://localhost:8094/mcp" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"guardrail_team_assign","arguments":{"project_name":"web-platform","team_id":7,"role_name":"Technical Lead","person":"Alice Developer"}}}'

# Check phase gate
curl -X POST "http://localhost:8094/mcp" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"guardrail_phase_gate_check","arguments":{"project_name":"web-platform","from_phase":2,"to_phase":3}}}'
```

---

## Batch Operations

When setting up a complete project, you may need to perform multiple team assignments. Here are recommended patterns for batch operations:

### Batch Team Assignment Pattern

```bash
#!/bin/bash
# batch_assign_teams.sh - Assign multiple team members in sequence

PROJECT_NAME="$1"

if [ -z "$PROJECT_NAME" ]; then
    echo "Usage: $0 <project_name>"
    exit 1
fi

# Define assignments as: "team_id|role_name|person_name"
declare -a ASSIGNMENTS=(
    "2|Solution Architect|Alice Johnson"
    "2|Domain Architect|Bob Smith"
    "4|Cloud Architect|Carol White"
    "7|Technical Lead|David Brown"
    "7|Senior Backend Engineer|Eve Davis"
    "9|Security Architect|Frank Miller"
    "10|QA Architect|Grace Wilson"
)

echo "Initializing team structure..."
curl -s -X POST "http://localhost:8094/mcp" \
    -H "Content-Type: application/json" \
    -d "{\"jsonrpc\":\"2.0\",\"method\":\"tools/call\",\"params\":{\"name\":\"guardrail_team_init\",\"arguments\":{\"project_name\":\"$PROJECT_NAME\"}}}"

echo "Assigning team members..."
for assignment in "${ASSIGNMENTS[@]}"; do
    IFS='|' read -r team_id role_name person <<< "$assignment"

    echo "  -> Assigning $person as $role_name to Team $team_id"
    curl -s -X POST "http://localhost:8094/mcp" \
        -H "Content-Type: application/json" \
        -d "{\"jsonrpc\":\"2.0\",\"method\":\"tools/call\",\"params\":{\"name\":\"guardrail_team_assign\",\"arguments\":{\"project_name\":\"$PROJECT_NAME\",\"team_id\":$team_id,\"role_name\":\"$role_name\",\"person\":\"$person\"}}}"
done

echo "Validating team sizes..."
curl -s -X POST "http://localhost:8094/mcp" \
    -H "Content-Type: application/json" \
    -d "{\"jsonrpc\":\"2.0\",\"method\":\"tools/call\",\"params\":{\"name\":\"guardrail_team_size_validate\",\"arguments\":{\"project_name\":\"$PROJECT_NAME\"}}}"

echo "Done!"
```

### Batch Role Reassignment Pattern

```bash
#!/bin/bash
# batch_reassign.sh - Unassign and reassign roles for restructuring

PROJECT_NAME="$1"

# First unassign old roles, then assign new ones
declare -a UNASSIGNMENTS=(
    "7|Old Technical Lead"
    "7|Legacy Developer"
)

declare -a NEW_ASSIGNMENTS=(
    "7|Technical Lead|New Lead Name"
    "7|Senior Backend Engineer|New Developer"
)

# Unassign old roles
for unassign in "${UNASSIGNMENTS[@]}"; do
    IFS='|' read -r team_id role_name <<< "$unassign"
    echo "Unassigning $role_name from Team $team_id"
    curl -s -X POST "http://localhost:8094/mcp" \
        -H "Content-Type: application/json" \
        -d "{\"jsonrpc\":\"2.0\",\"method\":\"tools/call\",\"params\":{\"name\":\"guardrail_team_unassign\",\"arguments\":{\"project_name\":\"$PROJECT_NAME\",\"team_id\":$team_id,\"role_name\":\"$role_name\"}}}"
done

# Assign new roles
for assign in "${NEW_ASSIGNMENTS[@]}"; do
    IFS='|' read -r team_id role_name person <<< "$assign"
    echo "Assigning $person as $role_name to Team $team_id"
    curl -s -X POST "http://localhost:8094/mcp" \
        -H "Content-Type: application/json" \
        -d "{\"jsonrpc\":\"2.0\",\"method\":\"tools/call\",\"params\":{\"name\":\"guardrail_team_assign\",\"arguments\":{\"project_name\":\"$PROJECT_NAME\",\"team_id\":$team_id,\"role_name\":\"$role_name\",\"person\":\"$person\"}}}"
done
```

### Validation Before Phase Transition

```bash
#!/bin/bash
# validate_phase_transition.sh - Check phase gate before transitioning

PROJECT_NAME="$1"
FROM_PHASE="$2"
TO_PHASE="$3"

echo "Checking phase gate from Phase $FROM_PHASE to Phase $TO_PHASE..."

# Validate team sizes first
echo "Validating team sizes..."
curl -s -X POST "http://localhost:8094/mcp" \
    -H "Content-Type: application/json" \
    -d "{\"jsonrpc\":\"2.0\",\"method\":\"tools/call\",\"params\":{\"name\":\"guardrail_team_size_validate\",\"arguments\":{\"project_name\":\"$PROJECT_NAME\"}}}"

# Check phase status for all teams in source phase
echo "Checking teams in Phase $FROM_PHASE..."
curl -s -X POST "http://localhost:8094/mcp" \
    -H "Content-Type: application/json" \
    -d "{\"jsonrpc\":\"2.0\",\"method\":\"tools/call\",\"params\":{\"name\":\"guardrail_team_status\",\"arguments\":{\"project_name\":\"$PROJECT_NAME\",\"phase\":\"Phase $FROM_PHASE\"}}}"

# Check phase gate requirements
echo "Checking phase gate requirements..."
curl -s -X POST "http://localhost:8094/mcp" \
    -H "Content-Type: application/json" \
    -d "{\"jsonrpc\":\"2.0\",\"method\":\"tools/call\",\"params\":{\"name\":\"guardrail_phase_gate_check\",\"arguments\":{\"project_name\":\"$PROJECT_NAME\",\"from_phase\":$FROM_PHASE,\"to_phase\":$TO_PHASE}}}"

echo "Validation complete. Review output above before proceeding."
```

### Error Handling in Batch Operations

When performing batch operations, handle validation errors gracefully:

```bash
#!/bin/bash
# batch_with_error_handling.sh

PROJECT_NAME="$1"
TEMP_DIR=$(mktemp -d)
FAILED_FILE="$TEMP_DIR/failed_assignments.txt"
SUCCESS_COUNT=0
FAILURE_COUNT=0

process_assignment() {
    local team_id=$1
    local role_name=$2
    local person=$3

    response=$(curl -s -X POST "http://localhost:8094/mcp" \
        -H "Content-Type: application/json" \
        -d "{\"jsonrpc\":\"2.0\",\"method\":\"tools/call\",\"params\":{\"name\":\"guardrail_team_assign\",\"arguments\":{\"project_name\":\"$PROJECT_NAME\",\"team_id\":$team_id,\"role_name\":\"$role_name\",\"person\":\"$person\"}}}")

    # Check if response indicates error
    if echo "$response" | grep -q '"IsError":true'; then
        echo "FAILED: $person as $role_name in Team $team_id"
        echo "$team_id|$role_name|$person" >> "$FAILED_FILE"
        ((FAILURE_COUNT++))
        return 1
    else
        echo "SUCCESS: $person as $role_name in Team $team_id"
        ((SUCCESS_COUNT++))
        return 0
    fi
}

# Process all assignments
# ... (assignment loop)

echo "---"
echo "Batch Operation Summary:"
echo "  Successful: $SUCCESS_COUNT"
echo "  Failed: $FAILURE_COUNT"

if [ $FAILURE_COUNT -gt 0 ]; then
    echo "Failed assignments saved to: $FAILED_FILE"
    echo "Review failures and retry if needed."
fi
```

---

## Related Documentation

- [team-tools.md](./team-tools.md) - Overview of all team tools
- [team-tools-management.md](./team-tools-management.md) - Individual tool reference
- [team-tools-phase-gates.md](./team-tools-phase-gates.md) - Phase gate details
- [team-tools-errors.md](./team-tools-errors.md) - Error codes for batch error handling
- [team-tools-validation.md](./team-tools-validation.md) - Validation rules
