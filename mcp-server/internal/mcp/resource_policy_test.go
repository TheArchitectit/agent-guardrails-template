package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/thearchitectit/guardrail-mcp/internal/auth"
)

// Spec 16 R16-01 / R16-04 — declared resource permission matrix and
// per-resource authorization over the real StreamableHTTP endpoint.
//
// R16-01 requires an exact operation-to-permission matrix with default deny for
// an unmapped resource. R16-04 requires each resource read to be authorized by
// the same principal decision model as a tool call. These tests drive the real
// endpoint (httptest + requireBearerWithRegistry), not the handlers directly.

// resourcePolicyRecords returns one credential that MAY read resources and one
// that may not, to prove the decision is applied per principal rather than
// granted by the bearer gate alone.
func resourcePolicyRecords() []auth.Record {
	return []auth.Record{
		{
			CredentialID: "res-reader-ok",
			PrincipalID:  "principal-res-reader",
			Scopes:       []string{auth.ScopeMCPRead},
			Role:         auth.RoleReader,
			Resources:    []string{"*"},
			Verifier:     auth.Digest(fullPathVerifierKey, "res-reader-secret"),
		},
		{
			// Authenticated, but holds only a REST scope: the MCP resource
			// read is outside the credential's scope, so it must be denied.
			CredentialID: "res-rest-only",
			PrincipalID:  "principal-res-rest-only",
			Scopes:       []string{auth.ScopeRESTRead},
			Role:         auth.RoleReader,
			Resources:    []string{"*"},
			Verifier:     auth.Digest(fullPathVerifierKey, "res-rest-only-secret"),
		},
	}
}

// TestResourcePolicyMatrixCoversEveryRegisteredResource proves the declared
// matrix is non-vacuous and cannot fall behind the registry: every resource the
// server registers has exactly one row, and every row names a registered
// resource. An unmapped URI denies.
func TestResourcePolicyMatrixCoversEveryRegisteredResource(t *testing.T) {
	s := &MCPServer{}
	registered := map[string]bool{}
	for _, spec := range s.resourceSpecs() {
		registered[spec.uri] = true
	}
	if len(registered) == 0 {
		t.Fatal("no resources registered; the coverage assertion would be vacuous")
	}

	declared := map[string]bool{}
	for _, row := range resourcePolicy {
		if row.URI == "" {
			t.Fatal("resource policy row with empty URI")
		}
		if declared[row.URI] {
			t.Fatalf("duplicate resource policy row for %s", row.URI)
		}
		declared[row.URI] = true
		if row.Rationale == "" {
			t.Fatalf("resource policy row %s has no rationale", row.URI)
		}
		if _, ok := classifyResource(row.URI); !ok {
			t.Fatalf("declared resource %s is not classifiable", row.URI)
		}
	}

	for uri := range registered {
		if !declared[uri] {
			t.Fatalf("registered resource %s has no declared permission row", uri)
		}
	}
	for uri := range declared {
		if !registered[uri] {
			t.Fatalf("declared resource %s is not registered by the server", uri)
		}
	}

	// Default deny: an undeclared resource is not classifiable.
	if _, ok := classifyResource("guardrail://not-a-real-resource"); ok {
		t.Fatal("undeclared resource must not be classifiable (default deny)")
	}
}

// TestStreamableHTTPResourceReadAuthorization proves allow AND deny on the real
// endpoint: a credential whose scope covers the read succeeds, and an
// authenticated credential whose scope does not is refused before any contents
// are produced.
func TestStreamableHTTPResourceReadAuthorization(t *testing.T) {
	cfg := secretConfig()
	cfg.MCPAPIKey = fullPathLegacyKey
	_, ts := newFullPathServer(t, cfg, resourcePolicyRecords())

	read := map[string]interface{}{"uri": "guardrail://config"}

	// Allowed: mcp:read scope + reader role.
	code, env := rpcPost(t, ts, "res-reader-secret", "resources/read", read)
	if code != http.StatusOK {
		t.Fatalf("authorized read status = %d, want 200", code)
	}
	if len(env.Result) == 0 {
		t.Fatalf("authorized read returned no result (error=%s)", string(env.Error))
	}

	// Denied: authenticated, but the credential holds no MCP read scope.
	code, env = rpcPost(t, ts, "res-rest-only-secret", "resources/read", read)
	if len(env.Result) > 0 {
		var rr struct {
			Contents []json.RawMessage `json:"contents"`
		}
		if err := json.Unmarshal(env.Result, &rr); err == nil && len(rr.Contents) > 0 {
			t.Fatalf("denied read still produced contents (status=%d): %s", code, string(env.Result))
		}
	}
	// The denial must not leak whether the resource exists.
	if got := string(env.Error); strings.Contains(got, "guardrail://config") || strings.Contains(got, "schema_version") {
		t.Fatalf("denial message leaks resource detail: %s", got)
	}
}

// TestAuthorizeResourceReadDeniesWithoutPrincipal proves the decision boundary
// denies when no server-controlled principal is present, rather than falling
// back to URI-sourced authority, and allows a principal that may read.
func TestAuthorizeResourceReadDeniesWithoutPrincipal(t *testing.T) {
	s := &MCPServer{}
	if err := s.authorizeResourceRead(context.Background(), "guardrail://config"); err == nil {
		t.Fatal("resource read with no principal must be denied")
	}
	if err := s.authorizeResourceRead(context.Background(), "guardrail://undeclared"); err == nil {
		t.Fatal("undeclared resource must be denied")
	}
	// A principal that may read is allowed (non-vacuous positive control).
	if err := s.authorizeResourceRead(authzTestContext(), "guardrail://config"); err != nil {
		t.Fatalf("authorized resource read denied: %v", err)
	}
}
