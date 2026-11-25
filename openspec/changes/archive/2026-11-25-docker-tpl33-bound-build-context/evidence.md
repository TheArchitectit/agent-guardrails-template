# TPL-33 verification

The ignore file sits at the MCP module build context root and preserves the
Go modules, Go source, web assets, and tracked prevention rules required by
the Dockerfile. Later exclusion rules take precedence inside these trees.

The three focused Docker context tests passed on Windows with Docker Desktop's
`desktop-linux` context and `MCP_DOCKER_CONTEXT_DAEMON=available`. The
daemon-backed fixture proved 10 synthetic local-only paths were absent and
five required inputs remained; fixture contents were not printed. Excluded
paths were `.env`, `.git/config`, `.cache/fixture`,
`__pycache__/fixture.pyc`, `web/.env`, `web/.git/config`,
`web/.cache/fixture`, `.guardrails/prevention-rules/.env`,
`.guardrails/prevention-rules/.git/config`, and
`.guardrails/prevention-rules/.cache/fixture`. Retained paths were `go.mod`,
`go.sum`, `cmd/server/main.go`, `web/index.html`, and
`.guardrails/prevention-rules/extracted-rules.json`. The helper's Dockerfile
path assertion was corrected to compare resolved paths on Windows.
The exact production command
`docker build --no-cache -f deploy/Dockerfile -t <unique-tag> .` succeeded
from `mcp-server` using the pinned Go 1.26.11 builder and distroless base.
The unique image tag was removed after the build; no unrelated containers,
volumes, or settings were touched.
