package web

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/thearchitectit/guardrail-mcp/internal/auth"
	"github.com/thearchitectit/guardrail-mcp/internal/config"
)

// TestRegistryConfiguredInvalidDeniesAllProtectedTraffic covers S-A0 / R16-03
// on the real web middleware: when a registry is configured but fails to load
// (malformed JSON, unreadable file, empty records, invalid verifier key),
// protected traffic is denied for every credential — including the legacy key.
// There is no silent legacy fallback.
func TestRegistryConfiguredInvalidDeniesAllProtectedTraffic(t *testing.T) {
	unreadable := filepath.Join(t.TempDir(), "missing-registry.json")
	emptyFile := filepath.Join(t.TempDir(), "empty-registry.json")
	if err := os.WriteFile(emptyFile, []byte("[]"), 0o600); err != nil {
		t.Fatalf("write empty registry: %v", err)
	}
	malformedFile := filepath.Join(t.TempDir(), "bad-registry.json")
	if err := os.WriteFile(malformedFile, []byte("{not-json"), 0o600); err != nil {
		t.Fatalf("write malformed registry: %v", err)
	}

	validRecord := `[{"credential_id":"cred-1","principal_id":"p1","verifier":"` +
		auth.Digest(testVerifierKey, "registered-secret") + `"}]`

	base := func() *config.Config {
		return &config.Config{
			MCPAPIKey: "legacy-mcp-key",
			IDEAPIKey: "legacy-ide-key",
		}
	}

	cases := []struct {
		name string
		mut  func(*config.Config)
	}{
		{
			name: "malformed inline JSON",
			mut: func(c *config.Config) {
				c.CredentialRegistryJSON = `[{"credential_id":"x"}`
				c.CredentialVerifierKey = testVerifierKey
			},
		},
		{
			name: "malformed file JSON",
			mut: func(c *config.Config) {
				c.CredentialRegistryFile = malformedFile
				c.CredentialVerifierKey = testVerifierKey
			},
		},
		{
			name: "unreadable file",
			mut: func(c *config.Config) {
				c.CredentialRegistryFile = unreadable
				c.CredentialVerifierKey = testVerifierKey
			},
		},
		{
			name: "empty inline records",
			mut: func(c *config.Config) {
				c.CredentialRegistryJSON = "[]"
				c.CredentialVerifierKey = testVerifierKey
			},
		},
		{
			name: "empty file records",
			mut: func(c *config.Config) {
				c.CredentialRegistryFile = emptyFile
				c.CredentialVerifierKey = testVerifierKey
			},
		},
		{
			name: "invalid verifier key",
			mut: func(c *config.Config) {
				c.CredentialRegistryJSON = validRecord
				c.CredentialVerifierKey = "short"
			},
		},
		{
			name: "empty verifier key with records",
			mut: func(c *config.Config) {
				c.CredentialRegistryJSON = validRecord
				c.CredentialVerifierKey = ""
			},
		},
	}

	credentials := []string{
		"legacy-mcp-key",
		"legacy-ide-key",
		"registered-secret",
		"anything-else",
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := base()
			tc.mut(cfg)

			for _, cred := range credentials {
				for _, path := range []string{"/api/rules", "/api/ingest", "/api/updates/check"} {
					code, _ := runAuth(cfg, http.MethodPost, path, cred)
					if code != http.StatusServiceUnavailable {
						t.Fatalf("configured-invalid registry: POST %s with %q got %d, want 503",
							path, cred, code)
					}
				}
			}

			// Public/read-only surface stays available so health checks work.
			code, _ := runAuth(cfg, http.MethodGet, "/api/stats", "")
			if code != http.StatusOK {
				t.Fatalf("public read path got %d, want 200", code)
			}
		})
	}
}

// TestRegistryConfiguredValidStillAuthenticates is the positive control for
// the fail-closed cases above.
func TestRegistryConfiguredValidStillAuthenticates(t *testing.T) {
	records := []auth.Record{{
		CredentialID: "cred-1",
		PrincipalID:  "principal-alpha",
		Scopes:       []string{auth.ScopeRESTRead, auth.ScopeRESTWrite},
		Role:         auth.RoleDeveloper,
		Resources:    []string{"*"},
		Verifier:     auth.Digest(testVerifierKey, "registered-secret"),
	}}
	raw, _ := json.Marshal(records)
	cfg := &config.Config{
		MCPAPIKey:              "legacy-mcp-key",
		IDEAPIKey:              "legacy-ide-key",
		CredentialRegistryJSON: string(raw),
		CredentialVerifierKey:  testVerifierKey,
	}

	code, principal := runAuth(cfg, http.MethodPost, "/api/rules", "registered-secret")
	if code != http.StatusOK {
		t.Fatalf("valid registry: got %d want 200", code)
	}
	if principal != "principal-alpha" {
		t.Fatalf("principal = %q, want principal-alpha", principal)
	}

	// Legacy key is confined and cannot mutate once a registry is present.
	code, _ = runAuth(cfg, http.MethodPost, "/api/rules", "legacy-mcp-key")
	if code != http.StatusForbidden {
		t.Fatalf("legacy key on privileged action: got %d want 403", code)
	}
}
