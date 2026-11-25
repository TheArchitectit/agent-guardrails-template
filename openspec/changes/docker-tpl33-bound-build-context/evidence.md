# TPL-33 verification

The ignore file sits at the MCP module build context root and preserves the
Go modules, Go source, web assets, and tracked prevention rules required by
the Dockerfile. Later exclusion rules take precedence inside these trees.

Two daemon-free pytest checks passed. The Docker fixture and production
image build remain pending. To run the context fixture where Docker is
available, set `MCP_DOCKER_CONTEXT_DAEMON=available` and run the targeted
pytest test. It uses a disposable image and synthetic contents only.
