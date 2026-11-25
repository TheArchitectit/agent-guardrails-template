# IDE-04 Execution Evidence

Validated in the isolated Windows worktree at the requested base commit. No commit, push, archive, dependency upgrade, or change to the lead checkout was made.

## Lint configuration

The package-root ESLint config uses only existing lockfile dependencies: ESLint 8, `@typescript-eslint/parser`, and `@typescript-eslint/eslint-plugin`. Core and TypeScript recommended rules plus unused-variable, explicit-`any`, and `no-debugger` errors are enabled. Empty class methods are allowed for intentional no-op lifecycle methods such as `GuardrailClient.dispose()`; the remaining recommended rules stay active.

## Negative controls

- A temporary invalid-syntax `.ts` file made `npm run lint` fail with a parser error.
- A temporary parseable `.ts` file with an unused binding made `npm run lint` fail with `@typescript-eslint/no-unused-vars`.
- Both fixtures were removed after their respective runs; final path checks confirmed they are absent.

## Package workflow

- Locked `npm ci` installed 247 packages. npm reported 14 development dependency advisories (4 moderate, 10 high); no dependency or lockfile changes were made.
- `npm run compile` passed. A repeated compile produced identical SHA-256 hashes for `out/extension.js` and `out/extension.js.map`; unrelated generated output was restored.
- `npm run lint` passed.
- Normal `npm test` passed its pretest sequence (compile, lint) and activation harness.

## Activation regression

`tests/regression/test_vscode_extension_initialization_regression_FAIL_4c35b1a8.js` is the single activation regression, run by `npm test` from the extension package. It imports the real checked-in entry point `ide/vscode-extension/out/extension.js`, which `pretest` compiles from `ide/vscode-extension/src/extension.ts`.

The test run against the committed pre-IDE-01 activation source failed with `TypeError: Cannot read properties of undefined (reading 'enabled')` before the first readiness request. The fixed source passed with the configured non-default URL and SecretStorage bearer key, configuration refresh, connected state on HTTP 200, and disconnected state when fetch rejects. The failure is recorded as resolved `FAIL-4c35b1a8` through `scripts/log_failure.py`; the ID in the test filename and header links the test to the registry entry. No fix commit is recorded because no commit was created.

## OpenSpec

Official OpenSpec CLI 2.2.0 on Node 22 passed strict validation for both `ide01-initialize-vscode-client` and `restore-vscode-extension-lint-config`.
