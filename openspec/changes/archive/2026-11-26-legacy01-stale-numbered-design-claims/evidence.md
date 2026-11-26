# LEGACY-01 evidence reconciliation

**Reviewed base:** `db0603c927c32b63124aa944c264200b45cd3af2`.
**Scope:** dated status/progress claims in Specs 07, 12, 16, 17, and 23.
This records source and documentation inspection only. No Go tests or CI gates
were run for this change; cited test names and historical run records are not
new execution claims.

Reconciled source documents: [Spec 07](../../../../docs/specs/07-versioned-policy-pack-governance.md),
[Spec 12](../../../../docs/specs/12-web-exposure-boundary.md),
[Spec 16](../../../../docs/specs/16-authentication-authorization-and-evidence-remediation.md),
[Spec 17](../../../../docs/specs/17-deployment-tls-and-secret-boundary.md), and
[Spec 23](../../../../docs/specs/23-live-tool-resource-listing-sequence.md).

| Spec | Reconciled statement | Current source/test evidence and residual |
|---|---|---|
| 07 | The earlier acceptance paragraph said the harness, lock schema, resolver, and runtime integration did not yet exist. It is now explicitly labeled as historical; the opening status remains PARTIAL (`docs/specs/07-versioned-policy-pack-governance.md:3-15,150-156`). | Current notes describe `scripts/check_policy_pack_lock.py`, runtime resolution (`mcp-server/internal/guardrails/policy_pack.go:538,963-974`), and named tests (`mcp-server/internal/guardrails/policy_pack_test.go:284-341`; `scripts/test_check_policy_pack_lock.py`). R07-4.3.3 remote acquisition and owner choices remain open (`docs/specs/07-versioned-policy-pack-governance.md:352-374`); this does not close them. |
| 12 | The older residual said the shared exposure registry remained unimplemented. It now points to the later Sec. 10 closure (`docs/specs/12-web-exposure-boundary.md:318-323`). | `mcp-server/internal/web/exposure.go:73` defines the route table; middleware uses it (`mcp-server/internal/web/middleware.go:52`); tests cover route completeness/default deny/consistency (`mcp-server/internal/web/exposure_test.go:29,65,92,195`). Public-route re-decision and public-profile `/metrics` remain owner decisions (`docs/specs/12-web-exposure-boundary.md:408-414`). |
| 16 | The opening source snapshot `178d076` is identified as the historical 2026-10-04 baseline, not the current inventory (`docs/specs/16-authentication-authorization-and-evidence-remediation.md:3-9`). The 2026-10-18 status table remains the later disposition (`:287-316`). | Current tests include `internal/web/middleware_authz_test.go:58` (`TestAuthzScopeRoleResourceMatrix`), `internal/mcp/streamable_http_authz_test.go:216` (`TestStreamableHTTPToolsCallAuthorizesAndDenies`), plus resource and argument-privacy tests. The document leaves bilateral Spec 15 and conditional Spec 14 open pending real consumers/owner evidence (`:374-376`). |
| 17 | The headline “Proposed; NOT implemented” is replaced with bounded PARTIAL status and the source observations are labeled as the 2026-10-03 baseline (`docs/specs/17-deployment-tls-and-secret-boundary.md:3-12`). The third companion note is linked. | Dated notes contain evidence for selected controls and explicitly retain limits: post-start drift (`docs/specs/17-implementation-notes-2.md:343-345`), webhook-at-rest plus DB/JWT/TLS rotation (`:478-480`), and DB/JWT/TLS staged cutover (`docs/specs/17-implementation-notes-3.md:72-73`). The production acceptance gate at `docs/specs/17-deployment-tls-and-secret-boundary.md:65` remains unclaimed. |
| 23 | The “no live listing test” statement is labeled as the pre-closure 2026-10-10 baseline (`docs/specs/23-live-tool-resource-listing-sequence.md:7-27`). The closure record stays intact. | `mcp-server/internal/mcp/streamable_http_listing_test.go:42` defines `TestStreamableHTTPListingSequence`; Sec. 5 records the historical live run and limits (`docs/specs/23-live-tool-resource-listing-sequence.md:84-112`). This audit did not rerun that Go test. |

Dates and historical observations were retained; edits do not alter the
separate `docs/platform-current-state.md` persistence audit or mark unresolved
requirements accepted.

## Validation

- `scripts/check-doc-links.sh docs/specs` under Docker `node:22-bookworm` —
  **PASS**, 109 internal links across 41 Markdown files.
- `scripts/check-doc-links.sh openspec/changes/legacy01-stale-numbered-design-claims`
  under Docker `node:22-bookworm` — **PASS**, 5 internal links across 6 Markdown
  files. The first package-only attempt was correctly rejected as vacuous before
  source-document links were added.
- Full-tree `scripts/check-doc-links.sh .` — **NOT_RUN to completion**. It was
  started in Docker and interrupted after no output for over three minutes;
  scoped source/package checks are the reported link evidence.
- Pinned OpenSpec CLI `@fission-ai/openspec@2.2.0`, command
  `openspec validate legacy01-stale-numbered-design-claims --strict`, Node
  `v22.23.5` Linux — **PASS**, “Change is valid”.
- Go tests and CI — **NOT_RUN**; this change updates documentation only.
- Scope check — `docs/platform-current-state.md` is unchanged; only the five
  named numbered specs and the active LEGACY-01 package are edited.
