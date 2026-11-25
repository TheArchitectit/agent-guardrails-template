# QA-05 Go advisory scan evidence

**Comparison baseline:** `e4f9b26`
**Scan platform:** Linux/amd64
**Scanner:** official `govulncheck` v1.10.0
**Vulnerability database:** modified 2026-11-24 15:56 UTC

The baseline source at `e4f9b26` matches the source reviewed for the initial
QA-05 audit. The before and after scans used the same vulnerability database.
From the `mcp-server/` module root, the scan command is:

```sh
GOTOOLCHAIN=local govulncheck ./...
```

The baseline used Go 1.25.5. The updated scan used Go 1.26.11. Govulncheck
reported these reachability tiers:

| Toolchain | Source-reachable findings | Imported-package findings without an apparent call path | Required-module findings without an apparent call path |
|---|---:|---:|---:|
| Go 1.25.5 | 45 | 19 | 39 |
| Go 1.26.11 | 0 | 0 | 4 |

The updated residuals are module-only findings: one in `golang.org/x/crypto`,
two in `golang.org/x/net`, and one in `github.com/mark3labs/mcp-go`. The scan
does not report standardized severity or CVSS values; these counts describe
govulncheck reachability classifications only.

The source and toolchain updates include `github.com/jackc/pgx/v5` 5.9.2,
`github.com/klauspost/compress` 1.20.2, `github.com/labstack/echo/v4` 4.16.3,
`golang.org/x/net` 0.60.0, and `golang.org/x/text` 0.42.0. For comparison, the
baseline pinned pgx 5.7.1, compress 1.17.9, Echo 4.13.3, x/net 0.33.0, and
x/text 0.21.0.

The Docker builder image is pinned to the verified multi-architecture index
`golang:1.26.11-alpine@sha256:ecb67f987aa1a57794c91d19ade7ae0209f2f0f5496aaef9e6ae85ddf92d02ab`.
All 32 Go setup steps in `team-validation.yml` and
`team-validation-fleet.yml` read `mcp-server/go.mod`, which declares Go
1.26.11. This aligns CI setup and the builder with the module toolchain.

## Build and image verification

The MCP module passed `go test -race -count=1 -timeout=15m ./...` with Go
1.26.11, scoped umask 0022, and allowed loopback access. The mutation-kill
package passed in 377.665 seconds. The Team CLI and Go example tests also
passed and met their discovery floors with 9 and 11 tests, respectively.
All three module builds passed with VCS stamping disabled. The nested
workspace's outer Git marker prevented default local VCS stamping.

A Linux/amd64 candidate image was built using a temporary Dockerfile variant
that mounted the combined CA bundle as a BuildKit secret only during APK and
Go dependency downloads. The normal local Docker build could not verify the
proxy's TLS certificate. The temporary mount was scoped to the download
steps. The candidate ran `--version` with network access disabled, and
`go version -m` confirmed Go 1.26.11 and the intended dependency versions.

The production CGO build's module-root scan,
`govulncheck -tags sqlite_omit_load_extension ./cmd/server`, reported zero
source-reachable findings, zero imported-package findings without an apparent
call path, and the same four required-module-only findings listed above. A
binary-mode scan reported those same four because the release binary is
stripped and govulncheck falls back to module information. This behavior is
consistent with the module scan and is not evidence of source call paths. The
read-only dependency enumeration command
`go list -buildvcs=false -mod=readonly -deps -tags sqlite_omit_load_extension ./cmd/server`
listed 388 packages. The packages referenced by the four residual advisories
were absent from that build's import graph.

The candidate also passed the exact AIGGP pin check at `8ccb4467` and the
current `fd2c11f` charset checks.

## Scope

The evidence covers the MCP Go module and the Linux/amd64 server build with
the stated vulnerability database snapshot. The four module-only findings
remain in the dependency graph for subsequent review.
