# Tasks

## 1. Initialize and preserve configuration

- [x] 1.1 Initialize the client with `context` before the first connection check; preserve command registration, disposal, and configuration-change refresh.
- [x] 1.2 Add an activation test where a non-default `serverUrl` and a SecretStorage key are supplied; assert the first fetch uses that URL and bearer key and a 200 response marks the status connected. Add a rejected-fetch control and assert disconnected status without activation throwing.

## 2. Verify the VS Code extension

- [x] 2.1 From `ide/vscode-extension/`, run `npm ci`, `npm run compile`, `npm run lint`, and the activation test command; record each result.
- [x] 2.2 Confirm existing explicit `guardrail.serverUrl` settings and configuration-change refresh still work in tests.

## Execution evidence

- `npm ci` in `ide/vscode-extension/` using Node 20: passed; installed the locked 247 packages. npm reported 14 dependency advisories (4 moderate, 10 high).
- `npm run compile`: passed; the tracked runtime entry point and source map were regenerated for the activation fix.
- `npm run lint`: passed with the project-scoped configuration added by IDE-04.
- `npm test`: passed the normal `pretest` sequence (compile, lint) and activation harness.
- Regression test `tests/regression/test_vscode_extension_initialization_regression_FAIL_4c35b1a8.js` passed configured URL/SecretStorage, configuration refresh, and unreachable-server controls.
- `npx --yes @fission-ai/openspec@2.2.0 validate ide01-initialize-vscode-client --strict` (Node 22): passed.
