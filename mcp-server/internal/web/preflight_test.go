package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/thearchitectit/guardrail-mcp/internal/config"
)

// newPreflightServer builds an Echo instance with the same ordering the real
// server uses: the Sec. 3.3.4 declared-preflight guard, then CORS, then a declared
// API route and a catch-all file/SPA fallback (as echo.Static registers).
func newPreflightServer(t *testing.T, cfg *config.Config) *echo.Echo {
	t.Helper()
	e := echo.New()
	e.Use(declaredPreflightOnly())
	origins, substituted, err := cfg.EffectiveCORSOrigins()
	if err != nil {
		origins = []string{"https://invalid.invalid"}
	}
	_ = substituted
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: origins,
		AllowMethods: cfg.CORSAllowedMethods,
		AllowHeaders: cfg.CORSAllowedHeaders,
		MaxAge:       cfg.CORSMaxAge,
	}))
	e.GET("/api/rules", func(c echo.Context) error { return c.NoContent(http.StatusOK) })
	// Catch-all fallback, as the Web UI mounts it.
	e.GET("/*", func(c echo.Context) error { return c.NoContent(http.StatusOK) })
	return e
}

// TestPreflightDeclaredRouteOnly covers Spec 12 Sec. 3.3.4: a CORS preflight for a
// declared route is answered (204), while an OPTIONS request for an undeclared
// path — which the catch-all fallback would otherwise match — is refused (404)
// and never reaches CORS. The check is on the real router + middleware chain.
func TestPreflightDeclaredRouteOnly(t *testing.T) {
	cfg := &config.Config{
		CORSAllowedOrigins: []string{"https://app.example.com"},
		CORSAllowedMethods: []string{"GET", "POST", "OPTIONS"},
		CORSAllowedHeaders: []string{"Authorization", "Content-Type"},
	}
	e := newPreflightServer(t, cfg)

	cases := []struct {
		name   string
		path   string
		want   int
		origin string
	}{
		{"declared route preflight", "/api/rules", http.StatusNoContent, "https://app.example.com"},
		{"undeclared path preflight refused", "/api/does-not-exist", http.StatusNotFound, "https://app.example.com"},
		{"undeclared nested path preflight refused", "/api/rules/extra/deep", http.StatusNotFound, "https://app.example.com"},
		{"unrelated path preflight refused", "/totally/undeclared", http.StatusNotFound, "https://app.example.com"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodOptions, tc.path, nil)
			req.Header.Set(echo.HeaderOrigin, tc.origin)
			req.Header.Set(echo.HeaderAccessControlRequestMethod, http.MethodGet)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("OPTIONS %s: got %d, want %d", tc.path, rec.Code, tc.want)
			}
		})
	}
}

// TestPreflightGuardLeavesNonOptionsUntouched ensures the guard acts only on
// OPTIONS: a GET to the catch-all fallback still resolves normally.
func TestPreflightGuardLeavesNonOptionsUntouched(t *testing.T) {
	cfg := &config.Config{CORSAllowedOrigins: []string{"https://app.example.com"}}
	e := newPreflightServer(t, cfg)

	for _, p := range []string{"/api/rules", "/some/spa/route"} {
		req := httptest.NewRequest(http.MethodGet, p, nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s: got %d, want 200", p, rec.Code)
		}
	}
}
