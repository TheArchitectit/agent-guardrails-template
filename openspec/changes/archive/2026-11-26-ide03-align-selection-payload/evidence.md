# IDE-03 evidence

## Server

The targeted Go resolver and HTTP handler tests passed in Docker Desktop's
Linux engine. The request cases cover source preservation, missing and empty
payloads, explicit null and invalid types, and dual-field conflicts.
I repeated the selection-targeted web package tests in a Linux-local Go 1.26.11
container copy; `go test ./internal/web -run 'Test.*Selection' -count=1` passed.

The registered Echo route was exercised against temporary PostgreSQL data
using a synthetic match rule. Canonical selection, legacy code and equal
dual-field requests returned HTTP 200 with the expected rule match. Removing
legacy fallback in the disposable source copy caused the legacy route request
to return HTTP 400. The result returned to HTTP 200 after restore. A later
replay after removing redundant equivalent fixtures could not reach the
disposable database; the route results above remain from the successful runs.

## Clients

All four builders now use the selection field. The regression test invokes
the compiled VS Code callback with a stubbed editor module and checks the POST
route and serialized body. The regression also checks the three non-VS Code
builder sources. Subsequent runtime request capture and route replay for all
four clients is recorded below.

In a Linux-local Node 22.20 container copy, `npm ci` completed from the
lockfile. `npm test` passed, including TypeScript compilation, ESLint, and
the initialization, configuration, and IDE selection regression scripts.
The install reported 14 dependency audit findings (4 moderate, 10 high); no
dependency changes were made. The template regression scan passed. Client
runtime HTTP results are documented below, including the limits of the
JetBrains and Neovim transport shims.

## Client request acceptance

I ran each request builder against a local HTTP recorder using the benign
source `  selection_probe` and language `go`. The compiled VS Code callback
ran under Node and made a real HTTP POST. Vim ran headlessly from a temporary
LF-normalized copy of its autoload file; Neovim ran its actual validation
module with a minimal `plenary.curl` adapter that made the HTTP POST. Both
editors emitted `POST /ide/validate/selection`. JetBrains compiled and ran the
actual `GuardrailService.kt` method in a disposable Gradle 8.10.2/JDK 21
project with Kotlin 1.9.22, Gson 2.10.1, and OkHttp 4.12.0; minimal
test-only IntelliJ component, annotation, and logger stubs replaced the IDE
runtime. It also emitted `POST /ide/validate/selection`. The captured request
bodies from all four clients contained the exact source in the `selection`
field and `language: go`.

I added optional captured-body replay to
`TestValidateSelectionRouteAgainstDisposablePostgres`. When enabled, it
requires exactly one captured JSON file for each of the four clients and
rejects missing or extra client JSON files. It posts each captured body
unchanged through the registered Echo route and asserts HTTP 200 plus the
expected synthetic rule match. On a fresh `ide_selection_test`
PostgreSQL database in a tmpfs-backed disposable container and isolated
network, all four captured requests passed, as did the route's canonical,
legacy, and equal-dual-field cases. Command:
`go test ./internal/web -run '^TestValidateSelectionRouteAgainstDisposablePostgres$' -count=1 -v`.
Configured replay directories missing a required client or containing an
extra `.json` client were also rejected with the corresponding diagnostic.
The replay used a unique scratch directory in `pr13-go-canonical-cap` with its
module cache and a separate Go build cache. The scratch route copy, database
container, and isolated network were removed afterward; no existing container
or volume was used or changed.

This verifies the actual JetBrains service request builder with test-only
IntelliJ stubs, the Vim and Neovim builders with their noted transport
adapters, and the compiled VS Code callback against the registered route
fixture. It does not exercise the full JetBrains IDE runtime or Neovim's real
`plenary.curl` implementation.
