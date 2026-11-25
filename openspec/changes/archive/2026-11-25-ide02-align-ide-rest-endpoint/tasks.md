# Tasks

## 1. Align documented REST defaults

- [x] 1.1 Updated default URL values and install examples for VS Code, JetBrains, Vim, and Neovim to `http://localhost:8081`; explicit per-client overrides remain intact.
- [x] 1.2 Added `tests/test_ide_rest_endpoint_contract.py` to enumerate the four clients and assert each default/example is 8081, the server web default is 8081, and the MCP default remains 8080. A negative fixture with a 8095 value is named in the failure assertion. Added the registry-linked shell regression entry `tests/regression/test_ide_rest_endpoint_contract_regression_FAIL-4490ed64.sh` to run the canonical Python and Go cases without duplicating assertions. Verified with `python3 -m pytest tests/test_ide_rest_endpoint_contract.py -q`.

## 2. Verify served endpoint and overrides

- [x] 2.1 Added `mcp-server/internal/web/ide_rest_endpoint_contract_test.go`, registering the production route table on an ephemeral loopback listener with synthetic readiness dependencies. Verified `cd mcp-server && go test ./internal/web -run '^TestIDERestEndpointContract$' -count=1 -v`; the named test must emit `PASS`.
- [x] 2.2 Ran locked `npm ci`, compile, lint, and the normal package `npm test` entry point. The Python contract suite checks explicit non-default URL pass-through for JetBrains, Vim, and Neovim; the compiled VS Code Configure regression defers SecretStorage and proves the saved URL is used for the first probe. A pre-fix control failed with a premature request to `http://localhost:8081/health/ready`; the fixed compiled test passed. Retained only intended regenerated `out/commands.js`, its source map, and `out/utils/client.js`; repeated compile produced identical Git blob hashes.

[Verification evidence](evidence.md)
