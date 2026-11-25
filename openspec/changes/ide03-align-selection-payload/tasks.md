# Tasks

## 1. Define a compatible handler contract

- [ ] 1.1 Make `selection` canonical; accept legacy `code` only when `selection` is absent; reject conflicting dual-field values with HTTP 400. Do not silently normalize source text.
- [ ] 1.2 Add Go handler tests for `selection` only (accepted), `code` only (accepted during migration), equal dual fields (accepted), conflicting fields (400), and missing/empty fields (400). Verify with `cd mcp-server && go test ./internal/web -run 'TestValidateSelectionPayload' -count=1`.

## 2. Migrate and verify IDE clients

- [ ] 2.1 Update VS Code, JetBrains, Vim, and Neovim request builders to send `selection`; add client contract tests asserting each request is accepted by the server fixture.
- [ ] 2.2 Compile VS Code with `cd ide/vscode-extension && npm ci && npm run compile`; run targeted client and handler tests and confirm legacy `code` compatibility remains.
