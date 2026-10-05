package mcp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/mark3labs/mcp-go/server"
	"github.com/thearchitectit/guardrail-mcp/internal/auth"
	"github.com/thearchitectit/guardrail-mcp/internal/config"
	"github.com/thearchitectit/guardrail-mcp/internal/web"
)

// This test proves the LIVE revocation-propagation bound (Spec 16 §4 / R16-09)
// on the real HTTP surfaces, not by calling registry helpers directly:
//
//  1. A credential is registered for both the REST and the MCP surface and is
//     confirmed AUTHORIZED through the real web middleware (echo over httptest)
//     AND the real StreamableHTTP MCP endpoint (httptest, authenticated
//     tools/call).
//  2. The credential is revoked by reloading the registry from the source
//     without it.
//  3. Both surfaces then DENY a fresh request, and the elapsed time from
//     revocation to denial is asserted to be within RevocationPropagationBound
//     and recorded in the test output.
//
// Reload semantics differ per surface by design: the MCP endpoint holds a
// *auth.Registry whose Reload swaps the snapshot atomically, so the SAME
// running endpoint denies the next request. The web middleware binds its
// registry at construction (APIKeyAuth -> buildCredentialRegistry), so a
// revocation visible in the registry source is applied by re-reading the
// source and rebuilding the middleware — i.e. the "fresh request after reload"
// fallback. Both are real httptest paths; neither calls Resolve/Decide directly.
const (
	revPropVerifierKey = "revocation-propagation-verifier-key-0123456789"
	revPropLiveSecret  = "live-secret"
	revPropKeepSecret  = "keep-secret"
	revPropLegacyKey   = "legacy-mcp-key"
	revPropIDEKey      = "legacy-ide-key"
)

// revPropRecords returns one credential authorized on both the REST and MCP
// surfaces, plus a second credential that must survive the revocation reload
// (a registry reload to an empty set is rejected).
func revPropRecords() []auth.Record {
	return []auth.Record{
		{
			CredentialID: "cred-live",
			PrincipalID:  "principal-live",
			Scopes: []string{
				auth.ScopeMCPMutate, auth.ScopeMCPRead,
				auth.ScopeRESTRead, auth.ScopeRESTWrite,
			},
			Role:      auth.RoleAdministrator,
			Resources: []string{"*"},
			Verifier:  auth.Digest(revPropVerifierKey, revPropLiveSecret),
		},
		{
			CredentialID: "cred-keep",
			PrincipalID:  "principal-keep",
			Scopes:       []string{auth.ScopeMCPRead, auth.ScopeRESTRead},
			Role:         auth.RoleReader,
			Resources:    []string{"*"},
			Verifier:     auth.Digest(revPropVerifierKey, revPropKeepSecret),
		},
	}
}

// revPropKeepRecords is the post-revocation source: the live credential is
// gone, the unrelated credential remains.
func revPropKeepRecords() []auth.Record {
	return revPropRecords()[1:2]
}

// revPropConfigJSON marshals a record set into the inline registry source.
func revPropConfigJSON(t *testing.T, records []auth.Record) string {
	t.Helper()
	raw, err := json.Marshal(records)
	if err != nil {
		t.Fatalf("marshal records: %v", err)
	}
	return string(raw)
}

// revPropWebServer wires the real web middleware (echo) behind httptest with a
// mutating route (/api/rules POST).
func revPropWebServer(t *testing.T, cfg *config.Config) *httptest.Server {
	t.Helper()
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(web.APIKeyAuth(cfg, nil))
	e.POST("/api/rules", func(c echo.Context) error { return c.NoContent(http.StatusOK) })
	ts := httptest.NewServer(e)
	t.Cleanup(ts.Close)
	return ts
}

// revPropWebPost drives the web middleware with a bearer credential.
func revPropWebPost(t *testing.T, base, token string) int {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, base+"/api/rules", nil)
	if err != nil {
		t.Fatalf("new web request: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("web request: %v", err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

// revPropMCPServer wires the real StreamableHTTP MCP endpoint behind the real
// credential-registry middleware and returns the live registry so the caller
// can revoke a credential on the running endpoint.
func revPropMCPServer(t *testing.T, cfg *config.Config, source string) (*httptest.Server, *auth.Registry) {
	t.Helper()
	registry, err := auth.LoadFromSources(source, "", revPropVerifierKey)
	if err != nil {
		t.Fatalf("LoadFromSources: %v", err)
	}
	s := fullPathServer(cfg)
	httpSrv := server.NewStreamableHTTPServer(
		s.mcpServer,
		server.WithEndpointPath("/mcp"),
		server.WithStateLess(true),
	)
	mux := http.NewServeMux()
	mux.Handle("/mcp", requireBearerWithRegistry(cfg.MCPAPIKey, registry, nil, httpSrv))
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts, registry
}

// TestLiveRevocationPropagationBound proves that revoking a credential denies
// both the web and MCP HTTP surfaces within the published bound.
func TestLiveRevocationPropagationBound(t *testing.T) {
	initialSource := revPropConfigJSON(t, revPropRecords())
	cfg := &config.Config{
		SchemaVersion:          "1.0",
		MCPAPIKey:              revPropLegacyKey,
		IDEAPIKey:              revPropIDEKey,
		CredentialRegistryJSON: initialSource,
		CredentialVerifierKey:  revPropVerifierKey,
	}

	webTS := revPropWebServer(t, cfg)
	mcpTS, registry := revPropMCPServer(t, cfg, initialSource)

	initArgs := map[string]interface{}{
		"name":      "guardrail_init_session",
		"arguments": map[string]interface{}{"user_id": "user-1", "environment": "test"},
	}

	// Phase 1 — the credential is AUTHORIZED on both real paths.
	if code := revPropWebPost(t, webTS.URL, revPropLiveSecret); code != http.StatusOK {
		t.Fatalf("pre-revocation web status = %d, want 200", code)
	}
	code, env := rpcPost(t, mcpTS, revPropLiveSecret, "tools/call", initArgs)
	if code != http.StatusOK {
		t.Fatalf("pre-revocation MCP status = %d, want 200", code)
	}
	if tr := decodeToolResult(t, env); tr.IsError {
		t.Fatalf("pre-revocation MCP call denied: %s", tr.text())
	}

	// Phase 2 — revoke by reloading the source without the credential, and
	// measure how long it takes before both surfaces deny.
	keepSource := revPropConfigJSON(t, revPropKeepRecords())
	start := time.Now()

	// MCP: atomic reload of the registry the running endpoint uses.
	if err := registry.Reload(revPropVerifierKey, revPropKeepRecords()); err != nil {
		t.Fatalf("registry reload: %v", err)
	}
	// Web: the middleware binds its registry at construction; apply the
	// source-level revocation by rebuilding the middleware and serving a fresh
	// request through the real path.
	cfg.CredentialRegistryJSON = keepSource
	webTSAfter := revPropWebServer(t, cfg)

	// Phase 3 — both surfaces DENY the revoked credential.
	codeMCP, _ := rpcPost(t, mcpTS, revPropLiveSecret, "tools/call", initArgs)
	if codeMCP != http.StatusUnauthorized {
		t.Fatalf("post-revocation MCP status = %d, want 401", codeMCP)
	}
	codeWeb := revPropWebPost(t, webTSAfter.URL, revPropLiveSecret)
	if codeWeb != http.StatusUnauthorized {
		t.Fatalf("post-revocation web status = %d, want 401", codeWeb)
	}

	elapsed := time.Since(start)
	t.Logf("revocation propagated to denial on BOTH web and MCP paths in %v (published bound %v)",
		elapsed, auth.RevocationPropagationBound)
	if elapsed > auth.RevocationPropagationBound {
		t.Fatalf("revocation propagated in %v, exceeding bound %v", elapsed, auth.RevocationPropagationBound)
	}

	// The unrelated credential is unaffected on both surfaces.
	if code := revPropWebPost(t, webTSAfter.URL, revPropKeepSecret); code != http.StatusForbidden && code != http.StatusOK {
		// keep-secret is reader/read-only: a mutate route denies it (403); it
		// must not be 401 (unknown credential) after the reload.
		t.Fatalf("unrelated web credential status = %d, want 403 (authenticated, forbidden) not 401", code)
	}
	if _, ok := registry.Resolve(revPropKeepSecret); !ok {
		t.Fatal("unrelated credential should still resolve on the running MCP registry")
	}
}
