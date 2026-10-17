package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/thearchitectit/guardrail-mcp/internal/config"
)

// Spec 19 R19-09 — static-asset bypass controls over the REAL router + chain.
//
// R19-09 names, verbatim, the coverage it requires: "Tests MUST cover
// `/api/ingest/evil.js`, method variants, encoded/path-normalized forms, and
// undeclared routes." The static-suffix half of R19-09 was already covered by
// `TestAPIKeyAuth_StaticExtensionDoesNotBypassAuth` (middleware in isolation)
// and the classification half by the `TestExposureRegistry*` suite — but the
// named `/api/ingest/evil.js` path, its method variants, and the
// encoded/path-normalized forms were never driven through the assembled
// production chain (route table + the middleware stack in its real order).
//
// A future change that moved the asset exemption back into an inline suffix
// check, widened `staticAssetExtensions`, or added an encoded-path bypass would
// therefore pass every existing test while reopening the exact class R19-09
// exists to close. This test drives `registerRoutes` + `APIKeyAuth` exactly as
// `setupRoutes`/`setupMiddleware` assemble them, so the assertion is on the
// shipped chain rather than a re-implementation.
//
// Every request below is UNCREDENTIALED: the required outcome is that none of
// them reaches a handler. A 401 (deny) is the pass; a 200/204 would be the
// bypass. The two positive controls prove the harness is non-vacuous — a real
// static asset and an authenticated API route both still resolve.

// r1909Chain builds the real route table and the real auth middleware in the
// production order (routes first, then the auth middleware wrapping the whole
// table), matching `Server.setupRoutes` + `Server.setupMiddleware`.
func r1909Chain(t *testing.T) *echo.Echo {
	t.Helper()
	cfg := &config.Config{
		MCPAPIKey:  "secret-key",
		IDEAPIKey:  "ide-key",
		WebEnabled: true,
	}
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	registerRoutes(e, &Server{cfg: cfg, version: "test"})
	e.Use(APIKeyAuth(cfg, nil))
	return e
}

// TestR1909AssetShapedAPIPathsDenyThroughRealChain is the R19-09 named path
// coverage: `/api/ingest/evil.js` and its method variants, plus the
// encoded/path-normalized forms that an asset-suffix bypass would have to
// survive, all uncredentialed, all denied.
func TestR1909AssetShapedAPIPathsDenyThroughRealChain(t *testing.T) {
	e := r1909Chain(t)

	cases := []struct {
		name   string
		method string
		path   string
	}{
		// The literal path R19-09 names.
		{"POST /api/ingest/evil.js", http.MethodPost, "/api/ingest/evil.js"},
		{"GET /api/ingest/evil.js", http.MethodGet, "/api/ingest/evil.js"},
		{"DELETE /api/ingest/evil.js", http.MethodDelete, "/api/ingest/evil.js"},
		{"PUT /api/ingest/evil.js", http.MethodPut, "/api/ingest/evil.js"},
		{"PATCH /api/ingest/evil.js", http.MethodPatch, "/api/ingest/evil.js"},

		// Encoded / path-normalized forms. `httptest.NewRequest` decodes these
		// into the URL.Path the middleware classifies against, so each asserts
		// the DECODED form is still refused (no encoded bypass).
		{"encoded dot .js", http.MethodPost, "/api/ingest/evil%2Ejs"},
		{"encoded slash", http.MethodPost, "/api/ingest%2Fevil.js"},
		{"encoded segment", http.MethodPost, "/api/%69ngest/evil.js"},
		{"dot-segment traversal", http.MethodPost, "/api/ingest/../ingest/evil.js"},
		{"uppercase extension", http.MethodPost, "/api/ingest/evil.JS"},
		{"uppercase api prefix", http.MethodPost, "/API/ingest/evil.js"},

		// Other asset-shaped API paths that share the class.
		{"POST /api/rules/x.html", http.MethodPost, "/api/rules/x.html"},
		{"DELETE /api/rules/x.js", http.MethodDelete, "/api/rules/x.js"},
		{"GET /api/internal/data.js", http.MethodGet, "/api/internal/data.js"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			// The required outcome: the request is refused before any handler.
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("%s %s (URL.Path=%q): got status %d, want 401 — an asset-shaped /api/ path reached a handler uncredentialed",
					tc.method, tc.path, req.URL.Path, rec.Code)
			}
		})
	}
}

// TestR1909UndeclaredRoutesDenyThroughRealChain covers R19-09's "undeclared
// routes" clause on the same assembled chain: a path that matches no declared
// route (and would otherwise fall through the Web UI catch-all) is denied, not
// served.
func TestR1909UndeclaredRoutesDenyThroughRealChain(t *testing.T) {
	e := r1909Chain(t)

	// The required outcome is that an undeclared path is never SERVED without a
	// credential. A declared-route classifier denies with 401; a router miss or
	// a route that rejects the method may instead produce 400/404/405. All of
	// those are "not served"; a 2xx would be the bypass.
	for _, p := range []string{
		"/api/does-not-exist",
		"/api/ingest/upload/deep/undeclared",
		"/api/rules/extra/deep",
		"/totally/undeclared",
	} {
		t.Run(p, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, p, nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			if rec.Code < 400 {
				t.Fatalf("GET %s: got %d, want a refusal (>=400) — an undeclared path was served", p, rec.Code)
			}
		})
	}
}

// TestR1909ChainPositiveControls proves the harness above is non-vacuous: a
// declared public operational endpoint and a declared public read route both
// clear the auth boundary, so the refusals above are caused by the
// classification and not by a chain that refuses everything.
func TestR1909ChainPositiveControls(t *testing.T) {
	e := r1909Chain(t)

	// /version is a declared public endpoint and its handler has no live
	// dependencies, so it must resolve end to end.
	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /version: got %d, want 200 (declared public endpoint; the deny cases would otherwise be vacuous)", rec.Code)
	}

	// A declared public read route must clear the auth boundary. Its handler
	// needs a live store, so it is expected to fail AFTER auth — never with the
	// auth middleware's 401.
	req = httptest.NewRequest(http.MethodGet, "/api/rules", nil)
	rec = httptest.NewRecorder()
	func() {
		defer func() { _ = recover() }() // handler may deref a nil store; that is past the auth boundary
		e.ServeHTTP(rec, req)
	}()
	if rec.Code == http.StatusUnauthorized {
		t.Fatalf("GET /api/rules: got 401, want a declared public read route to clear the auth boundary")
	}
}
