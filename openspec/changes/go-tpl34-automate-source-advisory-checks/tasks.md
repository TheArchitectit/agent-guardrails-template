# Tasks

## 1. Add structured advisory classification

- [ ] 1.1 Add a result classifier for govulncheck JSON: source-call-path findings fail; imported-only and module-only findings are reported separately; malformed output, scanner/database/toolchain errors, and zero analyzed packages fail closed.
- [ ] 1.2 Add sanitized fixture tests for source-reachable, imported-only, module-only, no-findings with analyzed packages, malformed output, scanner failure, and zero-package output. Verify with `python3 -m pytest mcp-server/tests/test_govulncheck_gate.py -q`.

## 2. Run the advisory job continuously

- [ ] 2.1 Add a PR and main-push workflow job that reads Go from `mcp-server/go.mod`, installs a pinned govulncheck version, runs `govulncheck -json ./...` from `mcp-server/`, passes output to the classifier, and retains a redacted report artifact.
- [ ] 2.2 Document the exact local command and run it with `GOTOOLCHAIN=local` from `mcp-server/`; verify a source-reachable fixture fails CI while module-only findings remain visible in a separate summary.
