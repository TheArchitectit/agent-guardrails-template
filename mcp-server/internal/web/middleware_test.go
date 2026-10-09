package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/thearchitectit/guardrail-mcp/internal/config"
)

// TestAPIKeyAuth_StaticExtensionDoesNotBypassAuth is a regression test for an
// authentication bypass: the middleware used to exempt any path ending in a
// static-asset extension, regardless of method or route. A request such as
// POST /api/ingest/x.js therefore reached the handler with no credentials.
func TestAPIKeyAuth_StaticExtensionDoesNotBypassAuth(t *testing.T) {
	cfg := &config.Config{MCPAPIKey: "secret-key", IDEAPIKey: "ide-key"}

	cases := []struct {
		name   string
		method string
		path   string
		want   int
	}{
		{"POST to .js under /api is authenticated", http.MethodPost, "/api/ingest/x.js", http.StatusUnauthorized},
		{"POST to .html under /api is authenticated", http.MethodPost, "/api/rules/x.html", http.StatusUnauthorized},
		{"DELETE to .js under /api is authenticated", http.MethodDelete, "/api/rules/x.js", http.StatusUnauthorized},
		{"GET to .js under /api is authenticated", http.MethodGet, "/api/secrets/x.js", http.StatusUnauthorized},
		{"GET of a real static asset stays public", http.MethodGet, "/static/app.js", http.StatusOK},
		{"GET of a root asset stays public", http.MethodGet, "/favicon.ico", http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := echo.New()
			handler := APIKeyAuth(cfg, nil)(func(c echo.Context) error {
				return c.NoContent(http.StatusOK)
			})

			req := httptest.NewRequest(tc.method, tc.path, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := handler(c)

			// echo returns HTTPError rather than writing it; normalise.
			code := rec.Code
			if err != nil {
				code = echoStatus(err)
			}
			if code != tc.want {
				t.Fatalf("%s %s: got status %d, want %d", tc.method, tc.path, code, tc.want)
			}
		})
	}
}

// TestSecurityHeaders_NoXSSProtection covers the spec 12-3.6 / audit gap 6
// hardening: the deprecated X-XSS-Protection header must not be emitted, and
// the headers that do the real work must still be present.
func TestSecurityHeaders_NoXSSProtection(t *testing.T) {
	e := echo.New()
	handler := securityHeadersMiddleware()(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	if err := handler(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := rec.Header().Get("X-XSS-Protection"); got != "" {
		t.Errorf("X-XSS-Protection must not be set, got %q", got)
	}
	if got := rec.Header().Get("Content-Security-Policy"); got == "" {
		t.Error("Content-Security-Policy must be set")
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
	}
	if got := rec.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("X-Frame-Options = %q, want DENY", got)
	}
}

// TestAPIKeyAuth_ValidKeyStillAuthenticates guards the happy path so the
// hardening above cannot be satisfied by rejecting everything.
func TestAPIKeyAuth_ValidKeyStillAuthenticates(t *testing.T) {
	cfg := &config.Config{MCPAPIKey: "secret-key", IDEAPIKey: "ide-key"}

	e := echo.New()
	handler := APIKeyAuth(cfg, nil)(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/ingest", nil)
	req.Header.Set("Authorization", "Bearer secret-key")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler(c)
	if err != nil {
		t.Fatalf("valid key rejected: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("valid key: got status %d, want 200", rec.Code)
	}
}

func echoStatus(err error) int {
	if he, ok := err.(*echo.HTTPError); ok {
		return he.Code
	}
	return http.StatusInternalServerError
}