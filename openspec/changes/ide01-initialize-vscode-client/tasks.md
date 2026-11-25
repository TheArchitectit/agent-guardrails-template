# Tasks

## 1. Initialize and preserve configuration

- [ ] 1.1 Initialize the client with `context` before the first connection check; preserve command registration, disposal, and configuration-change refresh.
- [ ] 1.2 Add an activation test where a non-default `serverUrl` and a SecretStorage key are supplied; assert the first fetch uses that URL and bearer key and a 200 response marks the status connected. Add a rejected-fetch control and assert disconnected status without activation throwing.

## 2. Verify the VS Code extension

- [ ] 2.1 From `ide/vscode-extension/`, run `npm ci`, `npm run compile`, `npm run lint`, and the activation test command; record each result.
- [ ] 2.2 Confirm existing explicit `guardrail.serverUrl` settings and configuration-change refresh still work in tests.
