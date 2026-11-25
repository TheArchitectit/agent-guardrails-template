# Tasks

## 1. Restore package-scoped linting

- [x] 1.1 Add a package-root ESLint configuration using only the locked ESLint 8, TypeScript parser, and plugin; preserve the existing package scripts and dependency lock.
- [x] 1.2 Run temporary invalid-syntax and parseable unused-binding fixtures through the package lint command, assert distinct parser/rule failures, and remove the fixtures.

## 2. Verify both VS Code packages

- [x] 2.1 From `ide/vscode-extension/`, run `npm ci`, `npm run compile`, `npm run lint`, and normal `npm test`; confirm `pretest` still runs compile, lint, then the activation test.
- [x] 2.2 Show the activation harness fails specifically against the original `extension.ts` and passes against the initialized activation flow; verify configuration refresh and the unreachable-server control still pass.
- [x] 2.3 Run strict validation for IDE-01 and IDE-04; inspect the final diff, confirm temporary fixtures are absent, and verify only the intended tracked runtime outputs changed.

## Execution evidence

- `npm ci` in `ide/vscode-extension/` using Node 20: passed; installed 247 locked packages without changing either package manifest or lockfile. npm reported 14 advisories in the locked development dependency tree (4 moderate, 10 high); dependencies were not upgraded.
- The package-scoped `.eslintrc.json` uses the existing ESLint 8 and TypeScript parser/plugin. Core and TypeScript recommended rules, unused-variable and explicit-`any` errors, and `no-debugger` are active. Empty lifecycle methods are allowed for the existing no-op `dispose()` hook; other recommended checks remain enabled.
- Controlled invalid-syntax fixture: `npm run lint` rejected it with a parser error. Controlled parseable unused-binding fixture: `npm run lint` rejected it with `@typescript-eslint/no-unused-vars`. Both temporary files were removed.
- `npm run compile`: passed. Repeated compilation reproduced identical bytes for `out/extension.js` and `out/extension.js.map`; other generated output was restored.
- `npm run lint`: passed.
- Normal `npm test`: passed its existing compile → lint pretest and the activation test.
- Original-source red control: `tests/regression/test_vscode_extension_initialization_regression_FAIL_4c35b1a8.js` run through normal `npm test` against the committed pre-IDE-01 `extension.ts` failed with `TypeError: Cannot read properties of undefined (reading 'enabled')`. The fixed source then passed the same test, including SecretStorage, refresh, successful response, and rejected-fetch controls. The failure is recorded as resolved `FAIL-4c35b1a8` by `scripts/log_failure.py`; the ID in the test filename and header links it to the registry entry. No fix commit is recorded because no commit was created.
- Official OpenSpec CLI 2.2.0 on Node 22 passed strict validation for both IDE-01 and IDE-04.
- Final diff review: no dependency or lockfile changes; no test fixtures remain. Only the activation runtime output and source map changed under `out/`.
