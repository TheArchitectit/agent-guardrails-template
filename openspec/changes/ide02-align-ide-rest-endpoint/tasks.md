# Tasks

## 1. Align documented REST defaults

- [ ] 1.1 Update default URL values and install examples for VS Code, JetBrains, Vim, and Neovim to `http://localhost:8081`; leave explicit per-client overrides intact.
- [ ] 1.2 Add `tests/test_ide_rest_endpoint_contract.py` to enumerate the four clients and assert each default/example is 8081, the server web default is 8081, and the MCP default remains 8080. Include a negative fixture with a 8095 value and assert it is named as a failure. Verify with `python3 -m pytest tests/test_ide_rest_endpoint_contract.py -q`.

## 2. Verify served endpoint and overrides

- [ ] 2.1 Add a Go handler integration test that sends a readiness/IDE REST request to the web listener and verifies the route responds; run `cd mcp-server && go test ./internal/web -run 'TestIDERestEndpointContract' -count=1`.
- [ ] 2.2 Compile VS Code with `cd ide/vscode-extension && npm ci && npm run compile`; verify JetBrains/Vim/Neovim tests preserve a user-specified non-default URL.
