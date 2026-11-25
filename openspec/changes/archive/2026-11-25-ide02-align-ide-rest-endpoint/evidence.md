# IDE-02 validation

All four IDE defaults and their setup examples use the REST listener on 8081.
The contract keeps the MCP listener on 8080 and checks the explicit custom URL
paths for JetBrains, Vim, and Neovim at source/fixture level. Native editor
sessions were outside the local acceptance run.

- Python endpoint contract passed all four cases, including the stale-default negative control.
- Go 1.26.11 race-enabled ready and starting controls both passed. The test registers the production route table on an ephemeral loopback listener with synthetic readiness dependencies.
- The direct regression shell passed Python 4/4 and emitted the exact Go test PASS line.
- Node 20 locked npm install and the normal npm test lifecycle passed compilation, lint, startup, and the deferred SecretStorage Configure regression. Fresh compilation reproduced the three changed tracked output files.
- The compiled Configure regression failed on the original unawaited reload and passed when the saved override was loaded before the first probe.
- Direct Windows pilot readiness checks returned HTTP 200 through both localhost and 127.0.0.1 on 8081.

Regression entries: `tests/regression/test_ide_rest_endpoint_contract_regression_FAIL-4490ed64.sh` and `tests/regression/test_vscode_configure_override_regression_FAIL-a01027ef.js`.
