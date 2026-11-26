# Future-state integration audit

**Reviewed source** `db0603c927c32b63124aa944c264200b45cd3af2`

The template can serve as a bounded check service in a larger tool platform. The adopting access and workflow services resolve trusted organization context, own independent action approval and enforce approved effects in the target system. A check receipt informs that decision.

| Boundary | Observed in the reviewed tree | Acceptance work |
|---|---|---|
| Transport and operator identity | The common tool path intersects scope, role and resource. The Go server supports MCP and REST. | Verify required calls through the adopted access and workflow services. Match the receiver, identity, profile and exact outcomes. See RUN-03 and OPS-02 in the gap register. |
| Optional MCP and REST surfaces | Optional dispatch and separately mounted REST routes need equivalent evidence at their registered entry points. | Enforce the shared authorization and redacted decision path before effects on every enabled surface. Keep optional features disabled in the supported deployment profile pending acceptance. [TPL-49](../changes/tpl49-enforce-optional-surface-authorization/proposal.md) |
| Session and deployment modes | The session map resides in the serving process. The Kubernetes example requests two replicas. | Qualify a supported profile and verify restart, reconnect and revocation. Coordinate rendered deployment with DEP-01 and recovery with OPS-03 and OPS-04. [TPL-50](../changes/tpl50-qualify-mcp-session-topology/proposal.md) |
| Document permissions | Generic lookups and search select by document identity, category or query. | Establish organization and source authority for every adopted shared path. Keep permission-sensitive external retrieval within its owning service contract. [TPL-51](../changes/tpl51-qualify-organization-scoped-documents/proposal.md) |
| Runtime inputs and audit | Startup binds profile controls and attaches a database audit store. | Verify used secret source, policy activation, event coverage, failure behavior, profile integrity and independently restorable state. See INV-02, TPL-45, DEP-01 and OPS-04. |
| Release evidence | The repository has gate, classifier, and evidence packages with separate acceptance work. | Bind artifact and policy identities for the adopting release. Record positive and fail-closed outcomes through the protected promotion path. See TPL-36 and RUN-06. |

The live Windows pilot establishes local readiness for its tested profile. Application admission, source ACLs, identity-provider and workflow bindings, target permissions, retained evidence, capacity, failover and production recovery are integration acceptance criteria. Test each adopting client independently so a successful transport handshake carries its actual coverage.

- [MCP handler and session state](../../mcp-server/internal/mcp/server.go)
- [Application startup](../../mcp-server/cmd/server/main.go)
- [Document store](../../mcp-server/internal/database/documents.go)
- [Kubernetes example](../../mcp-server/deploy/k8s-deployment.yaml)
- [Copilot integration boundary](../../integrations/copilot/README.md)
- [Gap register](2026-11-25-template-gap-register.md)
- [Verification queue](2026-11-25-template-verification-queue.md)
