# Tasks

## 1. Repair scenario structure

- [x] 1.1 For each of the seven baseline failing specs, add WHEN/GIVEN triggers and THEN outcomes while preserving requirement meaning; verify each with `openspec validate <spec> --strict`.
- [x] 1.2 Review the diff against the original requirement text to ensure only scenario structure changed and no requirement was silently dropped.

## 2. Verify full-tree strict validation

- [x] 2.1 Run `openspec validate --all --strict` and verify that every spec and active change is included and passes.
- [x] 2.2 If CI enforcement is in scope, add a required workflow invocation and verify its command path and failure behavior; otherwise record the local validation procedure without claiming CI enforcement.
