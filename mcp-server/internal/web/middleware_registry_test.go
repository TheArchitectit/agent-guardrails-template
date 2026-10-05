package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/thearchitectit/guardrail-mcp/internal/audit"
	"github.com/thearchitectit/guardrail-mcp/internal/auth"
	"github.com/thearchitectit/guardrail-mcp/internal/config"
)

const testVerifierKey = "web-test-verifier-key-0123456789"

// registryConfig builds a config whose credential registry contains a single
// record for the given secret, mapping it to an explicit principal with the
// scopes and role needed for REST mutations (Spec 16 section 4 catalog).
func registryConfig(t *testing.T, secret, principalID string) *config.Config {
	t.Helper()
	records := []auth.Record{{
		CredentialID: "cred-1",
		PrincipalID:  principalID,
		Scopes:       []string{auth.ScopeRESTRead, auth.ScopeRESTWrite, auth.ScopeMCPRead, auth.ScopeMCPMutate},
		Role:         auth.RoleDeveloper,
		Resources:    []string{"*"},
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

// runAuth drives the middleware with a bearer credential and returns the status
// plus the principal the handler observed.
func runAuth(cfg *config.Config, method, path, credential string) (int, string) {
	return runAuthInternal(cfg, nil, method, path, credential)
}

// runAuthInternal is runAuth with an optional audit logger.
func runAuthInternal(cfg *config.Config, logger *audit.Logger, method, path, credential string) (int, string) {
	e := echo.New()
	seenPrincipal := ""
	handler := APIKeyAuth(cfg, logger)(func(c echo.Context) error {
		if p, ok := c.Get("principal_id").(string); ok {
			seenPrincipal = p
		}
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(method, path, nil)
	if credential != "" {
		req.Header.Set("Authorization", "Bearer "+credential)
	}
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// Echo's c.Path() is the route pattern; set it so middleware path checks
	// behave as they do under real routing.
	c.SetPath(path)

	code := rec.Code
	if err := handler(c); err != nil {
		code = echoStatus(err)
	}
	return code, seenPrincipal
}

// TestAPIKeyAuth_RegisteredCredentialResolvesPrincipal covers the positive
// path: a registered credential yields the registry principal, which is not
// derived from any log hash.
func TestAPIKeyAuth_RegisteredCredentialResolvesPrincipal(t *testing.T) {
	cfg := registryConfig(t, "registered-secret", "principal-alpha")

	code, principal := runAuth(cfg, http.MethodPost, "/api/rules", "registered-secret")
	if code != http.StatusOK {
		t.Fatalf("registered credential: got status %d, want 200", code)
	}
	if principal != "principal-alpha" {
		t.Fatalf("principal = %q, want principal-alpha", principal)
	}
}

// TestAPIKeyAuth_UnregisteredCredentialDeniedPrivilegedAction covers the
// fail-closed requirement: once a registry is configured, an unregistered
// legacy key is confined to the approved method/path allowlist (Spec 16
// section 4) — no safe-method wildcard and no /ide/ prefix wildcard.
func TestAPIKeyAuth_UnregisteredCredentialDeniedPrivilegedAction(t *testing.T) {
	cfg := registryConfig(t, "registered-secret", "principal-alpha")

	cases := []struct {
		name   string
		method string
		path   string
		key    string
		want   int
	}{
		{"legacy key denied mutating action", http.MethodPost, "/api/rules", "legacy-mcp-key", http.StatusForbidden},
		{"legacy key denied on delete", http.MethodDelete, "/api/rules", "legacy-ide-key", http.StatusForbidden},
		{"legacy key allowed named read", http.MethodGet, "/api/stats", "legacy-mcp-key", http.StatusOK},
		{"legacy key denied unlisted GET", http.MethodGet, "/api/secrets", "legacy-mcp-key", http.StatusForbidden},
		{"legacy key denied ingest mutation", http.MethodPost, "/api/ingest", "legacy-mcp-key", http.StatusForbidden},
		{"legacy key denied ingest sync", http.MethodPost, "/api/ingest/sync", "legacy-mcp-key", http.StatusForbidden},
		{"legacy mcp key allowed updates check", http.MethodPost, "/api/updates/check", "legacy-mcp-key", http.StatusOK},
		{"legacy ide key allowed named ide validate", http.MethodPost, "/ide/validate/file", "legacy-ide-key", http.StatusOK},
		{"legacy mcp key denied ide validate", http.MethodPost, "/ide/validate/file", "legacy-mcp-key", http.StatusForbidden},
		{"legacy key denied unlisted ide action", http.MethodPost, "/ide/validate/other", "legacy-ide-key", http.StatusForbidden},
		{"legacy key denied HEAD on unlisted path", http.MethodHead, "/api/admin", "legacy-mcp-key", http.StatusForbidden},
		{"registered credential allowed mutating action", http.MethodPost, "/api/rules", "registered-secret", http.StatusOK},
		{"unknown credential denied", http.MethodPost, "/api/rules", "nobody", http.StatusUnauthorized},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, _ := runAuth(cfg, tc.method, tc.path, tc.key)
			if code != tc.want {
				t.Fatalf("%s %s with %q: got status %d, want %d", tc.method, tc.path, tc.key, code, tc.want)
			}
		})
	}
}

// TestAPIKeyAuth_LegacyKeyUnchangedWithoutRegistry guards the "nothing silently
// breaks" requirement: with no registry configured the legacy keys keep their
// full current surface.
func TestAPIKeyAuth_LegacyKeyUnchangedWithoutRegistry(t *testing.T) {
	cfg := &config.Config{MCPAPIKey: "legacy-mcp-key", IDEAPIKey: "legacy-ide-key"}

	for _, path := range []string{"/api/rules", "/api/ingest", "/api/updates/check"} {
		code, _ := runAuth(cfg, http.MethodPost, path, "legacy-mcp-key")
		if code != http.StatusOK {
			t.Fatalf("legacy key without registry: POST %s got %d, want 200", path, code)
		}
	}
}

// TestAPIKeyAuth_SecretNeverInErrorOrResponse ensures a presented secret is not
// echoed in a denial response.
func TestAPIKeyAuth_SecretNeverInErrorOrResponse(t *testing.T) {
	const secret = "bearer-secret-must-not-leak"
	cfg := registryConfig(t, "registered-secret", "principal-alpha")

	e := echo.New()
	handler := APIKeyAuth(cfg, nil)(func(c echo.Context) error { return c.NoContent(http.StatusOK) })
	req := httptest.NewRequest(http.MethodPost, "/api/rules", nil)
	req.Header.Set("Authorization", "Bearer "+secret)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler(c)
	if err == nil {
		t.Fatal("unknown credential should be denied")
	}
	if body := rec.Body.String(); body != "" && strings.Contains(body, secret) {
		t.Fatalf("response body leaked the secret: %q", body)
	}
	if msg := err.Error(); strings.Contains(msg, secret) {
		t.Fatalf("error leaked the secret: %q", msg)
	}
}
