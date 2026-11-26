# Give audit work items unique identifiers

## Decisions

The gap register is the allocation record for template audit work. Historical rows reserve their identifiers. Check the register and existing change paths before assigning an identifier. When an active change carries a numbered path, use the same identifier in its path and every queue or audit entry. Keep references to historical work with their original archive evidence.

Assign TPL-52 to optional surface authorization, TPL-53 to MCP session topology, and TPL-54 to organization-scoped documents. TPL-55 records this reconciliation. The three proposals, design files, task files, and spec deltas move together.

## Verification

Read IDs from the gap register table rows and reject repeated IDs. Verify that all three audit and queue entries use the newly allocated IDs, proposal links resolve, and the prior IDs still resolve to their original archives. Compare the moved packages byte-for-byte with their prior content and run pinned strict OpenSpec.
