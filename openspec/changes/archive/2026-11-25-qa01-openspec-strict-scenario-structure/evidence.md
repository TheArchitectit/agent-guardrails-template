# QA-01 verification

OpenSpec 2.2.0 strict validation of review baseline `421cb8a` reported
seven failures across the seven short-name published specs. The
implementation changes 25 existing scenario sentences to structured
GIVEN, WHEN, and THEN clauses. An independent reverse normalization
reproduced all seven baseline files byte-for-byte.

All seven repaired specs passed targeted strict validation. Before archive,
full-tree strict validation passed all 53 entries, including the existing
active changes and published specs. Run the local repository-wide check
from the repository root with the official OpenSpec 2.2.0 CLI:

```sh
OPENSPEC_TELEMETRY=0 openspec validate --all --strict
```

This evidence covers specification structure. The archived scope uses
the local procedure. Runtime proof remains governed by the individual
requirements and acceptance tasks.
