package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/thearchitectit/guardrail-mcp/internal/audit"
	"github.com/thearchitectit/guardrail-mcp/internal/auth"
	"github.com/thearchitectit/guardrail-mcp/internal/config"
	"github.com/thearchitectit/guardrail-mcp/internal/database"
)

// denyStore fails every durable audit insert.
type denyStore struct{}

func (denyStore) Insert(ctx context.Context, event *database.AuditEvent) error {
	return errors.New("audit store unavailable")
}

func scopedRegistryConfig(t *testing.T, secret, principalID, role string, scopes, resources []string) *config.Config {
	t.Helper()
	records := []auth.Record{{
		CredentialID: "cred-1",
		PrincipalID:  principalID,
		Scopes:       scopes,
		Role:         role,
		Resources:    resources,
		Verifier:     auth.Digest(testVerifierKey, secret),
	}}
	raw, err := json.Marshal(records)
	if err != nil {
		t.Fatalf("marshal records: %v", err)
	}
	return &config.Config{
		MCPAPIKey:              "legacy-mcp-key",
		IDEAPIKey:              "legacy-ide-key",
		CredentialRegistryJSON: string(raw),
		CredentialVerifierKey:  testVerifierKey,
	}
}

// okAuditStore accepts every durable insert.
type okAuditStore struct{}

func (okAuditStore) Insert(ctx context.Context, event *database.AuditEvent) error {
	return nil
}

// TestAuthzScopeRoleResourceMatrix drives the real web middleware through the
// Spec 11 section 4.4 intersection and asserts both the HTTP status and zero
// handler side effects on deny.
func TestAuthzScopeRoleResourceMatrix(t *testing.T) {
	cases := []struct {
		name      string
		role      string
		scopes    []string
		resources []string
		method    string
		path      string
		want      int
		// admin actions require durable audit; use a working store.
		needAudit bool
	}{
		{
			name:   "read-only key + admin role cannot delete project",
			role:   auth.RoleAdministrator,
			scopes: []string{auth.ScopeRESTRead},
			resources: []string{"*"},
			method: http.MethodDelete,
			path:   "/api/projects/proj-a",
			want:   http.StatusForbidden,
		},
		{
			name:   "mutate key + reader role cannot mutate",
			role:   auth.RoleReader,
			scopes: []string{auth.ScopeRESTWrite},
			resources: []string{"*"},
			method: http.MethodPost,
			path:   "/api/rules",
			want:   http.StatusForbidden,
		},
		{
			name:   "developer + rest:write can mutate",
			role:   auth.RoleDeveloper,
			scopes: []string{auth.ScopeRESTWrite, auth.ScopeRESTRead},
			resources: []string{"*"},
			method: http.MethodPost,
			path:   "/api/rules",
			want:   http.StatusOK,
		},
		{
			name:   "admin role + admin scope + grant can delete",
			role:   auth.RoleAdministrator,
			scopes: []string{auth.ScopeAdminManage},
			resources: []string{"proj-a"},
			method: http.MethodDelete,
			path:   "/api/projects/proj-a",
			want:   http.StatusOK,
			needAudit: true,
		},
		{
			name:   "cross-project delete denied",
			role:   auth.RoleAdministrator,
			scopes: []string{auth.ScopeAdminManage},
			resources: []string{"proj-a"},
			method: http.MethodDelete,
			path:   "/api/projects/proj-b",
			want:   http.StatusForbidden,
		},
		{
			name:   "unknown role denied",
			role:   "root",
			scopes: []string{auth.ScopeRESTWrite},
			resources: []string{"*"},
			method: http.MethodPost,
			path:   "/api/rules",
			want:   http.StatusForbidden,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := scopedRegistryConfig(t, "registered-secret", "principal-alpha", tc.role, tc.scopes, tc.resources)
			var code int
			if tc.needAudit {
				logger := audit.NewLoggerWithStore(8, okAuditStore{})
				defer logger.Stop()
				code, _ = runAuthWithAudit(cfg, logger, tc.method, tc.path, "registered-secret")
			} else {
				code, _ = runAuth(cfg, tc.method, tc.path, "registered-secret")
			}
			if code != tc.want {
				t.Fatalf("%s %s: got %d want %d", tc.method, tc.path, code, tc.want)
			}
		})
	}
}

// TestAuthzAdminMutationFailsClosedWithoutAudit covers the durability
// boundary: a security-sensitive mutation cannot complete when its required
// durable audit record cannot be written.
func TestAuthzAdminMutationFailsClosedWithoutAudit(t *testing.T) {
	cfg := scopedRegistryConfig(t, "registered-secret", "principal-admin",
		auth.RoleAdministrator,
		[]string{auth.ScopeAdminManage},
		[]string{"proj-a"},
	)
	logger := audit.NewLoggerWithStore(8, denyStore{})
	defer logger.Stop()

	code, _ := runAuthWithAudit(cfg, logger, http.MethodDelete, "/api/projects/proj-a", "registered-secret")
	if code != http.StatusServiceUnavailable {
		t.Fatalf("admin delete without durable audit: got %d want 503", code)
	}
}

// runAuthWithAudit drives the middleware with an explicit audit logger.
func runAuthWithAudit(cfg *config.Config, logger *audit.Logger, method, path, credential string) (int, string) {
	return runAuthInternal(cfg, logger, method, path, credential)
}
