# Tasks

## 1. Define a compatible handler contract

- [x] 1.1 Make `selection` canonical; accept legacy `code` only when `selection` is absent; reject conflicting dual-field values with HTTP 400. Do not silently normalize source text.
- [x] 1.2 Add Go handler tests for `selection` only (accepted), `code` only (accepted during migration), equal dual fields (accepted), conflicting fields (400), and missing/empty fields (400). Verified with the targeted Go tests and a disposable PostgreSQL-backed Echo route fixture.

## 2. Migrate and verify IDE clients

- [x] 2.1 Update VS Code, JetBrains, Vim, and Neovim request builders to send `selection`; capture each builder's emitted JSON and replay it through the registered Echo route against the disposable PostgreSQL fixture. All four requests returned HTTP 200 with the expected synthetic match; see `evidence.md` for commands and the JetBrains/Vim/Neovim runtime boundaries.
- [x] 2.2 Run locked `npm ci` and `npm test` on Node 22.20; pretest TypeScript compilation and ESLint, the client regressions, and `go test ./internal/web -run 'Test.*Selection' -count=1` passed. Legacy `code` compatibility is covered by the handler tests.
