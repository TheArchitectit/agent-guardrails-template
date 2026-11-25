# Tasks

## 1. Normalize and constrain paths

- [ ] 1.1 Change target normalization to remove only a literal `./` prefix and add a temporary-root test proving `.claude` remains in the installed path.
- [ ] 1.2 Resolve and constrain destination paths to the configured root and add a traversal test verifying no outside write occurs.

## 2. Verify installer compatibility

- [ ] 2.1 Run the focused installer test suite and verify existing non-hidden platform destinations still resolve as before.
