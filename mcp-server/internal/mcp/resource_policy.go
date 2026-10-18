package mcp

import (
	"context"
	"errors"

	"github.com/thearchitectit/guardrail-mcp/internal/auth"
)

// Declared resource operation-to-permission matrix (Spec 16 R16-01, R16-04).
//
// R16-01 requires a complete tool/resource operation-to-permission matrix and
// default deny for an unmapped resource. R16-04 requires that each resource
// read be authorized "using the same principal decision model as tools". Before
// this table existed the resource handlers produced contents for any caller
// that had already passed the bearer gate: there was no per-resource decision
// and no declared matrix, so a credential that could not call a read tool could
// still read every guardrail:// resource.
//
// Every registered resource URI has exactly one row here. A URI that is not in
// this table is denied (classifyResource returns known=false) and the gate test
// fails if a registered resource has no row, so the matrix cannot silently fall
// behind the registry.
type resourcePolicyRow struct {
	// URI is the exact guardrail:// resource identifier as registered.
	URI string
	// Kind is the privilege class of reading this resource. All current
	// resources are read-class; the field is explicit so a future
	// privileged resource cannot inherit read authority by omission.
	Kind auth.ActionKind
	// Rationale records why the resource has this class.
	Rationale string
}

// resourcePolicy is the single declared source of truth for resource reads.
var resourcePolicy = []resourcePolicyRow{
	{URI: "guardrail://config", Kind: auth.ActionRead,
		Rationale: "Non-secret configuration projection (R16-04); read-class like other read tools."},
	{URI: "guardrail://stats", Kind: auth.ActionRead,
		Rationale: "Usage counters only; no secret material."},
	{URI: "guardrail://agent-guardrails", Kind: auth.ActionRead,
		Rationale: "Static guardrail documentation."},
	{URI: "guardrail://workflows", Kind: auth.ActionRead,
		Rationale: "Static workflow documentation."},
	{URI: "guardrail://standards", Kind: auth.ActionRead,
		Rationale: "Static standards documentation."},
	{URI: "guardrail://four-laws", Kind: auth.ActionRead,
		Rationale: "Static policy documentation."},
	{URI: "guardrail://halt-conditions", Kind: auth.ActionRead,
		Rationale: "Static policy documentation."},
	{URI: "guardrail://pre-work-check", Kind: auth.ActionRead,
		Rationale: "Static checklist documentation."},
	{URI: "guardrail://git-safety", Kind: auth.ActionRead,
		Rationale: "Static policy documentation."},
	{URI: "guardrail://test-prod-separation", Kind: auth.ActionRead,
		Rationale: "Static policy documentation."},
	{URI: "guardrail://advisors", Kind: auth.ActionRead,
		Rationale: "Advisor catalog; no secret material."},
	{URI: "guardrail://policy-packs", Kind: auth.ActionRead,
		Rationale: "Resolved policy-set identity, digests and rule ids (Spec 07 R07-4.4.1); inert data, no secret material."},
	{URI: "guardrail://control-plane", Kind: auth.ActionRead,
		Rationale: "Measured component states and derived verdict (Spec 08 R08-4.2.5); bounded non-secret evidence, no credentials."},
}

// errResourcePermissionDenied is the stable, non-leaking denial returned to a
// caller whose principal may not read the requested resource. It never reveals
// whether the resource exists.
var errResourcePermissionDenied = errors.New("permission denied")

// classifyResource returns the ActionRequest for a declared resource URI, or
// known=false when the URI is not in the declared matrix. Unknown resources
// deny (R16-01 default deny).
func classifyResource(uri string) (auth.ActionRequest, bool) {
	for _, row := range resourcePolicy {
		if row.URI == uri {
			return auth.ActionRequest{
				Name:   uri,
				Kind:   row.Kind,
				Scopes: auth.RequiredScopeForKind(row.Kind, "mcp"),
			}, true
		}
	}
	return auth.ActionRequest{Name: uri, Kind: "unknown"}, false
}

// authorizeResourceRead applies the same scope × role × resource intersection
// the tool path uses before a resource handler produces any contents
// (Spec 16 R16-04). Identity comes only from the request context, never from
// the requested URI.
func (s *MCPServer) authorizeResourceRead(ctx context.Context, uri string) error {
	action, known := classifyResource(uri)
	if !known {
		// Undeclared resource: deny before any read side effect.
		return errResourcePermissionDenied
	}

	caller, ok := callerFromContext(ctx)
	if !ok || caller.PrincipalID == "" {
		// No server-controlled principal: deny. Never fall back to
		// URI-sourced or anonymous authority.
		return errResourcePermissionDenied
	}

	decision := auth.Decide(caller, action)
	// Record the decision with the same audit shape as a tool call. A resource
	// read is not a security-sensitive mutation, so a missing durable audit is
	// not fatal here (required=false).
	_ = s.mcpAuditDecision(ctx, caller, action, decision, false)
	if !decision.Allow {
		return errResourcePermissionDenied
	}
	return nil
}
