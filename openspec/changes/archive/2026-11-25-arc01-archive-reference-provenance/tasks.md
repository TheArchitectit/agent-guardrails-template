# Tasks

## 1. Verify archived outputs and references

- [x] 1.1 Enumerate every archived AIGGP delta path and verify the unique output count and omitted `isolation-levels` entry.
- [x] 1.2 Resolve each package-header retirement reference against the local repository; document any record that is absent and identify only a verifiable replacement.

## 2. Correct and validate the historical manifest

- [x] 2.1 Update the archive README and stale local references without editing archived proposals/deltas; verify its list matches the enumerated paths.
- [x] 2.2 Validate relative links and run `openspec validate --archived --strict` to confirm archived package artifacts remain valid.
