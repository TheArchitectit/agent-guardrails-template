# IDE-03 evidence

## Server

The targeted Go resolver and HTTP handler tests passed in Docker Desktop's
Linux engine. The request cases cover source preservation, missing and empty
payloads, explicit null and invalid types, and dual-field conflicts.

The registered Echo route was exercised against temporary PostgreSQL data
using a synthetic match rule. Canonical selection, legacy code and equal
dual-field requests returned HTTP 200 with the expected rule match. Removing
legacy fallback in the disposable source copy caused the legacy route request
to return HTTP 400. The result returned to HTTP 200 after restore. The last
replay after removing redundant equivalent fixtures could not reach the
disposable database. The three preceding distinct route cases had passed.

## Clients

All four builders now use the selection field. The regression test invokes
the compiled VS Code callback with a stubbed editor module and checks the POST
route and serialized body. The JetBrains, Vim and Neovim checks inspect the
source builders. Runtime client requests from those three remain open.

VS Code TypeScript compilation and the Node regression script passed with
dependencies from the same lockfile. The fresh locked npm install did not
finish, and the full pretest lint run did not complete. The template
regression scan passed. The only changed generated VS Code files are the
client JavaScript and source map.
