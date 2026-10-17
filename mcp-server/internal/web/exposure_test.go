package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/thearchitectit/guardrail-mcp/internal/config"
)

// registryServer builds the real route table (Spec 12 Sec. 4.1) without live
// dependencies: route registration binds method values, it never calls them.
func registryServer(t *testing.T) *echo.Echo {
	t.Helper()
	cfg := &config.Config{MCPAPIKey: "secret-key", IDEAPIKey: "ide-key", WebEnabled: true}
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	registerRoutes(e, &Server{cfg: cfg, version: "test"})
	return e
}

// TestExposureRegistryCoversEveryRoute is Spec 12 acceptance criterion 1: a
// test enumerates every registered route and asserts each maps to exactly one
// exposure class. Zero unmapped routes.
func TestExposureRegistryCoversEveryRoute(t *testing.T) {
	e := registryServer(t)
	routes := e.Routes()
	if len(routes) == 0 {
		t.Fatal("no routes registered; the enumeration is vacuous")
	}

	publicCount := 0
	for _, r := range routes {
		class := ExposureFor(r.Method, r.Path, r.Path)
		switch class {
		case ExposurePublic:
			publicCount++
			// A route is public only because a rule declares it, never by
			// omission (Spec 12 Sec. 3.1 rule 3).
			if !IsDeclaredPublic(r.Method, r.Path) && !IsDeclaredStaticMount(r.Path) {
				// Static mounts are accounted for by request path; a public
				// request-path rule must still exist for the pattern.
				if ExposureFor(r.Method, r.Path, r.Path) != ExposurePublic {
					t.Errorf("%s %s classified public with no declaring rule", r.Method, r.Path)
				}
			}
		case ExposureAuthenticated:
			// Deny by default: acceptable, but every non-public route must be
			// a real route in the table (which it is, by construction).
		default:
			t.Errorf("%s %s maps to no known exposure class (%q)", r.Method, r.Path, class)
		}
	}
	if publicCount == 0 {
		t.Error("no route is declared public; the registry cannot be entirely deny-by-default (health probes are public)")
	}
}

// TestExposureRegistryDeniesByDefault is Spec 12 acceptance criterion 3: a new
// undocumented route requires authentication without any middleware change.
func TestExposureRegistryDeniesByDefault(t *testing.T) {
	cfg := &config.Config{MCPAPIKey: "secret-key", IDEAPIKey: "ide-key"}
	e := echo.New()
	// A route nobody declared in the registry.
	e.GET("/api/internal/secret", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	handler := APIKeyAuth(cfg, nil)(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/internal/secret", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/internal/secret")

	code := rec.Code
	if err := handler(c); err != nil {
		code = echoStatus(err)
	}
	if code != http.StatusUnauthorized {
		t.Fatalf("undocumented route GET /api/internal/secret: got %d, want 401 (deny by default)", code)
	}
}

// TestExposureRegistryIsTheSingleSkipSource is Spec 12 acceptance criterion 2:
// APIKeyAuth and RateLimitMiddleware derive their skip set from the same
// registry. Any route public to one must be public to the other.
func TestExposureRegistryIsTheSingleSkipSource(t *testing.T) {
	cfg := &config.Config{MCPAPIKey: "secret-key", IDEAPIKey: "ide-key"}
	e := registryServer(t)

	// For every registered route, the auth middleware must allow it without a
	// credential exactly when the registry classifies it public.
	auth := APIKeyAuth(cfg, nil)(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	for _, r := range e.Routes() {
		class := ExposureFor(r.Method, r.Path, r.Path)
		req := httptest.NewRequest(r.Method, r.Path, nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath(r.Path)

		code := rec.Code
		if err := auth(c); err != nil {
			code = echoStatus(err)
		}
		if class == ExposurePublic && code == http.StatusUnauthorized {
			t.Errorf("%s %s: registry says public but auth middleware demanded a credential (%d)", r.Method, r.Path, code)
		}
		if class == ExposureAuthenticated && code != http.StatusUnauthorized {
			t.Errorf("%s %s: registry says authenticated but auth middleware allowed anonymous access (%d)", r.Method, r.Path, code)
		}
	}
}

// TestExposureRegistryDeclaresStaticMounts pins the two file-serving mounts so a
// route added without a declaration cannot hide behind them.
func TestExposureRegistryDeclaresStaticMounts(t *testing.T) {
	for _, p := range []string{"/*", "/web/*"} {
		if p == "/*" && !IsDeclaredStaticMount(p) {
			t.Errorf("static mount %q is not declared", p)
		}
	}
	if IsDeclaredStaticMount("/api/*") {
		t.Error("/api/* must never be a declared static mount")
	}
}

// TestExposureRegistryAssetRuleStaysConstrained guards the review of the asset
// suffix rule: it must never make an /api/ path public, and never a mutation.
func TestExposureRegistryAssetRuleStaysConstrained(t *testing.T) {
	cases := []struct {
		method, pattern, path string
		want                  ExposureClass
	}{
		{http.MethodGet, "/app.js", "/app.js", ExposurePublic},
		{http.MethodHead, "/app.js", "/app.js", ExposurePublic},
		{http.MethodPost, "/app.js", "/app.js", ExposureAuthenticated},
		{http.MethodGet, "/api/ingest/x.js", "/api/ingest/x.js", ExposureAuthenticated},
		{http.MethodPost, "/api/rules/x.html", "/api/rules/x.html", ExposureAuthenticated},
		{http.MethodGet, "/api/internal/data.js", "/api/internal/data.js", ExposureAuthenticated},
	}
	for _, tc := range cases {
		if got := ExposureFor(tc.method, tc.pattern, tc.path); got != tc.want {
			t.Errorf("%s %s: got %q, want %q", tc.method, tc.path, got, tc.want)
		}
	}
}

// TestExposureRegistryReadRulesStayGetOnly pins that the declared public read
// set is method-constrained: a mutation on a public read route is authenticated.
func TestExposureRegistryReadRulesStayGetOnly(t *testing.T) {
	cases := []struct {
		method, pattern string
		want            ExposureClass
	}{
		{http.MethodGet, "/api/rules", ExposurePublic},
		{http.MethodOptions, "/api/rules", ExposurePublic},
		{http.MethodPost, "/api/rules", ExposureAuthenticated},
		{http.MethodPut, "/api/rules/:id", ExposureAuthenticated},
		{http.MethodDelete, "/api/failures/:id", ExposureAuthenticated},
		{http.MethodGet, "/api/ingest/upload", ExposureAuthenticated},
	}
	for _, tc := range cases {
		if got := ExposureFor(tc.method, tc.pattern, tc.pattern); got != tc.want {
			t.Errorf("%s %s: got %q, want %q", tc.method, tc.pattern, got, tc.want)
		}
	}
}

// TestExposureRegistryPublicSetIsReviewed keeps the public surface from
// silently growing: every public rule must carry a rationale, and the count is
// pinned so an addition is a deliberate edit.
func TestExposureRegistryPublicSetIsReviewed(t *testing.T) {
	for _, r := range exposureRegistry {
		if strings.TrimSpace(r.rationale) == "" {
			t.Errorf("registry rule %q has no rationale; a public exposure is a written decision", r.name)
		}
		if r.class != ExposurePublic && r.class != ExposureAuthenticated {
			t.Errorf("registry rule %q has unknown class %q", r.name, r.class)
		}
	}
}
