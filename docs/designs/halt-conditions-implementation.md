# Halt Conditions — Implementation Notes and Examples

> Implementation notes, usage examples, file locations and references for the halt-conditions design.

**Part of:** [Halt Conditions Design](./halt-conditions-design.md)

---

## Integration Points

### Three Strikes Integration

The halt conditions system integrates with the Three Strikes Rule:

```
Attempt 1: Failure
  └─ Record attempt in halt_events with attempt_count=1
  └─ Return should_halt=false, recommended_action="retry"

Attempt 2: Failure
  └─ Record attempt in halt_events with attempt_count=2
  └─ Return should_halt=false, recommended_action="try_alternative"

Attempt 3: Failure
  └─ Record halt event with halt_type="execution", condition_name="three_strikes"
  └─ Return should_halt=true, severity="high"
  └─ Agent MUST halt and escalate to user
```

**Counter Reset Conditions:**
- New task (different scope)
- User provides new information
- User explicitly says "try again"
- New session started

**Counter Persistence:**
- Stored in `halt_events.attempt_count`
- Queryable via `guardrail_get_session_halts`

### Session State Integration

When `should_halt=true`:

1. Session state transitions to `halted`
2. No new operations allowed until acknowledged
3. Read operations may still be permitted (configurable)
4. All halt events must be acknowledged before resume

### Escalation workflow

1. Halt triggered → record in DB with severity.
2. If critical, auto-escalate: notify the user immediately. Otherwise, wait for acknowledgment.
3. User responds with one of: **resolve** (resume work), **escalate** (human takes over), or **dismiss** (resume with a warning).

---

## Implementation Notes

### Performance Considerations

- **Query Optimization:** All queries use indexed columns
- **Caching:** Halt config cached in memory, refresh every 60s
- **Batch Processing:** Multiple conditions checked in parallel
- **Event Retention:** Auto-archive events older than 90 days

### Security Considerations

- **Input Sanitization:** All context fields sanitized before storage
- **Secrets Masking:** Automatic redaction of secrets in `description`
- **Access Control:** Admin APIs require elevated permissions
- **Audit Logging:** All halt events immutable once recorded

### Error Handling

```
If DB unavailable during check:
  ├─ Log to local cache
  ├─ Return should_halt=true (fail safe)
  └─ severity=critical

If check logic fails:
  ├─ Log error
  ├─ Return should_halt=true
  └─ description="Halt check failed: [error]"
```

---

## Usage Examples

### Triggering a halt (code safety)

An agent tries to modify `src/auth.js` without reading it first, and with no rollback plan:

```typescript
const result = await guardrail_check_halt_conditions({
    session_token: "sess_abc123",
    current_context: {
        operation: "modify authentication logic",
        target_files: ["src/auth.js"],
        files_read: ["src/app.js"],   // auth.js NOT read
        attempt_number: 1
    },
    proposed_changes: { files_to_modify: ["src/auth.js"], has_tests: false, has_rollback_plan: false }
});
// → should_halt: true
//   halt_reasons: [
//     { condition_name: "modifying_unread_code", severity: "critical",
//       description: "Attempting to modify src/auth.js which has not been read" },
//     { condition_name: "no_rollback_plan", severity: "high" }
//   ]
```

### Three strikes

On the third failed attempt at a task, the check returns `three_strikes` (high) and `repeated_errors` (medium), with `recommended_action: "HALT and escalate. Recommend fresh session."`

### Acknowledging a halt

```typescript
const result = await guardrail_acknowledge_halt({
    session_token: "sess_abc123",
    halt_id: "halt_789xyz",
    resolution: "resolved",
    resolution_notes: "User confirmed they read the file in another session"
});
// → confirmed: true, session_can_resume: true
```

---

## File Locations

| Component | Path |
|-----------|------|
| Database Migration | `/db/migrations/NNN_add_halt_events_table.sql` |
| API Implementation | `/src/guardrails/halt_conditions.ts` |
| Configuration | `/config/halt_conditions.yaml` |
| Tests | `/tests/guardrails/halt_conditions.test.ts` |

---

## References

- [halt-conditions.md](../../skills/shared-prompts/halt-conditions.md) - Complete halt conditions list
- [AGENT_EXECUTION.md](../workflows/agent-execution.md) - Execution protocol and three strikes
- [AGENT_GUARDRAILS.md](../getting-started/agent-guardrails.md) - Core safety protocols

---

**Last Updated:** 2026-02-11
**Authored by:** TheArchitectit
**Document Owner:** Project Maintainers
**Line Count:** ~550

