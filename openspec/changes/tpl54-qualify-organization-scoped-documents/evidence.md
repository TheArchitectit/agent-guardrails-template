# TPL-54 source inventory and bounded evidence

## Source revision and scope

Inventory performed in detached worktree `guardrails-template-tpl54` at
`6df8e20369a74fffbb8f1bd56d768ab27945348c` (DIST-01 merge). This records the
repository-owned document implementation and its current access boundaries.
No connector, bank, or external service data was used.

The shared document backend in this revision is the repository-owned
PostgreSQL `DocumentStore`. The checked-in schema has a global `documents`
table and global unique slug, with content/category/path/metadata fields but no
organization, tenant, source-grant, or permission columns
(`mcp-server/internal/database/migrations/001_create_tables.up.sql:4-15`).
`models.Document` adds source, content hash, file path, and orphaned state, but
no organization or permission fields (`mcp-server/internal/models/document.go:11-27`).
The list/get/search handlers serialize `models.Document` objects rather than
an explicit bounded metadata projection. The model's JSON surface includes
content, slug, title, category, path, version, metadata, source, content hash,
file path, orphaned state, and timestamps; `SearchVector` alone is omitted
(`mcp-server/internal/models/document.go:11-27`). The current document SELECTs
do not load source/content hash/file path/orphaned columns, so their returned
values default rather than conveying those attributes; this does not create an
organization permission boundary.

## Registered shared-document entry points

| Registered operation | Handler and storage path | Current exposure / trusted scope |
|---|---|---|
| `GET /api/documents` | `listDocuments` calls `DocumentStore.List(category, limit, offset)` and `Count(category)`; rows and pagination are returned (`internal/web/server.go:319`, `internal/web/handlers_docs.go:16-47`, `internal/database/documents.go:71-117,256-271`). | Public in the exposure registry. `APIKeyAuthWithRegistry` returns to the handler before resolving a caller on public routes (`internal/web/exposure.go:124-129`, `internal/web/middleware.go:57-58`). No caller scope, organization, or source permission filters the SQL. |
| `GET /api/documents/search?q=...` | `searchDocuments` calls `DocumentStore.Search`; Postgres full-text ranking is global over `search_vector` (`internal/web/server.go:320`, `internal/web/handlers_docs.go:102-124`, `internal/database/documents.go:120-167`). | Public; query and returned documents are not organization- or source-filtered. |
| `GET /api/documents/:id` | `getDocument` parses the path as UUID and calls `GetByID` (`internal/web/server.go:321`, `internal/web/handlers_docs.go:50-63`, `internal/database/documents.go:31-48`). | Public by the `/api/documents/` prefix rule. A slug in the path is rejected as `invalid id format`; the REST route is not a slug lookup. The underlying UUID query has no organization predicate. |
| `GET /api/stats` | `getStats` calls global `DocumentStore.Count` and returns `documents_count` (`internal/web/server.go:351`, `internal/web/handlers_system.go:21-72`). | Public in the exposure registry; aggregate metadata has no organization boundary (`internal/web/exposure.go:134-136`). |
| `GET /ide/quick-reference` | `getQuickReference` tries the `quick-reference` slug, then `quick-reference-card`, then globally searches “quick reference”; it returns content, title, slug, and category (`internal/web/server.go:381`, `internal/web/handlers_ide.go:241-268`, `internal/database/documents.go:51-68,120-167`). | Not in the public-read document rules, so the middleware requires a credential. The handler uses the database context and logs `callerFromContext`, but does not filter the lookup/search by that caller, organization, or source permission. |
| `PUT /api/documents/:id` | `updateDocument` binds a `Document`, secret-scans content, then calls `DocumentStore.Update`; it invalidates caches and audits the slug (`internal/web/server.go:322`, `internal/web/handlers_docs.go:65-100`, `internal/database/documents.go:196-225`). | Not included in the GET-only public rules. With registry authorization, generic REST classification requires `rest:write` and derives the path resource ID; this is a document identifier grant, not an organization/source permission. In no-registry legacy migration mode the shared authorization intersection is skipped (`internal/auth/authz.go:234-259,603-654`, `internal/web/middleware.go:193-194`). |
| `POST /api/ingest`, `/api/ingest/upload`, `/api/ingest/sync` | `triggerIngest` and `syncRepo` start repository syncs; `uploadFiles` passes uploaded files to `SyncFromUpload` (`internal/web/server.go:352,359-360`, `internal/web/handlers_system.go:77-223`). The ingest service matches by global slug and invokes document `Create`/`Update` (`internal/ingest/service.go:45-109,160-233`). | Mutation routes use generic REST write classification when registry enforcement is active. The background ingest context and service carry no trusted organization or source permission. Upload/repository source labels describe origin type, not entitlement. |
| `DELETE /api/ingest/orphans/:id` | `deleteOrphan` loads by UUID, checks `Orphaned`, deletes, and invalidates by slug (`internal/web/server.go:363`, `internal/web/handlers_system.go:271-303`, `internal/database/documents.go:31-48,228-253`). | Generic REST mutation classification when registry enforcement is active, with no org/source filter. The current `GetByID` SELECT does not scan the `Orphaned` field, so the handler's orphan precondition appears false for rows returned by this getter; this path was not exercised. |
| `GET /api/ingest/orphans` and `GET /api/ingest/status` | Current handlers return an empty orphan list and a placeholder status; they do not query documents (`internal/web/server.go:361-362`, `internal/web/handlers_system.go:230-269`). | Read-only status surfaces, not current document retrieval paths. The orphan list is not an inventory of stored documents. |

`DocumentStore.GetBySlug` is used internally by ingest upserts and quick
reference, not by a registered REST slug route. `Create` is used by ingest;
`Update` is used by the PUT handler and ingest; `Delete` is called only by the
orphan deletion handler in the current web server. The SQL predicates for
`GetByID`, `GetBySlug`, list, search, count, update, and delete are global
(`mcp-server/internal/database/documents.go:31-68,71-167,196-271`). The schema's
global unique slug also prevents identical slugs from coexisting by
organization (`mcp-server/internal/database/migrations/001_create_tables.up.sql:4-15`).

## Cache, MCP, and other document-shaped surfaces

`internal/cache/redis.go:133-147` declares `KeyDocument` and `TTLDocument`;
`InvalidateOnDocumentChange` deletes the slug key and search-key pattern
(`internal/cache/redis.go:185-205`). A repository search found no document
cache get/set call sites; current list/search and quick-reference paths query
the database directly. Therefore there is no active document-result cache hit
path demonstrated by this source revision, while invalidation hooks remain.

`internal/mcp/tools_docs.go:15-210` contains handlers that read repository
files under `docs/standards`, `docs/workflows`, and other `docs` directories.
Those functions have no references from the current MCP tool registration and
do not use `DocumentStore`; they are not the shared database document API.
Separately, `internal/mcp/resource_registration.go:40-90` registers static
`guardrail://` documentation resources. They read repository files, not the
shared document table, and are outside the organization-scoped database
contract.

## Trusted identity and source authority gap

`auth.Caller` carries principal ID, credential ID, scopes, role, resource IDs,
and legacy class (`mcp-server/internal/auth/authz.go:88-99`). Credential records
and their resolved principals carry those same generic fields but no
organization or source-grant claim (`mcp-server/internal/auth/registry.go:64-105`).
The REST action classifier applies generic read/write scopes and an opaque path
resource; it has no document organization/source rule (`mcp-server/internal/auth/authz.go:234-259,603-674`). The public GET document rules bypass that
classifier entirely. Thus an authenticated principal can be attributed for
non-public calls, but current source does not bind document operations to a
trusted organization or current upstream permission.

The supported deployment profiles are `local`, `tailnet`, and `public`
(`mcp-server/internal/config/profile.go:10-29`). `ExposureForConfig` has a
public-profile override for metrics, docs, and OpenAPI paths; it does not alter
the document rules (`mcp-server/internal/web/exposure.go:229-239`). The public
document reads therefore remain public in every declared profile. No approved
permission source or document-specific shared-access profile was found in the
repository. The current document service is local/repository-owned; no
external retrieval interface or external enforcement owner is registered.
This inventory does not choose an unapproved source or qualify any of these
profiles for organization-shared documents.

## Existing local fixture results

The bounded local checks passed in `golang:1.26.11-bookworm`:

- `go test ./internal/models -run 'TestDocument_Validate|TestIsValidCategory|TestDocumentCategory_Constants' -count=1` passed. This proves document shape/category validation only; it does not cover access control.
- `go test ./internal/web -run 'TestExposureRegistryCoversEveryRoute|TestExposureRegistryIsTheSingleSkipSource' -count=1` passed. These route-table fixtures establish current exposure classification, including public GET document surfaces; they do not execute document handlers or storage queries.

No cross-organization duplicate-ID/slug, source revocation, cache-hit, or
caller-paired document fixture exists in the inspected test set. No storage,
handler, profile, or permission behavior was changed in this evidence increment.

## Remaining acceptance

Task 1.1 remains open because the repository does not provide the approved
organization/source permission authority or a document profile to select.
Tasks 1.2 and 1.3 remain open: there is no trusted organization binding,
source-permission enforcement, paired-identity/revocation/cache acceptance,
or external retrieval ownership evidence. Task 2.1 remains open pending that
scope decision and completion of the required acceptance work.
