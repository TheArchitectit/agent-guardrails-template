# Tasks

## 1. Establish the output contract

- [x] 1.1 Inventory command wrappers, forwarded format arguments, backend option support, existing serializers, and effects. Record the command-by-command support matrix in evidence.
- [x] 1.2 Implement selected-phase and all-phase status JSON, preserve status text, restore the existing query/history JSON dispatch paths, and reject unsupported global output modes before command handlers run. Additional command serializers remain open.

## 2. Verify and archive

- [x] 2.1 Run real CLI and backend checks against disposable data, including the temporary installed prefix, per-phase and all-phase status, text and supported JSON controls, and pre-effect rejection. The regression entry exercises the combined DIST-01 install layout and synthetic project fixture.
- [ ] 2.2 Review the matrix and regression evidence independently, run pinned strict OpenSpec and repository checks, and archive after acceptance.
