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

// This file proves Spec 16 R16-09's revocation-propagation bound using the
// PROCESS-WIDE registry shared by both real surfaces, not the per-surface
// rebuild fallback that streamable_http_revocation_test.go documents.
//
// The distinction matters: before this change the web middleware bound its
// registry at construction and nothing reloaded it in-process, so an operator
// revocation reached the web surface only after a rebuild/restart. Here the
// web middleware and the MCP endpoint authenticate against the SAME
// *auth.Registry; a single in-place ReloadOnce (the loop's tick, simulated
// here) must deny the very next request on BOTH surfaces, with no rebuild and
// no restart, inside RevocationPropagationBound.

// sharedReloadWebServer wires the real web middleware against a caller-owned
// registry via APIKeyAuthWithRegistry, so the test controls the shared object.
func sharedReloadWebServer(t *testing.T, cfg *config.Config, registry *auth.Registry) *httptest.Server {
	t.Helper()
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(web.APIKeyAuthWithRegistry(cfg, nil, registry, nil))
	e.POST("/api/rules", func(c echo.Context) error { return c.NoContent(http.StatusOK) })
	ts := httptest.NewServer(e)
	t.Cleanup(ts.Close)
	return ts
}

// TestSharedRegistryRevocationPropagatesInProcess proves that one in-place
// reload of the shared registry denies a revoked credential on both surfaces
// without any rebuild, within the published bound.
func TestSharedRegistryRevocationPropagatesInProcess(t *testing.T) {
	initialSource := revPropConfigJSON(t, revPropRecords())
	cfg := &config.Config{
		SchemaVersion:          "1.0",
		MCPAPIKey:              revPropLegacyKey,
		IDEAPIKey:              revPropIDEKey,
		CredentialRegistryJSON: initialSource,
		CredentialVerifierKey:  revPropVerifierKey,
	}

	// ONE registry, shared by both surfaces — exactly the wiring main.go now
	// installs via SetCredentialRegistry / APIKeyAuthWithRegistry.
	registry, err := auth.LoadFromSources(initialSource, "", revPropVerifierKey)
	if err != nil {
		t.Fatalf("LoadFromSources: %v", err)
	}

	webTS := sharedReloadWebServer(t, cfg, registry)

	s := fullPathServer(cfg)
	s.SetCredentialRegistry(registry)
	httpSrv := server.NewStreamableHTTPServer(
		s.mcpServer,
		server.WithEndpointPath("/mcp"),
		server.WithStateLess(true),
	)
	mux := http.NewServeMux()
	mux.Handle("/mcp", requireBearerWithRegistry(cfg.MCPAPIKey, registry, nil, httpSrv))
	mcpTS := httptest.NewServer(mux)
	t.Cleanup(mcpTS.Close)

	initArgs := map[string]interface{}{
		"name":      "guardrail_init_session",
		"arguments": map[string]interface{}{"user_id": "user-1", "environment": "test"},
	}

	// Phase 1 — the credential is AUTHORIZED on both real surfaces.
	if code := revPropWebPost(t, webTS.URL, revPropLiveSecret); code != http.StatusOK {
		t.Fatalf("pre-revocation web status = %d, want 200", code)
	}
	if code, env := rpcPost(t, mcpTS, revPropLiveSecret, "tools/call", initArgs); code != http.StatusOK {
		t.Fatalf("pre-revocation MCP status = %d, want 200", code)
	} else if tr := decodeToolResult(t, env); tr.IsError {
		t.Fatalf("pre-revocation MCP call denied: %s", tr.text())
	}

	// Phase 2 — revoke by ONE in-place reload of the shared registry. This is
	// the exact call the background loop makes on its tick; no middleware is
	// rebuilt and no process is restarted.
	start := time.Now()
	keepRecords := revPropKeepRecords()
	keepRaw, err := json.Marshal(keepRecords)
	if err != nil {
		t.Fatalf("marshal keep records: %v", err)
	}
	if err := registry.ReloadOnce(auth.Sources{
		InlineJSON:  string(keepRaw),
		VerifierKey: revPropVerifierKey,
	}); err != nil {
		t.Fatalf("ReloadOnce: %v", err)
	}

	// Phase 3 — both surfaces DENY the revoked credential on the very next
	// request, through the SAME still-running servers.
	codeMCP, _ := rpcPost(t, mcpTS, revPropLiveSecret, "tools/call", initArgs)
	if codeMCP != http.StatusUnauthorized {
		t.Fatalf("post-revocation MCP status = %d, want 401", codeMCP)
	}
	codeWeb := revPropWebPost(t, webTS.URL, revPropLiveSecret)
	if codeWeb != http.StatusUnauthorized {
		t.Fatalf("post-revocation web status = %d, want 401", codeWeb)
	}

	elapsed := time.Since(start)
	t.Logf("in-process revocation denied BOTH surfaces in %v (published bound %v)", elapsed, auth.RevocationPropagationBound)
	if elapsed > auth.RevocationPropagationBound {
		t.Fatalf("revocation propagated in %v, exceeding bound %v", elapsed, auth.RevocationPropagationBound)
	}

	// The unrelated credential survives on both surfaces (a reload must not
	// widen or narrow anything but the revoked record).
	if code := revPropWebPost(t, webTS.URL, revPropKeepSecret); code != http.StatusForbidden && code != http.StatusOK {
		t.Fatalf("unrelated web credential status = %d, want 403 (authenticated) or 200, not 401", code)
	}
	if _, ok := registry.Resolve(revPropKeepSecret); !ok {
		t.Fatal("unrelated credential should still resolve after the in-place reload")
	}
}

// TestSharedRegistryReloadFailureKeepsPriorSnapshot proves the fail-closed
// property of the reload path: a malformed source leaves the previous
// validated snapshot in force, so a transient bad source neither denies the
// surviving credential nor resurrects the revoked one.
func TestSharedRegistryReloadFailureKeepsPriorSnapshot(t *testing.T) {
	registry, err := auth.LoadFromSources(revPropConfigJSON(t, revPropRecords()), "", revPropVerifierKey)
	if err != nil {
		t.Fatalf("LoadFromSources: %v", err)
	}
	// A malformed replacement must be rejected.
	if err := registry.ReloadOnce(auth.Sources{InlineJSON: "{not json", VerifierKey: revPropVerifierKey}); err == nil {
		t.Fatal("malformed reload must return an error, not silently apply")
	}
	// The prior snapshot is untouched: the live credential still resolves.
	if _, ok := registry.Resolve(revPropLiveSecret); !ok {
		t.Fatal("a failed reload must not drop the still-valid credential")
	}
}
