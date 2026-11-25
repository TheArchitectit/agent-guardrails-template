# Tasks

## 1. Establish the inventory source and identities

- [ ] 1.1 Locate the index generator or document that no generator exists; verify with a repository search for the generation source and its inputs.
- [ ] 1.2 Enumerate all OpenSpec requirement files by full path and reconcile the fifteen-file count against `find openspec/specs -name spec.md`.
- [ ] 1.3 Compare overlapping short/verbose variants at requirement level and document distinct requirements and any owner-approved disposition.

## 2. Reconcile index and traceability

- [ ] 2.1 Update the index using the identified source-of-truth; verify every spec path and status row matches its own closure/status evidence.
- [ ] 2.2 Map each archived AIGGP change to its produced spec files and verify links resolve locally without asserting runtime closure.
- [ ] 2.3 Run `openspec validate --specs --strict` and compare the index rows to the validator's complete spec inventory.
